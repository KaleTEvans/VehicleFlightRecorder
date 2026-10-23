#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
PYTHON="${PYTHON:-python3}"

command -v go >/dev/null || {
    echo "Go 1.24+ is required. Install from https://go.dev/dl/ and add its bin directory to PATH." >&2
    exit 1
}
command -v "$PYTHON" >/dev/null || {
    echo "Python 3.11+ is required. Set PYTHON to its executable path." >&2
    exit 1
}
"$PYTHON" -c 'import sys; sys.exit(0 if sys.version_info >= (3, 11) else "Python 3.11+ is required; set PYTHON to a newer interpreter.")'
export GOCACHE="${GOCACHE:-$ROOT/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.cache/go-mod}"
export PIP_CACHE_DIR="${PIP_CACHE_DIR:-$ROOT/.cache/pip}"
export TMPDIR="$ROOT/.cache/tmp"
mkdir -p "$TMPDIR"
(cd "$ROOT/core" && go list ./... >/dev/null)
if [[ ! -x "$ROOT/analytics/.venv/bin/python" ]]; then
    "$PYTHON" -m venv "$ROOT/analytics/.venv" || {
        echo "Could not create the virtual environment. Install venv support for your Python, then retry." >&2
        exit 1
    }
fi
"$ROOT/analytics/.venv/bin/python" -c 'import sys; sys.exit(0 if sys.version_info >= (3, 11) else "Existing analytics/.venv needs Python 3.11+; recreate it with the selected interpreter.")'
"$ROOT/analytics/.venv/bin/python" -m pip install -e "$ROOT/analytics[dev]"
echo "Bootstrap complete. Run $ROOT/scripts/test_all.sh"
