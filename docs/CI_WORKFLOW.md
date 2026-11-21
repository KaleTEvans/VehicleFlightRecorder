# Development and beta verification

Give evidence findings early. Validate a reported failure locally, use focused
native cases while debugging, freeze the app change, then run full native
acceptance once. Prepare one signed candidate after validation; reuse the exact
approved IPA for TestFlight upload. Record actual timings instead of promising
an unmeasured turnaround. Keep car tests in one substantial visit/handoff.

## Fast native iteration

The simulator workflow accepts `test_scope=targeted` and comma-separated
`test_selectors`. Each selector is an existing XCTest target, class, or method.
Unknown selectors, empty targeted selections and shell arguments are rejected
before framework compilation. Example from the repository root:

```bash
GH_CONFIG_DIR="$PWD/.cache/gh" ./.tools/gh workflow run ios-preview.yml \
  --ref codex/milestone-7-hands-off-recording \
  -f upload_to_appetize=false \
  -f test_scope=targeted \
  -f test_selectors=VehicleFlightRecorderTests/TransportDebugModelTests,VehicleFlightRecorderUITests/EvidenceNavigationTests/testRestoredParkedValidationSetupRequiresFreshConfirmationOnlyWhenRearming
```

Targeted runs compile the actual simulator app and selected XCTest cases, retain
logs/results/screenshots (including failed-test attachments) and
`native-test-scope.json`, and require successful
output for every requested selector. Zero-test success cannot qualify. They
skip unsigned device compilation, the second device SDK job, full fixture
population assertions, simulator packaging and Appetize publication. The
artifact is `VehicleFlightRecorder-native-targeted`; it is not full acceptance.

## Final acceptance and delivery

Once local/targeted checks pass and app changes are frozen:

```bash
GH_CONFIG_DIR="$PWD/.cache/gh" ./.tools/gh workflow run ios-preview.yml \
  --ref codex/milestone-7-hands-off-recording \
  -f upload_to_appetize=false -f test_scope=full
```

Full is the default and forbids test filters. It runs both XCTest targets,
unsigned device compilation, the upload-compatible device SDK job, all native
fixture exports and independent Python replay/reconstruction. It retains the
usual acceptance and simulator artifacts. A filtered run cannot substitute for
this gate. The simulator app is built by `xcodebuild test`; a separate generic
simulator prebuild is no longer repeated.

Only afterward prepare a signed build with `upload_testflight=false`. Upload
only the exact user-approved retained IPA, verifying source/build/checksum.
Signed/upload workflows remain noncancellable; Appetize publication is explicit,
full-only and also protected from automatic cancellation. No upload is inferred
from a successful verification run.

## Caches and triggers

- Core/analytics runs on pushes to repository branches and manual dispatch.
  Push checks attach to PR commits, so a second PR trigger is unnecessary.
  Markdown-only pushes create no code-test run. Superseded core runs cancel.
- Preview verification cancels an older run for the same branch and scope;
  publication has a separate protected concurrency group.
- Go compiler/module caches use OS, architecture, pinned Go, pinned Xcode,
  toolchain/build flags and source/dependency hashes, with compatible incremental
  restore prefixes. Frameworks are still generated and verified from current
  source every run. Verified Go caches save before UI tests, so a later test
  failure does not discard the expensive compiler warm-up. Python caches retain
  downloads; editable installation is fresh.
- No virtual environment, DerivedData, signed app, signing keychain, certificate,
  profile, API key, phone original or test result is a dependency cache entry.
  Prepared IPA artifacts retain their separate checksum/approval contract.

References: [GitHub workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax)
and [official cache action](https://github.com/actions/cache). These changes
reduce repeat work; cold runners, simulator startup and full tests still take time.
