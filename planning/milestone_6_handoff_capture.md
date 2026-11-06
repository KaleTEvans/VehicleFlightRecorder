# Milestone 6 — adapter handoff capture

Status: DRAFT pending native acceptance and TestFlight delivery. Build: pending.
Do not repeat D2 on build 10.1 for this purpose; it resets/configures the adapter
before reading protocol settings. D1 is already validated and does not need a repeat.
This short packet replaces that extra diagnostic recording.

## What this captures

The new **Adapter handoff capture** workload sends only `STPR`, `STPRS`, and
`STPBRR`, in that order. No protocol close/reset/selection, vehicle request, voltage
poll or automatic retry is sent. It finishes and saves automatically after three
prompt-terminated readbacks, or ends with the evidence available if it cannot
continue. The active owner-loop limit is 15 seconds. Unknown/AUTO/error replies
stay raw and unverified; zero telemetry samples are intentional.

This reads shared adapter state after app reconnection, not OBDLink's private
traffic or historical settings. OBDLink cleanup, an accessory reconnection or power
loss may already have changed the state. A captured setting is not automatically
a verified vehicle protocol or permission for VFR to use an unsupported profile.

## One short parked visit

1. **Prepare indoors:** install the delivered build listed above. Keep this packet
   available. Existing D1 and 9.1 history/export tests are closed; no repeats.
2. Keep the car parked, engine idling under suitable ventilated conditions. In
   the official **OBDLink** app, establish the working SAE RPM/coolant readings.
   A screenshot of those labels/units is useful if one is not already saved.
3. **Leave the scanner plugged in and powered.** Do not cycle ignition, toggle
   Bluetooth or unplug the adapter between apps. Close OBDLink completely so it
   releases the accessory. Avoid its manual Disconnect/reset controls if possible;
   closing may still clear settings, which is an acknowledged limitation.
4. Open VFR. Enable **Vehicle is parked**, select **Engine running / stationary**,
   and choose **Adapter handoff capture**. Do not start Combined or Discovery
   diagnostic first: those workloads reset/configure the adapter.
5. Tap **Capture adapter settings** once. Keep VFR foreground with screen awake.
   It normally finishes in a few seconds; allow up to 15 seconds of active owner
   time. Note the readback count (normally **3/3**) and protocol/description/baud
   replies. **AUTO, ? or missing values are useful outcomes, not a reason to retry.**
6. When Recorder says **complete**, finish the connected work. Do not add a new
   two-minute recording, protocol search, factory reset or C1/C2 continuation.

If the scanner never connects, or capture cannot finish, note the exact state and
preserve what exists. Do not repeatedly press Capture or return for another case.
A new vehicle warning or abnormal behavior is a reason to stop the test.

## Finish indoors — one handoff

Open **Saved sessions, incidents and export → Adapter handoff capture → Export
session ZIP**. Send that ZIP, plus any reference screenshot already captured and
whether the scanner stayed powered throughout the handoff. Zero sensor samples
are expected; the six-file ZIP contains the adapter readbacks and their provenance.
For an incomplete capture that cannot export, retain its folder through Files →
On My iPhone → Vehicle Flight Recorder → Sessions. No protocol field needs to be
located in OBDLink's UI and no raw commands need to be typed by the user.
