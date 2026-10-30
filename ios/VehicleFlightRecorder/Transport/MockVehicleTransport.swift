import Foundation

// Mock recording remains Go-owned. This inert seam never opens physical streams.
final class MockVehicleTransport: VehicleTransport {
    let state: TransportState = .disconnected
    func connect() throws { throw VehicleTransportError.unavailableInBootstrap }
    func disconnect() {}
    func requestIdentity() throws { throw VehicleTransportError.notConnected }
}
