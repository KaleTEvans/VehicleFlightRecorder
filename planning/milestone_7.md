# Codex Handoff — Vehicle Flight Recorder, Milestone 7 Only

## Assignment and prerequisites

Implement **Milestone 7 — controlled road logging** in:

```text
/home/kaletevans/dev/VehicleFlightRecorder
```

This is a future implementation handoff. Writing it does not authorize road
logging, notification permission prompts, new vehicle commands or implementation.
Milestones 5 and 6 must first have accepted physical evidence: a signed build
installed on the actual iPhone, working MX+ streams, controlled standard reads,
stationary acquisition, raw provenance and retrieved/validated live exports.
Inspect their actual reports and artifacts; do not infer acceptance from filenames,
unsigned archives, synthetic transcripts or a successful TestFlight upload.

As of October 29, 2026, M5 code/native regression checks passed, but signing,
installation and physical acceptance remain pending. The user has applied for
Apple Developer enrollment and is awaiting approval. M6 has not started. Recheck
these facts when this assignment is activated. Finish independent code/test work
when physical access is unavailable and report remaining gates honestly.

Inspect `git status` and repository instructions, preserve unrelated changes, and
stay within this milestone. Keep the diagnostic-first MVP ahead of the separately
planned Driver Companion V2. A future cockpit/garage is not a prerequisite for
this milestone's confirmation or navigation.

## Authoritative context

Read the current complete plans and actual implementation:

```text
planning/alfa_flight_recorder_mvp_plan.md
planning/vehicle_health_enterprise_architecture.md
planning/driver_companion_v2_plan.md       # boundary awareness, not authorization
planning/milestone_5.md
planning/milestone_6.md
planning/milestone_8.md                  # boundary awareness only
docs/MILESTONE_4.md
docs/MILESTONE_5.md
docs/MILESTONE_6.md                      # must exist after prerequisite work
docs/BRIDGE_CONTRACT.md
docs/LIVE_TIMING.md
docs/DATA_SCHEMA.md
docs/INCIDENT_MODEL.md
docs/VEHICLE_PROFILE.md
docs/DEVELOPMENT.md
docs/IOS_DEVICE.md
docs/IOS_PREVIEW.md
```

Focus on MVP §§1.2, 6.1.1, 10.1–10.3, 11–15, **15.1**, 20.4, 21 and 22.
Priority: active human instructions, this handoff, MVP plan, enterprise/V2
boundaries. If required reports are absent, identify the prerequisite instead of
inventing implemented contracts. Reconcile conceptual diagrams with actual code.

Preserve approved name/version `Vehicle Flight Recorder` / `0.1.0`, bundle
`com.kaleevans.vehicleflightrecorder`, `com.obdlink`, iOS 17 floor and MFi PPID
`221699-0001` submission metadata. Preserve the versioned Go bridge, accepted
compiler/runtime pins and separate preview/device SDK profiles. Check actual
Apple upload requirements before changing distribution tooling; no moving latest
pins or public release work is required. Simulator remains explicitly mock-only.

## Required work

### 1. Define the controlled-road test configuration

Use the M6 stationary capability report to select a small validated diagnostic
workload. Record the diagnostic questions, available signals/responding ECUs,
source/decoder versions, conservative rate/timeout/backoff budgets and expected
coverage. Standard OBD is shared infrastructure; do not assume a separately
reachable TCM, dual ECMs or manufacturer signals merely because this is an Alfa.

Define the permitted test configuration and graduated sequence: deterministic
fixtures, stationary physical verification, a short controlled drive, then repeated
and representative long-duration operation. Record proposed durations and stop
criteria before each physical stage; report actual elapsed time afterward.
A useful short test does not establish multi-hour reliability. No theoretical
maximum-throughput search or aggressive traffic escalation belongs here.

Configure the app while parked. During motion, logging must not require interacting
with debug controls, permission dialogs, exports or test checklists. Use a passenger
or review results afterward when observations are needed. Preserve the one-tap
manual incident control. Do not intentionally induce a vehicle or ECU fault.

### 2. Implement a Go-owned automatic session policy

Use actual M6 observations to define and test activity evidence, freshness windows,
debounce, stop/end grace, degraded-start behavior and reconnect handling. Record
policy version/configuration and the evidence for chosen thresholds; do not invent
universal timing/RPM values or reuse mock thresholds as vehicle specifications.

Required distinctions:

