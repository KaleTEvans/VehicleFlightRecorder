# Session schemas — mock 1 and live 2

The Go recorder writes numeric observations and discrete events. Python validates
and summarizes those exact exports. Schema versions 1 (legacy/mock) and 2 (explicit live acquisition) are supported;
other versions fail explicitly. The sections below describe schema 1; the live
extension follows at the end. These contracts implement MVP sections 6–7 and retain its signal
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
| `incident_count` | Number of distinct incidents, independent of revision/trigger count |
| `incident_capture` | Optional pre/post/silence windows in milliseconds; required when incidents exist |
| `mock` | Required for mock source; scenario, seed, logical duration/step and generator version |

Mock metadata uses `scenario: healthy_drive`, signed 64-bit `seed`, planned
`duration_ms` (whole seconds, 2 minutes–24 hours), `step_ms: 100`, and
`generator_version: "1"` for healthy generation. Fault scenarios use generator
version `"2"`; supported names appear in [Milestone 2](MILESTONE_2.md). End time cannot exceed the planned mock duration.
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
New recorder exports carry `record_sequence`, contiguous from 1 across both logs,
including derived incident events. Python checks per-file and merged order and
start/end bounds. Legacy Milestone 1 exports with sequences entirely absent still
validate; their equal-time cross-file order is unspecified. Incident references
require sequences, so legacy files cannot silently claim precise trigger order. A completed session's
counts must match actual rows. No filename ordering is interpreted as causation.

## Rolling buffer

The default retention is 60 seconds; a positive whole-millisecond duration is
configurable. At latest timestamp T, retain records in **[T − window, T]**,
including the exact cutoff. Same-time events/observations preserve insertion order.
Backward time is rejected rather than sorted silently. Advancing logical time can
expire data without a new sample. Snapshots and inserted records own independent
copies, including event attributes. All full-session records remain on disk after
in-memory expiry. The implementation has one caller/owner; it is not a concurrent
recorder API. The incident engine borrows this buffer synchronously to determine pre-coverage.

## Mock evidence and empty files

Six pack-defined synthetic signals use 10 Hz, 2 Hz, or 1 Hz sampling. A 100 ms
logical step advances without wall-clock sleeps. `mock_phase_changed` events
describe engine start, idle, acceleration, cruise, deceleration, stop, and shutdown.
Samples exist at time zero and the inclusive final sample time. A default 12-minute
drive writes 30,966 observations and nine events (seven phases plus start/stop).

`incidents.jsonl` contains the revision journal described in
[the incident contract](INCIDENT_MODEL.md); it stays empty for healthy drives.
`raw/transport.jsonl` remains empty:
the source is normalized synthetic waveforms and has no raw adapter byte stream.
No adapter requests/responses are fabricated to fill that file. Python rejects
nonempty raw files as unsupported instead of silently ignoring evidence.
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
claimed. Full session logs remain authoritative; incident windows are
views/indexes over that evidence, not separate truth.

## Milestone 2 compatibility

The session schema remains 1 with additive `record_sequence` and
`incident_capture` fields. This reader supports both original Milestone 1 exports
and the incident extension; older Milestone 1 readers reject nonempty incident
files explicitly. The mock vehicle pack is version 0.2.0. Healthy waveform values,
rates, counts and phases are preserved, while serialized bytes change because of
sequence numbers and pack provenance. Fault-scenario catalogs add scoped mock
signals without changing the healthy six-signal catalog. Incident snapshots also
use schema 1; unsupported versions fail explicitly. See the incident contract for
revision semantics, coverage reasons and reconstruction.

## Milestone 6 live extension — schema 2

A live session uses `source: obd-live`, `profile_id: generic-obd-can11-500`,
`vehicle_pack_version: none`, no `mock`, and required `live` configuration:
`origin` (`physical`, `native-fixture`, `scripted-fixture`), `operating_state`
(`unknown`, `ignition_on_engine_off`, `stationary_running`), `workload` (`combined`,
`rpm_only`, `coolant_only`, `voltage_only`), `native_anchor_ns`,
`catalog_version: standard-live-can11-1`,
`protocol_profile: can11-single-frame-headers-pci`,
`report_file: capability.json`, `start_origin: manual`. Fixtures remain explicitly
identified even though they exercise the live schema. An origin label alone is
not proof that measurements were physically acquired.

