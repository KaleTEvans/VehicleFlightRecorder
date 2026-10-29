# Codex Handoff — Vehicle Flight Recorder, Milestone 5 Only

## Assignment and prerequisites

Implement **Milestone 5 — MX+ transport shell** in:

```text
/home/kaletevans/dev/VehicleFlightRecorder
```

This is a future implementation handoff, not authorization to begin this milestone
while writing the plan. Milestone 4 must be implemented and accepted first. Read
its actual report, contracts, generated artifacts and test results. Inspect the
checkout and `git status`; preserve legitimate changes. Do not rebuild preceding
milestones or implement Milestone 6 polling to compensate for missing prerequisites.

Physical acceptance requires a signed iOS-device build installed on the user's
real iPhone, the approved MX+ and a suitable parked test setup. Prepare that path
before claiming hardware success. An Appetize simulator ZIP cannot connect to the
physical scanner. Missing signing/account/device access does not prevent useful
code, transcript tests and unsigned native builds, but leaves physical acceptance
pending and the milestone PARTIALLY COMPLETE.

## Approved offline preparation extension — October 29, 2026

The user explicitly authorized bringing bounded M6 preparation into M5 while
Apple Developer enrollment approval is pending. This supersedes the prohibition
on beginning M6 work only for the following offline scope:

- Documented response framing/parsing and multiple-ECU/error transcript tests.
- Bounded prioritized scheduling, deadlines, backoff, cancellation and generation
  handling using an explicit simulated clock and adapter.
- A verified initial standard decoder/discovery subset and explicit capability states.
- Synthetic versioned timing/raw-provenance/capability exports with independent
  Go/Python validation and legacy mock compatibility.

Keep this logic isolated from the mobile transport. Do not expand its live command
allowlist, initialize/poll a real vehicle or claim physical acceptance. M5 still
requires the installed phone/MX+ checks below; full M6 live integration remains a
separate handoff. The implementation contract/results are in
`docs/OFFLINE_ACQUISITION.md` and `docs/MILESTONE_5.md`.

## Authoritative context

Read the full current MVP and enterprise plans, then:

```text
planning/milestone_4.md
planning/milestone_6.md           # boundary awareness only; do not implement it
planning/alfa_flight_recorder_mvp_plan.md
planning/vehicle_health_enterprise_architecture.md
docs/MILESTONE_3.md
docs/MILESTONE_4.md              # expected after prerequisite completion
docs/DATA_SCHEMA.md
docs/DEVELOPMENT.md
docs/IOS_PREVIEW.md
```

Priority: active human instructions, this handoff, MVP plan, enterprise roadmap.
Focus on MVP §§1.2, 2, 4–5, 6.1.1, 15–17, 20–22. Inspect the actual Swift
`VehicleTransport`, mock/MX+ implementations, bridge, project and workflows.
At planning time the MX+ implementation is only an offline stub; do not infer
hardware capability from its filename or plist configuration.

Preserve Vehicle Flight Recorder 0.1.0, bundle ID
`com.kaleevans.vehicleflightrecorder`, protocol `com.obdlink`, iOS 17 floor and
MFi PPID `221699-0001` as submission metadata. Increment build numbers if the
chosen development distribution requires it; do not casually rename/reversion
the approved marketing identity. Preserve the Milestone 4 toolchain contract.

The documented MX+ path is Bluetooth Classic/MFi via Apple External Accessory,
not BLE/CoreBluetooth. Recheck applicable official Apple/OBD Solutions documents
at implementation time and record their revisions. Approval is already reported
complete; investigate actual integration failures instead of recreating onboarding.

## Required work

### 1. Physical-device delivery prerequisite

Use the existing macOS runner/device-framework path to build the correct iOS
ARM64 application. Establish one documented signed development/ad-hoc/TestFlight
path appropriate to the user's available account and provisioning. Keep it
separate from unsigned Appetize packaging and label artifacts accurately.

Use repository/environment secrets for any required credentials; never request
that the user paste private keys, signing passwords or tokens into chat. Record
non-secret setup steps and any account action only the user can perform. Do not
claim an unsigned compile or uploaded IPA proves installation. Record the actual
installed build, iOS version and device test provenance. No public App Store
release or general distribution platform is required.