| Evidence/state | Required behavior |
| --- | --- |
| Scanner available / streams ready | Transport state only; do not create a drive |
| Ignition on, engine off | Preserve relevant connection evidence; no false automatic drive |
| Fresh validated engine-running evidence, stationary | Eligible for recording; motion is not required |
| Engine/module replies without validated running evidence | Running remains unknown; responses alone do not prove it |
| One expected module unavailable | Preserve startup fault/coverage evidence; evaluate an explicit degraded-start policy |
| Module dropout during a drive | Keep recording available evidence and gaps; do not terminate the drive immediately |
| Brief transport loss / reconnect | Preserve drive identity within the validated grace policy and create a new transport generation |
| Engine stop/start or short engine-off interval | Apply the tested grace/debounce policy; no duplicate or premature session closure |
| Sustained validated end condition | Finalize once; preserve remaining incident-window limitations |
| Invalid/stale/missing activity data | Expose uncertainty and apply a bounded documented policy; never substitute a fresh zero |

Engine and transmission reporting are candidate evidence, not a mandatory pair
of standard-OBD responders. Missing TCM coverage must not suppress all recording
or hide a startup fault. If automatic start cannot be validated with available
signals, keep the limitation explicit and retain deliberate recording controls;
do not claim the hands-off acceptance item passed. Verified enhanced reachability
belongs to M8, not an undocumented request added to satisfy M7.

Document state transitions such as waiting-for-activity, candidate, recording,
degraded/gap, end-grace, finalized, interrupted and failed. Go owns policy/session
identity and persistence; Swift reports native lifecycle and presents state.
Serialize competing start/stop/manual commands. Repeated callbacks, multiple views,
reconnects and overlapping lifecycle events must not create concurrent drives.
Transport generation remains independent of the drive ID.

Preserve startup context with a bounded approach compatible with the live schema.
If pre-start samples are included, explicitly define session start time and raw
references; do not insert earlier records into a contract that rejects them.
Alternatively reference retained connection evidence as pre-start context without
mislabeling it as drive telemetry. Record missing initial coverage honestly.

### 3. Keep acquisition independent of presentation

Run the M6 bounded scheduler under real monotonic time, independent of dashboard
refresh or navigation. Preserve stream/capture ownership, request cancellation,
partial-write uncertainty, sleep-aware deadlines and stale-generation rejection.
An old reply must not satisfy a new connection's request. UI refresh must not
create observations or inflate achieved rates.

Separate transport loss from individual ECU/request failures. Record gaps, backoff,
workload changes and restoration. One slow signal must not block all acquisition.
Automatic recovery must not replay stale commands or produce reconnect storms.
Missing execution time during suspension is lost coverage, not data to backfill.

Preserve manual incidents, applicable validated detectors and pre/post windows
(default 60/120 seconds unless the accepted configuration changes). Early drive
end, disconnect or app termination may leave a partial incident; export the reason.
No fabricated history or automatic causal diagnosis is allowed.

### 4. Verify hands-off behavior across actual iOS states

After initial setup, supported normal trips should not require opening the app
or pressing Start each time. Implement only platform behavior supported by current
primary Apple/OBD Solutions documentation and physical evidence. Recheck relevant
ExternalAccessory/background lifecycle rules at implementation time and retain
reference revisions. Declared background modes or notification permissions do not
prove wake, launch or continuous acquisition.

Test these as distinct cases: foreground; background while running; screen locked;
unlock/foreground return; previously launched but suspended; OS-terminated;
user-force-quit; relaunch with interrupted evidence; adapter already connected
before launch; later adapter availability; repeated disconnect/reconnect. Record
actual observation, timing, coverage, required user recovery and evidence for each.
Do not call a force-quit test an OS-termination test. If an OS state cannot be
reproduced reliably, mark it untested rather than relabeling a manual simulation.

Report unsupported wake/launch states explicitly, including unavoidable user
action. Do not attempt private APIs, notification tricks or a background mode
unrelated to accessory operation. Notifications cannot establish extra recording
or wake capability. Preserve evidence when execution ends unexpectedly.

### 5. Automatic recording confirmation — MVP §15.1

This is an approved M7 requirement. It is separate from internal accessory
connection notifications and from any future incident alerting feature.

Use a native notification adapter behind an injectable seam. Go emits a durable
successful automatic-session-start fact with session ID, start origin, time,
vehicle display identity and coverage state. Trigger confirmation only after
session initialization succeeds and the recorder begins accepting actual
telemetry. A connected adapter, attempted start, empty newly created directory or
failed start must never produce a recording-success confirmation. Manual starts
must not accidentally enter the automatic-start notification path.

Use the selected vehicle's actual display name and a generic fallback; no garage
feature or invented identification is required. Suggested copy follows the plan:
“Drive recording started”, vehicle display name and “Recording automatically”.
For a degraded start, communicate limited coverage without implying full support.
Use the existing dashboard/session screens for MVP routing; “cockpit” wording in
the roadmap does not authorize the V2 redesign.

