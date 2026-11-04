#if DEBUG
import SwiftUI

// Two real Go live-format recordings with scripted NO DATA discovery replies.
// Explicit UI-test launch only; no accessory or physical reading is simulated as fact.
enum HistoryExportFixture {
    static func make() throws -> RecorderService {
        let documents = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
        let root = documents.appendingPathComponent("NativeAcceptance/ExportUI-\(UUID().uuidString)/Sessions", isDirectory: true)
        let setup: Set<String> = ["STPC\r", "ATD\r", "ATE0\r", "ATL0\r", "ATM0\r", "ATS1\r", "ATH1\r", "ATD0\r", "ATCAF1\r", "ATCFC1\r", "ATR1\r", "ATAT0\r", "STP33\r", "STPTO500\r"]
        for (index, workload) in ["rpm_only", "coolant_only"].enumerated() {
            let base: Int64 = 1_000_000_000_000
            let anchor = Int64(1_700_000_000_000) + Int64(index) * 60_000
            let session = try GoLiveSession(root: root, operatingState: "stationary_running", workload: workload, reference: "Synthetic export regression: discovery NO DATA, zero decoded samples", origin: "native-fixture", wallMS: anchor, nativeNS: base)
            let core = session.core
            var generation: Int64 = 0
            try core.connected("{\"protocol\":\"com.obdlink\",\"name\":\"Scripted export fixture\"}", nativeNS: base, generation: &generation)
            try core.nativeEvent(generation, nativeNS: base, kind: "streams_ready", detail: "Scripted native fixture")
            for step in 0...540 {
                let now = Int64(step) * 100_000_000
                let command = try GoLiveSession.command(core, nativeNS: base + now, wallMS: anchor + now / 1_000_000)
                if command.isEmpty { continue }
                let request = String(decoding: command, as: UTF8.self)
                let response: String
                if setup.contains(request) { response = "OK\r>" }
                else if request == "STPR\r" { response = "33\r>" }
                else if request == "STI\r" { response = "STN2255 v5.8.1\r>" }
                else if request == "0100\r" { response = "NO DATA\r>" }
                else { throw VehicleTransportError.failed("Unexpected fixture request") }
                try core.accepted(generation, nativeNS: base + now + 1000, data: command)
                try core.receive(generation, nativeNS: base + now + 2000, data: Data(response.utf8))
            }
            try core.stop(base + 55_000_000_000)
        }
        return try RecorderService(history: root)
    }
}
struct HistoryExportUITestHost: View {
    @State private var service: RecorderService?
    @State private var error: String?
    var body: some View {
        NavigationStack {
            if let service {
                List {
                    NavigationLink("Saved sessions, incidents and export") { PhysicalHistoryView(service: service) }
                }.navigationTitle("Adapter Transport")
            } else if let error {
                Text(error).accessibilityIdentifier("history-export-fixture-error")
            } else { ProgressView("Preparing recorded export fixtures") }
        }.task {
            do { service = try HistoryExportFixture.make() }
            catch { self.error = error.localizedDescription }
        }
    }
}
#endif
