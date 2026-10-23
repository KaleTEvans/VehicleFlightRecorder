# Development

## Scope and prerequisites

Work at `~/dev/VehicleFlightRecorder`; this directory is the monorepo. Milestone 0
established the foundation; Milestone 0.5 prepares remote simulator builds and
browser previews. Milestone 1 adds the local Go recorder/healthy generator and
Python validation/summary commands. Follow [MILESTONE_1.md](MILESTONE_1.md) to
generate and inspect a real synthetic session. Read the plans in `planning/` and the active
milestone assignment before implementation. Explicit task instructions take
precedence, followed by the MVP plan and then the enterprise roadmap.

## Windows/WSL to GitHub to Appetize

No personal Mac is required for the remote workflow. Follow
[IOS_PREVIEW.md](IOS_PREVIEW.md) to push to GitHub, run **iOS Simulator Preview**
manually, download `VehicleFlightRecorder-ios-simulator`, configure optional
`APPETIZE_API_TOKEN` / `APPETIZE_PUBLIC_KEY` repository secrets, and observe the
mock-only app in Appetize. The unsigned simulator workflow needs no signing secrets.
It uses the existing XcodeGen definition, iOS 17 floor, and fixed approved identity.
There is no Go XCFramework reference in this app yet; this validates the SwiftUI
shell, not Go recorder integration. The workflow's ZIP is retained for 14 days
even if a subsequent Appetize upload fails.

Linux/WSL needs Bash, Git, Go 1.24+, Python 3.11+ with `venv` and pip, and network
access for initial Python dependency installation. Use a maintained toolchain;
the minimum Go directive is the language compatibility floor. Bootstrap never
uses sudo or installs system packages. Missing tools produce actionable errors.
Install Go following <https://go.dev/doc/install>; install Python and its venv
support through your preferred system package manager or existing version manager.

On the initial WSL machine, Go is at `/usr/local/go/bin/go` and Python 3.11.9 is
at `$HOME/.pyenv/versions/3.11.9/bin/python3`; default Python is too old. Use:

```bash
cd ~/dev/VehicleFlightRecorder
export PATH="/usr/local/go/bin:$PATH"
PYTHON="$HOME/.pyenv/versions/3.11.9/bin/python3" ./scripts/bootstrap.sh
./scripts/test_all.sh
```

On machines where Python 3.11+ and Go are already on PATH, simply run
`./scripts/bootstrap.sh`. Repeated bootstrap reuses `analytics/.venv` and ensures
the editable package and development dependencies are installed. To change Python
versions, recreate that virtual environment deliberately. The scripts keep default
Go/pip caches and temporary build files under ignored `.cache/` in the repository.
Go requires no third-party modules at this milestone; Python has no runtime
dependencies. Setuptools and pytest are bounded development dependencies, not an
exact lockfile. Fully pinned toolchains can follow when real dependencies arrive.

Equivalent manual Python setup (select Python 3.11+ first):

```bash
cd ~/dev/VehicleFlightRecorder/analytics
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -e ".[dev]"
pytest
```

No `PYTHONPATH` override is needed: the smoke test checks installed package metadata
as well as importing `vehicle_flight_recorder`.

## Checks

```bash
cd ~/dev/VehicleFlightRecorder
export GOCACHE="$PWD/.cache/go-build"
export GOMODCACHE="$PWD/.cache/go-mod"
export TMPDIR="$PWD/.cache/tmp"
mkdir -p "$TMPDIR"
gofmt -w core
(cd core && go test ./... && go vet ./...)
(cd analytics && source .venv/bin/activate && pytest)
./scripts/test_all.sh
git status
```

The root script runs Go tests/vet, Python tests, and simulator packaging validation
tests and returns nonzero if any fails. Packaging tests use synthetic metadata and
ZIP layouts; they do not compile or launch iOS. Run them individually with
`analytics/.venv/bin/python -m unittest discover -s scripts/tests -v`.
Python tests additionally build/run the Go mock-session command and consume its
actual files. Therefore standalone `pytest` requires Go on PATH as well as the
installed editable Python package. The integration check uses no heavy analytics
libraries or runtime dependencies. It compares two generated sessions byte for byte.
The Linux GitHub Actions workflow bootstraps and runs the same checks, plus Go
formatting validation. CI execution itself requires pushing to a GitHub repository.

