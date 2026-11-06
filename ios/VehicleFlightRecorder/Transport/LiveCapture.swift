import Foundation
import VehicleFlightRecorderCore

// One live session survives adapter generations. Only the transport executor
// calls this object; Swift never constructs diagnostic command bytes.
final class GoLiveSession {
    let core: MobileLiveRecorder
    init(root: URL, operatingState: String, workload: String, reference: String, origin: String = "physical", wallMS: Int64? = nil, nativeNS: Int64? = nil, requestedBridgeVersion: Int64 = RecorderService.supportedBridgeVersion) throws {
        guard MobileBridgeVersion() == requestedBridgeVersion, requestedBridgeVersion == RecorderService.supportedBridgeVersion else {
            throw BridgeFailure(category: "unsupported_version", operation: "initializeLive", detail: "Native/core bridge versions are incompatible")
        }
        let options = ["origin": origin, "operating_state": operatingState, "workload": workload, "reference_note": reference]
        let json = String(decoding: try JSONSerialization.data(withJSONObject: options), as: UTF8.self)
        var error: NSError?
        guard let value = MobileNewLiveRecorder(root.path, json, wallMS ?? Self.wall(), nativeNS ?? TransportClock.nanoseconds(), &error) else {
            throw error ?? VehicleTransportError.failed("Could not create live recorder")
        }
        core = value
    }
    static func wall() -> Int64 { Int64(Date().timeIntervalSince1970 * 1000) }
    static func command(_ core: MobileLiveRecorder, nativeNS: Int64, wallMS: Int64) throws -> Data {
        // gomobile imports NSData+NSError as throwing nonoptional Data; a nil
        // idle result is ambiguous. The Go string envelope preserves empty work.
        var error: NSError?
        let encoded = core.nextCommandBase64(nativeNS, wallMS: wallMS, error: &error)
        if let error { throw error }
        guard let bytes = Data(base64Encoded: encoded), bytes.count <= 65536 else {
            throw VehicleTransportError.failed("Invalid Go command envelope")
        }
        return bytes
    }
    func connect(metadata: [String: String]) throws -> GoLiveCapture {
        let json = String(decoding: try JSONSerialization.data(withJSONObject: metadata), as: UTF8.self)
        var generation: Int64 = 0
        try core.connected(json, nativeNS: TransportClock.nanoseconds(), generation: &generation)
        return GoLiveCapture(session: self, generation: generation)
    }
}
final class GoLiveCapture: TransportCapture {
    let session: GoLiveSession
    let generation: Int64
    init(session: GoLiveSession, generation: Int64) { self.session = session; self.generation = generation }
    func receive(_ data: Data) throws { try session.core.receive(generation, nativeNS: TransportClock.nanoseconds(), data: data) }
    func accepted(_ data: Data) throws { try session.core.accepted(generation, nativeNS: TransportClock.nanoseconds(), data: data) }
    func identity() throws -> Data { throw VehicleTransportError.failed("Live initialization owns adapter identification") }
    func event(_ kind: String, _ detail: String) throws {
        if ["foreground", "background", "inactive"].contains(kind) {
            try session.core.lifecycle(kind, nativeNS: TransportClock.nanoseconds())
        } else {
            try session.core.nativeEvent(generation, nativeNS: TransportClock.nanoseconds(), kind: kind, detail: String(detail.prefix(512)))
        }
    }
    func close(_ reason: String) throws { try session.core.disconnected(generation, nativeNS: TransportClock.nanoseconds(), reason: reason) }
}
struct LiveSnapshot: Decodable {
    let state: String
    let workload: String?
    let discovery: LiveDiscovery?
    let adapterState: AdapterStateReadback?
    var isHandoff: Bool { workload == "adapter_handoff" }
    var isDiagnostic: Bool { workload == "discovery_diagnostic" || workload == "can29_diagnostic" }
    let elapsedMs: Int64
    let initialized: Bool
    let error: String
    let readings: [LiveReading]
    let capabilities: [LiveCapability]
    let incidents: [IncidentInfo]
    var active: Bool { state != "complete" && state != "failed" }
}
struct LiveReading: Decodable, Identifiable {
    let signalId: String
    let ecu: String
    let value: Double
    let unit: String
    let quality: String
    let ageMs: Int64
    var id: String { signalId + "/" + ecu }
}
struct LiveCapability: Decodable, Identifiable {
    let key: String
    let ecu: String
    let state: String
    var id: String { key + "/" + ecu }
}

struct LiveDiscovery: Decodable {
    let attempts: Int
    let outcome: String
    let adapterStatus: String
    let nextAttemptMs: Int64
    let canStatus: String
    let canStatusAgeMs: Int64
    let setupComplete: Int
    let setupTotal: Int
    var title: String {
        switch outcome {
        case "not_requested": return "Not requested for voltage-only recording"
        case "waiting_for_setup": return "Waiting for adapter setup"
        case "not_attempted": return "Not yet attempted"
        case "pending": return "Request in progress"
        case "ok": return "Response received"
        case "partial": return "Partial response received"
        case "no_data": return "No data returned"
        case "timeout": return "Request timed out"
        case "protocol_error": return "Unrecognized response"
        default: return outcome.replacingOccurrences(of: "_", with: " ").capitalized
        }
    }
}

struct AdapterStateReadback: Decodable {
    let protocolNumber: String
    let description: String
    let baud: String
    let captured: Int
    enum CodingKeys: String, CodingKey {
        case protocolNumber = "protocol"
        case description, baud, captured
    }
}
