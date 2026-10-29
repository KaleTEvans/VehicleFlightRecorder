# Native bridge contract 1

This contract is implemented by `core/mobile` and consumed by the native
RecorderService actor. It is distinct from core/app release 0.1.0, telemetry
schema 1, mock pack 0.2.0 and future decoder versions. Older persisted mock
sessions are unchanged. A new client verifies `BridgeVersion() == 1` **before**
creating a controller. Unsupported native/core versions fail closed, not by
attempting to decode a best-effort snapshot. JSON evolution within a contract
may add fields; incompatible required fields/semantics require a new contract.

## Calls and results

| Call | Behavior |
| --- | --- |
| `BridgeVersion()` | int64 compatibility version; no allocation of a recorder/files |
| `CheckBridgeVersion(version)` | explicit Go compatibility check, unsupported_version on mismatch |
| `BridgeContractJSON()` | JSON version, limits and stable category names; no recorder initialization |
| `NewController(pack, sessions)` | absolute app-supplied paths, bounded mock pack validation, app-owned session root; creates no drive |
| `Start(scenario)` | starts one of the existing named mock scenarios with a new session ID; rejects active, failed or closed controller |
| `Advance(milliseconds)` | legacy explicit mock tick, 100–10,000 ms in 100 ms steps; active only |
| `AdvanceSession(id, milliseconds)` | same tick with mandatory current-session identity; stale/inactive generation returns cancelled |
| `Snapshot()` | JSON schema/bridge/core/pack versions, state, session, readings, modules, incidents, time and actual errors; no invented measurements |
| `Mark(label)` | Go-owned logical timestamp and manual capture; 1–256 label bytes, active session, 100 marks/drive |
| `Stop()` | active-only deliberate finalization at current mock time; repeated/idle stop is invalid_state |
| `Close()` | terminal, repeatable cleanup; joins producer and closes files; repeated close retains any close error |
| `SessionsJSON()` | legacy full JSON array, subject to index/response bounds; may reject oversized responses |
| `SessionsPageJSON(offset)` | 50-entry page, total_sessions, next_offset and has_more; offset must be 0 through total |
| `IncidentJSON(session, incident, offset)` | incident/trigger offsets and up to 200 canonical timeline rows, total/next/has_more |
| `ExportPath(id)` | finalized local evidence directory, required regular files and file bounds; Swift packages exact bytes, Python verifies content |

New native clients page history instead of aggregating it across the bridge.
The Sessions UI exposes more pages explicitly; first-page refresh preserves older
pages already displayed. Paths never come from a text field or arbitrary native
filesystem selector. Session IDs are bounded directory basenames with no path
separators/NUL; metadata IDs must agree with directory IDs. Evidence and incident
references cannot silently select a foreign session.

## Ownership and lifecycle

Each Go controller mutex serializes calls. It owns the recorder and rolling buffer.
The generator goroutine owns immutable inputs, writes a 64-record channel and
never accesses recorder state. Stop/close/error cancels and joins it; one pending
record can sit beyond the current tick. The recorder itself remains single-owner.

RecorderService construction performs bounded initial compatibility/pack/path
setup synchronously. Runtime recording/read/export operations are actor-isolated.
RecorderService owns the native MobileController and performs each actor operation
without suspension while calling Go. Refresh combines session-checked advance
and snapshot. Native cancellation is checked before an operation is admitted;
a synchronous Go operation already admitted completes atomically, rather than
pretending a cancelled Swift task rolled back disk writes. Each tick is bounded
to ten logical seconds. A queued old-session tick cannot advance a replacement
session. Native scene revisions prevent obsolete refresh results overwriting
new UI state. There is one awaited dashboard refresh at a time, without backlog.

Start after completion creates a new recorder/session. Close is terminal even
if no recording ever began. Deliberate Stop commits a complete session; Close
while recording preserves metadata as recording. File errors poison that active
recording and surface failed state; create a new controller before retrying a
poisoned instance. Native explicit close reports errors; deinit only performs
best-effort cleanup. Backgrounding pauses **mock** time without closing a drive;
resume does not backfill fabricated measurements.

On relaunch, a new controller can list valid preserved evidence and actual
incomplete counts. It does not adopt or repair an interrupted recording, remove
truncated rows, infer a final duration or present it as complete. Corrupt evidence
is reported with path/line context; users retain original files. Finalized native
export remains unavailable for incomplete sessions. Existing Python
`--allow-incomplete` is inspection, not an in-place repair facility.

## Errors

