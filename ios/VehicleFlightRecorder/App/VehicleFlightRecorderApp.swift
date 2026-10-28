import SwiftUI

@main
struct VehicleFlightRecorderApp: App {
    var body: some Scene {
        WindowGroup {
            #if DEBUG
            if ProcessInfo.processInfo.arguments.contains("--ui-test-evidence") {
                EvidenceUITestHost()
            } else {
                DashboardView()
            }
            #else
            DashboardView()
            #endif
        }
    }
}
