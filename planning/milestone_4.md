# Codex Handoff — Vehicle Flight Recorder, Milestone 4 Only

## Assignment

Implement **Milestone 4 — Go XCFramework integration hardening** in the existing
repository:

```text
/home/kaletevans/dev/VehicleFlightRecorder
```

This document prepares a future implementation task; its creation does not start
Milestone 4. Milestone 3 is complete and accepted. Inspect the current checkout,
`git status`, implementation, tests and recorded results before making changes.
Preserve legitimate local changes. Do not recreate the dashboard, recorder,
incident engine, exports or remote preview. Stop before physical MX+ integration.

## Authoritative context and product boundary

Read the complete current plans:

```text
planning/alfa_flight_recorder_mvp_plan.md
planning/vehicle_health_enterprise_architecture.md
```

Priority: active human instructions, this handoff, MVP plan, enterprise roadmap.
Read `planning/milestone_3.md`, `docs/MILESTONE_3.md`, `docs/DATA_SCHEMA.md`,
`docs/INCIDENT_MODEL.md`, `docs/DEVELOPMENT.md`, `docs/IOS_PREVIEW.md`, README and
current build/workflow sources. Focus on MVP §§1.2, 2, 5–7, 15, 20–22 and the
enterprise client, evidence, adapter and reliability boundaries.

The MVP is diagnostic-first: a generic recorder/standard diagnostic foundation
with the Giulia QV as its first physical validation vehicle. Keep vehicle-specific
meaning in packs. Driver Companion V2, GPS/motion, performance timing, cockpit
redesign, cloud services and enterprise expansion are not authorized here.

Preserve the approved identity:

```text
App/version:          Vehicle Flight Recorder / 0.1.0
Bundle identifier:    com.kaleevans.vehicleflightrecorder
iAP protocol:         com.obdlink
MFi PPID:             221699-0001 (submission metadata)
Minimum iOS:          17.0
Go module:            vehicleflightrecorder.local/core
Python package:       vehicle_flight_recorder
```

## Starting point — inspect rather than assume missing

Milestone 3 already provides a functional `core/mobile.Controller`, a serialized
Swift `RecorderService` actor, dashboard/session/incident/debug views, native ZIP
export and working trigger-evidence scrolling. Go owns recording/detection;
Swift owns native presentation and packaging; Python validates actual exports.
The mock generator is the data source; `MockVehicleTransport` remains an inert
byte-transport stub. Do not replace the generator with fake Swift telemetry.

Existing build inputs at handoff creation:

- `scripts/mobile_toolchain.env`: native compiler `go1.26.10` and matching
  gomobile/gobind/runtime revision `v0.0.0-20250911085028-6912353760cf`.
- `core/go.mod`: minimum Go 1.24.0; native compiler pin is deliberately different.
- `scripts/build_go_ios.sh`: macOS binding, currently simulator ARM64 only.
- `ios/project.yml`: XcodeGen source/resources/framework/test configuration.
- `.github/workflows/ios-preview.yml`: Xcode 16.4, real framework/app build,
  bounded fresh SDK-matched simulator startup, hosted/native UI tests, actual
  session export validation, app-only packaging and optional Appetize upload.
- `scripts/boot_ios_simulator.py`: per-command deadlines and startup diagnostics.

Earlier Go versions exposed an intermittent iOS cgo alignment crash. Preserve the
fixed compiler requirement; do not downgrade it, suppress GC or retry away native
crashes. Verify official compatibility documentation before deliberately changing
pins. Record exact versions and evidence, rather than selecting moving `latest`.

## Required work

### 1. Reproducible native artifacts

Harden the existing script/configuration instead of adding a second build system.
Use one source of truth for generator/runtime/compiler pins and detect mismatches
before compiling. Report actual Go, gomobile/gobind, Xcode, SDK and architecture.
Build from a clean checkout without developer-local frameworks or generated
projects. Keep generated artifacts and caches ignored; use project-local caches.
If caching is introduced, key it by all relevant sources, module sums, pins,
SDK/architecture and build inputs; stale cache reuse must not masquerade as a clean
build. Reproducibility means repeatable compatible outputs, not an unsupported
claim of byte-identical Apple binaries.

Retain the simulator ARM64 path. Prepare the framework's iOS-device ARM64 build
path needed by Milestone 5 and validate it with an unsigned device-target compile
on macOS. Keep device and simulator slices correctly identified; never pass a
simulator app to a phone or Appetize a device IPA. Signing, provisioning and
physical installation belong to the Milestone 5 prerequisite, not to this task.

### 2. Small, versioned bridge contract

Inventory existing public calls and document input/output schemas, ownership,
threading, state transitions and failure behavior. The API already supports mock
start/advance/stop/mark, snapshots, sessions, paged evidence, export paths and
close. Keep a small strings/JSON/bytes/scalars boundary; no exported Go object
hierarchy or vehicle-specific Swift rules.

Give the bridge an explicit compatibility contract separate from app, core,
telemetry schema, decoder and pack versions. Check compatibility before recording
and expose a clear unsupported-version error. Keep existing mock exports readable;
version any necessary contract change and test both supported and rejected
versions. Do not confuse a display-only core version with negotiated compatibility.

Specify byte ownership and bounded inbound/outbound delivery for the future live
transport, including cancellation and connection generations. Implement only the
seams exercised by current tests or the immediate Milestone 5 boundary; do not
invent an enterprise transport framework or implement OBD commands now.

