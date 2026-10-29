import SwiftUI

struct DashboardView: View {
    @StateObject private var model = RecorderModel()
    @Environment(\.scenePhase) private var phase
    @State private var confirmStop = false

    var body: some View {
        TabView {
            NavigationStack {
                ScrollView {
                    VStack(alignment: .leading, spacing: 24) {
                        HStack {
                            Label("MOCK / SIMULATOR", systemImage: "testtube.2")
                                .font(.caption.bold()).padding(8).background(.yellow.opacity(0.2), in: Capsule())
                            Spacer()
                            Text("10× time").font(.caption.monospaced()).foregroundStyle(.secondary)
                        }
                        VStack(alignment: .leading, spacing: 6) {
                            Text(model.snapshot?.state.capitalized ?? "Ready to record").font(.largeTitle.bold())
                            Text(model.snapshot?.elapsed ?? "00:00").font(.system(.title, design: .monospaced)).monospacedDigit().accessibilityIdentifier("drive-elapsed")
                            Text("Physical adapter: Not connected").font(.subheadline).foregroundStyle(.secondary)
                            Text("Mock source: \(model.snapshot?.mockTransportState ?? "Ready")").font(.caption).foregroundStyle(.secondary)
                        }
                        LazyVGrid(columns: [GridItem(.adaptive(minimum: 140))], spacing: 12) {
                            metric("Voltage", "electrical.adapter_voltage")
                            metric("Engine speed", "powertrain.engine.rpm")
                            metric("Vehicle speed", "powertrain.vehicle.speed")
                            metric("Coolant", "powertrain.coolant.temperature")
                        }
                        VStack(alignment: .leading, spacing: 8) {
                            Text("Module evidence").font(.headline)
                            ForEach((model.snapshot?.modules.keys.sorted() ?? []), id: \.self) { module in
                                LabeledContent(module, value: (model.snapshot?.modules[module] ?? "Not observed").replacingOccurrences(of: "_", with: " "))
                            }
                            Text("Synthetic response evidence. Unobserved modules have no claimed health status.").font(.caption).foregroundStyle(.secondary)
                        }
                        if let last = model.snapshot?.incidents.last {
                            NavigationLink { IncidentDetailView(incident: last, service: model.service) } label: {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text("Latest incident").font(.caption).foregroundStyle(.secondary)
                                    Text(last.title).font(.headline)
                                    Text(last.status.capitalized).font(.subheadline)
                                }.frame(maxWidth: .infinity, alignment: .leading)
                            }
                        } else { Text("No incidents in this drive").foregroundStyle(.secondary) }
                        if let notice = model.notice { Text(notice).font(.footnote).accessibilityAddTraits(.updatesFrequently) }
                        if let error = model.error {
                            Label(error, systemImage: "exclamationmark.triangle.fill").foregroundStyle(.red)
                                .accessibilityLabel("Recorder error: \(error)")
                        }
                        Text("Mock time pauses when the app is not active. This preview does not connect to a vehicle.")
                            .font(.caption).foregroundStyle(.secondary)
                    }.padding()
                }
                .safeAreaInset(edge: .bottom) { controls }
                .navigationTitle("Vehicle Flight Recorder").navigationBarTitleDisplayMode(.inline)
                .confirmationDialog("Stop this drive?", isPresented: $confirmStop, titleVisibility: .visible) {
                    Button("Stop and save", role: .destructive) { Task { await model.stop() } }
                } message: { Text("Pending captures will keep their available evidence and be marked partial.") }
            }.tabItem { Label("Dashboard", systemImage: "gauge.with.dots.needle.50percent") }
            NavigationStack { SessionsView(model: model) }.tabItem { Label("Sessions", systemImage: "clock.arrow.circlepath") }
            NavigationStack { IncidentsView(model: model) }.tabItem { Label("Incidents", systemImage: "flag") }
            NavigationStack { DebugView(model: model) }.tabItem { Label("Debug", systemImage: "wrench.and.screwdriver") }
        }
        .task(id: phase) {
            model.setSceneActive(phase == .active)
            await model.refresh(advance: false)
            guard phase == .active else { return }
            while !Task.isCancelled {
                do { try await Task.sleep(for: .milliseconds(250)) } catch { break }
                await model.refresh(advance: true)
            }
        }
        .sheet(isPresented: Binding(get: { model.shareURL != nil }, set: { if !$0 { model.shareURL = nil } })) {
            if let url = model.shareURL { SessionShareSheet(url: url) }
        }
    }
    private var controls: some View {
        VStack(spacing: 8) {
                        Button { Task { await model.mark() } } label: {
                            Label("MARK INCIDENT", systemImage: "flag.fill")
                                .font(.headline).frame(maxWidth: .infinity, minHeight: 56)
                        }
                        .buttonStyle(.borderedProminent).tint(.orange).disabled(!model.canMark)
                        .accessibilityHint("Preserves available evidence before and after this moment")
                        if model.snapshot?.active == true {
                            Button("Stop and save drive", role: .destructive) { confirmStop = true }
                                .buttonStyle(.bordered).frame(maxWidth: .infinity).disabled(model.busy)
                        } else {
                            Button { Task { await model.start() } } label: {
                                Label("Start mock drive", systemImage: "record.circle").frame(maxWidth: .infinity, minHeight: 44)
                            }.buttonStyle(.borderedProminent).disabled(model.busy || model.service == nil)
                            Text("Scenario: \(scenarioTitle(model.scenario)). Change it in Debug.").font(.caption).foregroundStyle(.secondary)
                        }
        }.padding().background(.regularMaterial)
    }
    private func metric(_ title: String, _ id: String) -> some View {
        let reading = model.snapshot?.signals.first { $0.signalId == id }
        return VStack(alignment: .leading, spacing: 8) {
            Text(title).font(.caption).foregroundStyle(.secondary)
            Text(reading?.display ?? "Not observed").font(.title3.bold()).monospacedDigit()
        }.frame(maxWidth: .infinity, minHeight: 64, alignment: .leading).padding()
            .background(Color.secondary.opacity(0.08), in: RoundedRectangle(cornerRadius: 16))
            .accessibilityElement(children: .combine)
    }
}
let scenarioNames = ["healthy_drive", "voltage_tcm_dropout", "tcm_dropout", "mechanical_fault", "adapter_disconnect", "manual_mark"]
func scenarioTitle(_ name: String) -> String { name.replacingOccurrences(of: "_", with: " ").capitalized }
