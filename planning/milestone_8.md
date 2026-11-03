# Codex Handoff — Vehicle Flight Recorder, Milestone 8 Only

## Assignment and prerequisites

Implement **Milestone 8 — Giulia-specific drivetrain expansion** in:

```text
/home/kaletevans/dev/VehicleFlightRecorder
```

This is a future implementation handoff. Writing the document does not authorize
implementation, new diagnostic transmissions or a road test. Begin only after
M5/M6 physical integration and M7 controlled-road logging have accepted reports,
retrieved live exports and measured coverage for the actual signed iPhone/MX+
configuration. Do not infer completed prerequisites from a simulator or compile.
Inspect `git status`, repository instructions and current sources; preserve
unrelated changes and existing private evidence.

The purpose is to improve evidence for the owner's intermittent drivetrain fault
on the **2017 Alfa Romeo Giulia Quadrifoglio**, 2.9L twin-turbo V6 / ZF 8-speed.
Generic standard discovery and recording remain shared infrastructure. Enhanced
Alfa requests, decoders, module mappings, applicability and rules belong in the
vehicle pack. Do not move this diagnostic work behind V2 trip/performance features.

As of this handoff's preparation on October 29, 2026, M5 automated implementation
checks passed but physical acceptance remains pending; the user's Apple Developer
enrollment is awaiting approval. M6–M8 have not started. Recheck actual status on
activation and identify missing gates without manufacturing physical evidence.

## Authoritative context

Read the complete current plans and accepted implementation reports:

```text
planning/alfa_flight_recorder_mvp_plan.md
planning/vehicle_health_enterprise_architecture.md
planning/driver_companion_v2_plan.md       # later-product boundary only
planning/milestone_6.md
planning/milestone_7.md
docs/MILESTONE_5.md
docs/MILESTONE_6.md                      # expected after prerequisite work
docs/MILESTONE_7.md                      # expected after prerequisite work
docs/VEHICLE_PROFILE.md
docs/DATA_SCHEMA.md
docs/INCIDENT_MODEL.md
docs/BRIDGE_CONTRACT.md
docs/LIVE_TIMING.md
docs/DEVELOPMENT.md
docs/IOS_DEVICE.md
docs/IOS_PREVIEW.md
```

Focus on MVP §§1.2, 6.1.1, 8–12, 15–17, 20–22 and 24, plus enterprise pack
provenance, validation and support-depth requirements. Priority: active human
instructions, this handoff, MVP plan, enterprise/V2 boundaries. Inspect actual
Go/Python/Swift contracts before changing schemas; illustrative plan names do not
supersede implemented package or signal identities.

Preserve approved app/version `Vehicle Flight Recorder` / `0.1.0`, bundle
`com.kaleevans.vehicleflightrecorder`, `com.obdlink`, iOS 17 deployment floor and
PPID `221699-0001` submission metadata. Preserve accepted compiler/runtime pins,
preview/device profiles, mock-only Simulator and the signed physical-device path.
No public App Store release is required.

## Required work

### 1. Prioritize diagnostic questions and attainable evidence

Review the owner's reported symptoms and the M6/M7 real evidence before selecting
signals. Summarize what is observed, what remains unknown and which competing
explanations need discrimination. Do not invent new symptoms or presume that an
electrical event, TCM fault or mechanical anomaly has already been established.
Use available project records first; ask only for missing diagnostic context that
materially changes the selection while continuing independent implementation.

Produce a small ranked evidence plan, for example:

| Question | Candidate evidence, subject to verification | Required limitation |
| --- | --- | --- |
| Did an electrical change precede communication loss? | Adapter-origin voltage, verified ECU voltage if available, raw timing, request/response and transport continuity | Adapter voltage is not automatically battery-terminal or ECU supply voltage |
| Was a communication interruption transport-wide or module-specific? | Native connection state, responding ECU identities, expected-request outcomes, validated ECM/TCM latency/availability | Missing/unimplemented polling is not a failed ECU |
| Did drivetrain operating state change around the mark? | Verified engine/transmission state, selected load/temperature/gear/boost values where useful and supported | Do not substitute standard generic data for an unavailable enhanced signal |
| What DTC state was actually observed? | Bounded documented read-only snapshots with module/status/raw provenance | Absence of a snapshot or a code is not proof that no fault occurred |

The table identifies questions, not guaranteed supported signals. Select an
explicit initial validated scope from documented, reachable definitions. Prefer
high-value evidence over an exhaustive proprietary catalog. Record unsupported,
unverified and blocked candidates and their reasons instead of guessing requests.

### 2. Obtain and inventory verified definitions

