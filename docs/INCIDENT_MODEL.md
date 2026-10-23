# Incident direction

No incident engine exists in Milestone 0. The MVP calls for configurable windows
of 60 seconds before and 120 seconds after a trigger, with a rolling pre-buffer.
Full-session storage stays authoritative; overlapping incidents link evidence
instead of creating independent copies of truth.

Later triggers include manual marks, voltage excursions, repeated module timeouts,
multi-module communication loss, new DTCs, and transport disconnects. Test voltage
thresholds must be labeled synthetic, not validated diagnostic limits. Adapter
loss must be distinguishable from ECU unavailability.

Incidents will include identity/session, trigger time/type, severity, window sizes,
capture status, and manual flag. Reconstruction preserves temporal order and
competing explanations; preceding an event does not establish its cause.
Implementation and fault scenarios belong to Milestone 2 after the recorder.
