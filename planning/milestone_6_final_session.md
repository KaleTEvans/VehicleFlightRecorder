# Milestone 6 — ordinary CAN29 recording and incident acceptance

Status: PASSED — November 10, 2026. The user reports the test flow worked as
intended. All five supplied build-13.1 exports independently validate, including
same-session loss/recovery, one Captured manual incident and one Partial incident;
both incidents reconstruct. VFR6.1/6.2 show engine-off RPM/coolant and VFR6.3/6.4
running single-signal reads; VFR6.5 is the continuous Combined case. No repeat
visit or extra screenshots are requested. See
[the acceptance/quality report](../docs/MILESTONE_6_STATIONARY_ACCEPTANCE.md).

Build: **Vehicle Flight Recorder 0.1.0 (13.1)**, native source
`0044c1e5f0ec181e86da8d1deb3de221f33145b5`. Uploaded November 6 at 7:28:21 p.m. CST;
Apple confirmed VALID / IN_BETA_TESTING. Native acceptance: 33 hosted / seven UI
tests, both device compile profiles and 19 actual native ZIPs. Current reader: 157
tests; all 11 retained phone captures validate. Instructions below are the
historical test protocol, not a new request. The user authorized formal M6 closeout
November 10, integrated through [PR #5](https://github.com/KaleTEvans/VehicleFlightRecorder/pull/5).
M7 has not started.

## What this build adds

Regular Combined, RPM-only, Coolant-only and Adapter-voltage-only workloads have
an explicit **OBD protocol** selection independent of workload. Choose
**CAN 29-bit / 500 kbps** for this test car. It is the new UI default and is
remembered; it is a selection, not automatic vehicle/protocol identification.
Active status and saved history show the selected protocol. Regular modes use
normal polling schedules, with the tested 21-step CAN29 runtime setup. Diagnostic
and adapter handoff modes retain their separate fixed contracts.

This packet completes ordinary-workload rate reporting, a real manual incident
window and **unplug/replug recovery inside the same active recording**. Native
fixtures alone do not establish those physical results. M7 road/hands-off work is
not enabled by this build.

## Prepare indoors

Install the build listed above. Keep this packet available and plan one **10–15
minute parked session**. Keep VFR foreground and the screen awake throughout.
Close OBDLink; do not run competing scanner apps during recordings. No tests of
background/force-quit/road behavior are requested here.

## One connected visit

Keep the vehicle parked with its parking brake applied. Run the engine only under
suitable ventilated conditions. Stop for any new warning or abnormal behavior.
Enable **Vehicle is parked** and select **CAN 29-bit / 500 kbps** for each regular
workload below. Record any actual error/unknown state without repeatedly retrying.

| Recording | Conditions / workload | Duration | What to note |
| --- | --- | --- | --- |
| F1 | Ignition on / engine off; RPM only | About 45 s | Whether RPM becomes a valid zero, stays unknown, or returns NO DATA; adapter connection alone is not engine-running proof |
| F2 | Engine running / stationary; RPM only | About 45 s | Fresh RPM values with full responder IDs |
| F3 | Engine running / stationary; Coolant only | About 45 s | Fresh coolant values with full responder IDs |
| F4 | Engine running / stationary; Combined | About 5–6 min | Instructions below: complete manual window, reconnect and intentionally shortened second window |

F1 is conditional on safely establishing ignition-on/engine-off for this car. If
that state is unclear, label it skipped and continue with the parked running
checks; no diagnosis or setting search is requested. Stop and save F1–F3 before
changing workload. The raw exports provide achieved individual/combined receipt
rates; display refreshes do not count as samples.

If F2/F3 show no ECU values after 45 seconds, preserve them and stop dependent
F4 testing. Do not switch back to the diagnostic workload to force a pass.

### F4 — one continuous Combined recording

1. Start Combined with Engine running / stationary and CAN 29-bit / 500 kbps.
   Confirm fresh RPM, coolant and adapter voltage appear.
2. After **01:15**, tap **Mark Incident** once and note the app timer. Keep recording
   for another **two full minutes**. This gives the first incident its 60 s pre /
   120 s post window.
3. Wait **at least 2 min 15 s after the first mark** (about **03:30** if you marked
   at 01:15), then unplug the scanner once **while F4 is still recording**.
   **Do not press Stop, quit VFR or start a replacement recording.** Wait for the
   connection loss, plug the scanner back in, and reconnect iOS Bluetooth if needed.
4. Keep VFR open. Allow up to **60 seconds after Bluetooth reconnection** for
   automatic recovery: setup/discovery repeat and fresh sensor readings return.
   A retained number or a Bluetooth connection alone is not recovery. Note whether
   restarting VFR or tapping Discover / Connect was needed; avoid those recovery
   actions until the automatic-recovery observation is complete.
5. If recovery fails, Stop and save the same F4 session; skip the next mark. Keep
   the successful first incident and the failed recovery as evidence. Do not repeat
   F1–F3 or return for another test.
6. If recovery succeeds, wait **at least 3 min 15 s after the first mark** (about
   **04:30** if you marked at 01:15), and allow at least 15 s of fresh readings
   after recovery. Tap **Mark Incident** a second time, then **Stop and
   save about 10 s later**. This second incident should show an honest partial
   window; the two marks are sufficiently separated to avoid merging their windows.

Report any new vehicle warning, stale-looking values, unexpected automatic stop,
manual reconnect action or app error. The 10-minute native session guard remains.
No extra long recording or repeated OBDLink screenshot is needed.

## Finish indoors — one evidence handoff

- Open saved sessions and confirm F4 says **Combined**, **CAN 29-bit / 500 kbps**.
- Open its first manual incident: expect **Captured**. Use **View evidence around
  trigger** and confirm the timeline is brought into view.
- If the second mark was made, expect **Partial**, with the shortened post-window
  explained. A reconnect gap may also be recorded; it must not be hidden.
- Export each actual F1–F4 ZIP (including failed/zero-reading cases). Export and
  evidence navigation were already accepted; only report a regression if one occurs.
- Return all ZIPs together, labeled F1/F2/F3/F4, with these notes: automatic recovery
  yes/no and approximate time; manual action if any; first/second incident status;
  any warnings/errors; F1 running state if skipped/unclear.

No per-case chat/upload or additional car visit is required. Raw/reconstruction
review determines which remaining M6 gates passed. Keep unverified gates open;
this packet does not itself close M6 or begin M7.
