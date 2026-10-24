import SwiftUI

struct SessionsView: View {
    @ObservedObject var model: RecorderModel
    var body: some View {
        List {
            Section {
                if model.sessions.isEmpty {
                    ContentUnavailableView("No drives yet", systemImage: "clock", description: Text("Start a mock drive from the dashboard. Saved drives stay on this simulator."))
                }
                ForEach(model.sessions) { row in
                    NavigationLink {
                        List {
                            Section("Drive") {
                                LabeledContent("Started", value: row.session.started.formatted())
                                LabeledContent("Duration", value: row.session.duration)
                                LabeledContent("State", value: model.sessionState(row.session))
                                LabeledContent("Samples", value: "\(row.session.observationCount)")
                                LabeledContent("Events", value: "\(row.session.eventCount)")
                                LabeledContent("Incidents", value: "\(row.session.incidentCount)")
                                LabeledContent("Size", value: ByteCountFormatter.string(fromByteCount: row.sizeBytes, countStyle: .file))
                            }
                            Section("Incidents") {
                                if row.incidents.isEmpty { Text("No incidents recorded") }
                                ForEach(row.incidents) { incident in
                                    NavigationLink { IncidentDetailView(incident: incident, service: model.service) } label: { IncidentRow(incident: incident) }
                                }
                            }
                            Section {
                                Button("Export session ZIP") { Task { await model.export(row.id) } }
                                    .disabled(row.session.state != "complete" || model.busy)
                                Text("Stop and finalize before exporting. Interrupted sessions remain preserved locally; this app does not repair them.")
                                    .font(.caption).foregroundStyle(.secondary)
                                if let error = model.error { Text(error).foregroundStyle(.red) }
                            }
                        }.navigationTitle("Drive details")
                    } label: {
                        VStack(alignment: .leading, spacing: 5) {
                            Text(row.session.started, style: .date).font(.headline)
                            Text("\(row.session.started.formatted(date: .omitted, time: .shortened)) · \(row.session.duration)")
                            Text("\(row.session.observationCount) samples · \(row.session.incidentCount) incidents · \(model.sessionState(row.session))").font(.caption).foregroundStyle(.secondary)
                        }
                    }
                }
            } header: { Text("Local mock drives") }
        }.navigationTitle("Sessions").refreshable { await model.refresh(advance: false) }
    }
}
struct IncidentRow: View {
    let incident: IncidentInfo
    var body: some View {
        VStack(alignment: .leading, spacing: 5) {
            Text(incident.title).font(.headline)
            Text(Date(timeIntervalSince1970: Double(incident.triggerTimeUnixMs) / 1000), style: .time)
            Text("\(incident.severity.capitalized) · \(incident.status.capitalized) · \(incident.triggers.count) trigger(s)")
                .font(.caption).foregroundStyle(.secondary)
        }
    }
}
struct IncidentsView: View {
    @ObservedObject var model: RecorderModel
    private var incidents: [IncidentInfo] {
        var items = Dictionary(uniqueKeysWithValues: model.sessions.flatMap(\.incidents).map { ($0.id, $0) })
        for item in model.snapshot?.incidents ?? [] { items[item.id] = item }
        return items.values.sorted { $0.triggerTimeUnixMs > $1.triggerTimeUnixMs }
    }
    var body: some View {
        List {
            if incidents.isEmpty { ContentUnavailableView("No incidents yet", systemImage: "flag", description: Text("Mark a moment during a drive, or choose a fault scenario in Debug.")) }
            ForEach(incidents) { incident in
                NavigationLink { IncidentDetailView(incident: incident, service: model.service) } label: { IncidentRow(incident: incident) }
            }
        }.navigationTitle("Incidents").refreshable { await model.refresh(advance: false) }
    }
}
struct IncidentDetailView: View {
    let incident: IncidentInfo
    let service: RecorderService?
    @State private var page: IncidentPage?
    @State private var rows: [EvidenceEntry] = []
    @State private var error: String?
    @State private var loading = false
    @State private var startOffset: Int64 = 0
    private var current: IncidentInfo { page?.incident ?? incident }
    var body: some View {
        List {
            Section("Capture") {
                IncidentRow(incident: current)
                LabeledContent("Configured pre / post", value: "\(Int(current.preWindowSeconds)) s / \(Int(current.postWindowSeconds)) s")
                LabeledContent("Available span", value: String(format: "%.1f s", current.capturedSeconds))
                Text("Coverage: \(current.incompleteReasons.isEmpty ? "Available requested window" : current.incompleteReasons.joined(separator: ", ").replacingOccurrences(of: "_", with: " "))")
                ForEach(current.coverageGaps) { gap in
                    Text("Gap: \(gap.reason.replacingOccurrences(of: "_", with: " ")) · \(Double(gap.startTimeUnixMs - current.triggerTimeUnixMs) / 1000, specifier: "%+.1f") to \(Double(gap.endTimeUnixMs - current.triggerTimeUnixMs) / 1000, specifier: "%+.1f") s")
                        .foregroundStyle(.orange)
                }
                if !current.relatedIncidentIds.isEmpty { Text("Related: \(current.relatedIncidentIds.joined(separator: ", "))").font(.caption) }
            }
            Section("Triggers · observed order") {
                ForEach(current.triggers) { trigger in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(scenarioTitle(trigger.triggerType)).font(.headline)
                        Text(String(format: "%+.3f s from first trigger", Double(trigger.timestampUnixMs - current.triggerTimeUnixMs) / 1000)).font(.caption.monospaced())
                        Text("\(trigger.category) · \(trigger.severity)\(trigger.label.map { " · " + $0 } ?? "")")
                        Text("\(trigger.evidence.file):\(trigger.evidence.line) · sequence \(trigger.evidence.recordSequence)")
                            .font(.caption.monospaced()).foregroundStyle(.secondary)
                        if let offset = page?.triggerOffsets[trigger.id] {
                            Button("View evidence around trigger") { Task { await load(reset: true, override: max(0, offset - 50)) } }.disabled(loading)
                        }
                    }
                }
                Text("Synthetic temporal evidence. A preceding event does not establish a cause.").font(.caption).foregroundStyle(.secondary)
            }
            Section("Canonical evidence timeline") {
                ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(String(format: "%+.3f s", Double(row.reference.timestampUnixMs - current.triggerTimeUnixMs) / 1000)).font(.caption.monospaced()).foregroundStyle(.secondary)
                        Text(row.title).font(.subheadline)
                        Text(row.detail).font(.caption)
                        Text("\(row.reference.file):\(row.reference.line)").font(.caption2.monospaced()).foregroundStyle(.secondary)
                    }
                }
                if let page {
                    Text("Records \(startOffset + 1)–\(startOffset + Int64(rows.count)) of \(page.totalRecords)").font(.caption).foregroundStyle(.secondary)
                    if page.hasMore { Button("Load next 200 records") { Task { await load(reset: false) } }.disabled(loading) }
                }
                if loading { ProgressView() }
                if let error { Text(error).foregroundStyle(.red) }
            }
        }.navigationTitle("Incident evidence")
            .task { await load(reset: true) }
            .toolbar { Button("Refresh") { Task { await load(reset: true) } }.disabled(loading) }
    }
    private func load(reset: Bool, override: Int64? = nil) async {
        guard !loading, let service else { return }
        loading = true
        defer { loading = false }
        do {
            let offset = override ?? (reset ? 0 : (page?.nextOffset ?? 0))
            let result = try await service.incident(session: incident.sessionId, id: incident.id, offset: offset)
            if reset { startOffset = offset }
            rows = reset ? result.timeline : rows + result.timeline
            page = result; error = nil
        } catch { self.error = error.localizedDescription }
    }
}
struct DebugView: View {
    @ObservedObject var model: RecorderModel
    var body: some View {
        Form {
            Section("Mock drive") {
                Picker("Scenario", selection: $model.scenario) {
                    ForEach(scenarioNames, id: \.self) { name in Text(scenarioTitle(name)).tag(name) }
                }.disabled(model.snapshot?.active == true || model.busy)
                Text("Selection applies to the next drive. 12 logical minutes run at approximately 10×; processing time can slow playback.")
                    .font(.caption).foregroundStyle(.secondary)
            }
            Section("Runtime") {
                LabeledContent("Core", value: model.snapshot?.coreVersion ?? "Unavailable")
                LabeledContent("Vehicle pack", value: model.snapshot?.packVersion ?? "Unavailable")
                LabeledContent("Recorder", value: model.snapshot?.state ?? "Initializing")
                LabeledContent("Physical adapter", value: "Not connected")
                LabeledContent("Raw traffic", value: "None · normalized mock source")
                Text(model.sessionLocation).font(.caption.monospaced()).textSelection(.enabled)
            }
            if let error = model.error { Section("Error") { Text(error).foregroundStyle(.red) } }
        }.navigationTitle("Debug")
    }
}
