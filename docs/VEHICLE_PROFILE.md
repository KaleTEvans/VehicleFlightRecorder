# Vehicle profiles — Milestone 1

The initial pack is `vehicle-packs/alfa-giulia-qv-2017`, profile ID
`alfa.giulia.qv.2017.2_9t`, describing the 2017 Giulia Quadrifoglio, 2.9L
twin-turbo V6, ZF 8-speed, with primary-drivetrain scope. That is applicability
metadata, not a claim of diagnostic access.

Manifest and catalog keep `schema_version: 1`, `pack_version: "0.1.0"`, and matching
profile IDs. The manifest status is now **mock_only**, with
`validated_signals: false`. Six signal definitions are explicitly `mock_only`:

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
includes explicit synthetic provenance. `polling.json` and `thresholds.json` remain
empty; no real diagnostic polling command, proprietary ID/address, decoder or
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
