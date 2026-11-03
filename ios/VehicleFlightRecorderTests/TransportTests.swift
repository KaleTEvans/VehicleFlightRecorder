import XCTest
@testable import VehicleFlightRecorder

private final class CaptureFixture: TransportCapture {
    var rows: [Data] = []
    var writes = Data()
    var closed = 0
    var used = false
    func receive(_ data: Data) throws { rows.append(data) }
    func identity() throws -> Data { guard !used else { throw VehicleTransportError.failed("Already sent") }; used = true; return Data("STI\r".utf8) }
    func accepted(_ data: Data) throws { writes.append(data) }
    func event(_ kind: String, _ detail: String) throws {}
    func close(_ reason: String) throws { closed += 1 }
}
private final class ChannelFixture: TransportChannel {
    var hasSpace = true
    var counts: [Int] = []
    var sent = Data()
    var closes = 0
    func write(_ data: Data) -> Int { let n = counts.isEmpty ? data.count : counts.removeFirst(); if n > 0 { sent.append(data.prefix(n)) }; return n }
    func close() { closes += 1 }
}
final class TransportTests: XCTestCase {
    func testFragmentedCoalescedReadsAndPartialZeroWrites() throws {
        let pump = TransportPump(), capture = CaptureFixture(), channel = ChannelFixture()
        try pump.begin(channel: channel, capture: capture); pump.ready(generation: pump.generation)
        XCTAssertTrue(channel.sent.isEmpty, "Connect must not send bytes")
        channel.counts = [1, 0, 2, 1]
        try pump.identify(); XCTAssertEqual(pump.pending.count, 3)
        pump.flush(generation: pump.generation)
        XCTAssertEqual(channel.sent, Data("STI\r".utf8)); XCTAssertEqual(capture.writes, channel.sent)
        for text in ["STI\rSTN", "2232 v5.10.3\r", ">\rSTN2232 v5.10.3\r>"] { pump.receive(Data(text.utf8), generation: pump.generation) }
        XCTAssertEqual(capture.rows.count, 3)
        XCTAssertThrowsError(try pump.identify())
        pump.disconnect(); pump.disconnect(); XCTAssertEqual(capture.closed, 1); XCTAssertEqual(channel.closes, 1)
    }
    func testDeadlineDisconnectAndStaleGeneration() throws {
        var time: TimeInterval = 0
        let pump = TransportPump(); pump.now = { time }
        let first = CaptureFixture(), channel = ChannelFixture()
        try pump.begin(channel: channel, capture: first); let old = pump.generation
        time = 11; pump.tick(); XCTAssertEqual(pump.state, .failed)
        time = 14
        let next = CaptureFixture(), nextChannel = ChannelFixture()
        try pump.begin(channel: nextChannel, capture: next); pump.ready(generation: old)
        XCTAssertEqual(pump.state, .connecting)
        pump.ready(generation: pump.generation)
        pump.receive(Data([255]), generation: old); XCTAssertTrue(next.rows.isEmpty)
        nextChannel.hasSpace = false; try pump.identify(); time = 20
        nextChannel.hasSpace = true; pump.flush(generation: pump.generation)
        XCTAssertEqual(pump.state, .failed); XCTAssertTrue(pump.pending.isEmpty)
        XCTAssertTrue(nextChannel.sent.isEmpty)
    }
    func testOverflowAndWriteErrorsStopTransport() throws {
        for overflow in [false, true] {
            let pump = TransportPump(), capture = CaptureFixture(), channel = ChannelFixture()
            try pump.begin(channel: channel, capture: capture); pump.ready(generation: pump.generation)
            if overflow { pump.receive(Data(repeating: 0, count: 65537), generation: pump.generation) }
            else { channel.counts = [-1]; try pump.identify() }
            XCTAssertEqual(pump.state, .failed); XCTAssertEqual(capture.closed, 1)
        }
    }
    func testNativeCaptureExactTranscript() throws {
        let docs = FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
        let capture = try GoConnectionCapture(root: docs.appendingPathComponent("NativeAcceptance/Connections"), metadata: ["source": "native-fixture", "protocol": "com.obdlink"])
        try capture.event("streams_ready", "Fixture streams")
        let request = try capture.identity(); XCTAssertEqual(request, Data("STI\r".utf8))
        try capture.accepted(Data(request.prefix(1))); try capture.accepted(Data(request.dropFirst()))
        for chunk in ["STI\rSTN", "2232 v5.10.3\r", ">"] { try capture.receive(Data(chunk.utf8)) }
        try capture.close("fixture disconnect")
        XCTAssertThrowsError(try capture.identity())
        XCTAssertTrue(FileManager.default.fileExists(atPath: capture.core.path()))
    }
}

