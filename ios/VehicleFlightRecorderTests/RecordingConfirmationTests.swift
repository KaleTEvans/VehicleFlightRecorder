import XCTest
@testable import VehicleFlightRecorder

@MainActor private final class FakeRecordingNotifications: RecordingNotificationAdapter {
    var status = "authorized"
    var requests = 0
    var submissions: [String] = []
    var fail = false
    var onSettings: (() -> Void)?
    func authorization() async -> String { onSettings?(); return status }
    func requestAuthorization() async throws -> Bool { requests += 1; status = "authorized"; return true }
    func submit(_ fact: AutomaticStartFact) async throws {
        submissions.append(fact.sessionId)
        if fail { throw VehicleTransportError.failed("Injected notification error") }
    }
}
private final class MemoryConfirmationReceipts: ConfirmationReceipts {
    var keys = Set<String>()
    func claim(_ session: String, stage: String) throws -> Bool { keys.insert(session + stage).inserted }
}
@MainActor final class RecordingConfirmationTests: XCTestCase {
    private let fact = AutomaticStartFact(sessionId: "live-11111111111111111111111111111111", startOrigin: "automatic", timeUnixMs: 1_700_000_000_000, vehicleName: "Vehicle", coverage: "limited_standard_obd")
    private func make(_ adapter: FakeRecordingNotifications, receipts: ConfirmationReceipts = MemoryConfirmationReceipts()) -> RecordingConfirmation {
        let suite = UserDefaults(suiteName: "VFR-confirmation-test-\(UUID().uuidString)")!
        let value = RecordingConfirmation(adapter: adapter, receipts: receipts, preferences: suite)
        value.enabled = true; value.isSessionActive = { _ in true }
        return value
    }
    func testForegroundAndDuplicateHaveOneInAppPath() async {
        let adapter = FakeRecordingNotifications()
        let controller = make(adapter)
        controller.isForeground = { true }
        await controller.consume(fact); await controller.consume(fact)
        XCTAssertNotNil(controller.banner); XCTAssertTrue(adapter.submissions.isEmpty)
    }
    func testBackgroundFailureAndRelaunchNeverRetry() async {
        let adapter = FakeRecordingNotifications(); adapter.fail = true
        let receipts = MemoryConfirmationReceipts()
        let first = make(adapter, receipts: receipts); first.isForeground = { false }
        await first.consume(fact); await first.consume(fact)
        let relaunched = make(adapter, receipts: receipts); relaunched.isForeground = { false }
        await relaunched.consume(fact)
        XCTAssertEqual(adapter.submissions, [fact.sessionId]); XCTAssertNil(first.banner)
    }
    func testDisabledDeniedAndEndedConsumeWithoutPermissionPrompt() async {
        for reason in ["disabled", "denied", "not_determined", "ended"] {
            let adapter = FakeRecordingNotifications(); adapter.status = reason
            let controller = make(adapter); controller.isForeground = { false }
            if reason == "disabled" { controller.enabled = false }
            if reason == "ended" { controller.isSessionActive = { _ in false } }
            await controller.consume(fact)
            controller.enabled = true; adapter.status = "authorized"; controller.isSessionActive = { _ in true }
            await controller.consume(fact)
            XCTAssertTrue(adapter.submissions.isEmpty, reason); XCTAssertEqual(adapter.requests, 0)
        }
    }
    func testSceneChangeDuringSettingsChoosesOnlyForeground() async {
        let adapter = FakeRecordingNotifications(), controller = make(FakeRecordingNotifications())
        let actual = make(adapter); var visible = false
        actual.isForeground = { visible }; adapter.onSettings = { visible = true }
        await actual.consume(fact)
        XCTAssertNotNil(actual.banner); XCTAssertTrue(adapter.submissions.isEmpty); XCTAssertNil(controller.banner)
    }
    func testManualStartAndInvalidSessionHaveNoConfirmation() async {
        let adapter = FakeRecordingNotifications(), receipts = MemoryConfirmationReceipts()
        let controller = make(adapter, receipts: receipts)
        await controller.consume(AutomaticStartFact(sessionId: fact.sessionId, startOrigin: "manual", timeUnixMs: fact.timeUnixMs, vehicleName: "Vehicle", coverage: "unknown"))
        await controller.consume(AutomaticStartFact(sessionId: "../other", startOrigin: "automatic", timeUnixMs: fact.timeUnixMs, vehicleName: "Vehicle", coverage: "unknown"))
        XCTAssertTrue(receipts.keys.isEmpty); XCTAssertTrue(adapter.submissions.isEmpty)
    }
    func testDurableExclusiveReceiptSurvivesNewStore() throws {
        let root = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let first = FileConfirmationReceipts(directory: root)
        XCTAssertTrue(try first.claim(fact.sessionId, stage: "selected"))
        XCTAssertFalse(try FileConfirmationReceipts(directory: root).claim(fact.sessionId, stage: "selected"))
        XCTAssertThrowsError(try first.claim("../other", stage: "selected"))
    }
    func testPermissionOnlyRequestedByExplicitSetupAndNeverForDenied() async {
        let adapter = FakeRecordingNotifications(); adapter.status = "not_determined"
        let controller = make(adapter)
        await controller.enableWithPermission(); await controller.enableWithPermission()
        XCTAssertEqual(adapter.requests, 1)
        adapter.status = "denied"; await controller.enableWithPermission()
        XCTAssertEqual(adapter.requests, 1)
    }
    func testTapRoutingRechecksActiveEndedColdAndMissingOriginalID() {
        let original = fact.sessionId
        let other = "live-22222222222222222222222222222222"
        XCTAssertEqual(RecordingRouteResolution.resolve(sessionID: original, activeID: original, active: true, savedIDs: []), .active)
        XCTAssertEqual(RecordingRouteResolution.resolve(sessionID: original, activeID: other, active: true, savedIDs: [original]), .saved)
        XCTAssertEqual(RecordingRouteResolution.resolve(sessionID: original, activeID: nil, active: false, savedIDs: [original]), .saved)
        XCTAssertEqual(RecordingRouteResolution.resolve(sessionID: original, activeID: other, active: true, savedIDs: []), .unavailable)
        XCTAssertEqual(RecordingRouteResolution.resolve(sessionID: "../unrelated", activeID: original, active: true, savedIDs: []), .unavailable)
    }

}
