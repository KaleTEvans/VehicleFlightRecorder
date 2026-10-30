# Signed physical-iPhone builds

The separate `Signed iPhone build` workflow prepares an App Store distribution
IPA and can explicitly upload it for TestFlight processing. It uses the accepted
Go/runtime pins and iOS ARM64 slice, with a separate Xcode 26.0.1 device
profile in scripts/device_toolchain.env. The accepted preview stays Xcode 16.4.
Simulator ZIPs remain separate. The first signed archive/export passed on October
30, 2026; phone installation and accessory acceptance remain pending.

## User setup

Status update — October 30, 2026: Apple Developer enrollment is approved. The
matching distribution certificate/PKCS#12 and App Store profile have been verified
locally and their three signing secrets configured in GitHub's `ios-device`
environment with explicit user approval. The API key, Key ID and Issuer ID are
also configured as encrypted environment secrets with separate user approval.
Workflow-only PR #4 is merged on master; the M5 app remains in draft PR #3.
Signed archive/export and native signature verification passed in
[run 41093880633](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/41093880633)
for source `81c8fbf4429c41bcb5294d265afae355466adbc3`, version `0.1.0 (2.1)`.
After correcting an iPad orientation validation rejection,
[run 41095225007](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/41095225007)
uploaded version `0.1.0 (4.1)` from source
`9aa6e9a9b0d0e70e29be966af4d394a0247931a7`. Apple returned `UPLOAD SUCCEEDED with
no errors`. Apple processing, phone installation and physical acceptance have not
been observed.

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

For the uploaded `0.1.0 (4.1)` build, open App Store Connect → Apps → Vehicle
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
checks source compatibility. Signing/export are now verified separately; Apple
processing and phone installation remain pending.

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
