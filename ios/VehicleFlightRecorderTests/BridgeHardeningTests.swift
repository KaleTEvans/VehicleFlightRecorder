import XCTest
@testable import VehicleFlightRecorder

final class BridgeHardeningTests: XCTestCase {
    private func service(at directory: URL? = nil) throws -> RecorderService {
        let pack = try XCTUnwrap(Bundle.main.url(forResource: "VehiclePack", withExtension: nil))
        return try RecorderService(pack: pack, sessions: directory ?? FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString))
    }
    func testCompatibilityRejectedBeforeStorageInitialization() throws {
        let missing = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        XCTAssertThrowsError(try RecorderService(pack: missing, sessions: missing, requestedBridgeVersion: 999)) {
            XCTAssertEqual(($0 as? BridgeFailure)?.category, "unsupported_version")
        }
        XCTAssertFalse(FileManager.default.fileExists(atPath: missing.path))
        XCTAssertThrowsError(try RecorderService(pack: missing, sessions: missing)) {
            XCTAssertEqual(($0 as? BridgeFailure)?.category, "missing_pack")
        }
    }
    func testStableErrorsAndTerminalClose() async throws {
        let recorder = try service()
        do { try await recorder.stop(); XCTFail("Idle stop succeeded") }
        catch { XCTAssertEqual((error as? BridgeFailure)?.category, "invalid_state") }
        do { try await recorder.start("unknown"); XCTFail("Unknown scenario succeeded") }
        catch { XCTAssertEqual((error as? BridgeFailure)?.category, "invalid_input") }
        try await recorder.start("healthy_drive")
        try await recorder.close()
        try await recorder.close()
        do { try await recorder.start("healthy_drive"); XCTFail("Closed service restarted") }
        catch { XCTAssertEqual((error as? BridgeFailure)?.category, "invalid_state") }
    }
    func testCancelledAndStaleRefreshCannotAdvanceNewDrive() async throws {
        let recorder = try service()
        try await recorder.start("healthy_drive")
        let old = try await recorder.snapshot().session!.id
        try await recorder.stop()
        try await recorder.start("healthy_drive")
        do { _ = try await recorder.refresh(sessionID: old); XCTFail("Old tick accepted") }
        catch { XCTAssertEqual((error as? BridgeFailure)?.category, "cancelled") }
        let current = try await recorder.snapshot().session!.id
        let cancelled = Task {
            withUnsafeCurrentTask { $0?.cancel() }
            return try await recorder.refresh(sessionID: current)
        }
        do { _ = try await cancelled.value; XCTFail("Cancelled tick accepted") }
        catch { XCTAssertEqual((error as? BridgeFailure)?.category, "cancelled") }
        let snapshot = try await recorder.snapshot()
        XCTAssertEqual(snapshot.elapsedMs, 0)
        try await recorder.stop()
    }
    @MainActor
    func testScenePauseErrorRecoveryAndInterruptedRelaunch() async throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let recorder = try service(at: directory)
        let model = RecorderModel(service: recorder)
        model.scenario = "unknown"
        await model.start()
        XCTAssertNotNil(model.error)
        await model.refresh(advance: false)
        XCTAssertNotNil(model.error, "A successful background refresh must not erase an action failure")
        model.scenario = "healthy_drive"
        await model.start()
        XCTAssertNil(model.error)
        model.setSceneActive(false)
        for _ in 0..<5 { await model.refresh(advance: true) }
        XCTAssertEqual(model.snapshot?.elapsedMs, 0)
        model.setSceneActive(true)
        await model.refresh(advance: true)
        XCTAssertEqual(model.snapshot?.elapsedMs, 2500)
        let id = try XCTUnwrap(model.snapshot?.session?.id)
        await model.shutdown()
        let reopened = try service(at: directory)
        let replacementModel = RecorderModel(service: reopened)
        await replacementModel.refresh(advance: false)
        let interrupted = try XCTUnwrap(replacementModel.sessions.first)
        XCTAssertEqual(interrupted.id, id)
        XCTAssertEqual(replacementModel.sessionState(interrupted.session), "Interrupted / incomplete")
        XCTAssertGreaterThan(interrupted.session.observationCount, 0)
        do { _ = try await reopened.archive(id); XCTFail("Incomplete export succeeded") }
        catch { XCTAssertEqual((error as? BridgeFailure)?.category, "invalid_state") }
        try await reopened.close()
    }
    func testMalformedAndOversizedBridgeResponsesAreVisible() {
        XCTAssertThrowsError(try decodeCore(SessionPage.self, "{bad}")) {
            XCTAssertEqual(($0 as? BridgeFailure)?.category, "corrupt_data")
        }
        XCTAssertThrowsError(try decodeCore(SessionPage.self, String(repeating: " ", count: 8 * 1024 * 1024 + 1))) {
            XCTAssertEqual(($0 as? BridgeFailure)?.category, "resource_limit")
        }
    }
    func testArchivePreflightRejectsOversizeWithoutReadingPayload() throws {
        let root = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: root) }
        try FileManager.default.createDirectory(at: root.appendingPathComponent("raw"), withIntermediateDirectories: true)
        for name in SessionExporter.files { FileManager.default.createFile(atPath: root.appendingPathComponent(name).path, contents: Data()) }
        let handle = try FileHandle(forWritingTo: root.appendingPathComponent("observations.jsonl"))
        try handle.truncate(atOffset: SessionExporter.maximumArchiveBytes + 1)
        try handle.close()
        XCTAssertThrowsError(try SessionExporter.archive(directory: root, output: root.appendingPathComponent("output"))) {
            XCTAssertEqual(($0 as? BridgeFailure)?.category, "resource_limit")
        }
    }
    @MainActor
    func testRefreshRecoversStoppedAndFailedCoreStateWithoutHidingFailure() async throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let recorder = try service(at: directory)
        let model = RecorderModel(service: recorder)
        await model.start()
        try await recorder.stop() // Core completes before the model consumes a queued tick.
        await model.refresh(advance: true)
        XCTAssertEqual(model.snapshot?.state, "complete")
        await model.start()
        let id = try XCTUnwrap(model.snapshot?.session?.id)
        let session = directory.appendingPathComponent(id)
        try FileManager.default.moveItem(at: session.appendingPathComponent("session.json"), to: session.appendingPathComponent("original-session.json"))
        try FileManager.default.createDirectory(at: session.appendingPathComponent("session.json"), withIntermediateDirectories: false)
        await model.stop() // Force the metadata rename to fail.
        XCTAssertNotNil(model.error)
        await model.refresh(advance: true)
        XCTAssertEqual(model.snapshot?.state, "failed")
        XCTAssertTrue(model.error?.localizedCaseInsensitiveContains("io failure") == true)
        await model.shutdown()
    }

}
