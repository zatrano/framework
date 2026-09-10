# Enforcement plan

Documentation defines the architecture. Tooling must eventually prove it. Today most application-layer rules are **documentation-only**.

---

## Documentation enforcement (now)

Humans and AI follow:

1. Root [AGENTS.md](../../AGENTS.md)
2. [STANDARD.md](STANDARD.md)
3. [conflicts.md](conflicts.md) — do not copy losers
4. [gaps.md](gaps.md) — do not fill with invention

Generated application `AGENTS.md` (`agents:generate`) does **not** yet include this constitution (G-M8).

---

## Tooling enforcement (existing)

| Mechanism | What it proves |
|---|---|
| Go compiler | types, imports that exist |
| `tests/architecture_test.go` | kernel bans, no framework→packages |
| `tests/consumer_architecture_test.go` | no PackageManager/Apply/Rollback/zatrano.lock |
| Kernel API freeze tests | `contracts.App`, request.go helpers |
| Registry CLI tests | no second resolver in `console` |
| `zatrano doctor` | layout dirs, imported.disabled, concrete contracts, enablement |
| `package:doctor` | package graph / Requires |
| `go vet` / staticcheck | ordinary Go |

Doctor is **warnings**, not a build breaker, and does not encode STANDARD §E–M.

---

## Tooling enforcement (proposed — do not implement until review)

Capability first; command name second.

**Capability: application architecture conformance.**

Checks:

1. Directory allowlist (canonical tree + generated optional dirs). Fail on `domain/`, `dtos/`, `usecase/`.
2. Import graph: controllers must not import `database/sql` for queries when orm is enabled (except `sql.ErrNoRows`, `sql.Tx` in services).
3. `orm.Transaction` only under `app/services` and `app/console`.
4. Mutating handler files (Store/Update) must reference `ValidateForm` when validation is enabled.
5. Web controllers under `controllers/web` must not be the unmodified `make:controller` JSON stub (heuristic).
6. No `With("string")` call pattern on queriers (string eager).
7. `app.Router().Put` banned — must be `routing.From`.
8. unique/exists in request rules ⇒ database enabled.

Delivery options (pick one in implementation phase):

- Extend `zatrano doctor` with an `architecture` group (preferred: one CLI, already known).
- `zatrano doctor --architecture` / stricter exit code in CI.
- Separate analyzer binary later if doctor remains warning-only.

Do **not** assume the flag name. Do **not** fold this into `package:install`.

---

## CI

| Now | After implementation |
|---|---|
| framework architecture tests | + consumer fixture app that must pass doctor architecture |
| packages tests | + optional golden app in examples repo |

Generated apps are not in this module; CI here tests the **framework**. Enforcement for consumers is doctor + their own `go test`.

---

## Drift detection

Drift is a **new pattern appearing without an ADR**.

Signals:

- New top-level `app/` directory not in STANDARD §C
- New generator not listed in STANDARD §Z
- Duplicate request/validation APIs
- ORM API used in a way tests do not cover (string With, nested TX)
- Security: CSRF except list widened, Fillable omitted
- `agents:generate` output diverging from constitution

Process:

1. Detect (doctor / analyzer / code review).
2. Classify: additive (ADR + STANDARD update) vs violation (reject).
3. Version the standard with the framework minor when rules break.

---

## Versioning

- Standard document version: `completeness.yaml` `version`.
- Breaking application rules ship with framework minor + CHANGELOG “Application engineering”.
- Deprecated patterns get an ADR `Superseded by`.
- Scaffolds bump independently via `zatrano new` templates; they must not silently change layering.

Until review acceptance, this standard is **0.1.0 forensic** and is not a compatibility promise.
