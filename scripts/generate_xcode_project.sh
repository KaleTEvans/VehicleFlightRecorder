#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
[[ "$(uname -s)" == Darwin ]] || {
    echo "Xcode project generation requires macOS for this workflow. On a Mac, install Xcode and XcodeGen, then rerun this script. WSL cannot validate the iOS app." >&2
    exit 1
}
command -v xcodegen >/dev/null || {
    echo "XcodeGen is required (brew install xcodegen). See docs/DEVELOPMENT.md." >&2
    exit 1
}
xcrun --sdk iphoneos --show-sdk-path >/dev/null || {
    echo "Select a full Xcode installation with xcode-select and complete Xcode's first-run setup." >&2
    exit 1
}
cd "$ROOT/ios"
xcodegen generate --spec project.yml
