# Signed physical-iPhone builds

The separate `Signed iPhone build` workflow prepares an App Store distribution
IPA and can explicitly upload it for TestFlight processing. It uses the accepted
Go/runtime pins and iOS ARM64 slice, with a separate Xcode 26.0.1 device
profile in scripts/device_toolchain.env. The accepted preview stays Xcode 16.4.
Simulator ZIPs remain separate. The first signed archive/export passed on October
30, 2026. Phone installation and the first real identity exchange are now verified;
the milestone report separates the remaining physical edge cases. Build 5.1 contains the
verified button-flicker fix, now accepted visually by the user. A subsequent
physical unplug/replug test exposed a recovery issue. The corrected build 6.1
passed native verification and uploaded at 17:20:58 CDT on October 30. On November
3, the user confirmed automatic recovery takes approximately 1–2 seconds after iOS
Bluetooth reconnects; M5 transport acceptance is complete. Unverified physical
edge cases remain listed in the M5 report.

## Milestone 6 beta

Build `0.1.0 (7.1)` uploaded on November 3, 2026 at 1:44:51 p.m. CST from
`06c5e207ae82aae259a9720271fc5f08df359511` in
[run 41705740118](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/41705740118).
The exact source passed 26 hosted native tests, two UI tests, both device compile
profiles and real native ZIP reconstruction. Apple confirmed VALID /
IN_BETA_TESTING at 1:45 p.m. CST; see [the M6 report](MILESTONE_6.md). Update from transport-only build 6.1 before testing
parked live controls. Its [original test-visit checklist](../planning/milestone_6_test_session.md)
is paused; use the current targeted packet below after its build is delivered.
Connected work stays in one parked visit, with history/export and one evidence
handoff afterward indoors. Per-case chat review is not required. Physical M6 acceptance and road/background
qualification remain pending.

**Current physical diagnosis:** on November 6, both requested RPM/coolant phone ZIPs
were independently validated. They retain build-7.1 setup success followed by
`NO DATA` from `0100`, leaving measurement requests gated. Why discovery received
no data remains unresolved; physical sampling is not accepted. The build-10.1
D1 diagnostic also returned no ECU data; the official app later showed RPM and
coolant, but its working protocol is unknown. The 11.1 handoff captured CAN 29/500; the next connected work uses the
[new CAN29 packet](../planning/milestone_6_can29_session.md) on the delivered 12.1 build.

Build **0.1.0 (9.1)** is available for internal TestFlight installation. It adds
workload/state to session history, requested readings and a zero-reading message
to Drive details, using metadata already stored in existing recordings. No scanner
connection or re-recording is needed. Source
`84dcc28fbfa53795795b2f3a54c88a6cb07fe0bf` passed 28 hosted tests, three UI tests,
both device compile profiles and five actual native ZIP validations in
[native run 42266159403](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42266159403).
[Signed run 42267707114](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42267707114)
uploaded that exact source on November 6 at 12:10:25 p.m. CST; Apple confirmed
VALID / IN_BETA_TESTING. This preserves the build-8.1 export correction. OBD
initialization, admission and decoders are unchanged. See [the report](MILESTONE_6.md).

The earlier diagnostic beta **0.1.0 (10.1)** used source
`24037f0f0bdab0c5e0caff3a98cb3d0185b80629`. It passed 29 hosted / four UI tests,
seven actual native ZIP validations, both device compile profiles and signed
[run 42276904829](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42276904829).
Apple confirmed upload COMPLETE, build VALID and IN_BETA_TESTING after upload
November 6 at 1:11:05 p.m. CST. Its Discovery diagnostic workload was used for
the [earlier one-visit packet](../planning/milestone_6_diagnostic_session.md);
regular workloads keep their prior initialization/schedules. D1 evidence is
received; successful physical ECU acquisition is still pending.

