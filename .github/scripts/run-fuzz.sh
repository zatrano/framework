#!/usr/bin/env bash
# Run a Go fuzz target for a bounded time.
# Ignores the known coordinator race (https://github.com/golang/go/issues/75804)
# where -fuzztime expiry is reported as "context deadline exceeded" with no crash.
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <FuzzTarget> <package> [fuzztime]" >&2
  exit 2
fi

target="$1"
pkg="$2"
fuzztime="${3:-3m}"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

set +e
go test "$pkg" -run='^$' -fuzz="$target" -fuzztime="$fuzztime" 2>&1 | tee "$tmp"
code="${PIPESTATUS[0]}"
set -e

if [[ "$code" -eq 0 ]]; then
  exit 0
fi

if grep -qE '[[:space:]]+[^[:space:]]+_test\.go:[0-9]+:' "$tmp"; then
  exit "$code"
fi
if grep -qiE 'fuzzing found a failing input|minimizing' "$tmp"; then
  exit "$code"
fi
if ! grep -q 'now fuzzing with' "$tmp"; then
  exit "$code"
fi

fails="$(grep -cE '^--- FAIL:' "$tmp" || true)"
if [[ "$fails" -ne 1 ]]; then
  exit "$code"
fi
if grep -qE "^--- FAIL: ${target} " "$tmp" && grep -qE '^[[:space:]]+context deadline exceeded[[:space:]]*$' "$tmp"; then
  echo "Ignoring Go fuzztime coordinator race (golang/go#75804): no crash, only context deadline exceeded."
  exit 0
fi

exit "$code"
