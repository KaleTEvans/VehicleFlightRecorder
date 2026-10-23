# Codex Handoff — Vehicle Flight Recorder, Milestone 1 Only

## Assignment

Implement **Milestone 1 — Go recorder + mock data** in the existing repository:

```text
~/dev/VehicleFlightRecorder
```

This is an implementation handoff for a future task. Creating this document does
not start implementation. Milestones 0 and 0.5 are complete, including a successful
GitHub macOS build, Appetize upload, and user-confirmed launch of the SwiftUI shell.
Preserve that foundation. Stop after Milestone 1; do not implement incident
detection or the interactive iOS dashboard.

## Authoritative context

Before editing, read both plans in full at their current locations:

```text
planning/alfa_flight_recorder_mvp_plan.md
planning/vehicle_health_enterprise_architecture.md
```

Use this order: the human's active task instructions, this milestone handoff, the
MVP plan, then the enterprise roadmap. Report any necessary architectural
deviation. The enterprise document constrains boundaries, not this milestone's
feature count. Pay particular attention to MVP sections 2, 5–8, 10, 12–13,
18–20, and 22. Scope the broader MVP examples to the current milestone.

Read `git status`, the current tree, `README.md`, `docs/DEVELOPMENT.md`,
`docs/IOS_PREVIEW.md`, and existing schema/profile/incident notes. Inspect the Go
packages, Python package, vehicle-pack placeholders, scripts, and Linux CI.
Preserve legitimate changes. Do not recreate the monorepo or move the plans.

Current names and identity:

```text
Go module:           vehicleflightrecorder.local/core
Python package:      vehicle_flight_recorder
App:                 Vehicle Flight Recorder, version 0.1.0
Bundle identifier:   com.kaleevans.vehicleflightrecorder
Vehicle profile:     alfa.giulia.qv.2017.2_9t
iAP protocol:        com.obdlink
MFi PPID:            221699-0001 (submission metadata only)
```

The developer uses Windows/WSL with no personal Mac. Xcode builds run on GitHub;
Appetize runs simulator bundles. Do not introduce a parallel web UI. There is no
Go XCFramework linked to the current static iOS shell; this milestone does not
need to integrate one or change that shell.

## Required work

### 1. Canonical telemetry and session contracts

Implement small, idiomatic Go models for observations and events in the existing
packages. Follow the MVP's versioned JSON representation: millisecond Unix
timestamps, session identity, canonical signal IDs, source, value, unit, and
quality; preserve optional provenance where applicable. Reject unsupported schema
versions and invalid required fields with useful errors. Do not silently rename
the MVP signal catalog to different enterprise ontology examples.

Define the minimal session metadata necessary to identify and finalize a drive:
schema version, session ID, start/end and lifecycle state, profile/pack version,
and reproducibility information for mock runs. Document fields that the plans
leave unspecified. Keep identifiers/timestamps injectable for deterministic tests.
Do not design an extensive enterprise schema or expose Go structs to Swift now.

### 2. Session lifecycle and append-only storage

Implement explicit start, append, and stop/finalize operations in Go. Use a
caller-supplied output location and prevent accidental overwrite of existing
sessions. Report invalid lifecycle transitions and propagate storage errors.
Do not label partially written sessions as successfully finalized.

Produce the planned structure:

```text
<session-directory>/
├── session.json
├── observations.jsonl
├── events.jsonl
├── incidents.jsonl
└── raw/
    └── transport.jsonl
```

`incidents.jsonl` exists but is empty until Milestone 2. Create the raw log from
the beginning; record only honest synthetic source evidence, clearly identified
as mock. A mock generator is not an OBD transcript: do not fabricate real adapter
requests, ECU responses, addresses, or manufacturer commands to populate the file.
Document an empty raw-byte stream if the current generator has no raw transport.

Write newline-delimited valid JSON, flush/close deliberately, and use safe session
metadata finalization. Define how readers recognize an incomplete session and
surface an interrupted/truncated write. Preserve evidence; never silently discard
corrupt lines or claim power-loss durability that has not been tested. Keep storage
append-only and inspectable; no SQLite/database dependency is needed.

### 3. Rolling buffer foundation

Retain at least the last configurable 60 seconds of observations/events in memory
while preserving the full session on disk. Define exact cutoff behavior and an
explicit policy for out-of-order timestamps. Test snapshot ordering and ownership
so callers cannot mutate retained evidence inadvertently. Expire old data by
logical event time, not wall-clock sleeps in tests. Incident triggers and post-window
capture remain Milestone 2 work.

### 4. Deterministic healthy mock drive

