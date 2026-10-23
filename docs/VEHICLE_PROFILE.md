# Vehicle profiles

The initial pack is `vehicle-packs/alfa-giulia-qv-2017`, profile ID
`alfa.giulia.qv.2017.2_9t`. It identifies the 2017 Giulia Quadrifoglio, 2.9L
twin-turbo V6, ZF 8-speed, scoped to the primary drivetrain network.

The four JSON files are bootstrap placeholders, not a finalized validation schema:
each has `schema_version: 1`, `pack_version: "0.1.0"`, and `profile_id`.
The manifest has vehicle metadata and explicit unvalidated status. `signals`,
`polling`, and `thresholds` arrays are empty. Empty arrays mean no definitions,
not supported vehicle functionality. No proprietary IDs, addresses, commands,
decoders, or real diagnostic thresholds have been guessed.

Future generic Go loaders consume versioned packs. Signal definitions will carry
availability, provenance/licensing, validation status, units, and sampling intent.
The MVP distinguishes standard OBD, adapter, manufacturer-specific, derived, and
mock-only signals. Adapter capability does not prove ECU reachability. Vendor
approval and MFi metadata belong to native integration documentation, not vehicle
diagnostic logic. Extended pack formats and support tiers await evidence.
