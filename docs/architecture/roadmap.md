# Implementation roadmap

**Boundary:** this extraction did not rewrite architecture, refactor apps, change public APIs, move directories, commit, tag, or push.

Work starts only after review of STANDARD + ADRs.

---

## Phase 0 — Review

- ADRs 0001–0008 **Accepted**.
- HTMX remains out of scope.
- Repository optional / FormRequest mandatory for writes.

## Phase 1 — Generators match STANDARD (this change)

- `make:controller`: web → View when `view` enabled; empty → HTML; `--api` / api scaffold → JSON.
- `make:service`: constructor only; no canonical `Handle()`.
- `make:request`: `--store` / `--update` / `--index` suffixes.
- Web `addons.go.tmpl` blank-imports default-enabled addons.
- `agents:generate` prepends the application constitution.
- `make:model` emits `Fillable()`.
- `make:auth` emits FormRequests + `ValidateForm`.

Still open: `PresetWeb` / `PresetAPI` (kernel/bootstrap — not this phase).

## Phase 2 — AI surface

- Constitution is prepended by `agents:generate` (done in Phase 1).
- Describe dump remains below the constitution.

## Phase 3 — Doctor architecture group

- Allowlist directories.
- Transaction location.
- ValidateForm on Store/Update when validation enabled.
- unique/exists ⇒ database enabled.

Exit code policy: warn vs fail — product decision.

## Phase 4 — Do **not** do

- Nested transactions, cursor pagination, typed ErrModelNotFound, query context — those are **ORM package** ADRs, not app-standard patches.
- `func Apply` / acquire redesign / `contracts.App` growth / HTMX kernel helpers.
- Mandatory repositories or UseCase generators.
- Automatic enable after acquire.

## Phase 5 — Examples repo (optional)

- One golden app per flow in `github.com/zatrano/examples`, built **only** with accepted STANDARD.
- Not a second architecture.

---

## Suggested review order

1. ADR-0001 layers  
2. ADR-0002 FormRequest  
3. ADR-0003 repositories  
4. ADR-0004 transactions  
5. ADR-0005 HTMX  
6. ADR-0006 authorization  
7. ADR-0007 controller generator  
8. ADR-0008 JSON vs jsonapi
