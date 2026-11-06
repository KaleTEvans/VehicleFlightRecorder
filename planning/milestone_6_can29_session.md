# Milestone 6 — CAN 29-bit diagnostic session

Status: READY. Install **Vehicle Flight Recorder 0.1.0 (12.1)** from TestFlight.
Source: `1340837ad24ef000d24e9dedf26e18ae9efa191e`. Uploaded November 6, 2026
at **4:11:57 p.m. CST**; Apple confirmed VALID / internal IN_BETA_TESTING.
This exact source passed 31 hosted native tests, six UI tests, both device compile
profiles and independent validation of 13 native ZIP exports.
Do not run this packet on 11.1. The prior handoff passed; no repeat is requested.
This is one parked visit with a single saved session and an optional reconnect
block only if initial ECU readings succeed. Export/review follows indoors.

## Purpose and expected behavior

Test the specific lead from the physical handoff: the scanner retained a
CAN 29-bit / 500 kbps description, whereas earlier VFR recordings selected
11-bit / 500 kbps. The new **CAN 29-bit diagnostic** workload explicitly sets
29-bit / 500 kbps, uses standard functional addressing and preserves full
responder IDs. A successful setup alone does not establish vehicle communication.
RPM and SAE engine coolant requests wait for valid supported-PID discovery.
No enhanced Alfa/TCM parameters or automatic protocol search are enabled.

## Prepare indoors

- Install the delivered build listed above and keep this packet available.
- No repeat of D1/D2, adapter handoff, or accepted 9.1 history/export checks.
- Plan about **5–10 minutes parked**. Keep the app foreground and screen awake.
  Background/locked-screen qualification is outside this diagnostic packet.

## One connected block

1. Park, apply the parking brake and idle under suitable ventilated conditions.
   Completely close OBDLink before VFR so only one app uses the scanner.
2. In VFR enable **Vehicle is parked**, select **Engine running / stationary**,
   and choose **CAN 29-bit diagnostic**. Start stationary recording once.
   Confirm the active status says **Profile: CAN 29-bit / 500 kbps**.
3. Let the first **two minutes** run. Note setup progress (normally **21/21**),
   discovery outcome/attempts and any full ECU IDs beside RPM/coolant readings.
   Screenshot the readings/IDs if available. Two differing RPM values are kept
   separately; the app must not turn an unknown reading into zero or label an
   address as TCM solely from its number.
4. **If setup fails, no ECU readings appear after two minutes, a new warning
   appears, or the app errors:** Stop and save; retain the exact state/error.
   Skip reconnect. Do not retry profiles, reset the scanner or repeat the handoff.
   Voltage alone does not pass ECU acquisition.
5. **Only if ECU readings appeared:** while the same recording remains active,
   unplug the scanner once. Wait until VFR reports the loss; plug it back in and
   reconnect it through iOS Bluetooth if needed. Keep the car parked and ignition
   unchanged. Allow up to **60 seconds after Bluetooth reconnection** for VFR to
   reconnect, repeat setup/discovery and obtain fresh readings. If it fails to
   recover, note the state; do not restart or tap Discover repeatedly. A failed
   recovery is useful evidence from this same session.
6. Stop and save. Close VFR before opening OBDLink. In OBDLink, record the live
   **SAE engine RPM** and **SAE engine coolant** labels, values and units, preferably
   with one screenshot. Note that it is a later comparison and coolant may drift.
   Do not substitute a TCM-specific temperature or search for the protocol menu.
   The already-established official-app success does not need repeated debugging.

A CAN status reply such as T:00 R:00 is retained adapter evidence, not proof of
successful ECU communication. This diagnostic mode has no Mark Incident control;
manual-window acceptance remains open and is not part of this short packet.

## Finish indoors — one evidence handoff

Open **Saved sessions, incidents and export → CAN 29-bit diagnostic → Export
session ZIP**. Send the single ZIP plus any screenshots and these brief notes:

- Did RPM and coolant appear? Which full ECU IDs were shown?
- Was reconnect run, and did fresh readings return without restarting VFR?
- Official-app SAE labels/values/units and any dashboard warning or error.

Export the failure session too, including a zero-sample result. No per-case chat
or additional car visit is needed. Evidence review determines whether this profile
worked and what remains for M6; this packet alone does not close the milestone.
