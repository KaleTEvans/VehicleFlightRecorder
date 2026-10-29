# Signed physical-iPhone builds

The separate `Signed iPhone build` workflow prepares an App Store distribution
IPA and can explicitly upload it for TestFlight processing. It uses the accepted
Go/runtime pins and iOS ARM64 slice, with a separate Xcode 26.0.1 device
profile in scripts/device_toolchain.env. The accepted preview stays Xcode 16.4. Simulator ZIPs remain separate. This workflow
has not yet been executed with signing assets or installed on a phone.

## User setup

An active Apple Developer Program account, App Store Connect app record and
approved bundle `com.kaleevans.vehicleflightrecorder` are required. Use marketing
version 0.1.0; build numbers come from workflow run/attempt. Manufacturer approval
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
appears as a runnable workflow. After this workflow definition is merged, select
the implementation branch and run it with upload disabled first to
verify signing/export. An App Store IPA is not directly sideloadable; choose
upload on a subsequent run for App Store Connect/TestFlight processing. Add the
owner as an internal tester, resolve Apple's processing/compliance prompts, and
install through TestFlight on the iPhone. Public release is not required. Actual
processing and installation remain user-observed checks, not inferred from upload.

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
checks source compatibility, while signing/processing/installation remain pending.
