#!/usr/bin/env bash
# altool can report a validation rejection while returning zero. Require its
# positive upload receipt as well as a successful process status.
set -euo pipefail
[[ $# -eq 1 && -f "$1" ]] || { echo 'Usage: upload_testflight.sh app.ipa' >&2; exit 1; }
: "${API_ID:?Missing App Store Connect Key ID}"
: "${API_ISSUER:?Missing App Store Connect Issuer ID}"
: "${API_PRIVATE_KEYS_DIR:?Missing App Store Connect private-key directory}"
upload_log=$(mktemp "${TMPDIR:-/tmp}/vfr-testflight.XXXXXX")
trap 'rm -f "$upload_log"' EXIT
status=0
xcrun altool --upload-app --type ios --file "$1" \
  --apiKey "$API_ID" --apiIssuer "$API_ISSUER" 2>&1 | tee "$upload_log" || status=$?
if [[ "$status" -ne 0 ]]; then
  echo 'TestFlight upload command failed.' >&2
  exit "$status"
fi
if grep -Eq 'UPLOAD FAILED|ERROR:' "$upload_log"; then
  echo 'Apple rejected the upload despite a zero process exit status.' >&2
  exit 1
fi
if ! grep -Eq '^[[:space:]]*UPLOAD SUCCEEDED([[:space:]]|$)|No errors uploading' "$upload_log"; then
  echo 'No affirmative Apple upload receipt was found; upload is unconfirmed.' >&2
  exit 1
fi
