# Current Milestone 3 preview work

The workflow now builds the real Go simulator XCFramework, links the native
dashboard, runs hosted native bridge/export tests and validates app-container ZIPs
in Python before packaging/upload. See [Milestone 3](MILESTONE_3.md) for current
status, commands, toolchain pinning and interactive acceptance. The historical
Milestone 0.5 results below describe the earlier static shell, not the new build.

# Milestone 0.5: native iOS preview from Windows/WSL

## Current acceptance status

The pipeline targets the existing SwiftUI shell. The
[remote macOS build and simulator artifact publication passed](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40019569605)
for source commit `1c443b2bd429efa25b7a42e3118f5745ab3bb465` on October 23, 2026.
The [Linux CI run passed](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40019569099)
for the same source. A subsequent
[build and automated Appetize upload passed](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40019983986)
for commit `2c4a3e96c9af8305f0b7541dafb7100a00b33759`.
The [actual Appetize preview](https://appetize.io/app/nrnzclustneju2ltjppkozdkta) and
[latest build artifact](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40019983986/artifacts/12530681097)
are available. **Browser launch: VERIFIED BY USER on October 23, 2026.** The user
opened the preview and confirmed all four expected labels: Vehicle Flight
Recorder, MOCK / SIMULATOR, Transport: Mock / Not Connected, and Recorder: Not
Started. This is user-observed browser evidence, not an automated screenshot.
**Milestone 0.5 is complete.** The deployed preview still represents its static
SwiftUI shell. See [MILESTONE_1.md](MILESTONE_1.md) for the later local Go/Python
recorder acceptance; that work has not yet been integrated into this native preview.

The actual downloaded ZIP passed local integrity and root-layout verification.
Its metadata identifies the approved app/version, `iPhoneSimulator`, Xcode 16.4,
and Simulator SDK 18.5. The artifact is
[VehicleFlightRecorder-ios-simulator](https://github.com/KaleTEvans/VehicleFlightRecorder/actions/runs/40019569605/artifacts/12529323361).
Its inner ZIP SHA256 is
`63eb3a62ba9ae5d3a8c01c955849470d461b87aeab82dece2ce7e3fccde0b98d`.
Artifacts expire after 14 days; rerun the workflow if needed.

## What runs where

| Location | Responsibility |
| --- | --- |
| Windows/WSL | Edit Swift/Go/Python, run Linux checks, commit and push |
| GitHub `macos-15` runner | XcodeGen generation, real unsigned ARM simulator build, verification and ZIP |
| GitHub Actions artifacts | Downloadable simulator ZIP, retained 14 days |
| Appetize | Browser-hosted iOS simulator, manual or optional automated upload |
| Physical iPhone, later | Separately signed device build, real MX+ / EASession tests |

The application remains native SwiftUI + future shared Go + offline Python. There
is no parallel web UI. The current app links ExternalAccessory.framework but has
no Go XCFramework reference. Its transport is unconditionally MockVehicleTransport;
both transport implementations reject connections/writes. No session, diagnostic
commands, proprietary data, or vehicle logs are included in the build.

## GitHub prerequisites and first push

Use the existing repository
<https://github.com/KaleTEvans/VehicleFlightRecorder>. Actions must be enabled, and
the account must have available macOS runner capacity/minutes. This remote URL is
HTTPS; its SSH equivalent is `git@github.com:KaleTEvans/VehicleFlightRecorder.git`.
Use whichever authentication method you have configured. Never embed tokens in URLs.

The initial local checkout had no commits or remote. Inspect before changing it:

```bash
cd ~/dev/VehicleFlightRecorder
git status
git remote -v
./scripts/test_all.sh
```

If `origin` is absent, add it once:

```bash
git remote add origin https://github.com/KaleTEvans/VehicleFlightRecorder.git
```

Review source and ignore rules, then make the initial commit and push the current
branch. Do not commit `.venv`, `.cache`, build output, credentials, or captures:

```bash
git add .
git diff --cached --stat
git commit -m "Bootstrap Vehicle Flight Recorder and add iOS simulator preview"
git push -u origin HEAD
```

Later runs only need the intended source changes committed and pushed. Keep the
workflow on the repository's default branch: GitHub requires it there for manual
dispatch to appear. No branch rename is required; check the actual default branch
under GitHub repository settings. This workspace has checksum-verified Linux AMD64
tools installed locally: GitHub CLI 2.103.2 and actionlint 1.7.12. They live in
ignored `.tools/` and are not application dependencies or committed binaries.
On a fresh workstation install them from the official
[GitHub CLI releases](https://github.com/cli/cli/releases) and
[actionlint releases](https://github.com/rhysd/actionlint/releases), verifying the
release checksums, or use your preferred trusted package manager.

For this WSL checkout, add the local tools and keep GitHub CLI configuration in the
ignored project cache. Complete browser authentication without pasting tokens into
chat or source files:

```bash
export PATH="$PWD/.tools:$PATH"
export GH_CONFIG_DIR="$PWD/.cache/gh"
gh auth login --hostname github.com --git-protocol https --web
gh auth setup-git
```

Optional CLI dispatch commands after authentication are:

```bash
gh workflow run ios-preview.yml --repo KaleTEvans/VehicleFlightRecorder
gh run list --workflow ios-preview.yml --repo KaleTEvans/VehicleFlightRecorder
```

## Manual Actions run

1. Open the repository in GitHub, choose **Actions → iOS Simulator Preview**.
2. Choose **Run workflow**, select the pushed branch, and choose whether to upload
   to Appetize. Upload defaults to enabled but is skipped if the token is absent.
3. Open the run. The log reports `xcode-select -p`, Xcode version, installed SDKs,
   runner architecture, and XcodeGen version. The runner supplies Xcode; Homebrew
   installs XcodeGen only if it is missing. We do not select an unverified Xcode path.
4. The workflow generates the existing `ios/project.yml` definition and builds
   scheme/target `VehicleFlightRecorder`, Debug, iOS 17+, ARM64 Simulator, with
   `CODE_SIGNING_ALLOWED=NO`. It does not build the future Go recorder/framework.
5. Packaging discovers exactly one top-level `.app` in Debug-iphonesimulator,
   checks approved display name, bundle ID, version, simulator platform and executable,
   confirms ARM64, and uses macOS `ditto` to preserve native bundle attributes.
   The ZIP is checked for corruption and for the `.app` at ZIP root.
6. Artifact publication precedes optional Appetize upload. Build, packaging, and
   configured-upload failures fail the run. Nothing suppresses these errors.

Workflow file: `.github/workflows/ios-preview.yml`. It is manual-only and uses
`contents: read`, a 30-minute timeout, and serialized runs to avoid racing updates
to a reused preview. Existing Linux CI continues to run on pushes/pull requests.
The first remote attempt compiled successfully but failed the packager's `lipo`
argument ordering; the corrected run above passed all build/packaging steps.

## Artifact download and manual Appetize upload

From the Actions run's **Summary → Artifacts**, download
**VehicleFlightRecorder-ios-simulator**. The run summary also links the artifact.
GitHub wraps the artifact in its own ZIP. Extract that outer download once; inside
is **VehicleFlightRecorder-simulator.zip**. Upload this inner ZIP unchanged at
<https://appetize.io/upload>. Its root is `VehicleFlightRecorder.app/`, not a
DerivedData directory or a `Payload/` directory. Use an Appetize account with
available simulator/streaming quota. An `.ipa` device build is not accepted here.

Missing Appetize secrets never block the simulator build or artifact. Uploading
manually is a complete alternative to configuring an API token.

## Optional automated Appetize upload

In GitHub, open **Settings → Secrets and variables → Actions → New repository
secret**. Enter secrets there, not in source files, chat, command history, or logs:

| Secret | Purpose |
| --- | --- |
| `APPETIZE_API_TOKEN` | API token from your Appetize account; enables automated upload |
| `APPETIZE_PUBLIC_KEY` | Optional existing preview/build key to update instead of creating a new one |

The workflow uses the official `appetizeio/github-action-appetize@v1.1.0`, verified
against its tagged inputs and the upstream documented release. It passes `apiToken`,
optional `publicKey`, `platform: ios`, and the local ZIP as `appFile`. A successful
response returns `publicKey`; only then does the summary create the browser link.
The repository now has both secret names configured. The public key returned by
the first successful upload was saved as `APPETIZE_PUBLIC_KEY`, so later uploads
update that preview. If a reused secret key is masked, retrieve the build link
from the Appetize dashboard.
A failed upload leaves the earlier artifact available; fix the token/key/quota or
use manual upload. Never print the API token to troubleshoot authentication.

**Exposure policy:** only this disconnected shell and dummy/mock data may be
uploaded. The action uses the account's access defaults; it does not enforce
private access or set authentication permissions. Appetize build links are normally
runnable by anyone with the link. Before enabling upload, review **Organization →
Session Defaults** and the build's **Settings → App Permissions**, and require
organization authentication if your plan supports it and privacy requires it.
Do not share run summaries/build links indiscriminately. A public GitHub repository
can expose artifact/run links. API-token secrecy alone does not make a preview private.
Uploads set a 120-second inactivity timeout; streaming quotas depend on the account.

## Browser smoke-test checklist

Open the actual successful upload link or the build from the Appetize dashboard,
select an iPhone simulator with iOS 17 or newer, and start the session. Confirm:

- The app launches and displays **Vehicle Flight Recorder**.
- The screen explicitly says **MOCK / SIMULATOR**.
- Transport reads **Mock / Not Connected**; recorder reads **Not Started**.
- The app stays on the shell without crashing or attempting hardware access.

Record the GitHub Actions run URL, artifact, source commit, Appetize build URL,
device/OS selection, and observed result. Upload success alone proves neither
launch nor correct rendering. Record failure/plan limits separately if no launch
can be observed. Do not mark Milestone 0.5 complete until the remote build and
browser launch are actually verified.

The computer-use connector was attempted from this WSL task and failed before
browser inspection with `sandboxCwd is not a local file URI`, including after a
kernel reset. The user performed and confirmed the browser smoke test directly;
no automated screenshot/launch is claimed.

| Simulator preview now | Device testing later |
| --- | --- |
| Unsigned simulator `.app` inside ZIP | Signed device `.app`/`.ipa`, valid provisioning/distribution |
| SwiftUI shell/mock state | Physical iPhone and MX+ |
| No Apple signing secrets required | Developer account/signing setup needed |
| Cannot connect to MX+ or prove background logging | EASession, background lifecycle, reconnect and vehicle safety validation |

## Troubleshooting

| Symptom | Next check |
| --- | --- |
| Workflow absent / no Run workflow button | Push workflow to default branch; enable Actions; check write access |
| Push rejected / authentication failed | Configure GitHub HTTPS or SSH authentication; verify access to the supplied repository |
| macOS job queued/blocked | Check runner availability, account billing/minutes and Actions policy |
| XcodeGen unavailable | Inspect Homebrew installation log; confirm minimum XcodeGen 2.42+ |
| Project/scheme missing | Check project generation log and the existing `ios/project.yml`; do not create a replacement target |
| Swift compilation fails | Fix the first real compiler error; Linux checks cannot prove iOS compilation |
| Missing Go framework | Current shell has none; if later introduced, supply a simulator-compatible XCFramework |
| No/multiple `.app` products or wrong identity | Inspect build-products directory and target settings; packaging deliberately fails |
| Signing/team requested | Confirm Simulator SDK/destination and `CODE_SIGNING_ALLOWED=NO`; do not add provisioning secrets |
| Appetize skipped | Configure `APPETIZE_API_TOKEN` and enable upload, or upload the saved ZIP manually |
| Appetize rejects ZIP | Upload inner ZIP containing `.app` at root, not the GitHub artifact wrapper or device IPA |
| Upload unauthorized/not found | Check account token and optional existing public key; do not expose credentials |
| Preview inaccessible / out of streaming time | Check organization/build permissions, simulator OS and plan quota |
| Artifact expired | Manually rerun the workflow; retention is 14 days |

## Local infrastructure checks

`./scripts/test_all.sh` includes Go tests/vet, analytics pytest, and seven metadata/
archive packaging tests. These tests explicitly use synthetic fixtures and do not
produce an installable app. The packager rejects local Linux execution.

For stronger workflow validation, install actionlint locally (for example under
ignored `.tools/`) and run it on both workflows. Shell syntax can be checked with:

```bash
for script in scripts/*.sh; do bash -n "$script"; done
analytics/.venv/bin/python -m unittest discover -s scripts/tests -v
.tools/actionlint .github/workflows/ci.yml .github/workflows/ios-preview.yml
```

## Official references

- [GitHub macOS hosted runners](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)
- [Workflow artifacts](https://docs.github.com/en/actions/tutorials/store-and-share-data)
- [Official Appetize action and release usage](https://github.com/appetizeio/github-action-appetize)
- [Tagged action inputs/output](https://github.com/appetizeio/github-action-appetize/blob/v1.1.0/action.yml)
- [Appetize iOS bundle format](https://docs.appetize.io/platform/app-management/uploading-apps/ios)
- [Appetize access permissions](https://docs.appetize.io/platform/app-management/app-permissions)
