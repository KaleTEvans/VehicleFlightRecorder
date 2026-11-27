import SwiftUI

// Display one real responder per signal. No averages, role inference or changes
// to recorded evidence; all sources remain visible in the disclosure and export.
enum LiveReadingSelection {
    static let order = ["powertrain.engine.rpm", "powertrain.coolant.temperature", "electrical.adapter_voltage",
                        "powertrain.vehicle.speed", "powertrain.oil.temperature", "powertrain.fuel.rate",
                        "powertrain.fuel.trim.short.bank1", "powertrain.fuel.trim.long.bank1",
                        "powertrain.fuel.trim.short.bank2", "powertrain.fuel.trim.long.bank2"]
    static func primary(_ readings: [LiveReading]) -> [LiveReading] {
        let groups = Dictionary(grouping: readings, by: \.signalId)
        return order.compactMap { signal in
            groups[signal]?.sorted {
                let ranks = ["valid": 0, "stale": 1, "unavailable": 2, "invalid": 3]
                let a = ranks[$0.quality, default: 4], b = ranks[$1.quality, default: 4]
                if a != b { return a < b }
                return $0.ecu < $1.ecu
            }.first
        }
    }
    static func title(_ signal: String) -> String {
        switch signal {
        case "powertrain.engine.rpm": return "Engine speed"
        case "powertrain.coolant.temperature": return "Coolant temperature"
        case "electrical.adapter_voltage": return "Adapter voltage"
        case "powertrain.vehicle.speed": return "Vehicle speed"
        case "powertrain.oil.temperature": return "Engine oil temperature"
        case "powertrain.fuel.rate": return "Engine fuel rate"
        case "powertrain.fuel.trim.short.bank1": return "Short-term fuel trim (bank 1)"
        case "powertrain.fuel.trim.long.bank1": return "Long-term fuel trim (bank 1)"
        case "powertrain.fuel.trim.short.bank2": return "Short-term fuel trim (bank 2)"
        case "powertrain.fuel.trim.long.bank2": return "Long-term fuel trim (bank 2)"
        case "powertrain.engine.calculated_load": return "Calculated engine load"
        case "powertrain.intake.manifold_pressure": return "Manifold absolute pressure"
        case "powertrain.intake.temperature": return "Intake air temperature"
        case "powertrain.throttle.position": return "Throttle position"
        case "powertrain.fuel.rail_pressure_gauge": return "Fuel rail gauge pressure"
        default: return signal
        }
    }
}
struct LiveReadingSummaryView: View {
    let readings: [LiveReading]
    var body: some View {
        ForEach(LiveReadingSelection.primary(readings)) { row in
            VStack(alignment: .leading, spacing: 4) {
                Text(LiveReadingSelection.title(row.signalId)).font(.headline)
                Text("\(row.value, specifier: "%.1f") \(row.unit == "degC" ? "°C" : row.unit) · \(row.quality)")
                Text("Source \(row.ecu) · \(row.ageMs / 1000) s old").font(.caption).foregroundStyle(.secondary)
            }.accessibilityIdentifier("primary-reading-\(row.signalId)")
        }
        DisclosureGroup("All responders (\(readings.count) readings)") {
            ForEach(readings) { row in
                Text("\(row.ecu) · \(LiveReadingSelection.title(row.signalId)): \(row.value, specifier: "%.2f") \(row.unit) · \(row.quality) · \(row.ageMs / 1000) s old").font(.caption)
            }
        }.accessibilityIdentifier("all-responders")
    }
}
struct LiveIncidentControls: View {
    @ObservedObject var model: TransportDebugModel
    let live: LiveSnapshot
    var body: some View {
        Button {
            Task { await model.markLive() }
        } label: {
            Label(model.markingIncident ? "Marking…" : model.incidentFeedback != nil ? "Incident marked" : "Mark Incident", systemImage: model.incidentFeedback != nil ? "checkmark.circle.fill" : "flag.fill")
                .frame(maxWidth: .infinity).padding(.vertical, 6)
        }.buttonStyle(.borderedProminent)
            .disabled(model.busy || model.markingIncident || !live.active)
            .sensoryFeedback(.success, trigger: model.markFeedbackSequence)
            .accessibilityIdentifier("mark-live-incident")
        if let feedback = model.incidentFeedback {
            Text(feedback).foregroundStyle(.green).accessibilityIdentifier("incident-mark-feedback")
        }
        Text("\(live.incidents.count) incident(s) in this recording").font(.subheadline)
            .accessibilityIdentifier("live-incident-count")
        if let last = live.incidents.last {
            Text("Latest capture: \(last.status) · 60 s before / 120 s after mark").font(.caption)
        }
    }
}
