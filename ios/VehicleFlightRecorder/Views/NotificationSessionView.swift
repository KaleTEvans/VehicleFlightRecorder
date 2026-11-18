import SwiftUI

// A notification tap never starts a recorder. Resolve the original ID each time;
// active -> dashboard; ended -> that summary; absent/corrupt -> visible error.
struct NotificationSessionView: View {
    let sessionID: String
    @ObservedObject var transport: TransportDebugModel
    @StateObject private var history = RecorderModel(historyOnly: true)
    @State private var lookupComplete = false
    var body: some View {
        Group {
            if transport.automatic?.sessionId == sessionID, transport.automatic?.enabled == true, let live = transport.live, live.active {
                List {
                    Text("Recording automatically · Limited standard OBD coverage").font(.headline)
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
                    while history.hasMoreSessions && !history.sessions.contains(where: { $0.id == sessionID }) { await history.loadMoreSessions() }
                    lookupComplete = true
                }
                try? await Task.sleep(for: .seconds(1))
            }
        }
    }
}
