# Alfa Flight Recorder — MVP Implementation Plan
## Target vehicle: 2017 Alfa Romeo Giulia Quadrifoglio
### Purpose
Build a small iOS test application that continuously records drivetrain telemetry and reconstructs intermittent incidents on the user's 2017 Alfa Romeo Giulia Quadrifoglio.

The first release is **not** a general-purpose scan tool and is **not** intended to replace MultiECUScan. Its narrow mission is:

> When an electrical, communication, or mechanical incident occurs while driving, preserve enough context before, during, and after the event to identify the most likely initiating failure.

This document is intended to be given directly to Codex as the implementation plan.

**Project status (October 2026):** Milestone 0 (repository/bootstrap) is complete. Apple/OBD Solutions have approved **Vehicle Flight Recorder** for the OBDLink MX+. Development machines are **Windows + WSL**; the only Apple hardware currently available is an **iPhone**. Complete **Milestone 0.5 — remote simulator build and browser preview** before proceeding with Milestone 1. Do not redo Milestone 0 or assume local access to macOS/Xcode.

---

# 1. Product goals

## 1.1 MVP goals

The test application must:

1. Run on iPhone.
2. Work without an actual OBDLink MX+ connection during initial development.
3. Abstract the physical adapter behind a transport interface.
4. Generate realistic mock drives and mock incidents.
5. Continuously record normalized telemetry.
6. Maintain a rolling pre-incident buffer.
7. Allow the driver to manually mark an incident.
8. Automatically create incidents from configured triggers.
9. Preserve all relevant samples surrounding an incident.
10. Show current connection/recording health in a simple dashboard.
11. Store complete drive sessions locally.
12. Export sessions in a format Python can consume easily.
13. Keep Alfa-specific definitions separate from the generic recorder.
14. Be designed so the real OBDLink transport can be inserted later without rewriting the application.
15. Support hands-off recording on the physical iPhone after initial setup, without requiring the user to open the app or press Start on every trip. Use validated vehicle-activity evidence rather than scanner connection alone (see §15 and Milestone 7).

## 1.2 Diagnostic-first scope and future product boundary

**Planning clarification — October 28, 2026:** The MVP exists to investigate the
owner's immediate intermittent vehicle problem. It remains diagnostic-first.
The generic recorder and standard OBD discovery are shared infrastructure; the
2017 Giulia QV is the first physical validation vehicle, not a dependency embedded
in the generic implementation. Verified Alfa-specific requests, decoders,
applicability and diagnostic rules belong in the vehicle pack.

The intended progression is:

1. Generic diagnostic foundation and supported standard-signal discovery.
2. Reliable physical-device recording and evidence-quality characterization.
3. Verified Alfa drivetrain expansion needed to investigate the immediate fault.
4. A separately planned Driver Companion V2, using demonstrated capabilities.
5. Enterprise expansion when evidence and product needs justify it.

Do not move Alfa diagnostic work behind trip/performance features. Conversely,
do not make the MVP depend on exhaustive proprietary-signal discovery. Prioritize
verified signals that answer the diagnostic questions. Capturing a real fault is
valuable evidence, but its unpredictable occurrence must not prevent testing
capture/reconstruction with controlled fixtures. Successful infrastructure tests
do not establish that the user's real fault has been diagnosed.

Generic support means discovering which implemented standard parameters the
connected vehicle reports, not assuming every standardized parameter exists on
every vehicle. Testing on the Alfa validates that configuration only. Broader
compatibility still needs representative vehicle/protocol testing.

GPS routes, motion-derived G measurements, performance timing, and a complete
cockpit-inspired consumer redesign belong to V2. Basic MVP usability improvements
remain in scope. This clarification does not authorize V2 implementation.

## 1.3 Explicit non-goals for the first iteration

Do **not** implement these yet:

- ELMA5 network access.
- ELMA6 network access.
- ABS/TVM/EPS/airbag polling.
- ECU coding or configuration.
- DTC clearing.
- actuator commands.
- CAN message reverse engineering.
- cloud infrastructure.
- user accounts.
- subscriptions.
- Public App Store release work beyond the minimum MFi/whitelisting configuration needed for development.
- generalized multi-brand support.
- LLM integration inside the iOS application.
- continuous GPS tracking.
- predictive maintenance models.

The first milestone is a reliable flight recorder with mocked vehicle data.

---

# 2. Technology strategy

The developer is comfortable with Go and Python but is not a native Swift developer. Minimize Swift-specific business logic.

## 2.1 Responsibility split

### Swift / SwiftUI
Swift should own only functionality that must be native to iOS:

- application lifecycle.
- SwiftUI screens.
- app permissions/capabilities.
- ExternalAccessory integration for the OBDLink MX+ using the confirmed iAP protocol `com.obdlink`.
- background execution configuration.
- file-sharing/export UI.
- notifications/haptics if later needed.
- a thin bridge between native transport bytes and the Go core.

### Go
Go should own the persistent domain/recorder logic:

- canonical telemetry types.
- vehicle profile loading.
- signal definitions.
- drive-session lifecycle.
- rolling telemetry buffers.
- incident detection.
- event correlation.
- mock vehicle generation.
- telemetry validation.
- normalized output.
- log-file creation.
- session metadata.
- health snapshots.
- adapter-agnostic command scheduling later.
- OBD/STN parsing later.

Compile the Go package for iOS using `gomobile bind`, which currently creates an XCFramework for Apple targets.

Keep the Swift ↔ Go API deliberately small. Prefer strings, byte arrays, integers, and JSON boundaries instead of exposing a large object hierarchy across the language bridge.

### Python
Python should own post-drive and weekly analysis:

- reading exported sessions.
- dataframe/time-series analysis.
- baselining.
- trend calculations.
- anomaly detection.
- incident reconstruction.
- chart/report generation.
- later LLM/AI report preparation.
- research notebooks and rapid diagnostic experiments.

Python does **not** need to run on the iPhone for the MVP.

## 2.2 Why this split

The physical MX+ connection requires Apple-native External Accessory APIs. OBD Solutions' March 25, 2026 iOS developer documentation confirms that the MX+ uses Bluetooth Classic v3.0 with Apple's MFi/iAP path rather than Bluetooth LE/Core Bluetooth. The documented accessory details are:

```text
Product Plan ID:       221699-0001
iAP protocol string:   com.obdlink
Accessory name:        OBDLink MX+
Manufacturer:          OBD Solutions LLC
Model number:          MX201
Bluetooth:             Classic v3.0 (no BLE)
```

Apple-native responsibilities therefore include discovery through `EAAccessoryManager`, verifying that the accessory advertises `com.obdlink`, and opening an `EASession` for that protocol. The application must declare `com.obdlink` under `UISupportedExternalAccessoryProtocols`.

The iOS app should also enable the `external-accessory` background execution mode so an active accessory session can continue delivering data while the UI is not foregrounded, subject to iOS lifecycle behavior.

Everything after the `EASession` input/output streams must remain independent from Apple frameworks.

Useful references:

- Apple External Accessory:
  https://developer.apple.com/documentation/externalaccessory
- Apple background execution modes:
  https://developer.apple.com/documentation/xcode/configuring-background-execution-modes
