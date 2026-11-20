# Milestone 7 — targeted recovery follow-up

**HOLD — signed 0.1.0 (20.1) has not been uploaded or delivered.**
Native verification passed (46 hosted / 9 UI tests). Do not start another car
visit using 14.1. Release this packet only after approved TestFlight delivery is
confirmed. Final source:
`19e83737ef623014c076dbc8f18d24a65efe0604`. Earlier signing-only candidates are
superseded and must not be uploaded.

Purpose: retest the failed handover/relaunch/export paths in one parked block.
Reuse the successful ordinary readings, full manual incident and earlier M6
baseline; no new OBDLink comparison or per-case uploads are required.

## Preparation indoors

Record replacement build, iPhone/iOS, adapter firmware, charging state and starting
battery. Close other scanner clients. Keep CAN29 / 500 kbps and limited standard
OBD coverage. No enhanced Alfa/TCM or vehicle-control commands are involved.
Optional recording notifications may remain disabled for this recovery block;
notification delivery and lock/background acceptance are separate open cases.

## One parked block — approximately 15–20 minutes

Use a ventilated parked location and normal engine operation. Stop the block for
a new vehicle warning or persistent acquisition failure. Do not repeatedly force
recovery or power-cycle the scanner until the available diagnostic evidence is
saved. A blue Bluetooth connected status alone is not proof of an app stream.

| Case | Action | Check |
| --- | --- | --- |
| R1 | Start VFR with scanner attached and engine running. Enable Vehicle is parked and Hands-off monitoring; allow up to 90 s. | One automatic drive; one primary RPM and coolant row, actual source shown. Expand All responders to see all sources. |
| R2 | Mark Incident once; continue about 10 s. | Button acknowledges the press, brief success feedback, incident count increases and latest capture status is visible. A short pre/post window may be Partial; no new full-window repeat is required. |
| R3 | Stop and save while engine remains running. Wait 60 s. | Monitor returns to waiting with Manual Stop override; no immediate new drive or repeated red startup errors. |
| R4 | Disable/re-enable monitoring once, allow up to 90 s. During the new drive force-quit, wait 30 s and reopen; allow up to 90 s again. | Explicit rearm works. Relaunch preserves the old drive as interrupted and may start a distinct drive after fresh evidence. No fabricated continuity or retry-budget latch. |
| R5 | During recording unplug scanner for about 20–30 s, plug it back in and reconnect Bluetooth; allow up to 90 s. | App recovers usable telemetry; any gap is visible. Record if Discover / Connect or another action is needed. |
| R6 | Stop/save, turn engine off normally for about 75 s, then restart. Observe whether fresh zero RPM was actually available. | Automatic override clears only after sustained fresh zeros; missing replies alone are unknown. If no zero replies arrive, use explicit off/on once, record that fact and treat zero-based rearm as unverified. |
| R7 | Tap Export M7 diagnostic bundle once, including immediately after any persistent failure. | Monitoring stops, preparation progress or a readable error appears, and the share sheet opens. Save one ZIP to Files. Dismiss and retry once indoors to verify repeat sharing. |

R6 does not by itself qualify all sustained-end cases. For an automatic drive
that has not been manually stopped, end timing still depends on ongoing fresh
zero RPM (60 s span) versus absent-RPM unknown grace (180 s). Do not add another
car block just to cover these while reviewing recovery.

## One handoff after the visit

Send one diagnostic ZIP and one note with R1–R7 outcomes, approximate delays,
visible errors, required manual recovery and ending battery/charging context.
Review saved interrupted/complete sessions indoors. If export fails, preserve
Files → On My iPhone → Vehicle Flight Recorder folders (Automation, Sessions and
ConnectionCaptures) and compress them together, as in the first visit. Do not
remove/reinstall the app or delete originals to recover.

After review, release only the remaining affected lock/background/notification,
normal sustained-end and later controlled-road/resource blocks. Milestone 7 stays
open; road use is not authorized by this recovery packet.
