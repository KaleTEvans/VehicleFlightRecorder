import SwiftUI

// A notification tap never starts a recorder. Resolve the original ID each time;
// active -> dashboard; ended -> that summary; absent/corrupt -> visible error.
struct NotificationSessionView: View {
    let sessionID: String
    @ObservedObject var transport: TransportDebugModel
    @StateObject private var history = RecorderModel(historyOnly: true)
    @State private var lookupComplete = false
    private var resolution: RecordingRouteResolution {
        RecordingRouteResolution.resolve(sessionID: sessionID, activeID: transport.automatic?.sessionId, active: transport.live?.active == true && transport.automatic?.enabled == true, savedIDs: Set(history.sessions.map(\.id)))
    }
    var body: some View {
        Group {
            if resolution == .active, let live = transport.live {
                List {
                    Text(transport.automatic?.acceptingTelemetry == true ? "Recording automatically · Limited standard OBD coverage" : "Preparing automatic recording").font(.headline)
                    Text("Session: \(sessionID)").font(.caption)
                    Text("Elapsed: \(live.elapsedMs / 1000) s")
                    ForEach(live.readings) { row in
                        Text("\(row.ecu) · \(row.signalId): \(row.value, specifier: "%.2f") \(row.unit) · \(row.quality)")
                    }
                    Button("Mark Incident") { Task { await transport.markLive() } }.disabled(transport.busy)
                    Button("Stop and save") { Task { await transport.stopLive() } }.disabled(transport.busy)
                }.navigationTitle("Active drive")
            } else if let row = history.sessions.first(where: { $0.id == sessionID }) {
                SessionDetailView(row: row, model: history)
            } else if lookupComplete {
                ContentUnavailableView("Session unavailable", systemImage: "exclamationmark.triangle", description: Text(history.error ?? "The original recording could not be found. No new drive was created."))
            } else { ProgressView("Opening recording…") }
        }
        .task {
            while !Task.isCancelled {
                await transport.refresh()
                if transport.automatic?.sessionId != sessionID {
                    await history.refresh(advance: false)
                    while history.hasMoreSessions && !history.sessions.contains(where: { $0.id == sessionID }) {
                        let count = history.sessions.count
                        await history.loadMoreSessions()
                        if history.sessions.count == count || history.error != nil { break }
                    }
                    lookupComplete = true
                }
                try? await Task.sleep(for: .seconds(1))
            }
        }
    }
}

// Re-evaluated at navigation/render time; notification arrival does not freeze
// active state, choose an unrelated drive or admit any recording command.
enum RecordingRouteResolution {
    case active, saved, unavailable
    static func resolve(sessionID: String, activeID: String?, active: Bool, savedIDs: Set<String>) -> Self {
        guard FileConfirmationReceipts.validID(sessionID) else { return .unavailable }
        if active, activeID == sessionID { return .active }
        return savedIDs.contains(sessionID) ? .saved : .unavailable
    }
}
