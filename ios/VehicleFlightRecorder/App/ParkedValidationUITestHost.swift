#if DEBUG
import SwiftUI

// The persisted Go preference is physically verified separately. This host
// exercises the real setup view with the reported restored on/local false state.
struct ParkedValidationUITestHost: View {
    @State private var monitoring = true
    @State private var parked = false
    var body: some View {
        NavigationStack {
            Form {
                Text("Scripted restored monitoring state · no physical vehicle")
                ParkedValidationSetupView(monitoringEnabled: monitoring, parked: $parked)
                Toggle("Hands-off monitoring", isOn: $monitoring)
                    .disabled(!monitoring && !parked)
                    .accessibilityIdentifier("automatic-monitoring-toggle")
            }.navigationTitle("Parked validation")
        }
    }
}
#endif
