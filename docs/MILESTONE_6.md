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


## Targeted discovery diagnostic build — delivered, physical test pending

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

Portable Go tests/vet, 122 Python tests and 29 script tests passed, together with
race checks for acquisition/mobile/OBD, actionlint and formatting. The updated
Python replay also independently revalidated both original physical ZIPs.

[Native run 42274732219](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42274732219)
passed on source `24037f0f0bdab0c5e0caff3a98cb3d0185b80629`: 29 hosted tests,
four UI tests and both device compile profiles, with zero failures. All seven
actual native ZIPs validated in Python, including diagnostic no-data (voltage only,
no invented ECU support) and later recovery (RPM/coolant only after support).
The shared status view passed no-data/retry/status and later-success UI checks;
existing history/repeated share-sheet regressions remain green.

[Native artifact 13533686241](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42274732219/artifacts/13533686241)
has SHA-256 `35ac43756107f3c0153cbeb29e1551a87608ed44b7431be2aa0bc74a620f32ed`.
Downloaded evidence is under ignored `build/milestone-6-native/24037f0/`; it remains
explicitly synthetic.

Signed `0.1.0 (10.1)` built that exact source through
[run 42276904829](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42276904829).
Archive/export/signature checks passed, with `UPLOAD SUCCEEDED with no errors`
at **19:11:05 UTC / 1:11:05 p.m. CST November 6**. Delivery ID:
`e57aa5f8-3806-4b75-999c-26b37d91185d`. Read-only Apple inspection confirmed
upload COMPLETE without errors/warnings, build VALID, `usesNonExemptEncryption: false`,
and internal state IN_BETA_TESTING. Apple lists the processed build at
19:11:47 UTC / 1:11:47 p.m. CST. Build 10.1 is available for internal TestFlight
installation. Follow-up documentation does not alter that binary; Appetize is
unchanged and PR #5 remains draft. The physical car run remains pending. The next human work
uses only the [single targeted diagnostic packet](../planning/milestone_6_diagnostic_session.md)
after delivery: D1, official-app reference, conditional D2, and ordinary combined /
reconnect continuation only when their written prerequisites pass. History/export
and one evidence handoff follow indoors. No per-case chat/upload is required.

## Diagnostic visit — initial user observations, November 6, 2026

The user reports two separate engine ECUs in OBDLink. RPM was readable under
either ECU and under the app's generic SAE PIDs category. Engine coolant was
reported available under SAE PIDs and a transmission-control-module category,
but absent from the individual engine-ECU menus. These are reported app selections;
exact PID labels, CAN addresses, responder/module identity and the meaning of the
TCM-category temperature have not been independently established. Do not merge
ECUs or label a responder TCM from its address/menu position, or treat menu absence
as a valid unsupported-PID response. Use the SAE engine RPM/coolant selections as
the planned standard-OBD comparison; keep other temperature selections separate.

The supplied VFR screenshot shows a completed 124-second recording, setup 21/21,
four discovery attempts with No data returned, adapter voltage 13.11 V (valid,
three seconds old) and CAN status as reported `T:00 R:00` (two seconds old).
The screenshot does not establish bus health, exact protocol, raw request/reply
provenance, build identity or whether it is D1 versus D2. Diagnostic session ZIPs,
reference labels/units and protocol if shown remain pending. No new command,
profile change or ECU mapping is justified solely by these menu observations.
Follow the existing D2 conditional gate; if already D2 with no ECU readings,
end the connected visit and skip C1/C2. One evidence handoff follows indoors.

## Validated D1 diagnostic export — November 6, 2026

The user supplied `live-d1dd3d808f5179464b8474742e35acab.zip` and confirmed it
was recorded **before** the successful OBDLink comparison (D1). SHA-256:
`fbd3145183a50b79318fd212876c26d8829a462396dd406d65025f6b8a5e9d46`.
The original/extracted evidence is private under ignored
`.cache/physical-tests/2026-11-06/`; no physical raw files are published.

Independent validation passes ZIP integrity, exact six-file layout, finalized
canonical records, both raw ledgers, configuration/readbacks and capability
provenance. The session began at 1:46:06 p.m. CST, lasted 124.462 seconds, and
records build 10.1 / physical origin / stationary_running / discovery_diagnostic.
Its 55 transactions and 188 raw rows retain:

- All 21 setup steps succeeded, including explicit functional header/normal
  addressing/filter acknowledgements, actual configured baud 500000, preset 33
  and STN firmware identity. Their physical command availability is now observed.
- Four `0100` requests at elapsed 12.728, 43.333, 73.931 and 104.533 seconds each
  received exactly `NO DATA\r\r>`. No valid ECU capability response was obtained.
- Eight `ATCS` replies each reported `T:00 R:00`. These remain opaque adapter
  evidence and do not establish healthy vehicle communication or a detected
  vehicle protocol.
- 22 decoded observations are all adapter input voltage. There are no RPM or
  coolant requests/observations, and no ECU capability rows. No interrupted or
  uncheckpointed evidence was found. Brief native lifecycle transitions are
  logged; no acquisition-pause warning is present.

The user could not find OBDLink's detected protocol. Record it as **unavailable**;
that optional field is not a reason for another car visit. This ZIP establishes
VFR's configured CAN 11-bit/500 kbps profile, not the protocol/addressing that
OBDLink used for its successful SAE readings. Those are separate facts. The
export does not capture the adapter's protocol before VFR resets/configures it.
Do not infer a protocol change, unsupported ECU, failed decoder or verified ECU
address map from these records alone.

D1 and reference-app observations are received; D2 export/status and reference
parameter labels/units remain unconfirmed. Do not ask to repeat D1 or locate the
protocol in the car. Review any already-saved D2 ZIP when available; C1/C2 cannot
be accepted from this D1 evidence. No implementation or new build is changed by
this evidence review.

## D2 interpretation and pre-reset protocol capture — design clarification

D2 is a before/after-reference comparison of the same VFR workload, potentially
revealing an order/readiness effect; it does not record OBDLink's commands or
prove which app configuration differs. Source review confirms both setup variants
begin STPC, ATD before selecting STP33. Thus build 10.1 overwrites reference-client
runtime settings before STPR/STPBRR are recorded. Do not claim it preserves or
identifies the working reference configuration.

The adapter provides read-only STPR, STPRS and STPBRR queries (FRPM E pp. 33–34).
A more informative future diagnostic revision would capture these on explicit
user start before any close/default/protocol-selection command, with no power
cycle between reference and VFR. This inspects shared accessory state, not the
other app's private data or traffic. It remains conditional: OBDLink shutdown or
accessory/client reconnection may already reset/change that state. Raw responses
must be labeled pre-existing adapter configuration, not a verified vehicle
protocol or permission to auto-select an unimplemented decoder/profile. If state
is unavailable, a reference-app diagnostic log/report would be a separate source.
No new initialization implementation is shipped by this clarification. Do not
request an additional visit only for an unchanged D2 on 10.1.

## Adapter-only handoff capture — delivered, physical result pending

The user did not run D2 and is willing to perform a handoff test if it can capture
settings left by OBDLink. The new opt-in `adapter_handoff` workload is separate
from Discovery diagnostic: only STPR, STPRS and STPBRR are sent, with optional
command echo tolerated. No STPC, ATD, STP33, vehicle query, voltage read, protocol
adoption or retry is part of this workload. An app/accessory reconnection may
still alter shared state before VFR can read it, so the result is unverified.

The versioned adapter-only contract is `profile_id` / protocol profile
`adapter-state-only`, catalog `adapter-state-1`, initialization
`obdlink-adapter-handoff-1`, selected protocol `unmodified-adapter-state`.
It has zero measurement schedules, observations/capabilities and manual incidents.
It auto-finalizes on three bounded readbacks, a transport loss, an unrecoverable
framing/deadline outcome, or its 15-second active owner-loop limit. Evidence-write
failures remain incomplete. It never reconnects/resynchronizes automatically.
AUTO, ? and other complete opaque replies are retained rather than converted to
vehicle protocol facts; ambiguity stays in the raw failure record.

Native UI distinguishes adapter readbacks from telemetry and explains zero samples
in handoff history. Python independently validates the adapter-only scope, exact
three-command order, no retries/generation reuse, receipt bytes and both prior
live setup versions. Existing recordings require no migration; new handoff
exports require the updated reader. The new [one short handoff packet](../planning/milestone_6_handoff_capture.md)
replaces another unchanged D2, with export/review indoors. Native acceptance is recorded below; no test on 10.1 can exercise this new mode.

