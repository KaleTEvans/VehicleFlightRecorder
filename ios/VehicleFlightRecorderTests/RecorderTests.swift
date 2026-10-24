import XCTest
@testable import VehicleFlightRecorder

final class RecorderTests: XCTestCase {
    private func makeService(_ name: String) throws -> RecorderService {
        let pack = try XCTUnwrap(Bundle.main.url(forResource: "VehiclePack", withExtension: nil))
        let docs = try FileManager.default.url(for: .documentDirectory, in: .userDomainMask, appropriateFor: nil, create: true)
        return try RecorderService(pack: pack, sessions: docs.appendingPathComponent("NativeAcceptance/\(name)/Sessions", isDirectory: true))
    }
    func testNativeHealthyManualAndVoltageExports() async throws {
        for scenario in ["healthy_drive", "voltage_tcm_dropout"] {
            let service = try makeService(UUID().uuidString)
            try await service.start(scenario)
            var before = try await service.snapshot()
            XCTAssertTrue(before.active)
            XCTAssertEqual(before.physicalAdapterState, "not_connected")
            do { _ = try await service.archive(before.session!.id); XCTFail("Active export allowed") } catch {}
            for step in 1...288 {
                try await service.advance()
                if step == 30, scenario == "healthy_drive" { _ = try await service.mark() }
            }
            before = try await service.snapshot()
            XCTAssertEqual(before.state, "complete")
            XCTAssertEqual(before.incidents.count, 1)
            let incident = try XCTUnwrap(before.incidents.first)
            XCTAssertEqual(incident.status, "captured")
            XCTAssertEqual(incident.triggers.count, scenario == "healthy_drive" ? 1 : 2)
            let page = try await service.incident(session: incident.sessionId, id: incident.id, offset: 0)
            XCTAssertEqual(page.timeline.count, 200)
            XCTAssertTrue(page.hasMore)
            XCTAssertEqual(page.timeline.map(\.reference.recordSequence), page.timeline.map(\.reference.recordSequence).sorted())
            let zip = try await service.archive(incident.sessionId)
            XCTAssertEqual(zip.pathExtension, "zip")
            XCTAssertTrue(FileManager.default.fileExists(atPath: zip.path))
            let documents = try FileManager.default.url(for: .documentDirectory, in: .userDomainMask, appropriateFor: nil, create: true)
            let output = documents.appendingPathComponent("NativeAcceptance/Exports", isDirectory: true)
            try FileManager.default.createDirectory(at: output, withIntermediateDirectories: true)
            try FileManager.default.copyItem(at: zip, to: output.appendingPathComponent(zip.lastPathComponent))
            let rows = try await service.sessions()
            XCTAssertEqual(rows.count, 1)
            XCTAssertEqual(rows[0].session.incidentCount, 1)
            XCTAssertGreaterThan(rows[0].sizeBytes, 0)
        }
    }
    @MainActor
    func testViewModelLifecycleAndDuplicateMarkGuard() async throws {
        let model = RecorderModel(service: try makeService(UUID().uuidString))
        XCTAssertFalse(model.canMark)
        await model.start()
        XCTAssertNil(model.error)
        XCTAssertTrue(model.canMark)
        async let first: Void = model.mark()
        async let second: Void = model.mark()
        _ = await (first, second)
        XCTAssertEqual(model.snapshot?.incidents.count, 1)
        XCTAssertEqual(model.snapshot?.incidents.first?.triggers.count, 1)
        await model.stop()
        XCTAssertFalse(model.canMark)
        XCTAssertEqual(model.snapshot?.incidents.first?.status, "partial")
        await model.mark()
        XCTAssertEqual(model.snapshot?.incidents.first?.triggers.count, 1)
    }
    func testExportFailureIsVisible() throws {
        let path = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        XCTAssertThrowsError(try SessionExporter.archive(directory: path, output: path))
    }
}
