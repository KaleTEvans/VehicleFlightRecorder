#if DEBUG
import SwiftUI
import VehicleFlightRecorderCore

enum AutomaticFixture {
    struct Result {
        let root: URL
        let directory: URL
        let sessionID: String
        let claimCount: Int
        let probe: Bool
        let startFact: AutomaticStartFact?
        let completion: AutomaticEndFact?
    }
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
        var startFact: AutomaticStartFact?, completion: AutomaticEndFact?
        let metadata = "{\"protocol\":\"com.obdlink\",\"name\":\"Explicit native M7 fixture\"}"
        let setup: Set<String> = ["STPC\r", "ATD\r", "ATE0\r", "ATL0\r", "ATM0\r", "ATS1\r", "ATH1\r", "ATD0\r", "ATCAF1\r", "ATCFC1\r", "ATR1\r", "ATAT0\r", "STP34\r", "STPTO500\r", "ATSH18DB33F1\r", "STCAF0\r", "ATCRA\r", "STFA\r"]
        let lastStep = ["scanner_disconnected", "rpm_unavailable", "engine_stopped"].contains(kind) ? 3450 : 2900
        for step in 0...lastStep {
            let now = Int64(step) * 100_000_000
            try automatic.tick(base + now, wallMS: anchor + now / 1_000_000)
            let end = automatic.claimCompletion(&error)
            if let error { throw error }
            if !end.isEmpty {
                guard completion == nil else { throw VehicleTransportError.failed("Duplicate drive completion") }
                completion = try decodeCore(AutomaticEndFact.self, end)
            }
            guard let core = automatic.live() else { continue }
            if core.path() != previous {
                try core.connected(metadata, nativeNS: base + now, generation: &generation)
                try core.nativeEvent(generation, nativeNS: base + now, kind: "streams_ready", detail: "Native M7 fixture")
                if kind == "startup_stopped" { try core.receive(generation, nativeNS: base + now, data: Data("old reply\r>".utf8)) }
                previous = core.path()
            }
            if ["reconnect", "scanner_disconnected"].contains(kind), step == 2400 { try core.disconnected(generation, nativeNS: base + now, reason: "Scripted M7 loss"); continue }
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
                        if kind == "rpm_unavailable", step >= 2400 {
                            reply = "NO DATA\r>"
                        } else {
                            let rpm = kind == "engine_off" || step < 250 || (kind == "engine_stopped" && step >= 2400) ? "00 00" : "0E D8"
                            reply = "18DAF110 04 41 0C \(rpm)\r18DAF117 04 41 0C \(rpm)\r>"
                        }
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
            if !fact.isEmpty {
                claims += 1
                startFact = try decodeCore(AutomaticStartFact.self, fact)
            }
        }
        let stopMs = Int64(lastStep + 10) * 100
        try automatic.setEnabled(false, wallMS: anchor + stopMs, nativeNS: base + stopMs * 1_000_000)
        let probe = kind == "engine_off"
        let directory = root.appendingPathComponent(probe ? "Automation/Probes" : "Sessions", isDirectory: true)
        let entries = try FileManager.default.contentsOfDirectory(at: directory, includingPropertiesForKeys: nil)
        guard let session = entries.first else { throw VehicleTransportError.failed("Missing native M7 evidence") }
        return Result(root: root, directory: directory, sessionID: session.lastPathComponent, claimCount: claims, probe: probe, startFact: startFact, completion: completion)
    }
}
struct AutomaticUITestHost: View {
    @State private var result: AutomaticFixture.Result?
    @State private var service: RecorderService?
    @State private var failure: String?
    @StateObject private var confirmations = RecordingConfirmation(
        preferences: UserDefaults(suiteName: "VFR-banner-fixture")!
    )
    var body: some View {
        NavigationStack {
            List {
                Text("Scripted M7 automatic drive · no physical vehicle")
                if let result {
                    Text("Successful start facts: \(result.claimCount)").accessibilityIdentifier("automatic-fixture-claims")
                }
                if let service { NavigationLink("Open automatic drive history") { PhysicalHistoryView(service: service) } }
                if let fact = result?.startFact {
                    NavigationLink("Open foreground banner fixture") {
                        Button("Present recording confirmation") {
                            Task {
                                confirmations.enabled = true
                                confirmations.isForeground = { true }
                                confirmations.isSessionActive = { $0 == fact.sessionId }
                                await confirmations.consume(fact)
                            }
                        }
                        .navigationTitle("Foreground fixture")
                    }
                }
                if let failure { Text(failure).accessibilityIdentifier("automatic-fixture-error") }
            }.navigationTitle("Automatic recording")
        }
        .modifier(RecordingBannerOverlay(confirmations: confirmations))
        .task {
            do { let value = try AutomaticFixture.make(kind: "running"); result = value; service = try RecorderService(history: value.directory) }
            catch { failure = error.localizedDescription }
        }
    }
}
#endif