Implement only MVP Scenario A: a healthy drive with engine start, idle,
acceleration, cruise, deceleration, stop, and shutdown/restart as appropriate.
Provide sufficient logical duration to exercise the rolling buffer. Use the
MVP's canonical signal names and label samples `source: mock`.

Keep generation, recorder lifecycle, and scheduling separable. Fixed seed, start
time, configuration and session identity must reproduce the same ordered telemetry
and events. Simulated time must advance without requiring a real 10–20 minute wait.
Document any difference between logical drive duration and execution time.

Alfa-specific metadata/configuration stays in the vehicle pack. Any signal
definitions added now must be explicitly `mock_only`, with synthetic ranges and
no claimed real-vehicle availability. Load/validate the profile only as needed;
do not invent proprietary IDs, diagnostic polling plans, or validated thresholds.

Add a small Go command under `core/cmd/` to generate a healthy session. Document
the exact command, flags, seed/time controls, output path, and overwrite behavior.
An example command shape is `go run ./cmd/mock-session`; choose and document its
actual interface rather than pretending that this command already exists.

### 5. Minimal Python validation and summary CLI

Extend the existing src-layout package, without renaming it. Provide:

```bash
python -m vehicle_flight_recorder validate <session-directory>
python -m vehicle_flight_recorder summarize <session-directory>
```

Read Go-produced metadata, observations and events, tolerating the intentionally
empty incident/raw files as documented. Validate supported schema versions,
required files/fields, timestamps/order, session identity and counts. Summarize
deterministically into structured JSON: session identity/state, logical duration,
observation/event counts, signal coverage, and incident count zero.

Exit nonzero with actionable diagnostics for invalid/corrupt input. Distinguish
an incomplete recording from a complete valid recording; document that contract.
No incident reconstruction, statistical analytics, plotting, dataframe framework,
AI, or research notebook is needed. Prefer the Python standard library.

### 6. Documentation and developer workflow

Document generation and analysis from a clean WSL checkout. Update relevant schema,
vehicle-profile and development notes with the actual implemented contracts.
Keep fixtures small, deterministic and synthetic. Generated bulk sessions belong
under ignored `build/` output, not committed personal-drive captures.

Keep `scripts/test_all.sh` and Linux CI passing. Preserve the existing manual
simulator workflow and approved app identity. Do not claim a mock session can be
started inside the iOS shell yet; the native runtime integration comes later.

## Required verification

Run and report actual results, not planned commands:

- `gofmt`, `go test ./...`, and `go vet ./...` under `core/`.
- Python package installation/import and `pytest` in `analytics/.venv`.
- `./scripts/test_all.sh` from the root.
- An actual Go-generated session, then Python `validate` and `summarize` on it.
- Repeat generation with fixed inputs and compare deterministic content.
- Tests for lifecycle errors, JSONL round trips, write-error propagation,
  incomplete/truncated sessions, rolling-buffer expiry/order, profile validation,
  mock determinism, empty incident files, unsupported schemas, and corrupt input.
- Inspect final `git status` and the changed tree; lint workflows if modified.

Use an injected clock instead of sleeping. Keep cross-language fixtures aligned:
the integration check must consume real Go output rather than only independently
handwritten Python fixtures. An unchanged static iOS preview is not evidence that
the recorder runs on iOS. Remote iOS validation is required if native/build inputs
change; otherwise preserve the established pipeline without an unnecessary upload.

## Milestone boundaries and safety

Do not implement incident detectors, manual marking, post-trigger capture, fault
scenarios, additional iOS screens, native export UI, Go XCFramework integration,
live EASession behavior, hardware polling, AT/ST/OBD/CAN commands, signing,
TestFlight, cloud services, or AI reporting. Milestone 2 owns incident behavior;
Milestone 3 owns the interactive native UI and initial runtime bridge.

The project remains read-only. No DTC clearing, coding, configuration, calibration,
actuation, undocumented manufacturer requests, or arbitrary CAN writes. Future
active diagnostic reads must be isolated, logged and rate-limited; uncertain
transport state must fail closed. Transport never interprets diagnostic meaning.

## Acceptance and reporting

Milestone 1 is complete when the Go command writes a valid complete healthy-drive
session, deterministic tests pass, the rolling buffer works, and Python validates
and summarizes the actual output. The existing iOS shell/preview remains available.

Report:

1. COMPLETE, PARTIALLY COMPLETE, or BLOCKED, with concrete remaining work.
2. Files changed and any decisions not fixed by the plans.
3. Exact generation/analysis commands and the resulting session summary.
4. Test commands/results, deterministic comparison, and storage/recovery limits.
5. Schema/interface changes and any architectural deviations.
6. Remote workflow evidence if native/build files changed; otherwise say unchanged.
7. Explicitly: **Milestone 2 not started.**
