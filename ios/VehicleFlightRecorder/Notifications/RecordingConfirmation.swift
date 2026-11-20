import Foundation
import SwiftUI
import UserNotifications
import UIKit
import Darwin

@MainActor protocol RecordingNotificationAdapter {
    func authorization() async -> String
    func requestAuthorization() async throws -> Bool
    func submit(_ fact: AutomaticStartFact) async throws
}
@MainActor final class SystemRecordingNotifications: RecordingNotificationAdapter {
    func authorization() async -> String {
        switch await UNUserNotificationCenter.current().notificationSettings().authorizationStatus {
        case .authorized, .provisional, .ephemeral: return "authorized"
        case .denied: return "denied"
        default: return "not_determined"
        }
    }
    func requestAuthorization() async throws -> Bool {
        try await UNUserNotificationCenter.current().requestAuthorization(options: [.alert])
    }
    func submit(_ fact: AutomaticStartFact) async throws {
        let content = UNMutableNotificationContent()
        content.title = "Drive recording started"
        content.body = "\(fact.vehicleName) · Recording automatically · Limited standard OBD coverage"
        content.sound = nil; content.badge = nil
        content.userInfo = ["session_id": fact.sessionId, "vehicle_name": fact.vehicleName]
        let request = UNNotificationRequest(identifier: "vfr.start.\(fact.sessionId)", content: content, trigger: nil)
        try await UNUserNotificationCenter.current().add(request)
    }
}
protocol ConfirmationReceipts {
    // True means exclusively/durably claimed. A partial/crashed claim is consumed.
    func claim(_ session: String, stage: String) throws -> Bool
}
final class FileConfirmationReceipts: ConfirmationReceipts {
    let directory: URL
    init(directory: URL) { self.directory = directory }
    func claim(_ session: String, stage: String) throws -> Bool {
        guard Self.validID(session), ["selected", "foreground"].contains(stage) else { throw VehicleTransportError.failed("Invalid confirmation receipt identity") }
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        let url = directory.appendingPathComponent("\(session).\(stage)")
        let descriptor = open(url.path, O_CREAT | O_EXCL | O_WRONLY, S_IRUSR | S_IWUSR)
        if descriptor < 0 {
            if errno == EEXIST { return false }
            throw VehicleTransportError.failed("Could not preserve confirmation receipt")
        }
        defer { close(descriptor) }
        let data = Data("claimed\n".utf8)
        let count = data.withUnsafeBytes { write(descriptor, $0.baseAddress, $0.count) }
        guard count == data.count, fsync(descriptor) == 0 else { throw VehicleTransportError.failed("Confirmation claim is uncertain; it will not be retried") }
        return true
    }
    static func validID(_ id: String) -> Bool { id.utf8.count == 37 && id.range(of: "^live-[0-9a-f]{32}$", options: .regularExpression) != nil }
}
struct RecordingRoute: Identifiable, Hashable { let id = UUID(); let sessionID: String }

