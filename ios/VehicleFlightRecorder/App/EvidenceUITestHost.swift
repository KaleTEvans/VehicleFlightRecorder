#if DEBUG
import SwiftUI

// Explicit UI-test launch only. Generate a real Go incident in an isolated
// temporary directory; ordinary previews and application launches use Dashboard.
struct EvidenceUITestHost: View {
    @State private var service: RecorderService?
    @State private var incident: IncidentInfo?
    @State private var error: String?
    var body: some View {
        NavigationStack {
            if let incident, let service {
                IncidentDetailView(incident: incident, service: service)
            } else if let error {
                Text(error).accessibilityIdentifier("evidence-fixture-error")
            } else {
                ProgressView("Preparing real mock incident")
            }
        }.task {
            do {
                let pack = Bundle.main.url(forResource: "VehiclePack", withExtension: nil)!
                let directory = FileManager.default.temporaryDirectory.appendingPathComponent("EvidenceUITest-\(UUID().uuidString)")
                let recorder = try RecorderService(pack: pack, sessions: directory)
                try await recorder.start("voltage_tcm_dropout")
                for _ in 0..<288 { try await recorder.advance() }
                let snapshot = try await recorder.snapshot()
                guard let captured = snapshot.incidents.first else {
                    throw NSError(domain: "UITest", code: 1, userInfo: [NSLocalizedDescriptionKey: "Go fixture generated no incident"])
                }
                service = recorder
                incident = captured
            } catch { self.error = error.localizedDescription }
        }
    }
}
#endif
