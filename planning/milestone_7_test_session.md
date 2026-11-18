# Milestone 7 — consolidated physical acceptance

Status: RELEASED FOR PARKED TESTING — November 18, 2026.
Install **Vehicle Flight Recorder 0.1.0 (14.1)** from internal TestFlight.
Exact app source: `4d6e857b2cc6c8457c73892597d529e0e1becd9e`.
[Upload run 44421246901](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/44421246901)
sent the unchanged, user-approved signed IPA at **5:30:38 p.m. CST November 18**.
Apple confirmed upload COMPLETE, build VALID / internal IN_BETA_TESTING.
Blocks A/B and the indoor handoff below are released. **Block C / road use remains
on hold until these results are reviewed.** Do not use M6 build 13.1 for this packet.
Each automatic drive has a 30-minute cap; the visit can contain several drives.

The milestone has three substantial blocks, with an analysis hold before road
use. Complete desk/native combinations first; collect parked cases in one visit,
then review history/export indoors. Batch defects and request only affected
retests. Reuse accepted M6 baseline/reference data rather than recording it again.

## Phone quick list — one parked visit

Read preparation/stop criteria below first. Plan **35–45 minutes**; send one bundle
and one visit note after finishing. Observe timer/freshness/confirmations in the
app; drive IDs and transport generations will be checked offline from the bundle.

| Case | Action | Expected result |
| --- | --- | --- |
| P1 | Engine off / ignition on: arm monitoring, wait 60 s | No drive or success confirmation merely from scanner/module replies |
| P2 | Start engine while parked, allow up to 90 s | Preparation then automatic recording with fresh RPM/coolant/voltage; one in-app confirmation if enabled |
| P3 | After drive timer 01:00, Mark Incident; wait another 02:00 | Full manual pre/post window, subject to actual coverage |
| P4 | After P3, unplug/replug scanner and reconnect Bluetooth; allow up to 90 s | Recording timer continues and fresh data return; no duplicate start confirmation |
| P5 | Normal short engine-off interval, about 15 s, then restart | Same drive; no premature end or extra start confirmation |
| P6 | Stop/save while running, wait 30 s; force-quit/reopen | No immediate automatic restart; manual override persists |
| P7 | Let valid engine-off replies span 75 s; if no replies, use explicit disable/re-enable once and note it | Zero-based override clear only counts when fresh zero replies exist; otherwise clearance is unverified |
| P8 | With no active drive, background VFR, start engine, wait 90 s | Automatic start without reopening/Start; tap its notification while active if present; then lock 90 s and check continuity |
| P9 | End P8 normally. For a separate start with phone already locked, wait 90 s before unlocking | Fresh automatic capture; leave this notification untapped until the drive ends, then tap to open its own summary |
| P10 | During a new active drive, force-quit 30 s, relaunch with scanner attached | Previous drive preserved as interrupted; fresh evidence may create a distinct drive, never fabricated continuity |
| P11 | Disarm, unplug scanner, re-arm while parked, then plug/pair it without Discover | Later accessory availability starts capture automatically once fresh activity qualifies |
| P12 | Stop/save, disable monitoring; review/export indoors | One **Export M7 diagnostic bundle**, actual retrieved ZIP, visit note with delays/issues/battery/charging/Focus context |

For normal automatic end, expect about **75 s with ongoing fresh zero RPM**, or
roughly **3–4 minutes when RPM replies disappear**. Missing replies are not zero.
Stop for a new warning, missing RPM at the P2 gate, repeated recovery failure or
storage error and return the available bundle. No per-case chat replies are needed.

## Block A — indoors, before the connected visit

- Install **0.1.0 (14.1)**. Reconfirm **Vehicle is parked** after relaunch/rearming
  when required by setup controls. Record app build,
  iPhone model/iOS, adapter identity/firmware and selected CAN29/500 profile.
- Read optional confirmation explanation. Choose whether to enable quiet
  notifications; resolve permission and Focus/settings now, not while driving.
  Denied/disabled/error combinations are primarily covered by native fake tests.
