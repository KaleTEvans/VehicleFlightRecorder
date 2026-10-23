# Milestone 1: record and inspect a healthy synthetic drive

Milestone 1 implements the local Go recorder and Python export inspection.
Milestones 0/0.5's SwiftUI app and preview workflow remain the static shell; running
this command does not start a drive inside the iPhone/Appetize UI. No incident
engine or real adapter communication is implemented.

## Setup and generate

From the repository root, use Go 1.24+ on PATH and the established Python 3.11+
virtual environment. On the original WSL workstation:

```bash
cd ~/dev/VehicleFlightRecorder
export PATH="/usr/local/go/bin:$PATH"
PYTHON="$HOME/.pyenv/versions/3.11.9/bin/python3" ./scripts/bootstrap.sh
export GOCACHE="$PWD/.cache/go-build"
export GOMODCACHE="$PWD/.cache/go-mod"
export TMPDIR="$PWD/.cache/tmp"
mkdir -p "$TMPDIR"
cd core
go run ./cmd/mock-session \
  --pack ../vehicle-packs/alfa-giulia-qv-2017 \
  --output ../build/healthy-demo \
  --session-id healthy-demo \
  --seed 42 \
  --start 2026-10-23T12:00:00Z \
  --duration 12m \
  --buffer-window 60s
cd ..
analytics/.venv/bin/python -m vehicle_flight_recorder validate build/healthy-demo
analytics/.venv/bin/python -m vehicle_flight_recorder summarize build/healthy-demo
```

The output directory must not already exist. Use another output path for a new
run; the command never overwrites a previous recording. `--pack` and `--output`
are required. All other shown values are deterministic defaults. Duration must
be whole seconds, 2 minutes–24 hours; start uses RFC3339 with at most millisecond
precision; buffer retention is a positive whole-millisecond duration. Seed is a
signed 64-bit integer. Stdout returns final metadata as JSON; errors go to stderr
with a nonzero exit code. Relative paths are relative to the command's working
directory, not the source file.

The drive spans 720 logical seconds, runs as fast as file writing permits, and
emits 30,966 observations, nine events, and zero incidents. Rates: four signals
at 10 Hz, one at 2 Hz, one at 1 Hz, including time zero and the final sample.
Seven phases cover engine start, idle, acceleration, cruise, deceleration, stop
and shutdown. All data is synthetic; no physical adapter is connected.

Python emits deterministic, sorted structured JSON with identity/state/duration,
actual counts, per-signal count/unit/first/last timestamps, and event-type counts.
The summary is not a mechanical diagnosis or an incident reconstruction.

## Reproduce a recording

Run the same Go command with a different fresh `--output` directory while keeping
all other inputs and pack contents unchanged. All five output files must match
byte for byte; paths and wall-clock generation times are not serialized. The seed,
start time, session identity, catalog order/rates, duration, generator version and
pack version define reproducibility. Tests compare identical files and check that
changing the seed changes waveforms. Cross-version Go/toolchain compatibility of
random waveforms is not claimed beyond the recorded generator/pack implementation.

## Inspect an interrupted recording

```bash
analytics/.venv/bin/python -m vehicle_flight_recorder summarize \
  build/<unfinalized-session> --allow-incomplete
```

Without the flag, well-formed but unfinalized metadata produces exit status 2.
With it, actual records are counted and reported as incomplete, without repairing
files. Malformed/missing evidence still produces exit status 1. Valid finalized
sessions return 0. A valid unfinalized inspection explicitly requested with the
flag returns 0 but retains `status: incomplete` and a null finalized duration.
Malformed JSONL errors include filename/line; a truncated final line is never
silently discarded. Command usage errors also use argparse's exit status 2.

See [DATA_SCHEMA.md](DATA_SCHEMA.md) for exact fields, ownership, ordering,
finalization and durability limits. See [VEHICLE_PROFILE.md](VEHICLE_PROFILE.md)
for the mock-only catalog and synthetic provenance. No personal captures or bulk
generated sessions belong in Git.

## Tests

```bash
./scripts/test_all.sh
```

This runs Go tests/vet, Python tests, and the existing simulator-packaging checks.
The Python suite builds and runs the actual Go command, validates its output,
checks expected counts, compares two complete sessions, and confirms overwrite
rejection. Therefore standalone `pytest` also requires Go on PATH. It does not
use a separately handwritten session as the only cross-language acceptance test.

Go tests cover the inclusive buffer cutoff, expiry, same-time ordering, independent
snapshots/event attributes, invalid lifecycle operations, poisoned write/finalize
failures, JSONL short writes, incomplete metadata retention, profile validation,
rate/phase correctness, cancellation, emitter failures and seeded determinism.
Python unit fixtures additionally cover malformed records, version/count/source
errors, timestamp bounds/order, truncated logs, missing files, non-finite values,
and read-only incomplete-session inspection.

The recorder is intentionally single-owner. Sequential logical-time tests use no
wall-clock sleeps. It is not yet a thread-safe iOS service. Incident capture and
fault scenarios belong to Milestone 2; the native bridge and UI belong to later
milestones. Existing iOS source/build inputs are not changed by Milestone 1.

## Acceptance evidence — October 23, 2026

**Milestone 1: COMPLETE locally. Milestone 2 not started.**

Executed from the repository root with the Go/Python environment above:

```bash
(cd core && go run ./cmd/mock-session \
  --pack ../vehicle-packs/alfa-giulia-qv-2017 \
  --output ../build/milestone-1-acceptance/healthy-a \
  --session-id healthy-demo --seed 42 --start 2026-10-23T12:00:00Z \
  --duration 12m --buffer-window 60s)
```

Repeated with the identical inputs and output directory `healthy-b`, then ran:

```bash
analytics/.venv/bin/python -m vehicle_flight_recorder validate \
  build/milestone-1-acceptance/healthy-a
analytics/.venv/bin/python -m vehicle_flight_recorder summarize \
  build/milestone-1-acceptance/healthy-a
```

Observed: `status: valid`, `state: complete`, `duration_seconds: 720`,
`observation_count: 30966`, `event_count: 9`, `incident_count: 0`, six signals.
All five files matched byte for byte between the two output directories.
The observations SHA256 was
`2365646bb0ae64e07aa928ae5b27f6f3cec15ef924af7030efe21f2acb5d6847`.

Final `gofmt`, `go test ./...`, `go vet ./...`, editable Python installation,
and `./scripts/test_all.sh` passed. Pytest: **43 passed**, including the real Go
export integration test. Existing simulator packaging checks: **7 passed**.
Repository diff/whitespace checks passed. No Swift, iOS project/build scripts,
Go mobile facade or GitHub workflow inputs changed, so no new native preview
build/upload was needed or claimed for this milestone. These are local acceptance
results recorded before commit; GitHub CI results are separate evidence.

Decisions left open by the plans are recorded in DATA_SCHEMA.md: numeric-only
observations, event session/source traceability, on-disk recording/complete states,
finalized counters, and explicit read-only incomplete inspection. There is no
major architecture deviation. The recorder is single-owner; interrupted data is
preserved and diagnosed rather than auto-repaired, and phone/power-loss durability
is not validated. The earlier three planning handoffs remain separate documents.
