# Driver Companion V2 — Product and Design Plan

Status: approved planning direction, October 29, 2026. This document records the
user-reviewed product scope and design concepts; it does not authorize immediate
implementation or establish that any live measurement has been validated.
Detailed layouts, numerical quality thresholds and release scope remain subject
to prototypes and real capability evidence.

## 1. Purpose and position in the roadmap

Evolve the diagnostic flight recorder into an engaging companion for performance
car owners: enjoy live instrumentation, explore drives, test performance in
appropriate track/closed-course sessions, compare results and understand vehicle
health from the same preserved evidence.

The sequence remains:

1. Complete the [diagnostic-first MVP](alfa_flight_recorder_mvp_plan.md).
2. Establish automatic generic OBD discovery and reliable real-car recording.
3. Add the verified Alfa package needed to investigate the owner's immediate issue.
4. Deliver this driver companion in evidence-gated increments.
5. Pursue [enterprise expansion](vehicle_health_enterprise_architecture.md) when
   broader coverage and infrastructure are justified.

The Alfa is the first validation vehicle. Generic discovery and recording remain
shared functionality; manufacturer-specific requests, decoders and diagnostic
rules remain in applicable vehicle packs. No enhanced pack must mean generic
functionality with honest coverage, not an unusable app.

V2 adds entertainment and everyday value without displacing diagnostic work.
A real intermittent fault need not occur on schedule to validate infrastructure,
but passing controlled tests does not mean the real fault has been diagnosed.

## 2. Entry criteria and release discipline

Before dependent V2 implementation, demonstrate:

- repeatable physical-device recording and retrievable, validated exports;
- a capability report for the actual vehicle/adapter and combined polling workload;
- documented background, locked-screen, reconnect and session-boundary behavior;
- an operational diagnostic investigation workflow and prioritized Alfa pack;
- measured storage/battery impact and explicit unresolved coverage limitations.

Design exploration with clearly labeled mock data can begin earlier. Unresolved
recorder failures block dependent features. Do not wait for exhaustive proprietary
signal discovery, and do not promise a metric merely because its inputs appear
in a standards catalog.

## 3. One recorder, distinct experiences

Performance and Diagnostics are **cockpit views**, not separate recording modes.
Switching views preserves the drive, incident detection, raw evidence and recording
state. UI refresh and acquisition frequency remain independent.

Any future performance-test polling profile is an explicit session configuration,
with recorded coverage tradeoffs. It must not silently disable diagnostic coverage.

The three experience themes are:

| Area | Purpose | Tone |
| --- | --- | --- |
| Cockpit | Live instruments and performance-test state | Immersive, responsive, motorsport-inspired |
| Drive debrief and personal records | Explore, compare and celebrate results | Visual, engaging, competitive with yourself |
| Vehicle health | Explain faults and inspect evidence | Calm, precise, restrained |

Recording health stays recognizable across all three.

## 4. Garage landing page

The home screen is a personal garage centered on the selected vehicle:

1. Vehicle name, known configuration and vehicle selector.
2. A three-quarter car model on a dark, subtly illuminated surface.
3. Plain-language connection and recording state, including elapsed recording time.
4. Primary action: Open Cockpit, or Return to Live Drive during recording.
5. Compact previews of the last drive, personal records and recent incidents/health.

The user has an Alfa 3D model available; asset delivery, suitability, usage rights,
format and mobile rendering cost must be assessed before integration. Do not claim
it is already in the repository. For vehicles without a suitable asset, use an
intentional stylized silhouette, optionally by body type. Model availability must
never gate telemetry functionality or imply verified vehicle support.

The model can communicate connection through subtle lighting, support a parked
inspection view, and provide a consistent vehicle image in drive/record cards.
Only associate component highlights with evidence of appropriate specificity.
A communication timeout does not prove that a highlighted component is defective.

