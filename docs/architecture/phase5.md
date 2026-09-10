# Phase 5 — Full Platform Conformance & Release Audit

Date: 2026-09-10

This is a forensic pre-release audit of the **current checkouts**, not a redesign. Architecture remains frozen (ADR-0011). No kernel, contracts, ORM, generator, or doctor heuristic changes were made in this phase.

Evidence bases:

| Tree | Branch | HEAD | Describe | Remote | Working tree |
|---|---|---|---|---|---|
| `github.com/zatrano/framework/v2` | `main` | `4771378` | `v2.1.0-7-g4771378` | ahead of `origin/main` by 6 | clean |
| `github.com/zatrano/packages` | `main` | `288bb2f` | nested-tag describe (root tag `v1.7.1` exists) | ahead of `origin/main` by 1 | **dirty** (unrelated local files; not part of HEAD) |

Go: `go1.25.7 windows/amd64`, module `go 1.25.0`. `CGO_ENABLED=0`, `CC=gcc`, gcc not on PATH.

Published tags (independent lines): framework **v2.1.0**, packages **v1.7.1**. Local main contains unreleased architecture freeze, doctor, Phase 4.5 fail-closed validation, and this audit.

---

## Executive Summary

ZATRANO’s local main is one engineering system: frozen STANDARD, kernel without a packages import, contracts as a dependency-neutral ABI, `From(app)` packages, generators that doctor-PASS, and CI that runs `go test ./...` (including doctor fixtures and generated-app checks), `go vet`, and `go test -race` on Ubuntu. Runtime tests passed on this host. The platform is architecturally coherent. It is **not** yet a tagged public pair that includes doctor + STANDARD freeze + fail-closed `unique`/`exists`. That gap is release process, not a second architecture.

---

## Final Recommendation

**GO WITH CONDITIONS**

No architectural contradiction, ABI break, boundary violation, generator STANDARD violation, or critical runtime failure was found on HEAD. Conditions are objectively verifiable release/hygiene steps before the next public tag. Known semantic/intentional/environmental limitations do not block.

---

## Platform Health

Scores are qualitative, from repository evidence. Not percentages.

| Area | Score | Note |
|---|---|---|
| Architecture | PASS | One canonical consumer path; no-second-way documented |
| Contracts / ABI | PASS | No kernel/packages imports; `App` has no package methods |
| Framework / Packages boundary | PASS | FW-DEP-001/002/003 tests; framework `go.mod` has zero third-party requires |
| Application Standard | PASS | STANDARD frozen; AGENTS.md routes to it |
| Machine Enforcement | PARTIAL | 24 catalog IDs; six Phase 3.5 SEMANTIC bypasses remain by design |
| Generators | PASS | empty/web/api/full doctor-PASS in `console` tests this run |
| Packages | PARTIAL | Enablement model coherent; ecosystem large; published v1.7.1 lacks fail-closed unique/exists; local packages tree dirty |
| Security | PASS | TrustedProxy, HSTS-on-HTTPS, production `TRUSTED_PROXIES=*`, CSRF, Fillable, unique/exists fail-closed on HEAD |
| Runtime Correctness | PASS | `go test ./...` both modules; fail-closed presence checks on packages HEAD |
| Testing | PASS | Unit, architecture, golden compile, doctor fixtures; race CI-enforced |
| Documentation | PARTIAL | STANDARD/ADRs coherent; freeze/4.5 still under CHANGELOG Unreleased while VERSION is 2.1.0 |
| CI | PARTIAL | tests + vet + race + starter-smoke; `doctor --strict` not a separate job; packages CI pins `framework@main` (origin) |
| Upgrade Isolation | PASS | Consumers use public modules + generators; no `internal/` in framework; `zatrano upgrade` does not rewrite app architecture (G-001) |
| Release Hygiene | PARTIAL | Independent version lines are intentional; HEAD ≠ last tag; packages pin `framework v2.0.28` |

---

## Discovery

### Framework

