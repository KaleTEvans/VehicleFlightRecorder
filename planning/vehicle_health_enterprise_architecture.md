# Vehicle Health Platform — Enterprise Architecture Roadmap
## Living document
### Status: Directional architecture; expand as the Alfa MVP generates evidence

**Development environment decision (October 2026):** The reference developer has Windows + WSL and an iPhone, but no Mac. Milestone 0 is complete; an intermediary **MVP Milestone 0.5** introduces GitHub Actions macOS builds and Appetize browser-hosted simulator previews. The app is approved for OBDLink MX+, but simulator previews cannot connect to physical MFi/Bluetooth hardware. This is a **delivery/testing strategy**, not a change to the core platform architecture.

---

# 1. Product vision

Evolve the Alfa Flight Recorder into a consumer vehicle-health platform that can continuously observe supported vehicles, capture incidents, identify degradation, and produce understandable maintenance/diagnostic insights.

The long-term product is **not** "one universal decoder that magically understands every car."

It is:

> A common telemetry, incident, analytics, and reporting platform combined with versioned vehicle-specific diagnostic packs.

The 2017 Alfa Romeo Giulia Quadrifoglio is vehicle pack #1 and serves as the reference implementation.

---

# 2. Long-term user value

The product should answer four progressively harder questions.

## Level 1 — What happened?

Example:

```text
At 14:22:17 the TCM stopped responding for 1.2 seconds.
Voltage began falling 650 ms earlier.
Three secondary modules reported communication faults afterward.
```

## Level 2 — Is this new?

```text
This is the first TCM dropout seen without a simultaneous adapter disconnect.
```

## Level 3 — Is the vehicle degrading?

```text
Warm-idle charging voltage has fallen relative to this vehicle's prior 8-week baseline.
```

## Level 4 — What should the owner do?

```text
Evidence favors a vehicle electrical/network issue over a primary transmission fault.
Prioritize charging/ground-path inspection before replacing the TCM.
```

The system should always preserve the evidence supporting higher-level conclusions.

---

# 3. Architectural principles

## 3.1 Evidence first, AI last

Never send millions of raw CAN frames to an LLM and ask it to diagnose the car.

Preferred pipeline:

```text
Raw vehicle data
      ↓
Decoder / normalization
      ↓
Quality validation
      ↓
Time-series store
      ↓
Event detection
      ↓
Incident correlation
      ↓
Baseline + anomaly analytics
      ↓
Structured findings
      ↓
AI explanation/report
```

The LLM explains and contextualizes computed findings. It should not be the primary anomaly detector.

## 3.2 Preserve raw evidence

Every normalized value should be traceable, where possible, to:

- adapter.
- vehicle pack version.
- ECU/module.
- request or raw frame.
- decoder version.
- timestamp.

Decoder updates must not make historical raw evidence unusable.

## 3.3 Separate platform from vehicle knowledge

Core services should never need to know that a car is specifically an Alfa, BMW, Ford, etc.

Vehicle-specific behavior lives in a versioned Vehicle Pack.

## 3.4 Read-only first

Enterprise expansion should prioritize safe passive observation and diagnostic reads.

Writing/coding/programming ECUs is a separate risk class and should not be required for the core business.

## 3.5 Gradual support levels

Do not promise the same depth for every vehicle.

Suggested support tiers:

- **Universal** — standardized OBD data.
- **Enhanced** — additional validated manufacturer signals.
- **Deep** — broad module-specific telemetry and incident intelligence.
- **Research/Beta** — incomplete community/experimental definitions.

Clearly label support depth per make/model/year/powertrain.

---

# 4. Platform layers

