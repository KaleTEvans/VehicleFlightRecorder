import Foundation
import SwiftUI
import UIKit

// ZIP STORE packaging only. Canonical evidence bytes and paths are unchanged.
enum SessionExporter {
    static let files = ["session.json", "observations.jsonl", "events.jsonl", "incidents.jsonl", "raw/transport.jsonl"]
    static func archive(directory: URL, output: URL) throws -> URL {
        var data = Data()
        var central = Data()
        for name in files {
            let file = directory.appendingPathComponent(name)
            let contents = try Data(contentsOf: file)
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
struct SessionShareSheet: UIViewControllerRepresentable {
    let url: URL
    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: [url], applicationActivities: nil)
    }
    func updateUIViewController(_ controller: UIActivityViewController, context: Context) {}
}
