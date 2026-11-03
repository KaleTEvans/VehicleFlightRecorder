# Offline acquisition preparation

This is the user-authorized extension to Milestone 5, prepared October 29, 2026
while Apple Developer enrollment approval is pending. It implements bounded Go
protocol/scheduler/decoder logic and independently validated synthetic exports.
M5 physical acceptance and M6 live integration remain pending.

## Ownership and command boundary

`core/obd` owns a fixed read catalog, framing, the narrow response parser,
decoders and per-ECU capability states. `core/acquisition` owns single-threaded,
explicit-clock scheduling, transaction history and report statistics.
`core/cmd/offline-acquisition` connects these to an in-process scripted adapter.
There is no Bluetooth, serial, socket, native stream or wall-clock acquisition
loop in these packages. They are not imported by `core/mobile`; generated native
bindings and Swift remain unchanged. The existing physical capture still permits
only its explicit once-per-connection STI identity read. No vehicle polling,
adapter setup/reset or permission prompt was added to the app.

The scheduler and capability book each have one owner; these are not concurrent
APIs. Native integration in M6 must preserve that ownership, clock mapping and
raw-evidence contract instead of calling them from UI refreshes or the accelerated
mock tick. New request entries cannot be injected through CLI input or pack data.

## Verified reference surface

| Offline key | Request | Decoding / meaning | Reference |
| --- | --- | --- | --- |
| support | `0100` + CR | Block 00 support bits; only implemented PIDs 05/0C are selected | ELM327DSJ, pp. 31–32 |
| coolant | `0105` + CR | Byte A − 40, degC | ELM327DSJ, p. 31 |
| rpm | `010C` + CR | Unsigned big-endian A/B divided by four, rpm | ELM327DSJ, pp. 31–32 |
| adapter_voltage | `STVR` + CR | Default two decimal places, volts at adapter ANALOG_IN; dashed value unavailable | OBDLink FRPM E, §8.5, p. 32 |