```text
┌────────────────────────────────────────────────────────────────────┐
│                         Consumer Experience                        │
│ iOS / Android / Web / Reports / Alerts / Maintenance Timeline      │
└────────────────────────────────┬───────────────────────────────────┘
                                 │
┌────────────────────────────────▼───────────────────────────────────┐
│                       Insight & Reporting Layer                    │
│ Findings | Weekly reports | Incident explanations | Recommendations│
└────────────────────────────────┬───────────────────────────────────┘
                                 │
┌────────────────────────────────▼───────────────────────────────────┐
│                         Analytics Platform                         │
│ Baselines | Trends | Anomalies | Correlation | Fleet comparison    │
└────────────────────────────────┬───────────────────────────────────┘
                                 │
┌────────────────────────────────▼───────────────────────────────────┐
│                         Canonical Vehicle Model                    │
│ Signals | Events | DTCs | Modules | Sessions | Incidents           │
└────────────────────────────────┬───────────────────────────────────┘
                                 │
┌────────────────────────────────▼───────────────────────────────────┐
│                             Vehicle Packs                          │
│ Alfa | BMW | Ford | GM | Porsche | ...                            │
└────────────────────────────────┬───────────────────────────────────┘
                                 │
┌────────────────────────────────▼───────────────────────────────────┐
│                         Protocol / Transport                       │
│ OBD-II | UDS | CAN | DoIP | J2534 | Adapter-specific transports    │
└────────────────────────────────────────────────────────────────────┘
```

---

# 5. Client architecture

## 5.1 iOS

Keep native iOS responsibilities thin:

- hardware communication.
- background/lifecycle integration.
- UI.
- local encryption/key management.
- local export/sync.
- permissions.
- safety UX.

Shared recorder/domain logic should remain in Go where practical.

### Reference implementation: OBDLink MX+

For vehicle pack #1, the confirmed iOS transport is Apple External Accessory over the MX+'s Bluetooth Classic/MFi interface. OBD Solutions documents:

```text
MFi Product Plan ID:  221699-0001
iAP protocol string:  com.obdlink
Accessory model:      MX201
Bluetooth:            Classic v3.0, no BLE
```

The iOS implementation therefore uses `EAAccessoryManager` + `EASession`, declares `com.obdlink` in `UISupportedExternalAccessoryProtocols`, and enables the `external-accessory` background mode. This adapter-specific code remains behind the generic transport contract and must not leak into vehicle packs or analytics.

The initial application identity used for OBDLink whitelisting is:

```text
Vehicle Flight Recorder
version 0.1.0
com.kaleevans.vehicleflightrecorder
```

Treat this as implementation metadata, not as a permanent enterprise product name.

## 5.2 Android

A future Android client should reuse the same Go domain package where possible.

Bluetooth and lifecycle implementations become Android-native adapters around the shared core.

## 5.3 Desktop/service tooling

Desktop diagnostic/research tools can directly use Go/Python without mobile bindings.

---

# 6. Shared Go core

The Go core is a candidate for a reusable cross-platform SDK.

Potential modules:

```text
vehiclecore/
├── canonical/
├── session/
├── incidents/
├── protocols/
├── scheduler/
├── decoder/
├── packs/
├── quality/
├── storage/
└── transport/
```

Long-term stable interfaces:

```text
Transport
Protocol
VehiclePack
SignalDecoder
PollingPlanner
Recorder
IncidentDetector
StorageSink
```

A transport should not contain vehicle-specific signal meaning.

A vehicle pack should not contain UI logic.

---

# 7. Python analytics platform

Python is the preferred environment for analytics and model development.

Long-term packages:

```text
vehicle_analytics/
├── ingestion/
├── quality/
├── feature_engineering/
├── baselines/
├── anomaly/
├── incidents/
├── fleet/
├── maintenance/
├── reporting/
└── ai/
```

Capabilities should grow in this order:

1. deterministic summaries.
2. within-drive anomaly detection.
3. within-vehicle historical baselines.
4. degradation/trend detection.
5. cross-vehicle fleet baselines.
6. maintenance forecasting.
7. AI-generated explanations.
8. confidence calibration.

Avoid sophisticated ML until enough real labeled data exists to justify it.

---

# 8. Canonical vehicle ontology

Create stable names that are independent of manufacturer terminology.

Examples:

