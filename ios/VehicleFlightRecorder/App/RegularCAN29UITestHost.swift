#if DEBUG
import SwiftUI

// Explicit test launch only. These are real native Go recordings with scripted
// bytes, never vehicle data. Exercise ordinary schedules, incidents and recovery.
enum RegularCAN29Fixture {
    struct Result {
        let root: URL
        let sessionID: String
        let snapshot: LiveSnapshot
        let clearedOnReconnect: Bool
    }
    static func make(kind: String) throws -> Result {
        let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
        let root = docs.appendingPathComponent("NativeAcceptance/Regular-\(kind)-\(UUID().uuidString)/Sessions", isDirectory: true)
        let workload = ["rpm_only", "coolant_only", "voltage_only"].contains(kind) ? kind : "combined"
        let base: Int64 = 1_000_000_000_000, anchor: Int64 = 1_700_000_000_000
        let session = try GoLiveSession(root: root, operatingState: "stationary_running", workload: workload, reference: "Scripted ordinary CAN29 \(kind); no vehicle/reference measurement", protocolProfile: StationaryProtocol.can29.rawValue, origin: "native-fixture", wallMS: anchor, nativeNS: base)
        let core = session.core
        var generation: Int64 = 0
        let metadata = "{\"protocol\":\"com.obdlink\",\"name\":\"Scripted ordinary CAN29 fixture\"}"
        try core.connected(metadata, nativeNS: base, generation: &generation)
        try core.nativeEvent(generation, nativeNS: base, kind: "streams_ready", detail: "Native fixture")
        let setup: Set<String> = ["STPC\r", "ATD\r", "ATE0\r", "ATL0\r", "ATM0\r", "ATS1\r", "ATH1\r", "ATD0\r", "ATCAF1\r", "ATCFC1\r", "ATR1\r", "ATAT0\r", "STP34\r", "STPTO500\r", "ATSH18DB33F1\r", "STCAF0\r", "ATCRA\r", "STFA\r"]
        let end = kind == "reconnect" ? 2800 : kind == "partial" ? 350 : 750
        var cleared = false
        for step in 0...end {
            let now = Int64(step) * 100_000_000
            if (kind == "reconnect" && step == 800) || (kind == "partial" && step == 200) {
                var error: NSError?
                _ = core.mark("Scripted ordinary CAN29 manual incident", nativeNS: base + now, error: &error)
                if let error { throw error }
            }
            if kind == "reconnect" && step == 2200 {
                try core.disconnected(generation, nativeNS: base + now, reason: "Scripted loss after complete incident window")
                let old = try decodeCore(LiveSnapshot.self, core.snapshot())
                guard old.readings.allSatisfy({ $0.quality == "unavailable" }) else { throw VehicleTransportError.failed("Disconnected readings shown fresh") }
                try core.connected(metadata, nativeNS: base + now, generation: &generation)
                let fresh = try decodeCore(LiveSnapshot.self, core.snapshot())
                cleared = fresh.readings.isEmpty && fresh.capabilities.isEmpty && !fresh.initialized
                try core.nativeEvent(generation, nativeNS: base + now, kind: "streams_ready", detail: "Reconnected scripted fixture")
            }
            let command = try GoLiveSession.command(core, nativeNS: base + now, wallMS: anchor + now / 1_000_000)
            if command.isEmpty { continue }
            let request = String(decoding: command, as: UTF8.self)
            let reply: String
            if setup.contains(request) { reply = "OK\r>" }
            else {
                switch request {
                case "STPR\r": reply = "34\r>"
                case "STPBRR\r": reply = "500000\r>"
                case "STI\r": reply = "STN2255 v5.8.1\r>"
                case "0100\r": reply = kind == "no_data" ? "NO DATA\r>" : "18DAF110 06 41 00 08 10 00 00\r18DAF117 06 41 00 08 10 00 00\r18DAF118 06 41 00 08 10 00 00\r>"
                case "010C\r": reply = "18DAF110 04 41 0C 0C 80\r18DAF117 04 41 0C 0E 10\r18DAF118 04 41 0C 0F A0\r>"
                case "0105\r": reply = "18DAF110 03 41 05 7B\r18DAF117 03 41 05 7A\r18DAF118 03 41 05 79\r>"
                case "STVR\r": reply = "13.45\r>"
                default: throw VehicleTransportError.failed("Unreviewed ordinary CAN29 request")
                }
            }
            try core.accepted(generation, nativeNS: base + now + 1000, data: command)
            let bytes = Data(reply.utf8), half = bytes.count / 2
            try core.receive(generation, nativeNS: base + now + 2000, data: Data(bytes.prefix(half)))
            try core.receive(generation, nativeNS: base + now + 3000, data: Data(bytes.dropFirst(half)))
        }
        try core.stop(base + Int64(end + 10) * 100_000_000)
        let snapshot = try decodeCore(LiveSnapshot.self, core.snapshot())
        let rows = try FileManager.default.contentsOfDirectory(at: root, includingPropertiesForKeys: nil)
        guard let id = rows.first?.lastPathComponent else { throw VehicleTransportError.failed("Missing ordinary CAN29 fixture") }
        return Result(root: root, sessionID: id, snapshot: snapshot, clearedOnReconnect: cleared)
    }
}
struct RegularCAN29UITestHost: View {
    @State private var result: RegularCAN29Fixture.Result?
    @State private var service: RecorderService?
    @State private var error: String?
    var body: some View {
        NavigationStack {
            List {
                Text("Scripted native ordinary recording · no vehicle").font(.caption)
                if let result {
                    LiveDiscoveryStatusView(live: result.snapshot)
                    Text("Manual incident: \(result.snapshot.incidents.first?.status ?? "missing")")
                        .accessibilityIdentifier("regular-manual-status")
                    Text("Connection generations: \(result.snapshot.generation ?? 0)")
                        .accessibilityIdentifier("regular-generations")
                }
                if let service { NavigationLink("Saved sessions, incidents and export") { PhysicalHistoryView(service: service) } }
                if let error { Text(error).accessibilityIdentifier("regular-fixture-error") }
            }.navigationTitle("Ordinary CAN29")
        }.task {
            do {
                let value = try RegularCAN29Fixture.make(kind: "reconnect")
                result = value
                service = try RecorderService(history: value.root)
            } catch { self.error = error.localizedDescription }
        }
    }
}
#endif
