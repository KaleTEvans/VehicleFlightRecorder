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
The next single parked block should concentrate on drive-to-probe handover,
stop/rearm, force-quit/relaunch, scanner reconnect and export after any failure;
check the improved incident feedback and compact readings in that same block.
Only after recovery passes should the remaining lock/background/notification and
normal sustained-end cases proceed. Road and representative resource/battery
qualification remain separate gated work. Milestone 8 has not started.