The source `540c385bb465ac6404ed67ab4f393923fe2274c7` passed portable CI:
Go tests/vet, 129 Python tests, 29 script tests, race checks and formatting/actionlint.
All three original phone ZIPs still independently validate with the updated reader.
[Native run 42288515502](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42288515502)
passed 30 hosted tests, five UI tests, both device compile profiles and validation
of ten native ZIPs. The three added exports prove retained settings, unknown
AUTO/? settings and a timeout capture without reset/profile selection/vehicle
queries or retries. Existing diagnostic/history/export UI acceptance stays green.
[Native artifact 13538973065](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42288515502/artifacts/13538973065)
has SHA-256 `fdd3e434c8a5afc0719dd3bc96f17e54a986216aba021c800b39cd32b32e1a28`.
Downloaded evidence/screenshot is under ignored `build/milestone-6-native/540c385/`.
Signed build **0.1.0 (11.1)** used the exact tested source through
[run 42290107051](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42290107051).
Archive/export/signature verification passed; Apple returned `UPLOAD SUCCEEDED
with no errors` at **20:42:16 UTC / 2:42:16 p.m. CST November 6**. Delivery ID:
`dad48319-d4b3-487f-b156-b51231e76ec6`. Read-only Apple API inspection confirmed
build VALID / internal IN_BETA_TESTING, uploadedDate 20:42:58 UTC. It is ready
for the short handoff packet; an unchanged D2 on 10.1 is not requested.
The physical handoff result remains unconfirmed; no protocol change or decoder
expansion is justified solely by the scripted fixture settings. Follow-up delivery
notes do not alter the tested binary; Appetize remains unchanged and PR #5 stays draft.

## Physical adapter handoff received — build 11.1

The supplied `VFR1.zip` is the requested physical `adapter_handoff` session,
`live-8fc1fc4898c0c3e1b1acb0c86ead71eb`, build **0.1.0 (11.1)** on the MX+ /
iOS 26.6.2. It began November 6, 2026 at **3:41:36.733 p.m. CST** and completed
in **1.396830542 seconds** with `handoff_readbacks_captured`. Session validation
and independent capability/raw replay passed: three successful transactions,
12 raw rows, one connection generation, no issues, no uncheckpointed evidence.
Zero observations/incidents are expected for this adapter-only workload.

| Query | Exact retained reply (echo/prompt omitted) |
| --- | --- |
| STPR | `A7 [AT]` |
| STPRS | `AUTO, ISO 15765-4 (CAN 29/500) [AT]` |
| STPBRR | `500000` |

Original requested/accepted/received bytes agree with these receipts. Only these
three read-only queries were sent: no STPC, ATD, STP33, vehicle request, retry or
reconnect. Thus the handoff capture itself passed its physical readback/export
check. Raw reply suffixes remain preserved; the reader does not normalize the
AT-style number into an ST preset or adopt a vehicle profile automatically.