```text
vehicle.speed
electrical.system_voltage
engine.speed
engine.coolant.temperature
engine.oil.temperature
engine.load
engine.boost.requested
engine.boost.actual
engine.boost.error
engine.bank1.fuel_trim.long
transmission.oil.temperature
transmission.current_gear
network.module.response_latency
network.module.online
diagnostics.dtc
```

Manufacturer-specific mapping:

```text
BMW DID X
   ┐
Alfa DID Y
   ├──→ engine.oil.temperature
Ford PID Z
   ┘
```

The canonical layer enables analytics to be reused between brands.

---

# 9. Vehicle Pack architecture

A Vehicle Pack should be a versioned artifact containing the knowledge needed to translate a supported vehicle into the canonical model.

Suggested structure:

```text
vehicle-pack/
├── manifest.json
├── identification.json
├── modules.json
├── networks.json
├── signals.json
├── requests.json
├── decoders.json
├── polling.json
├── dtcs.json
├── thresholds.json
├── rules.json
└── tests/
```

## Manifest

Should define:

- manufacturer.
- model.
- model years.
- trim.
- engine.
- transmission.
- region.
- network variants.
- required adapter capability.
- security/gateway requirements.
- support tier.
- pack version.

## Signal definition

Should define:

- canonical signal ID.
- source ECU.
- request type.
- identifier.
- decode formula.
- units.
- valid range.
- preferred sample rate.
- prerequisites.
- priority.
- provenance/licensing.
- validation status.

## Pack provenance

Every definition should know where it came from:

- standardized OBD.
- OEM documentation.
- licensed dataset.
- validated ODX/PDX data.
- documented adapter feature.
- internally reverse-engineered observation.
- community contribution.

Do not mix unverified community discoveries with production-grade definitions without labeling.

---

# 10. Standards and external diagnostic data

The platform should be designed to ingest or map established automotive diagnostic formats where licensing permits.

Important concepts to plan for:

- standardized OBD / OBD-on-UDS.
- ISO 14229 UDS.
- CAN / ISO 15765.
- ODX/PDX diagnostic data.
- J2534 pass-through interfaces.
- DoIP for newer vehicle architectures.
- secure gateway/authenticated diagnostic access.
- emerging service-oriented vehicle diagnostics.

ASAM ODX is particularly relevant conceptually because it represents diagnostic requests/responses, parameters, DTCs, ECU variants, and communication configuration in machine-readable form.

Do not build the enterprise architecture around reverse engineering alone.

## 10.1 Accessory ecosystem/commercial dependency

At enterprise scale, adapter support includes both protocol engineering and ecosystem/commercial dependencies. The MX+ is a concrete example: its iOS path is MFi/iAP over Bluetooth Classic and requires manufacturer whitelisting of third-party applications. This must be tracked as part of an adapter's support lifecycle, alongside technical compatibility.

For every adapter family, record:

- transport technology;
- required OS framework;
- vendor approval/whitelisting requirements;
- App Store or Play Store metadata obligations;
- SDK/license terms;
- protocol identifiers/entitlements;
- supported physical buses/pins;
- firmware requirements;
- supported diagnostic standards;
- security/pairing requirements.

This prevents the enterprise product from becoming accidentally dependent on an undocumented or non-scalable integration path.

---

# 11. Adapter abstraction

Do not make OBDLink MX+ a permanent architectural dependency.

Conceptual transport implementations:

```text
OBDLinkMXPlusExternalAccessory
OBDLinkCXBLE
GenericBLESerialAdapter
WiFiOBDAdapter
J2534PassThru
DoIPTransport
DirectCANResearchInterface
```

Shared contract:

```text
connect()
disconnect()
read()
write()
capabilities()
metadata()
health()
```

Capabilities should be queried rather than assumed.

Example capability model:

```json
{
  "raw_can": true,
  "iso15765": true,
  "uds": true,
  "multi_bus": false,
  "can_fd": false,
  "background_mobile": true
}
```

Vehicle packs can declare minimum transport capabilities.

## 11.1 Adapter certification/provisioning metadata

