# Milestone 2 — incident engine acceptance

Status: **COMPLETE** for synthetic Go/Python acceptance. Milestone 3 not started.
The SwiftUI app and Go mobile facade are unchanged; there is no native incident
integration or new Appetize build in this milestone.

The Go incident package adds rule evaluation and configurable capture windows;
the recorder persists revision indexes and canonical trigger/finalization events.
Python validates and reconstructs exact source evidence. See
[incident contracts](INCIDENT_MODEL.md), [schema](DATA_SCHEMA.md), and
[mock pack](VEHICLE_PROFILE.md). Observed facts and temporal correlation do not
establish electrical or mechanical causation.

## Reproduce

From the repository root with Go on PATH and bootstrap complete:

```bash
export PATH="/usr/local/go/bin:$PATH"
(cd core && go build -o ../build/mock-session ./cmd/mock-session)
build/mock-session --pack vehicle-packs/alfa-giulia-qv-2017 \
  --output build/voltage-demo --scenario voltage_tcm_dropout \
  --session-id voltage-demo --seed 42 --start 2026-10-23T12:00:00Z --duration 12m
analytics/.venv/bin/python -m vehicle_flight_recorder validate build/voltage-demo
analytics/.venv/bin/python -m vehicle_flight_recorder summarize build/voltage-demo
analytics/.venv/bin/python -m vehicle_flight_recorder incident \
  build/voltage-demo voltage-demo.incident-000001
```

Every output directory must be new. Scenario names: `healthy_drive`,
`voltage_tcm_dropout`, `tcm_dropout`, `mechanical_fault`, `adapter_disconnect`,
`manual_mark`. All execute accelerated logical time without sleeps. Flags
`--pre-window`, `--post-window` override pack settings; `--buffer-window` must be
at least the pre-window. A two-minute drive with a midpoint manual mark finishes
with partial post-coverage, even though session metadata is complete.

## Actual synthetic results

Fixed seed 42, start 2026-10-23T12:00:00Z, 12-minute duration. Actual exported
sessions, JSON summaries, CLI results, and reconstructed timelines are retained locally under
`build/milestone-2-acceptance` (ignored generated evidence).

| Scenario | Observations | Events | Incidents | Raw triggers | Capture | Evidence order |
| --- | ---: | ---: | ---: | ---: | --- | --- |
| A healthy_drive | 30,966 | 9 | 0 | 0 | — | All seven healthy phases |
| B voltage_tcm_dropout | 38,167 | 17 | 1 | 2 | captured, 182.9 s | Low voltage → latency → third timeout → restoration |
| C tcm_dropout | 38,167 | 16 | 1 | 1 | captured, 180 s | Stable voltage; latency → third timeout → restoration |
| D mechanical_fault | 45,368 | 13 | 1 | 1 | captured, 180 s | Boost mismatch → new synthetic DTC; duplicate DTC retained |
| E adapter_disconnect | 30,880 | 13 | 1 | 1 | partial, 180 s span | Disconnect → 2 s missing observations → reconnect |
| F manual_mark | 30,966 | 12 | 1 | 1 | captured, 180 s | Manual source event → indexed trigger → finalization |

B triggers occur at midpoint − 2 s (`voltage_excursion`) and midpoint + 0.9 s
(`module_timeout`), so the merged window extends to the latter trigger + 120 s.
C has only a communication trigger and no voltage trigger or invented electrical
cause. D uses `MOCK_BOOST_TRACKING`, not a claimed manufacturer DTC/decoder.
E has one `transport_disconnect`, zero independent ECU timeout incidents, explicit
`coverage_gap`, and a completed, recoverable session.

Actual Python reconstruction of B returns ID
`voltage_tcm_dropout.incident-000001`, two exact evidence references, status
`captured`, bounds 1792757098000–1792757280900, and 9,707 ordered canonical records.
C returns 9,552 records, D 11,352, E 7,664, F 7,749. A partial gap describes missing
observations; a full requested time span does not imply continuous coverage.

## Verification and limits

Actual checks passed: `gofmt`, all Go package tests and `go vet ./...`,
59 Python tests, and seven simulator-packaging tests through
`./scripts/test_all.sh`. These cover original healthy behavior,
voltage crossing/reset/quality, expected timeout counts/restoration, multi-module
correlation, DTC deduplication, distinct transport loss, manual lifecycle, exact
windows, near-start/early-stop partial evidence, silence and disconnect gaps,
overlap merge/link behavior, storage failure and deterministic exports.
Python tests consume actual Go output for every scenario, check exact trigger
order and source references, reconstruct persisted merged/linked incidents, and
reject malformed revisions, references, links, schemas, sequence and status.
Legacy Milestone 1 fixture/export tests remain in the suite.

No concurrent execution was introduced; the recorder remains single-owner.
Interrupted journals can be inspected with `--allow-incomplete`; no automatic
repair, crash recovery or device power-loss durability is claimed. Capture
revisions checkpoint progress; an interrupted capturing snapshot can conservatively
lag the most recent session sample. Full-session logs remain authoritative.

Native sources, project generation, build inputs and workflow files did not change,
so no new native build was needed. The latest Milestone 1 Linux workflow also
[passed](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40033132177).
Synthetic acceptance provides no real-car diagnostic validation.
