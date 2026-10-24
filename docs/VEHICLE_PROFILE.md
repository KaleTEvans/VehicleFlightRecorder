# Vehicle profiles — Milestone 2

The initial pack is `vehicle-packs/alfa-giulia-qv-2017`, profile ID
`alfa.giulia.qv.2017.2_9t`, describing the 2017 Giulia Quadrifoglio, 2.9L
twin-turbo V6, ZF 8-speed, with primary-drivetrain scope. That is applicability
metadata, not a claim of diagnostic access.

Manifest and catalog keep `schema_version: 1`, `pack_version: "0.2.0"`, and matching
profile IDs. The manifest status is now **mock_only**, with
`validated_signals: false`. The original six healthy signals remain explicitly `mock_only`:

| Canonical signal | Unit | Interval | Synthetic range |
| --- | --- | --- | --- |
| `powertrain.engine.rpm` | rpm | 100 ms | 0–8000 |
| `powertrain.vehicle.speed` | km/h | 100 ms | 0–300 |
| `powertrain.throttle.position` | % | 100 ms | 0–100 |
| `electrical.adapter_voltage` | V | 100 ms | 10–16 |
| `powertrain.engine.calculated_load` | % | 500 ms | 0–100 |
| `powertrain.coolant.temperature` | degC | 1000 ms | −40–130 |

These ranges bound synthetic generation only. They are not manufacturer
specifications, validated plausibility limits, or incident thresholds. Each signal
includes explicit synthetic provenance. `polling.json` remains empty; `thresholds.json` now contains explicitly
synthetic incident rules. No real diagnostic polling command, proprietary ID/address, decoder or
production voltage rule has been introduced.

The generic Go loader validates supported schemas, required manifest metadata,
catalog identity/version, mock-only/unvalidated status, unique IDs, positive
100-ms-aligned intervals, finite ordered two-element `mock_range`, units and
provenance. It rejects unknown JSON fields and trailing documents. This first
loader deliberately supports the mock pack format only; future real pack formats
must be explicitly introduced/tested rather than treated as already supported.

The generic generator maps canonical signal IDs to healthy synthetic waveforms,
checks expected units, samples at pack-defined intervals, and rejects unknown
waveforms or values outside declared mock ranges. No Alfa-specific name or
diagnostic condition is embedded in generic Go recorder/transport logic. The CLI
requires a pack path instead of hard-coding this vehicle as the generic default.

Future signal definitions must retain availability/provenance/licensing and
validation status. Adapter capability does not establish module reachability.
Vendor approval and MFi metadata remain native integration/submission concerns,
not generic domain constants. Additional vehicle coverage awaits real evidence.

Milestone 2 adds scenario-scoped `network.tcm_response_latency_ms` (ms, 100 ms,
0–5000) for B/C and `engine.boost.requested` / `engine.boost.actual` (kPa, 100 ms,
0–300) for D. These are synthetic generic scenario names, not proprietary decoded
signals or vehicle pressure specifications. Definitions list supported `scenarios`;
an absent list means enabled in every scenario. Healthy generation still emits six
signals. Scenario files declare ordered events, value overrides, transport gaps,
expected incident count and trigger order; offsets are relative to the drive's
logical midpoint. The loader verifies identity, aligned offsets, catalog ranges,
nonoverlapping overrides and matching disconnect/reconnect gap boundaries.

Synthetic rules configure critical modules ECM_1, ECM_2, TCM, three expected
response failures, voltage below 12 V / rapid drop 1.5 V in 1 s, reset at 13 V,
correlation within 1 s, pre/post 60/120 s and record-silence tolerance 2 s. These
labels and limits are test inputs only. Configuration/scenarios contain no real
addresses, PIDs, initialization sequences, adapter commands or validated limits.
