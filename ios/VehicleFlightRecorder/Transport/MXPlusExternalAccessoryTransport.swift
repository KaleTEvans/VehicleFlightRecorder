import Foundation
#if !targetEnvironment(simulator)
import ExternalAccessory
#endif

struct AccessoryDescriptor {
    let id: Int
    let protocols: [String]
    let metadata: [String: String]
}
protocol AccessoryDirectory: AnyObject {
    var accessories: [AccessoryDescriptor] { get }
    func observe(_ changed: @escaping () -> Void)
    func stopObserving()
    func open(_ id: Int) -> NativeStreamChannel?
}
#if !targetEnvironment(simulator)
final class EAAccessoryDirectory: AccessoryDirectory {
    private var observers: [NSObjectProtocol] = []
    var accessories: [AccessoryDescriptor] {
        EAAccessoryManager.shared().connectedAccessories.map { accessory in
            AccessoryDescriptor(id: accessory.connectionID, protocols: accessory.protocolStrings,
                metadata: ["name": accessory.name, "manufacturer": accessory.manufacturer, "model": accessory.modelNumber,
                           "firmware": accessory.firmwareRevision, "hardware": accessory.hardwareRevision,
                           "native_connection_id": String(accessory.connectionID)])
        }
    }
    func observe(_ changed: @escaping () -> Void) {
        guard observers.isEmpty else { return }
        for name in [NSNotification.Name.EAAccessoryDidConnect, .EAAccessoryDidDisconnect] {
            observers.append(NotificationCenter.default.addObserver(forName: name, object: nil, queue: nil) { _ in changed() })
        }
        EAAccessoryManager.shared().registerForLocalNotifications()
    }
    func stopObserving() {
        for observer in observers { NotificationCenter.default.removeObserver(observer) }; observers.removeAll()
        EAAccessoryManager.shared().unregisterForLocalNotifications()
    }
    func open(_ id: Int) -> NativeStreamChannel? {
        guard let accessory = EAAccessoryManager.shared().connectedAccessories.first(where: { $0.connectionID == id }),
              accessory.protocolStrings.contains("com.obdlink"),
              let session = EASession(accessory: accessory, forProtocol: "com.obdlink"),
              let input = session.inputStream, let output = session.outputStream else { return nil }
        return NativeStreamChannel(input: input, output: output, owner: session)
    }
}
#endif