Use applicable primary OEM/licensed diagnostic documentation and official adapter/
protocol references. Record document/version/section, ownership/license, acquisition
method and permitted fixture/publication use. A trusted reference tool provides
comparison evidence; its observed values alone do not establish request bytes,
decoder formulas or a redistribution license.

For each candidate definition record:

- Canonical signal ID, display meaning, units and source module/bus.
- Exact diagnostic request, addressing/response identity, service/identifier,
  framing and documented prerequisites once verified; never fill these with guesses.
- Byte length/layout, endianness, signedness, scale/offset, validity/status bits,
  reserved/unavailable encodings and physical meaning.
- Model/year/engine/transmission/module/software applicability and identity evidence.
- Required adapter/network routing and verified module reachability.
- Decoder/catalog/pack version, provenance/license and validation level.
- Poll priority/rate/deadline budget and diagnostic question served.
- Known reference comparisons, discrepancies and unsupported configurations.

Do not brute-force proprietary identifiers, sweep ECU addresses, infer a decoder
from changing bytes alone, bypass a gateway or use undocumented commands copied
from community snippets. Do not assume adapter MS-CAN/SW-CAN capabilities establish
Giulia network reachability. ELMA5/ELMA6 and ABS/TVM/EPS/airbag expansion remain
outside this initial drivetrain scope. Missing verified definitions leave the
relevant feature pending; they do not prevent completing unrelated pack tooling,
fixtures, validators and generic regression work.

### 3. Extend the pack format and applicability safely

Inspect the existing pack loader. The original shipped pack and thresholds were
explicitly synthetic; do not relabel them production-valid or activate them on a
live vehicle. Reuse any real-format work accepted in M6 and introduce only the
versioned data structures needed for verified enhanced definitions.

Keep vehicle-specific request definitions, decoders/mappings, module roles,
applicability, schedules, rule parameters and provenance in pack data. Generic
Go owns reusable protocol/decoder primitives, validation, scheduling and recorder
behavior. Swift remains presentation/native transport; Python remains validation,
reconstruction and analysis. No Alfa-name condition in the generic recorder or
vehicle-specific decoder duplicated in Swift.

Select packs using adequate actual vehicle/module identity evidence. A user-chosen
profile may be a documented test configuration but is not proof of ECU firmware
compatibility. Refuse or mark unverified mismatches explicitly. Distinguish:
unknown/not checked, advertised/documented, actually observed/decoded, unsupported,
and temporarily unavailable. Version capability caches against relevant vehicle,
module/software, adapter/protocol, catalog and pack identities. A reconnect or
changed configuration must not retain unsupported stale assumptions.

Test absence, rejection and disablement of the enhanced pack. Generic standard
operation must continue without inheriting Giulia thresholds or expected module
names. An invalid pack must not leave half-applied enhanced requests running;
reject it before transmission, surface the failure and use only an explicitly
reported valid generic configuration when available. Do not silently fall back to
mock values or switch an active drive's meaning without recording that transition.

### 4. Add a narrow, read-only enhanced command/decoder path

Extend the M6 command allowlist and transaction machinery using only verified
requests. Pack data is not an unrestricted arbitrary-byte terminal: validate
service/operation, destination, response ownership, lengths, applicability and
allowed prerequisites before native transmission. Do not treat any request in an
external pack as safe merely because the pack calls it a read.

No DTC clearing, ECU coding/configuration/calibration, security bypass, actuation,
firmware updates or arbitrary CAN writes. If a definition needs undocumented or
unapproved routing/session/security changes, mark that definition blocked and
preserve the reason. Resolve documented prerequisite effects and scope before
implementing them; never experiment against a moving vehicle.

For verified ECM1, ECM2 and TCM access, retain actual responding identities and
pack-owned role mappings. Do not create synthetic module identities for absent
responders or assume both ECMs must expose every candidate value. Preserve multiple
responses separately; no arbitrary first-response selection or averaging.

Cover positive/negative/status replies, unknown identifiers, no-data/timeout,
truncated/oversized replies, malformed lengths, split/coalesced transport chunks,
multiple modules, late/stale-generation responses and cancellation. Distinguish
transport failure, diagnostic rejection, unavailable value and decoder failure.
Preserve valid zero, unavailable sentinels, pressure conventions, temperature
units and status bits. Reference requested-versus-actual comparisons only when
units, meaning and time alignment are established; do not silently compare gauge
pressure with absolute pressure or values from different operating conditions.

### 5. Module heartbeat/latency and bounded workload

Prefer availability evidence from already scheduled verified reads over adding
unnecessary heartbeat traffic. Define what an expected request and a valid module
response mean for each module/configuration, including minimum request coverage,
freshness and timeout/rearm rules. If a dedicated heartbeat is necessary, it must
be a documented permitted read within the same request budget.

