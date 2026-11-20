import XCTest
@testable import VehicleFlightRecorder

final class AutomaticRecorderTests: XCTestCase {
    func testNativeAutomaticEngineOffManualWindowAndRecoveryExports() async throws {
        for kind in ["running", "engine_off", "reconnect", "startup_stopped"] {
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
            let bundles = docs.appendingPathComponent("NativeAcceptance/AutomationBundles", isDirectory: true)
            let bundle = try SessionExporter.automationBundle(documents: result.root, output: bundles)
            XCTAssertTrue(FileManager.default.fileExists(atPath: bundle.path))
            try await history.close()
        }
    }
    @MainActor func testFailedAcquisitionExportsOnFirstRequestAndPreservesOriginals() async throws {
        let value = try FailedAutomaticFixture.make()
        defer { try? value.owner.close(); try? FileManager.default.removeItem(at: value.root) }
        let original = try Data(contentsOf: value.original)
        let alias = FileManager.default.temporaryDirectory.appendingPathComponent("M7-root-alias-\(UUID().uuidString)")
        try FileManager.default.createSymbolicLink(at: alias, withDestinationURL: value.root)
        defer { try? FileManager.default.removeItem(at: alias) }
        let aliasZip = try SessionExporter.automationBundle(documents: alias, output: value.root.appendingPathComponent("AliasExports"))
        XCTAssertNotNil(try Data(contentsOf: aliasZip).range(of: original), "Canonical root spelling must preserve the same original bytes")

        let executor = TransportExecutor()
        let transport = MXPlusExternalAccessoryTransport(root: value.root.appendingPathComponent("ConnectionCaptures"), executor: executor, automatic: value.owner, live: value.live)
        let model = TransportDebugModel(documents: value.root, executor: executor, transport: transport)
        await model.refresh()
        XCTAssertEqual(model.automatic?.state, "failed")
        for _ in 0..<2 {
            let url = await model.exportAutomationBundle()
            let zip = try Data(contentsOf: XCTUnwrap(url))
            XCTAssertNotNil(zip.range(of: Data("ConnectionCaptures/".utf8)))
            XCTAssertNotNil(zip.range(of: original))
            XCTAssertNil(model.error)
            XCTAssertFalse(model.busy); XCTAssertFalse(model.exportingDiagnostics)
            XCTAssertEqual(try Data(contentsOf: value.original), original)
            XCTAssertFalse(try decodeCore(AutomaticSnapshot.self, value.owner.snapshot()).enabled)
        }
        try value.owner.setEnabled(true, wallMS: GoLiveSession.wall(), nativeNS: TransportClock.nanoseconds())
        XCTAssertNotEqual(value.owner.live()?.path(), value.live.core.path())
        XCTAssertEqual(try Data(contentsOf: value.original), original)
    }
    func testPrimaryReadingsSelectRealValidSourceWithoutRemovingEvidence() {
        let rows = [
            LiveReading(signalId: "powertrain.engine.rpm", ecu: "18DAF110", value: 0, unit: "rpm", quality: "stale", ageMs: 20_000),
            LiveReading(signalId: "powertrain.engine.rpm", ecu: "18DAF117", value: 960, unit: "rpm", quality: "valid", ageMs: 1000),
            LiveReading(signalId: "powertrain.engine.rpm", ecu: "18DAF118", value: 964, unit: "rpm", quality: "valid", ageMs: 1000),
            LiveReading(signalId: "powertrain.coolant.temperature", ecu: "18DAF110", value: 78, unit: "degC", quality: "valid", ageMs: 1000)]
        let selected = LiveReadingSelection.primary(rows)
        XCTAssertEqual(selected.count, 2)
        XCTAssertEqual(selected.first?.ecu, "18DAF117")
        XCTAssertEqual(selected.first?.value, 960)
        XCTAssertEqual(rows.count, 4)
        XCTAssertEqual(LiveReadingSelection.primary(Array(rows.reversed())).map(\.id), selected.map(\.id))
    }

}