Canonical observations/events carry schema 2 and share the existing contiguous
record sequence. Incident journal rows independently retain incident schema 1.
A single start anchor maps native continuous-clock offsets to canonical receipt
milliseconds; `wall_clock_adjusted` records changes without rewriting timestamps.
ECU measurement time is unknown, not receipt time under another name.

Every live observation requires `module` (responding CAN ECU ID, or `adapter`),
`bus: HS-CAN` (selected acquisition bus), decoder `1`, pack `none`, and provenance:
`transaction_id`, `generation`, `raw_refs`, `request_start_ns`,
`stream_accepted_ns`, `receive_ns`, `measurement_unix_ms: null`.
Raw references are positive increasing row numbers. Free-form `raw_request` and
`raw_response` are forbidden; exact bytes reside in the raw ledger. Each ECU reply
remains separate, including conflicting values; `7E9` does not imply TCM.
Adapter voltage is adapter input voltage, not a verified battery terminal reading.

The ZIP contains the five existing files plus `capability.json`, a version-2
`live-acquisition` report. It includes fixed configuration and initialization
versions, per-generation adapter metadata, documented schedule, raw transactions,
decoded observations, scoped capabilities, outcomes and measured rate/gap/latency
statistics. Rates describe decoded receipt cadence across the full session,
including setup. They do not establish underlying sensor refresh rates.
`vehicle_identity: unknown` and `reference_comparison: unmeasured` remain honest
until separately compared physical evidence is available.

The raw ledger rows contain `session_id`, contiguous `sequence`, `generation`,
`transaction_id` (empty for unsolicited bytes), `kind` (`requested`, `accepted`,
`received`, `late_received`, `unsolicited`), `monotonic_ns` (offset from native
anchor), and `bytes_base64`. Accepted bytes establish native stream acceptance,
not ECU receipt. Timed-out late bytes stay raw and do not become new observations.
All raw bytes must be appended successfully before the report checkpoint and
canonical observation. A failed evidence write stops acquisition and leaves the
session incomplete; a failed session cannot be finalized as healthy.

Python independently replays setup and decoding, checks report raw rows against
the raw file, and matches each canonical observation's value, ECU, transaction,
raw references, timestamps and decoder. Unknown fields, altered provenance,
missing rows and duplicate claims are rejected. Complete exports require an exact
six-file layout, finalized counts, all report observations, stopped report and
matching end time. Schema-1 mock exports continue requiring an empty raw file.

Report checkpoints use checked, synced temporary writes and atomic rename before
canonical append. An interruption can leave an older checkpoint, a well-formed
uncheckpointed raw tail, report observations not yet in the canonical prefix, or
a `.capability-*` temporary file. `--allow-incomplete` reports these explicitly;
it does not repair, drop truncated rows, infer completion, or substitute a report
observation into canonical history. Required files still must exist. Original
phone Files evidence is retained when app export is unavailable for an unfinished
session. No actual crash/power-loss durability claim is made.

Limits: 600 seconds checked when the native owner runs, 16 connection generations,
2,048 transactions, 16,384 raw records / 2 MiB payload, 8,192 decoded report
observations, 16 MiB capability report, 64 MiB reader budget per canonical/raw log
(and raw writer bound), and 128 MiB native ZIP. Exceeding bounds stops admission; original evidence remains. The
maximum is not an OS wake guarantee: after suspension, actual observed elapsed
time may exceed 600 seconds before the owner resumes and stops.

Independent future OBD/GPS/motion sources must define their own source timestamps,
clock alignment, sequence and late-arrival admission before extending the current
single-source ordered recorder. Receipt mapping here does not silently authorize
asynchronous ingestion or sensor fusion.
