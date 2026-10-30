import Combine
import XCTest
@testable import VehicleFlightRecorder

final class TransportDebugModelTests: XCTestCase {
    @MainActor func testPeriodicRefreshDoesNotDisableButtons() async {
        let model = TransportDebugModel()
        var busyChanges: [Bool] = []
        let subscription = model.$busy.dropFirst().sink { busyChanges.append($0) }
        defer { subscription.cancel() }

        for _ in 0..<4 { await model.refresh() }

        XCTAssertTrue(busyChanges.isEmpty, "Status polling must not flash disabled controls")
        XCTAssertFalse(model.busy)
        XCTAssertEqual(model.state, "disconnected")
    }

    @MainActor func testExplicitCommandStillDisablesControlsUntilComplete() async {
        let model = TransportDebugModel()
        var busyChanges: [Bool] = []
        let subscription = model.$busy.dropFirst().sink { busyChanges.append($0) }
        defer { subscription.cancel() }

        await model.disconnect()
        XCTAssertEqual(busyChanges, [true, false])
        busyChanges.removeAll()
        await model.refresh()
        XCTAssertTrue(busyChanges.isEmpty)
        XCTAssertFalse(model.busy)
    }
}
