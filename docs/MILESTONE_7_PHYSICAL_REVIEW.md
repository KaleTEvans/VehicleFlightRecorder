# Milestone 7 — first parked review and recovery work

November 20, 2026. Build **0.1.0 (14.1)**. Milestone 7 remains open;
road testing is on hold. No additional car visit is requested until a verified
replacement beta and one consolidated follow-up checklist are ready.

## Retrieved evidence

The original Files `Archive.zip` is retained privately under the ignored
`.cache/physical-tests/2026-11-20/m7-14.1/` directory, alongside unchanged
extracted files and independent validation reports. No phone evidence is committed.

- Original ZIP: 623,365 bytes; SHA-256
  `4562b613162e0be2eed9c4af12d02b475ac744030db18a3bdd98c23f23f33ee1`.
- ZIP CRC/path/duplicate checks passed. AppleDouble metadata is retained in the
  original archive but excluded from canonical session inspection.
- All **37 live captures** independently validate with explicit incomplete
  inspection: 19 saved sessions and 18 activity probes. Eight probes are
  incomplete and remain unchanged. Older manual/M6 captures are also present.
- Four M7 automatic drives contain **856 observations**. Their policy journal,
  separate probes and four durable confirmation claims validate together.
- Five available incident timelines reconstruct, including the fully Captured
  manual incident in the longest automatic recording. Other automatic manual
  incidents are Partial; they are not promoted to full windows.

## Physical observations and limits

The user observed no drive from an engine-off scanner, automatic start about
5–10 seconds after engine start, successful brief scanner reconnect, and recovery
after a roughly 15-second engine stop. The retained drives support successful
ordinary acquisition and a full manual incident window. These observations do
not qualify every lifecycle, stop/rearm or wake case.

The UI failed to acknowledge a successful incident mark and displayed multiple
CAN responders per reading. The M7 export action also failed; original Files
retrieval worked and supplied usable evidence.

All four native confirmation outcomes were `suppressed_disabled`. Successful
Go claims are not proof that any notification was displayed. All journal scene
phases were `unknown`; there are no reliable foreground/background boundaries
in this capture. Lock/background continuity therefore remains unqualified.

## Failure sequence

Times below are November 20, America/Chicago, reconstructed from retained clocks.

| Approximate time | Retained evidence |
| --- | --- |
| 14:49:11 | On returning from a drive to an activity probe, the first accepted `STPC` receives `STOPPED` and setup is blocked. This is an interruption, not an acknowledgement. |
| 14:56:38–15:02:00 | Longest automatic drive: about 321 seconds, 466 observations and a full manual window; one fresh connection generation follows a gap. |
| 15:02:00 onward | New monitoring probes accept `STPC` writes but receive no bytes across successive fresh generations. |
| 15:02:32 probe | Four startup timeouts exhaust the acquisition synchronization retry limit; originals are left incomplete. |
| 15:06 onward | New probes contain no transactions or connection evidence, while owner/resource events continue. |

The observed budget failure is consistent with **synchronization retry
exhaustion**, not the storage retention cap (roughly 4.6 MiB retained here).
The capture does not record the iOS External Accessory directory population, so
it cannot establish why later attempts lacked a usable app stream despite
Bluetooth showing connected. Do not assign a proven scanner/iOS root cause.

## Targeted implementation

- Versioned `quiet-start-1` for new automatic/probe captures: wait at least one
  second before setup, retain residual bytes without attributing them to a new
  request, require 500 ms quiet after residual bytes, and bound settling at three
  seconds. Native stream handover also delays reopening the previous channel.
- A strictly recognized `STOPPED` reply on initial `STPC` remains a failed
  transaction. At most two delayed retries are allowed; real `OK` and the full
  existing setup/support checks are still required. Ordinary manual setup is
  unchanged. Independent replay checks the declared version and retry evidence.
- Failed monitoring can be explicitly disarmed, exported and rearmed with a new
  capture. Failed originals are closed without repair or healthy finalization.
  Synchronization and evidence/storage budget errors use distinct messages.
- Diagnostic export closes native streams and disarms monitoring itself. It
  includes ConnectionCaptures, reports progress/errors, and canonicalizes native
  root path aliases. Export success means ZIP preparation, not confirmed sharing.
