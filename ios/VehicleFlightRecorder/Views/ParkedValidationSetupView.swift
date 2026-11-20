import SwiftUI

// A restored monitoring preference describes an enabled validation mode, not a
// new observation that the vehicle is parked. Fresh arming still needs consent.
struct ParkedValidationSetupView: View {
    let monitoringEnabled: Bool
    @Binding var parked: Bool
    var body: some View {
        if monitoringEnabled {
            Text("Parked validation mode is enabled")
                .font(.headline).accessibilityIdentifier("parked-validation-active")
            Text("Hands-off monitoring stays enabled when you reopen the app. Disable monitoring to change setup; confirm the vehicle is parked before enabling it again.")
                .font(.caption)
        } else {
            Toggle("Vehicle is parked", isOn: $parked)
                .accessibilityIdentifier("parked-confirmation-toggle")
        }
    }
}
