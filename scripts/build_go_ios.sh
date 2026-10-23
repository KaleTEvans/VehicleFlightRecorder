#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
# Apple SDKs and a full Xcode installation are required. WSL cannot build iOS.
[[ "$(uname -s)" == Darwin ]] || {
    echo "Go iOS binding requires macOS and Xcode's Apple SDKs. Run this script on a Mac; no XCFramework was built." >&2
    exit 1
}
for tool in go gomobile gobind xcrun; do
    command -v "$tool" >/dev/null || {
        echo "Missing $tool. See the Go XCFramework setup in docs/DEVELOPMENT.md." >&2
        exit 1
    }
done
xcrun --sdk iphoneos --show-sdk-path >/dev/null
xcrun --sdk iphonesimulator --show-sdk-path >/dev/null
export GOCACHE="${GOCACHE:-$ROOT/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.cache/go-mod}"
export TMPDIR="$ROOT/.cache/tmp"
mkdir -p "$ROOT/build/ios" "$TMPDIR"
cd "$ROOT/core"
# The initial facade exports version metadata only. Framework integration and
# reproducible mobile-tool version pinning are hardened in Milestone 4.
gomobile bind -target=ios,iossimulator -iosversion=17.0 \
    -o "$ROOT/build/ios/VehicleFlightRecorderCore.xcframework" ./mobile