private final class DirectoryFixture: AccessoryDirectory {
    var accessories: [AccessoryDescriptor] = []
    var changed: (() -> Void)?
    var registrations = 0
    var removals = 0
    var opens = 0
    var failOpen = false
    var last: NativeStreamChannel?
    func observe(_ changed: @escaping () -> Void) { registrations += 1; self.changed = changed }
    func stopObserving() { if changed != nil { removals += 1 }; changed = nil }
    func open(_ id: Int) -> NativeStreamChannel? {
        opens += 1
        if failOpen { return nil }
        let value = NativeStreamChannel(input: InputStream(data: Data()), output: OutputStream.toMemory())
        last = value; return value
    }
}

private func onTransport(_ executor: TransportExecutor, _ body: @escaping () throws -> Void) async throws {
    try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<Void, Error>) in
        let admitted = executor.submit {
            do { try body(); continuation.resume() }
            catch { continuation.resume(throwing: error) }
        }
        if !admitted { continuation.resume(throwing: VehicleTransportError.failed("Fixture mailbox full")) }
    }
}

extension TransportTests {
    func testPhysicalReplugDuringBackoffRecoversWithoutAnotherNotification() async throws {
        let executor = TransportExecutor(), directory = DirectoryFixture()
        let root = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let transport = MXPlusExternalAccessoryTransport(root: root, executor: executor, directory: directory)
        var time: TimeInterval = 0
        var old: NativeStreamChannel?
        var original: GoConnectionCapture?
        var generation = 0
        defer { executor.submit { transport.disconnect(); try? FileManager.default.removeItem(at: root) } }
        try await onTransport(executor) {
            transport.pump.now = { time }
            directory.accessories = [AccessoryDescriptor(id: 1, protocols: ["com.obdlink"], metadata: ["source": "fixture"])]
            try transport.connect()
            let streams = try XCTUnwrap(directory.last)
            old = streams; original = transport.capture
            transport.stream(streams.input, handle: .openCompleted)
            transport.stream(streams.output, handle: .openCompleted)
            XCTAssertEqual(transport.state, .ready)
            generation = transport.pump.generation
            try transport.requestIdentity()
            XCTAssertTrue(transport.capture!.core.snapshot().contains("\"accepted_bytes\":4"))
            time = 1
            directory.accessories = []
            directory.changed?()
        }
        // Notification callbacks enqueue onto the same owner before this barrier.
        try await onTransport(executor) {
            XCTAssertEqual(transport.state, .failed)
            XCTAssertTrue(original!.core.snapshot().contains("\"closed\":true"))
            time = 1.25
            directory.accessories = [AccessoryDescriptor(id: 2, protocols: ["com.obdlink"], metadata: ["source": "fixture"])]
            directory.changed?()
        }
        try await onTransport(executor) {
            XCTAssertEqual(directory.opens, 1, "A reconnect notification must honor backoff")
            XCTAssertEqual(transport.state, .failed)
            time = 5
            transport.tick() // No second connect notification or user action.
            XCTAssertEqual(directory.opens, 2)
            XCTAssertEqual(transport.state, .connecting)
            XCTAssertGreaterThan(transport.pump.generation, generation)
            XCTAssertNotEqual(original!.core.path(), transport.capture!.core.path())
            let streams = try XCTUnwrap(directory.last)
            transport.stream(old!.input, handle: .errorOccurred)
            XCTAssertEqual(transport.state, .connecting, "Old streams cannot fail the replacement")
            transport.stream(streams.input, handle: .openCompleted)
            transport.stream(streams.output, handle: .openCompleted)
            XCTAssertEqual(transport.state, .ready)
            XCTAssertNil(transport.pump.failure)
            let snapshot = transport.capture!.core.snapshot()
            XCTAssertTrue(snapshot.contains("\"requested_bytes\":0"))
            XCTAssertTrue(snapshot.contains("\"accepted_bytes\":0"))
            let lateNotification = directory.changed
            transport.disconnect()
            time = 100
            transport.tick()
            lateNotification?()
        }
        try await onTransport(executor) {
            XCTAssertEqual(directory.opens, 2, "Explicit Disconnect cancels periodic and queued recovery")
            XCTAssertEqual(transport.state, .disconnected)
            XCTAssertEqual(directory.registrations, 1)
            XCTAssertEqual(directory.removals, 1)
        }
    }

