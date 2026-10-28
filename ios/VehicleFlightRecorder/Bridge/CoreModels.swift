import Foundation

struct SessionInfo: Codable, Identifiable {
    let sessionId: String
    let startTimeUnixMs: Int64
    let endTimeUnixMs: Int64?
    let state: String
    let observationCount: Int64
    let eventCount: Int64
    let incidentCount: Int64
    var id: String { sessionId }
    var started: Date { Date(timeIntervalSince1970: Double(startTimeUnixMs) / 1000) }
    var duration: String {
        guard let end = endTimeUnixMs else { return "In progress / incomplete" }
        return String(format: "%02d:%02d", (end - startTimeUnixMs) / 60000, ((end - startTimeUnixMs) / 1000) % 60)
    }
}
struct Reading: Codable, Identifiable {
    let signalId: String
    let value: Double
    let unit: String
    let quality: String
    let timestampUnixMs: Int64
    var id: String { signalId }
    var display: String { quality == "valid" ? String(format: "%.1f %@", value, unit) : quality.capitalized }
}
struct EvidenceReference: Codable {
    let file: String
    let line: Int64
    let recordSequence: Int64
    let timestampUnixMs: Int64
}
struct TriggerInfo: Codable, Identifiable {
    let triggerId: String
    let triggerType: String
    let timestampUnixMs: Int64
    let severity: String
    let category: String
    let label: String?
    let evidence: EvidenceReference
    var id: String { triggerId }
}
struct CoverageGap: Codable, Identifiable {
    let startTimeUnixMs: Int64
    let endTimeUnixMs: Int64
    let reason: String
    var id: String { "\(startTimeUnixMs)-\(endTimeUnixMs)" }
}
struct IncidentInfo: Codable, Identifiable {
    let incidentId: String
    let sessionId: String
    let triggerTimeUnixMs: Int64
    let triggerType: String
    let severity: String
    let status: String
    let manual: Bool
    let preWindowSeconds: Double
    let postWindowSeconds: Double
    let availableStartUnixMs: Int64
    let availableEndUnixMs: Int64
    let windowEndUnixMs: Int64
    let triggers: [TriggerInfo]
    let coverageGaps: [CoverageGap]
    let incompleteReasons: [String]
    let relatedIncidentIds: [String]
    var id: String { incidentId }
    var title: String { triggerType.replacingOccurrences(of: "_", with: " ").capitalized }
    var capturedSeconds: Double { Double(availableEndUnixMs - availableStartUnixMs) / 1000 }
}
struct StoredSession: Codable, Identifiable {
    let session: SessionInfo
    let incidents: [IncidentInfo]
    let sizeBytes: Int64
    var id: String { session.id }
}
struct CoreSnapshot: Codable {
    let bridgeVersion: Int64
    let coreVersion: String
    let packVersion: String
    let state: String
    let elapsedMs: Int64
    let durationMs: Int64
    let logicalTimeUnixMs: Int64
    let session: SessionInfo?
    let signals: [Reading]
    let modules: [String: String]
    let incidents: [IncidentInfo]
    let mockTransportState: String
    let physicalAdapterState: String
    let error: String
    var active: Bool { state == "recording" }
    var elapsed: String { String(format: "%02d:%02d", elapsedMs / 60000, (elapsedMs / 1000) % 60) }
}
struct EvidenceRecord: Codable {
    let signalId: String?
    let eventType: String?
    let value: Double?
    let unit: String?
    let quality: String?
}
struct EvidenceEntry: Codable, Identifiable {
    let reference: EvidenceReference
    let record: EvidenceRecord
    var id: String { "\(reference.file):\(reference.line)" }
    var title: String { record.eventType ?? record.signalId ?? "Evidence" }
    var detail: String {
        if let value = record.value { return "\(value) \(record.unit ?? "") · \(record.quality ?? "unknown")" }
        return "Event · sequence \(reference.recordSequence)"
    }
}
struct IncidentPage: Codable {
    let incident: IncidentInfo
    let timeline: [EvidenceEntry]
    let triggerOffsets: [String: Int64]
    let totalRecords: Int64
    let nextOffset: Int64
    let hasMore: Bool
}
struct SessionPage: Codable {
    let sessions: [StoredSession]
    let totalSessions: Int64
    let nextOffset: Int64
    let hasMore: Bool
}
func decodeCore<T: Decodable>(_ type: T.Type, _ json: String) throws -> T {
    guard json.utf8.count <= 8 * 1024 * 1024 else {
        throw BridgeFailure(category: "resource_limit", operation: "decode", detail: "Bridge response exceeds 8 MiB")
    }
    do {
        let decoder = JSONDecoder()
        decoder.keyDecodingStrategy = .convertFromSnakeCase
        let value = try decoder.decode(type, from: Data(json.utf8))
        if let snapshot = value as? CoreSnapshot, snapshot.bridgeVersion != RecorderService.supportedBridgeVersion {
            throw BridgeFailure(category: "unsupported_version", operation: "decode", detail: "Unsupported snapshot bridge version")
        }
        return value
    } catch { throw BridgeFailure.translated(error, operation: "decode") }
}
