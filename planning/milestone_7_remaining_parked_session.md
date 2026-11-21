# Milestone 7 — remaining parked tests

**READY — November 20, 2026. Use TestFlight Vehicle Flight Recorder 0.1.0 (20.1).**
One parked visit, approximately **30–40 minutes**, followed by one diagnostic ZIP
and one results note. No new build or upload is required for this checklist.
This releases only the remaining parked observations; road testing stays on hold.

The delivered app source is `19e83737ef623014c076dbc8f18d24a65efe0604`;
[delivery record](../docs/MILESTONE_7.md#approved-201-testflight-delivery--november-20-2026) and
[accepted recovery results](../docs/MILESTONE_7_RECOVERY_ACCEPTANCE.md) retain
its provenance. If TestFlight shows a different build, record that version and
confirm its matching checklist before using it as acceptance evidence.

Do **not** repeat the accepted full incident window, responder-display check,
ordinary reconnect, interrupted-drive preservation, or export troubleshooting.
The starts and final export below support new observations rather than retesting
that baseline. Button styling and the source-only parked-display correction are
not acceptance items for this visit.

## Before starting the timed block

Keep the car parked in a ventilated location using normal engine operation.
Close OBDLink and other scanner apps. Use the tested **CAN 29-bit / 500 kbps**
profile and the ordinary automatic workload; do not select a diagnostic or manual
stationary recording. No enhanced Alfa/TCM commands or new protocol experiments.

1. Confirm build **20.1**. Record iPhone/iOS, adapter firmware, starting battery,
   charging state, Low Power Mode, Focus and approximate local start time.
2. With the engine off, open VFR and turn **Hands-off monitoring OFF**. If an old
   recording is active, save it first and label it setup, not a checklist drive.
3. Turn **Vehicle is parked ON**. On 20.1 this switch can appear off and disabled
   after reopening with monitoring enabled: turn monitoring off first, then set
   parked on. This known display issue does not require another defect report.
4. In **Optional recording confirmation**, turn **Recording confirmations ON**
   and tap **Allow quiet notifications**. Resolve the system permission before
   starting. The app should show permission **authorized**. If previously denied,
   use **Open notification settings** and return to VFR. Arrange for immediate
   visible notifications rather than Focus/scheduled suppression for this block.
   If you prefer not to grant permission, continue acquisition tests and mark
   notification display/tap cases **not observed**, not passed.
5. Enable **Hands-off monitoring** with the engine still off. Do not press
   **Start stationary recording**. Start the following sequence without
   per-case messages or exports.

## Phone quick list

| Case | Action | Record |
| --- | --- | --- |
| L1 | Foreground start, then engine off for up to 4 minutes without Stop | One in-app confirmation; drive saves automatically; delay |
| L2 | Start with another app visible; wait 90 s; tap active-drive notification; lock 2 minutes | Started before returning; correct active drive; coverage after unlock |
| L2 end | Engine off; leave VFR behind another app for 4 minutes | Whether it saved before reopening; any execution/data gap |
| L3 | Start with phone already locked; wait 90 s; keep notification untapped | Started before unlocking; one quiet notification |
| L3 end | Open VFR via its icon, engine off, wait up to 4 minutes; tap retained notification after save | Notification opens that ended drive, creates no drive |
| L4 | Monitoring off; confirmations off; scanner unplugged; rearm, start engine, plug/pair scanner | Later availability starts automatically without Discover or a new confirmation |
| L4 finish | Stop/save; force-quit 30 s; reopen and watch 30 s | Completed-drive Stop override survives relaunch; no fresh drive |
| Handoff | Disable monitoring, note battery/context, export once | One M7 diagnostic ZIP plus L1–L4 observations |

Detailed steps below define the prerequisites and what a pass means. Each drive
is comfortably below the existing **30-minute cap**. There is no 30-minute idle
or battery-endurance requirement in this visit.

If a start has not produced fresh RPM within its 90-second window, record the
state/error and stop its dependent notification/lock/end checks. Do not substitute
a manual recording, another protocol or repeated retries. A persistent failure
ends the visit: disarm and return the evidence collected so far. A missing notice
alone does not stop otherwise healthy acquisition tests.

## L1 — foreground confirmation and automatic ending

1. Keep VFR visible and start the engine. Allow up to **90 seconds** for an
   automatic drive with fresh readings. Note the approximate engine-start and
   first-recording times. A confirmation should say **Drive recording started**
   once, without requiring a notification tap. Do not mark another incident.
2. Once readings are arriving, turn the engine off normally. Leave monitoring
   enabled and VFR visible. **Do not press Stop and save**, toggle monitoring,
   force-quit, or export during this automatic-end observation.
3. Allow up to **4 minutes**. The recording should save once and return to
   waiting/no active drive. Note engine-off and observed-end times, any red error,
   and whether RPM actually displayed zero before it became stale/unavailable.

There are two distinct paths: repeated fresh **0 RPM** spanning 60 seconds can
end a drive; if replies disappear, approximately **3–4 minutes** allows freshness
expiry and the 180-second unknown-activity grace. The latter is an end caused by
lost activity evidence, not proof of sustained zero RPM. We identify the actual
path from the bundle. You do not need to recognize raw replies yourself.

If it has not ended after 4 minutes, note the state and delay, then Stop/save and
disable monitoring to preserve evidence. Do not repeat engine cycles to force a
pass. If otherwise healthy, explicit off/on rearm may prepare the independent
later cases; label the failed end and intervention. Stop the visit for persistent
acquisition errors or new vehicle warnings.

## L2 — background start, active notification, lock continuity and background end

1. Begin engine off, no active drive and monitoring enabled. If a prior manual
   Stop left an override, disable/re-enable monitoring once while parked and
   note this preparation. Return to **Adapter Transport** after any history view.
2. Put a neutral app such as Notes in the foreground; **do not force-quit VFR**.
   Start the engine while VFR remains behind that app. Wait **90 seconds**.
3. If its **Drive recording started** notification appears, tap it while the
   engine is still running. It should open **Active drive**, with elapsed time
   and fresh readings for this recording. Note the notification time and any
   duplicates. Return with Back to Adapter Transport for the remaining controls.
   If no notice appears, reopen via the app icon and record that absence.
4. With this drive running, lock the phone for **2 minutes**. Unlock and reopen
   VFR. Note whether the same drive remains active, elapsed time, fresh readings,
   any error and any duplicate confirmation. Logs will establish actual coverage;
   a moving timer alone does not establish continuous acquisition.
5. Turn the engine off, then leave Notes visible for **4 minutes** without
   touching VFR. Reopen VFR and note whether the drive has already ended. If it
   ends only on return, report that timing rather than calling background end
   successful. A visible problem can use the L1 evidence-preserving fallback.

We compare engine-start, app-return and recording timestamps to distinguish a
background start from a start triggered only by reopening. A submitted but unseen
notification does not qualify physical display/tap behavior.

## L3 — start while already locked and notification for an ended drive

1. Begin engine off with monitoring armed and no active drive/override. Keep
   recording confirmations on. Lock the phone **before starting the engine**.
   Start normally, wait **90 seconds**, and note any quiet start notification.
2. **Leave that notification untapped.** Unlock, then open VFR via its app icon.
   Note approximate lock, engine-start and unlock times; record whether a drive
   was already running and its elapsed time. If it starts only after unlocking,
   report that limitation. The bundle will verify the start boundary.
3. Return to Adapter Transport, turn the engine off, and leave VFR visible for
   up to **4 minutes** to let this drive save automatically. Do not Stop/save
   unless taking the L1 fallback. Remember its start time for matching history.
4. **After it has saved**, open Notification Center and tap its retained start
   notification. Expect that drive's saved details, not Active drive, an unrelated
   recording or a new drive. If the notice is missing, record **not observed**;
   do not repeat this entire case merely to obtain it.

Locking after a start (L2) and starting while already locked (L3) are separate
observations. Neither alone proves OS suspension, OS termination or automatic
launch after a user force-quit.

## L4 — later accessory availability, disabled confirmations and persisted Stop

1. Engine off and no active drive: turn monitoring **OFF**, then **Recording
   confirmations OFF**. Unplug the scanner using the previously accepted parked
   procedure. Leave VFR visible, confirm parked and turn monitoring **ON** while
   the scanner is unavailable.
2. Start the engine normally, then plug in the scanner and reconnect Bluetooth
   if needed. **Do not tap Discover / Connect.** Allow up to **90 seconds after
   Bluetooth reconnect** for automatic recording and fresh readings. Note any
   manual action actually needed. No new confirmation for this drive should
   appear, although older notices can remain in Notification Center.
3. Tap **Stop and save** once while the engine is still running. Force-quit VFR,
   wait **30 seconds**, reopen, and observe another **30 seconds**. The manual
   Stop override should remain active with no new drive. This is persistence
   after a completed drive, distinct from the already accepted active-drive
   interruption case. Do not wait for zero-RPM clearance or toggle monitoring
   during this observation.
4. Turn monitoring **OFF** and turn the engine off normally. Keep the app/files
   intact. No more engine starts or scanner cycles are requested.

## One handoff, indoors

Record ending battery, charging changes, approximate screen/locked time, thermal
or Low Power context and local finish time. Use **Export M7 diagnostic bundle**
once and save the actual ZIP to Files; send it with the note below. The accepted
export flow is a collection step, not another repeat-sharing test. Restore your
preferred notification/Focus settings afterward. No OBDLink comparison is needed.

If export fails, record its visible error; preserve the app's **Automation,
Sessions and ConnectionCaptures** folders through Files and compress them
together. Do not reinstall, delete originals or repeatedly power-cycle to recover.

Copy/paste this note; PASS / FAIL / NOT OBSERVED is enough where timing is unclear:

```text
Build / iPhone / iOS / adapter firmware:
Date / local start–finish / time zone:
Battery start–end / charging changes / Low Power / thermal context:
Notification permission / Focus or delivery restrictions:
L1: engine start __ / recording __ / one in-app confirmation __
    engine off __ / automatic save __ / zeros vs unavailable if visible __
L2: engine start __ / notification __ / tap active drive __
    locked __ to __ / same drive + fresh values __ / errors or duplicates __
    engine off __ / return __ / already saved or ended only on return __
L3: locked __ / engine start __ / notice __ / unlock __ / recording elapsed __
    engine off __ / saved __ / retained-notice tap opened correct summary __
L4: scanner available __ / automatic recording __ / Discover needed? __
    confirmations remained off __ / Stop override survived relaunch __
Any failed/omitted step, fallback/manual rearm, warning or red error:
ZIP filename:
```

## Agent review and remaining release boundary

Retain the unchanged ZIP privately, check archive integrity/bounds and capture
provenance, validate the whole automation bundle and each capture with explicit
incomplete inspection, and map only new captures to L1–L4 using the note. Reuse
previously accepted incident, reconnect and recovery evidence.

Report actual start/end reasons, app-state timing, freshness/gaps, notification
selection/submission versus owner-observed display/taps, and resource populations.
Foreground/context observations here supplement earlier accepted foreground
acquisition. Do not count missing notifications as delivery passes or a start
after foreground return as a background pass. Review retained interrupted
originals without treating older incomplete captures as new failures.

No further sustained-zero clearance attempt is requested: that condition remains
unqualified if this vehicle stops replying too early. Native fixtures cover the
policy; physical zero-end acceptance and unknown-grace end are recorded separately.
Do not infer OS suspension/termination from lock or background and do not ask the
owner to induce OS termination. Record unobserved states and supported product
limitations explicitly; a claimed supported state still needs evidence.

After this one handoff, resolve any required supported-state failures and review
the parked evidence before issuing a separate controlled-road checklist. That
later block covers short/repeated representative drives, measured storage/queue/
heap behavior, battery context and the initial 30-minute limit. This short parked
visit supplies a resource baseline, not battery/endurance acceptance. M7 remains
open and M8 is not started.
