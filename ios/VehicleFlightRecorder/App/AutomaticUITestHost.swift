#if DEBUG
import SwiftUI
import VehicleFlightRecorderCore

enum AutomaticFixture {
    struct Result { let root: URL; let directory: URL; let sessionID: String; let claimCount: Int; let probe: Bool }
    static func make(kind: String) throws -> Result {
        let documents = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
        let root = documents.appendingPathComponent("NativeAcceptance/Automatic-\(kind)-\(UUID().uuidString)", isDirectory: true)
        var error: NSError?
        guard let automatic = MobileNewAutomaticRecorder(root.path, "native-fixture", StationaryProtocol.can29.rawValue, &error) else { throw error ?? VehicleTransportError.failed("Missing native automatic fixture") }
        defer { try? automatic.close() }
        let base: Int64 = 1_000_000_000_000, anchor: Int64 = 1_700_000_000_000
        try automatic.setEnabled(true, wallMS: anchor, nativeNS: base)
        var previous: String?, generation: Int64 = 0, claims = 0
        var stopped = Set<String>()
        let metadata = "{\"protocol\":\"com.obdlink\",\"name\":\"Explicit native M7 fixture\"}"
        let setup: Set<String> = ["STPC\r", "ATD\r", "ATE0\r", "ATL0\r", "ATM0\r", "ATS1\r", "ATH1\r", "ATD0\r", "ATCAF1\r", "ATCFC1\r", "ATR1\r", "ATAT0\r", "STP34\r", "STPTO500\r", "ATSH18DB33F1\r", "STCAF0\r", "ATCRA\r", "STFA\r"]
        for step in 0...2900 {
            let now = Int64(step) * 100_000_000
            try automatic.tick(base + now, wallMS: anchor + now / 1_000_000)
            guard let core = automatic.live() else { throw VehicleTransportError.failed("No native activity engine") }
            if core.path() != previous {
                try core.connected(metadata, nativeNS: base + now, generation: &generation)
                try core.nativeEvent(generation, nativeNS: base + now, kind: "streams_ready", detail: "Native M7 fixture")
                if kind == "startup_stopped" { try core.receive(generation, nativeNS: base + now, data: Data("old reply\r>".utf8)) }
                previous = core.path()
            }
            if kind == "reconnect", step == 2400 { try core.disconnected(generation, nativeNS: base + now, reason: "Scripted M7 loss"); continue }
            if kind == "reconnect", step > 2400, step < 2500 { continue }
            if kind == "reconnect", step == 2500 {
                try core.connected(metadata, nativeNS: base + now, generation: &generation)
                try core.nativeEvent(generation, nativeNS: base + now, kind: "streams_ready", detail: "Native M7 restored")
            }
            if kind != "engine_off", step == (kind == "startup_stopped" ? 1100 : 1000) {
                _ = core.mark("Native M7 manual window", nativeNS: base + now, error: &error)
                if let error { throw error }
            }
            let bytes = try GoLiveSession.command(core, nativeNS: base + now, wallMS: anchor + now / 1_000_000)
            if !bytes.isEmpty {
                let request = String(decoding: bytes, as: UTF8.self)
                var reply: String
                if setup.contains(request) { reply = "OK\r>" }
                else {
                    switch request {
                    case "STPR\r": reply = "34\r>"
                    case "STPBRR\r": reply = "500000\r>"
                    case "STI\r": reply = "STN2255 v5.8.1\r>"
                    case "0100\r": reply = "18DAF110 06 41 00 08 10 00 00\r18DAF117 06 41 00 08 10 00 00\r>"
                    case "010C\r":
                        let rpm = kind == "engine_off" || step < 250 ? "00 00" : "0E D8"
                        reply = "18DAF110 04 41 0C \(rpm)\r18DAF117 04 41 0C \(rpm)\r>"
                    case "0105\r": reply = "18DAF110 03 41 05 78\r>"
                    case "STVR\r": reply = "13.45\r>"
                    default: throw VehicleTransportError.failed("Unexpected native M7 request")
                    }
                }
                if kind == "startup_stopped", request == "STPC\r", !stopped.contains(core.path()) { stopped.insert(core.path()); reply = "STOPPED\r\r>" }
                try core.accepted(generation, nativeNS: base + now + 1000, data: bytes)
                try core.receive(generation, nativeNS: base + now + 2000, data: Data(reply.utf8))
            }
            let fact = automatic.claimConfirmation(base + now + 3000, error: &error)
            if let error { throw error }
            if !fact.isEmpty { claims += 1 }
        }
        try automatic.setEnabled(false, wallMS: anchor + 291_000, nativeNS: base + 291_000_000_000)
        let probe = kind == "engine_off"
        let directory = root.appendingPathComponent(probe ? "Automation/Probes" : "Sessions", isDirectory: true)
        let entries = try FileManager.default.contentsOfDirectory(at: directory, includingPropertiesForKeys: nil)
        guard let session = entries.first else { throw VehicleTransportError.failed("Missing native M7 evidence") }
        return Result(root: root, directory: directory, sessionID: session.lastPathComponent, claimCount: claims, probe: probe)
    }
}
struct AutomaticUITestHost: View {
    @State private var result: AutomaticFixture.Result?
    @State private var service: RecorderService?
    @State private var failure: String?
    var body: some View {
        NavigationStack {
            List {
                Text("Scripted M7 automatic drive · no physical vehicle")
                if let result {
                    Text("Successful start facts: \(result.claimCount)").accessibilityIdentifier("automatic-fixture-claims")
                }
                if let service { NavigationLink("Open automatic drive history") { PhysicalHistoryView(service: service) } }
                if let failure { Text(failure).accessibilityIdentifier("automatic-fixture-error") }
            }.navigationTitle("Automatic recording")
        }.task {
            do { let value = try AutomaticFixture.make(kind: "running"); result = value; service = try RecorderService(history: value.directory) }
            catch { failure = error.localizedDescription }
        }
    }
}
#endif
