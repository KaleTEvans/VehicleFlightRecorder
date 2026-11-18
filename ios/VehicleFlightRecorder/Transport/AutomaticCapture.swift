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
    var recording: Bool { !sessionId.isEmpty }
}
struct AutomaticStartFact: Codable {
    let sessionId: String
    let startOrigin: String
    let timeUnixMs: Int64
    let vehicleName: String
    let coverage: String
}

extension GoLiveSession {
    convenience init(automaticCore: MobileLiveRecorder) { self.init(core: automaticCore) }
}
