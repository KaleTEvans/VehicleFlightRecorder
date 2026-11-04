# Milestone 6 — one parked test visit

Status: PAUSED pending investigation of the reported zero-sample RPM/coolant
recordings. Preserve and return those existing sessions together; no repeat car
visit or combined/reconnect testing is requested until evidence review. See
[the device issue](../docs/MILESTONE_6.md#physical-acquisition-issue--user-report-november-3-2026).
The checklist below remains the planned consolidated sequence after diagnosis.

## Current task — retrieve existing recordings indoors

Export-fix build **0.1.0 (8.1)** uploaded on November 3 at 7:00 p.m. CST;
Apple confirmed VALID / IN_BETA_TESTING. Update for this retrieval block;
do not repeat the car cases below.
No scanner connection is needed for this block.

- [ ] Open saved sessions and choose the existing finalized RPM-only recording.
  Tap Export session ZIP once; allow ZIP preparation, then save/share the archive.
- [ ] Dismiss the share sheet and export that same recording again to confirm
  repeat presentation. One copy of its ZIP is sufficient for the evidence handoff.
- [ ] Export the existing coolant-only recording and any already-recorded voltage
  session. Keep the original recordings; do not create replacements for this step.
- [ ] Return those ZIPs together, noting any export error or the session's displayed
  state. If an interrupted session cannot export, retain its original folder via
  Files / On My iPhone / Vehicle Flight Recorder / Sessions instead.

These are retrieval checks, not successful physical acquisition acceptance. Review
the saved raw replies before issuing a consolidated connected retest packet.

## Planned connected visit — remains paused

This checklist supersedes the
previous standalone “45-second voltage test, then return for review” request.
Run the listed checks in one visit when their continue criteria pass; return the
recordings together afterward. It follows the [shared test-session workflow](physical_test_sessions.md).

Target: the owner's 2017 Giulia QV and paired OBDLink MX+. Use the iPhone 16 Pro
(previously reported iOS 26.6.2), noting any device/OS/adapter changes.

Original acquisition baseline: **Vehicle Flight Recorder 0.1.0 (7.1)**, app source
`06c5e207ae82aae259a9720271fc5f08df359511`. Export-fix build **8.1**, source
`40f0a7fd62e44d44116371cecfee23d59cb44950`, preserves that acquisition path.
The connected visit remains paused; review the next retest packet before going.
Plan approximately **20–30 minutes parked**, with extra time
if using a prepared reference tool. This is an estimate, not measured reliability.
Each recording stays below the build's 10-minute owner-loop limit. The fixed
standard CAN 11-bit / 500 kbps profile, allowlist and conservative polling budgets
are documented in [the M6 report](../docs/MILESTONE_6.md); this checklist changes
scheduling, not the transmitted command set.

## Before going to the car

- [ ] Confirm the next approved retest build and read this entire checklist.
  Prepare a place for brief notes; no CI or Apple-account work is needed during the visit.
- [ ] Have the paired MX+, phone and any already available reference tool ready.
  If a reference is unavailable, mark that comparison pending; still collect the
  other cases during this visit.
- [ ] Plan suitable parked engine-running conditions. M6 does not authorize a
  road test. Close competing scanner apps; use one diagnostic client at a time.

## Controls and evidence conventions

Use the **Stationary recording** section. Enable **Vehicle is parked**. Set the
operating-state picker and workload before each Start; Stop and save before
changing either. These are separate recordings during the same visit, not separate
appointments. Note each case's start time (or add its case ID to the optional
reference note) so its saved session/ZIP can be identified afterward.

Run each short recording for about 45 seconds on the app timer, allowing setup to
finish. Capture repeated readings/age updates where supported; a retained number
alone does not prove fresh acquisition. Initialization and discovery repeat on a
new connection. Exporting each short recording immediately is unnecessary.

## Connected checklist — complete in order

