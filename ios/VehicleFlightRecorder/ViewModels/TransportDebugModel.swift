import Foundation

@MainActor final class TransportDebugModel: ObservableObject {
    @Published var state = "disconnected"
    @Published var notice = "Ready for physical adapter testing"
    @Published var raw = "No connection capture"
    @Published var error: String?
    @Published var export: URL?
    @Published var busy = false
    private var actionFailure: String?
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
    // A snapshot read must not disable controls on every half-second refresh.
    // The dedicated executor still serializes it with all transport commands.
    func refresh() async { await run(marksBusy: false) { _ in } }
    func connect() async { await run { try $0.connect() } }
    func disconnect() async { await run { $0.disconnect() } }
    func identify() async { await run { try $0.requestIdentity() } }
    func lifecycle(_ phase: String) async {
        let transport = self.transport
        if !executor.submit({ transport.pump.lifecycle(phase) }) {
            actionFailure = "Lifecycle event mailbox overflow"
            error = actionFailure
        }
    }
    func share() async {
        export = nil
        await run { transport in
            guard transport.state != .ready, transport.state != .connecting, let capture = transport.capture else { throw VehicleTransportError.failed("Disconnect before exporting the current capture.") }
            let source = URL(fileURLWithPath: capture.core.path())
            let output = FileManager.default.temporaryDirectory.appendingPathComponent(source.lastPathComponent)
            if FileManager.default.fileExists(atPath: output.path) { try FileManager.default.removeItem(at: output) }
            try FileManager.default.copyItem(at: source, to: output)
            Task { @MainActor [weak self] in self?.export = output }
        }
    }
    private func run(marksBusy: Bool = true, _ action: @escaping (MXPlusExternalAccessoryTransport) throws -> Void) async {
        guard !busy else { return }
        if marksBusy { busy = true }
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
                        if marksBusy { self.actionFailure = failure }
                        self.error = self.actionFailure ?? pumpFailure
                        // A pending refresh must not clear a command's busy state.
                        if marksBusy { self.busy = false }
                    }
                    continuation.resume()
                }
            }
            if !admitted {
                actionFailure = "Transport command mailbox is full"
                error = actionFailure
                if marksBusy { busy = false }
                continuation.resume()
            }
        }
    }
}
