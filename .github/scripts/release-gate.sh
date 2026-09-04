#!/usr/bin/env bash
# Local/CI release gate for the framework module (run from repo root or via bash).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

echo "== go vet =="
go vet ./...

echo "== go test =="
go test ./...

if [[ "${RACE:-0}" == "1" ]]; then
  echo "== go test -race =="
  go test -race ./... -count=1
fi

if [[ "${FUZZ:-0}" == "1" ]]; then
  echo "== fuzz (short) =="
  bash .github/scripts/run-fuzz.sh FuzzRouterPath ./tests/fuzz/ 20s
fi

echo "release gate passed"
