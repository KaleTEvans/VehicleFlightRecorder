import XCTest
@testable import VehicleFlightRecorder

final class AutomaticRecorderTests: XCTestCase {
    func testNativeAutomaticEngineOffManualWindowAndRecoveryExports() async throws {
        for kind in ["running", "engine_off", "reconnect"] {
            let result = try AutomaticFixture.make(kind: kind)
            XCTAssertEqual(result.claimCount, result.probe ? 0 : 1)
            let history = try RecorderService(history: result.directory)
            let rows = try await history.sessions()
            XCTAssertEqual(rows.count, 1)
            let row = try XCTUnwrap(rows.first)
            XCTAssertEqual(row.session.live?.startOrigin, result.probe ? "activity_probe" : "automatic")
            XCTAssertEqual(row.session.live?.policyVersion, "rpm-activity-1")
            XCTAssertEqual(row.session.state, "complete")
            if !result.probe { XCTAssertEqual(row.incidents.first?.status, "captured") }
            let archive = try await history.archive(result.sessionID)
            let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
            let output = docs.appendingPathComponent("NativeAcceptance/Exports", isDirectory: true)
            try FileManager.default.createDirectory(at: output, withIntermediateDirectories: true)
            try FileManager.default.copyItem(at: archive, to: output.appendingPathComponent(archive.lastPathComponent))
            try await history.close()
        }
    }
}