// All methods, stream callbacks and capture calls run on TransportExecutor.
final class MXPlusExternalAccessoryTransport: NSObject, VehicleTransport, StreamDelegate {
    let pump = TransportPump()
    var state: TransportState { pump.state }
    var notice = "No accessory selected"
    var capture: GoConnectionCapture?
    private let root: URL
    private let executor: TransportExecutor
    private let directory: AccessoryDirectory?
    private var timer: Timer?
    private var armed = false
    private var inputOpened = false
    private var outputOpened = false
    private var channel: NativeStreamChannel?
    private var selectedID: Int?
    init(root: URL, executor: TransportExecutor, directory: AccessoryDirectory? = nil) {
        self.root = root; self.executor = executor
        #if targetEnvironment(simulator)
        self.directory = directory
        #else
        self.directory = directory ?? EAAccessoryDirectory()
        #endif
        super.init()
    }
    func connect() throws {
        guard let directory else { throw VehicleTransportError.unavailableInBootstrap }
        guard !armed else { discover(); return }
        armed = true
        directory.observe { [weak self] in
            guard let self else { return }
            if !self.executor.submit({ [weak self] in self?.discover() }) {
                self.overloadLock.lock(); self.overloaded = true; self.overloadLock.unlock()
            }
        }
        timer = Timer.scheduledTimer(withTimeInterval: 0.5, repeats: true) { [weak self] _ in
            guard let self else { return }
            self.overloadLock.lock(); let overflow = self.overloaded; self.overloaded = false; self.overloadLock.unlock()
            if overflow {
                self.pump.fail("Native command mailbox overflow"); self.armed = false
                self.directory?.stopObserving(); self.timer?.invalidate(); self.timer = nil
                self.releaseSession(); return
            }
            self.pump.tick()
            if self.pump.state == .failed { self.releaseSession() }
        }
        discover()
    }
    private let overloadLock = NSLock()
    private var overloaded = false
    private func discover() {
        guard armed, let directory else { return }
        let all = directory.accessories
        if let selectedID, !all.contains(where: { $0.id == selectedID }) {
            pump.fail("Accessory disconnected"); releaseSession()
        }
        guard state != .connecting && state != .ready else { return }
        let matches = all.filter { $0.protocols.contains("com.obdlink") }
        guard matches.count == 1 else {
            notice = matches.isEmpty ? "No com.obdlink accessory (\(all.count) connected accessories)" : "Multiple com.obdlink accessories; disconnect all but the test adapter"
            return
        }
        let accessory = matches[0]
        do {
            var metadata = accessory.metadata
            metadata["protocol"] = "com.obdlink"; metadata["app_version"] = "0.1.0"
            metadata["build"] = Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "unknown"
            metadata["ios"] = ProcessInfo.processInfo.operatingSystemVersionString
            let log = try GoConnectionCapture(root: root, metadata: metadata)
            capture = log
            guard let streams = directory.open(accessory.id) else {
                try log.event("error", "EASession creation failed"); try log.close("session creation failed")
                throw VehicleTransportError.failed("EASession creation failed; check pairing, competing apps and provisioning")
            }
            try pump.begin(channel: streams, capture: log)
            self.channel = streams; selectedID = accessory.id
            let input = streams.input, output = streams.output
            inputOpened = false; outputOpened = false
            input.delegate = self; output.delegate = self
            input.schedule(in: .current, forMode: .default); output.schedule(in: .current, forMode: .default)
            input.open(); output.open()
            notice = metadata.sorted(by: { $0.key < $1.key }).map { "\($0.key): \($0.value)" }.joined(separator: " • ")
        } catch { notice = error.localizedDescription; pump.fail(error.localizedDescription); releaseSession() }
    }
    func requestIdentity() throws { try pump.identify() }
    func disconnect() {
        armed = false; timer?.invalidate(); timer = nil
        directory?.stopObserving()
        pump.disconnect(); releaseSession()
    }
    private func releaseSession() {
        channel?.close(); channel = nil
        selectedID = nil
    }
    func stream(_ stream: Stream, handle eventCode: Stream.Event) {
        guard let channel, stream === channel.input || stream === channel.output else { return }
        let generation = pump.generation
        if eventCode.contains(.errorOccurred) || eventCode.contains(.endEncountered) {
            pump.fail(stream.streamError?.localizedDescription ?? "Native stream ended"); releaseSession(); return
        }
        if eventCode.contains(.openCompleted) {
            if stream === channel.input { inputOpened = true } else { outputOpened = true }
            if inputOpened && outputOpened { pump.ready(generation: generation) }
        }
        if eventCode.contains(.hasBytesAvailable) {
            var bytes = [Int8](repeating: 0, count: 4096)
            // Bound work per callback so timers/close commands can make progress.
            for _ in 0..<16 {
                guard channel.input.hasBytesAvailable, state == .ready || state == .connecting else { break }
                let count = channel.input.read(&bytes, maxLength: bytes.count)
                if count < 0 { pump.fail("Input stream read failed"); releaseSession(); return }
                if count == 0 { break }
                pump.receive(Data(bytes.prefix(count)), generation: generation)
            }
            if state == .failed { releaseSession() }
        }
        if eventCode.contains(.hasSpaceAvailable) { pump.flush(generation: generation) }
    }
}
final class NativeStreamChannel: TransportChannel {
    let input: InputStream
    let output: OutputStream
    private var closed = false
    private let owner: AnyObject?
    init(input: InputStream, output: OutputStream, owner: AnyObject? = nil) { self.input = input; self.output = output; self.owner = owner }
    var hasSpace: Bool { !closed && output.hasSpaceAvailable }
    func write(_ data: Data) -> Int { data.withUnsafeBytes { output.write($0.bindMemory(to: Int8.self).baseAddress!, maxLength: data.count) } }
    func close() {
        guard !closed else { return }; closed = true
        for stream in [input as Stream, output as Stream] {
            stream.delegate = nil; stream.close(); stream.remove(from: .current, forMode: .default)
        }
    }
}
