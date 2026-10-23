#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
command -v go >/dev/null || { echo "Go is missing from PATH; see docs/DEVELOPMENT.md." >&2; exit 1; }
PYTHON="$ROOT/analytics/.venv/bin/python"
[[ -x "$PYTHON" ]] || { echo "Run scripts/bootstrap.sh first to install the Python package and pytest." >&2; exit 1; }
export GOCACHE="${GOCACHE:-$ROOT/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT/.cache/go-mod}"
export TMPDIR="$ROOT/.cache/tmp"
mkdir -p "$TMPDIR"
status=0
(cd "$ROOT/core" && go test ./...) || status=1
(cd "$ROOT/core" && go vet ./...) || status=1
(cd "$ROOT/analytics" && "$PYTHON" -m pytest) || status=1
(cd "$ROOT" && "$PYTHON" -m unittest discover -s scripts/tests -v) || status=1
exit "$status"
