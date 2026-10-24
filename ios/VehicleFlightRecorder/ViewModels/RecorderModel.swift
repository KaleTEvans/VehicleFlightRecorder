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
    var canMark: Bool { snapshot?.active == true && !busy && service != nil }
    func refresh(advance: Bool) async {
        guard let service, !busy, !refreshing else { return }
        refreshing = true
        defer { refreshing = false }
        do {
            if advance, snapshot?.active == true { try await service.advance() }
            snapshot = try await service.snapshot()
            refreshCount += 1
            if refreshCount % 8 == 0 || !advance { sessions = try await service.sessions() }
        } catch { self.error = error.localizedDescription; snapshot = try? await service.snapshot() }
    }
    func start() async {
        guard !busy, let service else { return }
        busy = true
        defer { busy = false }
        do {
            try await service.start(scenario)
            snapshot = try await service.snapshot()
            sessions = try await service.sessions()
            notice = nil; error = nil
        } catch { self.error = error.localizedDescription }
    }
    func stop() async {
        guard !busy, let service else { return }
        busy = true
        defer { busy = false }
        do {
            try await service.stop()
            snapshot = try await service.snapshot()
            sessions = try await service.sessions()
            notice = "Drive saved. Open Sessions to review or export."
        } catch { self.error = error.localizedDescription }
    }
    func mark() async {
        guard canMark, let service else { return }
        busy = true
        defer { busy = false }
        do {
            _ = try await service.mark()
            snapshot = try await service.snapshot()
            notice = "Incident marked. Capturing available evidence before and after this moment."
        } catch { self.error = error.localizedDescription }
    }
    func export(_ id: String) async {
        guard !busy, let service else { return }
        busy = true
        defer { busy = false }
        do { shareURL = try await service.archive(id) }
        catch { self.error = error.localizedDescription }
    }
}
