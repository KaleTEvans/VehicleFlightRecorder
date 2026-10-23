# Session schema 1 — Milestone 1

The Go recorder writes numeric observations and discrete events. Python validates
and summarizes those exact exports. Schema version 1 is supported; other versions
fail explicitly. These contracts implement MVP sections 6–7 and retain its signal
names. Enterprise ontology examples have not silently renamed the MVP catalog.

## Files and lifecycle

The caller supplies a **new** directory; an existing directory is never reused or
overwritten. The Go CLI creates:

```text
<session-directory>/
├── session.json
├── observations.jsonl
├── events.jsonl
├── incidents.jsonl
└── raw/
    └── transport.jsonl
```

Files use UTF-8 JSON, newline termination, and no blank JSONL lines. Directories
are created with private permissions (0700), files with 0600, subject to platform
behavior and umask. Output path is not serialized, so two different directories
can hold byte-identical sessions with the same configured identity.

`session.json` fields:

| Field | Contract |
| --- | --- |
| `schema_version` | Integer 1 |
| `session_id`, `source` | Nonempty strings; CLI source is `mock` |
| `profile_id`, `vehicle_pack_version` | Loaded pack identity/version |
| `start_time_unix_ms` | Positive signed 64-bit millisecond timestamp |
| `end_time_unix_ms` | Null while recording; at/after start when complete |
| `state` | `recording` or `complete` on disk |
| `observation_count`, `event_count` | Nonnegative integers; finalized totals only when complete |
| `incident_count` | Zero in Milestone 1 |
| `mock` | Required for mock source; scenario, seed, logical duration/step and generator version |

Mock metadata uses `scenario: healthy_drive`, signed 64-bit `seed`, planned
`duration_ms` (whole seconds, 2 minutes–24 hours), `step_ms: 100`, and
`generator_version: "1"`. End time cannot exceed the planned mock duration.
An intentionally early stop can finalize a shorter session; metadata retains the
planned generator duration for reproducibility. The CLI runs the configured full
duration. Start time plus planned duration must fit a signed 64-bit timestamp.

The recorder writes `recording_started` and `recording_stopped` events. It appends
records directly, checks short/write errors, syncs/closes all logs, then atomically
replaces metadata with a complete state and final counts. A close, sync, or metadata
failure is reported; failed sessions cannot subsequently be finalized as healthy.
Closing without Stop leaves on-disk state `recording`.

## Observations and events

Observations require `schema_version`, `timestamp_unix_ms`, `session_id`,
`signal_id`, `source`, finite numeric `value`, nonempty `unit`, and `quality`.
Quality is `valid`, `stale`, or `invalid`. Numeric zero is legitimate; missing/null
values are rejected, never coerced to zero. This implementation does not yet
represent null-valued missing measurements. Optional provenance: `module`, `bus`,
`raw_request`, `raw_response`, `decoder_version`, and `vehicle_pack_version`.
Provided pack versions must match session metadata.

Events require `schema_version`, `timestamp_unix_ms`, `session_id`, `source`,
`event_type`, and severity (`info`, `warning`, `error`). Optional `module` and
`attributes` preserve context; attributes, when present, must be a JSON object.
Go stores attribute bytes as `json.RawMessage` and clones them for ownership.
Session/source fields extend the MVP's short example for explicit traceability.

The recorder accepts globally nondecreasing logical timestamps, including equal
timestamps in stable insertion order, and matching session/source. Records before
session start, after the configured mock horizon, or backwards in time are rejected.
Python checks nondecreasing order independently in each file and start/end bounds;
separate logs do not encode a cross-file sequence number. A completed session's
counts must match actual rows. No filename ordering is interpreted as causation.

## Rolling buffer

The default retention is 60 seconds; a positive whole-millisecond duration is
configurable. At latest timestamp T, retain records in **[T − window, T]**,
including the exact cutoff. Same-time events/observations preserve insertion order.
Backward time is rejected rather than sorted silently. Advancing logical time can
expire data without a new sample. Snapshots and inserted records own independent
copies, including event attributes. All full-session records remain on disk after
in-memory expiry. The implementation has one caller/owner; it is not a concurrent
recorder API. Incident triggers/post-windows are not implemented.

## Mock evidence and empty files

Six pack-defined synthetic signals use 10 Hz, 2 Hz, or 1 Hz sampling. A 100 ms
logical step advances without wall-clock sleeps. `mock_phase_changed` events
describe engine start, idle, acceleration, cruise, deceleration, stop, and shutdown.
Samples exist at time zero and the inclusive final sample time. A default 12-minute
drive writes 30,966 observations and nine events (seven phases plus start/stop).

`incidents.jsonl` is deliberately empty. `raw/transport.jsonl` is also empty:
the source is normalized synthetic waveforms and has no raw adapter byte stream.
No adapter requests/responses are fabricated to fill that file. Python rejects
nonempty incident/raw files as unsupported instead of silently ignoring evidence.
Future real recording must preserve actual raw evidence alongside decoded values.

## Incomplete sessions and validation

On interruption, metadata can remain `recording` with zero/stale counts while
valid records already exist. Python returns nonzero by default for such sessions;
`--allow-incomplete` inspects well-formed preserved records, reports actual row
counts and `status: incomplete`, and leaves finalized duration null. Nothing
repairs files or rewrites metadata. Missing required files indicate an incomplete
initialization or damaged export and remain errors even with opt-in.

Malformed JSON, duplicate keys, non-finite numbers, unsupported schema, foreign
session/source, invalid timestamps/counts, and truncated/non-newline-terminated
JSONL rows fail with useful file/line diagnostics. An interrupted final row is
preserved and reported; it is not silently dropped. Complete metadata is never
inferred merely because all expected filenames exist.

The writer syncs files before committing metadata, but does not fsync parent
directories or guarantee storage-device power-loss durability. Atomic rename is
tested at the filesystem/API level; no actual phone crash/power-loss recovery is
claimed. Full session logs remain authoritative; later incident windows will be
views/indexes over that evidence, not separate truth.
