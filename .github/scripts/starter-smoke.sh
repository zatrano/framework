#!/usr/bin/env bash
# Scaffold a consumer app, blank-import a packages addon, run tests (includes /health).
set -euo pipefail

ROOT="${GITHUB_WORKSPACE:-$(pwd)}"
DEST="$(mktemp -d)/zsmoke"

go run ./cmd/zatrano new "$DEST" --module example.com/zsmoke --replace "$ROOT"

{
  echo ""
  echo "require github.com/zatrano/packages v0.0.0"
  echo "replace github.com/zatrano/packages => $ROOT/_packages"
} >> "$DEST/go.mod"

python3 - "$DEST/cmd/app/main.go" <<'PY'
import pathlib, sys
p = pathlib.Path(sys.argv[1])
text = p.read_text(encoding="utf-8")
old_imp = '\t"github.com/zatrano/framework/bootstrap"\n'
new_imp = '\t_ "github.com/zatrano/packages/billing"\n\n\t"github.com/zatrano/framework/bootstrap"\n'
if old_imp not in text:
    raise SystemExit("import block not found")
text = text.replace(old_imp, new_imp, 1)
old_app = "bootstrap.App(bootstrap.WithProviders(providers.All()...))"
new_app = 'bootstrap.App(bootstrap.WithAddons("billing"), bootstrap.WithProviders(providers.All()...))'
if old_app not in text:
    raise SystemExit("App() call not found")
p.write_text(text.replace(old_app, new_app, 1), encoding="utf-8")
PY

cp "$DEST/.env.example" "$DEST/.env"
(cd "$DEST" && go mod tidy)
(cd "$DEST" && go run ./cmd/app key:generate)
(cd "$DEST" && go test ./...)

# HTTP smoke (same /health the feature test covers, plus a live listener).
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
