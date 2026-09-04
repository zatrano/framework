#!/usr/bin/env bash
# Scaffold a consumer app, blank-import a packages addon, run tests (includes /health).
set -euo pipefail

FRAMEWORK="$(cd "$(dirname "$0")/../.." && pwd)"
PACKAGES="${PACKAGES_DIR:-}"
if [[ -z "$PACKAGES" || ! -d "$PACKAGES" ]]; then
  if [[ -d "$FRAMEWORK/../packages" ]]; then
    PACKAGES="$(cd "$FRAMEWORK/../packages" && pwd)"
  fi
fi
if [[ -z "$PACKAGES" || ! -d "$PACKAGES" ]]; then
  echo "packages checkout not found (set PACKAGES_DIR or clone sibling packages/)"
  exit 1
fi

DEST="$(mktemp -d)/zsmoke"
cd "$FRAMEWORK"
go run ./cmd/zatrano new "$DEST" --module example.com/zsmoke --replace "$FRAMEWORK"

# Nested driver modules are not covered by replace github.com/zatrano/packages => ...
# go mod edit overwrites duplicates if zatrano new already wrote the same replace.
(
  cd "$DEST"
  go mod edit -require github.com/zatrano/packages@v0.0.0
  go mod edit -replace "github.com/zatrano/packages=${PACKAGES}"
  nested=(
    database/driver/sqlite
    database/driver/mysql
    database/driver/pgsql
    database/driver/mssql
    database/driver/oracle
    database/driver/mongo
    mongo
    webauthn
    q
  )
  for rel in "${nested[@]}"; do
    if [[ -f "${PACKAGES}/${rel}/go.mod" ]]; then
      go mod edit -replace "github.com/zatrano/packages/${rel}=${PACKAGES}/${rel}"
    fi
  done
)

python3 - "$DEST/cmd/app/main.go" <<'PY'
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
old_imp = '\t"github.com/zatrano/framework/bootstrap"\n'
new_imp = '\t_ "github.com/zatrano/packages/billing"\n\n\t"github.com/zatrano/framework/bootstrap"\n'
if old_imp not in text:
    raise SystemExit("import block not found")
p.write_text(text.replace(old_imp, new_imp, 1), encoding="utf-8")
PY

cp "$DEST/.env.example" "$DEST/.env"
(cd "$DEST" && go mod tidy)
(cd "$DEST" && go run ./cmd/app key:generate)
(cd "$DEST" && go test ./...)

(cd "$DEST" && go run ./cmd/app serve --port 18080) &
pid=$!
trap 'kill "$pid" 2>/dev/null || true' EXIT
for i in 1 2 3 4 5 6 7 8 9 10; do
  if curl -sf "http://127.0.0.1:18080/health" | grep -q ok; then
    echo "health 200"
    exit 0
  fi
  sleep 1
done
echo "health endpoint did not become ready"
exit 1
