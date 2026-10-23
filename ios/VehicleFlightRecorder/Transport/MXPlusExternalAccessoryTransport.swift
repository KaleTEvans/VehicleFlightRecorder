import Foundation

/// Offline shell only; never discovers accessories, opens sessions, or sends bytes.
/// Later: EAAccessoryManager discovery, com.obdlink protocol validation, EASession
/// stream lifecycle, ordered byte forwarding, errors, and conservative reconnects.
/// All protocol commands and diagnostic meaning remain outside this transport.
final class MXPlusExternalAccessoryTransport: VehicleTransport {
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
