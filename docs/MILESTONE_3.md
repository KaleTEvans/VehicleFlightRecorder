# Milestone 3 — native dashboard and working mock bridge

Status: implementation in progress. Cloud native build, native export validation,
and observed Appetize interaction must pass independently before acceptance.
Milestone 4 not started. A UX review follows this functional milestone.

## Responsibilities and runtime

The existing Go recorder/detectors remain authoritative. `mobile.NewController`
loads the explicitly mock-only pack and an app-container Sessions directory.
Its string/scalar API exposes Start, Advance, Stop, Mark, Snapshot, SessionsJSON,
IncidentJSON (200-row pages), ExportPath and Close. Every controller operation
holds one mutex; only an immutable mock generator runs on a separate goroutine,
with a bounded 64-record channel and cancellation joined before stop/close.
Native calls run through a single Swift actor, away from the UI actor.

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
large MARK INCIDENT button. Stop requires a deliberate confirmation. Errors are
visible and markers are generated only by explicit actions.

Sessions: app-container history, duration/state, observation/event/incident
counts, size, incident links and finalized ZIP export. Incomplete sessions remain
preserved and are not offered as complete exports. Incidents: historical and live
incidents, severity/status, trigger references, requested/available coverage,
gaps and a paged canonical timeline. Debug: next-run scenario selection, versions,
recorder/location and truthful source/transport information.

The layout uses standard SwiftUI navigation, scalable text, labeled controls and
native sharing. Refine hierarchy, terminology and interaction after trying healthy
and fault drives; the final consumer visual design is not being fixed here.

## Build and native evidence

Go module/tool/runtime pin: `golang.org/x/mobile`
`v0.0.0-20250911085028-6912353760cf` (official revision
`6912353760cf05c80a2c20d05cfba571cb0e66fa`, requires Go 1.24.0).
Both gomobile and gobind install this exact version; module sums are committed.
The existing GitHub macos-15 workflow selects Go via core/go.mod and reports
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
Start voltage/TCM from Debug and allow it to complete: voltage must precede the
communication trigger in one captured incident. Export a finalized drive with the
native share sheet. Browser download availability is service-dependent; the
runner's actual native test export provides a separate verified retrieval path.
Do not claim an Appetize sharing interaction or observation from an upload alone.

Appetize interaction and browser sharing remain pending until observed. No live
accessory discovery, streams, commands, proprietary definitions or hardware
accuracy claims are introduced. Existing Milestone 4 bridge hardening (broader
lifecycle/error/version contracts, long-run reliability) remains future work.
