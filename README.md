# Vehicle Flight Recorder

A read-only vehicle flight recorder intended to preserve evidence around
intermittent faults. The initial reference vehicle is a 2017 Alfa Romeo Giulia
Quadrifoglio, 2.9L twin-turbo V6 with ZF 8-speed transmission.

## Status

Milestone 0 bootstrap is complete. Milestone 0.5 adds a manually triggered GitHub
macOS build and optional Appetize browser preview of the existing SwiftUI shell.
The [remote simulator build, artifact publication, and Appetize upload have passed](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40019983986).
The [browser preview](https://appetize.io/app/nrnzclustneju2ltjppkozdkta) launched
successfully and the user confirmed all four expected status labels on October 23,
2026. **Milestone 0.5 is complete.** There is no
recorder, telemetry generation, analysis CLI, or live vehicle communication yet.

## Architecture and layout

- `ios/`: SwiftUI, native lifecycle, raw-byte transport seam, future Go bridge/export UI.
- `core/`: Go module for the future generic telemetry, recorder, incidents, vehicle
  profiles, mock scenarios, and storage; `mobile` is the native entry point.
- `analytics/`: installable Python package for future offline analytics.
- `vehicle-packs/`: vehicle-specific data; the first pack has metadata and empty definitions.
- `docs/`: development workflow and concise data/profile/incident design notes.
- `scripts/`: bootstrap, tests, macOS project generation and future XCFramework build.
- `testdata/`: reserved fixture directories; `.github/workflows/`: Linux checks.

Swift stays thin, Go owns domain logic, and Python owns offline analysis. Transport
handles raw bytes; it must not interpret vehicle diagnostics.

## Local setup

Requires Bash, Git, Go 1.24+, and Python 3.11+ with `venv`/pip. Initial Python
dependency installation needs package-index access. No system packages are installed.

```bash
cd ~/dev/VehicleFlightRecorder
./scripts/bootstrap.sh
./scripts/test_all.sh
```

If needed, add the existing Go installation to `PATH` and select Python explicitly:

```bash
export PATH="/usr/local/go/bin:$PATH"
PYTHON="$HOME/.pyenv/versions/3.11.9/bin/python3" ./scripts/bootstrap.sh
./scripts/test_all.sh
```

Individual checks:

```bash
(cd core && go test ./... && go vet ./...)
(cd analytics && .venv/bin/python -m pytest)
```

## iOS browser preview from Windows/WSL

Push to [the GitHub repository](https://github.com/KaleTEvans/VehicleFlightRecorder),
then select **Actions → iOS Simulator Preview → Run workflow**. The workflow uses
`macos-15`, generates our XcodeGen project, and builds the real app for ARM iOS
Simulator without signing. Every successful build publishes artifact
`VehicleFlightRecorder-ios-simulator` with 14-day retention.

For automatic Appetize upload, add repository secret `APPETIZE_API_TOKEN`;
optional secret `APPETIZE_PUBLIC_KEY` updates an existing preview. Without a token,
the build still passes: download the artifact from the run, extract it once, and
upload its inner `VehicleFlightRecorder-simulator.zip` to Appetize manually.
Successful uploads add a preview link to the run summary. Review Appetize access
settings before sharing: build links can allow anyone holding the link to run the
app. Use mock data only. Upload can be disabled in the manual workflow form.

The initial screen must show **MOCK / SIMULATOR**, **Not Connected**, and
**Recorder: Not Started**. A successful upload still needs an observed launch.
Simulator bundles cannot be installed on an iPhone and cannot test the MX+.

See [the complete preview walkthrough](docs/IOS_PREVIEW.md) for initial Git setup,
secrets, artifact download, browser checks, and troubleshooting. WSL itself does
not run Xcode; GitHub provides the Mac. Optional local Mac setup and future Go
framework steps remain in [development instructions](docs/DEVELOPMENT.md).

## Safety and next steps

The default mock transport is disconnected. Both transport shells reject writes.
Future vehicle access remains read-only: no automatic DTC clearing, ECU coding,
configuration, calibration, actuation, undocumented commands, or arbitrary CAN
writes. Active reads must be isolated, logged, rate-limited, and fail closed.

After Milestone 0.5 remote verification, **Milestone 1** requires a separate instruction:
canonical observations/events,
session lifecycle, JSONL persistence, rolling buffer, deterministic healthy mock
drive, and a Python session summary CLI. None is implemented here.

Authoritative plans (kept under their original names):

- [MVP implementation plan](planning/alfa_flight_recorder_mvp_plan.md)
- [Enterprise architecture](planning/vehicle_health_enterprise_architecture.md)
- [Milestone 0.5 assignment](planning/milestone_0.5.md)

The active milestone assignment takes priority over broader implementation suggestions
in the plans. Enterprise services and expanded vehicle coverage remain future work.