The previous targeted beta **0.1.0 (11.1)** used source
`540c385bb465ac6404ed67ab4f393923fe2274c7`. It passed 30 hosted native tests,
five UI tests, ten native ZIP validations and both device compile profiles in
[native run 42288515502](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42288515502).
[Signed run 42290107051](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42290107051)
uploaded it November 6 at **2:42:16 p.m. CST**; Apple confirmed VALID /
IN_BETA_TESTING. **Adapter handoff capture** reads only STPR, STPRS and STPBRR
without reset/configuration or vehicle polling, then saves automatically. It may
capture settings left by the reference app, although cleanup/reconnection may
already have altered them. The received physical handoff passed all three
readbacks and retained CAN 29/500; no repeat of that packet or D2 is requested.
Existing regular/diagnostic workloads keep their setup and schedules.
The new mode intentionally records zero sensor samples. M6 remains unaccepted.

The current targeted beta is **0.1.0 (12.1)**, source
`1340837ad24ef000d24e9dedf26e18ae9efa191e`. Native
[run 42300865187](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42300865187)
passed 31 hosted tests, six UI tests, 13 native ZIP validations and both device
compile profiles. **CAN 29-bit diagnostic** explicitly selects the new protocol,
uses matching standard functional addressing, retains full response IDs and gates
RPM/coolant on same-generation supported-PID evidence. No automatic protocol
search or enhanced Alfa/TCM interpretation is enabled. The
[single parked packet](../planning/milestone_6_can29_session.md) batches acquisition,
conditional reconnect and a later SAE reference comparison, then export indoors.
Signed [run 42302376611](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/42302376611)
uploaded **0.1.0 (12.1)** on November 6 at **4:11:57 p.m. CST**, with no errors.
Apple confirmed build VALID / internal IN_BETA_TESTING (uploadedDate
22:13:53 UTC / 4:13:53 p.m. CST). The packet is ready. Documentation updates
after delivery do not change the tested/signed source.

## User setup

Status update — October 30, 2026: Apple Developer enrollment is approved. The
matching distribution certificate/PKCS#12 and App Store profile have been verified
locally and their three signing secrets configured in GitHub's `ios-device`
environment with explicit user approval. The API key, Key ID and Issuer ID are
also configured as encrypted environment secrets with separate user approval.
Workflow-only PR #4 and M5 app PR #3 are merged on master. M5 was accepted
on November 3, 2026; M6 live acquisition is under implementation.
Signed archive/export and native signature verification passed in
[run 41093880633](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/41093880633)
for source `81c8fbf4429c41bcb5294d265afae355466adbc3`, version `0.1.0 (2.1)`.
After correcting an iPad orientation validation rejection,
[run 41095225007](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/41095225007)
uploaded version `0.1.0 (4.1)` from source
`9aa6e9a9b0d0e70e29be966af4d394a0247931a7`. Apple returned `UPLOAD SUCCEEDED with
no errors`. Subsequent read-only API inspection confirmed upload COMPLETE and
build VALID; the user installed build 4.1 on an iPhone 16 Pro / iOS 26.6.2 and
supplied a valid MX+ identity exchange and closed capture. Reconnect and lock/
unlock passed per user report. Remaining physical edge cases are tracked in
[MILESTONE_5.md](MILESTONE_5.md).

An active Apple Developer Program account, App Store Connect app record and
approved bundle `com.kaleevans.vehicleflightrecorder` are required. Use marketing
version 0.1.0; build numbers use workflow run.attempt (for example, 1.1). Manufacturer approval
has already been reported complete; do not recreate it. Keep PPID 221699-0001 in
submission metadata when Apple requires it.

In GitHub, create environment `ios-device` and add these environment secrets:

| Secret | Value |
| --- | --- |
| IOS_DISTRIBUTION_P12_BASE64 | Base64 Apple Distribution certificate **with private key**, exported as password-protected PKCS#12 |
| IOS_DISTRIBUTION_P12_PASSWORD | Its nonempty export password |
| IOS_APPSTORE_PROFILE_BASE64 | Base64 App Store distribution profile for the exact bundle/team/certificate |
| ASC_KEY_ID | App Store Connect API key ID (only needed for upload) |
| ASC_ISSUER_ID | That API key's issuer ID (only needed for upload) |
| ASC_PRIVATE_KEY_BASE64 | Base64 downloaded AuthKey .p8 file (only needed for upload) |

