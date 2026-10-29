# Milestone 4 — bridge, lifecycle and native build hardening

Status: **IN PROGRESS**. Local implementation and checks pass. Clean macOS
framework/device/simulator builds, native unit/UI tests, actual export validation
and renewed interactive acceptance remain to be recorded. Milestone 5 not started.

## Scope and ownership

The real Go mock recorder and accepted minimal UI remain in place. Go owns all
recording, incident rules, persistence and evidence indexes. RecorderService is
the sole native actor owner. Swift owns presentation and bounded native ZIP
packaging. Python continues validating the unchanged telemetry/session schema 1.
No ExternalAccessory discovery, initialization, real polling or signing is added.

## Bridge contract 1

`BridgeVersion()` returns 1. `CheckBridgeVersion(version)` rejects unsupported
clients; `BridgeContractJSON()` advertises limits without creating storage.
Native initialization compares its supported contract before constructing the Go
controller, and snapshot decoding verifies `bridge_version`. This version is
independent of application/core, pack, decoder and persisted telemetry versions.

Existing API is retained. `AdvanceSession(sessionID, milliseconds)` provides a
session-bound tick and rejects stale/inactive generations as cancelled. Native
refresh performs advance plus snapshot without actor suspension between them.
`SessionsPageJSON(offset)` returns 50 entries plus total/next/has-more metadata;
the native Sessions screen exposes an explicit load-more action. Legacy
`SessionsJSON()` remains available with the same response-size bounds.

Errors retain the gomobile NSError path and a stable `VFR|category|operation|detail`
message. Native translation preserves category, operation and useful detail.
Categories: invalid_input, invalid_state, unsupported_version, missing_pack,
io_failure, corrupt_data, resource_limit, cancelled, transport_unavailable
(reserved for future transport integration), internal. Native string-returning
bindings still use explicit NSError pointers; primitive/error calls are throwing.

Close is terminal and idempotent; it cancels/joins the bounded mock producer,
closes evidence and preserves unfinished metadata as recording/incomplete. Stop
requires an active drive and finalizes deliberately. A poisoned recording cannot
restart in that controller. Deinitialization is best-effort cleanup; explicit
close reports failures. Relaunch reads preserved incomplete data without repair,
relabeling it complete or enabling finalized export.

## Bounds and file behavior

- JSON document/line: 1 MiB; serialized bridge response: 8 MiB.
- Canonical evidence file: 64 MiB; bounded history index: 2,000 directory entries.
- Session page: 50; timeline page: 200; indexed capture: 50,000 rows.
- Manual marks: 100 per drive; bounded producer channel: 64 records plus pending.
- Native archive payload: 128 MiB, preflighted before reading/allocating payload.
- Aggregate history metadata/response and incident snapshots: 8 MiB; timeline index: 16 MiB of raw rows.

Bounds fail visibly and preserve original files; they do not truncate canonical
records. Readers reject non-regular/symlink escapes, foreign identities, malformed
or truncated records, unknown typed fields and unsupported metadata. Sessions
with corrupt evidence produce explicit errors. Paging is presentation, not raw
log deletion. Archive limits are MVP resource limits, not retention policy.

Native scene revisions invalidate obsolete refresh results; cancellation is
checked before actor operations. Mock time pauses while inactive and does not
catch up on resume. Export/history work and all Go calls stay off the UI actor.

## Native build

`scripts/mobile_toolchain.env` is the source for compiler, matching tool/runtime
revision, Xcode, deployment floor and target slices. Configuration validation
rejects generator/runtime mismatches, unsupported compiler floors, unverified
module replacements and XcodeGen deployment mismatches. The existing Go 1.26.10
cgo alignment fix remains required; the portable Go module minimum is 1.24.0.

The runner builds ios/arm64 and iossimulator/arm64 in one XCFramework, verifies
slice metadata and architecture, then compiles an unsigned device app separately
from simulator tests. `native-build-manifest.json` records actual tools/SDKs,
source hashes, commit and slices. Unsigned device compilation is not installation,
provisioning, Bluetooth validation or a signed distribution path.

## Verification record

Local full suite: Go tests/vet, 59 Python tests and 19 script tests pass; actionlint
and shell syntax pass. Generated Objective-C declarations were inspected in WSL;
this is header generation, not native compilation. The new tests cover version
rejection, terminal close, stale generations, paged history, oversized/malformed/
escaping inputs and injected metadata-finalization failure. The repeated workload
is 20 complete 12-minute logical healthy drives (four logical hours), not a
four-hour physical or wall-clock run. Actual timing/race and native results will
be added after final verification.

Final state review also added a native regression for a stale recording snapshot
after the core completes or encounters I/O failure. Refresh retrieves current
state after an error without silently swallowing a secondary refresh failure.
Action errors persist across successful background refreshes and clear on a
successful deliberate action. The final native run must cover this correction.

A related history-error regression prevents a successful snapshot-only tick from
erasing a failed history-read message. Poll errors persist until a full refresh
actually succeeds, while recovered history clears the error. The fixture repairs
corrupt bytes explicitly in the test; the application still performs no repair.
