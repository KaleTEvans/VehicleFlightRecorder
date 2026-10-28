import SwiftUI

@MainActor
final class RecorderModel: ObservableObject {
    @Published var snapshot: CoreSnapshot?
    @Published var sessions: [StoredSession] = []
    @Published var scenario = "healthy_drive"
    @Published var busy = false
    @Published var error: String?
    @Published var notice: String?
    @Published var shareURL: URL?
    @Published var hasMoreSessions = false
    private var nextHistoryOffset: Int64 = 0
    private var revision: UInt64 = 0
    private var sceneActive = true
    private var closed = false
    let service: RecorderService?
    let sessionLocation: String
    private var refreshCount = 0
    private var refreshing = false

    init(service: RecorderService? = nil) {
        do {
            if let service {
                self.service = service
                self.sessionLocation = "Test app container"
            } else {
                guard let pack = Bundle.main.url(forResource: "VehiclePack", withExtension: nil) else {
                    throw NSError(domain: "Recorder", code: 2, userInfo: [NSLocalizedDescriptionKey: "The bundled mock vehicle pack is missing"])
                }
                let directory = try FileManager.default.url(for: .documentDirectory, in: .userDomainMask, appropriateFor: nil, create: true).appendingPathComponent("Sessions", isDirectory: true)
                self.service = try RecorderService(pack: pack, sessions: directory)
                self.sessionLocation = directory.path
            }
        } catch {
            self.service = nil
            self.sessionLocation = "Unavailable"
            self.error = error.localizedDescription
        }
    }
    func sessionState(_ session: SessionInfo) -> String {
        if session.state == "recording" {
            return snapshot?.session?.id == session.id && snapshot?.active == true ? "Recording" : "Interrupted / incomplete"
        }
        return session.state.capitalized
    }
    var canMark: Bool { snapshot?.active == true && !busy && !closed && service != nil }
    func setSceneActive(_ active: Bool) {
        sceneActive = active
        revision &+= 1 // Invalidate responses from the previous scene task.
    }
    private func applyHistory(_ page: SessionPage, append: Bool = false) {
        if append { sessions += page.sessions }
        else {
            let updated = Set(page.sessions.map(\.id))
            // Keep older pages already shown while refreshing the first page.
            sessions = page.sessions + sessions.filter { !updated.contains($0.id) }
        }
        nextHistoryOffset = append ? page.nextOffset : max(page.nextOffset, Int64(sessions.count))
        hasMoreSessions = Int64(sessions.count) < page.totalSessions
    }
    func refresh(advance: Bool) async {
        guard let service, !busy, !refreshing, !closed, !advance || sceneActive, !Task.isCancelled else { return }
        refreshing = true
        let currentRevision = revision
        defer { refreshing = false }
        do {
            let value = try await service.refresh(sessionID: advance && snapshot?.active == true ? snapshot?.session?.id : nil)
            guard currentRevision == revision, !Task.isCancelled else { return }
            snapshot = value
            refreshCount += 1
            if refreshCount % 8 == 0 || !advance {
                let page = try await service.sessionPage()
                guard currentRevision == revision, !Task.isCancelled else { return }
                applyHistory(page)
            }
            error = value.error.isEmpty ? nil : BridgeFailure.translated(
                NSError(domain: "Recorder", code: 1, userInfo: [NSLocalizedDescriptionKey: value.error]), operation: "refresh").localizedDescription
        } catch {
            guard currentRevision == revision, !Task.isCancelled else { return }
            self.error = error.localizedDescription
        }
    }
    func loadMoreSessions() async {
        guard !busy, !closed, hasMoreSessions, let service else { return }
        busy = true
        revision &+= 1
        defer { busy = false }
        do { applyHistory(try await service.sessionPage(offset: nextHistoryOffset), append: true); error = nil }
        catch { self.error = error.localizedDescription }
    }
    func shutdown() async {
        guard !closed, let service else { return }
        closed = true; revision &+= 1
        do { try await service.close() }
        catch { self.error = error.localizedDescription }
    }
    func start() async {
        guard !busy, !closed, let service else { return }
        busy = true
        revision &+= 1
        defer { busy = false }
        do {
            try await service.start(scenario)
            snapshot = try await service.snapshot()
            applyHistory(try await service.sessionPage())
            notice = nil; error = nil
        } catch { self.error = error.localizedDescription }
    }
    func stop() async {
        guard !busy, !closed, let service else { return }
        busy = true
        revision &+= 1
        defer { busy = false }
        do {
            try await service.stop()
            snapshot = try await service.snapshot()
            applyHistory(try await service.sessionPage())
            error = nil
            notice = "Drive saved. Open Sessions to review or export."
        } catch { self.error = error.localizedDescription }
    }
    func mark() async {
        guard canMark, let service else { return }
        busy = true
        revision &+= 1
        defer { busy = false }
        do {
            _ = try await service.mark()
            snapshot = try await service.snapshot()
            error = nil
            notice = "Incident marked. Capturing available evidence before and after this moment."
        } catch { self.error = error.localizedDescription }
    }
    func export(_ id: String) async {
        guard !busy, !closed, let service else { return }
        busy = true
        revision &+= 1
        defer { busy = false }
        do { shareURL = try await service.archive(id); error = nil }
        catch { self.error = error.localizedDescription }
    }
}
