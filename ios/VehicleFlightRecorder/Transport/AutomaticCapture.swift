import Foundation
import VehicleFlightRecorderCore

struct AutomaticSnapshot: Decodable {
    let acceptingTelemetry: Bool?
    let enabled: Bool
    let state: String
    let error: String
    let sessionId: String
    let manualStopOverride: Bool
    let maximumDriveSeconds: Int
    let quiet: Bool?
    let resumeSessionId: String?
    let resumeReason: AutomaticEndReason?
    var recording: Bool { enabled && !sessionId.isEmpty }
}
struct AutomaticStartFact: Codable {
    let sessionId: String
    let startOrigin: String
    let timeUnixMs: Int64
    let vehicleName: String
    let coverage: String
}

enum AutomaticEndReason: String, Decodable {
    case disconnected = "scanner_disconnected"
    case rpmUnavailable = "rpm_unavailable"
    case engineStopped = "three_zero_rpm_replies"
    case executionGap = "owner_execution_gap_exceeded_grace"
    case durationLimit = "controlled_recording_duration_limit"

    var title: String {
        switch self {
        case .disconnected: "Scanner disconnected"
        case .rpmUnavailable: "Engine RPM unavailable"
        case .engineStopped: "Engine stopped"
        case .executionGap: "Recording interrupted"
        case .durationLimit: "Recording time limit reached"
        }
    }
}

struct AutomaticEndFact: Decodable {
    let sessionId: String
    let reason: AutomaticEndReason
}

extension GoLiveSession {
    convenience init(automaticCore: MobileLiveRecorder) { self.init(core: automaticCore) }
}