Delivery requirements:

- Explain the optional confirmation during initial setup; request authorization
  in context. Provide a persisted enable/disable preference and a route to system
  settings when appropriate. Do not repeatedly prompt denied users or tie app
  startup/recording eligibility to permission approval.
- Default to quiet delivery without sound when authorized. Avoid badge updates,
  urgent/time-sensitive/critical escalation or an incident-alert feature unless
  separately requested. Respect system authorization and Focus settings.
- When visible, show a brief in-app confirmation and suppress a duplicate system
  presentation. In a physically supported background state, request a local
  notification. Handle scene changes during submission without generating both.
- Persist session-level deduplication across reconnect, engine stop/start,
  dashboard navigation and relaunch. Use a deterministic request identity tied
  to the drive, not transport generation or a new random ID on every callback.
- Define crash/uncertain-submission behavior explicitly. Do not assume native
  submission and Go persistence form an atomic transaction. Favor the specified
  at-most-one confirmation over retrying an uncertain submission on relaunch;
  a documented best-effort missed confirmation must never stop recording. A
  stable native request ID alone is not proof of deduplication after delivery.
- Record request/submission/failure, selected presentation path and applicable
  suppression reason for debugging. Distinguish submitted from displayed/seen;
  do not claim user visibility or guaranteed delivery from API success.
- Permission denial, disabled preference, notification API errors and missing
  presentation must leave recording running with visible in-app recording state.
  Consume/suppress stale start facts so later permission changes do not send a
  backlog of confirmations for ended sessions.

On tap, route by the original session ID. An active session opens its dashboard;
an ended session opens its summary. Recheck state at navigation time, including
cold launch and a session ending between submission and tap. Handle missing or
unreadable sessions visibly rather than opening an unrelated current drive or
creating a new one. Tapping is not a start command. Retain the application delegate
and notification-center delegate lifetimes needed for foreground and cold-launch
callbacks without leaking duplicate observers/delegates.

Verify with fakes first, then native/physical tests where relevant: successful
start; startup failure and no accepted telemetry; manual start; degraded start;
foreground/background transition; authorized/denied/not-determined settings;
disabled preference; notification errors; duplicate start facts; reconnect;
relaunch/uncertain submission; tap while active, after end, after cold launch and
for missing evidence. No system permission dialog should interrupt a road test.

### 6. Interruptions, storage and honest recovery

Inspect the actual M6 live schemas/readers before implementing recovery. Preserve
original raw logs, observation ordering, transaction references and checksums.
Do not silently trim a truncated final row, finalize a failed session as healthy,
or repair the sole copy of canonical evidence. Any recovered view/derivative must
retain original files, gaps and clear interrupted/recovered status.

Define relaunch policy for session ownership: resume the same logical drive only
when the persisted evidence/identity and current activity support it under a
versioned policy; otherwise preserve the interrupted session and start a distinct
new drive when eligible. Record the boundary explicitly. Do not infer continuous
coverage across termination. Keep notification deduplication consistent with the
actual drive IDs chosen by this policy.

Measure storage growth, raw volume, queue/heap behavior and sustained file I/O.
Choose/document battery and storage budgets from representative baseline
measurements before final acceptance. Include test duration, device/OS, workload,
foreground/locked time, charging state, temperature context when observable,
start/end battery levels and measurement limitations. A charging-only test cannot
establish normal battery drain. Report totals and per-hour rates with denominators.

Exercise low storage/write failure and interrupted output with controlled fixtures.
Define a user-visible storage-limit/failure response. Never silently discard raw
bytes, overwrite earlier drives or add destructive retention without an explicit
policy. Keep filesystem limits/resource bounds consistent across Go, Swift export
and Python; normal finalized exports must not be presented as valid if corrupt.

### 7. Road-workload capability and reference report

Extend the versioned M6 report with actual combined road workload, configuration
changes, requested/achieved observation rates per signal/responding ECU, counts,
response latency, gaps, stale time, error populations and available coverage.
Record total versus active acquisition intervals, pauses, disconnects and actual
drive duration. Include both individual stationary characterization and combined
road results; do not extrapolate one-signal performance to all signals together.

Relate coverage to the selected diagnostic questions and incident window needs.
If achieved rates are inadequate, reduce workload or narrow supported findings
with evidence. Do not claim ECU sensor refresh rate from changing values alone.
Maintain advertised/observed/unsupported/unavailable/unknown distinctions and
explicit generic fallback without enhanced Alfa assumptions.