This is a concrete **29-bit/500 kbps adapter-state lead**, whereas VFR's regular
and diagnostic acquisition explicitly select STP33 (11-bit/500 kbps). The
[manufacturer FRPM E](https://www.scantool.net/scantool/downloads/678/obdlink_frpm_e.pdf#page=33)
distinguishes ST presets 33 (11-bit Tx) and 34 (29-bit Tx); both can receive both
identifier sizes, so transmit addressing matters. Do not interpret raw `A7 [AT]`
as literal ST preset 7 or feed it directly to STP. The retained description and
baud readback provide the concrete evidence here.

A transmit-profile/addressing mismatch is now a supported diagnostic hypothesis
for the earlier NO DATA, not an established root cause. This capture contains no
ECU response and cannot prove which setting OBDLink used for its earlier live
readings, identify ECU/TCM addresses, or verify RPM/coolant acquisition. Cleanup
or accessory reconnection may still have changed state before the capture.

Next engineering scope: add and independently test an explicitly versioned
29-bit/500 kbps diagnostic profile, matching functional transmit addressing and
29-bit response parsing/provenance, before requesting another batched parked
session. Merely changing STP33 to STP34 is insufficient for the existing 11-bit
header/parser assumptions. No further repeat on 11.1 is requested. M6 physical
ECU acquisition and remaining acceptance gates stay open; this review ships no
new app binary or acquisition behavior.

ZIP SHA-256: `2d8e3f59716702322b7d2e70d42d66d991128645dd9ae10ffc068e87076c5796`.
The original ZIP, extracted six-file session and validation summary remain private
under ignored `.cache/physical-tests/2026-11-06/handoff-11.1/`.

## Explicit CAN 29-bit diagnostic profile — delivered, physical pending

The user authorized implementing the next targeted profile after the physical
11.1 handoff retained CAN 29/500. `can29_diagnostic` is a separate opt-in workload:
STP34, standard functional transmit header ATSH18DB33F1, normal-fixed addressing,
automatic receive filters and strict 34 / 500000 readbacks in a versioned 21-step
setup. Regular/11-bit diagnostic/handoff workloads keep their previous selection
and contracts. There is no automatic protocol search, arbitrary send, NVM protocol
write, enhanced PID or ECU write. Opening a connection still sends no commands.

The profile/catalog/initialization identifiers are documented in DATA_SCHEMA.md.
The Go parser and independent Python replay retain full 18DAF1xx responder IDs,
header/PCI and per-ECU raw provenance; reject wrong destination/11-bit/multiframe/
foreign data; and isolate duplicate or malformed responders. Capability rows now
sort actual addresses rather than iterating an 11-bit-only list. The 29-bit
namespace has a separate bounded reader budget; legacy limits remain unchanged.
No address is assigned a physical module name or a TCM temperature meaning.

Native UI exposes CAN 29-bit diagnostic and shows its active profile; saved
history identifies the workload. New tests cover two responders, supported and
unsupported readings, NO DATA/recovery, timeouts/late bytes, setup mismatches,
reconnect generation isolation, independent replay/corruption, native UI and ZIPs.
Portable checks passed: full Go tests/vet, targeted CAN29 configuration/parser/
reconnect tests, 144 Python tests, 29 script tests, race checks, formatting and
actionlint. All four retained phone captures independently validate. Native
acceptance and signed delivery passed as recorded below. The new build is ready
for one parked session; physical acquisition acceptance is still open. The [new single-visit packet](../planning/milestone_6_can29_session.md)
is ready for 12.1. It batches discovery/acquisition, conditional reconnect
and a later official-app SAE comparison, followed by one indoor ZIP handoff.
M6 physical ECU acquisition/reference/reliability/manual gates remain open.

Exact source `1340837ad24ef000d24e9dedf26e18ae9efa191e` passed
[native run 42300865187](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42300865187):
**31 hosted tests, six UI tests, both device compile profiles and 13 native ZIP
validations**, including CAN29 no-data/dual-responder recovery/timeout cases.
All available incidents reconstructed independently. Both push/PR portable CI
runs on that source passed. The native recovery export has 92 observations,
the timeout/recovery export 87 including one preserved timeout, and the no-data
export 19 adapter-voltage observations with four discovery NO DATA outcomes.
These are scripted native fixtures, not new physical measurements.

[Native artifact 13543509279](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42300865187/artifacts/13543509279)
has GitHub SHA-256 digest
`c2ee605f9c8963b1ea8f02db1901a4d068a8dadd2cbb8b4a15f4d9f9a59385f4`.
Downloaded evidence is under ignored `build/milestone-6-native/1340837/`.
Signed [run 42302376611](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42302376611)
built the exact accepted source as **0.1.0 (12.1)**. Archive/export/signature
verification passed; Apple returned UPLOAD SUCCEEDED with no errors at
**22:11:57 UTC / 4:11:57 p.m. CST November 6, 2026**. Delivery UUID:
`441594a2-3462-4a6e-bf6f-212db70f7252`. Read-only Apple API inspection confirmed
build VALID / internal IN_BETA_TESTING, uploadedDate **22:13:53 UTC /
4:13:53 p.m. CST**. The build is available for the ready single-visit packet.
PR #5 remains draft; Appetize is unchanged. Documentation follow-ups do not
alter the accepted/signed binary.

## Physical CAN29 acquisition verified — VFR2, build 12.1

The user supplied `VFR2.zip` and VFR/OBDLink screenshots. The physical session
`live-af1129593cd04473c608ef8cdcb899ca` ran on MX+ / iOS 26.6.2 / build **12.1**
from **November 6, 2026, 6:20:39.136 to 6:21:44.237 p.m. CST** (65.101 seconds).
The six-file ZIP independently validates, including canonical/report/raw replay:
**87 observations, 63 successful transactions, 284 raw rows**, no acquisition
issues, no timeouts/NO DATA/decode failures and no uncheckpointed evidence.

All 21 setup steps completed, including STP34, functional header and protocol/baud
readbacks. Four supported-PID transactions returned three distinct responders:
`18DAF110`, `18DAF117`, `18DAF118`. Each advertised RPM and coolant support and
subsequently supplied valid Mode 01 PID 0C/05 values. Addresses remain opaque;
this capture does not identify which is ECM/TCM or establish three physical ECUs.

| Source | Samples | Recorded range |
| --- | ---: | --- |
| RPM / 18DAF110 | 17 | 917.5–971 rpm |
| RPM / 18DAF117 | 17 | 920–971 rpm |
| RPM / 18DAF118 | 17 | 921–969 rpm |
| SAE coolant / 18DAF110 | 9 | 78–79 °C |
| SAE coolant / 18DAF117 | 9 | 78–79 °C |
| SAE coolant / 18DAF118 | 9 | 78 °C |
| Adapter input voltage | 9 | 12.47–12.52 V |

The VFR screenshot at 45 elapsed seconds matches the last retained raw-derived
readings: voltage 12.50 V, all three coolant values 78 °C, and RPM 958.5 / 959 /
962 for the respective IDs (the third RPM row is below the visible image crop).
The later OBDLink screenshot at 6:25 shows RPM 963, Engine Temp **177.8 °F =
81 °C**, Battery **12.5 V** and speed 0. It is sufficient for the requested basic
reference check; no additional screenshot is requested. The values are consistent
with a later idle/warming comparison, but it is not a simultaneous accuracy test
and the gauge screenshot does not expose selected PID/responder identity. Its
Battery label does not establish the same underlying voltage source as VFR.

Measured median consecutive receipt gaps were about 2.785 s for RPM and 5.956 s
for coolant; maxima about 4.053 / 6.713 s. Adapter-voltage median gap was 5.911 s.
Report full-session rate/gap statistics include initial setup and should not be
read as a promised steady polling/sensor-refresh rate. Brief lifecycle changes
are recorded without observed transaction failures; background continuity is
still unqualified. There is one connection generation and one final disconnect
12 ms before manual stop, consistent with native stopLive() cleanup closing the
transport before finalization. It is not evidence of unplug/replug recovery.

**Accepted at this scope:** real CAN29 supported-PID discovery, per-responder
RPM/coolant and adapter-voltage acquisition, native display/raw provenance and
successful save/export, plus a qualitative later reference comparison. This
resolves the zero-ECU-data symptom for the explicit 29-bit diagnostic profile.
It supports the addressing/profile explanation without proving which individual
setup difference caused every earlier failure. Regular Combined/RPM-only/coolant-
only modes still use the old profile and are not qualified by this result.

**Still open:** the planned two-minute/reconnect block (this ZIP is 65 seconds,
with no second connection), manual-incident physical acceptance, and remaining
M6 stationary/reliability gates. Reconnect outcome is being clarified from tests
already performed; no additional car visit or repeated baseline/reference capture
is requested. Future connected work remains batched. PR #5 stays draft; this
review ships no new app behavior and does not close M6 or start M7.

ZIP SHA-256: `398c50c24235ecff36e1f2f1b0a59b76b1d20fe3359a6504598960b2be2b3ce4`.
Original ZIP, both screenshots, extracted session, validation summary and comparison
notes remain private under ignored `.cache/physical-tests/2026-11-06/can29-12.1/`.

## Two-minute pre-unplug baseline verified — VFR3, build 12.1

The user identified `VFR3.zip` as the recording **before unplugging the scanner**.
It independently validates as physical CAN29 diagnostic session
`live-c8cde8284a610d375eb72e3bee946b5b`, build 12.1 / MX+ / iOS 26.6.2. It ran
**November 6, 2026, 6:15:50.720–6:17:53.475 p.m. CST**, lasting **122.755 seconds**.
This precedes VFR2 (6:20:39–6:21:44 p.m.); attachment numbering is not chronology.

The complete six-file export contains **187 observations, 516 raw rows, 110
successful transactions and one cancelled in-flight RPM request**. All 21 setup
steps and seven supported-PID discoveries succeeded. There are no acquisition
issues, timeouts, NO DATA, decode errors or uncheckpointed evidence.

| Source | Samples | Recorded range |
| --- | ---: | --- |
| RPM / 18DAF110 | 37 | 937–975 rpm |
| RPM / 18DAF117 | 37 | 935.5–975 rpm |
| RPM / 18DAF118 | 37 | 936–975 rpm |
| SAE coolant / 18DAF110 | 19 | 67–74 °C |
| SAE coolant / 18DAF117 | 19 | 67–74 °C |
| SAE coolant / 18DAF118 | 19 | 66–73 °C |
| Adapter input voltage | 19 | 12.47–12.51 V |

The final RPM request was stream-accepted and received three responder lines, but
had not received its terminating prompt when Stop cleanup disconnected the
transport at 122.735593 s, about 20 ms before finalization. Its raw bytes are
preserved and its outcome is cancelled; no observations are invented from that
unfinished transaction. Final RPM capability rows consequently say unavailable
while retaining advertised/observed evidence. This is consistent with orderly
cancellation during Stop, not a decoder failure or proof of a physical unplug.

Median consecutive receipt gaps were about **2.781 s RPM**, **5.645 s coolant**,
**5.643 s voltage**; maxima 4.321 / 6.381 / 7.210 s respectively. Brief native
lifecycle changes were recorded, including background intervals; this single
recording does not qualify general background reliability. Per-responder values
remain distinct and are not assigned physical module names.

**The planned two-minute baseline-duration/acquisition check is now satisfied.**
This supersedes the VFR2-only duration limitation above; do not request a repeat.
VFR3 and VFR2 demonstrate successful acquisition in two separate sessions. Each
contains only one connection generation, so automatic recovery through a physical
unplug/replug during an active recording is still unverified. User labeling
establishes VFR3 as pre-unplug; it does not by itself identify how VFR2 was started
or whether VFR recovered without manual intervention. Reconnect clarification
remains about work already performed; no new car visit is requested. Manual/
remaining reliability gates and ordinary-profile qualification stay open; PR #5
remains draft. This evidence review does not alter the delivered app or close M6.

ZIP SHA-256: `e94cbebdc127ee2b25deded3cd6766619d2ddeae9ab59f009d1ebb603ee8e668`.
The original ZIP, extracted `vfr3-session/` and `vfr3-validation-summary.json`
remain private under ignored `.cache/physical-tests/2026-11-06/can29-12.1/`.

## Ordinary CAN29 recording — implementation, acceptance pending

Following the accepted two-minute diagnostic baseline, the user authorized
proceeding with the remaining M6 work. The next change separates ordinary
workload from protocol selection: Combined, RPM-only, Coolant-only and Adapter-
voltage-only accept an explicit CAN11 or CAN29 profile. Omitted core options
retain legacy CAN11; the native test-car UI defaults to CAN29, clearly labels
and remembers that selection, and prevents changing it during an active session.
It is not automatic vehicle/protocol identification. Diagnostic/handoff modes
keep their fixed scope and reject an explicit override.

Regular CAN29 declares `obdlink-stp34-runtime-1` and reuses the tested fixed
21-step addressing/readback sequence. Ordinary schedules and manual incident
windows remain; diagnostic CAN-status polling is absent. Active status and
history show the actual declared protocol. Reconnect preserves one session and
its profile, clears old measurements/capabilities and requires fresh setup/support.
No new decoder, threshold, ECU mapping, enhanced signal or M7 policy is added.

Tests exercise all ordinary workload/profile combinations and legacy default,
complete/partial manual incidents, same-session reconnect/cache invalidation,
no-data/timeout and independent replay/reconstruction/corruption. Native coverage
adds the actual owner-loop accessory replacement path, ordinary history/manual
navigation and six native exports. All six retained phone captures still validate.
Portable Go tests/vet, 156 Python tests and 29 script tests passed; targeted
ordinary-profile replay/reconstruction and all six retained phone exports also
validate. OBD/acquisition/mobile race checks also passed. Native/delivery results will
be recorded after completion. The
[remaining one-visit packet](../planning/milestone_6_final_session.md) is draft
pending delivery; it groups individual workloads and a Combined incident/reconnect
recording, then one indoor retrieval block. No repeat diagnostic baseline or
reference screenshot is requested. M6 remains open and PR #5 stays draft.

**Milestone 7 not started.**
