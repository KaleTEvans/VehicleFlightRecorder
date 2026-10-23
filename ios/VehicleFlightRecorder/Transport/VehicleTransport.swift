import Foundation

enum TransportState: String {
    case disconnected = "Not Connected"
    case connecting = "Connecting"
    case connected = "Connected"
}

enum VehicleTransportError: Error {
    case unavailableInBootstrap
    case notConnected
}

/// Lifecycle and ordered, arbitrary byte chunks only. Parsing belongs in Go.
/// Future implementations must stop outbound traffic on uncertain connection state.
protocol VehicleTransport: AnyObject {
    var state: TransportState { get }
    var incomingBytes: AsyncThrowingStream<Data, Error> { get }
    func connect() async throws
    func disconnect()
    func write(_ data: Data) async throws
}
