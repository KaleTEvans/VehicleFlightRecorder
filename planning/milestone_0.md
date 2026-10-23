You are beginning implementation of the **Vehicle Flight Recorder** project.

Your immediate objective is to complete **Milestone 0 — repository/bootstrap only**, while understanding the full architectural direction of the project so that the foundation created now will support later milestones without unnecessary restructuring.

Do **not** implement Milestone 1 or later functionality yet.

## Working directory

The project root is:

```text
~/dev/VehicleFlightRecorder
```

Treat this directory as the repository root.

Before making any changes, inspect the contents of this directory and read the two architecture documents located there:

```text
~/dev/VehicleFlightRecorder/alfa_flight_recorder_mvp_plan.md
~/dev/VehicleFlightRecorder/vehicle_health_enterprise_architecture.md
```

These documents are the authoritative project plans.

Read both documents in full before designing or creating the repository structure.

The MVP document governs near-term implementation details.

The enterprise architecture document provides longer-term architectural constraints. Milestone 0 should not implement enterprise functionality, but decisions made now should avoid unnecessarily blocking that architecture later.

If this prompt and the documents differ, follow this priority:

1. Explicit instructions in this prompt.
2. `alfa_flight_recorder_mvp_plan.md`
3. `vehicle_health_enterprise_architecture.md`

Do not silently reinterpret major architectural decisions. If you believe a deviation is necessary, document it in your final report.

---

# Current project status

The application has already been approved/whitelisted by Apple/OBD Solutions for use with the **OBDLink MX+**.

Use these application identifiers as stable values:

```text
App name:              Vehicle Flight Recorder
Initial version:       0.1.0
CFBundleIdentifier:    com.kaleevans.vehicleflightrecorder

OBDLink MX+:
iAP protocol:          com.obdlink
MFi Product Plan ID:   221699-0001
Accessory model:       MX201
Bluetooth:             Bluetooth Classic / Apple MFi
```

The eventual real iOS transport will use:

```text
ExternalAccessory
EAAccessoryManager
EASession
UISupportedExternalAccessoryProtocols = com.obdlink
```

The application will eventually enable the iOS `external-accessory` background mode.

However:

**Do not implement live OBDLink communication in Milestone 0.**

Do not send AT, ST, OBD, CAN, or manufacturer-specific commands.

Do not implement real ECU polling.

Do not attempt Bluetooth communication through CoreBluetooth; the MX+ uses the External Accessory/MFi path.

The transport must remain abstracted so the real MX+ implementation can be added in a later milestone.

---

# Product context

The initial test vehicle is:

```text
2017 Alfa Romeo Giulia Quadrifoglio
2.9L twin-turbo V6
ZF 8-speed transmission
```

The first practical objective of the application is to act as a vehicle **flight recorder**.

Eventually it will continuously capture enough drivetrain, electrical, diagnostic, and network information to reconstruct intermittent vehicle faults and determine the initiating event rather than merely observing the resulting warning cascade.

The long-term product may support many consumer vehicle makes and models.

Therefore:

- generic recorder/domain logic must not be Alfa-specific;
- Alfa-specific definitions belong in a vehicle pack;
- transport logic must not contain vehicle-specific diagnostic meaning;
- Swift should contain as little business logic as practical;
- Go should own the recorder/domain layer;
- Python should own offline analytics and research.

The intended long-term flow is approximately:

```text
Vehicle
   ↓
Adapter / transport
   ↓
Protocol layer
   ↓
Vehicle pack / signal definitions
   ↓
Canonical telemetry
   ↓
Recorder
   ↓
Incident engine
   ↓
Session storage
   ↓
Python analytics
   ↓
Structured findings
   ↓
Later AI reporting
```

Do not implement these later layers beyond what Milestone 0 requires.

---

# Language responsibilities

Preserve the architecture defined in the project plans.

## Swift / SwiftUI

Swift should eventually own only functionality that needs to be native to iOS:

- app lifecycle;
- SwiftUI;
- ExternalAccessory integration;
- background execution integration;
- native permissions/capabilities;
- file sharing/export UI;
- thin Go bridge.

Avoid putting telemetry processing, incident logic, diagnostic rules, vehicle-specific logic, or analytics into Swift.