    func testFailedSessionCreationBacksOffStopsAndAllowsExplicitRetry() async throws {
        let executor = TransportExecutor(), directory = DirectoryFixture()
        let root = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let transport = MXPlusExternalAccessoryTransport(root: root, executor: executor, directory: directory)
        var time: TimeInterval = 0
        defer { executor.submit { transport.disconnect(); try? FileManager.default.removeItem(at: root) } }
        try await onTransport(executor) {
            transport.pump.now = { time }
            directory.accessories = [AccessoryDescriptor(id: 1, protocols: ["com.obdlink"], metadata: ["source": "fixture"])]
            directory.failOpen = true
            try transport.connect()
            XCTAssertEqual(transport.state, .failed)
            for (instant, expectedOpens) in [(1.0, 1), (2, 2), (4, 2), (6, 3), (20, 3), (100, 3)] {
                time = instant
                transport.tick()
                XCTAssertEqual(directory.opens, expectedOpens)
                XCTAssertEqual(transport.state, .failed)
            }
            XCTAssertTrue(transport.notice.contains("retries paused"))
            directory.changed?() // An unchanged notification cannot replenish retries.
        }
        try await onTransport(executor) {
            XCTAssertEqual(directory.opens, 3)
            XCTAssertTrue(transport.capture!.core.snapshot().contains("\"closed\":true"))
            directory.failOpen = false
            try transport.connect()
            XCTAssertEqual(directory.opens, 4)
            let streams = try XCTUnwrap(directory.last)
            transport.stream(streams.input, handle: .openCompleted)
            transport.stream(streams.output, handle: .openCompleted)
            XCTAssertEqual(transport.state, .ready)
            XCTAssertNil(transport.pump.failure)
        }
    }

    func testProtocolReadinessAndPhysicalRemovalAreReconciledWithoutNotifications() async throws {
        let executor = TransportExecutor(), directory = DirectoryFixture()
        let root = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let transport = MXPlusExternalAccessoryTransport(root: root, executor: executor, directory: directory)
        var time: TimeInterval = 0
        defer { executor.submit { transport.disconnect(); try? FileManager.default.removeItem(at: root) } }
        try await onTransport(executor) {
            transport.pump.now = { time }
            let waiting = AccessoryDescriptor(id: 7, protocols: [], metadata: ["source": "fixture"])
            let ready = AccessoryDescriptor(id: 7, protocols: ["com.obdlink"], metadata: ["source": "fixture"])
            directory.accessories = [waiting]
            try transport.connect()
            XCTAssertEqual(directory.opens, 0, "Never open an unauthenticated protocol")
            directory.accessories = [ready]; time = 2; transport.tick()
            XCTAssertEqual(directory.opens, 1)
            let first = try XCTUnwrap(directory.last)
            transport.stream(first.input, handle: .openCompleted)
            transport.stream(first.output, handle: .openCompleted)
            XCTAssertEqual(transport.state, .ready)
            let previous = try XCTUnwrap(transport.capture)
            directory.accessories = [waiting]; time = 4; transport.tick()
            XCTAssertEqual(transport.state, .failed)
            XCTAssertTrue(previous.core.snapshot().contains("\"closed\":true"))
            directory.accessories = [ready]; time = 6; transport.tick()
            XCTAssertEqual(directory.opens, 2)
            XCTAssertEqual(transport.state, .connecting)
            XCTAssertNotEqual(previous.core.path(), transport.capture!.core.path())
            XCTAssertTrue(transport.capture!.core.snapshot().contains("\"requested_bytes\":0"))
        }
    }
}

