# Milestone 7 reliability and acceptance matrix

November 20, 2026 — first 14.1 parked evidence reviewed. Ordinary automatic
acquisition and a full manual window validate; startup/reconnect and export
failures keep M7 open and road use on hold. See the
[physical review](MILESTONE_7_PHYSICAL_REVIEW.md). Targeted fixes require a verified
replacement beta before one consolidated follow-up. M6 acceptance is retained.

| Case | Implemented behavior / evidence | Physical status |
| --- | --- | --- |
| Ready scanner, ignition on / engine off | Separate RPM probe; no drive or success confirmation; Go and independent replay fixtures | User observed pass on 14.1; no scanner-only drive |
| Stationary engine running | Repeated fresh RPM starts one drive; actual admitted telemetry produces one durable fact | Still pending unless qualified below |
| One responder / missing TCM | Any validated running responder can qualify; no mandatory TCM/dual-ECM assumption; coverage limited | Deterministic policy scope; no physical distinct-module qualification |
| Brief stop/start | Fresh zero requires 60 s span; brief zero/dropout cannot end drive | User observed recovery after about 15 s stop |
| Sustained end | One finalization; pending incident becomes partial if post coverage ends | Pending |
| Brief adapter loss | Retain logical drive within 180 s unknown grace; new generation, setup/support and fresh values | Brief reconnect observed; later startup/reconnect failures remain open |
| Stale / missing RPM | Unknown rather than zero; bounded grace; recorded gap | Fixtures only |
| Manual Stop | Save once; persisted override until sustained off or explicit rearm; monitoring continues while armed | Pending |
| Foreground | Process-owned transport and independent owner timer | Pending |
| Other app foreground, VFR background | Existing external-accessory mode; record observed scene/coverage | Pending; no continuity claim from declarations |
| Screen lock / unlock | Continuous-clock deadlines; explicit observed scheduling gap | Pending; locked is not automatically suspended |
| Previously launched then OS-suspended | No fake backfill; long execution gap ends old logical drive | Untested OS state; do not relabel background as suspension |
| OS-terminated | Preserve originals; fresh-evidence new-drive policy on actual relaunch | Untested; forced termination is not OS termination |
| User force-quit | No promised wake/restart; deliberate relaunch preserves interrupted original and starts distinct drive only when eligible | Pending; unavoidable reopen limitation must remain explicit |
| Cold launch, adapter already attached | Process-owned restore arms persisted policy; no UI task required | Pending |
| Adapter becomes available later | Existing accessory observer/backoff plus owner policy | Pending |
| Relaunch during active recording | Never append/repair/adopt prior live files; journal identifies interrupted ID and boundary | Go fixtures; physical relaunch led to startup failure, recovery open |
| Notification authorized / denied / not determined / disabled | Quiet adapter behind fake seam, explicit permission-only setup, no recording gate | Native fake tests in CI; physical notification presentation pending |
| Notification reconnect / relaunch / failure / uncertain submission | Go durable claim plus native exclusive receipts; no retry after uncertainty | Native/Go tests; physical pending |
| Tap active / ended / missing session | Original ID only; re-resolve current state; active dashboard or exact saved summary/error | Native route resolver and saved-history UI passed; actual notification taps pending |
| Low space, write failures, truncated output | Stop admission, preserve originals; strict reader rejects malformed rows; explicit incomplete inspection only | Controlled storage/reader fixtures; no destructive phone fault requested |
| Heap, raw/file growth, native queues | Periodic observed resources; fixed queue/log/report/archive bounds, no retention deletion | Phone measurement pending |
| Short road / repeated / representative long drive | Parked analysis hold first; initial duration cap 30 minutes | Not released for road use; budgets pending |
| Battery / storage acceptance | Record context and observed per-hour populations; charging-only is inadequate for battery acceptance | Unmeasured; no budget claimed |

## Acceptance boundaries

The initial backend policy configuration uses a 30-minute drive cap and 5-minute
probe captures. Reaching the drive cap saves the drive and applies an override;
it must never silently start duplicate 30-minute chunks while the engine runs.
Longer continuous supported duration requires measured budgets, a versioned
configuration change, regression/native verification and a consolidated follow-up.

A physically untested supported state remains an open acceptance gate. Inherently
unsupported wake/force-quit behavior must be documented with required user action.
An observed silent gap cannot be labeled healthy background continuity. Repeated
normal logs and actual retrieved bytes are required before M7 can close.

## Result collection

Use the [consolidated packet](../planning/milestone_7_test_session.md). Record actual
build, device/OS, adapter firmware, configuration, planned/actual durations,
foreground/locked time, charging/battery/temperature context, visible warnings,
notification observations, required recovery and original bundle checksum.
Attach all visit evidence together; perform history/export and analysis indoors.

**Milestone 8 not started.**