- Go mobile binding:
  https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile

## 2.3 Confirmed OBDLink MX+ MFi application identity

Use the following **approved** application identity:

```text
App Store name:       Vehicle Flight Recorder
Initial version:      0.1.0
CFBundleIdentifier:   com.kaleevans.vehicleflightrecorder
MFi PPID:             221699-0001
iAP protocol:         com.obdlink
```

OBD Solutions requires the app name, version, bundle identifier, and a general functional overview in order to whitelist the iOS application for MX+ use. The current functional description should emphasize:

- read-only diagnostics and telemetry;
- continuous drivetrain/electrical logging;
- incident capture and reconstruction;
- DTC history without automatic clearing;
- export/post-drive analytics;
- no ECU programming, coding, calibration changes, or safety-critical control.

**Whitelisting/approval is complete.** Treat the bundle identifier as fixed. Do not casually rename it during development or assume simulator-only builds prove the whitelisted hardware connection works.

The OBDLink documentation notes that whitelisting does not appear to prevent development/TestFlight distribution, but Apple requires the MFi Product Plan ID `221699-0001` to be included in the Release Notes field when the app is submitted for App Store approval.

---

# 3. High-level MVP architecture

```text
┌────────────────────────────── iOS Application ──────────────────────────────┐
│                                                                            │
│   SwiftUI                                                                  │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │
│   │ Dashboard   │  │ Incidents   │  │ Sessions    │  │ Settings/Debug  │  │
│   └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └────────┬────────┘  │
│          └─────────────────┴─────────────────┴──────────────────┘           │
│                                      │                                     │
│                              Swift Go bridge                               │
│                                      │                                     │
│   ┌──────────────────────────────────▼──────────────────────────────────┐   │
│   │                         Go Telemetry Core                           │   │
│   │                                                                     │   │
│   │  Session Manager      Rolling Buffer       Incident Engine          │   │
│   │  Signal Registry      Mock Vehicle         File Recorder            │   │
│   │  Vehicle Profile      Event Timeline       Health Snapshot          │   │
│   └──────────────────────────────────┬──────────────────────────────────┘   │
│                                      │                                     │
│                         normalized events / bytes                           │
│                                      │                                     │
│   ┌──────────────────────────────────▼──────────────────────────────────┐   │
│   │                       Vehicle Transport                             │   │
│   │                                                                     │   │
│   │   MVP: MockTransport                                                │   │
│   │   Next: MXPlusExternalAccessoryTransport (`com.obdlink`)                           │   │
│   └─────────────────────────────────────────────────────────────────────┘   │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘

                              Session export
                                   │
                                   ▼
                         Python Analytics Package
```

---

# 4. Repository layout

Use a monorepo.

```text
alfa-flight-recorder/
├── README.md
├── docs/
│   ├── MVP_PLAN.md
│   ├── DATA_SCHEMA.md
│   ├── VEHICLE_PROFILE.md
│   └── INCIDENT_MODEL.md
│
├── ios/
│   ├── AlfaFlightRecorder.xcodeproj
│   └── AlfaFlightRecorder/
│       ├── App/
│       ├── Views/
│       ├── ViewModels/
│       ├── Transport/
│       ├── Bridge/
│       ├── Export/
│       └── Resources/
│
├── core/
│   ├── go.mod
│   ├── mobile/
│   ├── telemetry/
│   ├── recorder/
│   ├── incidents/
│   ├── vehicle/
│   ├── mock/
│   └── storage/
│
├── vehicle-packs/
│   └── alfa-giulia-qv-2017/
│       ├── manifest.json
│       ├── signals.json
│       ├── polling.json
│       ├── thresholds.json
│       └── scenarios/
│
├── analytics/
│   ├── pyproject.toml
│   ├── src/
│   │   └── alfa_analytics/
│   ├── notebooks/
│   └── tests/
│
├── scripts/
│   ├── build_go_ios.sh
│   ├── generate_mock_session.py
│   └── validate_session.py
│
└── testdata/
    ├── normal_drive/
    ├── voltage_tcm_dropout/
    ├── mechanical_fault/
    └── adapter_disconnect/
```

Keep the vehicle pack data-driven wherever practical. Avoid hard-coding "Giulia" conditions deep inside generic Go packages.

## 4.1 iOS project configuration required now

Even before a real MX+ session is opened, configure the Xcode project around the final transport contract.

### Info.plist

Declare the MX+ protocol:

```xml
<key>UISupportedExternalAccessoryProtocols</key>
<array>
    <string>com.obdlink</string>
</array>
```

### Background mode

Enable the External Accessory background capability (`external-accessory`).

### Bundle identity

Use:

```text
com.kaleevans.vehicleflightrecorder
```

This bundle identifier is approved/whitelisted and must remain unchanged unless new approval is obtained.

### Development transport implementations

Create these Swift types from the start:

```text
VehicleTransport                    protocol
MockVehicleTransport               implementation used immediately
MXPlusExternalAccessoryTransport   real implementation, initially feature-flagged/offline
```

The real MX+ transport must not contain OBD parsing or vehicle-specific logic. Its responsibilities are only:

- enumerate connected `EAAccessory` instances;
- match the accessory name/model where useful;
- verify `protocolStrings` contains `com.obdlink`;
- open/close `EASession(accessory:forProtocol:)`;
- schedule/open the session input/output streams;
- write outbound bytes supplied by the Go core;
- forward inbound bytes to the Go core;
- publish connection/disconnection/errors/stream health;
- reconnect conservatively after accessory loss;
- never issue diagnostic commands by itself.

---

# 5. Swift/Go API boundary

Keep this very small.

A suggested initial exported Go API:

```text
NewRecorder(configJSON string) -> Recorder

Recorder.StartSession(sessionJSON string, outputDirectory string) -> error
Recorder.StopSession() -> string
Recorder.PushObservation(observationJSON string) -> error
Recorder.MarkIncident(label string, timestampUnixMs int64) -> string
Recorder.GetDashboardJSON() -> string
Recorder.GetIncidentListJSON() -> string
Recorder.GetSessionStateJSON() -> string
Recorder.RunMockScenario(name string) -> error
Recorder.StopMockScenario() -> error
```

Later, when the MX+ is integrated:

```text
Recorder.PushTransportBytes(data []byte) -> error
Recorder.DrainOutboundBytes(maxBytes int) -> []byte
Recorder.OnTransportConnected(metadataJSON string)
Recorder.OnTransportDisconnected(reason string)
```

Do not expose Go structs directly to Swift unless necessary. JSON is acceptable for this MVP because telemetry sample rates are still modest and it keeps the cross-language boundary simple.

If performance later becomes a problem, replace hot paths with byte/binary payloads without changing the overall architecture.

## 5.1 `MXPlusExternalAccessoryTransport` implementation contract

The first real transport implementation should use Apple's External Accessory framework only; do not implement Core Bluetooth for the MX+.

Pseudo-flow:

