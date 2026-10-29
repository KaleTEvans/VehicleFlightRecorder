import Foundation

struct BridgeFailure: LocalizedError {
    let category: String
    let operation: String
    let detail: String
    var errorDescription: String? { "\(category.replacingOccurrences(of: "_", with: " ").capitalized): \(detail)" }

    static func translated(_ error: Error, operation: String) -> BridgeFailure {
        if let failure = error as? BridgeFailure { return failure }
        if error is CancellationError { return BridgeFailure(category: "cancelled", operation: operation, detail: "Operation cancelled") }
        let message = error.localizedDescription
        if let marker = message.range(of: "VFR|") {
            let fields = message[marker.upperBound...].split(separator: "|", maxSplits: 2, omittingEmptySubsequences: false)
            if fields.count == 3 { return BridgeFailure(category: String(fields[0]), operation: String(fields[1]), detail: String(fields[2])) }
        }
        let category = error is DecodingError ? "corrupt_data" : "io_failure"
        return BridgeFailure(category: category, operation: operation, detail: message)
    }
}