- Module: `github.com/zatrano/framework/v2`
- `VERSION`: `2.1.0` (identity test requires this string)
- `go.mod`: no `require` block (zero third-party runtime dependencies)
- Last tag: `v2.1.0` (`fc9047d`). Seven commits after tag on this HEAD.
- Unreleased on this HEAD: generator STANDARD align is on origin or not (remote is 6 behind HEAD); doctor, freeze, unique/exists docs, Phase 4.5 closure are local-to-ahead.

### Packages

- Module: `github.com/zatrano/packages` (no `/v2` suffix)
- Requires `github.com/zatrano/framework/v2 v2.0.28` with `replace => ../framework`
- Direct requires: `redis`, `x/crypto`, `x/sys`, `modernc.org/sqlite`, framework
- Nested modules: SQL drivers, `mongo`, `webauthn`, `qr` (heavy; separate tags)
- HEAD `288bb2f`: fail-closed unique/exists (not in `v1.7.1`)
- Dirty (not committed, **out of scope** for this audit’s HEAD claims): `ai/publish.go` and peers, `bootutil/cli.go` deletion, provider one-liners, `auth/stubs.go`. Must not ride a packages release.

---

## Documentation consistency

| Claim | Authoritative? | Matches source? |
|---|---|---|
| STANDARD.md is the application language | YES (ADR-0011, AGENTS.md) | YES |
| AGENTS.md is not STANDARD | YES | YES |
| ADRs 0001–0011 Accepted; 0010 fail-closed | YES | YES (`checkPresence` returns false on nil checker / checker err) |
| Framework never imports packages | YES | YES (`tests/architecture_test.go`) |
| Enabled ∩ Imported | YES | YES (bootstrap + enablement tests) |
| unique/exists fail-closed | YES on HEAD / ADR-0010 | YES in packages HEAD; **NO** in published `v1.7.1` |
| Evidence freeze “framework v2.1.0” | PARTIAL | Tag v2.1.0 predates doctor/STANDARD freeze commits |
| Historical phase 2/3 reports still describe fail-open as then-current | YES (history) | YES — do not treat as current |
| `package:preset` equals `--web` enablement | NO — G-M1 | Presets are empty slices (`bootstrap/presets.go`) |
| Mail / HTMX / Outbox / cursor pages | NOT PROVIDED | Confirmed absent as platform APIs |

No undocumented second application architecture was found in generators. Optional packages (`bus`, `jsonapi`, `make:repository`) are named INTENTIONAL ALTERNATIVE in STANDARD / ADR-0001/0003/0008.

---

## Framework ↔ packages boundary

Intended direction:

```text
application → framework (kernel, contracts, bootstrap, CLI)
application → packages (From(app), blank-import)
packages    → framework/v2 (contracts, kernel primitives)
framework   ↛ packages
contracts   ↛ kernel, ↛ packages
```

Evidence:

- `TestFrameworkDoesNotImportPackagesModule`
- `TestContractsDoNotImportFrameworkPackages`
- No `internal/` tree in the framework module
- Packages do not import `github.com/zatrano/framework/v2/internal`
- `HTTPBridge` uses `any` so contracts stay kernel-free (intentional ABI)

No circular Go import. Conceptual coupling is Enablement ∩ catalog names — intended.

---

## Module / dependency classification

### Framework

| Dep | Class |
|---|---|
| (none in `go.mod`) | FIRST-PARTY kernel only |

### Packages (root `go.mod` direct)

| Dep | Class |
|---|---|
| `github.com/zatrano/framework/v2` | FIRST-PARTY |
| `github.com/redis/go-redis/v9` | OPTIONAL RUNTIME (cache/queue redis) |
| `golang.org/x/crypto` | FIRST-PARTY crypto primitives |
| `modernc.org/sqlite` | DEVELOPMENT/TEST + in-tree ORM tests (drivers are nested modules) |
| `golang.org/x/sys` | transitive/support |

