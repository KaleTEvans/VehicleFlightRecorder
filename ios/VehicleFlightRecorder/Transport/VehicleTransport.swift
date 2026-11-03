import Foundation

enum TransportState: String { case disconnected, discovering, connecting, ready, failed }
enum VehicleTransportError: LocalizedError {
    case unavailableInBootstrap, notConnected, failed(String)
    var errorDescription: String? {
        switch self {
        case .unavailableInBootstrap: return "Physical accessories are unavailable in Simulator."
        case .notConnected: return "Accessory streams are not ready."
        case .failed(let message): return message
        }
    }
}

// Called only on the transport run loop. Deliberately no arbitrary-send API.
protocol VehicleTransport: AnyObject {
    var state: TransportState { get }
    func connect() throws
    func disconnect()
    func requestIdentity() throws
}

protocol TransportCapture: AnyObject {
    func receive(_ data: Data) throws
    func identity() throws -> Data
    func accepted(_ data: Data) throws
    func event(_ kind: String, _ detail: String) throws
    func close(_ reason: String) throws
}
protocol TransportChannel: AnyObject {
    var hasSpace: Bool { get }
    func write(_ data: Data) -> Int
    func close()
}

// A synchronous byte pump on one dedicated run loop: no unbounded async stream,
// no retained inbound queue. Each <=4 KiB read reaches Go/disk before the next.
// The only outbound queue is one Go-owned <=64 KiB buffer. No replay on failure.
final class TransportPump {
    private(set) var state: TransportState = .disconnected
    private(set) var failure: String?
    private(set) var generation = 0
    private(set) var pending = Data()
    private var capture: TransportCapture?
    private var channel: TransportChannel?
    private var deadline: TimeInterval = 0
    private var identityDeadline: TimeInterval?
    private var receivedSinceIdentity = false
    private var nextConnect: TimeInterval = 0
    private var failures = 0
    var now: () -> TimeInterval = { TimeInterval(TransportClock.nanoseconds()) / 1_000_000_000 }

    var reconnectReady: Bool { now() >= nextConnect }

    // Mark attempts that can fail before streams exist (for example EASession
    // creation). Each such failure must advance backoff, even after a prior fail.
    func prepareConnectionAttempt() throws {
        guard state != .connecting && state != .ready else { throw VehicleTransportError.failed("A connection is already active.") }
        guard reconnectReady else { throw VehicleTransportError.failed("Reconnect backoff is active.") }
        state = .discovering
        failure = nil
    }

    func begin(channel: TransportChannel, capture: TransportCapture) throws {
        guard state != .connecting && state != .ready else { channel.close(); try capture.close("duplicate connection rejected"); return }
        guard now() >= nextConnect else { channel.close(); try capture.close("reconnect backoff"); throw VehicleTransportError.failed("Reconnect backoff is active.") }
        generation += 1; self.channel = channel; self.capture = capture
        state = .connecting; failure = nil; deadline = now() + 10
    }
    func ready(generation: Int) {
        guard generation == self.generation, state == .connecting else { return }
        do { try capture?.event("streams_ready", "Both native streams opened"); state = .ready }
        catch { fail(error.localizedDescription) }
    }
    func receive(_ data: Data, generation: Int) {
        guard generation == self.generation, state == .ready || state == .connecting else { return }
        guard !data.isEmpty, data.count <= 65536 else { fail("Inbound chunk exceeds 64 KiB budget"); return }
        do { try capture?.receive(Data(data)); if identityDeadline != nil { receivedSinceIdentity = true } }
        catch { fail(error.localizedDescription) }
    }
    func identify() throws {
        guard state == .ready, let capture else { throw VehicleTransportError.notConnected }
        let bytes = try capture.identity()
        guard !bytes.isEmpty, bytes.count <= 65536, pending.isEmpty else { fail("Outbound budget exceeded"); throw VehicleTransportError.failed("Outbound budget exceeded") }
        pending = Data(bytes); deadline = now() + 5; identityDeadline = now() + 10; receivedSinceIdentity = false
        flush(generation: generation)
    }
    func flush(generation: Int) {
        guard generation == self.generation, state == .ready, let channel else { return }
        if !pending.isEmpty && now() >= deadline { fail("Write deadline exceeded; partial delivery uncertain"); return }
        // Bounded iterations; zero progress waits for a later space event/deadline.
        for _ in 0..<16 {
            guard !pending.isEmpty, channel.hasSpace else { return }
            let n = channel.write(pending)
            guard n >= 0, n <= pending.count else { fail("Native stream write failed; delivery uncertain"); return }
            if n == 0 { return }
            let accepted = Data(pending.prefix(n))
            do { try capture?.accepted(accepted) } catch { fail("Write occurred but evidence failed: \(error.localizedDescription)"); return }
            pending.removeFirst(n)
        }
    }
    func tick() {
        if state == .connecting && now() >= deadline { fail("Stream open deadline exceeded") }
        if state == .ready && !pending.isEmpty && now() >= deadline { fail("Write deadline exceeded; partial delivery uncertain") }
        if let limit = identityDeadline, now() >= limit {
            identityDeadline = nil
            if !receivedSinceIdentity {
                do { try capture?.event("identity_timeout", "No bytes received within 10 seconds; no retry") }
                catch { fail(error.localizedDescription) }
            }
        }
    }
    func lifecycle(_ state: String) {
        guard capture != nil else { return }
        do { try capture?.event(state, "Native scene transition") } catch { fail(error.localizedDescription) }
    }
    func fail(_ message: String) {
        guard state != .failed else { return }
        failure = String(message.prefix(512)); failures += 1
        nextConnect = now() + min(30, pow(2, Double(min(failures, 4))))
        do { try capture?.event("error", failure!) } catch { failure! += "; evidence error: \(error.localizedDescription)" }
        end("failed"); state = .failed
    }
    func disconnect() { end("disconnected"); state = .disconnected }
    private func end(_ reason: String) {
        channel?.close(); channel = nil; pending.removeAll(); identityDeadline = nil
        do { try capture?.close(reason) } catch { failure = [failure, error.localizedDescription].compactMap { $0 }.joined(separator: "; ") }
        capture = nil
    }
}