extension TransportTests {
    func testAccessoryEnumerationFailureAndObserverCleanup() throws {
        let executor = TransportExecutor(), directory = DirectoryFixture()
        let root = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let transport = MXPlusExternalAccessoryTransport(root: root, executor: executor, directory: directory)
        defer { transport.disconnect(); try? FileManager.default.removeItem(at: root) }
        try transport.connect(); XCTAssertEqual(directory.opens, 0)
        directory.accessories = [AccessoryDescriptor(id: 1, protocols: ["unsupported"], metadata: [:])]
        try transport.connect(); XCTAssertEqual(directory.opens, 0)
        let match = AccessoryDescriptor(id: 2, protocols: ["com.obdlink"], metadata: ["name": "Fixture"])
        directory.accessories = [match, match]
        try transport.connect(); XCTAssertEqual(directory.opens, 0); XCTAssertTrue(transport.notice.contains("Multiple"))
        directory.accessories = [match]; directory.failOpen = true
        try transport.connect(); XCTAssertEqual(transport.state, .failed)
        XCTAssertEqual(directory.registrations, 1)
        XCTAssertTrue(transport.capture!.core.snapshot().contains("\"closed\":true"))
        transport.disconnect(); transport.disconnect(); XCTAssertEqual(directory.removals, 1)
    }
    func testLaterNotificationDisconnectReconnectAndStaleStream() async throws {
        let executor = TransportExecutor(), directory = DirectoryFixture()
        let root = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        let transport = MXPlusExternalAccessoryTransport(root: root, executor: executor, directory: directory)
        await withCheckedContinuation { continuation in
            executor.submit {
                do {
                    try transport.connect()
                    directory.accessories = [AccessoryDescriptor(id: 1, protocols: ["com.obdlink"], metadata: ["source": "fixture"])]
                    directory.changed?()
                    // The notification enqueues discovery before this barrier.
                    executor.submit {
                        XCTAssertEqual(transport.state, .connecting)
                        let old = directory.last!
                        transport.stream(old.input, handle: .openCompleted); transport.stream(old.output, handle: .openCompleted)
                        XCTAssertEqual(transport.state, .ready)
                        let generation = transport.pump.generation
                        transport.disconnect()
                        try? transport.connect()
                        XCTAssertGreaterThan(transport.pump.generation, generation)
                        transport.stream(old.output, handle: .errorOccurred)
                        XCTAssertEqual(transport.state, .connecting)
                        transport.disconnect()
                        XCTAssertEqual(directory.registrations, 2); XCTAssertEqual(directory.removals, 2)
                        try? FileManager.default.removeItem(at: root)
                        continuation.resume()
                    }
                } catch { XCTFail(error.localizedDescription); transport.disconnect(); continuation.resume() }
            }
        }
    }
}

extension TransportTests {
    func testStreamEndErrorAndPendingWriteDisconnect() throws {
        for event: Stream.Event in [.endEncountered, .errorOccurred] {
            let executor = TransportExecutor(), directory = DirectoryFixture()
            directory.accessories = [AccessoryDescriptor(id: 1, protocols: ["com.obdlink"], metadata: ["source": "fixture"])]
            let root = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
            let transport = MXPlusExternalAccessoryTransport(root: root, executor: executor, directory: directory)
            try transport.connect()
            let streams = try XCTUnwrap(directory.last)
            transport.stream(streams.input, handle: .openCompleted); transport.stream(streams.output, handle: .openCompleted)
            transport.stream(streams.input, handle: event)
            XCTAssertEqual(transport.state, .failed)
            XCTAssertTrue(transport.capture!.core.snapshot().contains("\"closed\":true"))
            transport.disconnect(); try FileManager.default.removeItem(at: root)
        }
        let pump = TransportPump(), capture = CaptureFixture(), channel = ChannelFixture()
        try pump.begin(channel: channel, capture: capture); pump.ready(generation: pump.generation)
        channel.counts = [1, 0]; try pump.identify(); XCTAssertEqual(pump.pending.count, 3)
        let old = pump.generation; pump.disconnect()
        let next = ChannelFixture()
        try pump.begin(channel: next, capture: CaptureFixture()); pump.ready(generation: pump.generation)
        pump.flush(generation: old); pump.flush(generation: pump.generation)
        XCTAssertTrue(pump.pending.isEmpty); XCTAssertTrue(next.sent.isEmpty)
        XCTAssertEqual(capture.writes, Data("S".utf8)); pump.disconnect()
    }
}
