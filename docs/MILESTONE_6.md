# Milestone 6 — adapter initialization and standard OBD

Status: PARTIALLY COMPLETE — updated November 6, 2026. Implementation/native fixture
acceptance and signed TestFlight delivery passed; phone installation and physical
stationary acceptance are tracked
below. User authorized closing M5 and starting
M6. M5 PR #3 merged as `94c5949`; implementation branch is
`codex/milestone-6-live-acquisition`. No successful M6 physical acquisition is claimed; the reported device issue is
recorded below.

## Implemented scope

- Keep M5 discovery/recovery and Simulator mock UI; opening an accessory sends no
  application commands. Only an explicit parked-session Start enables live setup.
- Go owns fixed documented runtime setup, per-generation initialization and the
  read allowlist. Initial physical profile is ISO 15765-4 CAN, 11-bit, 500 kbps:
  STP 33, headers/PCI on, no DLC, bounded response timeout. No automatic protocol
  brute force, reset/reboot, NVM protocol write, ECU write or calibration.
- Initial verified catalog: mode 01 block 00, RPM 0C, coolant 05, STVR adapter
  input voltage. Other decoders remain unimplemented, not ECU-unsupported.
- Reuse the offline framing/scheduler as explicit live mode; leave version-1
  synthetic reports and legacy mock session validation intact. Native owner-loop
  monotonic time drives acquisition independently of SwiftUI refresh.
- Schema-2 live sessions reuse the canonical recorder and manual incident engine;
  live automatic fault thresholds remain disabled. Add exact raw ledger,
  transaction/generation references and a version-2 capability report. Mock
  sessions remain schema 1, five files; live exports explicitly add capability.json.
- Use a single fixed monotonic/wall anchor, receipt timestamps and unknown ECU
  measurement time. Log clock changes, suspension/gaps and physical disconnects.
- One in-flight request, conservative intervals and bounded resources/duration;
  no stale request replay. Reconnect preserves session history, clears capability
  scope and repeats documented setup before further measurements.
- Physical developer controls: parked acknowledgement, operating-state/workload,
  Start, Stop/save, manual mark, current coverage/age and history/evidence/export.
  No automatic drive start, notification prompt or V2 redesign.

