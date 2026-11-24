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
            transport.onAutomaticCompletion = { json in
                Task { @MainActor in
                    do { await RecordingConfirmation.shared.consumeCompletion(try decodeCore(AutomaticEndFact.self, json)) }
                    catch { await TransportDebugModel.shared.recordAutomaticEvent("confirmation_result", detail: "completion_decode_failed_no_retry") }
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
    func resumeAutomatic(_ sessionID: String) async {
        await run { try $0.resumeAutomatic(sessionID) }
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
    @Published var recoveryNotice: String?
    @Published var exportingDiagnostics = false
    @Published var live: LiveSnapshot?
    @Published var markingIncident = false
    @Published var incidentFeedback: String?
    @Published var markFeedbackSequence = 0
    private var markedSessionID: String?
    private var actionFailure: String?
    private let executor: TransportExecutor
    private let documents: URL
    private let transport: MXPlusExternalAccessoryTransport
    init(documents: URL? = nil, executor: TransportExecutor = TransportExecutor(), transport: MXPlusExternalAccessoryTransport? = nil) {
        let documents = documents ?? FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
        self.documents = documents; self.executor = executor
        self.transport = transport ?? MXPlusExternalAccessoryTransport(root: documents.appendingPathComponent("ConnectionCaptures", isDirectory: true), executor: executor)
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
    func markLive() async {
        guard !busy, !markingIncident else { return }
        markingIncident = true; incidentFeedback = nil
        await run { try $0.markLive() }
        markingIncident = false
        if actionFailure == nil {
            markedSessionID = live?.session?.id
            markFeedbackSequence += 1
            incidentFeedback = "Incident marked · Evidence capture in progress"
            let sequence = markFeedbackSequence
            Task { [weak self] in
                try? await Task.sleep(for: .seconds(2))
                if self?.markFeedbackSequence == sequence { self?.incidentFeedback = nil }
            }
        }
    }
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
        guard !busy else { return nil }; busy = true; exportingDiagnostics = true
        defer { busy = false; exportingDiagnostics = false }
        let transport = self.transport
        let documents = self.documents
        do {
            let output = try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<URL, Error>) in
                if !executor.submit({
                    do {
                        try transport.prepareDiagnosticExport()
                        transport.automaticNativeEvent("export_requested", detail: "Native streams stopped; preserve complete and incomplete originals")
                        let zip = try SessionExporter.automationBundle(documents: documents, output: FileManager.default.temporaryDirectory.appendingPathComponent("AutomationExports"))
                        transport.automaticNativeEvent("export_result", detail: "Diagnostic ZIP prepared; share/retrieval not yet confirmed")
                        continuation.resume(returning: zip)
                    } catch {
                        transport.automaticNativeEvent("export_result", detail: "Export failed: \(error.localizedDescription)")
                        continuation.resume(throwing: error)
                    }
                }) { continuation.resume(throwing: VehicleTransportError.failed("Transport command mailbox is full")) }
            }
            actionFailure = nil; error = nil
            busy = false
            await refresh()
            return output
        } catch {
            actionFailure = error.localizedDescription; self.error = actionFailure
            busy = false
            await refresh()
            return nil
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
                let canShareCapture = transport.capture != nil && transport.state != .ready && transport.state != .connecting
                // Historical pump errors remain in the evidence. A successfully
                // disarmed/exported owner must not redisplay one as a new error.
                let pumpFailure = transport.state == .failed ? transport.pump.failure : nil
                let automatic = transport.automaticSnapshot.flatMap { try? decodeCore(AutomaticSnapshot.self, $0) }
                var live: LiveSnapshot?
                if let value = transport.live {
                    do { live = try decodeCore(LiveSnapshot.self, value.core.snapshot()) }
                    catch { failure = error.localizedDescription }
                }
                Task { @MainActor [weak self] in
                    if let self {
                        if self.markedSessionID != live?.session?.id { self.incidentFeedback = nil }
                        self.automatic = automatic
                        self.state = state; self.notice = notice; self.raw = raw; self.live = live
                        self.canShareCapture = canShareCapture
                        if marksBusy { self.actionFailure = failure }
                        let recovering = automatic?.enabled == true && automatic?.error.isEmpty == true && (state == "failed" || live?.requiresReconnect == true)
                        self.recoveryNotice = recovering ? "Adapter synchronization interrupted; retrying a fresh connection. Coverage has a gap." : nil
                        self.error = self.actionFailure ?? automatic.flatMap { $0.error.isEmpty ? nil : $0.error } ?? (recovering ? nil : live.flatMap { $0.error.isEmpty ? nil : $0.error } ?? pumpFailure)
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