Enterprise adapters can have nontechnical onboarding requirements that must be modeled separately from runtime capabilities. The MX+ demonstrates this clearly: OBD Solutions requires third-party iOS applications to be whitelisted using the app name, version, bundle identifier, and functional overview. Apple App Store submission also requires the associated MFi Product Plan ID.

Add an adapter integration manifest concept such as:

```json
{
  "adapter_family": "obdlink_mx_plus",
  "mobile_platform": "ios",
  "transport": "external_accessory_iap",
  "protocol_string": "com.obdlink",
  "mfi_product_plan_id": "221699-0001",
  "manufacturer_whitelisting_required": true,
  "app_store_submission_metadata_required": true
}
```

Do not assume every future adapter is simply discoverable through a generic Bluetooth API. Adapter integrations may require MFi participation, vendor whitelisting, SDK licensing, authentication, certificates, or commercial agreements.

## 11.2 MX+ confirmed capability profile

The first adapter capability profile can be seeded from OBD Solutions' developer documentation:

```json
{
  "adapter_family": "obdlink_mx_plus",
  "raw_can_iso11898": true,
  "iso15765": true,
  "j1939": true,
  "iso9141": true,
  "iso14230": true,
  "j1850_vpw": true,
  "j1850_pwm": true,
  "hs_can_pins": [6, 14],
  "ms_can_pins": [3, 11],
  "sw_can_pins": [1],
  "bluetooth_classic": true,
  "ble": false
}
```

Physical adapter support is not equivalent to vehicle-wide module reachability; vehicle network routing, gateway policy, connector pin assignments, and required multiplexing must remain properties of the vehicle/network definition.

---

# 12. Polling planner

Enterprise support requires a scheduler rather than hard-coded loops.

Inputs:

- requested signal set.
- signal priority.
- ECU/module.
- allowed poll rate.
- transport throughput.
- protocol overhead.
- module/session requirements.
- current vehicle state.
- safety constraints.

Output:

- efficient polling plan.

The planner should prefer:

1. passive broadcast signal when validated.
2. grouped/multi-signal requests where supported.
3. active requests at the lowest rate sufficient for the diagnostic objective.

It must protect the vehicle network from unnecessary diagnostic traffic.

---

# 13. Data tiers

Do not upload everything forever.

## Tier A — raw evidence

High-volume:

- raw frames.
- raw adapter transactions.
- high-rate telemetry.

Retention can be short unless attached to an incident.

## Tier B — normalized telemetry

Medium-volume:

- decoded canonical signals.
- quality metadata.

Longer retention.

## Tier C — features

Low-volume:

- per-drive aggregates.
- distributions.
- warm-up time.
- voltage statistics.
- boost error distribution.
- response-latency percentiles.

Long retention.

## Tier D — findings

Very low-volume:

- incidents.
- trends.
- maintenance findings.
- AI reports.

Permanent/user-visible history.

---

# 14. Cloud architecture — future

Do **not** implement during the Alfa MVP, but preserve a path toward it.

Conceptual services:

```text
Mobile App
   │
   │ encrypted sync
   ▼
API / Ingestion
   │
   ├── Session metadata DB
   ├── Object storage for logs
   ├── Time-series/analytics store
   └── Queue/event stream
             │
             ▼
      Analytics workers
             │
             ▼
       Findings store
             │
             ▼
      Report generation
```

Go is a strong candidate for:

- public API.
- ingestion service.
- account/session services.
- report delivery.
- fleet metadata services.

Python is a strong candidate for:

- telemetry processing.
- anomaly jobs.
- feature generation.
- model training/inference.
- AI report assembly.

---

# 15. Privacy and security

Consumer vehicle telemetry can reveal sensitive behavior even without explicit GPS.

Treat telemetry as sensitive user data.

Enterprise requirements should eventually include:

- explicit consent.
- data minimization.
- encryption at rest/in transit.
- account-scoped authorization.
- vehicle ownership/association model.
- retention controls.
- delete/export capabilities.
- no sale of raw driving data.
- clear distinction between local-only and cloud features.
- audit trail for any active vehicle command.
- security review for adapter communications.

