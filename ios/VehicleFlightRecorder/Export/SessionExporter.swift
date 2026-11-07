import Foundation
import SwiftUI
import UIKit

// ZIP STORE packaging only. Canonical evidence bytes and paths are unchanged.
enum SessionExporter {
    static let files = ["session.json", "observations.jsonl", "events.jsonl", "incidents.jsonl", "raw/transport.jsonl"]
    static let maximumArchiveBytes: UInt64 = 128 * 1024 * 1024
    static func archive(directory: URL, output: URL) throws -> URL {
        let root = directory.standardizedFileURL.resolvingSymlinksInPath()
        var total: UInt64 = 0
        func preflight(_ names: [String]) throws {
            for name in names {
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
        try preflight(Self.files)
        let metadataURL = root.appendingPathComponent("session.json")
        let metadataSize = try metadataURL.resourceValues(forKeys: [.fileSizeKey]).fileSize ?? Int.max
        guard metadataSize <= 1024 * 1024 else { throw BridgeFailure(category: "resource_limit", operation: "archive", detail: "Session metadata exceeds 1 MiB") }
        let metadata = try JSONSerialization.jsonObject(with: Data(contentsOf: metadataURL)) as? [String: Any]
        guard let version = metadata?["schema_version"] as? Int, [1, 2].contains(version) else {
            throw VehicleTransportError.failed("Unsupported session export schema")
        }
        let extra = version == 2 ? ["capability.json"] : []
        try preflight(extra)
        let files = Self.files + extra
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
        let destination = output.appendingPathComponent(directory.lastPathComponent).appendingPathExtension("zip")
        try data.write(to: destination, options: .atomic)
        return destination
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
