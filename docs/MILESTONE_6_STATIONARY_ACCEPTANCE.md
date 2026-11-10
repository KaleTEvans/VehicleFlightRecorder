# Milestone 6 — physical stationary acceptance and quality report

Acceptance reviewed November 10, 2026. The user reports that the requested tests
worked as intended and notes only the brief transport-state change on Stop.
All five supplied build-13.1 ZIPs independently validate, and both real manual
incidents reconstruct with exact canonical/raw provenance.

**Result: the remaining M6 stationary acceptance checks pass at the scope below.**
The user authorized formal M6 closeout November 10, integrated through
[PR #5](https://github.com/KaleTEvans/VehicleFlightRecorder/pull/5). No additional
car visit or repeat of these tests is requested. M7 has not started.

## Setup and evidence identity

- Vehicle: the project's 2017 Alfa Romeo Giulia Quadrifoglio test car; exports
  themselves retain vehicle identity `unknown`, pack `none`, enhanced pack `none`.
- iPhone 16 Pro per user context; export metadata: iOS 26.6.2 (23G90), OBDLink MX+
  MX201, hardware 1.0.0, accessory-reported firmware 2.0.0. STN identity readbacks
  retain `STN2255 v5.8.1`; these are separate reported identity fields.
- App 0.1.0 (13.1), native source `0044c1e5f0ec181e86da8d1deb3de221f33145b5`;
  runtime initialization `obdlink-stp34-runtime-1`, explicit CAN29 / 500 kbps,
  profile `can29-normal-fixed-single-frame-headers-pci`, catalog `standard-live-can29-1`.
- The prior accepted native evidence remains 33 hosted / seven UI tests, both
  device compile profiles and 19 actual native ZIPs. The updated Python reader
  passes 157 tests and enforces session/report profile/catalog agreement.
- Original ZIPs, extracted six-file sessions, summaries and both reconstructions
  stay private under ignored `.cache/physical-tests/2026-11-10/ordinary-can29-13.1/`.
  No phone raw bytes, ZIPs or screenshots are published as fixtures.

## Received sessions

Times below are America/Chicago, CST, November 10. File numbering is a user label,
not a claim about execution order; the running-RPM case preceded the engine-off cases.

| File | Start CST | Selected state / workload | Duration | Readings | Transaction outcomes |
| --- | --- | --- | ---: | ---: | --- |
| VFR6.1 | 16:32:34.334 | Ignition on / engine off; RPM only | 52.984 s | 45 | 37 ok |
| VFR6.2 | 16:33:47.248 | Ignition on / engine off; Coolant only | 51.732 s | 21 | 29 ok |
| VFR6.3 | 16:31:04.080 | Engine running / stationary; RPM only | 50.620 s | 42 | 36 ok |
| VFR6.4 | 16:37:59.198 | Engine running / stationary; Coolant only | 51.906 s | 21 | 29 ok |
| VFR6.5 | 16:39:35.264 | Engine running / stationary; Combined | 320.047 s | 470 | 235 ok, 1 timeout |

Totals: **599 observations, 366 successful transactions and one timeout during the
intentional disconnect**, 1,650 raw rows, five complete exports, no uncheckpointed
raw rows or uncommitted observations. Each declaration, initialization sequence,
request, support advertisement, value, timestamp, generation and raw reference
passed independent replay. There are no decode/NO DATA outcomes. The Combined
report retains `drain_deadline_requires_reconnect`; it is preserved recovery
evidence rather than suppressed or described as an error-free connection.

All three responding IDs (`18DAF110`, `18DAF117`, `18DAF118`) advertised and supplied
RPM/coolant where requested. They remain opaque addresses: this does not establish
three physical ECUs, identify a TCM or equate generic coolant PID 05 to a
manufacturer-specific transmission-fluid temperature.

## Initial workload quality measurements

These are achieved successful-observation measurements for this stationary set,
not universal poll-rate guarantees or ECU sensor refresh rates. Full-session Hz
is **successful samples per responder / entire recorded duration**, including
setup, gaps and reconnect. Consecutive-gap median/max use actual successive
receipt offsets, including the inter-generation gap; they exclude only the
start-to-first and last-to-end edges. Completion latency is native stream full
acceptance to complete prompt-terminated response, not ECU measurement time.
A multi-responder transaction contributes one sample per responding address;
UI redraws and duplicate ZIPs contribute none.

| File / signal | Samples per responder | Full-session Hz per responder | Median consecutive gap | Largest consecutive gap | Completion latency min / p95 |
| --- | ---: | ---: | ---: | ---: | ---: |
| VFR6.1 / rpm | 15 | 0.2831 | 2.616 s | 2.702 s | 556.5 / 595.4 ms |
| VFR6.2 / coolant | 7 | 0.1353 | 5.600 s | 5.602 s | 556.2 / 565.9 ms |
| VFR6.3 / rpm | 14 | 0.2766 | 2.601 s | 2.674 s | 555.9 / 598.5 ms |
| VFR6.4 / coolant | 7 | 0.1349 | 5.635 s | 5.675 s | 549.4 / 623.3 ms |
| VFR6.5 / rpm | 94 | 0.2937 | 2.736 s | 41.977 s | 550.5 / 634.0 ms |
| VFR6.5 / coolant | 47 | 0.1469 | 5.674 s | 44.276 s | 546.9 / 632.6 ms |
| VFR6.5 / adapter_voltage | 47 | 0.1469 | 5.691 s | 43.006 s | 37.0 / 89.8 ms |

RPM and coolant rows have three distinct responders with the same counts/timing
population per transaction; adapter voltage has one adapter-origin responder.
Individual/Combined full-session Hz should not be compared as steady rates:
short individual sessions spend a larger fraction on setup. Typical consecutive
RPM gaps are about 2.60 s individually and 2.74 s Combined; coolant about 5.64 /
5.67 s. Configured 2 s / 5 s intervals, response completion time, other work and
minimum admission determine these observed gaps. The approximately 42–44 s
Combined gap includes the unplug, scheduling pause and new setup; it is not hidden
from session statistics.

Engine-off RPM (VFR6.1) is **45 valid zero samples**, 15 per responder; engine-off
coolant (VFR6.2) is 48 °C from all three addresses. Running RPM (VFR6.3) ranges
932–978.5 rpm across responders. Running coolant (VFR6.4) ranges 46–50 °C while
warming. Combined ranges are 917–982 rpm, 54–74 °C and adapter input 13.13–13.38 V.
This confirms that adapter/module availability and valid data can exist with the
engine off. M7 must distinguish fresh engine-running evidence from mere replies;
a zero or missing value must not be fabricated into the other state.

Reference context: the accepted November 6 VFR2/later OBDLink screenshot comparison
remains a basic sequential check of the unchanged decoders/units. It is not a
simultaneous accuracy reference for November 10's colder warm-up runs. Underlying
ECU measurement time/refresh and battery-terminal voltage accuracy remain unknown.

## Combined same-session recovery

VFR6.5 is one continuous 320.047-second recording and one session ID. Its native
accessory connection ID changes while Go generation advances 1→2.
Only one recording start exists; there is no intervening Stop or new recording.

| Offset from start | Evidence |
| ---: | --- |
| 84.236 s | First manual incident marked |
| 204.429 s | First full window finalized |
| 223.323–225.423 s | RPM request receives no completion and times out during unplug |
| 226.456 s | Drain deadline requires fresh connection; transport failure/disconnect logged |
| 247.887 s | New generation opened in the same recording |
| 247.925–261.004 s | All 21 runtime setup/readback steps repeat successfully |
| 262.112 s | Fresh generation-2 supported-PID response completes |
| 263.217 s | Fresh RPM from all three responder IDs |
| 264.317 / 264.904 s | Fresh coolant / adapter voltage |
| 295.740 s | Second manual incident marked |
| 320.025–320.047 s | Transport closes and manual Stop finalizes the session |

Generation 1 preserves 367 observations; generation 2 adds 103. Recovery takes
about **15.33 s from new app connection to fresh RPM**, or about 36.76 s from the
logged disconnect. Exact Bluetooth pairing/power-cycle times are not logged and
must not be inferred. Bytes demonstrate same-session recovery and fresh support;
they do not independently prove that Discover / Connect was never tapped. The
user reports the prescribed test worked as intended, with only the Stop-state
observation noted. Native automatic-owner recovery tests provide separate
synthetic coverage; no extra car visit is requested to repeat this accepted case.

The capture retains lifecycle transitions and an `acquisition_pause_observed`
event for a 21.50 s native scheduling gap around recovery. This does not qualify
continuous background, lock, suspension, termination or road behavior. Those are
M7 work. No old-generation bytes are decoded as new-generation measurements.

## Real manual incident reconstruction

Both incident journals and evidence references pass Python reconstruction:

- **First: Captured.** Trigger 84.236 s, requested/available interval 24.236–204.236 s:
  the full **60 s pre / 120 s post** window, no recorded coverage gaps/reasons.
  It completes before the intentional unplug.
- **Second: Partial.** Trigger 295.740 s, requested interval 235.740–415.740 s,
  recording ends at 320.047 s. About **24.306 s of post-trigger recording** exists.
  `post_window_unavailable` and `coverage_gap` are correctly retained, including
  recovery/startup silence in the pre-window and the final Stop disconnect edge.
  Missing data is not promoted to a captured window.

The user reports the test flow worked as intended. Native UI acceptance separately
verified ordinary history/profile labels and manual trigger navigation/scrolling.
Actual supplied phone ZIPs prove successful save/export and canonical/raw retrieval.
No new phone screenshots of already accepted navigation/export are required.

## Stop-state observation

`stopLive()` disables live acquisition, closes/cancels its transport, then finalizes
the Go session. Thus a brief **Ready → Disconnected** transition at Stop is
intentional. The view's automatic connection on appearance can subsequently open
an idle adapter connection; **Ready describes transport availability, not active
recording**. The native owner test verifies that reconnecting after Stop uses the
standalone transport path and does not resume live polling. All five provided
recordings end complete at Stop, with no later live observations in those sessions.

The closed live ZIPs do not capture why the UI returned to Ready afterward, so
its exact UI/task timing is not independently established here. The observed
brief transition does not by itself indicate a lost session or recording restart.
No app fix/new beta is required for the intentional Stop teardown. Clearer idle/
recording copy can be handled with future UX work.

## Acceptance boundaries and handoff

Accepted: installed signed iPhone/MX+ path, documented CAN29 setup/discovery,
generic supported reads without an enhanced pack, individual/Combined stationary
workload measurements, valid engine-off zero handling, same-session disconnect/
recovery with fresh generation evidence, full and partial manual incidents,
retrieved/reconstructed native phone exports and qualitative prior reference.

This meets the M6 stationary implementation/acceptance scope. M6 is formally
closed; M7 is not authorized or started by this closeout.
Road/hands-off/background behavior, universal vehicle compatibility, enhanced
TCM/module identity, internal sensor refresh and power/storage road budgets remain
unqualified. Preserve these limits and the measured recovery/setup delays in M7.

## Private input checksums

| ZIP | SHA-256 |
| --- | --- |
| VFR6.1.zip | `d3bcc74bfcfc5c84b9db6eb3003759f9256fa0c3d388979fd69787e025695e22` |
| VFR6.2.zip | `de7d500f07b944b7e3a0c2e00927db60265fa56a02a28edd43d7b637ee385d56` |
| VFR6.3.zip | `37d28d29b54ce51b3921ba8a98c048bc636565e0e21548e44980e3be6ff67da2` |
| VFR6.4.zip | `84cccb0ec8d48c243944e1459ad56115a896144a838cfe94aeb26cea58607bc4` |
| VFR6.5.zip | `850bbd3827954a92eb9d2e2795711a698c0491837ce01f93c7e8fe9c254afe96` |