Measure send/response timing with the accepted monotonic/transaction contract.
A missing observation is not automatically a module timeout. Suppress transport-
wide loss being counted as independent failures of every module, but retain the
actual coverage gap and uncertain outcomes. Do not erase a genuine ECU dropout
by removing that module from the expected set after it fails.

Integrate enhanced requests into the bounded, prioritized M6/M7 scheduler.
Characterize selected signals individually, then together with the complete
standard/enhanced workload. Protect critical electrical/network evidence from
starvation. Retry/backoff conservatively; slow or unavailable requests cannot
stall recording or create an unbounded queue. No stale replay after reconnect.

Record workload/configuration changes and their effect on achieved rates,
latencies, gaps, battery and storage. Revisit M7 budgets if expansion changes them;
a passed generic-only road run does not prove the expanded workload is reliable.
Requested rates and adapter specifications are not achieved per-signal rates.
ECU internal sensor refresh rate remains unknown without independent evidence.

### 6. Bounded DTC snapshots

Implement only documented read-only DTC queries supported by the validated
configuration. Preserve responding module, retrieval time, transaction/raw
references, exact code/status bytes and decoder/catalog provenance. Distinguish
stored/pending/permanent or OEM status meanings where the source defines them;
retain unknown status rather than assigning a guessed category or description.
Do not claim this milestone guarantees access to all drivetrain DTC stores.

Define a low-impact snapshot policy using the validated workload: appropriate
session boundaries, a parked developer action, or a bounded incident-associated
request where documentation and physical tests support it. Coalesce duplicate
requests and bound their cost. Do not add high-rate DTC polling or an unvalidated
post-incident request burst. If retrieval must wait until parked or reconnect,
record that delay and avoid describing it as a snapshot at trigger time.

Keep snapshot facts separate from differences/inferences. “Newly observed” can
mean first seen by this app; it does not prove exact ECU fault-onset time. Compare
only compatible, sufficiently complete snapshots. A failed/partial read must not
create mass removals; disappearance of a code is not evidence the app cleared it
or the fault is resolved. The app must never clear codes automatically or on request
through this milestone's UI/API.

### 7. Live incident correlation and defensible findings

Use verified enhanced observations, DTC snapshots and actual transport/module
health in the existing incident pipeline. Activate live rules only when their
applicability, units, thresholds, freshness and module expectations are justified
and recorded. Mock voltage/timeout/boost rules remain synthetic until separately
validated; the original scenario catalog is regression evidence, not OEM guidance.

Keep manual marking available even when automatic rule coverage is incomplete.
Preserve pre/post windows, overlap/merge behavior, exact ordered references,
raw transactions and coverage gaps. If the diagnostic workload cannot retain the
required window, expose the deficit rather than filling it with retained display
values or interpolated measurements presented as observations.

In Python reconstruction and readable findings distinguish:

1. Actual observations and their source/timing/quality.
2. Ordered/co-occurring events and measured intervals.
3. Diagnostic hypotheses, supporting and competing evidence, and uncertainty.
4. Missing observations or coverage that prevent a conclusion.

A voltage change preceding a timeout is temporal evidence, not established cause.
A code's presence is not automatically the initiating failure. Respect timestamp
uncertainty and transaction intervals before asserting event order. Align derived
views without rewriting original evidence or creating fabricated samples.
No LLM or automatic repair/replacement recommendation is required.

Exercise capture/reconstruction with controlled fixtures for positive, negative,
transport-loss, stale-data, overlapping-trigger and ambiguous-order cases. Also
use a safely performed physical manual mark to verify real enhanced provenance.
A real intermittent fault is valuable but unpredictable; its occurrence is not a
condition for finishing validated infrastructure. Conversely, do not claim the
owner's issue has been diagnosed or resolved without actual supporting findings.

### 8. Extend capability reporting and schema/export compatibility

Extend the versioned M6/M7 capability/evidence-quality report with enhanced pack
identity, applicability, verified request/decoder sources, validation levels,
module roles, DTC coverage and rule applicability. Include the complete tested
standard/enhanced signal set, vehicle/adapter/software configuration, operating
state/duration and individual-versus-combined workload evidence.

Report achieved successful observations per signal/ECU, request/reply/decode
populations, latency distributions, unavailable/stale/gap intervals and failures.
Reference comparison results must state the tool/version, units, comparable
operating state, timing limitations and unresolved discrepancies. Avoid competing
diagnostic clients unless their concurrent use is specifically supported/tested.
Do not export licensed definitions or private vehicle identifiers in public
fixtures/artifacts without an appropriate publication basis.

