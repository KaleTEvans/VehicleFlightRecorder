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
2026. Milestone 1 now provides a Go recorder, deterministic healthy mock drive,
JSONL session storage, rolling buffer, and Python validation/summary CLI. These
run locally from the command line; **Milestone 1 is complete** under its Go/Python
acceptance criteria. Milestone 3 now implements the native dashboard and Go bridge; cloud and interactive
acceptance status is tracked in [Milestone 3](docs/MILESTONE_3.md).
Milestone 2 adds deterministic incident detection, capture windows, fault scenarios
and Python timeline reconstruction. See [Milestone 2 results](docs/MILESTONE_2.md).
Live vehicle communication remains future work.

## Architecture and layout

- `ios/`: SwiftUI, native lifecycle, raw-byte transport seam, future Go bridge/export UI.
- `core/`: generic telemetry/session models, recorder/buffer, healthy mock generator,
  incident detectors/capture, pack validation, storage, and the `mock-session` command; `mobile` supplies the serialized native recorder facade.
- `analytics/`: installable Python validator and deterministic summary CLI.
- `vehicle-packs/`: vehicle-specific data; the first pack contains six healthy signals and three scenario-scoped synthetic signals.
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

## Record and inspect a healthy mock drive

With Go on PATH and bootstrap complete, run from the repository root:

```bash
(cd core && go run ./cmd/mock-session \
  --pack ../vehicle-packs/alfa-giulia-qv-2017 \
  --output ../build/healthy-demo)
analytics/.venv/bin/python -m vehicle_flight_recorder validate build/healthy-demo
analytics/.venv/bin/python -m vehicle_flight_recorder summarize build/healthy-demo
```

The output directory must be new. The default seed, start time and session ID are
fixed: the 12-minute logical drive writes 30,966 observations, nine events and no
incidents without waiting 12 real minutes. Raw transport and incident files exist
but remain empty for this healthy scenario. See [the Milestone 1 walkthrough](docs/MILESTONE_1.md) for
explicit flags, reproduction, incomplete-session inspection and storage limits.
Python integration tests also require Go on PATH because they inspect real Go output.

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

**Milestone 3 is complete:** the native dashboard, local Go-backed mock drives,
manual/automatic incidents, evidence navigation and finalized export passed native
validation and user acceptance. The minimal UI is accepted for initial testing;
Appetize file-retrieval limits are documented. **Milestone 4 not started.**

Authoritative plans (kept under their original names):

- [MVP implementation plan](planning/alfa_flight_recorder_mvp_plan.md)
- [Enterprise architecture](planning/vehicle_health_enterprise_architecture.md)
- [Milestone 0.5 assignment](planning/milestone_0.5.md)
- [Milestone 1 assignment](planning/milestone_1.md)

The active milestone assignment takes priority over broader implementation suggestions
in the plans. Enterprise services and expanded vehicle coverage remain future work.