No abandoned or architecturally illegal framework dependency. Nested driver modules are intentional (heavy).

---

## Contracts / ABI

`contracts.App` is kernel-complete (container, config, router, logger, context, encrypter, exceptions, reports, bootstrap, HTTP, lifecycle). It does not grow `Auth()` / `Queue()` / `AI()`. Package resolution is `From(app)` / `Make`.

`contracts.Router` is narrower than `kernel/routing` (no Put/Patch/Delete/Resource on the interface). STANDARD requires `routing.From(app)` (APP-ROUTE-002). INTENTIONAL, not drift.

---

## Application architecture

Canonical lifecycle in STANDARD §G–Y, golden.md, and generators matches:

```text
Route → Middleware → FormRequest (Authorize, ValidateForm) → Controller → ORM / From(app) → View | JSON | Redirect
```

Service is optional (§H). Repository is optional concrete only (APP-REP-001 forbids interfaces/generic bases). AuthController View+JSON mix is the documented exception.

Golden scenarios (User, Post, Category, Product, Order, File, Auth, Policy, Notification, AI) are **documentation + placement contracts**, not in-tree demo apps (`zatrano/examples` is the intended consumer home). Compile check: `tests/compatibility`.

---

## Machine enforcement

Catalog: **3** framework tests + **21** doctor IDs (**24** total). Warnings: APP-LAY-004, APP-CON-001, APP-PROV-001, APP-PROV-002. Errors otherwise.

Executed this audit:

| Command | Result |
|---|---|
| `zatrano doctor .` (framework root) | PASS, 1 layout warning (`no app/`) — **not** a framework failure |
| `zatrano doctor . --strict` (framework root) | exit 1 (that warning) — expected |
| `zatrano doctor` generated empty | PASS, 0 warnings |
| `zatrano doctor --strict` generated empty | PASS |
| `go test ./console` | PASS (includes `assertDoctorPass` on empty/web/api/full) |

Phase 3.5 six SEMANTIC stacks remain. Doctor does not prove SQL, IDOR, or Policy business correctness.

---

## Generators

Kernel: `zatrano new` empty/web/api/full, `add:web`/`add:api`, `make:controller`, `make:middleware`, `make:provider`, `make:command`, `make:service`, `make:exception`, `make:test`.

Package CLI (enablement required): `make:request`, `make:rule`, `make:model`, `make:policy`, `make:repository`, `make:view`, `make:auth`, plus other package commands.

This run: `go test ./console` PASS (356s), including doctor on all four scaffolds. `tests/compatibility` PASS. `go test ./console/generator` PASS.

No generator was found that emits UseCase/DTO/`app/usecases` or `Handle() error` as the HTTP architecture.

---

## Package ecosystem

Catalog (`console/catalog.go` + `kernel.Catalog`) lists foundation services, intelligence (`ai` service; `rag`/`agent` libraries), addon services, and libraries. Classification below uses **repository roles**, not invented maturity labels.

| Band | Examples | Status |
|---|---|---|
| Foundation services | session, validation, auth, authorization, hashing, cache, database, orm, view, queue, events, localization, filesystem, notification, health, … | STABLE on the enablement path (`Register`/`Boot`, `From(app)`) |
| Intelligence | `ai` service; `rag`/`agent` import-only libraries | STABLE roles; not App methods |
| Heavy nested | mongo, webauthn, qr, SQL drivers | STABLE as **separate modules** |
| Optional architecture | bus, jsonapi, resources, tenancy | INTENTIONAL ALTERNATIVE — not CRUD default |
| Libraries | collection, testing, totp, pagination, … | Import-only; do not put in `EnabledAddons` |
| `browser` package | Headless helper library | Not a platform E2E product (STANDARD NOT PROVIDED) |
| `redisx` | Library; cache owns Redis | Documented |
| Mail package | — | NOT PROVIDED (notification channels instead) |

HEAD unique/exists: fail-closed. Published `v1.7.1`: still the pre-4.5 checker. Enablement remains Enabled ∩ Imported.

