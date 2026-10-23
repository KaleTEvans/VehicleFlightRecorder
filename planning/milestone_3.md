# Codex Handoff — Vehicle Flight Recorder, Milestone 3 Only

## Assignment

Implement **Milestone 3 — iOS dashboard** in the existing repository:

```text
~/dev/VehicleFlightRecorder
```

This is a handoff for a future implementation task. **Prerequisites: Milestones 1
and 2 are implemented and accepted.** Inspect their real APIs, schemas, scenarios,
tests and command results before designing the native boundary. If either is
missing, report the dependency instead of implementing preceding milestones here.
Preserve completed Milestones 0/0.5. Stop before Milestone 4 integration hardening.

## Authoritative context

Read the full plans at their current locations:

```text
planning/alfa_flight_recorder_mvp_plan.md
planning/vehicle_health_enterprise_architecture.md
```

Priority: human's active instructions, this handoff, MVP plan, enterprise roadmap.
Read `planning/milestone_1.md`, `planning/milestone_2.md`, the current README/docs,
`git status`, and existing source/build/workflow configuration. Focus on MVP
sections 2, 4–5, 14–16, 19–22 and the enterprise engineering-environment guidance.
Preserve legitimate changes and report architectural deviations explicitly.

Approved identity remains fixed:

```text
App:                 Vehicle Flight Recorder
Version:             0.1.0
Bundle identifier:   com.kaleevans.vehicleflightrecorder
iAP protocol:        com.obdlink
MFi PPID:            221699-0001 (submission metadata, not runtime domain logic)
```

Inspect, rather than recreate, the current XcodeGen project and scheme
`VehicleFlightRecorder`, iOS 17 floor, SwiftUI source, transport seam, and scripts.
Milestone 0.5 originally previewed only the static SwiftUI shell; it did not link
the Go recorder. Check the repository's current state before choosing integration
steps. Keep Go module `vehicleflightrecorder.local/core` and Python package
`vehicle_flight_recorder`; do not introduce legacy Alfa names in generic APIs.

## Environment and integration boundary

The workstation is Windows/WSL; native compilation runs on GitHub macOS runners.
The existing `.github/workflows/ios-preview.yml` builds an unsigned ARM simulator
app, publishes a ZIP artifact, and optionally uploads it to Appetize. Review
`docs/IOS_PREVIEW.md` for the actual workflow and secret names:
`APPETIZE_API_TOKEN` and optional `APPETIZE_PUBLIC_KEY`. Do not recreate accounts,
expose secrets, or replace the workflow with a web/React dashboard.

**Milestone 3 needs real native-to-Go runtime integration.** A full mock drive in
the iOS app cannot be satisfied by hard-coded Swift samples, a remote Go server,
or a static dashboard. Implement the minimum bridge and simulator-compatible
XCFramework necessary now. Milestone 4 remains responsible for broader bridge
hardening, versioning/error contracts, lifecycle coverage and reproducibility work;
that sequencing must not be used to defer a functional Milestone 3 bridge.

## Required work

### 1. Minimal working Go ↔ Swift integration

Expose the existing Go recorder/scenario/session functionality through the small
`core/mobile` facade. Start from the actual Milestones 1–2 APIs and the MVP section
5 suggestions, using strings/JSON, bytes and scalars where practical. Expose only
the operations needed to start/stop a session/scenario, mark an incident, retrieve
dashboard/session/incident snapshots, and locate finalized exports.

Keep one clear owner for the core instance. Define serialization/threading and
lifecycle rules so view refreshes, marks and stop operations do not race. Translate
Go errors into visible native state; prevent unbounded polling/queues and avoid
blocking the UI thread. Do not silently swallow failed starts, writes or exports.

Use `gomobile bind` on the macOS runner for the correct simulator architecture and
link/embed the generated XCFramework through the existing project definition.
Update `scripts/build_go_ios.sh` and the preview workflow only as needed. Select
matching, recorded gomobile/gobind revisions compatible with the chosen Go/Xcode
toolchain; avoid installing two different moving revisions. Keep generated
frameworks/build products ignored and build them from source on the runner.

A clean cloud checkout must actually build the framework before the Swift app
when the target references it. Keep core Go/Python tests independent of Xcode.
Do not install Apple SDKs in WSL or claim Linux produced a native iOS build.

### 2. Dashboard and session controls

Extend the existing SwiftUI app with a sparse, readable dashboard. Display core
connection/mock state, recorder state, elapsed logical drive time, a small set of
current signals/units, relevant module health, last incident, and recording errors.
Use Go snapshots as the source of truth. Show missing/stale/invalid samples honestly
instead of filling them with believable fabricated values.

Provide deliberate start/stop behavior and accessible navigation. Keep **MOCK /
SIMULATOR** unmistakable in simulator previews. A local synthetic Go scenario can
produce telemetry while the native byte-transport shell stays disconnected;
distinguish mock generation from physical adapter connectivity. Never show MX+
connected just because a synthetic drive is running.

### 3. Sessions, incidents and developer debug views

- **Sessions:** list actual stored sessions with start/end, duration, sample and
  incident counts, state/size where available, and export action. Handle empty,
  active, stopped and incomplete sessions.
- **Incidents:** list actual Go incidents with trigger, severity, timestamp and
  capture state. A detail view shows an ordered evidence timeline and available
  pre/post coverage, including gaps/partial captures. Minimal plots are optional;
  timeline correctness is required.
- **Debug:** select the existing healthy/fault scenarios, show core/pack versions,
  recorder state, local session location and relevant existing health counters.
  Do not invent counters or imply raw adapter traffic exists when it does not.

