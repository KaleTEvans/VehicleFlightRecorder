#if DEBUG
import SwiftUI
import VehicleFlightRecorderCore

// Exercises the real Go/native bridge with no adapter and explicitly synthetic
// origin. Four accepted STPC writes receive no bytes, exhausting synchronization.
enum FailedAutomaticFixture {
    struct Result { let root: URL; let owner: MobileAutomaticRecorder; let live: GoLiveSession; let original: URL }
    static func make() throws -> Result {
        let root = FileManager.default.temporaryDirectory.appendingPathComponent("M7-recovery-\(UUID().uuidString)", isDirectory: true)
        var error: NSError?
        guard let owner = MobileNewAutomaticRecorder(root.path, "native-fixture", StationaryProtocol.can29.rawValue, &error) else { throw error ?? VehicleTransportError.failed("Missing recovery owner") }
        let base = TransportClock.nanoseconds() - 120_000_000_000
        let anchor = GoLiveSession.wall() - 120_000
        try owner.setEnabled(true, wallMS: anchor, nativeNS: base)
        guard let core = owner.live() else { throw VehicleTransportError.failed("Missing recovery capture") }
        var generation: Int64 = 0
        for attempt in 0..<4 {
            let now = base + Int64(attempt) * 10_000_000_000
            try core.connected("{\"protocol\":\"com.obdlink\",\"name\":\"Scripted timeout fixture\"}", nativeNS: now, generation: &generation)
            try core.nativeEvent(generation, nativeNS: now, kind: "streams_ready", detail: "Scripted timeout")
            let bytes = try GoLiveSession.command(core, nativeNS: now + 1_000_000_000, wallMS: anchor + Int64(attempt) * 10_000 + 1000)
            guard bytes == Data("STPC\r".utf8) else { throw VehicleTransportError.failed("Missing scripted close request") }
            try core.accepted(generation, nativeNS: now + 1_000_001_000, data: bytes)
            _ = try? GoLiveSession.command(core, nativeNS: now + 4_000_000_000, wallMS: anchor + Int64(attempt) * 10_000 + 4000)
            _ = try? GoLiveSession.command(core, nativeNS: now + 6_000_000_000, wallMS: anchor + Int64(attempt) * 10_000 + 6000)
            if attempt < 3 { try core.disconnected(generation, nativeNS: now + 7_000_000_000, reason: "Scripted native channel closes before fresh generation") }
        }
        _ = try? owner.tick(base + 40_000_000_000, wallMS: anchor + 40_000)
        let state = try decodeCore(AutomaticSnapshot.self, owner.snapshot())
        guard !state.enabled, state.state == "failed", state.error.contains("synchronization retries") else { throw VehicleTransportError.failed("Fixture did not reach the reported failure") }
        let capture = try GoConnectionCapture(root: root.appendingPathComponent("ConnectionCaptures"), metadata: ["source": "native-fixture", "protocol": "com.obdlink"])
        try capture.close("Scripted diagnostic connection")
        return Result(root: root, owner: owner, live: GoLiveSession(automaticCore: core), original: URL(fileURLWithPath: core.path()).appendingPathComponent("session.json"))
    }
}
@MainActor struct AutomaticRecoveryUITestHost: View {
    @State private var model: TransportDebugModel?
    @State private var failure: String?
    var body: some View {
        Group {
            if let model { TransportDebugView(model: model) }
            else if let failure { Text(failure).accessibilityIdentifier("recovery-fixture-error") }
            else { ProgressView("Preparing scripted failed acquisition") }
        }.task {
            guard model == nil else { return }
            do {
                let value = try FailedAutomaticFixture.make()
                let executor = TransportExecutor()
                let transport = MXPlusExternalAccessoryTransport(root: value.root.appendingPathComponent("ConnectionCaptures"), executor: executor, automatic: value.owner, live: value.live)
                transport.pump.fail("Previous native acquisition failure")
                let model = TransportDebugModel(documents: value.root, executor: executor, transport: transport)
                await model.refresh(); self.model = model
            } catch { failure = error.localizedDescription }
        }
    }
}
#endif