Go's gomobile error is still returned through NSError. The stable message prefix
is `VFR|category|operation|detail`; details can contain further separators. Native
BridgeFailure extracts the prefix, preserves operation/detail and supplies a
readable localized description. CancellationError, native decoding and packaging
errors also become categorized native failures. Validation errors do not silently
poison a healthy active recorder; storage failures do. Action failures remain
visible across background polling. Polling failures remain visible across
snapshot-only successes until a full refresh succeeds; a successful deliberate
user action also dismisses previous messages. An underlying persistent failure
is reported again on the next applicable refresh. Failed ticks retrieve current
core state so completion/I/O failure cannot leave the UI stuck on recording;
a secondary state-read failure is reported too.

| Category | Examples |
| --- | --- |
| invalid_input | unknown scenario, invalid ID/offset/step, oversized/empty label |
| invalid_state | active double start, idle stop/mark, closed controller, incomplete export |
| unsupported_version | bridge compatibility mismatch |
| missing_pack | required pack document absent at initialization |
| io_failure | OS read/write/rename/close failure |
| corrupt_data | malformed/truncated/foreign evidence, invalid typed documents, symlink escape |
| resource_limit | document/file/response/history/timeline/archive/mark budget exceeded |
| cancelled | stale/inactive generation, cancellation before native admission |
| transport_unavailable | reserved for future real transport, not simulated connectivity |
| internal | unexpected uncategorized core/native failure; not a successful empty response |

The current bindings return strings with explicit NSError pointers; the Swift
wrapper checks them before decoding. BOOL/error operations import as throwing.
Generated Objective-C declarations are inspected, then actual native compilation
and tests validate importer behavior. Never infer Swift signatures from Go alone.

## Resource limits

Current advertised limits: 1 MiB per JSON document/JSONL line, 8 MiB serialized
response, 64 MiB per evidence file, 2,000 root history entries, 50 sessions/page,
200 timeline rows/page, 50,000 indexed rows/capture and 100 manual marks/drive.
Native ZIP payload preflight is 128 MiB. Aggregate history metadata/response and retained incident snapshots are also
bounded to 8 MiB; the temporary timeline index is bounded to 16 MiB of raw rows.
Native decoding independently enforces
its response limit. These limits are conservative MVP implementation choices,
not measured phone-capacity or vehicle sampling promises.

Over-limit reads fail visibly; original canonical files are never truncated.
Files are required to be regular and remain within the canonical app-root path;
symlink escapes are rejected. Archive CRC and JSONL bytes remain unchanged.
No disk retention/deletion policy is introduced. Older exports remain consumable
by Python even where an app presentation/resource limit prevents viewing them.

## Future byte transport boundary — specified, not implemented here

Milestone 5 must use an explicit connection generation distinct from drive/session
identity. A reconnect ends the old generation, cancels queued outbound work and
rejects late callbacks. Native inbound Data and outbound buffers must transfer
owned copies before asynchronous retention; caller reuse must not mutate evidence.
Callback chunks have no protocol framing meaning; framing stays Go-owned in M6.

Proposed initial budgets are 64 KiB per submitted chunk and 256 KiB per direction
of queued bytes; M5 must validate/select actual budgets with its stream tests and
report them. Overflow must fail or record a coverage-loss/degraded-state event,
not silently discard data or block the UI. Reads/writes/close must share one
serialized ownership contract without holding a Go lock while waiting for a
native callback that needs that lock. Raw attempt/actual-write/receive order and
uncertain partial writes must remain distinct. No hardware byte queue, command
scheduler, EASession or speculative public byte API is implemented in M4.

See [live timing design](LIVE_TIMING.md) for the M6 acquisition/schema migration.

## Milestone 5 connection capture extension

`NewConnectionCapture(root, metadataJSON, unixMS, nativeNS)` returns a separate
one-generation owner. `Receive(bytes, ns)`, `IdentityRequest(ns)`,
`Accepted(bytes, ns)`, `Event(kind, detail, ns)`, `Close(reason, ns)`, `Path()` and
`Snapshot()` preserve transport-only evidence; none opens a drive or changes
schema-1 mock files. This is additive to bridge 1. The native dedicated run loop
calls Go synchronously; no Go call waits on native callbacks. IdentityRequest
returns owned `STI\r` bytes once, and Accepted validates their pending prefix.
Actual budgets and file/timing format supersede the earlier proposed queued-byte
budgets: see [Milestone 5](MILESTONE_5.md). No diagnostic framing/polling API exists.