## Go

Go is intended to become the primary shared runtime/domain layer for:

- canonical telemetry;
- recorder;
- sessions;
- incident detection;
- rolling buffers;
- mock vehicle generation;
- vehicle profiles;
- transport-independent scheduling;
- OBD/STN protocol handling later.

Milestone 0 only needs to establish the Go module/package structure and testing foundation.

## Python

Python will eventually own:

- exported-session analysis;
- dataframe/time-series analysis;
- anomaly research;
- baselines;
- trends;
- incident reconstruction;
- report generation;
- later AI-report preparation.

Milestone 0 only needs to establish a clean Python package and testing foundation.

---

# Environment constraint: WSL

This repository is currently being developed from WSL.

Assume:

```text
Linux / WSL development environment
No local Xcode installation
No ability to compile or launch an iOS application from WSL
```

Do not fake or claim successful iOS compilation.

Milestone 0's original acceptance criterion includes:

> iOS app launches.

Because Xcode requires macOS, split this into:

### What must be completed now in WSL

- Create the intended iOS source/project scaffold.
- Make the Swift source structurally valid and intentionally minimal.
- Create all configuration/source files that can reasonably be prepared without Xcode.
- Document the exact steps needed to open/generate/build the project later on a Mac.
- Do not introduce dependencies that unnecessarily complicate Mac setup.

### What remains pending

```text
Xcode project generation/opening
code signing
capability validation
simulator/device compilation
actual iOS launch
ExternalAccessory entitlement/device validation
```

Clearly identify those as **pending macOS validation**, not failures in the Linux implementation.

If generating a normal `.xcodeproj` reliably from Linux is impractical, prefer a reproducible project definition such as an XcodeGen `project.yml`, plus the Swift source tree.

Do not manually fabricate a complex `.xcodeproj` if it creates fragile generated metadata.

The goal is that later on macOS we can install Xcode/XcodeGen, generate/open the project, set the development team if necessary, and build it with minimal manual reconstruction.

---

# Milestone 0 objective

Complete:

> **Milestone 0 — repository/bootstrap**

According to the MVP plan, the intended deliverables are:

- monorepo;
- Xcode/SwiftUI application scaffold;
- Go module;
- Python package;
- build/development scripts;
- basic CI if reasonable;
- README containing setup instructions.

The acceptance goals are:

```text
Go:
go test ./... passes

Python:
tests pass

iOS:
source/project scaffold completed
actual launch pending macOS/Xcode validation because development currently occurs in WSL
```

Do not continue into Milestone 1 after satisfying these goals.

---

# Repository structure

Use the current directory as the root:

```text
~/dev/VehicleFlightRecorder/
```

Do not create another nested repository such as:

```text
~/dev/VehicleFlightRecorder/alfa-flight-recorder/
```

The current folder itself is the monorepo.

Create a clean structure based on the MVP plan, approximately:

```text
VehicleFlightRecorder/
├── README.md
├── .gitignore
├── .editorconfig
│
├── alfa_flight_recorder_mvp_plan.md
├── vehicle_health_enterprise_architecture.md
│
├── docs/
│   ├── DATA_SCHEMA.md
│   ├── VEHICLE_PROFILE.md
│   ├── INCIDENT_MODEL.md
│   └── DEVELOPMENT.md
│
├── ios/
│   ├── project.yml
│   └── VehicleFlightRecorder/
│       ├── App/
│       ├── Views/
│       ├── ViewModels/
│       ├── Transport/
│       ├── Bridge/
│       ├── Export/
│       └── Resources/
│
├── core/
│   ├── go.mod
│   ├── mobile/
│   ├── telemetry/
│   ├── recorder/
│   ├── incidents/
│   ├── vehicle/
│   ├── mock/
│   └── storage/
│
├── vehicle-packs/
│   └── alfa-giulia-qv-2017/
│       ├── manifest.json
│       ├── signals.json
│       ├── polling.json
│       ├── thresholds.json
│       └── scenarios/
│
├── analytics/
│   ├── pyproject.toml
│   ├── src/
│   │   └── vehicle_flight_recorder/
│   └── tests/
│
├── scripts/
│   ├── bootstrap.sh
│   ├── test_all.sh
│   ├── build_go_ios.sh
│   └── generate_xcode_project.sh
│
├── testdata/
│   ├── normal_drive/
│   ├── voltage_tcm_dropout/
│   ├── mechanical_fault/
│   └── adapter_disconnect/
│
└── .github/
    └── workflows/
        └── ci.yml
```