// The Go owner claims each durable successful-start fact first. This native seam
// claims selection before any API call; uncertainty means no retry, never a stop.
@MainActor final class RecordingConfirmation: NSObject, ObservableObject, UNUserNotificationCenterDelegate {
    static let shared = RecordingConfirmation()
    @Published var banner: String?
    @Published var route: RecordingRoute?
    @Published var permission = "not_determined"
    @Published var enabled: Bool { didSet { preferences.set(enabled, forKey: "recordingConfirmationsEnabled") } }
    private let adapter: RecordingNotificationAdapter
    private let receipts: ConfirmationReceipts
    private let preferences: UserDefaults
    var isForeground: () -> Bool = { UIApplication.shared.applicationState == .active }
    var isSessionActive: (String) async -> Bool = { _ in false }
    var onEvent: (String) -> Void = { _ in }
    init(adapter: RecordingNotificationAdapter? = nil, receipts: ConfirmationReceipts? = nil, preferences: UserDefaults = .standard) {
        self.adapter = adapter ?? SystemRecordingNotifications()
        self.receipts = receipts ?? FileConfirmationReceipts(directory: FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0].appendingPathComponent("Automation/ConfirmationReceipts", isDirectory: true))
        self.preferences = preferences; enabled = preferences.bool(forKey: "recordingConfirmationsEnabled")
        super.init()
    }
    func refreshPermission() async { permission = await adapter.authorization() }
    // Only the explicit setup button calls this, never launch, reconnect or Start.
    func enableWithPermission() async {
        enabled = true
        permission = await adapter.authorization()
        if permission == "not_determined" {
            do { _ = try await adapter.requestAuthorization() }
            catch { onEvent("permission_request_failed: \(error.localizedDescription)") }
        }
        await refreshPermission()
    }
    func consume(_ fact: AutomaticStartFact) async {
        guard fact.startOrigin == "automatic", FileConfirmationReceipts.validID(fact.sessionId), fact.timeUnixMs > 0 else { return }
        do {
            guard try receipts.claim(fact.sessionId, stage: "selected") else { return }
            guard enabled else { onEvent("suppressed_disabled: \(fact.sessionId)"); return }
            guard await isSessionActive(fact.sessionId) else { onEvent("suppressed_ended_or_missing: \(fact.sessionId)"); return }
            permission = await adapter.authorization()
            guard enabled, await isSessionActive(fact.sessionId) else { onEvent("suppressed_stale_or_disabled: \(fact.sessionId)"); return }
            if isForeground() {
                try presentInApp(session: fact.sessionId, name: fact.vehicleName)
            } else if permission == "authorized" {
                onEvent("submission_requested_background: \(fact.sessionId)")
                do { try await adapter.submit(fact); onEvent("submitted_not_proof_of_display: \(fact.sessionId)") }
                catch { onEvent("submission_failed_no_retry: \(fact.sessionId): \(error.localizedDescription)") }
            } else { onEvent("suppressed_\(permission): \(fact.sessionId)") }
        } catch { onEvent("receipt_failed_no_retry: \(fact.sessionId): \(error.localizedDescription)") }
    }
    private func presentInApp(session: String, name: String) throws {
        guard enabled, try receipts.claim(session, stage: "foreground") else { return }
        let message = "Drive recording started · \(name) · Limited standard OBD coverage"
        banner = message; onEvent("in_app_requested: \(session)")
        Task { [weak self] in
            try? await Task.sleep(for: .seconds(5))
            if self?.banner == message { self?.banner = nil }
        }
    }
    nonisolated func userNotificationCenter(_ center: UNUserNotificationCenter, willPresent notification: UNNotification, withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void) {
        // If a background request reaches a now-visible app, use just the in-app
        // path. It can never also receive a system banner from this delegate.
        let id = notification.request.content.userInfo["session_id"] as? String
        let name = notification.request.content.userInfo["vehicle_name"] as? String ?? "Vehicle"
        let own = notification.request.identifier.hasPrefix("vfr.start.")
        completionHandler([])
        if own, let id, FileConfirmationReceipts.validID(id), notification.request.identifier == "vfr.start.\(id)" {
            Task { @MainActor [weak self] in
                guard let self, await self.isSessionActive(id) else { return }
                do { try self.presentInApp(session: id, name: name) }
                catch { self.onEvent("foreground_receipt_failed_no_retry: \(id)") }
            }
        }
    }
    nonisolated func userNotificationCenter(_ center: UNUserNotificationCenter, didReceive response: UNNotificationResponse, withCompletionHandler completionHandler: @escaping () -> Void) {
        let id = response.notification.request.content.userInfo["session_id"] as? String
        let own = response.notification.request.identifier.hasPrefix("vfr.start.")
        Task { @MainActor [weak self] in
            if own, let id, FileConfirmationReceipts.validID(id), response.notification.request.identifier == "vfr.start.\(id)" {
                self?.route = RecordingRoute(sessionID: id)
                self?.onEvent("tap_route_requested: \(id)")
            }
            completionHandler()
        }
    }
}

@MainActor final class RecordingAppDelegate: NSObject, UIApplicationDelegate {
    private var resourceTimer: Timer?
    private var lifecycleObservers: [NSObjectProtocol] = []
    func application(_ application: UIApplication, didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil) -> Bool {
        #if !targetEnvironment(simulator)
        let confirmations = RecordingConfirmation.shared
        UNUserNotificationCenter.current().delegate = confirmations
        confirmations.isSessionActive = { await TransportDebugModel.shared.isAutomaticSessionActive($0) }
        confirmations.onEvent = { detail in Task { await TransportDebugModel.shared.recordAutomaticEvent("confirmation_result", detail: detail) } }
        lifecycleObservers = [
            (UIApplication.didBecomeActiveNotification, "foreground"),
            (UIApplication.willResignActiveNotification, "inactive"),
            (UIApplication.didEnterBackgroundNotification, "background")
        ].map { name, phase in
            NotificationCenter.default.addObserver(forName: name, object: nil, queue: .main) { _ in
                Task { @MainActor in await TransportDebugModel.shared.lifecycle(phase) }
            }
        }
        Task {
            await TransportDebugModel.shared.restoreAutomatic()
            await TransportDebugModel.shared.lifecycle(application.applicationState == .active ? "foreground" : application.applicationState == .background ? "background" : "inactive")
        }
        UIDevice.current.isBatteryMonitoringEnabled = true
        resourceTimer = Timer.scheduledTimer(withTimeInterval: 60, repeats: true) { _ in
            Task { @MainActor in
                let detail = NativeResourceSample.json()
                await TransportDebugModel.shared.recordAutomaticEvent("battery_sample", detail: detail)
            }
        }
        #endif
        return true
    }
    func applicationDidBecomeActive(_ application: UIApplication) { phase("foreground") }
    func applicationWillResignActive(_ application: UIApplication) { phase("inactive") }
    func applicationDidEnterBackground(_ application: UIApplication) { phase("background") }
    private func phase(_ value: String) {
        #if !targetEnvironment(simulator)
        Task { await TransportDebugModel.shared.lifecycle(value) }
        #endif
    }
}

@MainActor enum NativeResourceSample {
    static func json() -> String {
        let device = UIDevice.current
        let value: [String: Any] = ["battery_level": device.batteryLevel >= 0 ? device.batteryLevel as Any : NSNull(), "battery_state": device.batteryState.rawValue, "thermal_state": ProcessInfo.processInfo.thermalState.rawValue, "low_power": ProcessInfo.processInfo.isLowPowerModeEnabled, "measurement_scope": "application_execution_only", "ambient_temperature": "unknown"]
        guard let data = try? JSONSerialization.data(withJSONObject: value, options: [.sortedKeys]) else { return "{}" }
        return String(decoding: data, as: UTF8.self)
    }
}