```text
EAAccessoryManager.shared().connectedAccessories
        ↓
find OBDLink MX+ supporting `com.obdlink`
        ↓
EASession(accessory: accessory, forProtocol: "com.obdlink")
        ↓
open input/output streams
        ↓
inbound bytes ─────────────→ Go Recorder.PushTransportBytes(...)
outbound bytes ←──────────── Go Recorder.DrainOutboundBytes(...)
```

Implementation requirements:

1. Treat stream callbacks as arbitrary byte chunks; never assume one callback equals one OBD/STN response.
2. Preserve byte ordering.
3. Keep parser framing in Go, not Swift.
4. Record transport-level disconnects/errors as events.
5. Make reconnect behavior stateful so an interruption is distinguishable from an ECU timeout.
6. Allow the entire implementation to be replaced with `MockVehicleTransport` via dependency injection.
7. Keep all outbound vehicle activity disabled behind an explicit development feature flag until the command layer is tested with mocked transcripts.

## 5.2 MX+ hardware capabilities relevant to the MVP

The March 2026 OBDLink iOS developer document confirms these physical/protocol capabilities:

```text
HS-CAN:       pins 6 & 14
MS-CAN:       pins 3 & 11
SW-CAN:       pin 1
ISO/KWP:      pins 7 & 15
J1850:        pins 2 & 10

ISO 15765     CAN diagnostics
ISO 11898     raw CAN
ISO 14230     KWP2000
ISO 9141
SAE J1939
SAE J1850 VPW/PWM
```

For the first Giulia phase, remain scoped to the primary drivetrain network and ignore ELMA5/ELMA6 routing. The presence of MS-CAN/SW-CAN capability in the adapter does not imply that every Giulia ECU is directly reachable without the required vehicle-side pin routing/network selection.

---

# 6. Canonical telemetry model

Every observed value should be converted into a normalized representation before storage or analysis.

## 6.1 Observation

```json
{
  "schema_version": 1,
  "timestamp_unix_ms": 1788548400123,
  "session_id": "uuid",
  "signal_id": "powertrain.engine.rpm",
  "source": "mock",
  "module": "ECM_1",
  "value": 1843.0,
  "unit": "rpm",
  "quality": "valid"
}
```

Required fields:

- schema_version
- timestamp
- session_id
- signal_id
- source
- value
- unit
- quality

Optional:

- module
- bus
- raw_request
- raw_response
- decoder_version
- vehicle_pack_version

## 6.1.1 Live timing, freshness and future sources

Before live recording, define timestamp semantics explicitly and version any
schema changes. Preserve request-send and response-receive timing where available,
using a monotonic clock for elapsed durations and a documented mapping to session
wall time. Receipt time is not automatically the ECU's measurement time. Preserve
an actual source timestamp when available; otherwise leave measurement time
unknown rather than inventing precision.

Keep decoded observations traceable to raw transactions and source/decoder
versions. A dashboard may retain the last value, but must expose its age and
stale/invalid state. Refreshing the display must not create a new measurement.
Validate freshness against observed acquisition behavior and diagnostic needs.

Document an extension path for independent OBD, GPS and phone-motion sources,
including per-observation provenance, clock alignment and late-arriving data.
The current single-source/ordered-ingestion contract must be revisited explicitly
before adding asynchronous sources. Do not implement GPS, Core Motion, sensor
fusion or a speculative multi-source framework as part of this requirement.

## 6.2 Event

Events are discrete occurrences rather than continuously sampled values.

Examples:

- transport connected.
- transport disconnected.
- diagnostic request timeout.
- ECU response restored.
- new DTC detected.
- DTC status changed.
- voltage threshold crossed.
- user marked incident.
- recording started/stopped.

Example:

```json
{
  "schema_version": 1,
  "timestamp_unix_ms": 1788548405123,
  "event_type": "diagnostic_timeout",
  "module": "TCM",
  "severity": "warning",
  "attributes": {
    "request": "placeholder",
    "timeout_ms": 500
  }
}
```

## 6.3 Incident

An incident groups events and telemetry around a trigger.

```json
{
  "incident_id": "uuid",
  "session_id": "uuid",
  "trigger_time_unix_ms": 1788548405123,
  "trigger_type": "tcm_communication_loss",
  "pre_window_seconds": 60,
  "post_window_seconds": 120,
  "severity": "warning",
  "status": "captured",
  "manual": false
}
```

For the MVP use:

- 60 seconds pre-trigger.
- 120 seconds post-trigger.

Make these configurable.

---

# 7. Storage format

Start with newline-delimited JSON (JSONL/NDJSON), not SQLite.

Reasons:

- trivial to inspect manually.
- trivial for Go to write.
- trivial for Python/Pandas to read.
- append-only and resilient.
- no native database dependency in the first iOS/Go integration.
- schema evolution is easy.

Session directory:

```text
Sessions/
└── 2026-09-04T18-31-22Z_<uuid>/
    ├── session.json
    ├── observations.jsonl
    ├── events.jsonl
    ├── incidents.jsonl
    └── raw/
        └── transport.jsonl
```

`raw/transport.jsonl` should exist from the start, even while mocked. Later it can preserve exact adapter request/response bytes or lines.

Never throw away raw evidence simply because a decoder produced a normalized value.

A future production version can migrate hot telemetry to SQLite/Arrow/Parquet if file size or query latency justifies it.

---

# 8. Giulia QV vehicle profile

Initial profile ID:

```text
alfa.giulia.qv.2017.2_9t
```

The first profile should describe the known vehicle without embedding proprietary ECU requests yet.

Example manifest:

```json
{
  "profile_id": "alfa.giulia.qv.2017.2_9t",
  "manufacturer": "Alfa Romeo",
  "model": "Giulia Quadrifoglio",
  "model_year": 2017,
  "engine": "2.9L twin-turbo V6",
  "network_scope": "primary_drivetrain_only",
  "adapter": "OBDLink MX+",
  "status": "development"
}
```

---

# 9. Initial signal catalog

The following is a **logical monitoring catalog**, not a claim that every item can already be retrieved from the MX+ without manufacturer-specific definitions.

Signal availability must be labeled:

- `standard_obd`
- `adapter`
- `manufacturer_specific`
- `derived`
- `mock_only`

## 9.1 Highest priority: electrical/network health

These are central to the current diagnostic objective.

```text
electrical.adapter_voltage
electrical.system_voltage_if_available
network.adapter_connected
network.rx_bytes_per_second
network.tx_bytes_per_second
network.can_frames_per_second
network.request_latency_ms
network.request_timeout_count
network.ecm1_alive
network.ecm2_alive
network.tcm_alive
network.ecm1_response_latency_ms
network.ecm2_response_latency_ms
network.tcm_response_latency_ms
network.ecu_dropout_count
```

## 9.2 Engine state

```text
powertrain.engine.rpm
powertrain.vehicle.speed
powertrain.engine.calculated_load
powertrain.throttle.position
powertrain.accelerator.position_if_available
powertrain.coolant.temperature
powertrain.intake.temperature
powertrain.manifold.absolute_pressure
powertrain.mass_air_flow_if_available
powertrain.engine.runtime
```

## 9.3 Fuel/combustion

