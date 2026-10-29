# Codex Handoff — Vehicle Flight Recorder, Milestone 6 Only

## Assignment and prerequisites

Implement **Milestone 6 — adapter initialization + standard OBD** in:

```text
/home/kaletevans/dev/VehicleFlightRecorder
```

This is a future implementation handoff; writing it does not start this milestone.
Milestones 4 and 5 must be implemented and accepted first, with actual bridge,
stream, signing/install and physical MX+ evidence. Inspect `git status`, current
code, schemas, fixtures, workflows and reports. Preserve legitimate changes.
If a prerequisite is incomplete, identify it and complete independent work within
this milestone; do not silently implement earlier milestones or claim physical
acceptance using only mocks.

## Preparation status and current M5 boundary

Planning update — October 29, 2026: M5 implementation passed portable checks,
18 native unit tests, two UI tests and unsigned device builds. Its physical
acceptance remains pending; the user has applied for Apple Developer enrollment
and is awaiting approval. Enrollment, provisioning, installation and actual MX+
stream evidence must be rechecked before claiming M6 physical prerequisites.
This status note does not activate M6.

The implemented M5 byte seam is synchronous on a dedicated native run loop,
with Go-owned version-1 standalone connection captures and one explicitly enabled
adapter firmware identity read. It does not provide a generic arbitrary-send
function, command scheduler or vehicle parser. Extend the Go-owned allowlist and
bounded native delivery contract deliberately for M6; preserve one authoritative
raw byte history and migrate/cross-reference connection evidence into live drives
without fabricating transactions from callback boundaries. Preserve legacy mock
exports and explicitly distinguish standalone connection files from drive exports.

The accepted build contract keeps the preview on Xcode 16.4 and uses a separate
Xcode 26.0.1 device profile for current App Store Connect uploads, with shared
Go 1.26.10 / pinned x/mobile and iOS 17 deployment floor. Inspect the actual
config/scripts rather than collapsing the profiles or reusing the old preview SDK
for TestFlight. Recheck primary platform requirements when implementation starts.

## Reuse the authorized offline preparation

The user subsequently authorized an offline preparation extension within M5:
`core/obd`, `core/acquisition` and `core/cmd/offline-acquisition`, with a standalone
synthetic report and Python validator. Read `docs/OFFLINE_ACQUISITION.md` and the
actual tests before implementing overlapping logic. The narrow catalog/profile,
resource/deadline policies and explicit-clock harness are tested starting points,
not physically verified adapter configuration or sampling rates. They are not
imported by the mobile facade and do not enable new physical commands.

M6 still owns documented adapter initialization, actual format/protocol validation,
expanded verified catalog/discovery, native integration and safe request admission,
production clock/configuration identity, coordinated live-drive schema/export
migration and physical stationary/reference acceptance. Preserve the existing
STI-only physical boundary until the appropriate integration is deliberately
implemented and tested. A synthetic acquisition report is not a live drive or
a passed physical capability report; do not simply re-label its source.

## Authoritative context and scope

Read the full current plans:

```text
planning/alfa_flight_recorder_mvp_plan.md
planning/vehicle_health_enterprise_architecture.md
planning/milestone_4.md
planning/milestone_5.md
planning/milestone_7.md           # next-milestone boundary
planning/milestone_8.md           # pack-expansion boundary
```

Then read README, the implemented `docs/MILESTONE_4.md` and `docs/MILESTONE_5.md`,
`docs/DATA_SCHEMA.md`, `docs/INCIDENT_MODEL.md`, `docs/VEHICLE_PROFILE.md`, bridge,
development and device/preview instructions. Focus on MVP §§1.2, 2, 5–7, 6.1.1,
10.1–10.3, 15–17, 20–22 and enterprise evidence/provenance, adapter and polling
boundaries. Priority: active human instructions, this handoff, MVP plan,
enterprise roadmap. Reconcile conceptual examples with actual code; do not revive
legacy package names from illustrative plan snippets.

