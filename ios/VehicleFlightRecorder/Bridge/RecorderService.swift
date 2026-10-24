import Foundation
import VehicleFlightRecorderCore

// This actor is the sole native owner of the Go controller. Calls never run on
// the main actor; the UI awaits each refresh before scheduling the next one.
actor RecorderService {
    private let core: MobileController
    let sessionsDirectory: URL

    init(pack: URL, sessions: URL) throws {
        var error: NSError?
        guard let controller = MobileNewController(pack.path, sessions.path, &error) else {
            throw error ?? NSError(domain: "Recorder", code: 1, userInfo: [NSLocalizedDescriptionKey: "Could not initialize Go recorder"])
        }
        core = controller
        sessionsDirectory = sessions
    }
    func snapshot() throws -> CoreSnapshot { try decodeCore(CoreSnapshot.self, core.snapshot()) }
    func start(_ scenario: String) throws { try core.start(scenario) }
    func advance() throws { try core.advance(2500) }
    func stop() throws { try core.stop() }
    func mark() throws -> String { try core.mark("Manual incident") }
    func sessions() throws -> [StoredSession] { try decodeCore([StoredSession].self, core.sessionsJSON()) }
    func incident(session: String, id: String, offset: Int64) throws -> IncidentPage {
        try decodeCore(IncidentPage.self, core.incidentJSON(session, incidentID: id, offset: offset))
    }
    func archive(_ id: String) throws -> URL {
        let path = try core.exportPath(id)
        return try SessionExporter.archive(directory: URL(fileURLWithPath: path), output: FileManager.default.temporaryDirectory.appendingPathComponent("Exports", isDirectory: true))
    }
}
