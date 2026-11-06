# Milestone 6 — targeted discovery diagnostic visit

Status: DRAFT pending native acceptance and signed TestFlight delivery. Do not
start this packet until its delivered build number is recorded below.

Build: pending. This packet supersedes the paused general M6 checklist for the
next visit. Build 9.1 history labels and single-tap/repeat export have human
acceptance; do not repeat those standalone tests. The two November 3 zero-sample
ZIPs are already received and validated. No road test is included.

Plan **one 15–25 minute parked visit**, including optional continuation cases.
Stop and save between recordings. No chat response, per-case upload or waiting
for a new build is required during the visit. History/export and one evidence
handoff happen afterward indoors.

## Prepare indoors

- [ ] Update to the delivered build listed above. Keep this checklist available.
- [ ] In the official OBDLink app, prepare standard **Engine RPM** and **Engine
  coolant temperature** gauges. Adapter voltage alone does not establish ECU
  communication. Gauges can be configured while disconnected: Dashboards →
  three dots → Add Display, or press/hold an existing gauge → Display
  Configuration to choose its PID. Select standard/SAE readings, not enhanced
  manufacturer signals. [Official gauge instructions](https://support.obdlink.com/support/solutions/articles/43000678883-add-and-edit-dashboard-gauges).
- [ ] Have a place for short notes and reference screenshots. Record any change
  from iPhone 16 Pro / iOS 26.6.2 / MX+; note the OBDLink app version.
- [ ] Leave coding, calibration, DTC clearing and factory resets out of this test.
  Use one OBD client at a time, as [the manufacturer requires](https://support.obdlink.com/support/solutions/articles/43000715570).

## In the car — run in order

Keep the vehicle parked, parking brake set, engine idling in suitable ventilated
conditions. Note pre-existing dashboard warnings. If a new warning or abnormal
vehicle behavior appears, Stop/save and end the connected tests; retain evidence.
Do not operate these controls while driving.

### D1 — targeted diagnostic capture, 2 minutes

1. Close the OBDLink app and other scanner apps. Open Vehicle Flight Recorder and
   allow MX+ transport connection; use Discover / Connect once if needed.
2. Enable **Vehicle is parked**, select **Engine running / stationary**, then
   workload **Discovery diagnostic**. Optional reference note: `D1 before reference`.
3. Tap **Start stationary recording** once and leave the app in the foreground
   until its timer reaches **02:00**. Then **Stop and save**. The global 10-minute
   guard remains a limit, not the target duration for this test.
4. Note whether setup reaches **21/21**, the discovery outcome and attempt count,
   and whether adapter voltage, RPM and coolant appear with updating ages. Take
   one status screenshot near the end if convenient. CAN status is captured as
   reported; you do not need to interpret its counters.

`No data returned` is an informative result. RPM/coolant remain unknown and are
not polled without valid support evidence; adapter voltage can still be sampled.
An ordinary no-data reply does not require repeated Start/Connect taps. This
profile repeats discovery with conservative backoff (normally about 30 seconds
following no-data), so two foreground minutes capture multiple attempts.
If setup fails or the recorder errors/stops, preserve what exists and move to the
reference check only if the vehicle remains normal; do not keep retrying D1.

### R1 — independent standard-OBD reference, about 1 minute

1. Ensure VFR recording is stopped. Tap **Disconnect** in VFR, then close it.
2. Open the official OBDLink app and connect to the same scanner. Observe standard
   Engine RPM and Engine coolant temperature for roughly 30 seconds.
3. Save a screenshot with values/units, or the exact no-data/error result. Note
   the selected/detected protocol if the app displays it; do not guess or force
   alternative protocols to obtain a reading. A configured gauge that remains
   blank is a result to report, not a reason to select unrelated enhanced PIDs.
4. Disconnect and close OBDLink before returning to VFR. Sequential readings need
   not be numerically identical; note time and changing engine/coolant conditions.

### D2 — conditional diagnostic repeat, 2 minutes

**Only if D1 did not obtain both RPM and coolant**, repeat D1 once after R1, using
reference note `D2 after reference`. Leave the car/engine conditions unchanged.
Stop/save at **02:00**. This records whether the independent client or reconnection
changed the outcome. Do not add factory resets, custom commands or more repeats.

If neither diagnostic capture obtains the expected ECU readings, finish the visit
here and return the collected evidence indoors. R1 may also fail; that is useful
comparison evidence and does not establish a VFR decoder failure.

### C1 — conditional ordinary combined recording, up to 5 minutes

**Only if a diagnostic capture obtains both RPM and coolant and the vehicle
remains normal**, test the ordinary profile during this same visit:

1. Select **Combined**, Engine running / stationary, with note `C1 ordinary profile`.
   This uses the existing 16-step setup, not diagnostic setup.
2. Start and check that voltage/RPM/coolant ages update. If the expected ECU
   readings remain absent at **01:30**, Stop/save, note that difference from the
   diagnostic profile, and skip C2. Do not force an incident on a failed prerequisite.
3. If readings are present, tap **Mark Incident** once at about **01:30**. Continue
   to **05:00**, then Stop/save. This retains the 60-second pre / 120-second post
   evidence window in the same stationary recording.

### C2 — conditional reconnect, about 3 minutes

**Only after successful C1**, start a separate Combined recording with note
`C2 reconnect`. After about 30 seconds of readings, unplug/replug the scanner and
reconnect through iOS Bluetooth if required. Allow fresh initialization and at
least 30 seconds of recovered readings; stop/save by **03:00**. Record whether
recovery is automatic, requires Discover / Connect, or fails. Bluetooth recovery
and adapter initialization are separate waits. No repeated manual incident is
needed in this expected-gap capture.

## Finish indoors — one handoff

- [ ] Export every recording actually created: D1, optional D2, optional C1/C2.
  Use the workload/time/reference notes to identify them; keep originals.
- [ ] For successful C1, open its saved incident and evidence timeline. Note any
  failure or incomplete-window label; an export alone does not establish capture.
- [ ] Send all new ZIPs together, the R1 screenshot/error, and the notes below.
  Interrupted sessions remain original evidence: if ZIP export is unavailable,
  preserve their folder via Files → On My iPhone → Vehicle Flight Recorder → Sessions.

| Case | Setup/outcome | RPM / coolant / voltage observed? | Notes |
| --- | --- | --- | --- |
| D1 | 21/21? discovery outcome + attempts | | |
| R1 | official app outcome / protocol if shown | values + units or error | |
| D2 (if run) | same fields as D1 | | |
| C1 (if run) | ordinary 16-step setup; manual capture | | |
| C2 (if run) | disconnect/recovery result | | |

Raw-byte/provenance/reference review determines M6 acceptance afterward. A
successful diagnostic profile does not automatically qualify the ordinary
recording profile, and no road/background/M7 acceptance is claimed by this visit.