- No GPS, enhanced Alfa signals or vehicle control commands are requested.
- Set aside approximately **35–45 minutes parked**, allowing for engine-off grace
  and actual observations. Distinct recordings do not require separate visits or
  chat replies. Save one note of case labels/outcomes/times and battery/charging
  context. Actual times may differ; no short session proves long reliability.

## Block B — one parked visit, no road use

Keep parked with the parking brake applied and use appropriate ventilation for
engine operation. Configure controls while parked. Stop acquisition and retain
files for any new dashboard warning, abnormal vehicle behavior, repeated adapter
failure, storage error or missing data. Do not induce ECU/module faults. Do not
continue a dependent case if its written visible prerequisite fails.

### B1. Ignition on / engine off, then automatic stationary start

1. Close OBDLink and other scanner clients. Select tested **CAN29 / 500 kbps**.
   Enable **Vehicle is parked**, then **Hands-off monitoring**.
2. With ignition on / engine off, allow about **60 seconds**. Expect activity
   waiting and **no recorded drive / success confirmation**. Probe values need not
   be displayed here; the bundle will distinguish valid zero RPM from missing
   replies. Do not infer running from scanner/module availability.
3. Start the engine normally, remaining parked. Allow up to **90 seconds** for
   activity evidence, debounce, a fresh recording connection and setup. Expect
   preparation followed by recording that accepts RPM/coolant/voltage. Record
   actual observed start delay. If it never starts or has no RPM, stop dependent
   cases and return the bundle; do not invent another protocol or diagnostic run.
4. Expect at most one in-app start confirmation if enabled. A permission issue
   must not stop the drive. Record displayed versus absent separately.
5. When the **drive** timer is past 01:00, Mark Incident once; allow another two
   minutes of recording. Do not count pre-start probe time as drive time.

### B2. Active reconnect, brief stop/start, manual override and normal end

Continue the same active drive after B1's full manual window completes:

1. Unplug/replug the scanner using the accepted safe M6 procedure. Reconnect iOS
   Bluetooth if needed. Allow up to **90 seconds** for streams/setup/fresh data.
   Expect the same drive, a new generation and a visible gap; no new recording
   confirmation. If recovery fails, record any required Discover action and return
   evidence rather than repeating tests independently.
2. A short normal engine-off interval (about **15 seconds**, if appropriate for
   the vehicle) followed by restart should keep the same drive. No new drive or
   confirmation should be created. Do not intentionally create vehicle faults.
3. While running/parked, **Stop and save**. Wait about 30 seconds: no immediate
   automatic drive restart. Monitoring remains armed; the manual override is shown.
4. Force-quit VFR from the app switcher, then reopen while this completed-drive
   override is active and the engine is still running. It should remain suppressed.
   This checks persisted override after a completed drive, not active interruption.
5. Turn the engine off with ignition on if normal/appropriate. Allow **75 seconds**
   of valid zero RPM to clear the override. If the ECU stops replying instead,
   leave the state unknown and record it. Explicit disable/re-enable while parked
   may rearm the remaining independent cases; note that automatic zero-based
   clearance was not demonstrated. Do not claim zero-based acceptance from missing
   replies. Reconfirm Vehicle is parked if the re-enable control requires it.

### B3. Background and locked operation, separate observations

Proceed only if B1/B2 acquisition and ownership behaved correctly:

1. With monitoring armed and no drive active, put VFR in the background while
   another app is visible. Start the engine normally. Wait about 90 seconds,
   then reopen VFR. Record whether it started without opening VFR/pressing Start,
   whether fresh data exist, and whether a single quiet notification was observed.
   If present, tap this notification while the drive is active: it should open
   this active drive. No notification is required if the preference/permission is disabled; actual
   recording remains a separate check.
2. During this drive, lock the phone for **90 seconds**, then unlock/reopen.
   Record actual elapsed time and fresh coverage/gaps. Lock is not proof that iOS
   suspended the app. Do not label it as an OS-suspension test.
3. Let the engine-off policy end this drive normally where valid zero RPM remains
   observable; allow about 75 seconds. If it loses replies, allow roughly 3–4
   minutes for freshness expiry plus the 180 s unknown-activity grace. Record that
   path separately, not as a successful validated-zero end.