Rotation and detailed component exploration are parked interactions. Avoid
continuous decorative rendering that adds heat or battery cost; retain a static
fallback and reduced-motion behavior. The landing page reflects hands-off
recording and must not introduce a per-trip Start requirement.

## 5. Live cockpit views

Both views share recording/connection status, freshness conventions, the incident
marker's position and a familiar instrument hierarchy. Start with curated defaults
and a few configurable slots, not an unrestricted dashboard editor.

### Performance

Candidate composition:

- horizontal RPM band and prominent speed;
- boost instrument when supported;
- calibrated G-circle when motion support is available;
- throttle or pedal indicator with correct signal labeling;
- compact relevant temperatures;
- selected performance-test status when that feature qualifies for release.

Use one dominant instrument composition rather than equally weighted gauges.
Keep detailed celebrations and comparisons in the debrief. Unsupported, stale or
uncalibrated instruments are visibly unavailable; never substitute fabricated data.

### Diagnostics

Candidate composition:

- voltage trend and current value with source identified;
- module response state, response age and recording coverage;
- recent diagnostic events and incident capture status;
- selected temperatures and contextual RPM/speed;
- direct access to supporting evidence when appropriate.

Prefer short trend graphs where change over time matters. Module responsiveness
is not equivalent to mechanical health. Final instrument priorities follow MVP
field evidence and the owner's diagnostic needs.

## 6. Drive history and debrief

Maintain one drive history. Each drive offers:

- Overview: duration, supported distance estimate, moving/idle time, highlights
  and coverage.
- Performance: qualifying records, runs, supported G metrics and comparisons.
- Incidents: faults, manual markers, capture windows and evidence timelines.
- Replay: synchronized telemetry, plus a route when location was recorded.

Lead with an engaging drive card, highlights and qualifying personal improvements,
then allow deeper inspection. Link performance and health context: a run can show
its temperatures and any incident occurring during the same drive.

Begin calculations using real exports in Python; validate definitions before
moving reusable on-device calculations into Go. Results must be reproducible.
Distance and maximum speed must be labeled according to their measurement source
and coverage; use "maximum recorded speed" rather than implying a continuously
observed absolute maximum.

## 7. Personal scoreboard and records

Provide a full Records destination and a compact Garage preview. Candidate records:

- maximum recorded speed;
- best qualifying 0–60 mph and 0–100 km/h;
- best rolling-acceleration intervals;
- peak sustained left/right cornering G;
- peak acceleration and braking G;
- quarter-mile results only after separate feasibility/accuracy validation.

Records are vehicle-specific, filterable by date, and linked to their original
run/drive and supporting evidence. Users can exclude a run or reset displayed
records without deleting original observations. Define the reset period and
exclusion behavior explicitly so records do not unexpectedly reappear.

Distinguish best observed values from qualified personal records. Apply outlier,
gap, calibration and source-quality checks before awarding a record. A speed
spike or phone movement must not become a personal best. Brief peaks and sustained
G values need distinct definitions and labels.

Record the source, timing convention, quality and calculation version. Separate
noncomparable measurement methods or make their differences explicit; a rollout
result must not silently replace a standing-start record. Use concise quality
badges with details available on demand. Recalculations must not silently rewrite
the historical meaning of an achievement.

Use personal-best cards and run comparisons for entertainment. Keep initial
competition personal; public leaderboards, verification/moderation infrastructure
and social challenges are outside this release. Recorded top speed need not become
a challenge to exceed it. Dedicated tests are framed for track/closed-course use.

## 8. Location, motion and performance measurement

Location is opt-in; diagnostics continue without it. Provide control over location
in exports. Add route replay, GPS speed/distance, acceleration/braking/cornering
estimates and the G-circle only after their inputs are validated.

Motion requires a mounted phone, orientation calibration, gravity handling and
filtering. Detect or flag unsuitable mounting/calibration where feasible. Validate
clock alignment, noise rejection, physical-device behavior and battery impact.
Do not infer measurement accuracy from a smooth animation.

