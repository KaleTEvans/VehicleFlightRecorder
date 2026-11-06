import XCTest
import VehicleFlightRecorderCore
@testable import VehicleFlightRecorder

final class LiveRecorderTests: XCTestCase {
    func testNativeLiveBridgeHistoryEvidenceAndZIP() async throws {
        let documents = try FileManager.default.url(for: .documentDirectory, in: .userDomainMask, appropriateFor: nil, create: true)
        let root = documents.appendingPathComponent("NativeAcceptance/Live/\(UUID().uuidString)/Sessions", isDirectory: true)
        let base: Int64 = 1_000_000_000_000, anchor: Int64 = 1_700_000_000_000
        let session = try GoLiveSession(root: root, operatingState: "stationary_running", workload: "combined", reference: "Native scripted transcript, not physical measurements", origin: "native-fixture", wallMS: anchor, nativeNS: base)
        let core = session.core
        let initial = try decodeCore(LiveSnapshot.self, core.snapshot())
        XCTAssertEqual(initial.state, "waiting_for_adapter")
        XCTAssertTrue(initial.capabilities.isEmpty)
        XCTAssertTrue(initial.readings.isEmpty)
        var generation: Int64 = 0
        try core.connected("{\"protocol\":\"com.obdlink\",\"name\":\"Native scripted fixture\"}", nativeNS: base, generation: &generation)
        // No request before native stream readiness.
        XCTAssertTrue(try GoLiveSession.command(core, nativeNS: base, wallMS: anchor).isEmpty)
        try core.nativeEvent(generation, nativeNS: base, kind: "streams_ready", detail: "Native fixture")
        let replies = ["STPC\r": "OK\r>", "ATD\r": "ATD\rOK\r>", "ATE0\r": "ATE0\rOK\r>", "ATL0\r": "OK\r>", "ATM0\r": "OK\r>", "ATS1\r": "OK\r>", "ATH1\r": "OK\r>", "ATD0\r": "OK\r>", "ATCAF1\r": "OK\r>", "ATCFC1\r": "OK\r>", "ATR1\r": "OK\r>", "ATAT0\r": "OK\r>", "STP33\r": "OK\r>", "STPTO500\r": "OK\r>", "STPR\r": "33\r>", "STI\r": "STN2255 v5.8.1\r>", "0100\r": "7E8 06 41 00 08 10 00 00\r7E9 06 41 00 00 10 00 00\r>", "010C\r": "7E8 04 41 0C 0C 80\r7E9 04 41 0C 0F A0\r>", "0105\r": "7E8 03 41 05 7B\r>", "STVR\r": "13.45\r>"]
        var incidentID = ""
        for step in 0...2100 {
            let now = Int64(step) * 100_000_000
            if step == 800 {
                var error: NSError?
                incidentID = core.mark("Native manual evidence", nativeNS: base + now, error: &error)
                if let error { throw error }
            }
            let command: Data = try GoLiveSession.command(core, nativeNS: base + now, wallMS: anchor + now / 1_000_000)
            guard !command.isEmpty else { continue }
            let request = String(decoding: command, as: UTF8.self)
            let response = Data(try XCTUnwrap(replies[request], "Unreviewed request: \(request)").utf8)
            try core.accepted(generation, nativeNS: base + now + 1000, data: Data(command.prefix(1)))
            try core.accepted(generation, nativeNS: base + now + 2000, data: Data(command.dropFirst()))
            let half = response.count / 2
            try core.receive(generation, nativeNS: base + now + 3000, data: Data(response.prefix(half)))
            try core.receive(generation, nativeNS: base + now + 4000, data: Data(response.dropFirst(half)))
        }
        try core.stop(base + 211_000_000_000)
        let snapshot = try decodeCore(LiveSnapshot.self, core.snapshot())
        XCTAssertEqual(snapshot.state, "complete")
        XCTAssertEqual(snapshot.readings.count, 4)
        XCTAssertEqual(snapshot.incidents.count, 1)
        XCTAssertEqual(snapshot.incidents[0].status, "captured")
        let history = try RecorderService(history: root)
        let rows = try await history.sessions()
        XCTAssertEqual(rows.count, 1)
        XCTAssertEqual(rows[0].session.source, "obd-live")
        let page = try await history.incident(session: rows[0].id, id: incidentID, offset: 0)
        XCTAssertEqual(page.incident.status, "captured")
        XCTAssertFalse(page.timeline.isEmpty)
        let archive = try await history.archive(rows[0].id)
        let output = documents.appendingPathComponent("NativeAcceptance/Exports", isDirectory: true)
        try FileManager.default.createDirectory(at: output, withIntermediateDirectories: true)
        try FileManager.default.copyItem(at: archive, to: output.appendingPathComponent(archive.lastPathComponent))
        try await history.close()
    }
    func testNativeDiagnosticNoDataAndRecoveryExports() async throws {
        for recovery in [false, true] {
            let result = try DiscoveryDiagnosticFixture.make(recovery: recovery)
            let snapshot = result.snapshot
            XCTAssertTrue(snapshot.isDiagnostic)
            XCTAssertTrue(snapshot.initialized)
            let discovery = try XCTUnwrap(snapshot.discovery)
            XCTAssertEqual(discovery.setupComplete, 21)
            XCTAssertEqual(discovery.setupTotal, 21)
            XCTAssertGreaterThanOrEqual(discovery.attempts, 3)
            XCTAssertEqual(discovery.outcome, recovery ? "ok" : "no_data")
            XCTAssertEqual(discovery.canStatus, "T:00 R:00")
            XCTAssertEqual(snapshot.readings.contains { $0.signalId == "powertrain.engine.rpm" }, recovery)
            XCTAssertEqual(snapshot.readings.contains { $0.signalId == "powertrain.coolant.temperature" }, recovery)
            XCTAssertTrue(snapshot.readings.contains { $0.signalId == "electrical.adapter_voltage" })
            let service = try RecorderService(history: result.root)
            let archive = try await service.archive(result.sessionID)
            let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
            let output = docs.appendingPathComponent("NativeAcceptance/Exports", isDirectory: true)
            try FileManager.default.createDirectory(at: output, withIntermediateDirectories: true)
            try FileManager.default.copyItem(at: archive, to: output.appendingPathComponent(archive.lastPathComponent))
            try await service.close()
        }
    }

