#!/usr/bin/env bash
# Per-package line-coverage gate. Thresholds are policy, not a second test runner.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

failed=0
report=()

check_pkg() {
  local pkg="$1"
  local min="$2"
  local prof
  prof="$(mktemp)"
  if ! go test -coverprofile="$prof" -covermode=count "./${pkg}" >/dev/null; then
    report+=("${pkg}  TEST FAIL  (min ${min})")
    failed=1
    rm -f "$prof"
    return
  fi
  local pct
  pct="$(go tool cover -func="$prof" | awk '/^total:/{gsub(/%/,"",$NF); print $NF}')"
  rm -f "$prof"
  if awk -v p="$pct" -v m="$min" 'BEGIN { exit !(p+0 < m+0) }'; then
    report+=("${pkg}  ${pct}%  (min ${min})")
    failed=1
  fi
}

# Security-adjacent (>=90)
check_pkg kernel/safepath 90
check_pkg kernel/encryption 90
check_pkg kernel/middleware/csrf 90
check_pkg kernel/exceptions 90
check_pkg kernel/cookie 90
check_pkg kernel/container 90
check_pkg kernel/trustedproxy 90

# General kernel (>=80)
check_pkg kernel/log 80
check_pkg kernel/dirs 80
check_pkg kernel/config 80
check_pkg kernel/env 80
check_pkg kernel/http 80
check_pkg kernel/http/useragent 80
check_pkg kernel/routing 80
check_pkg kernel/pipeline 80
check_pkg kernel/report 80
check_pkg kernel/context 80
check_pkg kernel/middleware 80
check_pkg kernel/internal 80
check_pkg kernel/support 80
check_pkg kernel/support/files 80
check_pkg kernel/support/fn 80
check_pkg kernel/support/once 80
check_pkg kernel/support/uuid 80

# CLI / tools (>=60)
check_pkg console 60
check_pkg console/generator 60
check_pkg cmd/zatrano 60
check_pkg distribution/acquire 60
check_pkg distribution/registry 60
check_pkg distribution/manifest 60

if [[ "$failed" -ne 0 ]]; then
  echo "coverage below threshold:"
  printf '  %s\n' "${report[@]}"
  exit 1
fi
echo "coverage thresholds met"