Primary protocol references reviewed November 3, 2026:
[OBDLink FRPM E](https://www.scantool.net/scantool/downloads/678/obdlink_frpm_e.pdf),
sections 7.2, 8.5–8.6, pp. 12–19 and 32–34;
[ELM327DSJ](https://www.elmelectronics.com/wp-content/uploads/2016/07/ELM327DS.pdf),
pp. 31–32, 41–44 and 47. STP33 selects the runtime preset; ATSP would save to NVM
and is excluded. ATD restores runtime defaults without reboot. STPR must report 33.
The exact allowlist, schemas, budgets and tests are recorded below.

## Remaining acceptance gates

Successful stationary individual/combined acquisition and reference comparison,
remaining phone-export checks and manual incident reconstruction. The two requested
zero-sample phone recordings were retrieved and validated on November 6, as
recorded below; their discovery failure remains unresolved. Automated/native evidence and signed upload have passed as
recorded below. Do not label scripted/native fixtures as physical measurements.
M5 identity/reconnect acceptance does not establish ECU polling or road reliability.


## Implemented acquisition path

`core/obd/live.go` owns initialization version `obdlink-stp33-runtime-1`:
`STPC`, `ATD`, `ATE0`, `ATL0`, `ATM0`, `ATS1`, `ATH1`, `ATD0`, `ATCAF1`,
`ATCFC1`, `ATR1`, `ATAT0`, `STP33`, `STPTO500`, `STPR`, `STI` (CR terminated).
Every step needs one valid prompt-terminated acknowledgement; STPR must return
33, and STI must return a recognized STN identity. Setup is logged per fresh
connection. `ATD` restores runtime defaults; `ATM0` disables automatic parameter
memory. No ATSP, reset/reboot, arbitrary send or enhanced vehicle requests exist
in the physical live path.

Measurement allowlist: `0100` supported block 00, `010C` RPM, `0105` coolant,
`STVR` adapter input voltage. Decoders retain 11-bit CAN header, PCI and individual
responders. No VIN, DTC, speed, throttle, additional support blocks or proprietary
request is implemented. A missing decoder is not labeled ECU-unsupported.

`LiveRecorder` uses the existing scheduler in explicit live mode and the existing
canonical recorder in manual-only mode. Fixed requested intervals are RPM 2 s,
coolant/voltage 5 s, support 60 s. One request is in flight; minimum subsequent
admission is 500 ms after completion, timeout 2 s, bounded late-response drain 1 s.
Failure backoff is inherited from the tested scheduler. Initialization failures
require a fresh connection, with at most three resynchronizations per session;
resource/evidence failures stop admission. No pending bytes replay on reconnect.
Per-generation capability scopes are reset and setup/discovery repeat.

The physical session is explicit, generic and independent of the Alfa mock pack.
It records operating state, workload and optional reference notes. Unknown vehicle
identity and sensor refresh rate stay unknown. Manual incidents use retained
full-session logs, a 60 s pre-window and 120 s post-window, with silence and
transport gaps retained. See [schema migration](DATA_SCHEMA.md) and
[incident semantics](INCIDENT_MODEL.md). No synthetic threshold runs on live data.

The 100 ms native owner timer runs on the dedicated ExternalAccessory executor;
Go controls admission using `mach_continuous_time`. UI snapshots never poll.
Opening streams is separately recorded from creating an EASession. Native errors,
scene phases, observed owner gaps and wall changes become canonical events.
A timer is not a guarantee of execution while iOS suspends/terminates the app.
Physical controls require a parked acknowledgement and show age, quality,
capability scope, manual marking and Stop/save. Saved sessions use pack-free
history/evidence and ZIP packaging. Simulator root remains the accepted mock UI.

## Verification so far

- Full Go package tests passed after live facade/schema integration.
- Independent Python replay passed all five actual Go transcript exports: healthy,
  reconnect, malformed partial response, advertised unsupported, and late timeout.
- 18 live Python tests passed, including altered raw bytes, missing rows, invented
  values/measurement times, foreign generation/references, schema/configuration
  corruption, and explicit incomplete-prefix inspection.
- Legacy schema diagnostics retained. Full portable, native hosted/UI, race and
  actual native-export checks passed as recorded below. Fixture timings are
  synthetic, not measured adapter performance.

## Stationary physical acceptance plan (pending)

Use the single [M6 parked test-visit checklist](../planning/milestone_6_test_session.md)
under the [shared physical test-session workflow](../planning/physical_test_sessions.md).
It supersedes the earlier isolated 45-second test request and per-case review
cadence. Target one 20–30 minute parked visit, with extra time for any prepared
reference comparison. Short checks, combined manual capture and reconnect run
back-to-back when their written visible gates pass; each uses a distinct recording
where state/workload must be preserved. No chat approval or per-case upload is
needed between those listed parked checks.

Saved history, evidence navigation, individual ZIP exports and one evidence
handoff follow indoors. Agent raw/provenance/reference analysis still determines
final acceptance; failed prerequisites stop dependent tests and actual defects
may need a targeted follow-up. M6 remains partially complete. Road logging,
automatic start/stop, notifications and full lifecycle qualification remain M7.

## Accepted automated evidence — November 3, 2026

Tested application source: `06c5e207ae82aae259a9720271fc5f08df359511`.
Draft implementation PR: [#5](https://github.com/KaleTEvans/VehicleFlightRecorder/pull/5).
M5 remains accepted and merged as `94c5949`; this PR is not merged and M6 is not
physically accepted.

| Evidence | Result |
| --- | --- |
| Go tests/vet and Python | Portable CI passed on the tested source; 116 Python tests, including 18 live replay/corruption cases |
| Build/script checks | 29 script tests, Go formatting, diff whitespace and actionlint passed |
| Race checks | acquisition, mobile, recorder and storage checks passed during implementation |
| [Native run 41703813858](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/41703813858) | 26 hosted tests + 2 UI tests, zero failures; source-built device/simulator framework and both device SDK compile profiles passed |
| Native exported evidence | Healthy/manual mock, voltage/TCM mock and live-schema fixture ZIPs all independently validated and reconstructed in Python |
| WSL replay | Actual native ZIP bytes downloaded and revalidated; live fixture contains 267 decoded observations and one captured manual incident, with origin `native-fixture` |
| Signed delivery | [Run 41705740118](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/41705740118) passed signing/archive/export and `UPLOAD SUCCEEDED with no errors`; exact tested source above, build `0.1.0 (7.1)` |
| Physical M6 measurements/reference/reconnect | Pending; no stationary ECU data or no-warning result claimed |

The native acceptance artifact is
[13264423817](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/41703813858/artifacts/13264423817),
SHA-256 `d6a28d6103d726ca5a862d17d301f4403a8cf64200991a55c1c32b3a7ce801ff`.
Local retrieved evidence is under `build/milestone-6-native/06c5e20/`; independent
replay is under `build/milestone-6-native/06c5e20-replay/`. These are synthetic
native fixtures, not phone/car evidence, and are ignored generated artifacts.

Native regressions verify empty initial snapshots, Go command admission without
UI refresh, no command before streams-ready, Stop cancellation, subsequent M5
transport-only operation, scene evidence, compatibility rejection before storage,
legacy archive resource preflight and actual live ZIP packaging. Swift uses an
explicit Base64 command envelope to avoid nullable NSData/NSError idle ambiguity.
Reconstructed derivatives are now written outside canonical evidence directories;
the artifact replay verifies the original six-file live directory remains valid.
This final tooling/documentation adjustment does not change the tested app source.

Live runtime failures remain visible and preserve incomplete evidence. Mock
thresholds, enhanced Alfa definitions, automatic trip policy, notifications,
road logging and universal background wake claims are outside this implementation.


## Signed beta delivery — November 3, 2026

Build `0.1.0 (7.1)` uploaded successfully at **19:44:51 UTC / 1:44:51 p.m. CST**.
Apple delivery ID: `0209fc45-1ede-4cc2-b616-25565c2be7cd`. After the initial
PROCESSING state, read-only App Store Connect inspection confirmed build VALID,
`usesNonExemptEncryption: false`, and internal state IN_BETA_TESTING. Apple lists
the processed build at 19:45:34 UTC / 1:45:34 p.m. CST. Build 7.1 is available
for internal TestFlight installation. Installation and physical acceptance remain
separate gates.
The beta is built from the exact tested application source above; subsequent
report/reconstruction-output changes do not alter that binary. Appetize remains
mock-only and was not updated by these validation runs.

The independently revalidated native live ZIP is
`live-0e83a254e2aa85a210e442b90cd44aa6.zip`, SHA-256
`2142f3fed2fe02bf3fee23efee68812dbd117da41a6e9460664fe8b71bcf64fd`.
It is a declared native fixture, not physical evidence. The phone/car gate remains
open: short individual workloads, no new warnings, combined stationary capture,
manual incident/evidence navigation, reconnect generation and actual retrieved
exports/reference comparison. See the staged plan above.

## Physical acquisition issue — user report, November 3, 2026

The user reports separate RPM-only and coolant-only recordings lasting over
50 seconds each, with the vehicle parked, engine running and the app operating
state set to Engine running / stationary. Both saved-recording sample counts
were reported as zero. Exact build, initialization status, voltage result and
raw session exports have not yet been confirmed for these runs.

Treat these as failed acquisition checks pending diagnosis (M6-04/M6-05 context),
not evidence of unsupported parameters or an ECU fault. Both workloads depend
on shared initialization and a valid supported-PID response before admission.
Code review identifies that dependency but cannot establish the physical cause.
Request the existing saved-session ZIPs together; even zero-sample finalized
sessions retain raw requests/replies and the capability report. Review evidence
indoors and batch any fix/retest into the next packet. Combined/reconnect testing
is on hold; no repeat car visit is requested while investigating these recordings.

## Export retrieval correction — native acceptance passed

The user also reports that Export sometimes responds only after repeated taps.
Retrieval is a blocking issue for the pending RPM/coolant diagnosis. Code review
found session sharing driven by a URL observer and separate Boolean state on a
parent navigation destination. Standalone capture export also scheduled its
output URL in a separate MainActor task before the caller checked for completion.

The correction moves an item-based share request onto the visible session detail
page, returns the completed archive URL directly, gives each presentation a fresh
identity, and clears it on completion/dismissal. ZIP preparation and failures have
visible states. Separate capture sharing returns its completed file directly and
is disabled when no closed standalone capture exists. Zero-sample recordings are
valid export candidates when finalized; their raw evidence must remain intact.

Native regressions cover the physical nested-history route with scripted
zero-sample RPM/coolant sessions, one-tap presentation and repeated exports, model
failure cleanup, and independent Python validation of actual native ZIPs.
[Native run 41749441572](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/41749441572)
passed on source `40f0a7fd62e44d44116371cecfee23d59cb44950`: 28 hosted tests,
three UI tests and both device compile profiles, with zero failures. The new UI
case opens the real system share sheet on one tap, checks the correct file and
Save to Files action, dismisses, reopens the same recording and then exports the
other recording. No retry taps are used. All five actual native ZIPs validated,
including zero-sample RPM/coolant recordings with retained no-data transactions.

Native evidence artifact
[13285573344](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/41749441572/artifacts/13285573344)
has SHA-256 `cf80a77df4f3d6b9d89488e8902369b35543b7052c386ccb551b131f8ae32fc7`.
Retrieved ZIPs and independent WSL replay are under ignored generated directories
`build/milestone-6-native/40f0a7f/` and `40f0a7f-replay/`. These remain explicitly
synthetic fixtures; the phone export behavior still needs confirmation. Updated
beta delivery is recorded below. Existing recordings can also
be retrieved through Files / On My iPhone / Vehicle Flight Recorder / Sessions,
using the app's existing document-sharing configuration. No further car recording
is requested to test retrieval. The physical acquisition cause remains unknown
until the existing raw records are available.

## Export beta delivery — November 3, 2026

Build `0.1.0 (8.1)` uploaded at **01:00:32 UTC November 4 / 7:00:32 p.m. CST
November 3** through
[signed run 41750645334](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/41750645334).
The signed source is the exact native-tested commit
`40f0a7fd62e44d44116371cecfee23d59cb44950`. Archive/export/signature checks passed;
Apple returned `UPLOAD SUCCEEDED with no errors`, delivery ID
`0036e40f-7341-4158-80ec-0f2d4f9e95cc`. Read-only inspection confirmed upload
COMPLETE with no errors or warnings, build VALID, `usesNonExemptEncryption: false`,
and internal state IN_BETA_TESTING. Apple lists the processed build at
01:01:32 UTC November 4 / 7:01:32 p.m. CST November 3. Build 8.1 is available
for internal TestFlight installation.

This update corrects session/capture export presentation and error reporting; it
does not change OBD initialization, acquisition admission or decoders. Install it
for the [indoor retrieval block](../planning/milestone_6_test_session.md#current-task--retrieve-existing-recordings-indoors)
using the already-saved phone recordings. No scanner connection or new recording
is needed. Physical single-tap/repeat export and receipt of the original ZIPs remain
pending. The RPM/coolant diagnosis and connected checklist remain on hold.
Appetize was not updated; PR #5 remains draft and M6 is not physically accepted.

## Retrieved phone evidence — November 6, 2026

Both user-supplied ZIPs independently pass ZIP integrity, exact six-file layout,
canonical session validation and raw/capability provenance replay. They match the
requested separate physical workloads with operating state `stationary_running`.
They were recorded on November 3 by build 7.1; exporting an older session does
not replace the original acquisition-build metadata.

| Session ZIP | Started (CST, November 3) | Duration | Requested workload | Decoded samples |
| --- | --- | --- | --- | --- |
| `live-8f5db243d65905d746ca5c98ff50e420.zip` | 5:30:34 p.m. | 50.433 s | RPM only | 0 |
| `live-ff33e74ab5d7fac00ba8963ad98bca69.zip` | 5:34:09 p.m. | 48.074 s | Coolant only | 0 |

SHA-256, in the same order:
`d5286fcd23dfdb313dc1d4132960c6734e376b585e7317e69c3abe63ed2a06c7`,
`c489cd9e24827931121171b9015756526f78397079e0eb30762b7df352cbb5ed`.
Private original ZIPs and extracted evidence remain under ignored
`.cache/physical-tests/2026-11-06/`; no phone raw files are committed or published.

Each report retains 52 raw rows and 17 transactions: all 16 adapter setup steps
succeeded, including `STPR` reporting 33 and `STI` reporting STN2255 v5.8.1.
The subsequent `0100` supported-PID discovery received exactly `NO DATA\r\r>`.
No support capabilities were established, so the scheduler withheld `010C`/`0105`
measurement admission. Neither recording ran long enough for the 60-second
support retry after this first response. No value was discarded by a measurement
decoder: no RPM/coolant measurement command was sent in either capture.

This establishes the immediate reason for zero samples, not why the adapter
received no discovery data. It does not prove the vehicle lacks RPM/coolant
support or justify bypassing discovery, inventing measurements, changing protocol
presets or adding enhanced requests. The coolant recording includes a brief
background/native scheduling gap after the no-data reply; it does not explain
the earlier discovery response. Final report `disconnected` state accompanies
manual session shutdown and does not by itself establish an unexpected link loss.

Receipt confirms these two finalized sessions can be exported and reviewed;
single-tap/repeat presentation on the phone has not separately been confirmed.
No voltage ZIP was provided. M6 physical acquisition remains unaccepted. Review
and any prepared targeted retest stay consolidated; no new car visit is requested
by this evidence handoff.

## Saved-session setup visibility — native acceptance passed

The physical session metadata already stores workload, selected operating state
and physical/fixture origin, but the Swift history model ignored those fields.
The current change decodes that existing metadata, adds workload/state to the
session list, and leads Drive details with recording setup and requested readings.
Zero-sample recordings explicitly say that no decoded readings were recorded;
requested workload and selected engine state are not presented as measured facts.
It also preserves compatibility with schema-1 mock history. This is a current M6
usability correction; it does not depend on the later visual redesign. Acquisition
behavior is unchanged.

[Native run 42266159403](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42266159403)
passed on source `84dcc28fbfa53795795b2f3a54c88a6cb07fe0bf`: 28 hosted tests,
three UI tests and both device compile profiles. All five native ZIPs independently
validated in Python. The UI case checks both RPM/coolant list labels, their matching
detail workloads, requested readings, selected engine state, explicit fixture
origin and zero-reading text, then reveals the offscreen Export row and verifies
one-tap/repeat sharing with the correct file and Save to Files action. Schema-1
mock history remains compatible. There were no failures in this accepted run.

[Native evidence artifact 13528570447](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42266159403/artifacts/13528570447)
has SHA-256 `b2772420a717e141be8fd73025e555fcf5308fe65a6ec19f50efb45a864ff5c1`.
Retrieved evidence, including both recording-setup screenshots, is under ignored
`build/milestone-6-native/84dcc28/`. These are declared native fixtures; phone
confirmation of the new labels remains an optional indoor check.

## Saved-session display beta — November 6, 2026

Signed build `0.1.0 (9.1)` uploaded from exact tested source
`84dcc28fbfa53795795b2f3a54c88a6cb07fe0bf` through
[run 42267707114](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42267707114)
at **18:10:25 UTC / 12:10:25 p.m. CST**. Archive/export and signature checks passed;
Apple returned `UPLOAD SUCCEEDED with no errors`, delivery ID
`5505f8e4-d0a0-4f07-97ec-e4a9f4af6fc7`. Read-only Apple inspection confirmed build
VALID, `usesNonExemptEncryption: false`, and internal state IN_BETA_TESTING. Apple
lists the processed build at **18:11:53 UTC / 12:11:53 p.m. CST**. Build 9.1 is
available for internal TestFlight installation.
Existing physical sessions receive their stored workload/state labels after updating;
no re-recording, metadata migration or scanner connection is needed. Phone display
confirmation is optional indoors. The two requested physical ZIPs are already
received and independently validated; successful ECU acquisition remains unaccepted.
Subsequent documentation changes do not alter the tested binary. PR #5 remains
draft, Appetize is unchanged, and no additional car test is requested by this update.


## Targeted discovery diagnostic build — in progress

The user confirmed all build-9.1 history and single-tap/repeat export checks pass.
These are closed human UI checks; the two earlier zero-sample phone ZIPs are also
already validated. The user has seen adapter voltage in the official app but has
not yet established live RPM/coolant there. The next visit therefore includes a
sequential independent standard-OBD comparison, not another voltage-only proof.

The new opt-in `discovery_diagnostic` workload uses initialization version
`obdlink-stp33-diagnostic-2`. It retains the fixed STP33 CAN 11-bit / 500 kbps
profile and adds, after selection: `ATSH7DF`, `STCAF0`, `ATCRA`, `STFA`, `STPBRR`.
The baud reply must be exactly 500000 before reads are admitted. Runtime header,
addressing and filters are explicit; no NVM protocol write, reboot, scanning,
user-supplied frame or enhanced request is introduced. These are diagnostic
controls, not an established fix for the vehicle's no-data response.

Sources reviewed November 6:
[OBDLink FRPM E](https://www.scantool.net/scantool/downloads/678/obdlink_frpm_e.pdf),
pp. 14, 18, 33–34, 37, 39, 57; [ELM327DSJ](https://www.elmelectronics.com/wp-content/uploads/2016/07/ELM327DS.pdf),
pp. 15–16. A standard OBD request can open the selected channel; absent STPO alone
is not established as this failure's cause. ATCS is retained as an opaque bounded
adapter reply, not a guessed numeric-counter decoder or bus-health assertion.

Diagnostic scheduling requests support and `ATCS` at 15-second nominal intervals;
no-data failures back off to 30 seconds. Existing minimum 500 ms admission,
single in-flight request, two-second deadlines, raw/resource limits, 600-second
owner-loop guard and reconnect initialization remain. Voltage is sampled at five
seconds; RPM/coolant at two/five seconds only after valid same-generation support.
Regular workloads retain initialization version 1 and their existing schedules.

The native screen adds setup progress, discovery attempts/outcome/eligibility and
CAN-status reply age. Unknown ECU support is explicit; adapter voltage does not
imply ECU communication. Raw requests, readbacks, failures and observations stay
in the canonical six-file live export. Python independently accepts both declared
setup versions, verifies diagnostic schedules/readbacks/status receipt and
same-generation support admission, and continues reading old phone ZIPs unchanged.

Portable Go tests/vet, 122 Python tests and 29 script tests passed. Native fixture
acceptance, signed delivery and the car run remain pending. The next human work
uses only the [single targeted diagnostic packet](../planning/milestone_6_diagnostic_session.md)
after delivery: D1, official-app reference, conditional D2, and ordinary combined /
reconnect continuation only when their written prerequisites pass. History/export
and one evidence handoff follow indoors. No per-case chat/upload is required.

**Milestone 7 not started.**
