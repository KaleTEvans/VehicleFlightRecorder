import SwiftUI

struct TransportDebugView: View {
    @ObservedObject var model: TransportDebugModel = .shared
    @ObservedObject private var confirmations = RecordingConfirmation.shared
    @Environment(\.scenePhase) private var phase
    @State private var enableIdentity = false
    @State private var share: SessionShareRequest?
    @State private var parked = false
    @State private var operatingState = "ignition_on_engine_off"
    @State private var workload = "combined"
    @State private var reference = ""
    @AppStorage("stationaryProtocolProfile") private var protocolProfile = StationaryProtocol.can29.rawValue
    private var isRegularWorkload: Bool { ["combined", "rpm_only", "coolant_only", "voltage_only"].contains(workload) }
    var body: some View {
        NavigationStack {
            List {
                Section("PHYSICAL ADAPTER / TRANSPORT TEST") {
                    Text("Vehicle Flight Recorder").font(.headline)
                    Text("Transport: \(model.state)")
                    Text(model.notice)
                    if let notice = model.recoveryNotice { Text(notice).font(.caption).accessibilityIdentifier("adapter-recovering") }
                    if workload == "adapter_handoff" { Text("Adapter-only handoff capture · VFR leaves the vehicle protocol unchanged.").font(.caption) }
                    else if workload == "can29_diagnostic" { Text("Standard OBD stationary diagnostic · CAN 29-bit / 500 kbps. Enhanced Alfa / TCM signals are not enabled.").font(.caption) }
                    else { Text("Standard OBD stationary testing · \(isRegularWorkload ? StationaryProtocol.title(protocolProfile) : StationaryProtocol.can11.title). Enhanced Alfa / TCM signals are not enabled.").font(.caption) }
                    if let error = model.error { Text(error).foregroundStyle(.red).textSelection(.enabled) }
                    Button("Discover / Connect") { Task { await model.connect() } }.disabled(model.busy)
                    Button("Disconnect") { Task { await model.disconnect() } }.disabled(model.busy || model.automatic?.enabled == true)
                }
                Section("Hands-off recording — parked validation") {
                    Toggle("Vehicle is parked", isOn: $parked)
                        .disabled(model.automatic?.enabled == true)
                    Text("Enable while parked. VFR watches for repeated fresh engine RPM; connecting the scanner alone does not create a drive. Initial validation is limited to 30-minute recordings. Road and background behavior still require physical acceptance.")
                        .font(.caption)
                    Toggle("Hands-off monitoring", isOn: Binding(get: { model.automatic?.enabled == true }, set: { value in
                        Task { await model.setAutomatic(value, profile: protocolProfile) }
                    }))
                    .disabled(model.busy || (model.automatic?.enabled != true && (!parked || model.live?.active == true)))
                    .accessibilityIdentifier("automatic-monitoring-toggle")
                    Text("Activity: \(model.automatic?.state.replacingOccurrences(of: "_", with: " ") ?? "Disabled")")
                        .accessibilityIdentifier("automatic-state")
                    if model.automatic?.recording == true, let live = model.live {
                        Text(model.automatic?.acceptingTelemetry == true ? "Recording automatically · Limited standard OBD coverage" : "Preparing automatic recording · Waiting for fresh telemetry").font(.headline)
                            .accessibilityIdentifier("automatic-recording-active")
                        Text("Elapsed: \(live.elapsedMs / 1000) s · \(live.protocolTitle)")
                        LiveIncidentControls(model: model, live: live)
                        Button("Stop and save") { Task { await model.stopLive() } }
                            .buttonStyle(.borderless).disabled(model.busy)
                        Text("Stop saves this drive and prevents another automatic start until sustained engine-off evidence, or until you disable and re-enable monitoring. Monitoring continues while armed.").font(.caption)
                        LiveReadingSummaryView(readings: live.readings)
                    } else if model.automatic?.enabled == true {
                        Text("No drive recording. Activity evidence is kept separately; waiting for validated engine-running activity.").font(.caption)
                        if model.automatic?.manualStopOverride == true { Text("Manual Stop override is active.").font(.caption) }
                    }
                    NavigationLink("Saved sessions, incidents and export") { PhysicalHistoryView() }
                        .disabled(model.automatic?.recording == true)
                    Button("Export M7 diagnostic bundle") {
                        Task { if let url = await model.exportAutomationBundle() { share = SessionShareRequest(url: url) } }
                    }.buttonStyle(.borderless)
                        .disabled(model.busy || model.exportingDiagnostics || share != nil)
                        .accessibilityIdentifier("export-m7-diagnostics")
                    if model.exportingDiagnostics { ProgressView("Preparing diagnostic ZIP…") }
                    if let error = model.error { Text(error).foregroundStyle(.red).textSelection(.enabled).accessibilityIdentifier("automatic-export-error") }
                    Text("Export stops monitoring, saves an active healthy drive and preserves failed/interrupted originals. The bundle includes all activity probes, policy/resource logs and connection captures; no repair is performed.").font(.caption)
                }
                Section("Optional recording confirmation") {
                    Text("A quiet confirmation can tell you when an automatic drive has started accepting telemetry. It does not enable recording or background execution. Configure permissions while parked.").font(.caption)
                    Toggle("Recording confirmations", isOn: $confirmations.enabled)
                    Button("Allow quiet notifications") { Task { await confirmations.enableWithPermission() } }
                        .disabled(!parked || model.automatic?.recording == true)
                    Text("Notification permission: \(confirmations.permission.replacingOccurrences(of: "_", with: " "))").font(.caption)
                    if confirmations.permission == "denied" {
                        Button("Open notification settings") {
                            if let url = URL(string: UIApplication.openSettingsURLString) { UIApplication.shared.open(url) }
                        }.disabled(!parked || model.automatic?.recording == true)
                    }
                }
                if model.automatic?.enabled != true {
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
                    if isRegularWorkload {
                        Picker("OBD protocol", selection: $protocolProfile) {
                            ForEach(StationaryProtocol.allCases, id: \.rawValue) { profile in
                                Text(profile.title).tag(profile.rawValue)
                            }
                        }.disabled(model.live?.active == true)
                            .accessibilityIdentifier("stationary-protocol-picker")
                        Text("Select the vehicle's tested profile. CAN 29-bit / 500 kbps passed initial acquisition on this test car; no automatic protocol search is performed.").font(.caption)
                    }
                    if workload == "discovery_diagnostic" || workload == "can29_diagnostic" {
                        Text("Record about 2 minutes, then Stop and save. This mode records discovery attempts, adapter CAN status and voltage; RPM/coolant wait for valid support evidence. Keep this app open and the vehicle parked.").font(.caption)
                    }
                    TextField("Reference tool / reading (optional)", text: $reference)
                        .disabled(model.live?.active == true)
                    if workload == "adapter_handoff" {
                        Text("Capture the adapter state after using OBDLink. Leave the scanner powered. This mode sends only three read-only settings queries and saves automatically, normally within a few seconds; its active owner-loop limit is 15 seconds. App reconnection may already change the prior state.").font(.caption)
                    } else { Text("Explicit Start enables documented adapter setup and read-only OBD requests. Keep parked; each session ends after 10 minutes of elapsed native time when the owner loop runs. Background continuity is unverified.").font(.caption) }
                    Button(workload == "adapter_handoff" ? "Capture adapter settings" : "Start stationary recording") {
                        Task { await model.startLive(operatingState: operatingState, workload: workload, reference: String(reference.prefix(256)), protocolProfile: isRegularWorkload ? protocolProfile : "") }
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
                }
                Section("Developer identity check") {
                    Toggle("Enable adapter firmware read", isOn: $enableIdentity)
                    Text("One firmware identity read per connection. Keep the vehicle parked. This does not verify vehicle communication.").font(.caption)
                    Button("Read adapter identity once") { Task { await model.identify() } }
                        .disabled(!enableIdentity || model.state != "ready" || model.busy || model.live?.active == true || model.automatic?.enabled == true)
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
            .navigationDestination(item: $confirmations.route) { route in
                NotificationSessionView(sessionID: route.sessionID, transport: model)
            }
            .safeAreaInset(edge: .top) {
                if let banner = confirmations.banner { Text(banner).font(.subheadline).padding().frame(maxWidth: .infinity).background(.thinMaterial).accessibilityIdentifier("automatic-recording-confirmation") }
            }
            .task { await model.restoreAutomatic(); await model.connect(); await confirmations.refreshPermission(); while !Task.isCancelled { await model.refresh(); try? await Task.sleep(for: .milliseconds(500)) } }
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
                Text("Profile: \(live.protocolTitle)").font(.caption)
                    .accessibilityIdentifier(live.protocolProfile == StationaryProtocol.can29.rawValue ? "discovery-can29-profile" : "discovery-profile")
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