The MVP is diagnostic-first, investigating the owner's intermittent problem.
Standard OBD support belongs to the generic layer; the Giulia QV is the first
physical validation vehicle. Verified enhanced Alfa definitions remain pack-owned
and follow in Milestone 8. A successful standard recording is not a diagnosis or
proof that TCM/manufacturer-specific signals are accessible.

Preserve the approved app/version/bundle (`Vehicle Flight Recorder`, `0.1.0`,
`com.kaleevans.vehicleflightrecorder`), iAP `com.obdlink`, iOS 17 floor and PPID
`221699-0001` metadata. Preserve the accepted native build/bridge contracts,
mock-only Appetize path and signed physical-device distribution path.

## Required work

### 1. Documented, tested adapter initialization

Implement a small Go-owned adapter transaction/initialization state machine on
the M5 ordered byte transport. Swift remains responsible for ExternalAccessory
streams and native lifecycle, not command strings, parsing or diagnostic policy.

Obtain the applicable official OBD Solutions/STN and standard protocol references
before selecting the exact initialization/read sequence. Record document versions,
supported firmware/protocol assumptions and fixtures. Do not guess AT/ST commands,
proprietary IDs, configuration effects or vehicle parameters. Reuse the verified
M5 identification exchange where applicable instead of implementing a second path.

Cover echo/formatting/prompt behavior, fragmented/coalesced bytes, bounded response
buffers, command completion, adapter status/error responses, no-data/timeout,
malformed/truncated replies and cancellation. A read chunk is not a response and
a prompt alone is not valid telemetry. Model adapter and vehicle protocol state
separately. Expose initialization failure honestly; never substitute mock values.

Enable live reads only after transcript/parser tests pass. Keep a documented
allowlist of necessary adapter setup and read-only standard diagnostic requests;
no arbitrary terminal or external pack-supplied unrestricted byte sequences.
Reset/reinitialization effects, if required by documentation, must be explicit,
bounded and logged, not an automatic response to every slow request.

### 2. Automatic supported-standard-parameter discovery

Implement known standard requests and decoders once in the shared Go diagnostic
layer. Query the connected vehicle for the supported subset of those implemented
parameters using the documented support-discovery mechanism. Discovery is bounded
in time, requests and supported blocks; no brute-force proprietary probing.

Keep these capability states distinguishable:

| State | Meaning |
| --- | --- |
| Unknown / not checked | No adequate discovery evidence yet |
| Advertised support | A valid applicable support response advertises the parameter |
| Observed / decoded | A real reply passed the implemented decoder/quality checks |
| Not supported | Valid protocol evidence establishes absence of support for that ECU/configuration |
| Temporarily unavailable | Supported/previously observed parameter currently cannot be acquired |

An arbitrary timeout or `NO DATA` text alone must not universally mean unsupported.
Define interpretation against the specific validated protocol and responding ECU.
Preserve raw support responses, discovery time, decoder version and unknown or
partial results. Avoid retrying known unsupported parameters during normal
recording; revalidate when vehicle, adapter or connection/protocol identity changes.
Cache only with sufficient configuration identity and provenance; do not reuse
one car's results as generic support facts. Unknown vehicle identity is an
explicit limitation, not permission for indiscriminate cache reuse.

Preserve responding ECU identity for multiple replies, including conflicting
values, negative responses and partial success. Do not arbitrarily select the
first reply, average ECUs or label every responder as TCM. Clearly state the
initial tested diagnostic protocol(s); unsupported legacy formats fail explicitly
rather than broadening into an untested universal parser.

### 3. Generic fallback and first real measurements

A live standard-OBD session must work without a matching enhanced Alfa pack.
Provide a generic standard catalog/decoder/configuration path and a clearly
reported coverage level. Keep generic standard knowledge separate from enhanced
manufacturer definitions. Unidentified or unmatched vehicles must not inherit
Alfa thresholds, critical-module names or synthetic scenario rules.

