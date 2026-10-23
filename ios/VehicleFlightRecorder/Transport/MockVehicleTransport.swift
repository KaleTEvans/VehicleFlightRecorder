import Foundation

/// The default transport is deliberately inert until mock scenarios are added.
final class MockVehicleTransport: VehicleTransport {
    let state: TransportState = .disconnected
    let incomingBytes = AsyncThrowingStream<Data, Error> { continuation in
        continuation.finish()
    }

    func connect() async throws {
        throw VehicleTransportError.unavailableInBootstrap
    }

    func disconnect() {}

    func write(_ data: Data) async throws {
        throw VehicleTransportError.notConnected
    }
}