For each derived metric, specify:

- required signals, units, source and applicability;
- algorithm and version;
- start/end definitions and rollout convention where relevant;
- required sampling/coverage and maximum acceptable gaps;
- invalidation rules, uncertainty and displayed precision;
- provenance linking results to original evidence;
- reference validation and repeatability evidence.

Initial timing candidates are 0–60, 0–100 km/h and configurable rolling intervals.
Horsepower, stopping distance, quarter-mile timing and sophisticated lap analysis
are candidates, not commitments. External high-rate GPS is a later integration
option if evidence justifies it; no device-specific accuracy promise is made here.

## 9. Visual identity and navigation

Working direction: premium automotive instrumentation with restrained sci-fi
details. Exact appearance is to be explored, not fixed by this prose.

- Graphite surfaces, dimensional lighting, off-white text and tabular numerals.
- Fine instrument markings and one configurable vehicle accent.
- Consistent semantic colors; amber/red attention states must remain distinct
  from decorative accents, including an Alfa red theme.
- Brief purposeful transitions, reduced-motion support and readable daylight UI.
- Clear typography and accessible contrast; avoid tiny text, decorative code,
  excessive glow and constantly pulsing panels.
- Stale values must never resemble current measurements.

Initial navigation: **Garage · Cockpit · Drives · Records**.
Vehicle health and incident detail are reachable from Garage and individual drives;
Diagnostics is the live cockpit view. A persistent recording indicator returns
the user to the active cockpit.

Prototype Garage, Performance cockpit and Drive debrief as connected screens first.
Then apply the same structural language to Diagnostics, emphasizing evidence and
trends. Validate readability and interaction before expanding customization.

## 9.1 Current-drive banner and Live Activity

**Approved requirement — October 29, 2026:** Expand the persistent recording
indicator into a current-drive summary, both inside the app and on the Lock Screen/
supported Dynamic Island surfaces through a Live Activity. This summarizes the
current session, not all-time personal records. The MVP auto-start confirmation
(§15.1 of the MVP plan) remains a separate, one-time notification.

Expanded content:

| Field | Definition |
| --- | --- |
| Recording state | Recording, coverage degraded, or ended |
| Drive time | Elapsed session duration, including stationary engine-running time |
| Max speed | Highest valid speed recorded in the current drive |
| Cornering G | Highest qualifying sustained lateral G magnitude in the current drive |
| Best 0–60 | Best completed qualifying 0–60 mph run in the current drive |

Use explicit units and labels. Space-limited layouts prioritize recording status
and elapsed time; show additional metrics only where they remain readable.
Selecting other display units must not silently change a 0–60 mph test into a
0–100 km/h test. The field retains its test definition.

### Best 0–60 behavior

- Display `0–60: —` until a complete qualifying run exists; an explanatory detail
  can state that no qualifying run has been recorded.
- Require captured standing-start/launch evidence, crossing of 60 mph and the
  metric contract's sampling, gap and quality criteria. Merely reaching 60 mph
  somewhere during a drive is insufficient.
- Never extrapolate a result from a run that did not reach 60 mph. Do not predict
  a completion time to fill the banner.
- Interpolation between valid samples bracketing 60 mph may estimate the crossing
  time only within the validated timing method and allowed gaps. It does not
  excuse missing launch evidence or insufficient sampling.
- After multiple qualifying runs, show the best comparable result from this drive.
  Preserve timing/rollout convention and provenance; do not mix conventions.
- Incomplete or unreliable runs cannot replace a qualifying best. Their details
  may be inspected in the debrief.
- Keep the field's meaning stable; do not automatically substitute peak acceleration
  G when a 0–60 result is absent.

### Interaction and data quality

- Tapping returns to the active cockpit, or the corresponding debrief after ending.
- Show unavailable metrics as unavailable, never zero. Reuse the debrief/scoreboard
  algorithms, quality filters and source labels rather than a separate calculation.
