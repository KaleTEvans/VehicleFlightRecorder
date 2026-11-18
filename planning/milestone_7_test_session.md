# Milestone 7 — consolidated physical acceptance

Status: PREPARED, NOT RELEASED — November 18, 2026. Native verification and signed
beta delivery are in progress. Do not use M6 build 13.1 for these cases. A tested
M7 build/source and delivery result must be inserted here before requesting a
visit. No car visit is requested by this draft packet.

The milestone has three substantial blocks, with an analysis hold before road
use. Complete desk/native combinations first; collect parked cases in one visit,
then review history/export indoors. Batch defects and request only affected
retests. Reuse accepted M6 baseline/reference data rather than recording it again.

## Block A — indoors, before the connected visit

- Install the exact M7 beta listed in the released packet. Record app build,
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
   waiting and **no recorded drive / success confirmation**. Valid zero RPM is
   acceptable; missing RPM is unknown and must not be shown as fresh zero.
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
4. Close/reopen the app while this completed-drive override is active. It should
   remain suppressed. This checks persisted override, not interrupted recovery.
5. Turn the engine off with ignition on if normal/appropriate. Allow **75 seconds**
   of valid zero RPM to clear the override. If the ECU stops replying instead,
   leave the state unknown and record it; explicit disable/re-enable is available
   while parked. Do not claim zero-based end acceptance from missing responses.

### B3. Background and locked operation, separate observations

Proceed only if B1/B2 acquisition and ownership behaved correctly:

1. With monitoring armed and no drive active, put VFR in the background while
   another app is visible. Start the engine normally. Wait about 90 seconds,
   then reopen VFR. Record whether it started without opening VFR/pressing Start,
   whether fresh data exist, and whether a single quiet notification was observed.
   No notification is required if the preference/permission is disabled; actual
   recording remains a separate check.
2. During this drive, lock the phone for **90 seconds**, then unlock/reopen.
   Record actual elapsed time and fresh coverage/gaps. Lock is not proof that iOS
   suspended the app. Do not label it as an OS-suspension test.
3. Let the engine-off policy end this drive normally where valid zero RPM remains
   observable; allow at least 75 seconds. If it loses replies, record unknown-
   activity grace (up to 180 s), not a successful validated-zero end.
4. With the phone already locked and monitoring armed, start another normal parked
   engine session. Wait about 90 seconds, then unlock. This is **start while locked**,
   distinct from locking after a drive already started. Record coverage and
   confirmation observations. Notification tap should open this active drive.
5. End this drive normally; a tap on the old notification afterward should show
   its own saved summary, not start a new drive or open an unrelated drive.

### B4. Force-quit/relaunch and adapter availability

1. Once another automatic stationary drive is accepting data, deliberately
   force-quit VFR, wait 30 seconds, then relaunch. The old recording must remain
   visibly incomplete/interrupted. Fresh activity may start a **distinct** drive;
   it must not claim continuous coverage or append repaired rows to the old files.
   This is user force-quit, not OS termination.
2. After initial setup, repeat app launch with adapter already connected versus
   scanner availability after launch. Record which needed user recovery. These
   can be observed during B1–B4; do not create separate trips for them.
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