### 3. Serialization, lifecycle and resource ownership

Audit the actor/mutex/channel ownership model. Ensure one authoritative session
owner, bounded work/queues, cancellable refreshes, producer cancellation/join,
and no UI-thread blocking or use-after-close callbacks. Define behavior for
repeated/concurrent start, stop, mark, refresh, export and close. Avoid deadlocks
between native callbacks, the Swift actor and the Go lock. An old task must not
mutate a new session after stop/restart.

Retain current mock semantics: logical time pauses while inactive; returning to
the foreground does not invent measurements or catch up by fabricating samples.
Exercise scene changes, view recreation, repeated sessions and interrupted app
relaunch using native tests. This is lifecycle hardening, not proof of real
ExternalAccessory background recording or universal wake-on-connect.

Specify interrupted-session handling explicitly. Preserve original evidence and
honest incomplete labels. Any recovery implementation must be narrow, tested,
non-destructive and distinguish recovered/incomplete from normally complete;
never overwrite corrupt rows or silently finalize a recording on launch.

### 4. Error translation, bounded data and storage behavior

Translate bridge failures into stable categories plus useful diagnostic context:
invalid input/state, unsupported version, missing pack, transport unavailable,
I/O failure and cancellation. Native string-returning methods retain explicit
NSError pointers in current bindings; test generated signatures instead of
assuming every Go error automatically becomes a throwing Swift method.

Do not silently return an empty successful snapshot, drop a failed mark/export,
or allow stale error state to hide recovery. Preserve valid zero values and
missing/stale/invalid distinctions. Exercise malformed/oversized JSON, invalid
identifiers/paths, paging limits, large history, missing/corrupt files and write/
finalization failures. Enforce bounds without truncating authoritative evidence
silently; report coverage loss and overflow explicitly.

### 5. Prepare live timing without implementing live acquisition

Document the current single-source, nondecreasing-time/sequence ingestion contract
and the proposed live mapping required by MVP §6.1.1. Distinguish request-send,
response-receive, optional measurement time, monotonic elapsed time and session
wall time. Refreshing a retained value must not create a new observation.

Identify migration points in Go/Python/native models, raw records and exports.
Define how clock changes, reconnects and unavailable source timestamps will be
represented. Defer implementation of actual live timing/protocol acquisition to
Milestone 6. GPS, motion, asynchronous source fusion and speculative multi-source
infrastructure remain out of scope.

### 6. Regression coverage and documentation

Keep the accepted UI behavior, especially evidence jumps, accessible controls,
scenario selection, actual export and explicit simulated-source labeling.
The native UI test must continue to verify viewport movement for both trigger
buttons and repeat selection; a successful data query is not a scrolling test.
No visual redesign is needed.

Write `docs/MILESTONE_4.md` and update bridge/build/development instructions with
exact APIs, compatibility rules, lifecycle/resource policy, build commands,
artifact slices and remaining Milestone 5 prerequisites. Record the duration and
configuration of a representative repeated-session/long-run test; do not claim
unbounded reliability from a brief successful loop.

## Required verification and acceptance

| Check | Evidence required |
| --- | --- |
| Linux core/analytics | Go formatting/tests/vet, focused race checks for concurrency, Python pytest, `./scripts/test_all.sh`; all mock and legacy export contracts retained |
| Build configuration | actionlint, shell/config tests; mismatched pins/versions fail clearly |
| Clean macOS simulator | Source-built framework and app, actual toolchain/slices recorded, native unit/UI tests pass |
| Device preparation | Real unsigned device-target framework/app compile with correct architecture; no claim of signing/install/hardware validation |
| Lifecycle/errors | Repeated/concurrent operations, cancellation/close, scene transitions, injected I/O/contract failures and honest incomplete-session handling |
| Native exports | Actual native-created ZIPs pass Python validate/summarize/incident reconstruction; record commit, destination and artifact provenance |
| Interactive regression | Updated Appetize launch, mock recording, incident navigation and native share presentation; distinguish automation from human observation |

Preserve workflow dispatch, minimal permissions, secret handling, bounded simulator
startup/diagnostics, `.app`-at-ZIP-root checks, application-only packaging and
artifact publication before optional Appetize upload. Do not recreate account
setup or expose tokens. Existing browser download limitations are documented;
use runner-extracted native exports for independent file validation.

Milestone 4 is complete when the existing integration has a tested version/error/
lifecycle contract, reproducible clean native builds and no regression in accepted
mock flows. Hardware/provisioning prerequisites remain explicitly future work.
Missing actual native evidence is pending, not replaced by WSL compilation.

## Boundaries and completion report

No EAAccessory discovery, EASession, real adapter initialization, diagnostic polling,
new vehicle definitions, automatic real-drive start, GPS/motion, V2 UI, cloud or AI.
Do not invent physical observations or treat capability declarations as proof.
Do not start Milestone 5 as part of this assignment.

Report COMPLETE / PARTIALLY COMPLETE / BLOCKED with exact remaining items; changed
files/API and compatibility decisions; commands/results and actual native build,
artifact and preview links; lifecycle/load-test configuration; observed versus
unobserved behavior; unsigned device preparation versus physical validation; and
any deviations. End explicitly: **Milestone 5 not started.**