Location should be opt-in and not required for core diagnostics.

---

# 16. AI reporting architecture

Weekly report pipeline:

```text
Sessions
   ↓
Data quality checks
   ↓
Weekly feature aggregation
   ↓
Vehicle baseline comparison
   ↓
Model/fleet baseline comparison
   ↓
Rules/anomaly engine
   ↓
Structured findings JSON
   ↓
Evidence selection
   ↓
LLM
   ↓
User report
```

Example structured finding:

```json
{
  "finding_id": "charging_voltage_trend",
  "category": "electrical",
  "severity": "attention",
  "confidence": 0.91,
  "current": {
    "warm_idle_voltage_median": 13.38
  },
  "baseline": {
    "warm_idle_voltage_median": 13.61
  },
  "change": -0.23,
  "supporting_sessions": 9,
  "related_incidents": [
    "uuid-1",
    "uuid-2"
  ]
}
```

The LLM gets the finding and evidence summary, not unrestricted control over raw data interpretation.

The report should explicitly distinguish:

- observation.
- statistically detected change.
- diagnostic hypothesis.
- maintenance recommendation.
- urgency.

---

# 17. Baseline hierarchy

Use three baselines when available.

## A. Vehicle-specific baseline

"What is normal for this exact car?"

This should have the highest relevance for gradual degradation.

## B. Cohort baseline

"What is normal for similar vehicles?"

Cohort dimensions:

- make/model.
- model year range.
- engine.
- transmission.
- software/calibration if known.
- mileage bands.
- climate/ambient conditions where appropriate.

## C. engineering/specification threshold

"Is this outside a known safe/expected range?"

These baselines answer different questions and should not be conflated.

---

# 18. Incident correlation engine

A long-term differentiator should be ordered causal evidence.

Incident graph example:

```text
Electrical voltage delta
        │
        ▼
ECU response latency rises
        │
        ▼
TCM timeout
        │
        ├─────────────┐
        ▼             ▼
ABS communication   Cluster warning
DTC                 message
```

Store:

- temporal ordering.
- confidence.
- evidence source.
- competing explanations.

Do not automatically claim causation merely because A preceded B.

Use phrases such as:

- preceded.
- correlated with.
- repeatedly co-occurred.
- likely initiating event.
- causal relationship not established.

---

# 19. Vehicle support rollout strategy

Do not start with "all cars."

Recommended sequence:

## Phase 1
2017 Alfa Romeo Giulia Quadrifoglio — one real reference car.

## Phase 2
Giulia/Stelvio variants sharing architecture/powertrain where data supports reuse.

## Phase 3
One high-volume enthusiast platform with strong diagnostic documentation/community.

Selection criteria:

- meaningful owner interest.
- available diagnostic definitions.
- compatible adapter access.
- sufficient test vehicles.
- no prohibitive secure-gateway limitation for read operations.

## Phase 4
Generic standardized OBD support across broad vehicle population.

This could be released earlier technically, but deep support should remain the differentiating feature.

## Phase 5
Additional deep-support Vehicle Packs.

---

# 20. Business/product support matrix

Each vehicle should expose a machine-readable support matrix.

Example:

```text
2017 Giulia Quadrifoglio

Universal OBD                 VERIFIED
Engine deep telemetry         BETA
Transmission telemetry        BETA
Network health                BETA
ABS                           NOT SUPPORTED
TVM                           NOT SUPPORTED
Body network                  NOT SUPPORTED
Weekly health report          BETA
Incident reconstruction       VERIFIED
Active service functions      NOT OFFERED
```

This avoids pretending all brands are equally supported.

---

# 21. Data validation program

Enterprise credibility will depend on validation more than feature count.

For every signal:

1. known source/provenance.
2. decoding unit tests.
3. plausible range tests.
4. static/bench validation.
5. comparison to trusted scan tool/OEM data.
6. road validation.
7. software/version tagging.

