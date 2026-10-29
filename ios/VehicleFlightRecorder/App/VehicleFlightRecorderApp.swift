import SwiftUI

@main
struct VehicleFlightRecorderApp: App {
    var body: some Scene {
        WindowGroup {
            #if DEBUG
            if ProcessInfo.processInfo.arguments.contains("--ui-test-evidence") {
                EvidenceUITestHost()
            } else {
                RecorderRootView()
            }
            #else
            RecorderRootView()
            #endif
        }
    }
}

private struct RecorderRootView: View {
    var body: some View {
        #if targetEnvironment(simulator)
        DashboardView()
        #else
        TransportDebugView()
        #endif
    }
}