```text
engine.bank1.short_term_fuel_trim
engine.bank1.long_term_fuel_trim
engine.bank2.short_term_fuel_trim
engine.bank2.long_term_fuel_trim
engine.lambda.bank1_if_available
engine.lambda.bank2_if_available
engine.misfire.bank1_if_available
engine.misfire.bank2_if_available
```

## 9.4 Turbo/air

Initially placeholders where manufacturer-specific access is required.

```text
engine.boost.actual
engine.boost.requested
engine.boost.error
engine.turbo.bank1.wastegate_command
engine.turbo.bank2.wastegate_command
engine.turbo.bank1.status
engine.turbo.bank2.status
```

## 9.5 Transmission

Initially profile placeholders until real diagnostic IDs are known.

```text
transmission.current_gear
transmission.requested_gear
transmission.oil_temperature
transmission.input_speed
transmission.output_speed
transmission.torque_converter_slip
transmission.response_latency_ms
transmission.online
```

## 9.6 DTC state

```text
diagnostics.generic.pending_dtc_count
diagnostics.generic.confirmed_dtc_count
diagnostics.ecm1.dtc_set
diagnostics.ecm2.dtc_set
diagnostics.tcm.dtc_set
```

Do not clear DTCs automatically.

---

# 10. Sampling strategy

Do not treat all values equally.

Create sampling classes:

```text
FAST       10–20 Hz
MEDIUM      2–5 Hz
SLOW        0.2–1 Hz
EVENT       on change
PASSIVE     continuous
```

Proposed initial classes:

### FAST
- RPM
- throttle
- accelerator
- vehicle speed
- actual/requested boost when available
- adapter/system voltage

### MEDIUM
- load
- MAP
- MAF
- transmission data
- fuel trims

### SLOW
- coolant temperature
- intake temperature
- engine runtime
- DTC health checks
- module heartbeat

### EVENT
- connection state
- ECU timeout.
- DTC added/removed/status change.
- incident marker.

These classes are planning targets, not promised rates or verified adapter
capabilities. Actual poll rates must be adjusted based on MX+ throughput, ECU
behavior, and the number of active diagnostic requests. In particular, a nominal
10–20 Hz FAST class does not establish that each FAST signal can be sampled at
that rate simultaneously.

## 10.1 Automatic standard-signal discovery and generic fallback

Milestone 6 must implement standard supported-parameter discovery in the shared
Go diagnostic layer. Implement known standard requests and decoders once, then
query the connected vehicle for its supported subset. Preserve responding ECU
identity rather than combining distinct responses into an unexplained value.
Discovery is bounded to implemented standards and documented reads; it is not
brute-force probing for unknown proprietary identifiers.

Keep these states distinct: advertised support, successfully observed/decoded,
not supported, supported but temporarily unavailable, and unknown/not checked.
A timeout is not proof that a signal is unsupported. Do not repeatedly request
known unsupported parameters during normal recording. Revalidate capabilities
when the vehicle or relevant connection configuration changes; cached results
must retain their identity and provenance rather than becoming universal facts.

If no enhanced vehicle pack matches, retain supported generic diagnostics.
Manufacturer-specific requests and rules require an applicable validated pack;
never apply Alfa thresholds or module assumptions to an unidentified vehicle.
The application should expose actual coverage and explain unavailable features.
This fallback is an architectural requirement, not a claim of broad production
vehicle certification during the Alfa MVP.

## 10.2 Capability and evidence-quality report

Export a machine-readable capability report with a readable summary. Associate
it with the tested vehicle/configuration and relevant sessions. Include:

- adapter model, firmware when available, transport and diagnostic protocol;
- vehicle identity when available, its source, and any unresolved identification;
- application, schema, decoder and vehicle-pack versions;
- signal identifier, source ECU, units, provenance and validation status;
- discovery status, attempted/successful requests, failures and decode errors;
- requested interval and achieved successful-observation rate per signal;
- response-latency distribution, inter-arrival gaps and freshness limitations;
- test duration, vehicle operating state and the complete concurrent signal set;
- comparison against a trusted reference and unresolved discrepancies.

Measure a bounded selection individually and under the intended combined polling
workload. Include representative stationary and driving conditions; perform
configuration/discovery work while parked. Do not maximize traffic simply to find
a theoretical ceiling. Retain the tested conservative configuration as an output.
There is no single universal frequency for a signal independent of workload.

Report successful replies separately from changes in returned values. Equal
values may be legitimate steady measurements; neither equal nor changing values
alone proves the ECU's internal sensor refresh rate. Mark that refresh rate
unknown unless there is supporting evidence. A fast log-writing or UI-refresh
loop must not inflate the reported acquisition rate.

## 10.3 Bounded, prioritized polling

Implement a modest scheduler sufficient for the diagnostic MVP, not the entire
enterprise polling platform. Set bounded timeouts, request budgets, priorities
and retry/backoff behavior. A slow or unavailable signal must not indefinitely
stall unrelated recording. Use the lowest useful rates demonstrated by tests;
reduce workload when latency/errors indicate degraded communication.

Preserve the distinction between adapter loss and individual ECU failures.
Record missed coverage and scheduling changes as evidence. Deprioritizing failed
requests must not erase the module dropout that the recorder is meant to capture.
No companion/performance sampling mode may displace required diagnostic coverage
in this MVP.

---

# 11. Incident engine

Create an `IncidentDetector` interface in Go.

```text
Evaluate(observation/event) -> zero or more incident triggers
```

Initial detectors:

## 11.1 Manual marker

User taps **MARK INCIDENT**.

Always create an incident immediately.

## 11.2 Voltage excursion

Mock threshold for testing only:

```text
voltage < configurable lower threshold
OR
rapid negative delta > configurable threshold within N milliseconds
```

Do not treat this test threshold as a validated Giulia diagnostic limit.

## 11.3 ECU timeout sequence

Trigger if:

- TCM fails N consecutive expected responses.
- ECM1 fails N consecutive expected responses.
- ECM2 fails N consecutive expected responses.

## 11.4 Multi-module communication event

Trigger if two or more critical modules become unavailable within a short correlation window.

## 11.5 New DTC

Any newly observed DTC should create or attach to an incident.

## 11.6 Transport disconnect

An unexpected MX+/mock transport disconnect while a drive is active should become an event and possibly an incident.

---

# 12. Rolling buffer behavior

The recorder must continuously retain at least the last 60 seconds of observations/events in memory.

When an incident triggers:

1. snapshot/copy the previous 60 seconds.
2. mark the trigger.
3. continue capturing the next 120 seconds.
4. link overlapping triggers rather than duplicating large windows unnecessarily.
5. persist the incident index.
6. retain the complete full-session log regardless.

The full drive log is authoritative. Incident windows are indexes/views into the full session, not separate truth.

---

# 13. Mock vehicle system

Do this before any MX+ work.

Create deterministic mock scenarios so UI and incident logic can be built and tested without the car.

Required scenarios:

## Scenario A — Healthy drive

10–20 minute compressed/simulated drive containing:

- engine start.
- idle.
- acceleration.
- cruise.
- deceleration.
- stop.
- restart or shutdown.

No incidents.

## Scenario B — Voltage disturbance followed by TCM loss

Expected timeline:

