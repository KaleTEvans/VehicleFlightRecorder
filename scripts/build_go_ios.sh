#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
[[ "$(uname -s)" == Darwin ]] || {
  echo "Go iOS binding requires macOS and Xcode Apple SDKs; no framework built." >&2
  exit 1
}
python3 "$ROOT/scripts/mobile_build_config.py"
source "$ROOT/scripts/mobile_toolchain.env"
export DEVELOPER_DIR="/Applications/Xcode_${MOBILE_XCODE_VERSION}.app/Contents/Developer"
xcodebuild -version
export GOCACHE="${GOCACHE:-$ROOT/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.cache/go-mod}"
export GOPATH="$ROOT/.cache/go-path" GOBIN="$ROOT/.tools" TMPDIR="$ROOT/.cache/tmp"
export PATH="$GOBIN:$PATH"
mkdir -p "$GOBIN" "$TMPDIR" "$ROOT/build/ios"
xcrun --sdk iphonesimulator --show-sdk-path
xcrun --sdk iphoneos --show-sdk-path
rm -rf "$ROOT/build/ios/VehiclePack"
cp -R "$ROOT/vehicle-packs/alfa-giulia-qv-2017" "$ROOT/build/ios/VehiclePack"
cd "$ROOT/core"
[[ "$(go env GOVERSION)" == "$MOBILE_GO_VERSION" ]] || {
  echo "Native binding requires $MOBILE_GO_VERSION (iOS cgo alignment fix); found $(go env GOVERSION)." >&2
  exit 1
}
go install "golang.org/x/mobile/cmd/gomobile@$MOBILE_VERSION"
go install "golang.org/x/mobile/cmd/gobind@$MOBILE_VERSION"
go version -m "$GOBIN/gomobile"
go version -m "$GOBIN/gobind"
MODULE_VERSION=$(go list -m -f '{{.Version}}' golang.org/x/mobile)
[[ "$MODULE_VERSION" == "$MOBILE_VERSION" ]] || { echo "Resolved mobile runtime differs from tool pin" >&2; exit 1; }
gomobile bind -target="$MOBILE_TARGETS" -iosversion="$MOBILE_IOS_VERSION" \
  -o "$ROOT/build/ios/VehicleFlightRecorderCore.xcframework" ./mobile

python3 "$ROOT/scripts/verify_ios_framework.py" \
  "$ROOT/build/ios/VehicleFlightRecorderCore.xcframework" \
  --output "$ROOT/build/ios/native-build-manifest.json"
