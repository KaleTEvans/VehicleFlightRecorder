import Foundation
import SwiftUI
import UIKit

// ZIP STORE packaging only. Canonical evidence bytes and paths are unchanged.
enum SessionExporter {
    static let files = ["session.json", "observations.jsonl", "events.jsonl", "incidents.jsonl", "raw/transport.jsonl"]
    static let maximumArchiveBytes: UInt64 = 128 * 1024 * 1024
    static func archive(directory: URL, output: URL, bundleFiles: [String]? = nil, archiveName: String? = nil) throws -> URL {
        let root = directory.standardizedFileURL.resolvingSymlinksInPath()
        var total: UInt64 = 0
        func preflight(_ names: [String]) throws {
            for name in names {
                guard !name.hasPrefix("/"), !name.split(separator: "/").contains(".."), name.utf8.count <= 1024 else {
                    throw VehicleTransportError.failed("Invalid evidence bundle path")
                }
                let file = root.appendingPathComponent(name)
                let attributes = try FileManager.default.attributesOfItem(atPath: file.path)
                guard attributes[.type] as? FileAttributeType == .typeRegular,
                      file.resolvingSymlinksInPath().path == file.path else {
                    throw BridgeFailure(category: "corrupt_data", operation: "archive", detail: "Export requires local regular evidence files")
                }
                let size = (attributes[.size] as? NSNumber)?.uint64Value ?? UInt64.max
                guard size <= maximumArchiveBytes - total else {
                    throw BridgeFailure(category: "resource_limit", operation: "archive", detail: "Session exceeds the 128 MiB export limit; original evidence is preserved")
                }
                total += size
            }
        }
        let files: [String]
        if let bundleFiles {
            guard !bundleFiles.isEmpty, bundleFiles.count <= 20_000, Set(bundleFiles).count == bundleFiles.count else {
                throw VehicleTransportError.failed("Invalid evidence bundle population")
            }
            try preflight(bundleFiles); files = bundleFiles
        } else {
            try preflight(Self.files)
            let metadataURL = root.appendingPathComponent("session.json")
            let metadataSize = try metadataURL.resourceValues(forKeys: [.fileSizeKey]).fileSize ?? Int.max
            guard metadataSize <= 1024 * 1024 else { throw BridgeFailure(category: "resource_limit", operation: "archive", detail: "Session metadata exceeds 1 MiB") }
            let metadata = try JSONSerialization.jsonObject(with: Data(contentsOf: metadataURL)) as? [String: Any]
            guard let version = metadata?["schema_version"] as? Int, [1, 2].contains(version) else {
                throw VehicleTransportError.failed("Unsupported session export schema")
            }
            let extra = version == 2 ? ["capability.json"] : []
            try preflight(extra); files = Self.files + extra
        }
        let overhead = files.reduce(UInt64(22)) { $0 + UInt64(76 + 2 * $1.utf8.count) }
        guard total + overhead <= maximumArchiveBytes else { throw VehicleTransportError.failed("Evidence bundle exceeds the 128 MiB ZIP limit; originals are preserved") }
        var data = Data()
        var central = Data()
        for name in files {
            let file = root.appendingPathComponent(name)
            let contents = try Data(contentsOf: file, options: .mappedIfSafe)
            guard UInt64(contents.count) <= maximumArchiveBytes, UInt64(data.count + contents.count) <= maximumArchiveBytes + 4096 else {
                throw BridgeFailure(category: "resource_limit", operation: "archive", detail: "Evidence grew beyond export limit")
            }
            guard contents.count < Int(UInt32.max), data.count < Int(UInt32.max) else {
                throw NSError(domain: "Export", code: 1, userInfo: [NSLocalizedDescriptionKey: "Session exceeds supported ZIP size"])
            }
            let filename = Data(name.utf8)
            let crc = crc32(contents)
            let offset = UInt32(data.count)
            data.le(UInt32(0x04034b50)); data.le(UInt16(20)); data.le(UInt16(0)); data.le(UInt16(0))
            data.le(UInt16(0)); data.le(UInt16(0x21)); data.le(crc)
            data.le(UInt32(contents.count)); data.le(UInt32(contents.count)); data.le(UInt16(filename.count)); data.le(UInt16(0))
            data.append(filename); data.append(contents)
            central.le(UInt32(0x02014b50)); central.le(UInt16(20)); central.le(UInt16(20)); central.le(UInt16(0)); central.le(UInt16(0))
            central.le(UInt16(0)); central.le(UInt16(0x21)); central.le(crc)
            central.le(UInt32(contents.count)); central.le(UInt32(contents.count)); central.le(UInt16(filename.count))
            central.le(UInt16(0)); central.le(UInt16(0)); central.le(UInt16(0)); central.le(UInt16(0)); central.le(UInt32(0)); central.le(offset)
            central.append(filename)
        }
        let start = UInt32(data.count)
        data.append(central)
        data.le(UInt32(0x06054b50)); data.le(UInt16(0)); data.le(UInt16(0)); data.le(UInt16(files.count)); data.le(UInt16(files.count))
        data.le(UInt32(central.count)); data.le(start); data.le(UInt16(0))
        try FileManager.default.createDirectory(at: output, withIntermediateDirectories: true)
        let destination = output.appendingPathComponent(archiveName ?? directory.lastPathComponent).appendingPathExtension("zip")
        try data.write(to: destination, options: .atomic)
        return destination
    }
    // Call on the transport executor only after monitoring and recording stop.
    // Original bytes (including incomplete evidence) are copied, never repaired.
    static func automationBundle(documents: URL, output: URL) throws -> URL {
        let root = documents.standardizedFileURL.resolvingSymlinksInPath()
        var names: [String] = []
        func collect(_ directory: URL) throws {
            var enumerationFailure: Error?
            guard let enumerator = FileManager.default.enumerator(at: directory, includingPropertiesForKeys: [.isRegularFileKey, .isSymbolicLinkKey], errorHandler: { _, error in enumerationFailure = error; return false }) else {
                throw VehicleTransportError.failed("Could not read the automation evidence directory")
            }
            for case let file as URL in enumerator {
                let values = try file.resourceValues(forKeys: [.isRegularFileKey, .isSymbolicLinkKey])
                guard values.isSymbolicLink != true else { throw VehicleTransportError.failed("Evidence bundle contains a symbolic link") }
                guard values.isRegularFile == true, file.lastPathComponent != "owner.lock" else { continue }
                guard file.path.hasPrefix(root.path + "/") else { throw VehicleTransportError.failed("Nonlocal automation evidence") }
                names.append(String(file.path.dropFirst(root.path.count + 1)))
                guard names.count <= 20_000 else { throw VehicleTransportError.failed("Automation bundle file limit") }
            }
            if let enumerationFailure { throw enumerationFailure }
        }
        guard FileManager.default.fileExists(atPath: root.appendingPathComponent("Automation/journal.jsonl").path), FileManager.default.fileExists(atPath: root.appendingPathComponent("Automation/state.json").path) else {
            throw VehicleTransportError.failed("No complete automation journal/state is available; original files remain in Files")
        }
        try collect(root.appendingPathComponent("Automation", isDirectory: true))
        let sessions = root.appendingPathComponent("Sessions", isDirectory: true)
        for directory in (try? FileManager.default.contentsOfDirectory(at: sessions, includingPropertiesForKeys: nil)) ?? [] where directory.lastPathComponent.hasPrefix("live-") {
            let metadata = try? JSONSerialization.jsonObject(with: Data(contentsOf: directory.appendingPathComponent("session.json"))) as? [String: Any]
            let live = metadata?["live"] as? [String: Any]
            if live?["policy_version"] as? String == "rpm-activity-1" || metadata == nil { try collect(directory) }
        }
        return try archive(directory: root, output: output, bundleFiles: names.sorted(), archiveName: "M7-diagnostics-\(UUID().uuidString)")
    }
    private static let crcTable: [UInt32] = (0..<256).map { value in
        var remainder = UInt32(value)
        for _ in 0..<8 { remainder = (remainder >> 1) ^ (remainder & 1 == 1 ? 0xedb88320 : 0) }
        return remainder
    }
    private static func crc32(_ contents: Data) -> UInt32 {
        var crc: UInt32 = 0xffffffff
        for byte in contents { crc = (crc >> 8) ^ crcTable[Int((crc ^ UInt32(byte)) & 0xff)] }
        return crc ^ 0xffffffff
    }
}
private extension Data {
    mutating func le<T: FixedWidthInteger>(_ value: T) {
        var little = value.littleEndian
        Swift.withUnsafeBytes(of: &little) { append(contentsOf: $0) }
    }
}
struct SessionShareRequest: Identifiable {
    let id = UUID()
    let url: URL
}
struct SessionShareSheet: UIViewControllerRepresentable {
    let url: URL
    var completed: () -> Void = {}
    func makeUIViewController(context: Context) -> UIActivityViewController {
        let controller = UIActivityViewController(activityItems: [url], applicationActivities: nil)
        let completion = completed
        controller.loadViewIfNeeded()
        controller.completionWithItemsHandler = { _, _, _, _ in
            // Clear the matching request for system cancel/Save to Files as well
            // as SwiftUI gesture dismissal. File generation has already finished.
            DispatchQueue.main.async { completion() }
        }
        return controller
    }
    func updateUIViewController(_ controller: UIActivityViewController, context: Context) {}
}
