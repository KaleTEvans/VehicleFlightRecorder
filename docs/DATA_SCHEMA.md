# Data schema direction

Milestone 0 has no telemetry types or storage implementation. This summarizes
MVP sections 6–7 and enterprise evidence/provenance requirements for later work.

Observations use `schema_version`, `timestamp_unix_ms`, `session_id`, `signal_id`,
`source`, `value`, `unit`, and `quality`. Optional module/bus, raw request/response,
decoder version, and pack version preserve provenance. Canonical IDs remain
manufacturer-independent. The MVP naming catalog governs near-term work;
enterprise ontology examples are directional, not a silent renaming migration.

Events record discrete occurrences with timestamp, type, severity, and attributes.
Incidents index related evidence and configurable pre/post windows. Never infer
causation solely from ordering or discard raw evidence after decoding.

The planned session directory contains `session.json`, `observations.jsonl`,
`events.jsonl`, `incidents.jsonl`, and `raw/transport.jsonl`, including mock raw
evidence when recording is implemented. Start with append-only JSONL, not SQLite.
Full-session logs are authoritative; incident windows are views/indexes over them.
Schema compatibility, recovery, and later storage migrations require future tests.
