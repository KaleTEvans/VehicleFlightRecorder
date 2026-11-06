#if DEBUG
import SwiftUI

// Real native Go diagnostic sessions with scripted replies only. Explicit test
// launch; Release and normal Simulator roots never fabricate a physical session.
enum DiscoveryDiagnosticFixture {
    struct Result {
        let snapshot: LiveSnapshot
        let root: URL
        let sessionID: String
    }
    static func make(recovery: Bool, can29: Bool = false, timeoutRPM: Bool = false) throws -> Result {
        let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
        let root = docs.appendingPathComponent("NativeAcceptance/Discovery-\(UUID().uuidString)/Sessions", isDirectory: true)
        let base: Int64 = 1_000_000_000_000, anchor: Int64 = 1_700_000_000_000
        let session = try GoLiveSession(root: root, operatingState: "stationary_running", workload: can29 ? "can29_diagnostic" : "discovery_diagnostic", reference: "Scripted diagnostic \(recovery ? "recovery" : "no_data") fixture; no physical measurements", origin: "native-fixture", wallMS: anchor, nativeNS: base)
        let core = session.core
        var generation: Int64 = 0
        try core.connected("{\"protocol\":\"com.obdlink\",\"name\":\"Scripted diagnostic fixture\"}", nativeNS: base, generation: &generation)
        try core.nativeEvent(generation, nativeNS: base, kind: "streams_ready", detail: "Native fixture")
        let setup: Set<String> = ["STPC\r", "ATD\r", "ATE0\r", "ATL0\r", "ATM0\r", "ATS1\r", "ATH1\r", "ATD0\r", "ATCAF1\r", "ATCFC1\r", "ATR1\r", "ATAT0\r", "STP33\r", "STPTO500\r", "ATSH7DF\r", "STCAF0\r", "ATCRA\r", "STFA\r"]
        var attempts = 0
        var timedOut = false
        var late: (Int64, Data)?
        for step in 0...1100 {
            let now = Int64(step) * 100_000_000
            if let pending = late, now >= pending.0 {
                try core.receive(generation, nativeNS: base + now, data: pending.1)
                late = nil
            }
            let command = try GoLiveSession.command(core, nativeNS: base + now, wallMS: anchor + now / 1_000_000)
            if command.isEmpty { continue }
            let request = String(decoding: command, as: UTF8.self)
            var reply: String
            if setup.contains(request) || (can29 && ["STP34\r", "ATSH18DB33F1\r"].contains(request)) { reply = "OK\r>" }
            else {
                switch request {
                case "STPR\r": reply = can29 ? "34\r>" : "33\r>"
                case "STPBRR\r": reply = "500000\r>"
                case "STI\r": reply = "STN2255 v5.8.1\r>"
                case "ATCS\r": reply = "T:00 R:00\r>"
                case "0100\r":
                    attempts += 1
                    reply = recovery && attempts > 1 ? "7E8 06 41 00 08 10 00 00\r>" : "NO DATA\r>"
                case "010C\r": reply = "7E8 04 41 0C 0C 80\r>"
                case "0105\r": reply = "7E8 03 41 05 7B\r>"
                case "STVR\r": reply = "13.45\r>"
                default: throw VehicleTransportError.failed("Unreviewed native diagnostic request")
                }
            }
            if can29 {
                reply = reply.replacingOccurrences(of: "7E8", with: "18DAF110")
                if request == "0100\r" && recovery && attempts > 1 { reply = "18DAF110 06 41 00 08 10 00 00\r18DAF118 06 41 00 00 10 00 00\r>" }
                if request == "010C\r" { reply = "18DAF110 04 41 0C 0C 80\r18DAF118 04 41 0C 0F A0\r>" }
            }
            try core.accepted(generation, nativeNS: base + now + 1000, data: command)
            if timeoutRPM && request == "010C\r" && !timedOut {
                timedOut = true
                late = (now + 2_100_000_000, Data(reply.utf8))
                continue
            }
            let bytes = Data(reply.utf8), half = bytes.count / 2
            try core.receive(generation, nativeNS: base + now + 2000, data: Data(bytes.prefix(half)))
            try core.receive(generation, nativeNS: base + now + 3000, data: Data(bytes.dropFirst(half)))
        }
        let snapshot = try decodeCore(LiveSnapshot.self, core.snapshot())
        try core.stop(base + 111_000_000_000)
        let rows = try FileManager.default.contentsOfDirectory(at: root, includingPropertiesForKeys: nil)
        guard let id = rows.first?.lastPathComponent else { throw VehicleTransportError.failed("Missing diagnostic fixture") }
        return Result(snapshot: snapshot, root: root, sessionID: id)
    }
}
struct DiscoveryDiagnosticUITestHost: View {
    var can29 = false
    @State private var result: DiscoveryDiagnosticFixture.Result?
    @State private var error: String?
    var body: some View {
        NavigationStack {
            List {
                Text("Scripted native diagnostic fixture · no vehicle").font(.caption)
                if let result { LiveDiscoveryStatusView(live: result.snapshot) }
                if let error { Text(error).accessibilityIdentifier("diagnostic-fixture-error") }
                Button("Load supported reply fixture") { load(recovery: true) }
            }.navigationTitle("Discovery diagnostic")
        }.task { load(recovery: false) }
    }
    private func load(recovery: Bool) {
        do { result = try DiscoveryDiagnosticFixture.make(recovery: recovery, can29: can29); error = nil }
        catch { self.error = error.localizedDescription }
    }
}
#endif