Start with a bounded documented set such as engine RPM, vehicle speed, coolant,
throttle or calculated load **where discovery confirms support**. Provide real
adapter-voltage acquisition as a separately identified adapter-origin measurement;
it is not automatically an ECU-measured battery-terminal voltage. Track units,
source ECU/adapter, decoder version, provenance, quality and acquisition age.
No promised parameter is guaranteed merely because it is standardized.

Never display missing values as fresh zeroes. Valid zero RPM/speed is meaningful;
unsupported, stale, invalid, timed-out and not-yet-observed must remain distinct.
Failure of one signal must not erase valid measurements from other responders.
Preserve accepted mock scenarios and tests as a separate explicit source.

### 4. Live timestamps and raw transaction provenance

Implement the live timing contract defined in M4 and MVP §6.1.1 before recording
physical observations. Preserve request-send and response-receive timing where
available, using monotonic durations and an explicit mapping to session wall time.
Define how partial writes and reconnects affect send/completion timestamps.

Receipt time is not automatically ECU measurement time. Preserve actual source
time only when supplied and understood; otherwise leave it unknown. Handle wall
clock adjustments, suspension, gaps, late responses and cross-connection data
explicitly while respecting recorder ordering and sequence invariants. Do not
silently sort evidence, rewrite timestamps to hide gaps or fabricate precision.

Assign transaction/connection identities and preserve exact inbound/outbound raw
bytes, direction, actual send/receive status and timing. Every decoded observation
must reference its originating transaction/raw evidence and source/decoder version.
Retain unsuccessful requests, malformed replies and uncertain transmission as
evidence without representing them as successful observations. A view refresh or
repeated presentation of a retained sample must not increment acquisition counts.

Keep the current single-source ordered-ingestion boundary explicit. Document the
future extension points for GPS/phone-motion provenance, alignment and late data,
but do not implement sensors, fusion or a speculative asynchronous-source engine.

### 5. Small bounded, prioritized polling scheduler

Implement only the scheduler needed for the diagnostic MVP, not the full enterprise
planner. Define a conservative configurable signal set, priority, request budget,
intervals, response deadlines, retry/backoff and total concurrency limits. Begin
with serialized transactions unless documented/tested transport behavior justifies
more. Never allow unbounded queues or overlapping replies with ambiguous ownership.

Live acquisition must use real monotonic time and an owner independent of the
dashboard refresh task. Do not drive physical requests with the accelerated mock
`Advance` loop or stop acquisition merely because a view disappears. Continue
only where native lifecycle/stream access actually permits it; record suspension
and coverage gaps honestly, without claiming unverified background execution.

A slow, unavailable or malformed signal must not indefinitely stall unrelated
recording. Use achieved timing/error evidence to reduce workload conservatively;
record configuration, backoff and coverage changes. Cancel pending/partly sent
work on disconnect; do not replay old requests after reconnect. Reinitialize and
revalidate relevant capabilities under a new connection generation as needed.

A failed ECU request is not an adapter disconnect; adapter loss must not be
reported as independent failure of every ECU. Retain module dropout/timeout and
missed coverage evidence when deprioritizing retries. Do not activate mock-only
incident thresholds on live measurements. Live detector/rule applicability must
be explicit and validated; manual marking and canonical incident capture remain
available without inventing a vehicle-specific diagnosis.

Requested poll rates are targets, not capabilities. Use the lowest useful rates
supported by conservative tests. Do not import nominal 10–20 Hz mock/FAST rates
as per-signal live guarantees or increase traffic to find a theoretical ceiling.

### 6. Initial capability and evidence-quality report

Export a versioned machine-readable report and readable summary, linked to actual
tested configuration and session(s). It must include:

- Adapter/model, firmware when available, native transport and diagnostic protocol.
- Vehicle identity when available, identity source and unresolved identification.
- App, bridge, schema, decoder, standard catalog and optional enhanced-pack versions.
- Signal identifier, responding ECU/adapter, units, provenance and validation status.
- Advertised/observed/unsupported/unavailable/unknown state and discovery evidence.
- Attempted/successful requests, timeouts, protocol failures and decode errors.
- Requested interval versus achieved successful-observation rate per signal/ECU.
- Response-latency distribution, inter-arrival gaps, stale intervals and sample counts.
- Measurement interval/denominator, pauses/disconnects, test duration, operating
  state and complete concurrent signal set, with workload/configuration changes.
- Reference-comparison method/results, limitations and unresolved discrepancies.

Measure a bounded selection individually and together under the intended initial
stationary workload. Keep configuration/discovery work parked. Define report
statistics and their populations clearly; distinguish successful transaction,
per-ECU response and decoded observation counts. Do not inflate rates with UI
refreshes, duplicate exports or repeated storage of the last value.

Equal values may be legitimate steady measurements. Neither equal nor changing
values alone establishes the ECU's internal sensor refresh rate; report it unknown
without independent evidence. Record achieved rate as observations over a defined
elapsed interval, showing interruptions rather than hiding them from all totals.

This milestone delivers the **initial stationary report**. Label driving/combined
road-workload performance unmeasured until M7 extends it. Compare suitable values
against a trusted reference under documented comparable conditions. Avoid running
competing diagnostic clients simultaneously unless that configuration itself is
supported/tested; sequential comparisons need explicit timing/context limitations.
Do not claim universal vehicle compatibility from the one Alfa validation car.

### 7. Session/schema/export integration

Record a real stationary-vehicle session through the existing recorder, with
actual raw evidence, decoded observations, events, capability provenance and
manual incident capture. Keep transport-only captures distinguishable from drives.
Provide the minimal developer controls/coverage display needed for this test;
retain the current UI and honest source labels, without a consumer redesign.

At M3, exports have a strict five-file layout, mock-specific validation and a
Python reader that rejects nonempty raw evidence. Inspect the M4/M5 changes and
explicitly migrate Go models/writer, Python validation/reconstruction, Swift
models, native archive packaging and runner export-validation scripts together.
Define the report location/reference and version any incompatible changes. Test
legacy mock exports and supported live formats; unknown versions, broken raw
references, foreign identities and truncated files must still fail clearly.
Do not merely relax validation to accept unexplained bytes or extra files.

Actual native-generated live exports must be retrieved from the phone, then
validated/summarized and reconstructed with the Python CLI. Keep failed or
interrupted sessions recoverable with honest completeness labels; no silent
repair or loss of authoritative raw evidence.

### 8. Hands-off readiness, without premature drive automation

Collect live engine/module availability and engine-running evidence useful to M7.
Validate what is observable in ignition-on/engine-off versus stationary running.
Scanner availability or engine/transmission replies alone may not imply running.
Do not assume standard OBD reaches a distinct transmission ECU; report missing
coverage and defer verified enhanced reachability to the appropriate pack work.

Do not require motion before preserving diagnostic observations. Do not let a
missing module erase startup faults. These observations inform, but do not yet
implement, the automatic start/stop/debounce/grace/reconnect policy in M7.
Record available background/locked-screen behavior on the real phone; the full
hands-off wake/termination/reliability acceptance matrix remains M7 work.

M7 also owns the approved automatic recording confirmation in MVP §15.1: after a
successful automatic session begins accepting telemetry, one quiet confirmation
per session, foreground in-app versus background local notification, optional
permission/preference and active/ended-session tap routing. M6 should expose
reliable session identity, start origin, successful admission and coverage facts
needed by that handoff, without adding automatic-drive policy, user-facing
notifications or permission prompts now. Internal accessory connect notifications
are not user-facing recording confirmations. No V2 cockpit redesign is required.

