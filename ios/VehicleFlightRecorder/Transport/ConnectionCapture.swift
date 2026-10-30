import Foundation
import Darwin
import VehicleFlightRecorderCore

enum TransportClock {
    static func nanoseconds() -> Int64 {
        var info = mach_timebase_info_data_t()
        mach_timebase_info(&info)
        return Int64(Double(mach_continuous_time()) * Double(info.numer) / Double(info.denom))
    }
}

final class GoConnectionCapture: TransportCapture {
    let core: MobileConnectionCapture
    static func clock() -> Int64 { TransportClock.nanoseconds() }
    init(root: URL, metadata: [String: String]) throws {
        guard MobileBridgeVersion() == 1 else { throw VehicleTransportError.failed("Incompatible bridge") }
        let json = String(decoding: try JSONSerialization.data(withJSONObject: metadata), as: UTF8.self)
        var error: NSError?
        guard let value = MobileNewConnectionCapture(root.path, json, Int64(Date().timeIntervalSince1970 * 1000), Self.clock(), &error) else {
            if let error { throw error }
            throw VehicleTransportError.failed("Could not create connection capture")
        }
        core = value
    }
    func receive(_ data: Data) throws { try core.receive(data, ns: Self.clock()) }
    func accepted(_ data: Data) throws { try core.accepted(data, ns: Self.clock()) }
    func identity() throws -> Data {
        let bytes: Data? = try core.identityRequest(Self.clock())
        guard let bytes else { throw VehicleTransportError.failed("Missing Go command") }
        return bytes
    }
    func event(_ kind: String, _ detail: String) throws { try core.event(kind, detail: detail, ns: Self.clock()) }
    func close(_ reason: String) throws { try core.close(reason, ns: Self.clock()) }
}

// A bounded mailbox for UI/notification commands; stream callbacks run directly
// on this thread. Disk work and the Go boundary never run on the UI thread.
final class TransportExecutor {
    private let thread: Thread
    private let lock = NSLock()
    private var pending = 0
    init() {
        thread = Thread {
            let timer = Timer(timeInterval: 1, repeats: true) { _ in }
            RunLoop.current.add(timer, forMode: .default)
            while !Thread.current.isCancelled { RunLoop.current.run(until: Date(timeIntervalSinceNow: 1)) }
            timer.invalidate()
        }
        thread.name = "VFR.ExternalAccessory"; thread.start()
    }
    @discardableResult func submit(_ body: @escaping () -> Void) -> Bool {
        lock.lock()
        guard pending < 32 else { lock.unlock(); return false }
        pending += 1; lock.unlock()
        let task = TransportTask { [self] in
            defer { lock.lock(); pending -= 1; lock.unlock() }
            body()
        }
        task.perform(#selector(TransportTask.run), on: thread, with: nil, waitUntilDone: false)
        return true
    }
    deinit { thread.cancel() }
}
private final class TransportTask: NSObject {
    let body: () -> Void
    init(_ body: @escaping () -> Void) { self.body = body }
    @objc func run() { body() }
}