Never paste signing keys/passwords/tokens into chat. Keep local credentials outside
tracked sources; .p12 and .mobileprovision are ignored. A certificate downloaded
from Apple without its corresponding private key is insufficient. With WSL and no
Mac, generate the private key/CSR using OpenSSL, upload only the CSR in Apple's
Certificates portal, download the resulting distribution certificate, then combine
that certificate and the retained private key into a password-protected PKCS#12.
The Apple portal/account holder must perform account enrollment, certificate/profile
issuance, agreement acceptance and app-record setup. Provisioning assets cannot
be manufactured from the simulator build.

Reference: [GitHub signing on macOS runners](https://docs.github.com/en/actions/how-tos/deploy/deploy-to-third-party-platforms/sign-xcode-applications),
[Apple TestFlight overview](https://developer.apple.com/help/app-store-connect/test-a-beta-version/testflight-overview/).
The runner imports assets into a temporary keychain and cleans them afterward.
Only the IPA, non-secret provenance and framework build manifest are retained.

GitHub requires a workflow-dispatch definition on the default branch before it
appears as a runnable workflow. Workflow-only
[PR #4](https://github.com/KaleTEvans/VehicleFlightRecorder/pull/4) supplies that
definition. Select the workflow ref and set `source_ref` to the Milestone 5 branch
or a full 40-character commit SHA (abbreviated SHAs are treated as ref names).
Until M5 is merged, select `codex/milestone-5-mxplus-transport` as the workflow
ref for its corrected upload receipt check. The bootstrap on master predates that
fix and can show a false green result when altool reports a rejection but exits
zero. A blank source uses the selected workflow ref. Run with upload disabled first to
verify signing/export. An App Store IPA is not directly sideloadable; choose
upload on a subsequent run for App Store Connect/TestFlight processing. Add the
owner as an internal tester, resolve Apple's processing/compliance prompts, and
install through TestFlight on the iPhone. Public release is not required. Actual
processing and installation remain user-observed checks, not inferred from upload.

## Install the first TestFlight build

For an uploaded build (the first accepted installation is `0.1.0 (4.1)`), open App Store Connect → Apps → Vehicle
Flight Recorder → TestFlight → iOS. Wait for Apple to finish processing the build; a
successful transport upload alone is not confirmation that Apple accepted it.
Open the matching version/build and resolve any processing or compliance prompts
in the account. Account/legal declarations remain the account holder's decision.

Under Internal Testing, create a private group (for example, `Owner Testing`),
add the account holder as a tester, and add the processed build to that group.
On the physical iPhone, install Apple's TestFlight app, accept the invitation with
the intended Apple Account, and install Vehicle Flight Recorder. This does not
publish the app to the public App Store. Follow Apple's
[internal tester steps](https://developer.apple.com/help/app-store-connect/test-a-beta-version/add-internal-testers/)
if the group or invitation controls differ.

Record the installed marketing/build version, iPhone model and iOS version before
starting the parked checks below. The physical build opens Adapter Transport;
the mock dashboard remains the simulator path. Installing or launching successfully
is one acceptance row, not proof that the scanner connection works.

## Parked acceptance

Record build/commit, iPhone model/iOS, adapter model/firmware and test time. Pair
MX+ through iOS settings and close other apps holding its session. Open Adapter
Transport and verify startup discovery opens both streams to `ready`. Use
Discover / Connect for a later explicit retry. Confirm
requested/accepted bytes remain zero until the identity action is enabled and
tapped. Read identity once; preserve the actual response or timeout. Disconnect,
export the closed JSONL capture and validate it using the Python command in
MILESTONE_5.md. Repeat connect/disconnect and ensure no automatic stale replay.

Preserved captures from earlier launches are also available in Files → On My
iPhone → Vehicle Flight Recorder → ConnectionCaptures. Copy evidence before
editing or removing anything in the app documents folder.

Record foreground, background, lock/unlock, suspension, OS termination and user
force-quit separately, including timing and recovery steps. Missing final close
means interruption; do not edit it into a successful capture. No state is assumed
to support universal background wake or hands-off drive creation. Appetize cannot
exercise physical Bluetooth/MFi. Milestone 6 adds controlled vehicle reads; M7
owns automatic drive policy.

## Upload SDK requirement

[Apple requires Xcode 26 / iOS 26 SDK or later for App Store Connect uploads](https://developer.apple.com/news/upcoming-requirements/?id=04282026a)
since April 28, 2026. The device profile inherits the fixed Go/mobile pins and iOS
17 deployment floor, overriding only Xcode to 26.0.1 (SDK 26.0). Preview CI also
compiles an unsigned Release archive with that profile. This separate compile
checks source compatibility. Signing/export, Apple processing and the first phone
installation are now verified separately; physical coverage is tracked in the M5
report.

## Preparing the certificate from WSL

Once the account is enrolled, these commands let the account holder prepare
signing material without owning a Mac. Run from the project root. OpenSSL asks
for passwords interactively; retain them locally and use GitHub's secret editor.
Upload only `distribution.csr` to Apple's certificate-creation page, selecting an
Apple Distribution certificate. Download that certificate as `distribution.cer`
into `.cache/signing`, then run the conversion/export commands below.

```bash
umask 077
mkdir -p .cache/signing
openssl genpkey -algorithm RSA -aes-256-cbc -pkeyopt rsa_keygen_bits:2048 \
  -out .cache/signing/distribution.key
openssl req -new -key .cache/signing/distribution.key \
  -out .cache/signing/distribution.csr
# After downloading the issued Apple certificate:
openssl x509 -inform DER -in .cache/signing/distribution.cer \
  -out .cache/signing/distribution.pem
openssl pkcs12 -export -inkey .cache/signing/distribution.key \
  -in .cache/signing/distribution.pem -out .cache/signing/distribution.p12
```

Create the matching App Store Connect distribution profile in Apple's portal
using this certificate and the approved explicit app ID. Download the profile
locally. Encode the .p12 and .mobileprovision with `base64 -w0` when populating the
corresponding GitHub environment secrets. Keep these values out of chat and
tracked files. Do not regenerate a private key after issuing a certificate; a
new key would require a new certificate. The .cache folder is ignored.

Apple enrollment: [Enroll in the Apple Developer Program](https://developer.apple.com/programs/enroll/).


The device workflow records the actual checked-out source commit separately from
the dispatching workflow ref. Its bootstrap can therefore be available on master
while building the unmerged M5 branch for hardware acceptance. It fails early if
the selected source lacks the device-profile/signing scripts. Making a workflow
available is not completion of Milestone 5 or authorization for a public release.

## Focused replug recovery check

Accepted November 3, 2026 for build 6.1: the user reports automatic app reconnection
in approximately 1–2 seconds after iOS Bluetooth reconnects. The procedure below
is retained for future regression checks; it is not an outstanding M5 gate.

The user accepted the flicker correction and initial later attachment in build
5.1, but its automatic recovery after unplugging an established connection failed.
For corrected recovery build **0.1.0 (6.1)**, keep the vehicle parked, establish `ready`,
unplug MX+, then reconnect it through iOS Bluetooth while leaving the app running.
Allow the configured backoff plus up to one two-second discovery interval once
`com.obdlink` is available. Expect a fresh capture, zero requested/accepted bytes,
no automatic identity resend, and no need to restart or press Connect. Export the
old closed capture before replacing it if possible; retained files are also in
Files → On My iPhone → Vehicle Flight Recorder → ConnectionCaptures.

Also verify that pressing the app's Disconnect intentionally leaves it disconnected
until Discover / Connect is pressed. Three failed opens produce an explicit retry-
paused notice; a new candidate set or explicit Connect permits another attempt.
These checks do not establish OS wake or execution during suspension/termination.

The recovery build also encodes the user's existing “None of the algorithms
mentioned above” answer as `ITSAppUsesNonExemptEncryption = NO`. No cryptography
behavior was added by these transport fixes. Reassess this metadata when shipped
app/dependency encryption functionality changes; signing/API credentials are build
inputs and are not bundled as app encryption functionality. Build 5.1 predates
this metadata and still required its own questionnaire.
