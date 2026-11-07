# Physical test sessions

Approved workflow preference — November 3, 2026: minimize separate car visits.
Prepare one complete acceptance checklist per milestone, grouped into substantial
connected-test blocks where dependencies require more than one visit. This changes
how tests are scheduled; it does not waive physical evidence or milestone gates.

## Default workflow

1. **Prepare at the desk.** Finish implementation, transcript/failure tests, native
   checks and signed delivery. Verify the exact TestFlight build is available.
   Prepare the whole checklist, prerequisites, equipment/reference needs, expected
   results, stop/continue rules and evidence mapping before requesting a car visit.
2. **Run the connected block.** Perform the listed parked checks back-to-back,
   using separate recordings where operating state/workload must be distinguishable.
   Short checks remain a gate before longer acquisition, but the user may continue
   immediately when the checklist's observable criteria pass. No chat round trip,
   per-case upload or new build is required between those checks unless the packet
   explicitly identifies an evidence-dependent hold.
3. **Finish disconnected work afterward.** Saved history, evidence scrolling,
   export/share, file transfer and Python analysis normally happen indoors after
   the connected work. Capture a live manual mark while connected; review it later.
4. **Return one evidence handoff per visit.** Collect the individual exports and
   a short result list together. The agent checks integrity, raw provenance,
   rates/gaps and references after the visit, then reports acceptance or one
   consolidated list of remaining issues.
5. **Batch fixes and retests.** Finish all independent fixes and automated checks
   before offering another build. Provide one retest list with affected case IDs,
   the reason each must be repeated and a short relevant regression check. Reuse
   accepted evidence for unchanged paths/configurations; do not repeat every
   milestone test for an unrelated presentation correction.

One visit can contain several app recordings. One evidence handoff can contain
several ZIPs. These are not promises of a bulk-recording or bulk-export feature;
current builds may require exporting each saved session individually, indoors.

## Contents of a ready-to-run test packet

| Item | Required content |
| --- | --- |
| Build/configuration | Exact build/source, vehicle/adapter/phone, allowed protocol/commands and workload; validated prerequisites |
| Schedule | Ordered case IDs, estimated connected time, state changes, recording duration and explicit holds |
| Procedure | Controls to use, observable expected result, when to continue, what to skip/stop after failure |
| Evidence | Which recording/ZIP covers which case, minimal notes/screenshots, reference comparison setup and one handoff list |
| Afterward | Desk-only checks, agent analysis and acceptance criteria; pending/unsupported cases stay explicit |

Estimate planning time separately from measured acquisition duration. Do not ask
the user to sit in the car waiting for CI, Apple processing or routine log review.
Arrange reference tools before the visit; record unavailable reference evidence
as pending rather than inventing a comparison or repeating all unrelated cases.

## Gates and failures

A short initialization/read check before a longer parked recording is an
**in-session gate**, with prewritten visible criteria. Passing it allows the next
listed parked block during the same visit; final acceptance still waits for raw
export analysis. A new protocol, command set or road workload requiring review of
prior physical evidence is an **analysis hold**; identify it before the visit.

On a failed case, record the result and preserve evidence. Stop dependent cases;
continue only listed independent cases whose prerequisites still hold. A parameter
explicitly unsupported does not become a failed ECU, and a successful manual
recovery does not turn a failed automatic-recovery case into a pass. New warnings,
uncertain transport state or other abnormal vehicle behavior stop transmissions;
do not troubleshoot by improvising commands or inducing vehicle faults.

Tests involving motion follow accepted stationary evidence and the milestone's
approved road configuration. Set controls/permissions while parked; inspect files
and change configuration after stopping. Distinguish locked screen, OS suspension,
OS termination and force-quit; an unreproduced condition remains untested.

## Remaining milestone blocks

| Milestone | Planned grouping | Required boundary |
| --- | --- | --- |
| 5 | Already accepted; reuse its identity/transport evidence | No new M5 car visit to adopt this workflow; M6 rechecks recovery with actual polling |
| 6 | One parked visit, approximately 20–30 minutes plus any prepared reference-tool time; all short, combined and reconnect recordings together | [Current runnable checklist](milestone_6_test_session.md); no road testing |
| 7 | One consolidated parked policy/notification/lifecycle block; after analysis, a controlled-road block; representative repeated/long drives grouped into normal use where suitable | M6 accepted first; stationary results qualify the road workload. Durations/budgets must be set before release from actual evidence |
| 8 | One parked block for the complete agreed, verified enhanced scope and references; after analysis, one planned combined-road block plus necessary targeted follow-up | M7 accepted first; new enhanced reads require stationary qualification before road use |

M7/M8 entries are planning structure, not runnable test instructions or permission
to start those milestones. Reliability measurements may require multiple drives
or reproducible lifecycle conditions; do not promise one car visit for all of M7.
An actual defect or evidence-dependent gate can require a follow-up, but group
that follow-up work into a complete packet rather than issuing isolated requests.