## Boundaries and layout

- `ios/VehicleFlightRecorder`: App and Views contain a static SwiftUI shell.
  ViewModels, Bridge, and Export reserve native presentation/integration seams.
  Transport contains a generic raw-byte contract and two inert implementations.
- `core/`: local module `vehicleflightrecorder.local/core`; no hosting ownership
  implied. `mobile.Version()` is the only initial exported facade function.
  Telemetry/session models, recorder/buffer, healthy mock, profile loader and JSONL
  storage now run through `cmd/mock-session`. Incidents remain reserved for Milestone 2.
- `analytics/`: src-layout package with strict streaming validation and JSON summary CLI.
- `vehicle-packs/`: manufacturer metadata and six mock-only synthetic signal definitions.
- `testdata/`: reserved scenario directories; integration fixtures are generated
  at test time and bulk sessions stay in ignored `build/` output.
- `docs/`, `scripts/`, `.github/workflows/`: workflow and verification foundation.

Swift owns native lifecycle/UI/transport, Go owns future domain behavior and protocol
parsing, Python owns offline analysis. No vehicle-specific meaning belongs in
transport or generic Go code. Future boundaries favor scalar values, bytes, and JSON.
The local recorder and summary CLI exist; there are no incident detectors, native
recorder bridge, cloud, AI, enterprise services, or diagnostic commands.
Both transports remain disconnected; connect throws a bootstrap-unavailable error,
write throws not-connected, and inbound streams finish without yielding data.
The dashboard defaults to MockVehicleTransport and never initiates a connection.

Safety invariants: read-only vehicle access; no automatic DTC clearing, coding,
configuration, calibration, actuation, undocumented commands, or arbitrary CAN
writes. Future active requests must be isolated from passive capture, logged, and
rate-limited. Uncertain transport state must fail closed. Future driving UI must
minimize driver interaction. Preserve raw evidence and distinguish inference.

## iOS project on macOS

The deployment floor is **iOS 17.0**, a deliberate compatibility choice sufficient
for this simple SwiftUI shell without requiring the newest OS. Use a current full
Xcode installation with iOS SDKs and XcodeGen 2.42+. Swift language mode is 5;
there are no third-party Swift libraries. XcodeGen is a build tool only.

1. Check out this repository on a Mac. Install Xcode, launch it once, accept its
   license and install the iOS platform/simulator components. Ensure
   `xcode-select -p` points inside the full Xcode app. If necessary, use
   `sudo xcode-select --switch /Applications/Xcode.app/Contents/Developer`.
2. Install XcodeGen (for Homebrew users: `brew install xcodegen`).
3. From the repository root:

   ```bash
   ./scripts/generate_xcode_project.sh
   open ios/VehicleFlightRecorder.xcodeproj
   ```

4. Select the VehicleFlightRecorder scheme and an installed iOS simulator. Build
   and Run with Xcode. A simulator does not need a development team. An unsigned
   compile check can also be run from the repository root:

   ```bash
   xcodebuild -project ios/VehicleFlightRecorder.xcodeproj \
     -scheme VehicleFlightRecorder -configuration Debug \
     -sdk iphonesimulator -destination 'generic/platform=iOS Simulator' \
     -derivedDataPath build/DerivedData CODE_SIGNING_ALLOWED=NO build
   ```

5. For a physical iPhone, sign into Xcode with an Apple ID, select the development
   team under Signing & Capabilities, and enable Developer Mode on the device.
   Validate provisioning with your Apple Developer account; TestFlight/App Store
   distribution requires the appropriate Apple Developer Program membership.
   Do not change the approved bundle identifier. Team selection in the generated
   project is local; for repeat generation, pass `DEVELOPMENT_TEAM=YOUR_TEAM_ID`
   to xcodebuild or deliberately add your team setting to `project.yml`.