You may make small improvements to this structure if justified.

Use **VehicleFlightRecorder** rather than legacy `AlfaFlightRecorder` naming for new generic application/module names because the approved application and future platform are brand-neutral.

Alfa-specific naming should live under:

```text
vehicle-packs/alfa-giulia-qv-2017/
```

Do not rename the two supplied planning documents.

---

# Git initialization

If this directory is not already a Git repository:

```bash
git init
```

Create a sensible `.gitignore` covering at minimum:

- Go build artifacts;
- Python virtual environments and caches;
- pytest caches;
- coverage artifacts;
- macOS files;
- Xcode DerivedData;
- Xcode user-specific state;
- generated XCFrameworks;
- generated build directories;
- IDE temporary files;
- local environment/secrets files.

Do not ignore source-controlled project configuration.

Do not commit or create secrets.

---

# Go bootstrap

Initialize the Go module under:

```text
core/
```

Choose a sensible module name for local development that does not depend on owning an external GitHub domain.

Keep packages intentionally small.

At Milestone 0, packages may contain documentation and minimal placeholder types/functions only where needed to ensure compilation.

Expected package directories:

```text
core/mobile
core/telemetry
core/recorder
core/incidents
core/vehicle
core/mock
core/storage
```

Do not implement the Milestone 1 recorder yet.

Do not implement mock drive generation yet.

Do not implement telemetry persistence yet.

Add minimal unit tests so:

```bash
cd ~/dev/VehicleFlightRecorder/core
go test ./...
```

passes.

Use idiomatic Go.

Run:

```bash
gofmt
go vet ./...
go test ./...
```

before finishing.

---

# Python bootstrap

Create a modern Python package under:

```text
analytics/
```

Use:

```text
pyproject.toml
src layout
pytest
```

Package name:

```text
vehicle_flight_recorder
```

Do not add heavy analytics dependencies yet unless they are necessary for Milestone 0.

Prefer a minimal dependency footprint.

Establish:

```text
analytics/src/vehicle_flight_recorder/
analytics/tests/
```

Provide at least one lightweight test validating that the package installs/imports correctly.

The documented development flow should work with a normal virtual environment, for example:

```bash
cd ~/dev/VehicleFlightRecorder/analytics
python3 -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"
pytest
```

If you choose different commands, document them clearly.

Do not implement the Milestone 1 session analyzer yet.

---

# iOS bootstrap

Create the smallest sensible SwiftUI shell for:

```text
Vehicle Flight Recorder
Bundle identifier:
com.kaleevans.vehicleflightrecorder

Version:
0.1.0
```

Use a minimum deployment target that is reasonable for a new application in 2026; document the choice.

Do not add third-party Swift libraries unless strongly justified.

The initial application should conceptually contain:

```text
VehicleFlightRecorderApp
ContentView / DashboardView
```

The first UI can simply display:

```text
Vehicle Flight Recorder

Development Mode

Transport: Mock / Not Connected
Recorder: Not Started
```

No real telemetry UI is necessary in Milestone 0.

Establish directories/interfaces for:

```text
App
Views
ViewModels
Transport
Bridge
Export
Resources
```

---

# Transport abstraction

Create the Swift-side transport abstraction now, but keep implementations minimal.

Define a protocol conceptually equivalent to:

```text
VehicleTransport
```

It should represent lifecycle and raw byte transport, not OBD semantics.

It should eventually support:

- connect;
- disconnect;
- connection state;
- outbound bytes;
- inbound bytes;
- transport errors.

Do not put:

- PIDs;
- CAN IDs;
- Alfa definitions;
- ECU addresses;
- DTC logic;
- ST commands;
- AT commands;

inside the transport abstraction.

Create:

```text
MockVehicleTransport
```

as the active/default implementation.

For Milestone 0 it may simply expose a disconnected/mock state.

Also create the structural shell for:

```text
MXPlusExternalAccessoryTransport
```