## Required verification

| Check | Required result/evidence |
| --- | --- |
| Protocol/transcript tests | Initialization, prompt/echo/status handling, fragmentation/coalescing, multiple ECU replies, malformed/oversized/incomplete input and unknown formats |
| Discovery tests | Valid support blocks, partial discovery, unsupported versus unavailable, unknown identity, cache invalidation and no enhanced-pack fallback |
| Scheduler tests | Bounded backlog/in-flight work, timeout fairness, conservative backoff, disconnect cancellation, no stale replay, no invented success |
| Timing/provenance tests | Monotonic latency, wall-clock change, unknown measurement time, sequence/tie handling, late response and exact raw reference round trip |
| Schema/export regression | Old mock and new live fixtures, corrupted/foreign/unsupported inputs, capability report and native archive validation in Go/Python/Swift |
| Native mock regression | Clean framework/app build, native unit/UI tests, evidence navigation, actual mock exports and Appetize flow remain functional |
| Physical stationary test | Real installed app, MX+, documented setup, initialization/discovery, supported signals, real voltage source, prolonged stable bounded acquisition and clean reconnect |
| Initial quality report | Individual and combined stationary workload, actual rates/latencies/gaps, reference comparison, unsupported/unavailable states and unresolved coverage |
| Actual phone export | Retrieved live session passes Python validate/summarize and incident reconstruction with raw provenance; record configuration/commit/build and artifact location |

Use `gofmt`, Go tests/vet and relevant race checks, Python pytest,
`./scripts/test_all.sh`, workflow lint/config tests and actual macOS native builds.
Keep physical records local unless deliberately sanitized for fixtures; redact
identifiers from public artifacts. Distinguish synthetic transcripts, native mock
exports and physical captures in reports. Never intentionally induce a vehicle
fault. Stop transmissions and preserve evidence if tests cause vehicle warnings,
unexpected traffic or other abnormal behavior; do not claim safety merely because
no warning was noticed in one short test.

Before the prolonged physical test, document planned duration, signal set,
conservative budget and stop criteria. Report actual duration/results instead of
inventing universal sampling or reliability thresholds. Missing device access
leaves physical acceptance pending; it does not justify fabricated evidence.

## Acceptance and reporting

Milestone 6 is complete when documented initialization and bounded standard
acquisition work on the physical setup, generic fallback works without enhanced
Alfa assumptions, unavailable signals do not stall or appear as fresh zeroes, and
a retrieved stationary live session/report passes provenance-aware Python analysis.
Capture a manual incident to exercise real pre/post evidence retrieval without
needing the owner's unpredictable fault to occur. Preserve observed disconnect/
reconnect behavior and explicit limitations. Broader road reliability, measured
phone power/storage budgets and automatic session policy are M7 acceptance.

Write `docs/MILESTONE_6.md` and update schema, profile, bridge, development and
physical testing instructions. Include exact supported protocol/decoder surface,
primary references, initialization allowlist, state/timing/queue policies, report
schema/calculations, measured rates, complete tested workload, actual exports,
coverage limitations and handoff evidence for M7/M8.

No road-test automation, full hands-off drive policy, undocumented manufacturer
requests, enhanced Alfa thresholds inferred from mocks, GPS/motion/performance
features, V2 redesign, AI diagnostics, cloud or enterprise platform. Never clear
DTCs, code/configure/calibrate ECUs, actuate or send arbitrary CAN writes.

Report COMPLETE / PARTIALLY COMPLETE / BLOCKED, remaining gates, exact changes,
commands/results and local/native/physical evidence separately; signed build and
artifact provenance; report/reference comparisons; schema compatibility; observed
versus unknown capabilities; lifecycle limitations and deviations. Evidence of
correlation is not proof of causation or a resolved vehicle diagnosis. End:
**Milestone 7 not started.**
