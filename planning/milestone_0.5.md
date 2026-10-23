# Codex Handoff — Vehicle Flight Recorder, Milestone 0.5 Only

## Assignment

Implement **Milestone 0.5 — GitHub Actions macOS build + Appetize.io browser preview** in the **existing** repository at:

```text
~/dev/VehicleFlightRecorder
```

**Milestone 0 is already complete.** Do not rebuild the repository, generate a new project unnecessarily, rename its app, or begin Milestone 1. This assignment builds and validates a remote iOS Simulator preview pipeline for the current SwiftUI application; the recorder/mock driving implementation remains a subsequent task.

## Authoritative context

Before editing anything, read in full:

```text
~/dev/VehicleFlightRecorder/alfa_flight_recorder_mvp_plan.md
~/dev/VehicleFlightRecorder/vehicle_health_enterprise_architecture.md
```

Pay particular attention to **MVP §22 / Milestone 0.5**, §2 (language boundaries), §4 (app/config), §16–17 (adapter transport), and enterprise §22 (engineering environments). **Read the current source tree and `git status`** before choosing paths or commands. Preserve any legitimate changes already present. The Markdown filenames above are fixed, even if the repository has additional documents.

Approved app identity:

```text
App name:             Vehicle Flight Recorder
CFBundleIdentifier:   com.kaleevans.vehicleflightrecorder
App version:          0.1.0
OBDLink iAP protocol: com.obdlink
MFi PPID:             221699-0001
```

The development workstation is **Windows + WSL**. There is **no personal Mac**; there is an iPhone that will be used later for real MX+ connectivity. Native iOS builds are performed on GitHub's **macOS** hosted runners. Appetize previews **iOS Simulator** bundles only. A simulator must use the mock transport; it cannot connect to a physical OBDLink MX+ over Apple's External Accessory/iAP interface.

## Required work

1. **Inspect and preserve Milestone 0.** Identify the current Xcode/XcodeGen project, scheme, app target, deployment target, source structure, Go framework references (if any), tests and development scripts. Do not blindly copy assumed project names from earlier prompts.

2. Add **`.github/workflows/ios-preview.yml`** using an explicit manual `workflow_dispatch` trigger and least-privilege `contents: read` permissions. Use an available GitHub-hosted macOS runner (initially `macos-15`, adjusting only when necessary). Print the selected Xcode version in the job so builds can be diagnosed. Use project generation only if the existing repo needs it.

3. **Build the real SwiftUI app for iOS Simulator with `xcodebuild`**, not a fake test program. Signing must be disabled for the simulator build (`CODE_SIGNING_ALLOWED=NO`), and the initial configuration must select `MockVehicleTransport` / disconnected-development state. The simulator must not open `EASession` or send OBD/CAN commands. If Xcode references a Go XCFramework, ensure that the cloud build provides the correct simulator binary or existing compile-only bridge. Do not claim actual Go recorder integration if it is only a placeholder.

4. Discover the generated `.app` bundle, verify the expected app identity/bundle is present, package it as a ZIP containing `<ActualAppName>.app` at ZIP root, and always publish the ZIP via `actions/upload-artifact`. Build or packaging errors must fail the job; do not suppress errors.

5. Add an **optional Appetize upload step**, using the official `appetizeio/github-action-appetize` GitHub Action (check the currently documented release and inputs). Take the API token from the **GitHub repository secret `APPETIZE_API_TOKEN`**. Optionally reuse **`APPETIZE_PUBLIC_KEY`** as a secret to update the same preview instead of generating a new app. If the token isn't configured, the simulator build and artifact upload must still pass and the workflow summary must tell the user how to upload the ZIP manually. Never print credentials or place secrets in repository files.

6. Display the preview link or Appetize public key in the workflow summary **when the upload actually succeeds**, and document where the build ZIP is downloaded from the GitHub Actions run. Do not invent a preview URL before a successful upload. Keep Appetize exposure/privacy settings explicit; use dummy/mock data only.

7. Update README and development documentation for **Windows/WSL → Git push → manually run Actions → artifact/Appetize → browser preview**, including prereqs, exact GitHub secret names, common failure causes, and the distinction between simulator tests and later signed TestFlight/physical-iPhone tests. Do **not** introduce a React UI; preserve SwiftUI + Go + Python.

8. Run all available local tests (Go and Python), as well as any workflow/YAML static checks available in WSL. If the repository already has CI checks, keep them working. Provide actionable instructions for user-only interactions that Codex cannot execute (creating secrets, selecting an Actions workflow, downloading an artifact, observing the preview).

## Milestone boundaries

Do **not** implement:

- Milestone 1's Go recorder, mock-drive engine, JSONL storage or analytics;
- Milestone 2 incident logic or Milestone 3 dashboard screens beyond the already-existing Milestone 0 shell;
- OBDLink initialization, live `EASession` connections or any AT/ST/CAN requests;
- code signing, App Store submissions, TestFlight device pipeline or provisioning secrets;
- a web/React dashboard, cloud database, subscription system or enterprise features.

Do not accidentally change the fixed approved bundle identifier. Use an unmistakable `MOCK / SIMULATOR` indication for previews.

## Acceptance and reporting

**Must produce:** a valid macOS workflow, a real simulator build ZIP via GitHub Actions, and clear documentation. **Target end-to-end validation:** actual successful remote build and observed launch in Appetize's browser simulator. If credentials/repository access are not available, complete everything possible, then explicitly mark the remote build/preview acceptance **PENDING USER VERIFICATION** — do not claim success.

Report:

1. Every modified/created file, with explanation.
2. Commands and local test results actually executed.
3. Exact cloud workflow trigger path, build-artifact name and upload behavior.
4. Any dependencies on the user (GitHub remote, secret, Appetize account, manual preview check).
5. Real GitHub Actions outcome/URL and Appetize URL if observed; otherwise an explicit pending checklist.
6. Any deviation from the written plans or pre-existing Milestone 0 architecture.
7. State explicitly: **Milestone 1 not started.**

## References for implementation verification

- https://docs.github.com/en/actions/reference/runners/github-hosted-runners
- https://docs.github.com/en/actions/tutorials/store-and-share-data
- https://github.com/appetizeio/github-action-appetize
- https://docs.appetize.io/platform/app-management/uploading-apps/ios
