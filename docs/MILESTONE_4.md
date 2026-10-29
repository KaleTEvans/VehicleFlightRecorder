# Milestone 4 — bridge, lifecycle and native build hardening

Status: **COMPLETE** — user acceptance recorded October 29, 2026.
Implementation and automated acceptance pass: clean device/simulator framework
builds, unsigned device compilation, 11 hosted unit tests, two native UI tests,
actual native export validation and Appetize publication. The user confirmed all
fresh-session smoke checks passed: healthy recording, manual capture, saved
incident details, evidence scrolling and native share-sheet presentation.
**Milestone 5 not started.** Earlier pending statements below are historical and
superseded by this closeout; signing and physical hardware access remain future work.

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
catch up on resume. Runtime recording, export and history calls run on RecorderService's actor.
Service construction still performs synchronous, bounded compatibility/initial
pack/path setup; this is not an asynchronous transport bootstrap. No live hardware
I/O is introduced. The default bundled pack remains small; larger/live startup
work must not be moved onto the UI actor in later milestones.

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
erasing a failed history-read message. Across background polling, errors persist until a full refresh
actually succeeds, while recovered history clears the error. Successful deliberate
actions can also dismiss previous messages; persistent faults are reported again
on the next applicable refresh. The fixture repairs
corrupt bytes explicitly in the test; the application still performs no repair.

## Recorded pre-final native evidence

Implementation source `86dabbe` passed clean device+simulator framework validation,
unsigned device compilation, nine hosted unit tests, two native UI tests,
native-generated ZIP validation/reconstruction, packaging and Appetize upload in
[run 40770251975](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40770251975).
Xcode 16.4 (16F6), Go 1.26.10 darwin/arm64, iOS SDKs 18.5; actual framework
identifiers are ios-arm64 and ios-arm64-simulator. Source hashes were checked
against the downloaded manifest. Source `d35def2` additionally passed the new
completion/failure recovery regression: ten hosted tests and two UI tests in
[run 40801000337](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40801000337).
The final history-error regression source will be recorded separately below.

Final Go race coverage (Go sources unchanged by the later Swift-only fixes):
`go test -race ./mobile -count=1 -v` passed in 61.421 s. Within it, 20 healthy
12-minute logical drives completed in 26.512 s wall time, with sampled heap
358,000 bytes before and 975,136 after GC at the end. That includes retained
current-session state; these samples are not a peak-memory or long-duration phone
measurement. Every producer was joined. No physical reliability claim follows.

The computer-use skill was read and the current helper was attempted. It failed
before Windows initialization with `sandboxCwd is not a local file URI:
file:///home/kaletevans/dev/VehicleFlightRecorder`. No Windows/browser action or
new Appetize observation is claimed. A short fresh-session user check has been
requested; prior Milestone 3 acceptance is not relabeled as new M4 observation.

## Final native evidence — October 28, 2026

[Run 40802263720](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40802263720)
passed for implementation source `f82ef4d2c9c4f316ca735be671b82737352d514e`.
Actual tools: Go **1.26.10 darwin/arm64**, Xcode **16.4 / 16F6**, device and
simulator SDKs **18.5**, deployment floor **17.0**. Both framework slices passed
metadata/ARM64 checks; the unsigned device app compiled and its retained plist
identifies `iPhoneOS`, separately from simulator packaging.

Eleven hosted unit tests passed with zero failures (3.077 s execution), covering
compatibility/missing pack, stable error categories, terminal close, cancelled/
stale ticks, scene pause/resume, incomplete relaunch, decoding/archive bounds,
completion/I/O state recovery and persistent history errors. Both native UI tests
passed (84.560 s execution): both trigger jumps/repeat tap still reveal the
canonical heading, and background/resume followed by termination/relaunch leaves
the unfinished drive honestly labeled interrupted. Exported UI screenshots were
visually reviewed. These are native simulator observations, not Appetize or physical
phone/background-accessory evidence.

Two real native-generated ZIPs were copied from the tested app container. The
runner validated and reconstructed them; the downloaded copies independently
passed local Python `validate`, `summarize` and `incident` commands by actual ID:

| Native session / scenario | Evidence | ZIP SHA256 |
| --- | --- | --- |
| `mock-1793238595719029000` / healthy + manual | 30,966 observations, 12 events, one captured manual trigger | `9ffcf82885b8b315ae73b84029d32827e0714dcc40c5e89bf599935fcbe9a995` |
| `mock-1793238597104378000` / voltage + TCM | 38,167 observations, 17 events, one captured incident with two ordered triggers | `7999d7b119bff868a26a97bf6809f34f1e1b835351c51da77f3e776f87d64888` |

[Native evidence artifact](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40802263720/artifacts/12867265523),
[simulator ZIP artifact](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40802263720/artifacts/12867265536).
Artifacts retain 14 days. Local copies, source-hash verification and independent
CLI outputs are under ignored `build/milestone-4-native-evidence-f82ef4d`.
Native manifest hashes match the current Go/framework inputs. Both Linux CI
checks for the implementation commit passed. Appetize upload passed to the
[existing preview](https://appetize.io/app/nrnzclustneju2ltjppkozdkta).

Remaining: start a fresh browser session; run a healthy drive, mark at or after
01:00 logical time, allow 120 more logical seconds, stop, inspect the saved
incident/evidence jump, and confirm Export presents the native share sheet.
Actual exported data already has independent native/Python evidence; browser
file-download limitations remain documented. No signed installation, accessory
connection, real background logging, or Milestone 5 work is claimed.

## Human acceptance and closeout — October 29, 2026

The user explicitly confirmed “All checks passed” for the requested fresh Appetize
healthy drive: manual mark after 01:00 logical time, two further simulated minutes,
stop/save, incident opening, evidence-button scrolling and Export share-sheet
presentation. This completes the final M4 acceptance item. It does not establish
browser file retrieval, a signed phone installation or physical accessory behavior.
Native-created ZIP retrieval/validation is independently recorded above.
Milestone 4 is complete; Milestone 5 has not started.
