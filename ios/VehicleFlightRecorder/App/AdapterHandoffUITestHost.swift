#if DEBUG
import SwiftUI

enum AdapterHandoffFixture {
    static func make(kind: String) throws -> DiscoveryDiagnosticFixture.Result {
        let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
        let root = docs.appendingPathComponent("NativeAcceptance/Handoff-\(UUID().uuidString)/Sessions", isDirectory: true)
        let base: Int64 = 1_000_000_000_000, anchor: Int64 = 1_700_000_000_000
        let session = try GoLiveSession(root: root, operatingState: "stationary_running", workload: "adapter_handoff", reference: "Scripted adapter-only \(kind); unverified state, no vehicle", origin: "native-fixture", wallMS: anchor, nativeNS: base)
        let core = session.core
        var generation: Int64 = 0
        try core.connected("{\"protocol\":\"com.obdlink\",\"name\":\"Scripted handoff fixture\"}", nativeNS: base, generation: &generation)
        try core.nativeEvent(generation, nativeNS: base, kind: "streams_ready", detail: "Native fixture")
        var seen = 0
        let expected = ["STPR\r", "STPRS\r", "STPBRR\r"]
        for step in 0...200 {
            let now = Int64(step) * 100_000_000
            let bytes = try GoLiveSession.command(core, nativeNS: base + now, wallMS: anchor + now / 1_000_000)
            if bytes.isEmpty { continue }
            let request = String(decoding: bytes, as: UTF8.self)
            guard seen < expected.count, request == expected[seen] else { throw VehicleTransportError.failed("Unexpected handoff write") }
            seen += 1
            try core.accepted(generation, nativeNS: base + now + 1000, data: bytes)
            if kind == "timeout" { continue }
            let value: String
            switch request {
            case "STPR\r": value = kind == "unknown" ? "AUTO" : "34"
            case "STPRS\r": value = kind == "unknown" ? "AUTO" : "ISO 15765-4 CAN 29/500"
            default: value = kind == "unknown" ? "?" : "500000"
            }
            // Retain the echoed command: reference apps may leave echo enabled.
            let reply = Data((request + value + "\r>").utf8), half = reply.count / 2
            try core.receive(generation, nativeNS: base + now + 2000, data: Data(reply.prefix(half)))
            try core.receive(generation, nativeNS: base + now + 3000, data: Data(reply.dropFirst(half)))
        }
        let snapshot = try decodeCore(LiveSnapshot.self, core.snapshot())
        let rows = try FileManager.default.contentsOfDirectory(at: root, includingPropertiesForKeys: nil)
        guard let id = rows.first?.lastPathComponent else { throw VehicleTransportError.failed("Missing handoff fixture") }
        return DiscoveryDiagnosticFixture.Result(snapshot: snapshot, root: root, sessionID: id)
    }
}
struct AdapterHandoffUITestHost: View {
    @State private var result: DiscoveryDiagnosticFixture.Result?
    @State private var error: String?
    var body: some View {
        NavigationStack {
            List {
                Text("Scripted native adapter handoff · no vehicle").font(.caption)
                if let result { LiveDiscoveryStatusView(live: result.snapshot) }
                if let error { Text(error).accessibilityIdentifier("handoff-fixture-error") }
                Button("Show unknown settings fixture") { load(kind: "unknown") }
            }.navigationTitle("Adapter handoff capture")
        }.task { load(kind: "retained") }
    }
    private func load(kind: String) {
        do { result = try AdapterHandoffFixture.make(kind: kind); error = nil }
        catch { self.error = error.localizedDescription }
    }
}
#endif