`package:preset` api/web lists are **empty** (G-M1). Scaffolds `--web`/`--api` own enablement. INTENTIONAL limitation until aligned — not a generator STANDARD break (`zatrano new` is the canonical path).

---

## Auth / security (conformance, not a pentest)

Verified present in tests/source:

- Production `TRUSTED_PROXIES=*` fails bootstrap
- Right-to-left / trusted X-Forwarded-For (kernel trustedproxy tests)
- HSTS only when the request is treated as HTTPS
- Production Secure cookies (application HTTP tests)
- CSRF middleware; API except path
- Fillable / mass assignment on models
- Parameterized query builders
- unique/exists fail-closed on packages HEAD (`presence_test.go`, `checkPresence`)
- MFA/TOTP/WebAuthn/apitoken exist as packages — not implied by empty scaffold

`exists:` is still not IDOR protection (ADR-0010). EncryptCookies is opt-in, not default-global.

---

## Runtime verification (this host, 2026-09-10)

```text
go test ./...          PASS   framework and packages
go vet ./...           PASS   framework and packages
go test -race ./...    NOT EXECUTABLE — ENVIRONMENTAL LIMITATION
```

Race evidence: `go test -race ./kernel/middleware` → `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`. CI job `security.yml` / `go-test-race` runs `go test -race ./...` on Ubuntu — **CI ENFORCED**, not local.

Golden/compatibility: `go test ./tests/compatibility` PASS. Architecture tests: `go test ./tests` PASS.

---

## Configuration / lifecycle / observability

- Boot `APP_ENV` snapshot is the environment truth (previously closed).
- Kernel `.env.example` is HTTP/boot keys; package keys live in package `.env.example`.
- Health: empty scaffold registers kernel `/up` and `/health`; `health` package is optional richer checks.
- Lifecycle: `BootstrapContext` / `StartContext`, Start-failure `errors.Join`, Enabled ∩ Imported — covered by runtime tests.
- Inspector/pulse/observability are optional packages, not kernel requirements.

---

## Upgrade isolation

Applications depend on:

- `github.com/zatrano/framework/v2` public API
- `github.com/zatrano/packages` public `From(app)` / enablement
- Generated `app/` tree

They do not depend on a framework `internal/` package. `zatrano upgrade` does not regenerate application architecture (G-001). Risk: consuming unexported kernel types or `app.Router().Put` (doctor + STANDARD). Absolute compatibility is not claimed (pre-3.0 policy).

---

## CI vs local

| Check | Local this audit | CI |
|---|---|---|
| `go test ./...` | PASS | `tests.yml` linux/windows |
| `go vet ./...` | PASS | `static-analysis.yml`, `security.yml` |
| `go test -race` | NOT EXECUTABLE | `security.yml` Ubuntu |
| doctor fixtures / generated PASS | PASS (`./console`, `./tests`) | included in `go test ./...` |
| `doctor --strict` on scaffolds | PASS (manual empty app) | not a dedicated job; `assertDoctorPass` is errors-only |
| starter-smoke (`zatrano new` + doctor + HTTP) | not re-run as bash (Windows); covered by console tests | `starter_smoke` job |
| packages live MySQL/Postgres | not run (no local services) | packages `tests.yml` linux |

CI checks out the **sibling origin `main`**, not this unpushed HEAD. Unreleased doctor/4.5 is therefore **LOCAL + this clone** until push.

---

## Findings matrix