`ios/project.yml` and the explicit Resources/Info.plist are source-controlled;
the generated `.xcodeproj` is ignored and reproducible. No hand-authored Xcode
metadata or fabricated entitlement is needed. Project structure follows the
[XcodeGen specification](https://github.com/yonaskolb/XcodeGen/blob/master/Docs/ProjectSpec.md).
The static app does not depend on a Go framework to build. Icons, distribution
assets, and App Store packaging are outside this bootstrap.

## Approved accessory identity

Apple/OBD Solutions application whitelisting is already approved for:

```text
Application:       Vehicle Flight Recorder
Version:           0.1.0
Bundle identifier: com.kaleevans.vehicleflightrecorder
iAP protocol:      com.obdlink
Accessory model:   MX201
Transport:         Bluetooth Classic / Apple MFi
MFi PPID:          221699-0001
```

The PPID is App Store submission metadata; include it in the submission Release
Notes per the supplied MVP plan. It is not a runtime command or domain constant.

The project links ExternalAccessory.framework, declares
`UISupportedExternalAccessoryProtocols = [com.obdlink]`, declares
`UIBackgroundModes = [external-accessory]`, and records Background Modes capability
intent in XcodeGen. Verify those settings in the generated target. Later MX+ work
uses EAAccessoryManager and EASession, not CoreBluetooth. Capability, signing,
provisioning/entitlement requirements and actual background behavior require Mac
and physical-device validation. Configuration alone does not prove live access.

## Future Go XCFramework workflow

This workflow is prepared but not validated from WSL. On a Mac with Go and full
Xcode, install matching gomobile/gobind tools using the official
[gomobile documentation](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile).
For an initial tooling evaluation (then record/pin the resolved x/mobile revision):

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
go install golang.org/x/mobile/cmd/gobind@latest
export PATH="$(go env GOPATH)/bin:$PATH"
gomobile init
./scripts/build_go_ios.sh
```

Mobile tooling can require a newer Go release than the core's minimum. If binding
reports missing `golang.org/x/mobile/bind` packages, add the same evaluated x/mobile
revision as a module dependency during the Mac integration milestone. It is not
introduced as an untested dependency of the current Linux core.

The script checks platform, tools, and both Apple SDKs, then invokes
`gomobile bind -target=ios,iossimulator -iosversion=17.0` on `./mobile`, targeting
`build/ios/VehicleFlightRecorderCore.xcframework`. Only version metadata is exposed
now. Framework linking, real recorder API design, compatible toolchain pinning,
and bridge/lifecycle validation remain future work, with integration hardening
in Milestone 4. Building an XCFramework does not build or launch an iOS app.

## WSL limitations and pending validation

WSL can run Go/Python and packaging checks and workflow linting. Mac scripts exit
nonzero with clear platform messages on Linux. GitHub's macOS workflow performs
project generation, unsigned simulator compilation, and packaging remotely. The
first corrected remote build passed; the actual ZIP was downloaded and verified.
Automated Appetize upload passed, and the user confirmed the browser launch and
all expected mock/disconnected status labels on October 23, 2026. Milestone 0.5
is complete. See IOS_PREVIEW.md for run evidence and the real preview link.
Signing, device compilation/distribution, capability/entitlement validation, future
framework binding, and ExternalAccessory behavior require later work. A simulator
preview cannot prove background hardware behavior or live MX+ connectivity.

The initial Codex sandbox launcher could not locate bubblewrap even though an
outside-sandbox check returned version 0.4.0. Shell verification for this bootstrap
therefore requires approved outside-sandbox execution; this is a host tooling
limitation, not an application dependency. All project artifacts stay in this root.

## Scope decisions relative to the plans

The Milestone 1 implementation follows its dedicated handoff and preserves the
completed bootstrap/preview. Generic names use
VehicleFlightRecorder/vehicle_flight_recorder; original planning filenames stay.
XcodeGen replaces the planned checked-in Xcode project, and launch acceptance is
validated through the remote build and user-observed browser launch. The new pack
signals are explicitly mock-only; real polling/threshold arrays remain empty.
The JSONL finalization and schema choices are documented in DATA_SCHEMA.md. No
enterprise implementation or Milestone 2 work is included.
