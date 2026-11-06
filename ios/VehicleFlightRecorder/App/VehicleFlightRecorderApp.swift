import SwiftUI

@main
struct VehicleFlightRecorderApp: App {
    var body: some Scene {
        WindowGroup {
            #if DEBUG
            if ProcessInfo.processInfo.arguments.contains("--ui-test-evidence") {
                EvidenceUITestHost()
            } else if ProcessInfo.processInfo.arguments.contains("--ui-test-history-export") {
                HistoryExportUITestHost()
            } else if ProcessInfo.processInfo.arguments.contains("--ui-test-discovery") {
                DiscoveryDiagnosticUITestHost()
            } else if ProcessInfo.processInfo.arguments.contains("--ui-test-handoff") {
                AdapterHandoffUITestHost()
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