```text
T-10s   normal
T-2s    voltage begins falling
T-500ms abnormal voltage delta
T0      TCM latency rises
T+300ms TCM timeout
T+600ms repeated timeout
T+900ms diagnostic communication event
T+2s    TCM communication restored
```

Expected result:
- one incident.
- initiating observations visible before the timeout.
- voltage event precedes TCM communication loss.

## Scenario C — TCM loss without voltage disturbance

Expected result:
- one communication incident.
- voltage remains within normal mock baseline.
- diagnostic summary must not claim voltage caused it.

## Scenario D — mechanical/turbo anomaly

Simulate:

- increasing requested boost.
- actual boost fails to track.
- error exceeds configured threshold.
- eventual mock DTC.

Expected result:
- powertrain incident distinct from communication incident.

## Scenario E — adapter disconnect

Expected result:
- transport loss event.
- recording/session stays recoverable.
- reconnection can continue the same session or create a clearly defined segment.

## Scenario F — manual incident

User taps MARK INCIDENT during otherwise normal data.

Expected result:
- full pre/post window available.

All scenarios must be deterministic under a fixed random seed.

---

# 14. Initial iOS layout

Keep the UI intentionally sparse.

## 14.1 Dashboard

Primary screen:

```text
┌──────────────────────────────────┐
│ Alfa Flight Recorder             │
│                                  │
│ ● MOCK / ● MX+ / ○ DISCONNECTED  │
│ RECORDING  00:42:18              │
│                                  │
│ Voltage             13.6 V       │
│ Engine RPM          1,842        │
│ Speed               46 mph       │
│ ECM 1               ONLINE       │
│ ECM 2               ONLINE       │
│ TCM                 ONLINE       │
│                                  │
│ Last incident       None         │
│                                  │
│      [ MARK INCIDENT ]           │
│                                  │
│ Sessions  Incidents  Debug       │
└──────────────────────────────────┘
```

Avoid trying to display every signal on this screen.

## 14.2 Incidents

List:

```text
Sep 4  2:14 PM
Communication incident
TCM timeout
2m 17s into drive

Sep 2  5:42 PM
Manual marker
Dashboard warning observed
```

Tap opens incident timeline.

## 14.3 Incident detail

Display:

- trigger.
- severity.
- correlated events.
- pre/post window.
- simple plots for critical signals.
- raw evidence/export link.

For MVP, plots can be minimal or postponed until the recorder is stable.

## 14.4 Sessions

Show:

- start/end.
- duration.
- sample count.
- incident count.
- data size.
- export button.

## 14.5 Debug

Developer-only screen:

- transport.
- mock scenario selector.
- Go core version.
- vehicle pack version.
- recorder state.
- file path.
- observations/sec.
- dropped observations.
- queue depth.
- most recent raw messages.

This screen will be invaluable during MX+ integration.

---

# 15. App lifecycle

For the mock MVP:

```text
launch app
→ load vehicle profile
→ initialize Go recorder
→ select MockTransport
→ user starts drive OR auto-start debug scenario
→ recorder starts session
→ observations stream
→ incidents detected
→ user stops drive
→ recorder closes session
→ session appears in Sessions
```

Later real-device lifecycle:

```text
MX+ becomes available
→ ExternalAccessory identifies it
→ app opens EASession
→ Go core initializes adapter
→ validated engine/drivetrain activity detected
→ session auto-starts without a per-trip app-open or Start action
→ drive logs continuously
→ ignition/vehicle activity ends
→ session closes after grace period
```

**Hands-off recording requirement (user clarification, October 28, 2026):**
After initial installation/setup, the normal experience should not require the
user to open the app each time they start the car. A scanner connection is a
transport event, not sufficient proof that a drive has started: the adapter may
be available with ignition on and engine off.

Use fresh engine and transmission module reporting as candidate activity evidence,
combined with a validated engine-running signal (for example RPM when available).
Module responses alone may also occur with ignition on and engine off, so do not
assume their presence proves that the engine is running. Validate the actual
signals, module reachability, freshness windows, debounce and start/stop grace
periods on the target car before fixing the policy. Do not require vehicle
movement before recording; preserve startup and stationary engine-running data.

Once recording, loss of engine or transmission responses must remain incident
and coverage evidence, rather than immediately stopping the session or silently
suppressing recording. Define a conservative degraded-start policy when one
module is unavailable; do not discard a potential startup fault because both
modules are not reporting. Brief disconnects/reconnects and engine stop/start
must not create duplicate sessions or prematurely end a drive. Expose the actual
recording state, coverage limits, and any required recovery action clearly.

Milestones 5–7 must verify accessory discovery, background/locked-screen operation,
and supported reconnect/wake behavior on the physical iPhone, including the
previously launched, suspended, OS-terminated and user-force-quit app states.
Treat these as distinct test cases; do not promise connection-triggered launch
in every state without evidence. Record platform limitations and any unavoidable
user action explicitly. Simulator tests cannot establish this requirement.

Do not implement automatic ignition inference until real adapter behavior can be observed.

## 15.1 Automatic recording confirmation

**Approved requirement — October 29, 2026:** Confirm an automatically started
drive with a user-facing notification. This is separate from internal accessory
connection notifications and belongs to Milestone 7.

Example wording:

> Drive recording started
> Giulia Quadrifoglio · Recording automatically
> Tap to open the cockpit.

Use the selected vehicle's actual display name, with a generic fallback. Trigger
only after session startup succeeds and the recorder begins accepting telemetry.
Adapter connection, a start attempt, or a failed session must never produce a
successful-recording confirmation. A degraded start must not imply full coverage.

Requirements:

- Issue at most one start confirmation per automatically started session. Use
  session identity to deduplicate across reconnects and app lifecycle transitions.
- Brief adapter loss, engine stop/start, cockpit switching and recovery of the
  same session must not generate another start confirmation.
- While the app is visible, show a brief in-app confirmation instead of a duplicate
  system alert. In supported background states, request a local system notification.
- Tapping opens the active drive/cockpit; if that session has ended, open its
  summary. In the MVP use the existing dashboard/session screens for these routes.
- Explain and request notification permission during setup. Provide an enable/
  disable preference; default to quiet delivery without sound when authorized.
- Permission denial, disabled notifications or delivery failure must not prevent
  recording. Retain visible in-app recording state without repeatedly prompting.
- Respect system notification and Focus settings. Record submission/failure for
  debugging without claiming the user saw the notification or guaranteeing display.

Acceptance: verify successful auto-start, failed start, foreground/background
presentation, permission denial, disabled preference, reconnect/relaunch
deduplication and tapping before/after session end. Use physical-device evidence
for supported background lifecycle behavior. Notification work does not itself
establish automatic wake or recording support in additional iOS lifecycle states.

