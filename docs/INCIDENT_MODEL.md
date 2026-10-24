# Incident engine — Milestone 2

Go evaluates canonical observations/events. `incidents.Detector` emits trigger
candidates; the single-owner engine captures their windows. Recorder/storage keep
full-session evidence authoritative. No diagnostic transactions are performed.
All shipped thresholds and fault definitions have explicit synthetic provenance.

## Rules and rearming

The mock pack configures a valid-quality voltage crossing below 12 V or a drop of
at least 1.5 V between consecutive valid samples within 1 s. One trigger latches
until voltage returns to at least 13 V without another qualifying drop. Wrong
units, stale/invalid samples and disconnected transport do not trigger voltage
rules. The normal synthetic shutdown to 12.6 V does not cross these rules.

Three consecutive `diagnostic_timeout` events with `expected_response: true` for
a configured critical module emit one `module_timeout`. An explicit
`response_restored` resets that module's count/loss; further timeouts while latched
do not flood incidents. An unrelated missing signal is never an ECU timeout.
Loss thresholds reached for two configured modules within 1 s also emit
`multi_module_loss`; restoring enough modules to leave fewer than two losses
rearms correlation. Counts and critical module names come from pack data.

`dtc_observed` emits `new_dtc` once per module/code; duplicate observations remain
in the event log. A synthetic `dtc_removed` event rearms that code; this is a
canonical observation, not a vehicle command. DTC categories are `powertrain` or
`diagnostics`. `transport_disconnected` has its own trigger, clears communication
counters and suppresses timeout/voltage/DTC interpretation until
`transport_connected`. Repeated disconnect events do not retrigger.

`Recorder.MarkIncident(label, timestamp)` creates an immediate `manual_marker` and
returns its incident ID. It requires a configured, actively recording session and
a nonblank label of at most 256 bytes. Repeated manual marks remain discoverable,
even when their evidence windows merge. Native button integration is future work.

## Capture and overlap

Default pre/post windows are 60/120 seconds. Pre-retention must fit the configured
rolling buffer. Capture uses inclusive logical timestamp boundaries, without
sleeps. The engine reads the recorder-owned buffer synchronously, retains no
sample pointers, and stores only indexes over full-session logs.

A trigger whose requested window overlaps an **actively capturing** incident joins
that incident; its post deadline extends to the latest trigger plus the post
window. The original primary trigger stays unchanged, all triggers remain ordered,
and severity is the highest trigger severity. If multiple active captures overlap,
the oldest active incident is chosen deterministically. A capture completes when a
record reaches its deadline; a trigger evaluated on that deadline can extend it
before finalization. Later triggers create new incident IDs and symmetric links to
already completed captures whose requested windows overlap. Evidence is never
copied into incident-specific observation files. Related windows can contain the
same canonical records without duplicating stored samples.

Missing pre-history and early stop produce explicit incomplete reasons. Transport
loss records a half-open gap [disconnect, reconnect); exported gap endpoints
also delimit the interval shown in reconstruction. Silence beyond the configured
2 s tolerance records a gap from last-record + 2 s to the next record. Gaps clip to
available capture bounds. `partial` with `coverage_gap` can span the full requested
180 s while still having missing evidence inside it. A preceding voltage event is
temporal evidence; it does not establish a cause of TCM loss.

## Persistence and reconstruction

`incidents.jsonl` is an append-only revision journal. Each row contains a complete
incident snapshot with `schema_version: 1`, identity, increasing `revision` starting
at 1, primary trigger, configured window sizes, requested/available bounds,
severity, `manual`, `triggers`, `related_incident_ids`, `coverage_gaps`, and
`incomplete_reasons`. Readers select the latest revision per ID. Updates occur on
triggers, gap changes, completion, stop, and added links. Completed capture facts
are immutable; subsequent revisions may add related IDs. Capturing snapshots are
checkpoints, so available-end coverage can lag later session samples until the
next update. Unfinalized sessions never imply complete capture.

Every trigger references an exact canonical filename, 1-based line, timestamp,
record sequence and signal/event type. `record_sequence` is contiguous across
observations and events, including generated `incident_triggered` and
`incident_capture_finalized` events. These derived events do not recursively
trigger detectors. Trigger and incident counts are distinct.

Python validates revisions, immutable evidence, source references, bounds, status,
links and total ordering. `incident <directory> <id>` returns incident metadata and
canonical rows within available bounds, ordered by timestamp then sequence. It
keeps gaps and triggers explicit and makes no causal diagnosis. Unknown IDs and
corrupt files fail. `--allow-incomplete` inspects preserved well-formed logs without
repairing them. Raw transport remains empty; nonempty raw logs require a future
supported decoder. Storage failures prevent complete metadata; no power-loss
recovery, concurrent recorder API, or real-vehicle diagnostic accuracy is claimed.