    func testNativeAdapterOnlyHandoffExports() async throws {
        for kind in ["retained", "unknown", "timeout"] {
            let result = try AdapterHandoffFixture.make(kind: kind)
            XCTAssertEqual(result.snapshot.state, "complete")
            XCTAssertFalse(result.snapshot.active)
            XCTAssertTrue(result.snapshot.isHandoff)
            XCTAssertTrue(result.snapshot.readings.isEmpty)
            XCTAssertTrue(result.snapshot.capabilities.isEmpty)
            let state = try XCTUnwrap(result.snapshot.adapterState)
            XCTAssertEqual(state.captured, kind == "timeout" ? 0 : 3)
            if kind == "retained" { XCTAssertEqual(state.protocolNumber, "34") }
            if kind == "unknown" { XCTAssertEqual(state.protocolNumber, "AUTO"); XCTAssertEqual(state.baud, "?") }
            let service = try RecorderService(history: result.root)
            let rows = try await service.sessions()
            XCTAssertEqual(rows.first?.session.profileId, "adapter-state-only")
            let archive = try await service.archive(result.sessionID)
            let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
            let output = docs.appendingPathComponent("NativeAcceptance/Exports", isDirectory: true)
            try FileManager.default.createDirectory(at: output, withIntermediateDirectories: true)
            try FileManager.default.copyItem(at: archive, to: output.appendingPathComponent(archive.lastPathComponent))
            try await service.close()
        }
    }

    func testNativeCAN29NoDataRecoveryAndTimeoutExports() async throws {
        for kind in ["no_data", "recovery", "timeout"] {
            let result = try DiscoveryDiagnosticFixture.make(recovery: kind != "no_data", can29: true, timeoutRPM: kind == "timeout")
            let snapshot = result.snapshot
            XCTAssertTrue(snapshot.isDiagnostic)
            XCTAssertTrue(snapshot.initialized)
            XCTAssertEqual(snapshot.workload, "can29_diagnostic")
            XCTAssertEqual(snapshot.discovery?.setupComplete, 21)
            XCTAssertEqual(snapshot.discovery?.outcome, kind == "no_data" ? "no_data" : "ok")
            let ecuRows = snapshot.readings.filter { $0.ecu != "adapter" }
            if kind == "no_data" { XCTAssertTrue(ecuRows.isEmpty) }
            else {
                XCTAssertEqual(Set(ecuRows.map { $0.ecu }), Set(["18DAF110", "18DAF118"]))
                XCTAssertTrue(ecuRows.contains { $0.signalId == "powertrain.coolant.temperature" && $0.value == 83 })
                XCTAssertTrue(ecuRows.contains { $0.signalId == "powertrain.engine.rpm" && $0.value == 800 })
                XCTAssertTrue(ecuRows.contains { $0.signalId == "powertrain.engine.rpm" && $0.value == 1000 })
            }
            let service = try RecorderService(history: result.root)
            let rows = try await service.sessions()
            XCTAssertEqual(rows.first?.session.profileId, "generic-obd-can29-500")
            let archive = try await service.archive(result.sessionID)
            let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
            let output = docs.appendingPathComponent("NativeAcceptance/Exports", isDirectory: true)
            try FileManager.default.createDirectory(at: output, withIntermediateDirectories: true)
            try FileManager.default.copyItem(at: archive, to: output.appendingPathComponent(archive.lastPathComponent))
            try await service.close()
        }
    }

}