| ID | Area | Finding | Severity | Evidence | Required Action | Release Blocking |
|---|---|---|---|---|---|---|
| P5-H1 | Release | VERSION/tag `2.1.0` while HEAD is `v2.1.0-7` (doctor, STANDARD freeze, 4.5 docs) | HIGH | `VERSION`, `git describe`, CHANGELOG Unreleased | Next public framework tag must bump version/CHANGELOG | NO (process) |
| P5-H2 | Packages | Fail-closed unique/exists is packages HEAD `288bb2f`, not tag `v1.7.1` | HIGH | packages CHANGELOG Unreleased; `checkPresence` | Tag packages after 4.5 if the public story includes fail-closed | NO |
| P5-H3 | Packages clone | Dirty unrelated `publish.go` / `bootutil` / stubs | HIGH | `git status` packages | Do not commit into the 4.5 packages tag | NO |
| P5-M1 | CLI | `PresetAPI`/`PresetWeb` empty vs `--web`/`--api` enablement | MEDIUM | `bootstrap/presets.go`, G-M1 | Align later or document as non-canonical | NO |
| P5-M2 | Pin | packages `go.mod` requires framework `v2.0.28` | MEDIUM | packages README / go.mod | Independent versioning is allowed; bump pin when tagging if desired | NO |
| P5-M3 | CI | `doctor --strict` not a CI job; `assertDoctorPass` ignores warnings | MEDIUM | `console/new_test.go`, `tests.yml` | Optional CI addition | NO |
| P5-M4 | Docs | completeness/README “evidence v2.1.0” predates freeze commits | MEDIUM | `docs/architecture/README.md` | Update evidence line at next tag | NO |
| P5-L1 | Docs | Phase 2/3 reports still say fail-open as then-current | LOW | historical reports | Keep as history | NO |
| P5-L2 | API | `contracts.Router` ⊂ typed router | LOW | contracts vs routing | INTENTIONAL (APP-ROUTE-002) | NO |

---

## Conformance matrix

| Concern | Standard | Implementation | Generator | Doctor | Tests | CI | Docs | Status |
|---|---|---|---|---|---|---|---|---|
| Kernel ↛ packages | YES | YES | n/a | FW-DEP-001 | YES | YES | YES | PASS |
| Contracts ABI | YES | YES | n/a | FW-DEP-002/003 | YES | YES | YES | PASS |
| Consumer layout | YES | YES | `zatrano new` | APP-LAY-* | YES | YES | YES | PASS |
| FormRequest writes | YES | YES | `make:request` | APP-REQ-* | YES | YES | YES | PASS |
| Web vs API | YES | YES | `--web/--api` | APP-CTL-003/004 | YES | YES | YES | PASS |
| TX ownership | YES | YES | n/a | APP-CTL-005 | YES | YES | YES | PARTIAL (file-level) |
| unique/exists | YES | HEAD fail-closed | n/a | APP-VAL-001 structural | YES | packages tests | YES | PASS on HEAD |
| AuthZ Policy | YES | package | `make:policy` | none (SEMANTIC) | package | package | YES | SEMANTIC |
| HTMX | NOT PROVIDED | absent | none | none | n/a | n/a | YES | INTENTIONAL |
| Nested TX | NOT PROVIDED | no savepoints | none | none | docs | n/a | YES | INTENTIONAL |
| Query ctx | NOT PROVIDED | no ctx API | none | none | n/a | n/a | YES | INTENTIONAL |
| Semantic doctor bypasses | documented | possible | n/a | 3.5 | adversarial tests | YES | YES | SEMANTIC |

---

## Package matrix (catalog bands)

Per-package essays for ~80 trees would invent completeness. Status is by **role evidence** (catalog Kind, Register/Boot, tests in `go test ./...`).

