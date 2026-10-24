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
    private func response(_ call: (NSErrorPointer) -> String) throws -> String {
        var error: NSError?
        let result = call(&error)
        if let error { throw error }
        return result
    }
    func snapshot() throws -> CoreSnapshot { try decodeCore(CoreSnapshot.self, response { core.snapshot($0) }) }
    func start(_ scenario: String) throws { try core.start(scenario) }
    func advance() throws { try core.advance(2500) }
    func stop() throws { try core.stop() }
    func mark() throws -> String { try response { core.mark("Manual incident", error: $0) } }
    func sessions() throws -> [StoredSession] { try decodeCore([StoredSession].self, response { core.sessionsJSON($0) }) }
    func incident(session: String, id: String, offset: Int64) throws -> IncidentPage {
        try decodeCore(IncidentPage.self, response { core.incidentJSON(session, incidentID: id, offset: offset, error: $0) })
    }
    func archive(_ id: String) throws -> URL {
        let path = try response { core.exportPath(id, error: $0) }
        return try SessionExporter.archive(directory: URL(fileURLWithPath: path), output: FileManager.default.temporaryDirectory.appendingPathComponent("Exports", isDirectory: true))
    }
}
