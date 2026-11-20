import SwiftUI

@main
struct VehicleFlightRecorderApp: App {
    @UIApplicationDelegateAdaptor(RecordingAppDelegate.self) private var appDelegate
    var body: some Scene {
        WindowGroup {
            #if DEBUG
            if ProcessInfo.processInfo.arguments.contains("--ui-test-evidence") {
                EvidenceUITestHost()
            } else if ProcessInfo.processInfo.arguments.contains("--ui-test-history-export") {
                HistoryExportUITestHost()
            } else if ProcessInfo.processInfo.arguments.contains("--ui-test-discovery") || ProcessInfo.processInfo.arguments.contains("--ui-test-can29") {
                DiscoveryDiagnosticUITestHost(can29: ProcessInfo.processInfo.arguments.contains("--ui-test-can29"))
            } else if ProcessInfo.processInfo.arguments.contains("--ui-test-regular-can29") {
                RegularCAN29UITestHost()
            } else if ProcessInfo.processInfo.arguments.contains("--ui-test-parked-setup") {
                ParkedValidationUITestHost()
            } else if ProcessInfo.processInfo.arguments.contains("--ui-test-automatic-recovery") {
                AutomaticRecoveryUITestHost()
            } else if ProcessInfo.processInfo.arguments.contains("--ui-test-automatic") {
                AutomaticUITestHost()
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
