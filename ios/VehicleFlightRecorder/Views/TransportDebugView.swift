import SwiftUI

struct TransportDebugView: View {
    @StateObject private var model = TransportDebugModel()
    @Environment(\.scenePhase) private var phase
    @State private var enableIdentity = false
    @State private var showExport = false
    var body: some View {
        NavigationStack {
            List {
                Section("PHYSICAL ADAPTER / TRANSPORT TEST") {
                    Text("Vehicle Flight Recorder").font(.headline)
                    Text("Transport: \(model.state)")
                    Text(model.notice)
                    Text("Vehicle recording and polling become available in later milestones.").font(.caption)
                    if let error = model.error { Text(error).foregroundStyle(.red).textSelection(.enabled) }
                    Button("Discover / Connect") { Task { await model.connect() } }.disabled(model.busy)
                    Button("Disconnect") { Task { await model.disconnect() } }.disabled(model.busy)
                }
                Section("Developer identity check") {
                    Toggle("Enable adapter firmware read", isOn: $enableIdentity)
                    Text("One firmware identity read per connection. Keep the vehicle parked. This does not verify vehicle communication.").font(.caption)
                    Button("Read adapter identity once") { Task { await model.identify() } }
                        .disabled(!enableIdentity || model.state != "ready" || model.busy)
                }
                Section("Raw connection evidence") {
                    Button("Export closed connection capture") { Task { await model.share(); showExport = model.export != nil } }
                        .disabled(model.busy || model.state == "ready" || model.state == "connecting")
                    Text("JSONL log retains full bytes. Preview shows the latest 32 records, up to 256 bytes each. Stream acceptance does not establish adapter receipt.").font(.caption)
                    Text(model.raw).font(.system(.caption, design: .monospaced)).textSelection(.enabled)
                }
            }
            .navigationTitle("Adapter Transport")
            .sheet(isPresented: $showExport) { if let url = model.export { SessionShareSheet(url: url) } }
            .task { await model.connect(); while !Task.isCancelled { await model.refresh(); try? await Task.sleep(for: .milliseconds(500)) } }
            .onChange(of: phase) { _, value in
                Task { await model.lifecycle(value == .active ? "foreground" : value == .background ? "background" : "inactive") }
            }
        }
    }
}
