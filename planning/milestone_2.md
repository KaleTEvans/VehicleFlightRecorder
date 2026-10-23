# Codex Handoff — Vehicle Flight Recorder, Milestone 2 Only

## Assignment

Implement **Milestone 2 — incident engine** in:

```text
~/dev/VehicleFlightRecorder
```

This is a future implementation handoff, not permission to begin code changes
merely because the file exists. **Prerequisite: Milestone 1 has been implemented
and accepted.** Inspect its actual recorder, buffer, writer, CLI, schemas and tests.
Do not assume the earlier handoff's example APIs were implemented verbatim. If the
prerequisite is missing or incomplete, report that instead of implementing two
milestones together. Stop before the Milestone 3 native dashboard.

## Authoritative context

Read both authoritative plans in full:

```text
planning/alfa_flight_recorder_mvp_plan.md
planning/vehicle_health_enterprise_architecture.md
```

Priority: the human's current instructions, this handoff, MVP plan, enterprise
roadmap. Pay particular attention to MVP sections 6–7, 11–13, 18–22, and enterprise
evidence/provenance and incident-correlation principles. Report deviations rather
than silently broadening the architecture.

Read `planning/milestone_1.md`, the current README and docs, `git status`, the
source tree and Milestone 1 acceptance evidence. Preserve existing changes and the
working GitHub/Appetize pipeline. Keep all work under the existing repository root.

The fixed application identity is Vehicle Flight Recorder, version 0.1.0,
`com.kaleevans.vehicleflightrecorder`. The approved protocol is `com.obdlink`;
MFi PPID `221699-0001` is submission metadata, not a domain constant. The Go module
is `vehicleflightrecorder.local/core`; Python is `vehicle_flight_recorder`.
The initial profile is `alfa.giulia.qv.2017.2_9t`.

Windows/WSL remains the local development environment. Native builds run on GitHub
macOS; Appetize previews are mock-only. Real iPhone/MX+ communication remains later
work. Preserve SwiftUI + Go + Python and a thin native boundary.

## Required work

### 1. Incident contracts and Go detector boundary

Implement Go incident/trigger models and a small detector interface in the existing
packages. Evaluate canonical observations/events; do not bake vehicle addresses,
adapter commands or SwiftUI state into detection.

Follow the MVP incident representation: incident/session identity, trigger time
and type, configurable pre/post window sizes, severity, capture status and manual
flag. Preserve trigger/evidence references and enough metadata to reconstruct the
ordered timeline. Document additions and supported schema versions.

Keep observed facts, correlation and diagnostic hypotheses distinct. A voltage
drop preceding a TCM timeout is temporal evidence, not proof of causation.

### 2. Manual and automatic triggers

Implement the initial detector set from MVP section 11 at the scope needed for
the defined scenarios:

- A programmatic manual marker, creating an incident immediately; the SwiftUI
  button is Milestone 3 work.
- A configurable low-voltage and/or rapid-drop trigger with synthetic test limits.
- Repeated expected-response timeouts for critical modules; configure consecutive
  failure count and restoration behavior instead of hard-coding Alfa knowledge.
- Correlation of multiple critical-module losses within a configurable interval.
- A newly observed synthetic DTC creating or attaching to an incident.
- Unexpected mock transport loss during an active session, represented separately
  from an ECU timeout and linked to an incident when configured.

Use only mock data/configuration and explicit canonical events. Detect response
failure from known expected-response context; absence of an unrelated signal is
not automatically an ECU failure. Define threshold crossing, reset/rearm, repeated
trigger and duplicate-DTC behavior so a sustained fault does not flood incidents.
Thresholds are mock-only, not validated Giulia diagnostic limits.

### 3. Incident capture windows

Use the Milestone 1 rolling buffer to retain the default **60 seconds before** and
**120 seconds after** a trigger; both are configurable. Logical timestamps, not
test sleeps, determine capture boundaries. Record actual available coverage if a
trigger occurs near the start/end of a session or while data is missing.

Persist incident metadata/indexes in `incidents.jsonl` and relevant canonical
events in the existing session event log. Full-session observations/events remain
authoritative; incidents reference their evidence window rather than creating
independent competing copies of the drive.

Choose and document a deterministic overlap policy consistent with the MVP:
overlapping triggers link or merge without duplicating large evidence windows.
Keep original trigger times/types and manual markers discoverable. Separate raw
trigger count from captured-incident count in summaries/tests. Document JSONL
update/finalization semantics so readers reconstruct one coherent incident state.

Stopping a session before post-capture completes must preserve partial evidence
and explicit incomplete coverage. Reuse existing storage/error handling. Do not
claim finalized 120-second coverage for an interrupted session.

### 4. Deterministic mock fault scenarios

Extend the existing Go generator/scenario runner and CLI; preserve Scenario A.
Implement the four primary fault scenarios from MVP section 13:

