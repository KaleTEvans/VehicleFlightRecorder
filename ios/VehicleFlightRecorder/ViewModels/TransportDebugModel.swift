import Foundation

@MainActor final class TransportDebugModel: ObservableObject {
    static let shared = TransportDebugModel()
    @Published var automatic: AutomaticSnapshot?
    private var restoredAutomatic = false
    func restoreAutomatic() async {
        guard !restoredAutomatic else { return }; restoredAutomatic = true
        let profile = UserDefaults.standard.string(forKey: "stationaryProtocolProfile") ?? StationaryProtocol.can29.rawValue
        await run { transport in
            transport.onAutomaticConfirmation = { json in
                Task { @MainActor in
                    do { let fact = try decodeCore(AutomaticStartFact.self, json); await RecordingConfirmation.shared.consume(fact) }
                    catch { await TransportDebugModel.shared.recordAutomaticEvent("confirmation_result", detail: "fact_decode_failed_no_retry") }
                }
            }
            try transport.restoreAutomatic(profile: profile)
            try transport.connect()
        }
    }
    func setAutomatic(_ enabled: Bool, profile: String) async {
        if !enabled { await recordAutomaticEvent("battery_sample", detail: NativeResourceSample.json()) }
        await run { try $0.setAutomatic(enabled, profile: profile) }
        if enabled { await recordAutomaticEvent("battery_sample", detail: NativeResourceSample.json()) }
    }
    func recordAutomaticEvent(_ kind: String, detail: String) async {
        let transport = self.transport
        if !executor.submit({ transport.automaticNativeEvent(kind, detail: detail) }) { error = "Automatic event mailbox overflow" }
    }
    func isAutomaticSessionActive(_ id: String) async -> Bool {
        let transport = self.transport
        return await withCheckedContinuation { continuation in
            if !executor.submit({
                let value = transport.automaticSnapshot.flatMap { try? decodeCore(AutomaticSnapshot.self, $0) }
                continuation.resume(returning: value?.sessionId == id && value?.enabled == true)
            }) { continuation.resume(returning: false) }
        }
    }
    @Published var state = "disconnected"
    @Published var notice = "Ready for physical adapter testing"
    @Published var raw = "No connection capture"
    @Published var error: String?
    @Published var busy = false
    @Published var canShareCapture = false
    @Published var live: LiveSnapshot?
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
    func startLive(operatingState: String, workload: String, reference: String, protocolProfile: String = "") async {
        await run { try $0.startLive(operatingState: operatingState, workload: workload, reference: reference, protocolProfile: protocolProfile) }
    }
    func stopLive() async { await run { try $0.stopLive() } }
    func markLive() async { await run { try $0.markLive() } }
    func identify() async { await run { try $0.requestIdentity() } }
    func lifecycle(_ phase: String) async {
        let transport = self.transport
        if !executor.submit({ transport.lifecycle(phase) }) {
            actionFailure = "Lifecycle event mailbox overflow"
            error = actionFailure
        }
    }
    func share() async -> URL? {
        guard !busy else { return nil }
        busy = true
        defer { busy = false }
        let transport = self.transport
        do {
            let output = try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<URL, Error>) in
                let admitted = executor.submit {
                    do {
                        guard transport.state != .ready, transport.state != .connecting, let capture = transport.capture else {
                            throw VehicleTransportError.failed("Disconnect before exporting the current capture.")
                        }
                        let source = URL(fileURLWithPath: capture.core.path())
                        let output = FileManager.default.temporaryDirectory.appendingPathComponent(source.lastPathComponent)
                        if FileManager.default.fileExists(atPath: output.path) { try FileManager.default.removeItem(at: output) }
                        try FileManager.default.copyItem(at: source, to: output)
                        continuation.resume(returning: output)
                    } catch { continuation.resume(throwing: error) }
                }
                if !admitted { continuation.resume(throwing: VehicleTransportError.failed("Transport command mailbox is full")) }
            }
            actionFailure = nil; error = nil
            return output
        } catch {
            actionFailure = error.localizedDescription; self.error = actionFailure
            return nil
        }
    }
    func exportAutomationBundle() async -> URL? {
        guard !busy else { return nil }; busy = true
        defer { busy = false }
        let transport = self.transport
        do {
            let output = try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<URL, Error>) in
                if !executor.submit({
                    do {
                        let state = transport.automaticSnapshot.flatMap { try? decodeCore(AutomaticSnapshot.self, $0) }
                        guard state?.enabled != true else { throw VehicleTransportError.failed("Disable hands-off monitoring before exporting diagnostics") }
                        if let live = transport.live, try decodeCore(LiveSnapshot.self, live.core.snapshot()).active { throw VehicleTransportError.failed("Stop recording before exporting diagnostics") }
                        let documents = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
                        let zip = try SessionExporter.automationBundle(documents: documents, output: FileManager.default.temporaryDirectory.appendingPathComponent("AutomationExports"))
                        continuation.resume(returning: zip)
                    } catch { continuation.resume(throwing: error) }
                }) { continuation.resume(throwing: VehicleTransportError.failed("Transport command mailbox is full")) }
            }
            actionFailure = nil; error = nil; return output
        } catch { actionFailure = error.localizedDescription; self.error = actionFailure; return nil }
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
                let canShareCapture = transport.capture != nil && transport.state != .ready && transport.state != .connecting
                let pumpFailure = transport.pump.failure
                let automatic = transport.automaticSnapshot.flatMap { try? decodeCore(AutomaticSnapshot.self, $0) }
                var live: LiveSnapshot?
                if let value = transport.live {
                    do { live = try decodeCore(LiveSnapshot.self, value.core.snapshot()) }
                    catch { failure = error.localizedDescription }
                }
                Task { @MainActor [weak self] in
                    if let self {
                        self.automatic = automatic
                        self.state = state; self.notice = notice; self.raw = raw; self.live = live
                        self.canShareCapture = canShareCapture
                        if marksBusy { self.actionFailure = failure }
                        self.error = self.actionFailure ?? automatic.flatMap { $0.error.isEmpty ? nil : $0.error } ?? live.flatMap { $0.error.isEmpty ? nil : $0.error } ?? pumpFailure
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
