# Architectural gap report

Classification: CRITICAL / HIGH / MEDIUM / LOW.

No APIs were changed to close these gaps. Each item proposes a canonical decision for review.

---

## CRITICAL

### G-C1 — Application layer is not compiler-enforced — **reduced (Phase 3)**

- **Now:** `zatrano doctor` fails (exit 1) on forbidden directories/types, controller transactions, mixed View/JSON, string eager loads, bare `{Resource}Request`, `validation.Make` in controllers, persist-without-ValidateForm, unique/exists without database. Generated empty/web/api/full apps doctor-PASS. CI runs doctor on scaffold smoke.
- **Remaining:** Go still compiles a second architecture if doctor/CI is skipped. Phase 3 does not make the compiler reject UseCase packages.

### G-C2 — Request taxonomy undefined in generators — **addressed (generator)**

- **Evidence (was):** generic `make:request`; auth used `validation.Make`.
- **Now:** `make:request --store|--update|--index`; `make:auth` writes FormRequests and `ValidateForm`.
- **Remaining:** dashboard stubs still call `validation.Make`; no doctor heuristic yet.

---

## HIGH

### G-H1 — Transaction owner unspecified in code

- **Evidence:** `orm.Transaction` is a free function. Services are empty stubs.
- **Problem:** Controllers or random helpers start TX; nested calls blow up (`NOT SUPPORTED`).
- **Decision:** ADR-0004 — service owns TX; `QueryTx` inside; enqueue jobs after commit.
- **Enforcement:** forbid `orm.Transaction` outside `app/services` and `app/console`.

### G-H2 — Authorization split (Gate vs dashboard RBAC stubs)

- **Evidence:** `packages/authorization` is Gate/Policy. Dashboard stubs contain roles/permissions files not wired as the package API. Gate binds in **auth** boot.
- **Problem:** Controllers may check string roles.
- **Decision:** ADR-0006 — Policy/Gate only. Dashboard stubs are UI, not AuthZ.
- **Enforcement:** doctor: `role ==` in controllers.

### G-H3 — `unique` / `exists` silent pass — **classified (ADR-0010)**

- **Evidence:** `checkPresence` returns true when no PresenceChecker; checker errors fail the rule.
- **Classification:** documented limitation + **correctness** problem. Not a security control. Not authorization.
- **Decision:** apps that use these rules MUST enable `database`. Never use `exists:` as IDOR protection. Runtime unchanged this phase.
- **Enforcement:** Phase 3 doctor: unique/exists in Rules() ⇒ database enabled.

### G-P2 — `authorization.ResponseFor` is JSON-only

- **Evidence:** `gate.go` always returns `http.JSON` 403.
- **Decision:** API uses `ResponseFor`; web uses `http.Abort(403)`. Do not mix.
- **Enforcement:** doctor later; STANDARD §Q now.

### G-H4 — HTMX assumed by some product language, absent in code

- **Evidence:** zero `htmx` / `hx-` matches in framework and packages.
- **Problem:** Agents will invent fragment responses.
- **Decision:** ADR-0005 — HTMX is `NOT SUPPORTED`. Views + Redirect + flash only.
- **Enforcement:** architecture doc; later reject `hx-` helper packages unless an official addon lands.

### G-H5 — `make:controller` always JSON — **addressed**

- **Now:** View when `view` is enabled; HTML on empty; JSON for `--api` and api scaffold web package.

---

## MEDIUM

### G-M1 — Empty `PresetAPI` / `PresetWeb` vs scaffold `EnabledAddons`

- **Evidence:** kernel presets are empty slices; scaffolds list view/health/validation.
- **Problem:** `package:preset` and `zatrano new --web` disagree.
- **Decision:** presets MUST equal scaffold enablement lists (implementation after review).

### G-M2 — Web `addons.go` blank-import gap — **addressed**

- **Now:** web `addons.go.tmpl` blank-imports assets, health, localization, view.

### G-M3 — `contracts.Router` ⊂ typed router

- **Evidence:** App.Router() lacks Put/Patch/Delete/Resource.
- **Decision:** application routes use `routing.From(app)` only.

### G-M4 — `routing.Controller` registrar is Get/Post only

- **Evidence:** `kernel/routing/controller.go`.
- **Decision:** REST write routes on `*Router` groups, not `Controller()`. Optional future: widen `RouteRegistrar` (API change — review).

### G-M5 — jsonapi / resources optional and weakly wired

- **Evidence:** `packages/jsonapi`, `factory` `make:resource`. Not required by API home controller.
- **Decision:** ADR-0008 — default API body is `http.JSON` maps or models. jsonapi is opt-in.

### G-M6 — Query has no `context.Context`

- **Evidence:** ORM querier does not take ctx.
- **Consequence:** request cancellation does not abort SQL.
- **Decision:** `NOT SUPPORTED` today. Do not wrap fake context APIs. Future ORM work is a package ADR, not an app-layer workaround.

### G-M7 — No typed not-found error

- **Evidence:** `sql.ErrNoRows` and formatted `FindOrFail`.
- **Decision:** HTTP maps `sql.ErrNoRows` and FindOrFail to 404. Do not invent `ErrModelNotFound` in apps.

### G-M8 — Generated application `AGENTS.md` is a describe dump — **addressed (preamble)**

- **Now:** `agents:generate` prepends the application constitution. Describe dump remains below. Architecture source of truth is still framework `AGENTS.md` + `docs/architecture`.

### G-M9 — Tenancy package has no application STANDARD

- **Evidence:** `packages/tenancy` exists (`From(app)`, header/domain resolver). Phase 0 scan understated this.
- **Decision:** optional addon. Golden scenarios do not use it. Do not invent `app/tenants` or a tenant layer until a dedicated ADR. Same rule as other packages: `tenancy.From(app)`.
- **Enforcement:** documentation only until a product wants multi-tenant golden coverage.

---

## LOW

### G-L1 — Cursor pagination, nested TX, savepoints

Documented `NOT SUPPORTED`. No app-level substitute.

### G-L2 — EncryptCookies not default-global

Do not assume encrypted cookies.

### G-L3 — Examples live in `github.com/zatrano/examples` (not this workspace)

Canonical examples in this spec are sketches, not that repo.

### G-L4 — `make:resource` registration

Verify command is actually registered before documenting as required CLI.

### G-L5 — Application architecture doctor — **addressed (Phase 3)**

`zatrano doctor` encodes STANDARD high-confidence rules. Exit 1 on errors. Catalog: `docs/architecture/rules.md`. `--fix` is still absent.

### G-L6 — Phase 2 directory audit

Golden scenarios fit STANDARD §C. **No CRITICAL gap** for undocumented directories.

### G-L7 — Disk + DB file replace has no distributed transaction

Documented limitation. Do not invent UnitOfWork. Possible disk orphans.
