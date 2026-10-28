# Milestone 3 — native dashboard and working mock bridge

Status: **PARTIALLY COMPLETE**. Native implementation, local checks, and the
fixed-compiler macOS simulator build/native export validation pass. Interactive
Appetize recording/manual marking and basic usability have user confirmation.
The user screenshot confirms a captured automatic incident with both triggers
in order and full available coverage. Evidence-button scrolling correction
awaits a rebuilt preview and human verification; exported-file retrieval remains
unconfirmed;
they are separate from automated native acceptance.
Milestone 4 not started. A UX review follows this functional milestone.

## Responsibilities and runtime

The existing Go recorder/detectors remain authoritative. `mobile.NewController`
loads the explicitly mock-only pack and an app-container Sessions directory.
Its string/scalar API exposes Start, Advance, Stop, Mark, Snapshot, SessionsJSON,
IncidentJSON (200-row pages), ExportPath and Close. Every controller operation
holds one mutex; only an immutable mock generator runs on a separate goroutine,
with a bounded 64-record channel and cancellation joined before stop/close.
Native recording, snapshot, timeline and export calls run through a single Swift
actor, away from the UI actor. Go string-returning binding methods preserve an
explicit NSError parameter; the Swift wrapper checks it before decoding results.
Boolean/error methods import as throwing Swift methods. Snapshot freshness marks
retained samples stale on disconnect or after the silence limit without changing
the original canonical observations on disk.

Swift schedules one awaited refresh every 250 ms and advances 2,500 logical ms
per active refresh (approximately 10×). There is no refresh backlog. The mock
clock pauses when the scene is inactive; no background recording claim is made.
Start/stop/mark/export have busy guards. Stop finalizes at the current logical
time, preserving partial post-capture. Fault scenarios run locally inside the app.
The physical adapter always remains disconnected. Module status comes from
explicit synthetic evidence; unobserved modules are labeled honestly.

## Screens and UX review

Dashboard: mock/accelerated-time labels, recorder/elapsed time, a small signal
set with units and quality, module evidence, latest incident, start/stop and a
large MARK INCIDENT button pinned above the tab bar. Stop requires a deliberate confirmation. Errors are
visible and markers are generated only by explicit actions.

Sessions: app-container history, duration/state, observation/event/incident
counts, size, incident links and finalized ZIP export. Incomplete sessions remain
preserved and are not offered as complete exports. Incidents: historical and live
incidents, severity/status, trigger references, requested/available coverage,
gaps and a paged canonical timeline with direct jumps around each trigger. Debug: next-run scenario selection, versions,
recorder/location and truthful source/transport information.

The layout uses standard SwiftUI navigation, scalable text, labeled controls and
native sharing. Refine hierarchy, terminology and interaction after trying healthy
and fault drives; the final consumer visual design is not being fixed here.

## Build and native evidence

Go module/tool/runtime pin: `golang.org/x/mobile`
`v0.0.0-20250911085028-6912353760cf` (official revision
`6912353760cf05c80a2c20d05cfba571cb0e66fa`, requires Go 1.24.0).
Both gomobile and gobind install this exact version; module sums are committed.
The module minimum remains Go 1.24.0 for the platform-independent core. Native
binding pins **Go 1.26.10**, verified against the official stable release listing.
The existing GitHub macos-15 workflow selects that compiler explicitly and reports
actual Xcode/SDKs. It builds a static ARM64 iOS Simulator XCFramework from source
before XcodeGen references it. Device slices/signing remain future work.

The workflow runs focused hosted XCTest cases on an available iPhone simulator:
real bridge invocation, changing snapshots, healthy/manual and voltage/TCM capture,
paged timeline order, native ZIP export, state mapping, duplicate-mark guard and
visible export errors. It copies ZIPs directly from the tested app container,
records simulator ID and commit, then validates and reconstructs those exact ZIPs
with Python. The native-acceptance artifact retains exports, validation output,
screenshot and xcresult. These exports are not substituted Linux fixtures.

