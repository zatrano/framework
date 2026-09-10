# Enforcement plan

STANDARD defines the architecture. `zatrano doctor` plus framework architecture tests prove the **high-confidence** subset. The rest stays documentation, generators, and application tests.

HTMX is not analyzed (ADR-0005). Repositories are never required (ADR-0003).

---

## Levels

| Level | What it proves |
|---|---|
| COMPILER | types and imports that exist |
| TEST | framework package boundaries (`tests/architecture_test.go`, `contracts/import_test.go`) |
| AST / FILESYSTEM | consumer STANDARD rules via `zatrano doctor` |
| GENERATOR | `zatrano new` empty/web/api/full must doctor-PASS |
| DOCUMENTATION | rules that would false-positive if automated |

Catalog: [rules.md](rules.md) · [rules.yaml](rules.yaml)

---

## CLI (canonical)

One command: **`zatrano doctor`**. Not a second `check` binary.

```text
zatrano doctor [path] [--json] [--strict]
```

- **error** → process exit **1**
- **warning** → exit **0** unless `--strict`
- `--json` → `{rule,severity,file,line,found,why,how,see}`

`package:doctor` remains enablement/graph health. Do not fold STANDARD into `package:install`.

---

## CI (this repository)

1. `go test ./...` — includes doctor fixtures (PASS and FAIL) and generated-app doctor PASS (`TestNew*` / `TestDoctorCleanStarter`).
2. `starter-smoke.sh` — `zatrano new` then `zatrano doctor` (must exit 0).

A consumer CI should run `zatrano doctor` after tests. Architecture errors must fail the job.

---

## What doctor does **not** claim

- Policy/Gate *business* correctness or IDOR
- That a multi-write *must* use a transaction
- CSRF / cookie / CORS configuration completeness
- Fillable lists vs mass-assignment intent
- Relationship API signatures beyond banning `With("name")`
- HTMX

---

## Drift

A new `app/` directory or layer type that doctor does not know is still a STANDARD/ADR change, not a local invention. Add a rule only when false positives are low.