but **do not implement real EASession behavior yet unless the MVP document explicitly places compile-only scaffolding in Milestone 0**.

It is acceptable for this type to contain documentation/TODOs describing the later integration.

No real MX+ commands may be transmitted in this milestone.

---

# External Accessory project configuration

Prepare the iOS project configuration to support the approved MX+ application identity.

The generated Xcode project should eventually contain:

```xml
<key>UISupportedExternalAccessoryProtocols</key>
<array>
    <string>com.obdlink</string>
</array>
```

Prepare the project for the `external-accessory` background mode as described in the MVP plan.

Do not use CoreBluetooth for MX+.

Document in `docs/DEVELOPMENT.md` that the OBDLink application whitelist is already approved for:

```text
Vehicle Flight Recorder
com.kaleevans.vehicleflightrecorder
```

Do not change this bundle identifier.

Also document:

```text
MFi PPID: 221699-0001
```

as App Store submission metadata, not runtime protocol logic.

Do not expose these constants across unrelated domain packages.

---

# Go ↔ Swift boundary

Do not build a large bridge in Milestone 0.

Establish only the intended architectural boundary.

The project plans expect Go eventually to be built for iOS through:

```text
gomobile bind
```

and exposed as an XCFramework.

Create:

```text
scripts/build_go_ios.sh
```

with:

- clear dependency checks;
- useful errors if `gomobile` is unavailable;
- comments explaining expected macOS/iOS tooling;
- no false claim that WSL can produce a usable iOS application.

If `gomobile bind` cannot reasonably target iOS from the current Linux environment, detect/report that condition gracefully.

Do not add elaborate Swift-Go APIs before Milestone 1 establishes the actual core types.

---

# Vehicle pack bootstrap

Create:

```text
vehicle-packs/alfa-giulia-qv-2017/
```

This establishes vehicle pack #1.

At Milestone 0, configuration files should contain valid schemas/placeholders and explanatory documentation rather than guessed diagnostic definitions.

Do not invent:

- proprietary Alfa DIDs;
- ECU request addresses;
- CAN decoding formulas;
- manufacturer-specific polling commands.

It is acceptable for files such as:

```text
signals.json
polling.json
thresholds.json
```

to contain empty arrays/objects plus version/vehicle metadata.

`manifest.json` should identify the pack and target vehicle without pretending that unsupported signal definitions already exist.

For example, model metadata may include:

```text
make: Alfa Romeo
model: Giulia Quadrifoglio
model_year: 2017
engine: 2.9L twin-turbo V6
```

Keep the schema generic enough for future vehicle packs.

---

# Documentation bootstrap

Create concise initial versions of:

```text
docs/DATA_SCHEMA.md
docs/VEHICLE_PROFILE.md
docs/INCIDENT_MODEL.md
docs/DEVELOPMENT.md
```

These should summarize decisions already present in the supplied plans.

Do not invent large specifications that belong to later milestones.

`DEVELOPMENT.md` should explain:

- WSL/Linux prerequisites;
- Go setup;
- Python setup;
- running all currently available tests;
- repository structure;
- architecture boundaries;
- how iOS project generation will work on macOS;
- Xcode requirement;
- Apple Developer/code-signing requirement;
- approved bundle ID;
- External Accessory protocol;
- eventual Go XCFramework workflow;
- known WSL limitations.

---

# Root README

Create a useful but concise `README.md`.

It should include:

1. Project purpose.
2. Current status: Milestone 0.
3. Initial target vehicle.
4. Architecture summary.
5. Repository layout.
6. Prerequisites.
7. Bootstrap instructions.
8. Test commands.
9. iOS/macOS limitation.
10. Reference to both planning documents.
11. Safety/read-only principle.
12. Next planned milestone: Milestone 1, but do not implement it.

Avoid marketing language.

---

# Development scripts

Provide at least:

## `scripts/bootstrap.sh`

Prepare/check the Linux/WSL Go and Python development environment without making dangerous system-wide changes.

It should be idempotent where practical.

Do not automatically install system packages with `sudo`.

Instead detect missing dependencies and print instructions.

## `scripts/test_all.sh`

Run all tests currently available:

```text
Go
Python
```

It should exit non-zero if either fails.

