# Vehicle Flight Recorder

An iPhone application for preserving diagnostic evidence around intermittent
vehicle faults: timestamped telemetry, raw adapter traffic, manual/automatic
incidents, and export for Python reconstruction.

The first physical validation vehicle is a **2017 Alfa Romeo Giulia Quadrifoglio**
(2.9L twin-turbo V6, ZF 8-speed). Recording, standard OBD discovery and analysis
are shared infrastructure; verified manufacturer-specific knowledge belongs in
vehicle packs. The diagnostic MVP takes priority over the later Driver Companion.

## Current status

**October 29, 2026:** Milestones 0–4 are complete and merged into `master`.
[Milestone 5 is in draft PR #3](https://github.com/KaleTEvans/VehicleFlightRecorder/pull/3):
transport implementation and automated checks passed, including the authorized
offline acquisition preparation. Apple Developer enrollment approval, signing,
iPhone installation and actual MX+ acceptance remain pending.

| Milestone | Scope | Status / details |
| --- | --- | --- |
| 0 / 0.5 | Repository, remote macOS build and browser preview | Complete; [preview guide](docs/IOS_PREVIEW.md) |
| 1 | Go recorder, rolling buffer, mock drives and JSONL | Complete; [report](docs/MILESTONE_1.md) |
| 2 | Incident detection, capture windows and reconstruction | Complete; [report](docs/MILESTONE_2.md) |
| 3 | Native dashboard, sessions, incidents and export | Complete and merged; [report](docs/MILESTONE_3.md) |
| 4 | Bridge, lifecycle and device/simulator build hardening | Complete and merged; [report](docs/MILESTONE_4.md) |
| 5 | MX+ byte transport, connection evidence and offline preparation | Partially complete; [report](docs/MILESTONE_5.md) |
| 6 | Adapter initialization, standard OBD and live acquisition | Offline foundations prepared within M5; live integration pending; [handoff](planning/milestone_6.md) |
| 7 | Hands-off road logging, start confirmations and reliability | Planned; [handoff](planning/milestone_7.md) |
| 8 | Verified Giulia drivetrain signals and incident correlation | Planned; [handoff](planning/milestone_8.md) |

## What works today

- **Mock iOS recorder:** live readings, start/stop, one-tap manual incidents,
  automatic synthetic triggers, session history, evidence timelines and ZIP export.
- **Physical transport implementation:** discovery of `com.obdlink` accessories,
  bounded ExternalAccessory streams, connection generations, deadlines and raw
  connection captures. Its debug screen is selected on device builds; physical
  behavior still needs acceptance on the installed app.
- **Offline acquisition:** prompt/response parsing, supported-parameter bits,
  RPM/coolant/adapter-voltage decoding, bounded scheduling and simulated failure
  scenarios, with independently validated raw/timing/capability reports.
- **Python analysis:** validate/summarize sessions, reconstruct incidents, inspect
  connection captures and validate synthetic acquisition reports.

Simulator/Appetize always uses the mock recorder. Physical builds expose only an
explicitly enabled, once-per-connection firmware identity read (`STI`); opening a
connection sends no commands. The offline acquisition packages are not linked
into the app and enable no additional scanner requests.

Real vehicle polling, automatic trip start/stop and user-facing recording
confirmations are upcoming work. An uploaded simulator build or unsigned device
compile does not establish hardware connectivity or background recording.

## Develop and test in WSL

Keep this checkout and generated work under:

```text
/home/kaletevans/dev/VehicleFlightRecorder
```

Portable development requires Bash, Git, Go **1.24+**, and Python **3.11+** with
`venv`/pip. Initial bootstrap downloads dependencies into project-local caches
and creates `analytics/.venv`.

```bash
cd /home/kaletevans/dev/VehicleFlightRecorder
# If Go is installed here but absent from PATH:
export PATH="/usr/local/go/bin:$PATH"
./scripts/bootstrap.sh
./scripts/test_all.sh
```

Set `PYTHON=/path/to/python3` when invoking bootstrap to select another Python.
The test script runs Go tests/vet, Python tests (including actual Go CLI exports)
and build-script tests. It requires Go on `PATH`. Native framework builds use the
separately pinned **Go 1.26.10**, not merely the portable minimum.

See [development instructions](docs/DEVELOPMENT.md) for individual checks, native
build commands and toolchain configuration.

## Run a mock drive locally

From the project root after bootstrap:

```bash
mkdir -p build
(cd core && go run ./cmd/mock-session \
  --pack ../vehicle-packs/alfa-giulia-qv-2017 \
  --output ../build/healthy-demo)
analytics/.venv/bin/python -m vehicle_flight_recorder validate build/healthy-demo
analytics/.venv/bin/python -m vehicle_flight_recorder summarize build/healthy-demo
```

Use a new output directory on each run; existing evidence is never overwritten.
The default drive covers 12 logical minutes without waiting 12 real minutes and
writes 30,966 observations, nine events and no incidents. Its raw transport file
is legitimately empty because the source is synthetic waveforms.

Other mock scenarios are `voltage_tcm_dropout`, `tcm_dropout`, `mechanical_fault`,
`adapter_disconnect` and `manual_mark`; select one with `--scenario` and a new
output directory. They test the pipeline, not actual vehicle thresholds or faults.
Use an exported incident ID to reconstruct its timeline:

```bash
analytics/.venv/bin/python -m vehicle_flight_recorder incident \
  /path/to/session-directory INCIDENT_ID
```

See [mock recorder usage](docs/MILESTONE_1.md), [scenario/incident results](docs/MILESTONE_2.md)
and the [incident contract](docs/INCIDENT_MODEL.md).

## Run offline acquisition scenarios

This exercises the real Go parser/scheduler against a scripted adapter, without
Bluetooth, vehicle access or wall-time waits:

```bash
mkdir -p build/offline-preparation
(cd core && go run ./cmd/offline-acquisition \
  --scenario edge_cases \
  --output ../build/offline-preparation/edge-cases.json)
analytics/.venv/bin/python -m vehicle_flight_recorder validate-acquisition \
  build/offline-preparation/edge-cases.json
```

Use a new output filename on each run. Scenarios cover 12 logical seconds:

| Scenario | Expected synthetic evidence |
| --- | --- |
| `healthy` | 37 transactions, 51 observations |
| `edge_cases` | 36 transactions, 46 observations; timeout, late reply, cancellation and reconnect |
| `partial_replies` | 37 transactions, 34 observations; 17 partial outcomes retaining valid ECU replies |

Reports preserve raw bytes, transaction/generation references, receipt timing,
unknown ECU measurement time, capability states and requested/achieved rates.
These fixture results are not measured vehicle capabilities. Read the
[offline acquisition contract](docs/OFFLINE_ACQUISITION.md) for supported formats,
limits, references and remaining live integration.

## Preview the iOS app in a browser

WSL runs the portable tooling; GitHub macOS runners build the native SwiftUI app.
Run **Actions → iOS Simulator Preview → Run workflow** in
[GitHub](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/workflows/ios-preview.yml),
selecting the intended branch. The workflow builds both framework slices,
compiles unsigned device targets, runs native unit/UI tests, validates actual
native-generated exports and publishes `VehicleFlightRecorder-ios-simulator`.
The simulator ZIP contains the `.app` at its root and has 14-day artifact retention.

Optional Appetize upload uses `APPETIZE_API_TOKEN`; `APPETIZE_PUBLIC_KEY` updates
an existing preview. Disable upload for build/test-only runs. For manual upload,
download the artifact, extract its outer GitHub ZIP, and upload the inner simulator
ZIP. The [existing Appetize preview](https://appetize.io/app/nrnzclustneju2ltjppkozdkta)
contains the previously accepted mock build; each upload/run has separate provenance.
Start a fresh browser session after updating a build. Review access settings
before sharing a runnable link.

The app identifies itself as **MOCK / SIMULATOR**. Recording, incident navigation
and share-sheet presentation have human acceptance; browser file retrieval has
limitations, so actual exported files are validated separately in native CI.
Appetize cannot exercise the physical scanner. Full details:
[preview and troubleshooting guide](docs/IOS_PREVIEW.md).

## Install on the physical iPhone

The separate **Signed iPhone build** workflow prepares a signed device IPA and
optionally uploads it for TestFlight processing. Its definition must first be
available on the default branch to use manual dispatch. Enrollment approval,
signing assets, App Store Connect setup, installation and physical acceptance
are still pending; [follow the device setup guide](docs/IOS_DEVICE.md).

The shared native configuration pins Go/x/mobile and the iOS 17 deployment floor.
Preview uses Xcode 16.4; the separate device profile uses Xcode 26.0.1 / iOS 26 SDK.
See [shared pins](scripts/mobile_toolchain.env) and [device profile](scripts/device_toolchain.env).
Approved identity: **Vehicle Flight Recorder 0.1.0**,
`com.kaleevans.vehicleflightrecorder`, iAP `com.obdlink`, MFi PPID `221699-0001`.
Manufacturer approval is already recorded; installed-device behavior is a separate gate.

Exported standalone connection files can be inspected with:

```bash
analytics/.venv/bin/python -m vehicle_flight_recorder validate-connection \
  /path/to/connection-capture.jsonl
```

## Architecture and documentation

| Location | Responsibility |
| --- | --- |
| `ios/` | SwiftUI, native lifecycle, ExternalAccessory streams and export/share presentation |
| `core/` | Generic recorder/incidents, mock generation, pack loading, mobile facade and isolated offline OBD/acquisition packages |
| `analytics/` | Python validation, summaries and provenance-aware reconstruction |
| `vehicle-packs/` | Pack definitions; the shipped Alfa mock signals/rules remain explicitly synthetic |
| `scripts/` / `.github/workflows/` | Local checks, framework/project generation, native CI, previews and signed-device preparation |
| `docs/` / `planning/` | Implemented contracts/evidence and future milestone handoffs |
| `testdata/` | Scenario fixture guidance; automated tests also create synthetic evidence directly |

Swift owns native integration and presentation; Go owns recording/protocol/domain
logic; Python owns offline analysis. Transport carries bytes without deciding
vehicle health. Preserve the distinction between observations, correlation and
diagnostic hypotheses.

- [MVP implementation plan](planning/alfa_flight_recorder_mvp_plan.md)
- [Milestone handoffs: 0](planning/milestone_0.md), [0.5](planning/milestone_0.5.md),
  [1](planning/milestone_1.md), [2](planning/milestone_2.md), [3](planning/milestone_3.md),
  [4](planning/milestone_4.md), [5](planning/milestone_5.md), [6](planning/milestone_6.md),
  [7](planning/milestone_7.md), [8](planning/milestone_8.md)
- [Data schema](docs/DATA_SCHEMA.md), [bridge contract](docs/BRIDGE_CONTRACT.md),
  [live timing design](docs/LIVE_TIMING.md), [vehicle profile](docs/VEHICLE_PROFILE.md)
- [Driver Companion V2 plan](planning/driver_companion_v2_plan.md) — later garage,
  cockpit, location/motion and performance features, with their own evidence gates
- [Enterprise roadmap](planning/vehicle_health_enterprise_architecture.md) — directional future work

The MVP is read-only: no DTC clearing, ECU coding/configuration/calibration,
actuation, undocumented requests or arbitrary CAN writes. Verified reads must be
bounded, logged and rate-limited. Configure and test physical access while parked;
the UI must not require attention while driving. The active milestone handoff
and current human instructions define implementation scope.