Keep this to MVP screens and state mapping; do not build a polished scan tool,
new settings platform, or enterprise UI. SwiftUI views must not refer directly to
EAAccessory/EASession or implement vehicle-specific rules.

### 4. MARK INCIDENT

Provide one large, accessible MARK INCIDENT control during an active session.
Route it through the Go facade into the actual Milestone 2 manual marker and let
Go own timestamping, trigger linking and capture windows. Reflect the resulting
incident state and any errors. Haptic feedback can be a thin native enhancement;
simulators do not prove physical haptics or driving usability.

Define disabled/error behavior when no session is active. Do not let incidental
view recomputation generate duplicate marks. Incident triggers/detection, window
completion and overlap policy remain Go logic.

### 5. Real session export

Use an iOS app-container directory passed to Go for recording. Export a real
finalized session in the existing Python-readable layout, preserving metadata,
observations, events, incident records and raw evidence files. Use the native
share/export UI; any temporary archive creation is packaging, not domain analysis.

Document what happens for active/incomplete sessions; prefer finalized sessions
unless a safe snapshot is deliberately supported. Avoid exposing arbitrary paths,
leaving archive failures invisible, or changing schemas to fit a Swift-only model.

Validate the actual iOS-generated export with the existing Python `validate`,
`summarize`, and `incident` commands. Appetize share/download features depend on
the service: if native sharing cannot yield a downloadable file there, document
and implement a narrow mock-only simulator export retrieval path using an
available supported service/runner mechanism. Do not substitute a separately
generated Linux fixture and call it an iOS export.

If browser-only retrieval is unavailable, report the export observation separately
as pending, with a concrete later device/simulator procedure. Do not add a cloud
storage backend or a new web frontend to bypass the limitation.

### 6. Functional preview pipeline

Preserve manual workflow dispatch, `contents: read`, unsigned simulator builds,
approved identity checks, correct `.app`-at-ZIP-root packaging, and artifact upload
before optional Appetize upload. Keep token-free manual upload working. Use only
synthetic data in previews/artifacts and preserve Appetize account access settings.

The preview must execute the Go mock scenario locally inside the app. Maintain
bounded UI update frequency, use the core's logical clock for accelerated drives,
and clearly label accelerated time. Do not promise real accessory background
execution or uninterrupted recording after the simulator session ends.

### 7. Documentation

Update development, bridge/build, export and preview instructions with exact
commands, chosen toolchain revisions, and the new interactive flow. Document
which responsibilities belong to Swift, Go and Python. Describe any remaining
Milestone 4 hardening work without using it to hide missing functional integration.

## Required verification

Run and report:

- `gofmt`, Go tests/vet (and race checks for introduced concurrency), Python pytest,
  and `./scripts/test_all.sh` in WSL.
- Workflow/actionlint and shell/configuration checks for changed build inputs.
- Focused native tests for snapshot/view-model state mapping, bridge invocation,
  lifecycle/error states, duplicate mark prevention, and export URLs/packaging.
  Run native tests on the GitHub macOS runner with an available simulator; report
  the actual destination/result rather than claiming local WSL execution.
- A clean macOS workflow that builds the real Go simulator framework and SwiftUI
  app, packages/publishes the simulator ZIP, and uploads when configured.
- Observed Appetize launch of the newly built app: start a healthy mock drive,
  verify changing Go-backed telemetry, mark an incident, allow pre/post capture,
  stop the drive, and inspect the saved session/incident timeline.
- At least one existing fault scenario in the native app with the expected
  captured incident and evidence ordering; do not rely only on a static mock label.
- Export an actual native session and validate/summarize it in Python; reconstruct
  an incident by its actual ID. Record the artifact's provenance and any browser
  retrieval limitations explicitly.

A workflow passing is not an observed interactive test. If automation is unavailable,
use direct user observation with explicit steps/results and distinguish it from
automated evidence. If credentials, service quota, browser control or file retrieval
block a check, complete independent work and mark that acceptance item pending.

## Milestone boundaries and safety

No live EASession/discovery, adapter initialization, AT/ST/OBD/CAN commands, real
ECU polling, CoreBluetooth workaround, signing/TestFlight/App Store delivery,
cloud backend, account system, AI reports, or new vehicle support. Preserve the
approved External Accessory configuration without interpreting it as proof of
hardware access. Simulator mock execution cannot validate physical MX+, background
accessory streams, phone power behavior or driver safety.

Go owns recorder, telemetry processing, scenarios, incident decisions, storage
and profile meaning. Swift owns UI, presentation mapping, native lifecycle,
transport seam and share/export UI. Python owns offline analytics. No automatic
DTC clearing, coding, calibration, configuration, actuation or arbitrary CAN writes.
Future active reads must be isolated/logged/rate-limited and fail closed when
transport state is uncertain. UI must minimize future driver interaction.

## Acceptance and reporting

Milestone 3 is complete when a clean remote build runs the real Go-backed mock
drive through the native dashboard in Appetize, sessions/incidents/manual marks
work, and an actual generated session can be exported for Python validation.
Unobserved browser behavior or unverified native export keeps the corresponding
acceptance item pending. Do not mark the milestone complete based on screenshots
of hard-coded telemetry or a placeholder bridge.

Report:

1. COMPLETE, PARTIALLY COMPLETE, or BLOCKED and remaining acceptance items.
2. Files changed, actual mobile API/ownership model and toolchain revisions.
3. Local and native test commands/results; real workflow/build-artifact URLs.
4. Actual Appetize link, observed interactive flow, and who observed it.
5. Provenance of the native export and Python validation/reconstruction results.
6. Architecture deviations, browser/device limits, and deferred Milestone 4 work.
7. Explicitly: **Milestone 4 not started.**
