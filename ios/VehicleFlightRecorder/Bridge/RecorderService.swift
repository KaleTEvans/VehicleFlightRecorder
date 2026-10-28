import Foundation
import VehicleFlightRecorderCore

// Sole native owner. Each actor operation performs synchronous Go calls without
// suspension, so a tick's advance/snapshot cannot interleave with start or stop.
actor RecorderService {
    private let core: MobileController
    let sessionsDirectory: URL
    private var closed = false
    private var closeFailure: BridgeFailure?
    static let supportedBridgeVersion: Int64 = 1

    init(pack: URL, sessions: URL, requestedBridgeVersion: Int64 = RecorderService.supportedBridgeVersion) throws {
        guard MobileBridgeVersion() == requestedBridgeVersion, requestedBridgeVersion == Self.supportedBridgeVersion else {
            throw BridgeFailure(category: "unsupported_version", operation: "initialize", detail: "Native/core bridge versions are incompatible")
        }
        var error: NSError?
        guard let controller = MobileNewController(pack.path, sessions.path, &error) else {
            throw BridgeFailure.translated(error ?? NSError(domain: "Recorder", code: 1, userInfo: [NSLocalizedDescriptionKey: "Could not initialize Go recorder"]), operation: "initialize")
        }
        core = controller
        sessionsDirectory = sessions
    }
    deinit { try? core.close() }

    private func perform<T>(_ operation: String, _ body: () throws -> T) throws -> T {
        do {
            try Task.checkCancellation()
            guard !closed else { throw BridgeFailure(category: "invalid_state", operation: operation, detail: "Recorder is closed") }
            return try body()
        } catch { throw BridgeFailure.translated(error, operation: operation) }
    }
    private func response(_ call: (NSErrorPointer) -> String) throws -> String {
        var error: NSError?
        let result = call(&error)
        if let error { throw error }
        return result
    }
    private func currentSnapshot() throws -> CoreSnapshot { try decodeCore(CoreSnapshot.self, response { core.snapshot($0) }) }
    func snapshot() throws -> CoreSnapshot { try perform("snapshot") { try currentSnapshot() } }
    func start(_ scenario: String) throws { try perform("start") { try core.start(scenario) } }
    func advance() throws {
        try perform("advance") {
            let before = try currentSnapshot()
            guard let id = before.session?.id else { throw BridgeFailure(category: "invalid_state", operation: "advance", detail: "No active drive") }
            try core.advanceSession(id, milliseconds: 2500)
        }
    }
    func refresh(sessionID: String?) throws -> CoreSnapshot {
        try perform("refresh") {
            if let sessionID { try core.advanceSession(sessionID, milliseconds: 2500) }
            return try currentSnapshot()
        }
    }
    func stop() throws { try perform("stop") { try core.stop() } }
    func mark() throws -> String { try perform("mark") { try response { core.mark("Manual incident", error: $0) } } }
    func sessions() throws -> [StoredSession] { try perform("sessions") { try decodeCore([StoredSession].self, response { core.sessionsJSON($0) }) } }
    func sessionPage(offset: Int64 = 0) throws -> SessionPage {
        try perform("sessionPage") { try decodeCore(SessionPage.self, response { core.sessionsPageJSON(offset, error: $0) }) }
    }
    func incident(session: String, id: String, offset: Int64) throws -> IncidentPage {
        try perform("incident") { try decodeCore(IncidentPage.self, response { core.incidentJSON(session, incidentID: id, offset: offset, error: $0) }) }
    }
    func archive(_ id: String) throws -> URL {
        try perform("archive") {
            let path = try response { core.exportPath(id, error: $0) }
            return try SessionExporter.archive(directory: URL(fileURLWithPath: path), output: FileManager.default.temporaryDirectory.appendingPathComponent("Exports", isDirectory: true))
        }
    }
    func close() throws {
        if closed {
            if let closeFailure { throw closeFailure }
            return
        }
        closed = true
        do { try core.close() }
        catch {
            let failure = BridgeFailure.translated(error, operation: "close")
            closeFailure = failure
            throw failure
        }
    }
}