Version incompatible schema changes explicitly. Coordinate Go models/writer and
pack loading, raw/transaction references, Python validation/reconstruction, Swift
models/coverage UI, archive packaging and native export-validation scripts. Test
legacy mock exports and accepted generic-live exports as well as enhanced data.
Unknown versions, foreign references, mismatched packs, truncated logs and invalid
DTC status data must still fail clearly. Do not relax validators to hide missing
raw provenance or label incomplete snapshots/incident capture as complete.

Retrieve an actual enhanced-session export from the physical phone, record its
checksum and provenance, and run Python validate/summarize/incident reconstruction
using actual IDs. Preserve original private captures; label sanitized/synthetic
fixtures and transformations separately. No cloud upload backend is required.

## Required verification and staged physical acceptance

| Layer/case | Required evidence |
| --- | --- |
| Provenance/applicability | Verified source/license inventory; matching and mismatched vehicle/module/software; unverified definition disabled |
| Pack/schema | Missing/invalid/unsupported pack, all-or-nothing admission, standard-only fallback, version/cache invalidation and old exports |
| Decoder fixtures | Exact request/reply, lengths, endianness/sign/scale, sentinels/status/zeroes, malformed/negative/multiple replies and documented units |
| Command boundary | Only verified read allowlist entries; reject arbitrary operations, DTC clear/write/security bypass and stale queued requests |
| Scheduler/module health | Bounded combined workload, timeout fairness, backoff, transport versus module failure, reconnect cancellation and measured latency |
| DTC handling | Known/unknown status, partial/failed snapshots, newly observed versus onset, deduplication and no false removals/clearing |
| Incident evidence | Positive/negative/ambiguous/stale/gap fixtures, overlap/windows, exact raw references and observation/correlation/hypothesis separation |
| Native regression | Actual framework/app, hosted unit/UI tests, mock and generic-live export/evidence workflows, recording confirmation remains correct |
| Physical stationary | Signed build/configuration, verified modules/signals, conservative reads and reference comparisons before road use |
| Physical combined workload | Controlled drive(s), adequate diagnostic coverage, no unacceptable disturbance and rechecked lifecycle/battery/storage budgets |
| Phone export | Retrieved enhanced evidence, checksums and Python validation/summary/reconstruction; private real records separate from public fixtures |

Run Go formatting/tests/vet and relevant race checks, Python pytest, scripts/
workflow lint and real native CI. Build transcript and pack tests before enabling
physical enhanced reads. Plan stationary test duration/budgets/stop criteria before
execution, then repeat the accepted controlled-road sequence only for definitions
that pass. Stop transmissions and preserve evidence if unexpected vehicle warnings,
traffic or other abnormal behavior appear; do not intentionally induce faults.

Physical acceptance must name each actually reached module and verified signal,
reference conditions, software/configuration and unresolved limitations. Do not
claim all Giulia variants, all three drivetrain modules or every candidate signal
were validated from one configuration. If no usable enhanced definition can be
verified, tooling can be complete while the milestone remains partially complete;
generic-only fallback is not acceptance of manufacturer expansion.

## Completion report and post-MVP decision gate

Write `docs/MILESTONE_8.md`; update vehicle profile/pack documentation, schema,
incident/rule contracts, bridge/export/analysis and physical-test instructions.
Include the ranked diagnostic plan, definition inventory/provenance, accepted
support matrix, pack/decoder/rule versions, exact command allowlist, measured
workload/report, real export locations/checksums, native/physical results and
remaining gaps. Preserve prior milestone evidence instead of retroactively
claiming expanded coverage for earlier standard-only tests.

M8 is complete when the agreed diagnostic-priority enhanced scope has verified
pack-owned definitions, actual physical acquisition/reference evidence, bounded
combined-workload behavior and retrieved provenance-aware incident exports,
while generic fallback and prior recording/notification behavior remain valid.
Missing desired definitions and unsupported modules stay explicit. If a proposed
scope reduction removes a central diagnostic objective, obtain a concrete scope
decision rather than declaring success from whatever happened to be available.

This concludes the planned diagnostic MVP milestone sequence, not proof of a
resolved fault or authorization for the next product. Review achieved capabilities,
coverage deficiencies, real findings and support needs before choosing follow-up
Alfa diagnostics or separately starting the reviewed Driver Companion V2 plan.
The enterprise roadmap remains directional; do not invent a Milestone 9 or begin
GPS/motion, garage, performance timing, cockpit redesign, cloud, accounts or AI.

Report COMPLETE / PARTIALLY COMPLETE / BLOCKED with exact remaining gates and
local/native/physical evidence separately; supported versus unknown features,
actual findings versus hypotheses, references/licensing, schema compatibility,
measured rates/budgets, signed build/artifact provenance and deviations. Complete
independent work before requesting user-only credentials/device observations;
never request private keys or tokens in chat. End explicitly:
**Diagnostic MVP acceptance status: [state with remaining items]. Driver Companion V2 not started.**