### 2. Discovery and connection lifecycle

Implement `MXPlusExternalAccessoryTransport` behind the shared transport seam.
Enumerate already connected accessories at startup and observe subsequent
connection/disconnection notifications with balanced observer lifetimes. Select
an accessory advertising `com.obdlink`; name/model/manufacturer metadata provide
additional evidence, not a substitute for the protocol contract. Handle zero,
one, multiple, unsupported and disappearing accessories explicitly.

Create/retain an EASession, schedule its input/output streams on one documented
native executor/run loop, and open/close deterministically. State must distinguish
disconnected, discovering/connecting, streams-ready and failed/interrupted as
needed; a Bluetooth association alone is not streams-ready or recording.

Make connect/disconnect idempotent with bounded connect deadlines and conservative
reconnect/backoff. Unregister observers, unschedule/close streams and finish the
appropriate consumers exactly once. Prevent duplicate sessions/observers and
late callbacks or buffered output from an old connection affecting a new one.
Assign connection generations/identities and clear stale outbound requests.
Do not spin forever, create reconnect storms or hide repeated failures.

### 3. Ordered arbitrary byte transport

Forward inbound chunks in order through the established Go boundary. A stream
callback is not a complete response: permit split prompts, split lines, combined
responses and arbitrary chunk boundaries. Transport code does not parse OBD,
choose PIDs, interpret ECU health or apply Alfa rules.

Handle partial writes, zero-byte progress, temporary lack of stream space,
end/error events and disconnect during a write. Report bytes actually accepted
separately from requested writes and uncertain outcomes. Serialize writes; never
replay stale or partly sent commands automatically after reconnect.

Use bounded queues and explicit overflow/backpressure policy for incoming and
outgoing bytes. If evidence cannot be preserved, emit a loss/error condition and
stop or degrade honestly rather than silently dropping chunks or blocking the UI.
An AsyncThrowingStream default buffer is not a bounded-queue design. Reconnection
must give consumers a valid new stream or a documented persistent-stream contract,
not reuse a finished stream without explanation.

### 4. Raw evidence and narrow transport debug view

Expose actual accessory metadata, transport state, connection generation, receive/
transmit counts, bounded queue/overflow status, stream errors and raw received/
transmitted chunks. Show only measured counters. Keep raw evidence before higher
level decoding, preserving direction, ordering, byte-safe encoding, connection
identity, receipt/write timing and actual versus attempted transmission.

This transport can be active before a drive exists. Define a versioned
connection-capture format or a compatible recorder integration without labeling
transport-only captures as driving sessions. Reuse Milestone 4 timing contracts;
M6 adds request/response correlation and decoded measurements. If canonical raw
files or schemas change now, update Go, Python, Swift export and validation
together. Existing Python currently rejects nonempty raw evidence; do not simply
turn off that rejection without a defined validated raw format. Preserve old mock
exports, including legitimately empty raw files.

The debug view is for transport evidence, not an arbitrary terminal. Display raw
send/receive history with bounded UI memory while preserving authoritative disk
logs. Redact personal vehicle identifiers from public CI artifacts; retain actual
local evidence with appropriate provenance. No cloud upload backend is required.

### 5. Strict command boundary for this milestone

Opening a connection must not automatically initialize the adapter or start vehicle
polling. All command generation remains Go-owned; Swift sends only permitted bytes
supplied by that boundary. Keep unrestricted raw command entry disabled.

Test outbound plumbing with fakes/transcripts first. For a physical round-trip
check, a single fixed, documented **adapter-identification read** may be exposed
as a separately enabled developer action after its exact command/response and
non-vehicle scope are verified from official documentation and covered by tests.
Do not guess command strings. This exception is not a reset/init sequence,
continuous polling, an ECU request or a new arbitrary-send facility. Keep its raw
request/response as evidence, including errors and empty replies.

If no such verified read is available, send nothing and record any unsolicited
bytes honestly; no spontaneous response is guaranteed. Leave physical round-trip
verification pending rather than manufacturing responses or importing M6 work.
An identity-only exchange does not prove diagnostic protocol or vehicle access.

