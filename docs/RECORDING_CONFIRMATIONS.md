# Automatic recording confirmations

Implementation contract: M7, November 18, 2026. Native fake/resolver and regression verification passed on source `4d6e857`;
physical notification presentation and lifecycle observations remain pending.

A confirmation means an automatically initialized drive has admitted actual
telemetry. It does not certify all modules, continuous coverage, user visibility
or a diagnosed fault. Manual starts, mere connection and attempted starts do not
produce it. Vehicle identity is generic unless independently supplied/verified;
the present implementation says `Vehicle` and `Limited standard OBD coverage`.

| Condition | Chosen behavior |
| --- | --- |
| Preference disabled | Claim/consume fact; no presentation; normal recording continues |
| App visible | One brief in-app banner; no duplicate system banner |
| Supported physical background + authorization | Quiet local request with stable `vfr.start.<drive-id>` identity |
| Denied / not determined | No background request or repeated prompt; normal recording continues |
| Explicit parked setup button | Request alert authorization only when not determined; never request sound/badge |
| Denied setup | Link to iOS system settings; no eligibility dependence |
| Background submission reaches newly foreground app | Delegate suppresses system presentation and may show one in-app banner |
| API/receipt failure | Record failure; no retry of an uncertain claim; acquisition continues unless evidence storage itself fails |
| Engine stop/start, reconnect, view changes | Same logical drive ID; no second successful-start confirmation |
| Relaunch | Consumed facts remain consumed; interrupted drive stays preserved; new eligible drive has new ID |
| Ended/missing drive before dispatch | Consume/suppress; do not send backlog after enabling permission |
| Tap | Original ID; active dashboard, exact ended summary, or visible missing/unreadable state; never start a new drive |

## Persistence and uncertainty

Go appends/syncs the success fact after actual telemetry, then writes its claimed
state and journal before passing the fact to native. Native exclusively creates
and syncs a per-drive selection receipt before an API or presentation side effect.
It uses a separate foreground receipt to suppress duplicate in-app presentation.
Files are local/private. No callback-based retry, random request ID or transport-
generation identity is used for confirmations.

The native API and Go disk state are not one atomic transaction. A crash between
claim and submission can miss a confirmation. That best-effort loss is explicit;
a relaunch must not retry an uncertain submission and risk a second delivery.
This is not a guarantee against arbitrary filesystem corruption or OS behavior.
Submission success means submitted, not displayed/seen. Focus and system policy
may affect actual presentation; permission cannot establish accessory wake.

## Debugging evidence

Drive canonical events contain initialization/success facts, policy transitions,
observed lifecycle and applicable confirmation outcomes. `Automation/journal.jsonl`
also records pre-start and post-drive claim/submission/suppression/failure evidence.
The combined diagnostic bundle preserves the journal, state, receipts, probes
and automatic drives without editing their original bytes. Disable monitoring
before export so the capture set is stable. Normal session ZIPs stay six files.

Native fake tests cover permission/preferences, foreground/background selection,
scene changes during settings, manual/invalid/ended facts, errors, duplicate
callbacks, durable receipts and relaunch without retry. Actual notification-center
presentation and physical cold-launch/tap behavior require the grouped phone test.

References checked November 18:
[authorization](https://developer.apple.com/documentation/usernotifications/asking-permission-to-use-notifications),
[delegate lifecycle](https://developer.apple.com/documentation/usernotifications/unusernotificationcenterdelegate).