Compare suitable values against MultiECUScan or another trusted source under
recorded comparable conditions and units. Do not run competing diagnostic clients
simultaneously unless that configuration is supported and deliberately tested.
Sequential references require timing/operating-state limitations. Preserve
unresolved discrepancies; no reference run is proof that every signal is correct.

### 8. Retrieved real drives and reproducible analysis

Retrieve finalized exports from the actual phone, record checksums/build/commit,
then run Python validate, summarize and incident reconstruction using actual IDs.
Include a controlled manual incident in a safe setup so the pre/post pipeline
can be verified without waiting for the intermittent fault. Confirm evidence
navigation and native share presentation; separately prove actual file retrieval.

Retain unmodified private physical captures with configuration and provenance.
Create sanitized fixtures only when needed, documenting transformations and
preserving relative timing/reference consistency. Public CI artifacts must not
expose VINs, accessory serials or other personal vehicle identifiers. Label fixture,
native mock and physical evidence separately. No cloud ingestion is required.

## Required verification and acceptance matrix

| Layer/case | Required evidence |
| --- | --- |
| Automatic policy | Deterministic activity/freshness/debounce/grace/degraded-start fixtures; KOEO, stationary running, stop/start, dropout and duplicate callbacks |
| Timing/transport | Continuous-clock deadlines, suspended/resumed writes, partial send, stale replies, bounded queues, backoff and no replay |
| Recording confirmation | §15.1 cases above, durable per-session deduplication, foreground/system choice, permission/preference/error independence and active/ended/missing tap routing |
| Persistence/recovery | Startup/finalization I/O errors, interruption/truncated rows, preserved originals, explicit gaps/status and no duplicate session adoption |
| Native regression | Source-built framework and app, hosted unit/UI tests, mock recording/evidence jump/export and generic live-schema regression |
| Physical lifecycle | Every foreground/background/locked/suspended/termination/wake case recorded separately with signed build/device/OS/adapter provenance |
| Controlled drives | Short, repeated and representative long-duration runs with planned/actual duration and workload, no normal-operation missing/corrupt sessions |
| Fault distinction | Controlled transport/module-loss fixtures and safe physical disconnect checks; no deliberately induced ECU/vehicle fault |
| Quality/reference | Combined workload report, measured rates/latencies/gaps, comparison setup and unresolved differences |
| Battery/storage | Representative measurement context, raw/log growth and per-hour budgets, limitations and observed threshold compliance |
| Phone exports | Retrieved original files/checksums; Python validate/summarize/reconstruct and manual incident pre/post evidence |

Run Go formatting/tests/vet and relevant race tests, Python pytest, script/config
checks, actionlint and actual native CI. Add meaningful clock/notification/transport
injection seams; do not weaken validators or skip accepted UI regressions to pass.
Use Appetize for mock behavior only. Record notification submission behavior and
physical background observations separately from simulated delegate callbacks.

If warnings, unexpected network activity or other abnormal behavior appear during
physical testing, stop transmissions and preserve evidence. “No visible warning”
is one observation, not universal proof of network safety. Document the criterion
for acceptable disturbance and remaining uncertainty with the measured workload.

## Completion report and scope boundary

Write `docs/MILESTONE_7.md`, a detailed reliability matrix and notification behavior
report; update schema/bridge/lifecycle, physical testing, export/analysis and setup
instructions where changed. Record policy/configuration versions, state diagram,
permissions/preferences, durable notification identities, actual native/physical
results, test durations, budgets, known wake limits, artifacts and private evidence
locations. Keep capability reports machine-readable and reproducible.

M7 is complete when repeated physical logs and retrieved exports demonstrate the
validated workload, supported hands-off lifecycle, honest interruption handling,
required recording confirmations and measured diagnostic coverage/battery/storage
budgets. A documented inherently unsupported iOS state is an explicit product
limitation; an untested required supported state or unresolved recording defect
remains an acceptance gap. Do not claim universal wake or a diagnosed vehicle fault.

No undocumented enhanced requests, M8 pack expansion, GPS/motion/performance UI,
garage/V2 cockpit redesign, accounts, cloud or AI. No DTC clearing, ECU coding,
calibration/configuration changes, actuation or arbitrary CAN writes. Reporting
state and diagnostic reads do not authorize general control of the vehicle.

Report COMPLETE / PARTIALLY COMPLETE / BLOCKED with exact remaining gates and
local/native/physical evidence separated. Distinguish implemented policy from
measured platform capability and notification submission from user visibility.
Do all independent work before requesting user-only account/device actions; never
ask for private credentials in chat or re-request existing authorization. End:
**Milestone 8 not started.**