Reference: [Apple notification authorization](https://developer.apple.com/documentation/usernotifications/asking-permission-to-use-notifications).

---

# 16. OBDLink transport abstraction

Swift protocol:

```swift
protocol VehicleTransport {
    var state: TransportState { get }
    func connect() async throws
    func disconnect()
    func write(_ data: Data) async throws
    var incomingBytes: AsyncStream<Data> { get }
}
```

Implement now:

```text
MockTransport
MXPlusExternalAccessoryTransport   // compile-ready, feature-flagged, no vehicle commands yet
```

Use `MockTransport` as the default runtime transport for simulator/Appetize builds; the approved MX+ transport remains disabled until real-device stream tests are ready.

Do not allow SwiftUI views to refer directly to `EAAccessory`, `EASession`, or OBDLink types.

Only the concrete MX+ transport implementation should know those APIs.

---

# 17. Real MX+ integration plan

The required OBDLink iOS transport information is now known, and app whitelisting/approval is complete. Keep actual diagnostic transmission disabled until EASession stream behavior and safety have been validated on the user's physical iPhone and MX+.

Confirmed OBDLink values:

```text
Product Plan ID:       221699-0001
iAP protocol:          com.obdlink
Accessory name:        OBDLink MX+
Manufacturer:          OBD Solutions LLC
Model number:          MX201
Bluetooth transport:   Classic v3.0 / MFi iAP
```

Implementation sequence:

1. link the ExternalAccessory framework.
2. enable the `external-accessory` background capability.
3. declare `com.obdlink` in `UISupportedExternalAccessoryProtocols`.
4. enumerate `EAAccessoryManager.shared().connectedAccessories`.
5. select an accessory that advertises `com.obdlink`; use name/model metadata only as additional validation rather than as the protocol contract.
6. create `EASession(accessory:forProtocol: "com.obdlink")`.
7. schedule/open input and output streams.
8. forward every inbound byte chunk to the Go core in order.
9. forward outbound bytes produced by the Go command scheduler to the accessory.
10. log raw transport traffic before higher-level parsing where practical.
11. surface stream open/error/end/disconnect states as canonical transport events.
12. add conservative reconnection handling and distinguish adapter disconnects from ECU response failures.

During the current development phase, the concrete transport may establish and validate the accessory/session lifecycle but should not automatically transmit vehicle diagnostic commands. Command transmission should be enabled only after mocked command/response transcripts and parser behavior are covered by tests.

The app's current whitelist identity is:

```text
Name:                 Vehicle Flight Recorder
Version:              0.1.0
CFBundleIdentifier:   com.kaleevans.vehicleflightrecorder
```

The approved app identity must remain stable. Any future identity change would require re-verifying approval with OBD Solutions and updating this document.

For eventual App Store submission, include MFi Product Plan ID `221699-0001` in the Release Notes field as instructed by OBD Solutions.

---

# 18. Go package design

## `telemetry`

Types:

- Observation
- SignalDefinition
- SignalValue
- Quality
- Unit
- ModuleID

## `recorder`

- Recorder
- Session
- RollingBuffer
- Writer
- HealthSnapshot

## `incidents`

- Detector
- Trigger
- Incident
- CorrelationWindow
- detector implementations

## `vehicle`

- VehicleProfile
- SignalRegistry
- PollingPlan
- ThresholdDefinition

## `mock`

- MockVehicle
- Scenario
- ScenarioRunner
- deterministic waveform generators

## `storage`

- JSONL writer.
- session metadata.
- atomic/finalization helpers.
- recovery of an interrupted session.

## `mobile`

A small gomobile-exportable facade around the packages above.

Do not expose internal packages directly to Swift.

---

# 19. Python analytics skeleton

Create this now even though the first report is later.

Functions:

```python
load_session(path)
load_observations(path)
load_events(path)
load_incidents(path)
validate_session(session)
summarize_session(session)
reconstruct_incident(session, incident_id)
```

Initial CLI:

```bash
python -m alfa_analytics validate <session_dir>
python -m alfa_analytics summarize <session_dir>
python -m alfa_analytics incident <session_dir> <incident_id>
```

The output should be deterministic structured JSON plus optional human-readable text.

No AI model is required initially.

---

# 20. Testing requirements

## 20.1 Go unit tests

Minimum:

- rolling buffer expiration.
- incident pre-window retained.
- post-window completion.
- overlapping incident merging/linking.
- JSONL writer.
- interrupted-session recovery.
- manual marker.
- threshold detector.
- module-timeout detector.
- mock determinism.
- vehicle profile validation.

## 20.2 Python tests

- parse exported fixture.
- session sample counts.
- timestamp ordering.
- incident reconstruction.
- missing/corrupt line handling.
- schema-version rejection/compatibility.

## 20.3 Swift tests

Keep minimal:

- view-model state mapping.
- MockTransport connection/disconnection.
- Go bridge invocation.
- session export URL generation.

Most logic should remain testable outside Xcode.

---

## 20.4 Physical-device reliability and acceptance evidence

Milestone 7 must retain an explicit acceptance matrix backed by physical-iPhone,
MX+ and vehicle evidence. Simulator results are insufficient. Cover:

- repeated complete drives, including a representative long-duration session;
- foreground, background and locked-screen recording;
- the distinct wake/termination states and limitations defined in §15;
- ignition-on/engine-off, stationary running, stop/start and end-of-drive grace;
- adapter disconnect/reconnect and individual module response loss;
- absence of duplicate sessions or premature closure during brief interruptions;
- interrupted-session recovery that preserves valid evidence and labels gaps;
- finalized exports retrieved and validated by the Python tools;
- observed phone battery impact, storage growth and raw-log volume per hour;
- request/response health and any observed vehicle-network disturbance.

Record test configuration, duration, expected behavior, observed result, evidence
location and unresolved limitations. Define acceptable battery/storage budgets
from representative measurements before acceptance; do not invent universal
limits. A recoverable incomplete session must never be presented as complete.
Do not intentionally induce a vehicle fault to satisfy an acceptance case; use
controlled transcripts/fixtures for cases unsuitable for physical testing and
label the resulting evidence accordingly.

# 21. Safety constraints

This application should initially be **read-only**.

Rules:

1. Never clear DTCs automatically.
2. Never perform ECU configuration.
3. Never execute actuators while driving.
4. Never transmit undocumented manufacturer-specific commands merely because they were observed elsewhere.
5. Separate passive monitoring from active diagnostic polling.
6. Log every active request with timestamp/module.
7. Rate-limit diagnostic requests.
8. Fail closed: if transport state is uncertain, stop transmitting.
9. The UI should never require driver attention while moving.
10. Marking an incident should be one large tap with haptic confirmation.
11. Analysis must distinguish observed evidence from inference.

---

# 22. MVP milestones

## Milestone 0 — repository/bootstrap

**Status: COMPLETE (per developer; do not regenerate completed scaffolding).**

Deliverables:

- monorepo.
- Xcode SwiftUI app.
- Go module.
- Python package.
- build scripts.
- basic CI if convenient.
- README with setup instructions.

Acceptance:
- iOS app launches.
- `go test ./...` passes.
- Python tests pass.

## Milestone 0.5 — GitHub macOS build + Appetize browser preview

**Status: NEXT.** This is a narrow, independent infrastructure milestone, intentionally placed between the completed bootstrap and Go recorder/mock-data implementation. **Do not renumber or combine it with Milestone 1.**

### Goal and development constraints

Develop locally in `~/dev/VehicleFlightRecorder` under Windows/WSL; compile native SwiftUI using **GitHub Actions on a macOS runner**; open the resulting **iOS Simulator app** from any Windows browser using **Appetize.io**. Retain the native SwiftUI + shared Go + Python design. Do **not** introduce a parallel React/web dashboard merely to solve the lack of a Mac.

A simulator is suitable for checking the minimal app, navigation and mocked UI state. It **cannot** establish Bluetooth Classic/MFi `EASession` access to the physical MX+. Real adapter tests still require a separately signed iOS **device** build installed on the real iPhone (later via TestFlight or another valid provisioned distribution path). A simulator `.app` ZIP and a device `.ipa` are different artifacts; do not interchange them.

### Prerequisites outside the repository

- Existing GitHub remote for the local repo, with Actions enabled; check remote status rather than creating a new repository blindly.
- Sufficient available GitHub Actions minutes on a macOS runner, e.g., `macos-15` (pin/adjust Xcode on that runner only after inspecting what it provides).
- An Appetize.io account and API token **if** automated uploads are desired. Store the token as the GitHub Actions secret `APPETIZE_API_TOKEN`. Do not commit it or echo it to logs.
- Optional repository secret `APPETIZE_PUBLIC_KEY` to update one existing preview rather than create a new one on every run. Confirm the intended privacy/sharing settings of the preview before sharing its link.
- No Apple code signing, certificates, developer-team credentials or App Store submission is needed for an unsigned iOS **Simulator** build.

### Deliverables

1. Add `.github/workflows/ios-preview.yml` with `workflow_dispatch` (manual triggering by default), `permissions: contents: read`, and a suitable pinned macOS runner image. Avoid per-push uploads or unnecessary compute spend at this stage.
2. Checkout the repo, print `xcodebuild -version` / `xcode-select -p`, install/generate the iOS project only **if** the existing Milestone 0 layout requires XcodeGen, and use the **actual project/workspace and scheme names** found in the repo. Do not recreate a working Xcode project or invent names.
3. Build a native iOS **Simulator** target with `xcodebuild` and `CODE_SIGNING_ALLOWED=NO` (generic simulator build destination is acceptable). Use `MockVehicleTransport` unconditionally for this configuration. The app should show an explicit `MOCK / SIMULATOR` state and never attempt to open an MFi accessory session.
4. Discover the actual `*.app` bundle under the Xcode build-products directory, verify that exactly the intended app exists, and make a `.zip` with the `.app` bundle at ZIP root (not its enclosing `Debug-iphonesimulator` folder). Fail clearly if packaging finds no app.
5. Upload that ZIP as a GitHub Actions **artifact** using `actions/upload-artifact`. This must succeed **even when Appetize is not yet configured**. Keep a sensible retention setting and no private keys, raw vehicle logs, or secrets in published artifacts.
6. When the API token exists, use the official `appetizeio/github-action-appetize` action with `platform: ios`, `appFile: <zip>`, `apiToken: ${{ secrets.APPETIZE_API_TOKEN }}`, and optionally `publicKey: ${{ secrets.APPETIZE_PUBLIC_KEY }}`. Verify the referenced action version and inputs against the upstream README at implementation time. Make this upload optional; missing credentials should not break an otherwise valid simulator build.
7. Surface the Appetize **public key / preview URL** (or a clear way to retrieve it) in the workflow summary without exposing credentials. Note that Appetize usage/streaming limits are account-plan dependent. If upload cannot be automated, describe how to download the GitHub ZIP and upload it manually.
8. Update `README.md` and `docs/DEVELOPMENT.md` (or the existing equivalent) with: WSL-to-GitHub push and Actions run instructions, where to find artifacts, how to configure secrets, how to open Appetize, how to diagnose build failures, and a concise simulator-versus-device comparison.
9. Keep all Linux/WSL Go and Python tests passing. Do not implement the Go recorder, telemetry storage, mock scenarios, Swift dashboard screens, OBD/CAN requests, background hardware behavior, App Store delivery or TestFlight signing in Milestone 0.5.

### Build/preview implementation details

- Inspect `ios/project.yml`, `.xcodeproj`, `.xcworkspace`, the app scheme, deployment target, existing Swift sources, and any generated Go bridge before deciding the exact CI commands. A project may use Swift Package Manager, XcodeGen or a checked-in Xcode project; adapt rather than replacing the bootstrapped layout.
- If the iOS shell already links a generated Go XCFramework, the cloud build must produce the **correct iOS Simulator variant** or use the Milestone 0 compile-only/mock bridge as designed. Do not silently claim Go recorder execution when only an iOS placeholder is running. Do not break the eventual Go binding/architecture to get an artificial green build.
- Compile errors (including missing frameworks or incorrect deployment targets) must cause CI failure and be fixed. Do not introduce blanket ignores or `|| true` for build/test commands.
- The app should launch in an iPhone-shaped Appetize simulator with the Milestone 0 shell and explicit disconnected mock status. Transport errors should not crash previews.
- Prefer a manual first run; after the pipeline is stable, adding PR build checks without Appetize upload can be considered separately.

### Acceptance criteria

**CI and build**
- A clean GitHub Actions macOS workflow run **passes** and generates the intended `.app` for iOS Simulator.
- A downloadable ZIP containing the simulator `.app` is attached to the run.
- The run makes no use of code-signing secrets and does not attempt a real MX+ connection.

**Browser smoke test**
- With the optional Appetize step configured (or after manual upload), the ZIP runs as an app in the browser, with the expected application name and initial SwiftUI screen.
- The UI clearly identifies the active data source as mock/disconnected.
- If Appetize is unavailable or its plan limits prevent testing, record that separately: **CI build may be complete, Appetize acceptance remains pending**. Do not claim an interactive test was performed unless it was observed.

**Project integrity**
- `go test ./...` and Python tests still pass wherever dependencies are available.
- Existing Milestone 0 files and application identifier `com.kaleevans.vehicleflightrecorder` are preserved.
- No Milestone 1 recorder code or real OBDLink commands are implemented.

**Handoff**: Codex must report all actual commands and CI results, the workflow URL/build artifact location (if available), preview URL or manual-upload steps, unmet prerequisites, and any remaining cloud/device testing limits. A workflow file merely being present does **not** satisfy the milestone; a successful remote run and observed browser launch are the target. If Codex cannot operate the user's GitHub/Appetize accounts, it must prepare everything possible and mark external verification **pending**, not "passed".

**Relevant official references**:
- GitHub macOS hosted runners: https://docs.github.com/en/actions/reference/runners/github-hosted-runners
- GitHub workflow artifacts: https://docs.github.com/en/actions/tutorials/store-and-share-data
- Official Appetize GitHub Action: https://github.com/appetizeio/github-action-appetize
- Appetize iOS upload format: https://docs.appetize.io/platform/app-management/uploading-apps/ios

## Milestone 1 — Go recorder + mock data

Deliverables:

- observation/event models.
- JSONL writer.
- session lifecycle.
- rolling buffer.
- mock healthy drive.

Acceptance:
- running mock scenario produces a valid session directory.
- Python CLI can summarize it.

## Milestone 2 — incident engine

Deliverables:

- manual marker.
- voltage event.
- module timeout event.
- post/pre capture.
- four primary mock fault scenarios.

Acceptance:
- each scenario yields expected incident count and causal ordering.

## Milestone 3 — iOS dashboard

Deliverables:

- dashboard.
- sessions.
- incidents.
- debug screen.
- mock selector.
- MARK INCIDENT.

Acceptance:
- full mock drive can be run in the **Appetize browser-hosted iOS Simulator** using a GitHub Actions-built app; later also validate on the physical iPhone when an installable device build is available.
- generated session can be exported.

## Milestone 4 — Go XCFramework integration hardening

Deliverables:

- reproducible `gomobile bind` script.
- versioned Go bridge.
- error translation.
- lifecycle tests.

Acceptance:
- clean Xcode checkout can build with generated core framework.

## Milestone 5 — MX+ transport shell

The OBDLink documentation and app approval are already available. Start this **after** the mock/bridge milestones and after preparing a separately signed physical-device test path; do not treat a GitHub/Appetize simulator preview as proof of real MX+ connectivity.

Deliverables:

- ExternalAccessory capability.
- protocol string configuration.
- accessory enumeration.
- EASession stream management.
- raw send/receive debug view.
- no vehicle polling yet.
- approved offline preparation while enrollment is pending: bounded response parsing,
  a simulated-clock scheduler, verified initial decoders and synthetic provenance/
  capability exports; keep this isolated from the physical transport (see milestone_5.md).

Acceptance:
- app reliably opens/closes the MX+ session and records raw adapter responses.

## Milestone 6 — adapter initialization + standard OBD

Deliverables:

- OBDLink initialization sequence based on official docs.
- automatic standard OBD discovery and explicit generic fallback (§10.1).
- first real signals.
- adapter voltage.
- session recorded from stationary vehicle.
- initial capability report, including response timing and observed signal quality (§10.2).
- defined live timestamp/freshness semantics and raw-transaction provenance (§6.1.1).
- bounded prioritized scheduler with tested timeout/backoff behavior (§10.3).

Acceptance:
- no dashboard warnings caused by the app.
- stable connection for a prolonged test.
- clean disconnect/reconnect.
- supported/unsupported/temporarily unavailable states remain distinguishable.
- exported report distinguishes requested rates from achieved observations.
- unavailable signals do not stall the recorder or appear as fresh zero values.
- generic operation does not require or silently apply an Alfa-specific pack.

## Milestone 7 — controlled road logging

Deliverables:

- limited validated polling schedule.
- hands-off session start/stop using validated engine/drivetrain activity, as defined in §15; scanner connection alone must not start a drive.
- tested background, locked-screen and reconnect lifecycle with documented wake/launch limitations.
- real drive export.
- automatic recording confirmation with permission handling and session-level deduplication (§15.1).
- capability report under the actual combined road-logging workload.
- physical-device reliability matrix and measured battery/storage impact (§20.4).
- comparison against MultiECUScan/known values.

Acceptance:
- repeatable real-world logs.
- after initial setup, no per-trip app-open or Start action in the supported, physically verified lifecycle states.
- ignition-on/engine-off does not create a false drive; stationary engine-running activity is recorded.
- module dropout, brief adapter loss and engine stop/start preserve evidence without premature session closure or duplicate sessions.
- unsupported wake/launch states and any required user action are documented explicitly.
- no unacceptable network disturbance.
- no missing/corrupt sessions in normal tested operation; interrupted sessions retain recoverable evidence with honest completeness labels.
- retrieved exports pass Python validation and preserve timing/source provenance.
- measured acquisition rates and coverage are sufficient for the selected diagnostic objectives; unresolved deficiencies are documented rather than hidden.
- tested battery/storage budgets and lifecycle limitations are recorded.
- auto-start notifications satisfy §15.1 without duplicate alerts or false recording-success claims; recording works when notifications are unavailable.

## Milestone 8 — Giulia-specific drivetrain expansion

Only after capturing stable standard data.

Deliverables:

- ECM1/ECM2/TCM definitions as verified.
- manufacturer-specific signals.
- module heartbeat/latency.
- DTC snapshots.
- initial real incident correlation.
- prioritized signal selection tied to the owner's diagnostic questions, with documented provenance and reference validation.
- capability report extended for verified pack signals under the combined workload.

Acceptance:
- manufacturer-specific definitions remain outside the generic core.
- generic fallback still works without the Alfa enhanced definitions.
- diagnostic findings distinguish observations, correlation and hypotheses.
- controlled capture/reconstruction tests are recorded separately from real-fault evidence; do not claim the vehicle issue is resolved without supporting findings.

## Post-MVP decision gate — Driver Companion V2

Use the validated capability reports and reliable diagnostic recordings to decide
which trip/performance features are supportable. The diagnostic objective and
Alfa package work above retain priority. The reviewed
[Driver Companion V2 plan](driver_companion_v2_plan.md) records the subsequent
product direction, staged releases, metric-quality contracts and cockpit design.
It does not expand current MVP implementation scope. Enterprise infrastructure
remains conditional on demonstrated needs.

---

# 23. Next Codex task — Milestone 0.5 only

**Milestone 0 is already completed.** The next Codex instruction should implement **Milestone 0.5 only**, using the existing repository at `~/dev/VehicleFlightRecorder` and reading both this plan and `vehicle_health_enterprise_architecture.md` before changes. The self-contained handoff is also available in `vehicle_flight_recorder_milestone_0_5_codex_prompt.md` when that file is copied into the repository.

Suggested prompt:

> Read the complete `alfa_flight_recorder_mvp_plan.md` and `vehicle_health_enterprise_architecture.md` in `~/dev/VehicleFlightRecorder`, then implement only **Milestone 0.5 — GitHub macOS build + Appetize browser preview**. Milestone 0 is complete: inspect and preserve the existing scaffold; do not recreate it. Add a manually triggered GitHub Actions macOS workflow that builds the current SwiftUI app for iOS Simulator without signing, creates a correctly structured zipped `.app`, always uploads it as a GitHub artifact, and optionally sends it to Appetize using the official GitHub Action when the API-token secret is configured. Document WSL/GitHub/Appetize setup and how to verify the browser preview. Simulator builds must use mock transport and never open a real MX+ session. Preserve the approved bundle ID. Do not implement Milestone 1 or any real OBD commands. Run available tests and report actual CI/preview evidence versus steps requiring my credentials or interaction.

After Milestone 0.5 has been verified, start **Milestone 1 — Go recorder + mock data** as a separate Codex task. Avoid asking Codex to generate the entire product in one pass.

---

# 24. Definition of success for the first real-car version

The initial Alfa test application succeeds when:

1. the MX+ can remain connected reliably for a complete drive.
2. logging continues with the phone screen locked/backgrounded where iOS permits.
3. all selected main-drivetrain signals are timestamped and persisted.
4. adapter/network health is recorded alongside engine data.
5. a dashboard-warning event can be marked in one tap.
6. at least 60 seconds of context before the mark is available.
7. the following 120 seconds are preserved.
8. DTCs are preserved rather than cleared.
9. a Python tool can reconstruct the exact ordered timeline.
10. the recorder does not itself produce diagnostic disturbances.

Only after these are demonstrated should weekly AI reporting become a priority.