Later iOS tests can be incorporated from macOS.

## `scripts/generate_xcode_project.sh`

If using XcodeGen:

- detect macOS;
- detect `xcodegen`;
- generate the project from `ios/project.yml`;
- fail with a useful explanation on WSL/Linux.

## `scripts/build_go_ios.sh`

Prepare the future Go → XCFramework workflow.

Detect unsupported environment/tooling and explain what must be run on macOS rather than failing mysteriously.

---

# CI

If practical without scope creep, add a basic GitHub Actions workflow that runs:

```text
go test ./...
Python tests
```

on Linux.

Do not add an iOS/macOS CI job unless it is simple and does not require signing/secrets.

CI is secondary to a clean local bootstrap. Do not spend excessive time on it.

---

# Safety requirements

These are architectural invariants even at bootstrap stage.

The project is initially **read-only**.

Never introduce code that:

- clears DTCs automatically;
- performs ECU coding;
- performs configuration changes;
- performs calibration;
- actuates vehicle systems;
- sends undocumented manufacturer-specific commands;
- writes arbitrary CAN frames to the vehicle.

Future active diagnostic requests must be isolated from passive monitoring and logged.

Transport uncertainty must eventually fail closed.

The UI must eventually minimize driver interaction while moving.

Do not implement anything contrary to these rules during bootstrap.

---

# Engineering quality

Use:

- clear package/module boundaries;
- idiomatic Go;
- conventional Python packaging;
- simple Swift;
- deterministic scripts;
- comments for architectural intent rather than obvious syntax;
- no unnecessary abstractions;
- no premature frameworks;
- no speculative dependencies.

Do not over-engineer Milestone 0.

At the same time, avoid shortcuts that would obviously need to be discarded in Milestone 1.

Prefer straightforward code that establishes stable seams.

---

# Required verification

Before declaring Milestone 0 complete, run and report the actual result of every command available in the current WSL environment.

At minimum:

```bash
cd ~/dev/VehicleFlightRecorder/core
go test ./...
go vet ./...
```

And:

```bash
cd ~/dev/VehicleFlightRecorder/analytics
# activate/create environment as documented
pytest
```

Then:

```bash
cd ~/dev/VehicleFlightRecorder
./scripts/test_all.sh
```

If Git is initialized:

```bash
git status
```

Inspect the final repository tree.

Do not state that iOS builds or launches unless you actually ran it through Xcode on macOS.

Instead explicitly report:

```text
iOS scaffold: complete
iOS compilation: pending macOS/Xcode validation
iOS launch: pending macOS/Xcode validation
MX+ live connection: intentionally not implemented in Milestone 0
```

---

# Definition of done

Milestone 0 is complete when:

- both supplied architecture documents have been read;
- the monorepo structure exists;
- Git repository/bootstrap files exist;
- Go module compiles and all Go tests pass;
- Python package installs/imports and all Python tests pass;
- root test script passes;
- the minimal SwiftUI application source is scaffolded;
- iOS project generation/configuration is prepared for macOS;
- bundle identifier is `com.kaleevans.vehicleflightrecorder`;
- `com.obdlink` External Accessory support is represented in project configuration;
- the MX+ transport abstraction exists but no live diagnostic communication occurs;
- the Alfa Giulia QV vehicle-pack directory exists without invented proprietary diagnostic definitions;
- documentation explains the development workflow and WSL/macOS split;
- README accurately describes current project state;
- no Milestone 1 implementation has been started.

---

# Final response format

When finished, do not simply say "done."

Provide:

## 1. Milestone status

State whether Milestone 0 is:

```text
COMPLETE
PARTIALLY COMPLETE
BLOCKED
```

and explain any blocker.

## 2. Files created

Summarize the important files/directories created.

## 3. Architectural decisions

List any decisions you made that were not explicitly dictated by the supplied plans.

## 4. Verification

Show the commands actually executed and whether they passed.

## 5. macOS validation remaining

Explicitly identify everything that must later be validated using Xcode/macOS.

## 6. Deviations

Identify any difference between the implementation and either planning document.

## 7. Next milestone

Describe, but **do not implement**, the first tasks of Milestone 1.

Do not proceed into Milestone 1 without a new instruction.