### 6. Hands-off lifecycle groundwork

The product should eventually record without a per-trip app-open/Start action.
For M5, establish and measure the accessory lifecycle only. Verify discovery and
connection behavior when already connected, foregrounded, backgrounded, locked,
suspended, OS-terminated and user-force-quit; list each state separately and record
unsupported/untested cases. Capture actual timing/errors and any required user
recovery action. Do not infer that declared background modes guarantee wake or
continuous execution in every state.

Scanner connection is not a drive-start signal. Ignition-on/engine-off can make
it available. Engine/transmission reporting and engine-running evidence need
real-data validation in M6; automatic drive policy and the full reliability matrix
belong to M7. A future missing TCM response must remain diagnostic evidence, not
prevent all recording. Do not add automatic real-drive creation in this milestone.

### 7. Preserve simulator and architectural boundaries

Simulator/Appetize remains explicitly mock-only and must never discover an
accessory, open EASession or transmit to a vehicle. Use a clear compile/runtime
selection policy; real-device mode must not silently fall back to mock readings
when its connection fails. Retain the accepted dashboard, evidence navigation,
marking and exports. SwiftUI must not import accessory-specific implementation
logic; Go/Python remain free of Apple framework dependencies.

## Required verification

| Layer | Minimum cases and evidence |
| --- | --- |
| Mock stream tests | Fragmented/coalesced chunks, partial/zero writes, stream end/error, bounded overflow, cancellation and pending-write disconnect |
| Lifecycle tests | Existing accessory enumeration, later notification, unsupported protocol, failed session creation, repeated connect/disconnect, reconnect generations, observer cleanup and stale callbacks |
| Command boundary | Ordinary connection sends no commands; unsupported/arbitrary writes fail closed; optional single identity read has exact documented transcript tests |
| Local/native regression | Go tests/vet/race where relevant, Python compatibility/raw validation, script tests/lint, native bridge/unit/UI tests and unchanged Appetize mock flows |
| Device build/install | Signed ARM64 artifact, successful real-iPhone installation, exact build/device/iOS/adapter configuration |
| Physical transport | Repeated stream open/close, actual ordered raw inbound/outbound round trip, clean disconnect/reconnect, no stale command replay; vehicle parked |
| Lifecycle evidence | Separate foreground/background/locked/wake/termination cases and limitations; no simulator substitution |

Use realistic bounded stream fixtures and a narrow adapter seam so most failures
are testable without hardware. Do not intentionally disrupt ECUs or induce a
vehicle fault. If a physical test produces unexpected vehicle warnings or other
abnormal behavior, stop transmissions and preserve evidence for review.

## Acceptance, boundaries and completion report

Milestone 5 is complete only after the real installed app reliably opens/closes
the approved MX+ session, preserves actual raw response evidence and demonstrates
a controlled round trip without vehicle polling. Fake transports and successful
signing are separate evidence, not substitutes. Missing physical/lifecycle checks
remain explicitly pending; report what is supported rather than promising every
iOS launch state. Do all independent implementation/testing before requesting
missing user-only credentials or device observations, and do not re-request an
already granted authorization.

Write `docs/MILESTONE_5.md` with transport ownership/state diagram, queue/deadline/
reconnect policy, raw format/version, signing/install instructions, primary command
references, actual test matrix/results and remaining M6/M7 work. Update README,
development, bridge and preview instructions where needed.

No standard OBD discovery/poll loop, proprietary diagnostics, vehicle-specific
threshold activation, automatic drive policy, GPS/motion/performance UI, cloud or
AI. Never clear DTCs, code/configure/calibrate ECUs, actuate components or provide
arbitrary CAN writes. The app is hands-off in user experience; permitted future
read-only diagnostic requests must still be controlled, logged and rate-limited.

Report COMPLETE / PARTIALLY COMPLETE / BLOCKED; exact files/contracts and changes;
local/native/signed/physical results separately; artifact/build/preview links;
actual raw evidence and provenance; wake/background limitations and pending user
steps; primary documentation and deviations. End explicitly:
**Milestone 6 not started.**