4. With the phone already locked and monitoring armed, start another normal parked
   engine session. Wait about 90 seconds, then unlock. This is **start while locked**,
   distinct from locking after a drive already started. Record coverage and
   confirmation observations. Leave this second notification untapped; open VFR
   from its app icon instead, so the notice is available for the ended-drive check.
5. End this drive normally, then tap its retained notification in Notification
   Center. It should show its own saved summary, not start a drive or open an
   unrelated drive. If Focus/settings hide a notice, record that presentation/tap
   check as unobserved; logs distinguish submission from actual visibility.

### B4. Force-quit/relaunch and adapter availability

1. Once another automatic stationary drive is accepting data, deliberately
   force-quit VFR, wait 30 seconds, then relaunch. The old recording must remain
   visibly incomplete/interrupted. Fresh activity may start a **distinct** drive;
   it must not claim continuous coverage or append repaired rows to the old files.
   This is user force-quit, not OS termination.
2. The relaunch above tests an already attached scanner. Separately, disable
   monitoring, unplug the scanner, then re-enable monitoring while parked (and
   reconfirm Vehicle is parked if needed). Plug/pair the scanner without tapping
   Discover / Connect. Allow up to 90 seconds for fresh automatic capture. Record
   any required recovery. This later-availability check stays in the same visit.
3. After a force-quit, do not assume automatic wake. Record required reopening as
   a product limitation. OS termination and verified OS suspension remain separate
   untested rows unless their exact conditions are reproducible and evidenced.

## Return indoors — one evidence handoff

1. Stop/save any active recording, then **disable hands-off monitoring**. Note
   the end battery percentage, charging state, approximate screen/locked times,
   ambient/thermal context, warnings and actual durations/case outcomes.
2. Review saved history and the manual incident; verify the evidence button and
   share sheet. Completed drives use ordinary session ZIPs. Interrupted originals
   must not be labeled complete or silently repaired.
3. Use **Export M7 diagnostic bundle** once. Retrieve the actual ZIP outside the
   phone, not just the share sheet. It includes automatic drives, retained probes,
   journal/state/receipts and incomplete evidence. Send that bundle and the single
   visit note together. No per-case uploads or chat approval between cases.
4. Agent: retain the original privately, compute SHA-256, extract without editing,
   run `validate-automatic --allow-incomplete`, inspect all drives/probes and
   reconstruct the actual manual incident. Malformed/truncated rows must still
   fail validation; preserve originals and report them instead of trimming them.

If the bundle exceeds the 128 MiB archive cap, retain files via Files/Finder rather
than delete evidence. This is a visible resource limit, not permission to discard
raw data. The runtime 512 MiB total retention guard also preserves originals.

## Analysis hold — before any road block

Review B1–B4 actual phone evidence and all distinct lifecycle/notification results.
Confirm the permitted workload, source/build, actual start/end/reconnect delays,
manual window, coverage and resource growth. Record untested/unsupported states
and user recovery explicitly. A failing required supported state remains open.
Set disturbance, battery and storage budgets from actual representative baseline;
charging-only evidence cannot establish battery drain. No universal wake claim.

## Block C — controlled road, only after written release

Not runnable yet. Proposed sequence: one approximately **10-minute** controlled
drive, then at least two representative normal-use drives within the initial
30-minute cap, with actual durations/context recorded. Interact only while parked;
use a passenger for observations or inspect afterward. Keep the one-tap manual
incident available but do not require driver interaction while moving.

A representative longer continuous drive requires an agreed duration/resource
configuration from measured baseline, verified before release. Never equate a
10-minute drive with multi-hour reliability. Collect repeated/long evidence in
substantial blocks or suitable normal use. Compare readings sequentially with
OBDLink/another trusted tool under recorded comparable state/units, never competing
clients without explicit support. Return all exports and notes together.

The final matrix must cover actual foreground/background/locked/suspension/
termination/relaunch cases separately, rates/latencies/stale/gap populations,
reference limitations, battery/charging/storage budgets and no missing/corrupt
normal-operation sessions. M7 cannot close on mock or compiler results alone.

**Milestone 8 not started.**