Primary sources checked October 29, 2026:
[ELM327DSJ](https://www.elmelectronics.com/wp-content/uploads/2016/07/ELM327DS.pdf),
[OBDLink FRPM E](https://www.scantool.net/scantool/downloads/678/obdlink_frpm_e.pdf).
Header/PCI examples and 11-bit responder identity are described in ELM327DSJ
pp. 41–44, 47. These references establish example encoding/semantics, not actual
MX+ firmware behavior, vehicle availability or a full licensed SAE catalog.
No copied manual or proprietary definition is committed.

The initial catalog deliberately contains no speed/load/throttle or enhanced Alfa
definitions until suitable applicable source definitions are verified. Only block
00 is needed for the two implemented standard PIDs; continuation/discovery of a
larger catalog remains M6 work. `STVR` is a separate adapter measurement, not an
ECU supply or battery-terminal voltage. No voltage calibration command exists.

## Framing and parsing

Profile `can11-single-frame-headers-pci` requires ASCII text, CR/LF-separated
lines, a final `>` prompt, visible 7E8–7EF response IDs and the single-frame PCI
length byte. Compact/spaced hex, optional exact command echo, multiple responders,
known adapter status text, valid zero values, negative mode-01 replies and voltage
unavailability are covered. ECU IDs remain addresses; they are never labeled TCM
or dual ECM roles by assumption.

Each callback is arbitrary bytes. Harmless whitespace around prompts is tolerated
across chunk boundaries; authoritative raw records still retain those bytes. Framer limits: 64 KiB per input, 8 KiB per
incomplete response, 16 completed frames/call. Parser limits: 32 lines, 256 bytes
per line, at most eight CAN data bytes with a valid single-frame payload length.
Headerless, DLC-decorated, 29-bit, multiframe and other legacy formats are rejected
explicitly. M6 must verify actual adapter formatting before admitting physical
replies; no initialization commands are inferred or sent here.

Malformed data from one responder is retained as a per-responder failure while
other valid responders survive. Ambiguous duplicate replies from one ECU are
excluded. Unknown/malformed bytes never become fresh zero measurements. Framing
overflow/non-ASCII poisons the framer until an explicit reset and blocks the
scheduler generation; raw evidence is preserved before decoding.

## Scheduling, timing and failure behavior

A schedule has at most 16 fixed-catalog entries, one in-flight transaction and one
due time per entry. Oldest due time wins; priority (0 first) breaks ties, then key.
This bounds backlog and prevents a perpetually due failing signal starving all
others. Completion schedules the next due time without accumulating missed ticks.
Failures use interval-based exponential backoff, capped at 30 seconds (or the
already-longer configured interval, so failure never speeds up polling); successful
or partially successful transactions reset that entry's backoff. A global 100 ms
admission gap and test configuration bounds are **offline harness choices**, not
validated or recommended vehicle polling rates.

Only support/adapter reads are eligible before discovery. Standard reads require
advertised or observed support from at least one responding ECU. Each responder
retains its own advertisement/observation/availability state and evidence
transaction. Timeout/NO DATA cannot establish unsupported. Cache state resets on
explicit generation change; prior transaction evidence remains in the report.
Configuration/vehicle-aware persistent caching is still a live integration task.

Callers advance nondecreasing monotonic nanoseconds explicitly, over at most 24
logical hours. There are no sleeps. One positive Unix-ms anchor maps receipt time
as `anchor + floor(monotonic_ns / 1,000,000)`. Source measurement time remains null.
Latency is response-completion receipt minus last-byte stream acceptance, not ECU
internal processing or sensor-refresh time. Real cross-language/native clock
admission, wall-clock-change events and lifecycle behavior remain M6 verification.

Actual accepted prefixes are recorded separately from requested bytes. Zero write
progress does not succeed or spin; partial progress is bounded by a deadline.
Timeout before any acceptance can release the slot. A partly accepted incomplete
command blocks the generation until reconnect. A fully accepted timed-out command
enters draining: no new request is sent until its prompt is consumed. Late bytes
are logged as late_received, never used as observations. Failure to reach the
prompt within one additional logical second blocks the connection. Reconnect is
explicit, cancels old work and clears framing/capabilities; old generations cannot
feed or complete current work. No automatic stale replay occurs.

Malformed complete replies can finish as protocol_error and let unrelated requests
continue after the admission gap. Unexpected multiple response boundaries,
unsolicited data, unrecoverable framing or evidence-budget failure block further
admission. M6 must bind that state to conservative actual reconnect/backoff;
calling a timer alone cannot disambiguate a late same-PID reply.

## Version-1 synthetic report contract

`report_version: 1`, `kind: offline-acquisition`, `source: synthetic`, profile and
catalog `standard-offline-1` identify this standalone JSON artifact. It is not a
canonical drive export, physical capability report or a replacement for the M5
connection capture. Existing schema-1 mock sessions, empty raw files, ZIP layouts
and their strict Python reader remain unchanged.

The report contains:

- Clock anchor, logical duration, final generation/state, unknown vehicle identity
  and an explicitly unmeasured reference comparison.
- Full concurrent schedule with requested intervals, deadlines and priorities.
- Contiguous raw records: sequence, generation, transaction identity, direction/
  kind, monotonic time and exact Base64 bytes. Unsolicited bytes have no invented
  transaction ID. Timed-out late receives remain attached to their original ID.
- Transactions: admission/end, optional full-acceptance time, accepted-byte count,
  exact raw references, status strings and outcome. Partial success preserves good
  observations and failed responders separately.
- Observations: transaction/generation, ECU or adapter identity, canonical signal,
  value/unit, receipt/mapped timestamps, null measurement time and decoder version.
- Final-generation capability states/evidence and per-signal/ECU statistics over
  all observations in the logical run, including reconnect gaps.
- Explicit issues for coverage/resource/synchronization failures.

Budgets: 512 transactions, 4,096 raw records, 512 KiB total raw bytes, 2,048
observations and 8 MiB serialized output. Reads are bounded independently in
Python. Exhaustion stops admission visibly; no bounded history silently drops
old authoritative bytes. Reports are created exclusively with private permissions
and cannot overwrite a previous file. Failed writes remain failed/truncated
artifacts; there is no repair/finalization inference.

Rate = decoded observation count / total logical elapsed seconds, including gaps.
Maximum gap includes leading silence, consecutive inter-arrival gaps and trailing
silence. Latency min/p95 use successful per-observation populations (nearest-rank
95th percentile). Multiple ECUs produce separate observations/statistics, not an
inflated successful-request count. Requested versus achieved rates are explicit.
Steady repeated values are valid and cannot prove internal ECU refresh rate.

Python independently checks schema/version/types, fixed commands, accepted
prefixes, raw sequences and transaction/generation/time references, decoded values
against original narrow-format response bytes, advertised support bits and
observation-derived statistics. Duplicate/foreign observations, unknown fields,
non-finite values, fabricated measurement times and unsupported versions fail.

## Reproduce and inspect

From the project root, using the configured Go and Python environments:

```bash
mkdir -p build/offline-preparation
(cd core && go run ./cmd/offline-acquisition \
  --scenario healthy --output ../build/offline-preparation/healthy.json)
(cd core && go run ./cmd/offline-acquisition \
  --scenario edge_cases --output ../build/offline-preparation/edge-cases.json)
analytics/.venv/bin/python -m vehicle_flight_recorder validate-acquisition \
  build/offline-preparation/healthy.json
analytics/.venv/bin/python -m vehicle_flight_recorder validate-acquisition \
  build/offline-preparation/edge-cases.json
(cd core && go run ./cmd/offline-acquisition \
  --scenario partial_replies --output ../build/offline-preparation/partial-replies.json)
analytics/.venv/bin/python -m vehicle_flight_recorder validate-acquisition \
  build/offline-preparation/partial-replies.json
```

Use new output filenames on subsequent runs. All three scenarios cover 12 logical
seconds without wall-time waiting and are byte-reproducible. Healthy: 37 admitted
transactions, 51 observations, one end-of-run cancellation. Edge cases: 36
transactions, 46 observations, one timeout/late reply, partial-write cancellation
and explicit reconnect to generation 2, plus end-of-run cancellation. These are
fixture counts, never measurements of a car or phone. The `partial_replies`
scenario preserves healthy replies alongside a malformed second ECU: 37
transactions, 34 observations and 17 explicitly partial outcomes, independently
validated in Python.

Cross-language tests build and run the actual Go CLI twice, compare exact outputs,
then mutate the evidence to check rejection. Go covers fragmentation/coalescing,
resource limits, zeroes/unavailable values, multi-ECU failures, clock regressions,
backoff/draining/fairness, generation isolation and snapshot ownership. A bounded
fuzz target exercises arbitrary framing/parser input. See MILESTONE_5.md for the
recorded test results and pending physical acceptance.

## Remaining M6 work

Actual adapter initialization/format negotiation, expanded verified standard
catalog/discovery, production native byte admission, live-drive schema/ZIP migration,
capability identity/configuration caching, native lifecycle integration, physical
stationary acquisition/reference comparisons and retrieved real exports remain
M6 work. Its implementation should reuse these tested packages and replace the
scripted adapter through a deliberately reviewed seam. No M7 automatic drive policy,
recording notification or M8 enhanced command was introduced.