Run from the repository root:

```bash
./scripts/test_all.sh
(cd core && go test -race ./mobile)
.tools/actionlint .github/workflows/ci.yml .github/workflows/ios-preview.yml
# macOS runner only:
./scripts/build_go_ios.sh
./scripts/generate_xcode_project.sh
```

Official references: [Go mobile binding](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile),
[XcodeGen source/resource and framework configuration](https://github.com/yonaskolb/XcodeGen/blob/master/Docs/ProjectSpec.md).

## Interactive acceptance procedure

After a successful preview upload, open the existing Appetize build. Confirm the
new four-tab dashboard launches, physical adapter remains disconnected, and a
healthy drive updates telemetry. At 01:00 logical time or later, tap MARK INCIDENT
once, allow 120 more logical seconds, stop and inspect its captured timeline.
Select voltage/TCM in Debug before starting a new drive. Use the dashboard
elapsed timer, not wall time: the synthetic voltage drop starts at 05:58,
and the third TCM timeout occurs at 06:00.9. One automatic incident should
appear around 06:01 without a manual mark. Allow the timer past 08:01 for
the post-trigger capture, or let the drive finish at 12:00. Voltage must
precede the communication trigger in that incident. The approximate 10×
playback rate can be slower in Appetize; a fixed wall-time estimate is not
an acceptance criterion. Export a finalized drive with the
native share sheet. Browser download availability is service-dependent; the
runner's actual native test export provides a separate verified retrieval path.
Do not claim an Appetize sharing interaction or observation from an upload alone.

Remaining Appetize checks require observed results. No live
accessory discovery, streams, commands, proprietary definitions or hardware
accuracy claims are introduced. Existing Milestone 4 bridge hardening (broader
lifecycle/error/version contracts, long-run reliability) remains future work.


## Recorded verification

Local `./scripts/test_all.sh`: all eight Go packages and vet pass, 59 Python
tests pass, seven simulator-packaging tests pass. `go test -race ./mobile` passes
on the final facade. Both workflows pass actionlint; shell syntax and Go formatting
checks pass. Linux CI for implementation commit `1a95b03` also
[passed](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40049868589).

The final fixed-compiler native run for implementation source `8a6f5c4`
[passed](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40051022426).
Actual destination: **iPhone 16 Pro, iOS 18.5, ARM64**, UUID
`DC4CD8B3-4457-4153-9087-A0D7A2F9BFD9`, Xcode **16.4** and Go **1.26.10**.
All three hosted XCTest cases passed with zero failures (2.414 s test execution,
separate from toolchain build and simulator boot). Native ZIP extraction and
Python reconstruction passed, then packaging and Appetize upload passed. The
corresponding [Linux CI passed](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40051022333).
The initial compile exposed explicit NSError handling for string-returning
bindings; the wrapper was corrected before these successful tests.

| Actual iOS export | Observations | Events | Incidents / triggers | Capture |
| --- | ---: | ---: | ---: | --- |
| healthy_drive, `mock-1792806439360090000` | 30,966 | 12 | 1 / 1 | captured |
| voltage_tcm_dropout, `mock-1792806440545480000` | 38,167 | 17 | 1 / 2 | captured |

The voltage export preserves `voltage_excursion` before `module_timeout` in one
incident; the manual capture spans 180 s and the merged voltage/TCM capture spans
182.9 s. These are actual app-container exports from the native XCTest, not Linux
substitutes. All three Python CLI commands also passed locally on these exact
native exports and actual incident IDs.

healthy_drive ZIP SHA256: `08e8802b0f001bea974b699b0ce95483f5abddb82f2398c6da5011ea5981046a`.
voltage_tcm_dropout ZIP SHA256: `be7f5e65399880692b53fad8954505bb95f69323a038048235f5d39309d1da96`.

[Native test/export artifact](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40051022426/artifacts/12543781312),
[simulator ZIP](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40051022426/artifacts/12543706411).
These expire after 14 days. Copies and local CLI results are retained under ignored
`build/milestone-3-native-evidence-8a6f5c4`. The artifact screenshot is the simulator
home screen after XCTest exits; it is not UI interaction evidence.

The [Appetize preview](https://appetize.io/app/nrnzclustneju2ltjppkozdkta) was updated
successfully. User observations for recording, manual marking, automatic fault
creation, share-sheet presentation and basic usability are recorded below.
The Windows computer-use skill was read and attempted for this task, but its
node_repl helper rejected the WSL workspace before initialization with
`sandboxCwd is not a local file URI: file:///home/kaletevans/dev/VehicleFlightRecorder`.
No automated Appetize launch or browser sharing result is claimed.

Architectural scope: no web frontend or cloud storage was added. Swift packages
finalized evidence; Go generates/records/captures it; Python analyzes actual native
exports. The native simulator artifact is unsigned and the framework intentionally
contains only the simulator ARM64 slice. Broader lifecycle, recovery, long-run,
version/error-contract and device-build hardening remain future Milestone 4 work.


### Native cgo alignment correction

A second native run on Go 1.24.0 exposed an intermittent crash in the generated
`Controller.IncidentJSON` cgo callback: `bulkBarrierPreWrite: unaligned arguments`.
The stack points to a 4-byte-aligned argument block during a Go write barrier,
matching the [upstream iOS cgo bug/fix](https://github.com/golang/go/commit/d5b950399).
The [canonical upstream issue](https://github.com/golang/go/issues/46893) and
[related report](https://github.com/golang/go/issues/75889) identify the compiler
alignment fix delivered in Go 1.26. Native builds now require Go 1.26.10 rather
than suppressing garbage collection, patching generated wrappers, or retrying away
the failure. The corrected run above passes native tests and actual-export reconstruction
with Go 1.26.10 and uploads that build to Appetize. Earlier native artifacts are
historical evidence, not a claim that the old compiler was reliable.

## Human testing feedback — 2026-10-28

The user reports recording and manual incident marking work, and basic usability
looks good with further refinement desired. The initial voltage/TCM attempt ended before the scheduled fault was observed.
After the six-minute logical-time checkpoint was clarified, the user repeated
the test and confirmed automatic incident creation works. The user considers
the minimal UI suitable for initial testing. The supplied screenshot confirms Captured status, two triggers, voltage at
+0.000 s followed by module timeout at +2.900 s, 60 s / 120 s configured
coverage and 182.9 s available span. The user reports the evidence-jump button
appears unresponsive. Code inspection shows it replaces the timeline page below
the fold without moving the viewport. The fix adds explicit scrolling to a
stable timeline anchor after loading and independent button styling for List
rows. Native build and renewed interactive verification are required before
closing that issue.

Export presents a native share sheet with Save to Files. The user reports the
file menu then disappears. Share presentation is observed; successful saving,
file retrieval, and browser download are unconfirmed. Do not treat menu
dismissal as proof of a saved ZIP.

### Simulator startup reliability follow-up

Preview run `40752847105` compiled the Go framework and Swift app, then remained
in the simulator selection/boot step. Its original `simctl bootstatus -b` had
no process timeout, and selection used the first available device without matching
its runtime to the selected SDK. Cancellation was requested, including forced
cancellation when the runner did not promptly stop; completed runner logs were
not yet available when this correction was prepared. The exact underlying
CoreSimulator hang is therefore unconfirmed.

The workflow now explicitly selects Xcode 16.4, creates a fresh iPhone 16 Pro on
the runtime matching that Xcode's simulator SDK, bounds each `xcrun` call and
limits boot completion to 240 seconds. The whole startup step is capped at eight
minutes, with runtime/destination/failure/device/process evidence included in the
native artifact. Startup failure still fails native acceptance; it does not skip
tests or publish an unverified preview. Focused synthetic tests check runtime
selection, unavailable-runtime failure and timeout propagation; only a macOS run
can verify real simulator startup.
