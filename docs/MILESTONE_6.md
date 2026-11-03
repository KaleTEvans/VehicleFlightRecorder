# Milestone 6 — adapter initialization and standard OBD

Status: PARTIALLY COMPLETE — November 3, 2026. Implementation/native fixture
acceptance and signed TestFlight delivery passed; phone installation and physical
stationary acceptance are tracked
below. User authorized closing M5 and starting
M6. M5 PR #3 merged as `94c5949`; implementation branch is
`codex/milestone-6-live-acquisition`. No M6 physical result is claimed.

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

Updated phone installation, actual stationary individual/combined
workloads and reference comparison, retrieved phone exports and manual incident
reconstruction. Automated/native evidence and signed upload have passed as
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

Do not perform a prolonged test until short initialization/individual checks pass.
Close competing scanner apps, park safely and use the approved TestFlight build.
Capture ignition-on/engine-off separately from stationary engine-running. Verify
no new dashboard warnings; stop if behavior changes. First test voltage-only,
RPM-only and coolant-only in separate short sessions (about 45 s each), exporting
all three. Check RPM zero versus unavailable honestly; compare values to a trusted
reference separately, recording tool/version, conditions, timestamps and units.
Do not run a competing diagnostic app concurrently on the same adapter.

Then run a combined five-minute stationary session: mark after at least 90 s and
continue at least 120 s afterward. Check session/incident history, evidence jump,
and ZIP export. Separately unplug/replug/reconnect during a short combined test;
verify a new generation, initialization/discovery, explicit gap and no replay.
No complete coverage should be claimed across that interruption.

Retrieve private phone ZIPs for Python replay and incident reconstruction. Record
adapter/STN firmware, iOS/app build, operating state, requested versus achieved
rate, latency, maximum gaps, timeouts/decode errors and unresolved reference
comparison. M6 remains incomplete until actual stationary results and evidence
are accepted. Road logging, automatic start/stop, notifications and full lifecycle
qualification remain Milestone 7.

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

**Milestone 7 not started.**
