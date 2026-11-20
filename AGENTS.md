# Vehicle Flight Recorder working conventions

Keep project work under `/home/kaletevans/dev/VehicleFlightRecorder` and preserve
unrelated user edits. Phone originals and credentials stay in ignored local
storage; commit only sanitized findings and provenance.

For a reported failure:
1. Inspect/validate the supplied evidence and give the user findings early.
2. Reproduce it with local tests before starting expensive macOS runs.
3. Use a targeted native run for affected cases during debugging. It is never
   full acceptance; see `docs/CI_WORKFLOW.md` for selectors and provenance.
4. Freeze app changes, then run full native acceptance once. Repeat only for
   new failures, relevant changes or unresolved concerns.
5. Prepare one final signed candidate after validation. Upload the exact retained
   user-approved IPA; do not rebuild approved bytes for upload.

CI changes, test corrections and app fixes should be locally checked before
pushing. Documentation-only changes do not need native builds. Batch required
phone/car tests into one substantial visit and one evidence handoff; reuse
accepted baseline results. Do not request another car visit for a UI-only fix
that native tests can cover. Keep physical and simulated acceptance distinct.
