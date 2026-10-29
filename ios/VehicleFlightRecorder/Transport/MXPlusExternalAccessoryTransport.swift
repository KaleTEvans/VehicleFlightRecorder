import Foundation
#if !targetEnvironment(simulator)
import ExternalAccessory
#endif

// All methods, stream callbacks and capture calls run on TransportExecutor.
final class MXPlusExternalAccessoryTransport: NSObject, VehicleTransport, StreamDelegate {
    let pump = TransportPump()
    var state: TransportState { pump.state }
    var notice = "No accessory selected"
    var capture: GoConnectionCapture?
    private let root: URL
    private let executor: TransportExecutor
    private var observers: [NSObjectProtocol] = []
    private var timer: Timer?
    private var armed = false
    private var inputOpened = false
    private var outputOpened = false
    private var channel: NativeStreamChannel?
    #if !targetEnvironment(simulator)
    private var session: EASession?
    private var selectedID: UInt?
    #endif
    init(root: URL, executor: TransportExecutor) { self.root = root; self.executor = executor; super.init() }
    func connect() throws {
        #if targetEnvironment(simulator)
        throw VehicleTransportError.unavailableInBootstrap
        #else
        guard !armed else { discover(); return }
        armed = true
        let manager = EAAccessoryManager.shared()
        for name in [NSNotification.Name.EAAccessoryDidConnect, .EAAccessoryDidDisconnect] {
            observers.append(NotificationCenter.default.addObserver(forName: name, object: nil, queue: nil) { [weak self] _ in
                guard let self else { return }
                if !self.executor.submit({ [weak self] in self?.discover() }) {
                    // Mailbox cannot carry bytes. Surface admission failure on
                    // the next native timer instead of silently adding work.
                    self.overloadLock.lock(); self.overloaded = true; self.overloadLock.unlock()
                }
            })
        }
        manager.registerForLocalNotifications()
        timer = Timer.scheduledTimer(withTimeInterval: 0.5, repeats: true) { [weak self] _ in
            guard let self else { return }
            self.overloadLock.lock(); let overflow = self.overloaded; self.overloaded = false; self.overloadLock.unlock()
            if overflow { self.pump.fail("Native command mailbox overflow"); self.armed = false }
            self.pump.tick()
            if self.pump.state == .failed { self.releaseSession() }
        }
        discover()
        #endif
    }
    private let overloadLock = NSLock()
    private var overloaded = false
    private func discover() {
        #if !targetEnvironment(simulator)
        guard armed else { return }
        let all = EAAccessoryManager.shared().connectedAccessories
        if let selectedID, !all.contains(where: { $0.connectionID == selectedID }) {
            pump.fail("Accessory disconnected"); releaseSession()
        }
        guard state != .connecting && state != .ready else { return }
        let matches = all.filter { $0.protocolStrings.contains("com.obdlink") }
        guard matches.count == 1 else {
            notice = matches.isEmpty ? "No com.obdlink accessory (\(all.count) connected accessories)" : "Multiple com.obdlink accessories; disconnect all but the test adapter"
            return
        }
        let accessory = matches[0]
        do {
            let log = try GoConnectionCapture(root: root, metadata: ["name": accessory.name, "manufacturer": accessory.manufacturer, "model": accessory.modelNumber, "firmware": accessory.firmwareRevision, "hardware": accessory.hardwareRevision, "protocol": "com.obdlink", "native_connection_id": String(accessory.connectionID), "app_version": "0.1.0", "build": Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "unknown", "ios": ProcessInfo.processInfo.operatingSystemVersionString])
            capture = log
            guard let session = EASession(accessory: accessory, forProtocol: "com.obdlink"), let input = session.inputStream, let output = session.outputStream else {
                try log.event("error", "EASession creation failed"); try log.close("session creation failed")
                throw VehicleTransportError.failed("EASession creation failed; check pairing, competing apps and provisioning")
            }
            let streams = NativeStreamChannel(input: input, output: output)
            try pump.begin(channel: streams, capture: log)
            self.session = session; self.channel = streams; selectedID = accessory.connectionID
            inputOpened = false; outputOpened = false
            input.delegate = self; output.delegate = self
            input.schedule(in: .current, forMode: .default); output.schedule(in: .current, forMode: .default)
            input.open(); output.open()
            notice = "\(accessory.name) • \(accessory.modelNumber) • firmware \(accessory.firmwareRevision)"
        } catch { notice = error.localizedDescription; pump.fail(error.localizedDescription); releaseSession() }
        #endif
    }
    func requestIdentity() throws { try pump.identify() }
    func disconnect() {
        armed = false; timer?.invalidate(); timer = nil
        for observer in observers { NotificationCenter.default.removeObserver(observer) }; observers.removeAll()
        #if !targetEnvironment(simulator)
        EAAccessoryManager.shared().unregisterForLocalNotifications()
        #endif
        pump.disconnect(); releaseSession()
    }
    private func releaseSession() {
        channel?.close(); channel = nil
        #if !targetEnvironment(simulator)
        session = nil; selectedID = nil
        #endif
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
            var bytes = [UInt8](repeating: 0, count: 4096)
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
    init(input: InputStream, output: OutputStream) { self.input = input; self.output = output }
    var hasSpace: Bool { !closed && output.hasSpaceAvailable }
    func write(_ data: Data) -> Int { data.withUnsafeBytes { output.write($0.bindMemory(to: UInt8.self).baseAddress!, maxLength: data.count) } }
    func close() {
        guard !closed else { return }; closed = true
        for stream in [input as Stream, output as Stream] {
            stream.delegate = nil; stream.close(); stream.remove(from: .current, forMode: .default)
        }
    }
}