For every diagnostic rule:

1. documented rationale.
2. positive test cases.
3. negative test cases.
4. false-positive review.
5. confidence score.
6. vehicle applicability constraints.

---

# 22. Engineering environments

## Mobile
- Native SwiftUI + thin ExternalAccessory bridge, shared Go XCFramework.
- Local development in Windows/WSL; no personal macOS hardware assumed.
- **Compile on GitHub Actions macOS runners** using Xcode. Build/ZIP unsigned `.app` bundles for iOS Simulator; publish them as CI artifacts.
- **Use Appetize.io for browser-hosted iOS simulator smoke tests** of mock UI/flows. Automated Appetize uploads are optional, authenticated with repository secrets; keep a manual artifact-upload path.
- **Use a separate signed device distribution pipeline later** (for example TestFlight) to test `EASession`, actual MX+ communication, real background logging and Bluetooth reconnect behavior on the physical iPhone.
- Appetize / iOS Simulator cannot validate physical accessory sessions, in-vehicle reliability, power management or driver safety behavior.
- Preserve one application/UI codebase; do not add a web UI solely for Windows-based previewing.

## Go
- cross-platform core tests run independently of Xcode.
- fuzz tests for parsers.
- golden test fixtures for protocol responses.

## Python
- pyproject-based package.
- pandas/polars as justified later.
- notebooks for research only.
- production analytics implemented as tested modules, not notebook-only logic.

## Vehicle research
Maintain sanitized/golden captures:

```text
captures/
manufacturer/model/year/scenario/
```

Do not commit proprietary/licensed diagnostic data to public repositories.

---

# 23. Observability for the product itself

The recorder needs to know when **it** is failing.

Capture:

- adapter connection uptime.
- bytes/frames received.
- request success rate.
- request latency.
- parser errors.
- dropped samples.
- queue depth.
- storage errors.
- app background/foreground transitions.
- Go bridge errors.
- current vehicle pack.
- decoder version.

A diagnostic product that silently loses data is worse than one that clearly says coverage was incomplete.

---

# 24. Reliability requirements before scale

Before supporting additional vehicles, the Alfa implementation should demonstrate:

- repeatable multi-hour recording.
- recovery from adapter disconnect.
- recovery from app relaunch.
- storage recovery after unexpected termination.
- deterministic incident windows.
- low or known battery impact on phone.
- no observable vehicle-network disruption.
- data-quality scoring.
- versioned schemas.
- reproducible Python analysis.

Only then should broad vehicle support become an engineering priority.

---

# 25. Enterprise evolution roadmap

## Stage 0 — personal test tool

Goal:
Diagnose intermittent issues on one Giulia.

Build:
- mock transport.
- recorder.
- incidents.
- MX+ connection.
- real drives.
- Python reconstruction.

## Stage 1 — Alfa product prototype

Goal:
Reliable deep support for a narrow Alfa configuration.

Build:
- validated drivetrain pack.
- baseline analytics.
- weekly deterministic report.
- simple AI explanation.
- vehicle health history.

## Stage 2 — platform extraction

Goal:
Prove that the architecture is not Alfa-specific.

Build:
- second vehicle pack.
- shared canonical ontology.
- adapter capability model.
- pack validation tooling.
- migration/versioning system.

A second vehicle is the true test of abstraction.

## Stage 3 — cloud/fleet intelligence

Goal:
Learn across vehicles.

Build:
- secure ingestion.
- cohort baselines.
- anonymized aggregate statistics.
- cloud analytics.
- remote reports.

## Stage 4 — commercial consumer application

Goal:
Support multiple makes with explicit coverage tiers.

Build:
- onboarding/VIN identification.
- compatibility matrix.
- account/billing.
- report delivery.
- support tooling.
- privacy controls.
- production telemetry and incident infrastructure.

## Stage 5 — advanced predictive health

Goal:
Detect degradation before DTCs occur where evidence supports it.

Build:
- validated degradation models.
- maintenance outcome feedback.
- parts/service correlation.
- confidence-calibrated recommendations.

