# Live timing and provenance migration design

Status: the original M4 migration design is retained below. Milestone 6 now
implements the bounded live contract described in the final section; physical
stationary acceptance remains pending. **M4 itself did not implement live polling**.
No GPS, Core Motion, source fusion or asynchronous ingestion engine is introduced.

## Current recording contract

Mock sessions use one deterministic source and Go-owned logical Unix milliseconds.
The native refresh calls advance by 2,500 ms, with each facade call limited to
10,000 ms. Pausing UI mock playback does not change measurement time or create
samples. The writer accepts nondecreasing timestamps, with contiguous sequence
numbers across observations/events. Equal times retain insertion order; the
incident engine and Python reconstruction refer to exact file/line/sequence.
Raw transport files are legitimately empty for normalized synthetic waveforms.

These properties are tested mock behavior. Physical requests must never run on
the mock clock, at accelerated playback rates, or simply because a UI redraws.

## Proposed live clock mapping

At live session initialization record a wall-time anchor and a monotonic anchor
from the acquisition owner. Capture request admission, actual send/partial-write
progress and response receive/completion using the same monotonic clock where
possible. Compute durations from monotonic differences. Map a receive instant to
canonical session time using the anchors rather than following wall-clock jumps.
Document units/precision and verify cross-language conversions; do not label
coarse receipt timestamps as microsecond ECU measurement accuracy.

Receipt/completion is not an ECU measurement timestamp. Preserve an actual source
time only when its meaning/clock is known. Otherwise measurement time remains
unknown. Track wall-clock adjustment and new connection/time-anchor information
as explicit events; never rewrite prior evidence to make a clock jump disappear.
Define suspend/resume behavior against clocks actually available on iOS/Go and
test it before claiming elapsed coverage. Missing delivery creates a gap; resume
must not manufacture past measurements.

## Provenance and freshness

M5 raw capture must retain direction, connection generation, sequence, exact byte
encoding, native receipt/write time and attempted versus accepted/uncertain bytes.
M6 assigns transaction identity and relates decoded observations to raw references,
request-send and response-receive intervals, ECU/adapter identity and decoder/
catalog/pack versions. Use a nullable/absent source measurement timestamp when
unknown. One transport chunk can contain partial or multiple responses; it is
not a complete transaction. Partial writes and late replies need explicit status.

The canonical observation time and per-source measurement/receipt metadata must
have distinct documented meanings. Retain valid zeroes and real repeated replies.
A dashboard can display the retained value with age/stale status, but refreshing
that display is not an acquisition. Rate reports must count successfully decoded
real observations over a defined acquisition interval and show interruptions.
Changing values alone is not proof of ECU sensor refresh rate.

## Ordering and generation policy

Keep one serialized live acquisition owner initially. Request scheduling and
reply handling must preserve raw ingestion order and an explicit connection
identity. A late reply cannot satisfy a new connection's request. Either map late
measurements into a compatible receive-ordered canonical contract with original
source timing retained, or deliberately version/change the ordering contract.
Do not silently sort or nudge timestamps to satisfy the old writer.

Session generation is not transport generation: one drive may contain reconnects,
and a connection may exist before a drive. Keep gaps and identities explicit.
The M4 session-bound mock tick is not a complete real reconnect policy.

## Coordinated migration points

Before adding live data, inspect/update these together:

1. Go telemetry models/validation, writer/metadata/sequence rules and facade JSON.
2. Raw connection/transaction schema, byte encoding and reference integrity.
3. Incident references, quality/coverage semantics and live rule applicability.
4. Python session/raw validation, compatibility routing and reconstruction.
5. Swift decoding and native archive's strict file list/report inclusion.
6. Capability report version, timing populations and requested/achieved rates.
7. Native export validators and regression fixtures for old mock plus new live data.

Existing schema 1 exports remain unchanged in M4. Version incompatible persisted
changes in M5/M6 explicitly and test unknown-version rejection; do not weaken the
Python empty-raw check without defining supported raw content. Keep schemas and
bridge compatibility versions distinct.

For later OBD, GPS and phone-motion concurrency, every source needs independent
provenance/clock mapping, uncertainty and late-arrival semantics. Revisit the
single-source ordering contract before admitting asynchronous producers. This
extension path reserves concepts only, not speculative implementation or a V2
feature commitment.

## Implemented Milestone 6 live timing and freshness

The dedicated native ExternalAccessory executor owns stream callbacks, bounded UI
commands and a 100 ms timer. The timer asks Go for the next allowlisted request;
SwiftUI's 500 ms refresh only reads a snapshot. Timers may be delayed or suspended
by iOS, and no dispatch cadence is represented as a measurement rate.

At explicit session Start, the app captures Unix milliseconds and native
`mach_continuous_time` nanoseconds. Go retains both anchors. All subsequent native
clocks must be nondecreasing; transaction offsets are native time minus its
anchor. Canonical receipt time is Unix anchor plus elapsed whole milliseconds.
Equal milliseconds retain canonical record sequence order.

`request_start_ns` records Go admission, `stream_accepted_ns` the complete native
write acceptance, and `receive_ns` the final response-framing chunk. Latency is
response completion minus stream acceptance. Native acceptance is not adapter or
ECU acknowledgement. ECU measurement time is unknown/null. Raw fragmented chunks
retain their actual native offsets; no value is timestamped by UI refresh.

Wall-clock offset changes above one second produce a `wall_clock_adjusted` event
without remapping old or future canonical receipt times. Observed owner gaps over
three seconds produce `acquisition_pause_observed` with unknown cause; scene
foreground/background/inactive transitions are separate evidence, not proof of
OS suspension. Disconnect/reconnect have their own events and generations.

Snapshots retain the last decoded value and show age. After 15 seconds it is
stale; disconnected or currently unavailable capability states show unavailable.
No missing value becomes numeric zero. Reconnection clears the displayed old
values and capability scope, repeats setup and re-discovers support. Historical
observations remain unchanged with their original transaction/decoder identity.

Capability statistics count successfully decoded observations per signal/ECU over
the full elapsed session, including setup and interruptions. Maximum gap includes
initial and terminal uncovered spans; latency includes min and p95. Nominal
intervals, actual receipts, equal/changing values and internal sensor refresh are
distinct concepts. Sensor refresh stays unknown; physical rates require actual
phone exports and a reference comparison.

The first live contract remains a single ordered OBD source. GPS/motion must define
source clocks, alignment, late-arrival admission and provenance before changing
that invariant. They are not implemented here. Hands-off sessions, reliable
background behavior and wake limitations remain Milestone 7 acceptance work.
