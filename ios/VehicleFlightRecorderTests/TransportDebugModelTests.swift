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

    #if targetEnvironment(simulator)
    @MainActor func testActionFailurePersistsUntilSuccessfulCommandClearsIt() async {
        let model = TransportDebugModel()
        await model.connect() // Physical discovery is unavailable in Simulator.
        XCTAssertNotNil(model.error)
        let failure = model.error
        await model.refresh()
        XCTAssertEqual(model.error, failure, "Polling must preserve explicit action failures")
        await model.disconnect()
        XCTAssertNil(model.error, "A successful recovery action clears the stale error")
        await model.refresh()
        XCTAssertNil(model.error)
    }
    #endif
}