---

# 26. Architectural decisions to postpone

Do not lock these in during MVP:

- cloud provider.
- production time-series database.
- Kafka vs other queue.
- GraphQL vs REST.
- final mobile persistence engine.
- on-device vs cloud anomaly models.
- final LLM provider.
- subscription model.
- data-retention pricing.
- final Vehicle Pack distribution mechanism.

The Giulia project should produce evidence to inform these choices.

---

# 27. Key enterprise risks

## Vehicle-definition availability

The main scaling bottleneck is manufacturer-specific diagnostic knowledge.

Mitigation:
- structured vehicle packs.
- provenance.
- licensing strategy.
- ODX/PDX ingestion where available.
- narrow verified support.

## Secure gateways

Newer vehicles may require authorized access even for certain diagnostics.

Mitigation:
- capability detection.
- supported gateway integrations.
- avoid bypass strategies.
- maintain read-only fallback.

## False diagnosis

Incorrect maintenance advice can be expensive.

Mitigation:
- evidence-first architecture.
- confidence scores.
- deterministic findings.
- explicit uncertainty.
- validation fleet.
- never let an LLM invent sensor evidence.

## Vehicle-network interference

Aggressive polling can create faults.

Mitigation:
- safe polling planner.
- passive-first strategy.
- rate limits.
- per-module validation.
- transport health monitoring.

## Data volume

Raw CAN can generate huge datasets.

Mitigation:
- tiered storage.
- incident-preserved raw data.
- feature aggregation.
- configurable retention.

---

# 28. Near-term next steps

Do these in order.

0. **Milestone 0.5:** verify a GitHub macOS simulator build and Appetize browser preview of the existing SwiftUI shell, without changing the Go/Python domain boundaries.
1. Complete the Alfa mock recorder and run it through the same simulator-preview pipeline.
2. Integrate the already-approved MX+ on a **signed physical-iPhone build** after separate device testing is prepared; simulator previews do not substitute for this.
3. Capture baseline healthy drives.
4. Validate data against existing scan tools.
5. Capture at least one real intermittent incident.
6. Build the first Python incident reconstruction.
7. Decide which signals actually matter diagnostically.
8. Create the first deterministic weekly summary.
9. Build several weeks of vehicle-specific baseline data.
10. Only then add an AI narrative layer.
11. Add a second vehicle only after the first architecture is stable.

---

# 29. Living-document questions

As the project evolves, add decisions here.

- Which Giulia main-bus signals can be passively decoded reliably?
- Which ECM/TCM manufacturer requests are safe during driving?
- What throughput can the MX+ sustain in this specific configuration?
- Does iOS keep the ExternalAccessory session reliably alive for full drives on a **physical** iPhone (not Appetize)?
- Are macOS CI simulator builds fully reproducible from a clean WSL-developed checkout?
- Does the current GitHub Actions + Appetize setup adequately test the mock-driven UI, or is another test service justified?
- What signal sampling rates are diagnostically useful vs wasteful?
- How much data does a typical hour of driving generate?
- Which incident features actually identify the initiating failure?
- What false-positive rate is acceptable?
- Which second vehicle best tests the Vehicle Pack abstraction?
- What diagnostic-data sources can be licensed for broader brand coverage?
- When is cloud processing justified over local-only analysis?

This section should be updated after each significant field-testing phase.

---

# 30. Current strategic recommendation

Do not attempt enterprise features yet.

Build the Alfa recorder so that its **interfaces are enterprise-shaped**, while its **scope remains personal and narrow**.

The most valuable near-term output is not a polished consumer application. It is a trustworthy dataset containing:

- normal drives.
- known mock failure cases.
- real intermittent incidents.
- raw evidence.
- normalized signals.
- DTC timelines.
- network-health timelines.
- verified interpretations.

Once the system can accurately explain a real event on the reference Giulia, the project will have evidence that the architecture is useful. The second supported vehicle will then prove whether the architecture is genuinely portable.
