import SwiftUI

struct TransportDebugView: View {
    @StateObject private var model = TransportDebugModel()
    @Environment(\.scenePhase) private var phase
    @State private var enableIdentity = false
    @State private var share: SessionShareRequest?
    @State private var parked = false
    @State private var operatingState = "ignition_on_engine_off"
    @State private var workload = "combined"
    @State private var reference = ""
    var body: some View {
        NavigationStack {
            List {
                Section("PHYSICAL ADAPTER / TRANSPORT TEST") {
                    Text("Vehicle Flight Recorder").font(.headline)
                    Text("Transport: \(model.state)")
                    Text(model.notice)
                    if workload == "adapter_handoff" { Text("Adapter-only handoff capture · VFR leaves the vehicle protocol unchanged.").font(.caption) }
                    else if workload == "can29_diagnostic" { Text("Standard OBD stationary diagnostic · CAN 29-bit / 500 kbps. Enhanced Alfa / TCM signals are not enabled.").font(.caption) }
                    else { Text("Standard OBD stationary testing · generic CAN 11-bit / 500 kbps. Enhanced Alfa / TCM signals are not enabled.").font(.caption) }
                    if let error = model.error { Text(error).foregroundStyle(.red).textSelection(.enabled) }
                    Button("Discover / Connect") { Task { await model.connect() } }.disabled(model.busy)
                    Button("Disconnect") { Task { await model.disconnect() } }.disabled(model.busy)
                }
                Section("Stationary recording") {
                    Toggle("Vehicle is parked", isOn: $parked).disabled(model.live?.active == true)
                    Picker("Operating state", selection: $operatingState) {
                        Text("Ignition on / engine off").tag("ignition_on_engine_off")
                        Text("Engine running / stationary").tag("stationary_running")
                    }.disabled(model.live?.active == true)
                    Picker("Workload", selection: $workload) {
                        Text("Combined").tag("combined")
                        Text("Discovery diagnostic").tag("discovery_diagnostic")
                        Text("CAN 29-bit diagnostic").tag("can29_diagnostic")
                        Text("Adapter handoff capture").tag("adapter_handoff")
                        Text("RPM only").tag("rpm_only")
                        Text("Coolant only").tag("coolant_only")
                        Text("Adapter voltage only").tag("voltage_only")
                    }.disabled(model.live?.active == true)
                    if workload == "discovery_diagnostic" || workload == "can29_diagnostic" {
                        Text("Record about 2 minutes, then Stop and save. This mode records discovery attempts, adapter CAN status and voltage; RPM/coolant wait for valid support evidence. Keep this app open and the vehicle parked.").font(.caption)
                    }
                    TextField("Reference tool / reading (optional)", text: $reference)
                        .disabled(model.live?.active == true)
                    if workload == "adapter_handoff" {
                        Text("Capture the adapter state after using OBDLink. Leave the scanner powered. This mode sends only three read-only settings queries and saves automatically, normally within a few seconds; its active owner-loop limit is 15 seconds. App reconnection may already change the prior state.").font(.caption)
                    } else { Text("Explicit Start enables documented adapter setup and read-only OBD requests. Keep parked; each session ends after 10 minutes of elapsed native time when the owner loop runs. Background continuity is unverified.").font(.caption) }
                    Button(workload == "adapter_handoff" ? "Capture adapter settings" : "Start stationary recording") {
                        Task { await model.startLive(operatingState: operatingState, workload: workload, reference: String(reference.prefix(256))) }
                    }.disabled(!parked || model.busy || model.live?.active == true)
                    if let live = model.live {
                        Text("Recorder: \(live.state) · \(live.elapsedMs / 1000) s")
                        LiveDiscoveryStatusView(live: live)
                        if !live.isDiagnostic && !live.isHandoff {
                            Button("Mark Incident") { Task { await model.markLive() } }
                                .disabled(model.busy || !live.active)
                        }
                        Button("Stop and save") { Task { await model.stopLive() } }
                            .disabled(model.busy || !live.active)
                        if !live.isDiagnostic && !live.isHandoff { Text("\(live.incidents.count) manual incident(s). Allow 60 seconds before marking and 120 seconds afterward for a full window.").font(.caption) }
                        ForEach(live.readings) { row in
                            VStack(alignment: .leading) {
                                Text(row.signalId).font(.caption)
                                Text("\(row.ecu): \(row.value, specifier: "%.2f") \(row.unit) · \(row.quality) · \(row.ageMs / 1000) s old")
                            }
                        }
                        ForEach(live.capabilities) { row in
                            Text("\(row.key) / \(row.ecu): \(row.state)").font(.caption)
                        }
                    }
                    NavigationLink("Saved sessions, incidents and export") { PhysicalHistoryView() }
                        .disabled(model.live?.active == true)
                }
                Section("Developer identity check") {
                    Toggle("Enable adapter firmware read", isOn: $enableIdentity)
                    Text("One firmware identity read per connection. Keep the vehicle parked. This does not verify vehicle communication.").font(.caption)
                    Button("Read adapter identity once") { Task { await model.identify() } }
                        .disabled(!enableIdentity || model.state != "ready" || model.busy || model.live?.active == true)
                }
                Section("Raw connection evidence") {
                    Button("Export closed connection capture") {
                        Task { if let url = await model.share() { share = SessionShareRequest(url: url) } }
                    }
                        .disabled(model.busy || !model.canShareCapture || share != nil)
                    Text("Recorded sessions use Saved sessions, incidents and export above. This button shares a separate closed transport capture.").font(.caption)
                    if let error = model.error { Text(error).foregroundStyle(.red).accessibilityIdentifier("connection-export-error") }
                    Text("JSONL log retains full bytes. Preview shows the latest 32 records, up to 256 bytes each. Stream acceptance does not establish adapter receipt.").font(.caption)
                    Text(model.raw).font(.system(.caption, design: .monospaced)).textSelection(.enabled)
                }
            }
            .navigationTitle("Adapter Transport")
            .sheet(item: $share) { request in
                SessionShareSheet(url: request.url) {
                    if share?.id == request.id { share = nil }
                }
            }
            .task { await model.connect(); while !Task.isCancelled { await model.refresh(); try? await Task.sleep(for: .milliseconds(500)) } }
            .onChange(of: phase) { _, value in
                Task { await model.lifecycle(value == .active ? "foreground" : value == .background ? "background" : "inactive") }
            }
        }
    }
}

