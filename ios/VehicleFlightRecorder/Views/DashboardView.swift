import SwiftUI

struct DashboardView: View {
    private let transport: any VehicleTransport = MockVehicleTransport()

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            Text("Vehicle Flight Recorder")
                .font(.largeTitle)
            #if targetEnvironment(simulator)
            Text("MOCK / SIMULATOR")
                .font(.headline)
            #else
            Text("Development Mode / MOCK")
                .font(.headline)
            #endif
            Text("Transport: Mock / \(transport.state.rawValue)")
            Text("Recorder: Not Started")
        }
        .padding()
    }
}