| Package | Purpose | Public API | Tests | Docs | Enablement | Standard | Security | Status |
|---|---|---|---|---|---|---|---|---|
| session, auth, validation, authorization, hashing, database, orm, view, cache, queue, events, filesystem, localization, notification, health, … | Foundation capabilities | `From(app)` | in `go test ./...` | PACKAGES.md | Enabled ∩ Imported | matches | fail-closed secrets/redis where specified | STABLE |
| ai | Chat providers | `ai.From` | yes | PACKAGES.md | service | no App.AI() | log_prompts opt-in | STABLE |
| rag, agent | Intelligence libraries | import-only | yes | CHANGELOG 1.7.0 | **not** EnabledAddons | yes | — | STABLE library |
| redisx | Redis helper | library | yes | yes | cache owns conn | yes | — | STABLE library |
| mongo, webauthn, qr, drivers | Heavy | nested modules | nested CI | PACKAGES.md | enable/`db:setup` | yes | separate go.mod | STABLE nested |
| bus, jsonapi, resources, tenancy | Optional | package API | yes | STANDARD names them optional | enable if used | INTENTIONAL ALTERNATIVE | — | STABLE optional |
| testing | HTTP TestCase | library | yes | STANDARD §Y | import | yes | — | STABLE |
| browser | Headless helper | library | package tests | not platform E2E | import | NOT PROVIDED as E2E | — | library ≠ product |
| validation unique/exists | DB-backed rules | PresenceChecker | presence_test.go | ADR-0010 | database binds checker | fail-closed HEAD | fail-closed | HEAD STABLE; v1.7.1 stale |

---

## Release Blockers

No release blockers identified.

---

## Conditions

For the **next public release pair** (not for using local HEAD in development):

1. **Framework tag:** bump `VERSION`, `packages/version` fallback if shipped together, README badge, and CHANGELOG Unreleased → a new `vX.Y.Z` so doctor + STANDARD freeze are not silently hanging off the already-published `v2.1.0` tag.
2. **Packages tag:** publish fail-closed unique/exists (`288bb2f` or successor) so public `unique`/`exists` matches ADR-0010. Do **not** include the dirty `publish.go` / `bootutil` WIP.
3. **Evidence lines:** at tag time, set architecture README / completeness.yaml evidence to the new versions (today they still say v2.1.0 / v1.7.1).
4. **Push/CI:** origin `main` does not yet contain this HEAD; CI on GitHub cannot validate unpushed commits. Push is out of scope for this phase but is required for CI to match the audit clone.

These are verifiable. They do not require architecture changes.

---

## Intentional Limitations

Confirmed still true in source:

- ORM query has no `context.Context` (G-M6)
- Nested transactions / savepoints not supported (STANDARD; `orm.Transaction` has no nest guard beyond SQL)
- Jobs-after-commit is SEMANTIC (no outbox)
- Disk + DB file replace is not a distributed TX
- Browser E2E is not a platform product
- EncryptCookies is not default-global
- `package:preset` empty vs scaffold enablement (G-M1)
- unique ignore-ID / extra WHERE not implemented (CSV extras ignored; not fail-open)
- Concatenated `"uni"+"que:"` bypasses APP-VAL-001
- Six Phase 3.5 semantic doctor-PASS stacks
- HTMX, mail package, cursor pagination, typed `ErrModelNotFound` — NOT PROVIDED
- `contracts.Router` narrower than typed router
- Race testing **locally** NOT EXECUTABLE (GCC/CGO); CI Ubuntu still runs `-race`

---

## Historical items (verified, not auto-fixed)

| Item | Current state |
|---|---|
| unique ignore-ID | Unimplemented; extra CSV ignored |
| concat unique doctor bypass | SEMANTIC, unchanged |
| ORM ctx | Still absent |
| nested TX | Still unsupported |
| jobs-after-commit | SEMANTIC |
| outbox | NOT PROVIDED |
| disk+DB TX | Documented limitation |
| browser E2E | NOT PROVIDED |
| EncryptCookies default | Still opt-in |
| preset vs scaffold | Still empty presets |

---

## Decision logic

- No fundamental architectural defect.
- Contracts and FW↛packages hold under test.
- Generators produce doctor-clean canonical apps.
- Security hardening tests remain green.
- Runtime `go test ./...` / `go vet ./...` PASS on both modules.
- Documentation is accurate for HEAD; public tags lag HEAD.

Therefore not NO-GO. Not unconditional GO for “ship this tree as v2.1.0”. **GO WITH CONDITIONS** for the next coordinated tag.

Phase 5 does not start an examples-repo implementation and does not unfreeze the kernel.