struct PhysicalHistoryView: View {
    @StateObject private var model: RecorderModel
    init(service: RecorderService? = nil) {
        _model = StateObject(wrappedValue: RecorderModel(service: service, historyOnly: true))
    }
    var body: some View {
        SessionsView(model: model)
            .task { await model.refresh(advance: false) }
    }
}

// Shared with the native scripted acceptance host. This view only renders Go
// snapshots; it cannot send commands or infer unsupported ECU parameters.
struct LiveDiscoveryStatusView: View {
    let live: LiveSnapshot
    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if live.isHandoff {
                if let state = live.adapterState {
                    Text("Adapter settings captured: \(state.captured)/3")
                        .accessibilityIdentifier("handoff-count")
                    Text("Protocol number: \(state.protocolNumber.isEmpty ? "Not received" : state.protocolNumber)")
                        .accessibilityIdentifier("handoff-protocol")
                    Text("Protocol description: \(state.description.isEmpty ? "Not received" : state.description)")
                    Text("Baud rate reply: \(state.baud.isEmpty ? "Not received" : state.baud)")
                        .accessibilityIdentifier("handoff-baud")
                    Text("Read before any VFR reset or protocol selection. These are unverified adapter settings; OBDLink cleanup or reconnection may already have changed them.")
                        .font(.caption).accessibilityIdentifier("handoff-caveat")
                    if !live.active && state.captured < 3 {
                        Text("Some settings were not received. The completed capture preserves the available raw evidence.").font(.caption)
                    }
                }
            } else if let discovery = live.discovery {
                if live.workload == "can29_diagnostic" {
                    Text("Profile: CAN 29-bit / 500 kbps").font(.caption)
                        .accessibilityIdentifier("discovery-can29-profile")
                }
                Text("Adapter setup: \(discovery.setupComplete)/\(discovery.setupTotal)")
                    .accessibilityIdentifier("discovery-setup")
                Text(live.initialized ? "Adapter setup verified" : "Waiting for verified adapter initialization").font(.caption)
                Text("ECU discovery: \(discovery.title)")
                    .accessibilityIdentifier("discovery-outcome")
                Text("Discovery attempts: \(discovery.attempts)")
                    .accessibilityIdentifier("discovery-attempts")
                if discovery.outcome == "no_data" || discovery.outcome == "timeout" || discovery.outcome == "protocol_error" {
                    Text("ECU support is still unknown unless listed below. RPM/coolant requests wait for valid support evidence.")
                        .font(.caption).accessibilityIdentifier("discovery-unknown")
                }
                if live.active && discovery.nextAttemptMs >= 0 {
                    Text("Next discovery attempt eligible in about \((discovery.nextAttemptMs + 999) / 1000) s while this app remains active.")
                        .font(.caption).accessibilityIdentifier("discovery-retry")
                }
                if live.isDiagnostic {
                    Text(discovery.canStatus.isEmpty ? "CAN status: not read yet" : "CAN status reply: \(discovery.canStatus) · \(discovery.canStatusAgeMs / 1000) s ago")
                        .font(.caption.monospaced()).accessibilityIdentifier("discovery-can-status")
                    Text("Adapter status is retained as reported; it does not confirm vehicle communication.").font(.caption)
                }
            } else {
                Text(live.initialized ? "Adapter setup verified; capabilities below" : "Waiting for verified adapter initialization").font(.caption)
            }
        }.accessibilityElement(children: .contain)
    }
}
