#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
[[ "$(uname -s)" == Darwin ]] || {
  echo "Go iOS binding requires macOS and Xcode Apple SDKs; no framework built." >&2
  exit 1
}
source "$ROOT/scripts/mobile_toolchain.env"
export GOCACHE="${GOCACHE:-$ROOT/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.cache/go-mod}"
export GOPATH="$ROOT/.cache/go-path" GOBIN="$ROOT/.tools" TMPDIR="$ROOT/.cache/tmp"
export PATH="$GOBIN:$PATH"
mkdir -p "$GOBIN" "$TMPDIR" "$ROOT/build/ios"
xcrun --sdk iphonesimulator --show-sdk-path
rm -rf "$ROOT/build/ios/VehiclePack"
cp -R "$ROOT/vehicle-packs/alfa-giulia-qv-2017" "$ROOT/build/ios/VehiclePack"
cd "$ROOT/core"
go install "golang.org/x/mobile/cmd/gomobile@$MOBILE_VERSION"
go install "golang.org/x/mobile/cmd/gobind@$MOBILE_VERSION"
go version -m "$GOBIN/gomobile"
go version -m "$GOBIN/gobind"
gomobile bind -target=iossimulator/arm64 -iosversion=17.0 \
  -o "$ROOT/build/ios/VehicleFlightRecorderCore.xcframework" ./mobile