- Incident marking acknowledges only a successful Go result, with a checkmark,
  brief success text/haptic, incident count and capture status.
- Primary live display selects one actual valid responder per signal using a
  stable source order, with source ID shown. All responders remain in a disclosure
  and all original data/policy inputs remain recorded; no averages or ECU role
  guesses are introduced.
- Application lifecycle notifications and initial process state supply scene
  evidence. Accessory directory counts, remaining native retries and export
  outcomes provide missing diagnostics. No undocumented vehicle commands or
  private Bluetooth APIs are added.

Regression fixtures reproduce residual bytes, `STOPPED`, and four accepted/no-byte
startup timeouts. Native coverage must include failed-owner export/share on the
first tap, preserved original bytes, fresh rearm, source selection and incident
success/failure feedback. These are synthetic checks; physical recovery remains
open until an updated beta is tested.

## Follow-up scope

Reuse successful ordinary reads, the full incident window and accepted M6 data.
The [held recovery packet](../planning/milestone_7_recovery_test_session.md)
collects one 15–20 minute parked block. It should concentrate on drive-to-probe handover,
stop/rearm, force-quit/relaunch, scanner reconnect and export after any failure;
check the improved incident feedback and compact readings in that same block.
Only after recovery passes should the remaining lock/background/notification and
normal sustained-end cases proceed. Road and representative resource/battery
qualification remain separate gated work. Milestone 8 has not started.


## Retained recovery beta — not delivered

Final signed candidate: **0.1.0 (20.1)**, prepared by
[signing-only run 44825527697](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/44825527697).
Signing/export passed and the Apple upload step was skipped. This is not a
TestFlight delivery. Original 14.1 upload approval does not cover a new payload.

- Signed app source: `19e83737ef623014c076dbc8f18d24a65efe0604`.
- IPA SHA-256: `8cebea51d08bd7eda3957c7647b4b8ccc53c705099aac8bec673be78b8b64cf3`.
- Retained signed artifact: `14674560791`; outer ZIP SHA-256
  `d2d0ee752ff9ff3660de3203e5a48fe5c90f4ef51a44bfd8b76c118716c2e739`.
  Downloaded ZIP CRC/digest and IPA app/version/build/source provenance verify.
- Final native test revision: `4666c1b66713070f69678e72015d0d397277f832`.
  Git comparison with signed source changes only
  `ios/VehicleFlightRecorderUITests/EvidenceNavigationTests.swift` to reveal the
  whole row before the test's single tap. App/core/build inputs are identical.
- Final native gate: **passed**
  [run 44827283375](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/44827283375):
  **46 hosted tests, 9 UI tests, both iOS SDK build profiles, 23 native session
  ZIPs and 4 whole automation bundles**. All available incidents reconstruct.
  First-tap/repeated M7 diagnostic sharing after real Go synchronization
  exhaustion passes; its Save to Files screenshot was reviewed.
- Native artifact `14675387888`, SHA-256
  `22f63569114b4b44b1d18e0bf003629f085414f49286636df4b65c45b026f0c7`,
  was downloaded with CRC/digest verification. Local independent replay again
  validates all 23 native ZIPs and four automation bundles.
- Portable validation: all Go tests/vet; acquisition/mobile race checks and focused
  final four-timeout recovery race checks; **185 Python** and **34 script** tests;
  workflow lint. Final branch portable CI passes. All 37 phone captures and their
  whole automation bundle still validate under the stricter reader.

Superseded runs are not acceptance evidence. One reproduction omitted the old
connection's disconnect before reconnect; the corrected sequence passes directly
in Go. The next native run passed all 46 hosted tests and eight existing UI tests,
but the new UI test tapped a partly clipped row's center below the visible screen.
The final test requires the full row above the bottom safe area, without adding
any repeat-tap workaround. Signing-only 17.1 and 19.1 candidates are superseded;
one incorrect-source preparation was canceled. None were uploaded to Apple.

The native gate is complete. The held recovery packet remains unreleased until
explicit approval, upload and processing of this exact 20.1 IPA are complete.
Physical recovery and the remaining M7 acceptance matrix still need actual
phone evidence; automated success does not close the milestone.
