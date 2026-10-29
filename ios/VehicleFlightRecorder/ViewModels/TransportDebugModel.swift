import Foundation

@MainActor final class TransportDebugModel: ObservableObject {
    @Published var state = "disconnected"
    @Published var notice = "Ready for physical adapter testing"
    @Published var raw = "No connection capture"
    @Published var error: String?
    @Published var export: URL?
    @Published var busy = false
    private let executor = TransportExecutor()
    private let transport: MXPlusExternalAccessoryTransport
    init() {
        let root = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0].appendingPathComponent("ConnectionCaptures", isDirectory: true)
        transport = MXPlusExternalAccessoryTransport(root: root, executor: executor)
    }
    deinit {
        let transport = self.transport
        executor.submit { transport.disconnect() }
    }
    func refresh() async { await run { _ in } }
    func connect() async { await run { try $0.connect() } }
    func disconnect() async { await run { $0.disconnect() } }
    func identify() async { await run { try $0.requestIdentity() } }
    func lifecycle(_ phase: String) async {
        let transport = self.transport
        if !executor.submit({ transport.pump.lifecycle(phase) }) { error = "Lifecycle event mailbox overflow" }
    }
    func share() async {
        await run { transport in
            guard transport.state != .ready, transport.state != .connecting, let capture = transport.capture else { throw VehicleTransportError.failed("Disconnect before exporting the current capture.") }
            let source = URL(fileURLWithPath: capture.core.path())
            let output = FileManager.default.temporaryDirectory.appendingPathComponent(source.lastPathComponent)
            if FileManager.default.fileExists(atPath: output.path) { try FileManager.default.removeItem(at: output) }
            try FileManager.default.copyItem(at: source, to: output)
            Task { @MainActor [weak self] in self?.export = output }
        }
    }
    private func run(_ action: @escaping (MXPlusExternalAccessoryTransport) throws -> Void) async {
        guard !busy else { return }; busy = true
        let transport = self.transport
        await withCheckedContinuation { (continuation: CheckedContinuation<Void, Never>) in
            let admitted = executor.submit {
                var failure: String?
                do { try action(transport) } catch { failure = error.localizedDescription }
                let state = transport.state.rawValue
                let notice = transport.notice
                let raw = transport.capture?.core.snapshot() ?? "No capture"
                let pumpFailure = transport.pump.failure
                Task { @MainActor [weak self] in
                    if let self {
                        self.state = state; self.notice = notice; self.raw = raw
                        if let failure = failure ?? pumpFailure { self.error = failure }
                        self.busy = false
                    }
                    continuation.resume()
                }
            }
            if !admitted { error = "Transport command mailbox is full"; busy = false; continuation.resume() }
        }
    }
}