| Scenario | Required evidence and result |
| --- | --- |
| B: voltage disturbance then TCM loss | Voltage evidence precedes latency/timeouts, then restoration; one correlated captured incident under the documented overlap policy |
| C: TCM loss with stable voltage | Communication incident; no voltage trigger or invented electrical cause |
| D: mechanical/turbo anomaly | Requested/actual boost mismatch followed by a synthetic DTC; powertrain incident distinguishable from communication loss |
| E: adapter disconnect | Transport-loss evidence, explicit coverage gap/recovery, and recoverable session; do not report all ECUs as independently failed |

Also cover Scenario F by injecting a manual mark during a healthy run and verifying
its available pre/post evidence. These are synthetic generator events, never real
diagnostic transactions. Scenario D can use its synthetic DTC as the incident
trigger; this milestone does not require a new unvalidated mechanical diagnosis
or proprietary boost decoder.

Keep fixed seed/time/identity controls. Expose scenario selection through the
existing Go command and document names, expected incidents/trigger order, and
accelerated execution. Do not add a Swift scenario picker yet. Put vehicle-specific
mock thresholds/scenario data in the vehicle pack where appropriate, explicitly
labeling synthetic provenance.

### 5. Python incident reconstruction

Extend the existing Python loader/validator/summary to understand incident records,
capture status, trigger links and partial coverage. Add a minimal deterministic CLI:

```bash
python -m vehicle_flight_recorder incident <session-directory> <incident-id>
```

Return structured JSON with the trigger(s), actual capture coverage, and an ordered
timeline of relevant observations/events/evidence references. Make missing data,
unknown IDs, corrupt files and unsupported schemas explicit. Avoid claims of
electrical causation for Scenario C or a mechanical diagnosis beyond synthetic
Scenario D evidence. Preserve the existing `validate` and `summarize` interfaces.

No dataframe library, plots, statistical baselines, machine learning or LLM is
required for this deterministic reconstruction.

### 6. Documentation and integration

Update incident/schema/profile notes to match actual capture and overlap behavior.
Document CLI reproduction for all scenarios, expected counts and ordered evidence.
Extend the existing Linux test flow and keep root tests/CI working. Preserve the
Milestone 0.5 native preview; it remains a static shell until Milestone 3.

## Required verification

Run `gofmt`, Go tests/vet, Python `pytest`, and `./scripts/test_all.sh`. Use race
checking where concurrent recorder/scenario execution is introduced. Test:

- Manual mark, threshold crossing/non-crossing and reset/rearm behavior.
- Timeout counts/restoration, stable-voltage communication loss, new versus
  repeated DTCs, multi-module correlation and distinct transport loss.
- Exact pre/post boundaries, trigger near session start, early stop, missing-data
  coverage, overlapping triggers, and completion without duplicate incident IDs.
- Deterministic output for every mock scenario using fixed inputs.
- Scenario A stays healthy with zero incidents; each fault scenario has its
  documented captured-incident count and trigger ordering.
- Persisted incident reconstruction from actual Go output in Python; partial,
  overlapping and malformed incident records are handled as documented.

Generate and validate/summarize real synthetic sessions for B–E and a manual mark;
run the Python incident command on their actual incident IDs. Tests must assert
ordered evidence, not only that a JSON file or incident exists. Separate incident
count from trigger count when overlap combines triggers. Avoid wall-clock sleeps.

Report actual command results. Do not claim real-car diagnostic accuracy from
synthetic fixtures. Inspect final changes and lint CI if modified. If native/build
inputs change, validate through the existing macOS workflow and report evidence;
an unchanged placeholder UI is not proof of native incident-engine integration.

## Milestone boundaries and safety

Do not build the iOS dashboard, session/incident screens, native export or MARK
INCIDENT button. Do not integrate the Go XCFramework into Swift yet merely to
display synthetic incident counts. Milestone 3 introduces that minimal integration;
Milestone 4 hardens it.

No live accessory discovery/EASession, adapter initialization, AT/ST/OBD/CAN
commands, manufacturer definitions, DTC clearing, coding, configuration,
calibration or actuation. No signing/TestFlight pipeline, cloud, accounts, broad
multi-brand expansion, AI reporting, or production mechanical fault diagnosis.
Future active diagnostic reads remain isolated, logged and rate-limited, and
uncertain transport state must fail closed.

## Acceptance and reporting

Milestone 2 is complete when detectors and capture windows behave deterministically,
all four primary fault scenarios and a manual mark produce the expected preserved
evidence, and Python reconstructs those incidents from real Go-generated logs.
Milestone 1's healthy generation and storage/summary behavior must still pass.

Report:

1. COMPLETE, PARTIALLY COMPLETE, or BLOCKED with evidence and remaining work.
2. Files changed, detector/configuration decisions and overlap policy.
3. Actual commands/results and a scenario table of captured incidents, raw
   triggers, capture coverage and expected versus observed ordering.
4. At least one Python reconstruction result and the stable-voltage negative case.
5. Schema compatibility, recovery/concurrency limitations and architectural deviations.
6. Any native workflow evidence if relevant, clearly separating synthetic from hardware tests.
7. Explicitly: **Milestone 3 not started.**
