# Architectural gap report

Classification: CRITICAL / HIGH / MEDIUM / LOW.

No APIs were changed to close these gaps. Each item proposes a canonical decision for review.

---

## CRITICAL

### G-C1 — Application layer is not compiler-enforced

- **Evidence:** No architecture test forbids controller transactions or extra layers. Generators no longer emit `Handle()` or JSON web stubs.
- **Current:** A competent engineer can still put queries in controllers, or invent UseCases, and still compile.
- **Problem:** Two ZATRANO apps can still diverge until doctor architecture lands (Phase 3).
- **Current:** A competent engineer can put queries in controllers, or invent UseCases, and still compile.
- **Problem:** Two ZATRANO apps will not look the same.
- **Impact:** AI and humans diverge; STANDARD is documentation-only.
- **Decision:** STANDARD §G–J (Controller → optional Service → ORM). Repositories optional.
- **Enforcement:** AST checks (roadmap): ban `orm.Transaction` in `app/http/controllers`; ban new `usecase`/`dto` directories.

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

### G-H3 — `unique` / `exists` silent pass

- **Evidence:** validation PresenceChecker; without database the rule passes.
- **Problem:** Duplicate rows and IDOR-looking ids validate as OK.
- **Decision:** applications that use these rules MUST enable `database`. Doctor warns when rules contain unique/exists but database is not enabled.
- **Enforcement:** `package:doctor` / validation boot check.

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

### G-M8 — Generated application `AGENTS.md` is a describe dump

- **Evidence:** `console/agents.go` — “do not edit by hand”.
- **Problem:** Application AIs never see this STANDARD.
- **Decision:** `agents:generate` should prepend a pointer to the framework STANDARD (or embed constitution). Roadmap.

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

### G-L5 — No `zatrano check` / architecture doctor for app layer

Capability defined in `enforcement.md`. Name TBD.
