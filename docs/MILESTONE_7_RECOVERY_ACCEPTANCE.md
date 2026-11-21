# Milestone 7 — 20.1 parked recovery review

November 20, 2026. The owner completed the consolidated recovery block on
**0.1.0 (20.1)**. Ordinary start/display, manual marking, Stop override, fresh
relaunch with preserved interruption, scanner reconnect and retrieved diagnostic
export pass within the observed parked scope. M7 remains open; road and broader
lifecycle/notification/resource qualification are not accepted by this review.

## Retrieved originals and independent checks

The original M7 diagnostic ZIP and unchanged extraction are retained privately in
ignored `.cache/physical-tests/2026-11-20/m7-20.1/`. No phone originals are committed.

- ZIP: 4,784,621 bytes, 297 entries; SHA-256
  `dcb050924a45b4d45772db6876b956c4518b3b978e3be8fea321e6c92c6fa8b5`.
- CRC, duplicate/path/symlink and archive bounds checks pass.
- All **37 retained M7 captures** independently validate with explicit incomplete
  inspection: nine automatic drives and 28 probes, including older 14.1 evidence.
  Whole automation state/journal/claim provenance validates with nine claims.
- The 20.1 configuration identifies **five new drives and ten new probes**.
  New drives contain **563 observations**: four complete and one preserved as
  incomplete after force-quit. Original interrupted bytes remain unchanged.
- Scanner loss/recovery retains one drive ID and advances to generation two.
  Its explicitly logged coverage gap is **38.715 seconds**; no continuous-data
  claim is made for that interval. Full setup/support precede fresh recovery.
- New lifecycle/accessory diagnostics are present. An export preparation result
  and the later export request are retained, and the owner retrieved this valid
  ZIP through the app. Notification selection remains disabled; no delivery or
  lock/wake acceptance is inferred from lifecycle callbacks alone.

## Recovery results

| Case | Result and boundary |
| --- | --- |
| R1 automatic start/readings | Owner confirms automatic start, one RPM/coolant/voltage row and All responders disclosure. New drives contain independently decoded physical readings. |
| R2 incident feedback | Owner confirms mark acknowledgement; the new manual window is Partial as allowed by the short test. Prior full-window acceptance is reused. Button styling refinement is deferred to the later UI work. |
| R3 Stop override | Owner confirms no restart during 60 s with the engine running. Probe evidence continues without duplicate drives until explicit rearm. |
| R4 force-quit/relaunch | One old drive remains well-formed incomplete; journal identifies it as interrupted, and a distinct drive starts after fresh running evidence. Parked setup display issue recorded below. |
| R5 scanner reconnect | Owner confirms recovery; one drive retains its identity, advances generation, repeats setup and resumes fresh values with an explicit gap. |
| R6 engine-off/rearm | Explicit off/on rearm works. Sustained-zero automatic override clearance remains unqualified; this capture lacks the required zero duration. |
| R7 export | App diagnostic ZIP retrieved and independently validated. No further Files fallback/export is requested for this visit. |

## Why 75 seconds engine-off did not clear the override

A zero-RPM reply means the car actually returned an RPM reading whose value was
zero. `NO DATA` means no usable RPM reading arrived; it leaves activity unknown.
The existing versioned policy requires fresh zero readings spanning 60 seconds
before it clears a manual Stop override. It does not use elapsed time with the
engine switched off as a substitute for evidence.

The relevant probe contains seven successful zero-RPM response frames (14
responder observations) spanning **15.636 seconds**, then five `NO DATA`
transactions. Running replies resume at about 97 s into that probe. The
60-second zero condition was not met, so preserving the override matches current
policy. Explicit off/on rearm was successful. No policy thresholds or missing-data
interpretation are changed by this review, and no additional car visit is needed
to explain this result.

## Parked setup display correction

`Hands-off monitoring` is restored from the Go owner's persisted preference;
`Vehicle is parked` was a screen-local confirmation reset to false on relaunch.
The disabled unchecked switch therefore misrepresented the enabled validation
setup. It did not stop the owner's successful fresh recording.