- Preserve valid maxima and completed runs across brief disconnections, but expose
  degraded coverage/stale updates clearly. A running duration is not proof that
  fresh telemetry is arriving.
- Update without repeated notifications, flashing or record celebrations. Use
  bounded updates appropriate to each surface; do not promise cockpit-rate updates.
- End the active presentation when the drive ends; a clearly ended summary may
  offer access to the debrief. Never leave a stopped session labeled Recording.
- Dismissing either surface does not stop recording. Notification, Live Activity
  and recorder lifecycles remain separate.
- Offer a privacy option to hide performance values externally while retaining
  recording state and duration. Respect system preview and Live Activity settings.

### Platform feasibility and staged delivery

Validate automatic Live Activity creation, updating, ending and stale-state behavior
under the actual supported iOS/device/background-accessory lifecycle before
promising hands-off availability. A Live Activity does not grant recording runtime.
If system authorization or lifecycle limits prevent it, recording and the in-app
banner continue; document any required user action. Do not add cloud/push
infrastructure solely to evade a limitation without a separately reviewed decision.

V2.0 provides the in-app banner using validated fields and includes the external
surface feasibility work. Deliver the external surface when validated. Cornering
G and best 0–60 populate only after their V2.1/V2.2 measurement gates pass; the
banner does not accelerate those dependencies or require fabricated placeholder
measurements.

Acceptance includes both cockpit views, session end, reconnect, stale telemetry,
permission/settings restrictions, compact/expanded layouts, external privacy,
dismissal without recording loss, and physical-device lifecycle tests. Check
no-run, never-reached-60, invalid-run, valid-run and multiple-run cases; confirm
that displayed results agree with exported evidence and the debrief. Measure
resource impact alongside recording and verify updates do not degrade acquisition.

Reference: [Apple ActivityKit](https://developer.apple.com/documentation/ActivityKit).

## 10. Architecture

- Swift: native sensor acquisition, permissions, lifecycle, rendering and UI.
- Go: recording, source coordination, session logic and validated metric algorithms.
- Python: exploratory analysis, independent validation and deeper offline analysis.
- Vehicle packs: enhanced signal definitions, applicability and diagnostic rules.

Explicitly extend the existing single-source, ordered-ingestion contract before
introducing asynchronous sensors. Preserve per-source timing, clock mapping,
late-arrival handling and observation provenance. Keep raw observations separate
from derived results, version schema migrations, and retain recalculation ability.
Profile actual workloads before changing the Swift/Go bridge representation.

Remain local-first. Accounts, cloud sync, subscriptions, fleet analytics, AI
diagnosis and broad manufacturer certification are not V2 prerequisites.

## 11. Incremental releases and acceptance

| Release | Scope | Gate |
| --- | --- | --- |
| V2.0 | Garage/cockpit identity, drive journal, OBD-backed debrief and eligible personal records | Reproducible summaries, honest coverage, readable UI and unchanged recorder reliability |
| V2.1 | Optional location/motion, replay, G-circle and qualifying G records | Physical sensor validation, calibration, source synchronization and acceptable resource use |
| V2.2 | Performance tests, qualifying acceleration records and richer comparisons | Explicit metric contracts, reference comparison and repeatability under supported conditions |

These are planning increments, not promised dates or mandatory bundles. Release
useful subsets when their evidence is ready. Do not delay a validated drive journal
for unresolved performance timing or publish an unvalidated record to fill a card.

Verify that switching cockpit views never loses evidence, missing inputs cannot
become zeros, excluded/invalid runs cannot win records, and export/reanalysis can
reproduce results. Test schema evolution against existing diagnostic recordings.

## 12. Next planning decisions

Before implementation, review connected screen prototypes, the supplied car asset,
portrait/landscape priorities, the initial configurable instrument slots and the
first metrics justified by the capability report. Set quantitative quality and
resource budgets from measurements. Keep scope decisions and experimental results
explicit rather than treating this design direction as proof of feasibility.