| Case | Operating state / workload | Procedure and visible result |
| --- | --- | --- |
| M6-01 | Ignition on, engine off / Adapter voltage only | Launch and connect; note pre-existing dashboard warnings. Start, allow initialization, observe voltage/age for about 45 s, then Stop/save. Setup should complete and readings should update without a new warning |
| M6-02 | Ignition on, engine off / RPM only | Start a new 45 s recording. Note whether RPM is observed as zero, explicitly unsupported/unavailable, or not yet known. A missing reading must not be described as fresh zero. Stop/save |
| M6-03 | Engine running, stationary / Adapter voltage only | Start the engine between recordings, choose the matching operating state, then record voltage alone for about 45 s. Stop/save; retain this same-state individual workload for comparison with combined acquisition |
| M6-04 | Engine running, stationary / RPM only | Record about 45 s. Where supported, RPM should be observed with updating age; retain the actual responding ECU labels. Stop/save |
| M6-05 | Engine running, stationary / Coolant only | Record about 45 s. Where supported, coolant should be observed in the displayed unit with updating age; a steady value is valid. Stop/save |
| M6-06 | Engine running, stationary / Combined | If the gate below passes, record for 5 min. At about 90 s, tap Mark Incident once; continue to 300 s (5 min), then Stop/save. This allows the configured 60 s pre-window and 120 s post-window without a disconnect in the clean capture |
| M6-07 | Engine running, stationary / Combined | Start a separate 2–3 min recording, allow setup and about 30 s of readings. Unplug the scanner, note the disconnect, then replug/reconnect through iOS Bluetooth if needed. Allow fresh initialization and about 30 s of recovered readings, then Stop/save. Note whether recovery required Discover / Connect or an app restart |

### Continue gate — no chat response required

After M6-01, continue the short cases in this visit if setup completes, voltage
readings update and no new vehicle warning or abnormal behavior appears. Before
M6-06, the short cases must show stable transport/setup and their actual coverage:
required available readings update; explicitly unsupported parameters are noted
and are not a reason to invent values or repeatedly retry them. Engine-off absence alone does not block the listed engine-running checks. If an
expected supported engine-running reading remains unavailable/unknown, record it
and skip the dependent combined test pending analysis. Do not treat visual plausibility as decoder or
reference acceptance; exports will be independently checked afterward.

A successful M6-06 permits the listed M6-07 reconnect check during this same visit.
The gap in M6-07 is expected evidence; it does not establish continuous coverage.
Initialization after reconnect takes additional time beyond Bluetooth attachment.
If automatic recovery fails, note that failure; one Discover / Connect attempt
can document manual recovery, but does not count as automatic recovery passing.

If setup fails, recording cannot be saved, new dashboard warnings appear or the
vehicle behaves abnormally, stop transmission and preserve what was saved. Skip
the dependent cases and finish the handoff indoors. If an incomplete session
cannot be exported, note that result and retain its original app files. Do not
wait in the car for a
new build or improvise additional vehicle commands.

## Reference comparison in the same visit, if prepared

- [ ] After stopping the recorder, collect comparable readings from the prepared
  trusted reference, sequentially rather than running competing scanner clients.
- [ ] Record reference tool/version, time, units and operating conditions for
  voltage/RPM/coolant where available. Adapter voltage is not automatically the
  same measurement as an ECU supply or battery-terminal voltage.
- [ ] Record changed conditions or timing limitations; no available reference
  means the comparison remains pending, without repeating unrelated passed cases.

## Indoors afterward — scanner connection unnecessary

- [ ] Open saved sessions and match the seven case IDs to their start times/ZIPs.
  Mark any skipped or interrupted case honestly; do not silently omit it.
- [ ] Open the M6-06 manual incident. Check its capture status/window and that
  **View evidence around trigger** scrolls to the canonical timeline.
- [ ] Export the saved recordings individually to Files. Check the share sheet and
  actual retrieval here; the build has no bulk-export control. Collect the ZIPs
  together for one handoff, with screenshots only for failures/unexpected results.
- [ ] Send the case results and collected files together. No per-case chat reply
  or upload is required during the connected block.

Minimal result sheet (start time can substitute for an unavailable session ID):

| Case | Pass / fail / not run | Start time or ZIP | Notes: warnings, missing readings or recovery action |
| --- | --- | --- | --- |
| M6-01 | | | |
| M6-02 | | | |
| M6-03 | | | |
| M6-04 | | | |
| M6-05 | | | |
| M6-06 | | | |
| M6-07 | | | |
| Reference | | | Tool, conditions and units; or unavailable |

## Agent review after the visit

Validate original ZIPs/checksums, all six live files, raw/transaction provenance,
actual observations/ECU identities, requested versus achieved rates, gaps and
reference notes. Reconstruct M6-06's manual incident; inspect M6-07 for a fresh
connection generation, repeated setup/discovery and no stale request replay.
Visible recovery cannot by itself prove those raw invariants. Report incomplete
coverage or unsupported features explicitly and return one consolidated result.

If fixes are needed, prepare one new tested build and a targeted retest packet
covering the failed/affected cases plus a relevant regression check. Preserve the
accepted cases whose code/configuration is unchanged. M6 remains partially complete
until this physical evidence is accepted; **Milestone 7 has not started**.