While monitoring is enabled, the UI now shows **Parked validation mode is enabled**
and explains how to change setup. It does not assert the car is currently parked.
When monitoring is disabled, a fresh editable parked confirmation is required
before rearming. No parked declaration is silently persisted or inferred from
Bluetooth/RPM. Manual-override help now explains fresh zero readings and explicit
rearm. Native UI coverage exercises the reported restored on/local false state,
disable/confirm/rearm and relaunch presentation. This display fix is not in the
already installed 20.1 binary and can travel with the next grouped beta.

## Work sequencing and remaining gates

[The updated CI workflow](CI_WORKFLOW.md) adds validated targeted native runs,
compiler/dependency caching, push-only code CI with Markdown-only exclusion,
protected publication and cancellation of superseded verification. Local script
checks and workflow lint precede targeted native validation; final full native
acceptance follows on the frozen app source. Native results are recorded below. No new IPA upload or another car visit is requested here.

Retain these physical passes and the M6 baseline. Remaining M7 work includes
normal sustained-end qualification where available, deliberately labeled
background/lock/wake/notification cases, and representative road/resource/battery
qualification after the applicable release gates. One parked recovery visit
does not close M7 or start M8.


## Verified implementation and CI behavior

Frozen code source: `0f51b2ae2df6b7eb48ed3a129f68cde22bc17330`.
The application view implementation remains unchanged from `28d3fc5`; the
follow-up corrects only the switch test interaction and CI evidence/cache handling.

- Local verification: **40 script tests**, workflow lint and Markdown link checks.
  No Go/activity-policy code changed. Portable CI passes Go tests/vet, **185 Python
  tests** and the 40 script tests, once for the code push.
- Focused run [44839886220](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/44839886220)
  passes the restored parked setup/rearm/relaunch UI case. Its machine-readable
  record declares `scope=targeted`, exactly the requested case, and
  `full_acceptance=false`. The five existing model tests also passed in the earlier
  focused attempt; a row-label tap in its new UI test was corrected to target the
  actual switch control and verify each changed value.
- Targeted artifact `14681350205`, SHA-256
  `c03348c31e0ad195b9ed1193c47592c1c1a21ea3ab4190b6f9c2231d2ae73035`,
  was downloaded with CRC/digest verification; its restored-mode screenshot was
  reviewed. Targeted runs skipped device acceptance and preview publication.
- One final full run [44840987248](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/44840987248)
  passes **46 hosted tests, 10 UI tests, both SDK profiles, 23 native ZIPs and
  four automation bundles**. Its record declares full scope and 56 successful
  cases. Independent local replay of the downloaded exports reconstructs all
  available incidents.
- Full artifact `14681118107`, SHA-256
  `b24e386d398a6e0ce00d506045c4c47dad2677625f6f0c8ea159cf947fd415db`,
  passed download CRC/digest checks. Evidence is retained under ignored
  `build/milestone-7/ci-full-0f51b2a/`.
- Full preview restored the exact pinned Go compiler/module cache saved before
  the focused UI run. The separate upload-compatible SDK build passed too.
- Repository run history confirms one portable code run for the frozen code
  commit, with no duplicate PR test run. Documentation-only follow-up `6ec850d`
  triggered no code workflow, while the full run continued on its original
  frozen source. Failed-test screenshots are collected independently of test
  success for future focused diagnosis.

These workflow changes are available on the M7 branch and recorded in
[AGENTS.md](../AGENTS.md) for future work. No new signed candidate or Apple upload
was needed for this source-only correction. The installed TestFlight build
remains 20.1; include the display fix in the next grouped beta. Physical
recovery passes are retained; the remaining M7 gates are unchanged.

## Next parked checklist prepared — November 20, 2026

At the owner's request, the [remaining parked checklist](../planning/milestone_7_remaining_parked_session.md)
is ready for the existing **20.1** build. It groups automatic ending, background
and locked starts/coverage, active/ended notification taps, later scanner
availability, disabled confirmations and completed-Stop persistence into one
30–40 minute visit and one ZIP/note handoff. No new IPA, repeat recovery block or
separate UI car test is needed. New observations remain pending; road use and
representative resource qualification stay on hold until parked review.
