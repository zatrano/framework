# AGENTS.md — ZATRANO Application Engineering Constitution

This file is the AI entry point for building **ZATRANO applications**.

It is **not** the architecture specification.

It is **not** the generated `AGENTS.md` that `zatrano agents:generate` writes into an application root (that file is a live describe dump: routing primitives, catalog, doctor checks). Do not confuse the two.

Authoritative specification:

- [`docs/architecture/README.md`](docs/architecture/README.md)
- [`docs/architecture/STANDARD.md`](docs/architecture/STANDARD.md)

Evidence bases: this repository (`github.com/zatrano/framework/v2`), `github.com/zatrano/packages`, generated `zatrano new` output, CLI generators, tests.

Status of this standard: **FROZEN (Phase 4, ADR-0011).** ADRs 0001–0011 accepted. `zatrano doctor` enforces the high-confidence subset. Phase 3.5 records the adversarial boundary. Kernel, contracts, and ORM remain frozen. Application generators must match this constitution, `docs/architecture/STANDARD.md`, and `docs/architecture/golden.md`.

---

## 1. Architectural constitution

1. ZATRANO is a **kernel + packages** platform. The kernel is HTTP, container, config, routing, lifecycle, CLI. Packages bind with `From(app)` / `app.Make`. `contracts.App` does not grow package methods.
2. An application is a **consumer**. Shape: Controller → optional FormRequest → optional application Service → `pkg.From(app)` / `orm.*`.
3. There is **one canonical way**. If two approaches work, the standard already chose one. Do not preserve a second way.
4. Repository evidence is authoritative. Do not copy Laravel, Rails, Nest, Clean Architecture, or generic Go layouts.
5. Do not invent layers that generators and packages do not own: UseCase, Action, DTO, DomainService, Entity-as-DDD, ValueObject, Handler-instead-of-Controller.
6. HTMX is **not** a ZATRANO API. Web rendering is `http.View` + the `view` package. Do not invent fragment/HTMX architecture.
7. Framework must never import `github.com/zatrano/packages`.
8. Enablement is **Enabled ∩ Imported**. Blank-import without enablement does not boot. Enablement without import does not boot.

---

## 2. Mandatory reading order

Before writing application code:

1. This file.
2. [`docs/architecture/STANDARD.md`](docs/architecture/STANDARD.md) — the A–Z language (**authoritative**).
3. [`docs/architecture/golden.md`](docs/architecture/golden.md) — exact files, verbs, Policy API, tests.
4. Neighboring generated/canonical code in the same application (`app/http/controllers`, `app/routes`, `app/providers`).
5. The relevant package public API (`From`, `Register`/`Boot`, tests).
6. [`docs/architecture/examples.md`](docs/architecture/examples.md) for short sketches (`golden.md` wins on conflict).
7. [`docs/architecture/no-second-way.md`](docs/architecture/no-second-way.md) if a second implementation looks tempting.
8. [`docs/architecture/conflicts.md`](docs/architecture/conflicts.md) if two patterns appear in the tree.
9. [`docs/architecture/gaps.md`](docs/architecture/gaps.md) if the feature has no canonical home.

Then use the generator. Then write tests. Then run doctor, tests, and `go vet`.

---

## 3. Canonical rules (short)

| Concern | MUST |
|---|---|
| Directory | `dirs.CanonicalConsumerDirs()` plus generated scaffold dirs. Do not invent `domain/`, `internal/usecase/`, `handlers/`. |
| Routes | `app/routes/web` and `app/routes/api`. Register via `RouteServiceProvider` → `ApplyWeb` / `ApplyAPI`. Use `routing.From(app)` for Put/Patch/Delete/Resource. |
| Controllers | `app/http/controllers/{web,api,admin}`. Methods: `(req *http.Request) *http.Response`. |
| Input (writes) | `validation.FormRequest` in `app/http/requests`. `ValidateForm` then controller. |
| Input (reads) | `{Resource}IndexRequest` when the index accepts any query (`page`, `q`, `sort`, filters). Path-only Show/Destroy: `req.Param` + Policy. |
| Validation | `packages/validation` only. Do not re-validate the same rules in the ORM or service. |
| Authorization | Gate/Policy (`packages/authorization`) **before** data access. Dashboard role stubs are not the API. |
| Persistence | `orm.Query[T]()`, `Find`, `Create`, `With(loader funcs)`. |
| Transactions | `orm.Transaction` inside an application service. Controllers do not start transactions. |
| Services | `app/services` only when more than one model write, an explicit transaction, or reuse from HTTP and console/job. Size of the project is not a reason. |
| Repositories | Optional thin wrappers. Not mandatory. Do not invent interfaces for every model. |
| Responses | Web: `http.View` / `Redirect`. API: `http.JSON`. Do not mix in one method. |
| Packages | `pkg.From(app)`. Never `app.Auth()`. |
| Tests | `tests/` + `packages/testing.TestCase` for HTTP. |

---

## 4. Forbidden behavior

- Invent directories, layers, or parallel implementations.
- Copy a local violation because it already exists (dashboard stubs may call `validation.Make`; `make:auth` may mix View/JSON — do not spread those into resource CRUD).
- Put ORM query chains, rule maps, or Gate definitions in controllers (controllers may *call* Gate and FormRequest).
- Use `bus.Dispatch` as the default application layer (optional package, not the CRUD path).
- Treat `jsonapi` or `make:resource` as required API shape (optional helpers).
- Bypass FormRequest with ad-hoc `map[string]string` for mutating endpoints when `validation` is enabled.
- Nest `orm.Transaction` (not supported).
- Invent cursor pagination, savepoints, or typed `ErrModelNotFound` (not in the ORM).
- Introduce reflection autowiring or constructor injection magic. Construct services with `NewX()` or resolve packages with `From(app)`.
- Enable packages automatically after `package:acquire` unless `--enable` was passed.
- Put Cursor or any AI trailer in git commits.

---

## 5. Architecture lookup

| I need to… | Read |
|---|---|
| Place a file | STANDARD §C · golden.md |
| Name a type | STANDARD §B, §E, §G |
| Validate input | STANDARD §F · ADR-0010 |
| Query / relate / transact | STANDARD §K–M · golden.md |
| Authenticate / authorize | STANDARD §P–Q · ADR-0006 |
| Web vs API | ADR-0009 · two controllers |
| Render HTML | STANDARD §S |
| Upload files | STANDARD §V · golden.md §6 |
| Test | STANDARD §Y · golden.md §17 |
| Generate | STANDARD §Z |
| See if it is missing | `gaps.md` |
| See if two ways exist | `no-second-way.md` · `conflicts.md` |
| Frozen spec | STANDARD.md · ADR-0011 · phase4.md |
| See if CI can prove it | `enforcement.md` · `rules.md` · `phase3.5.md` |

---

## 6. Implementation workflow

1. Identify the golden flow (CRUD, relationship, filter, auth, authz, transaction, file, async, AI).
2. Inspect the relevant canonical example in `golden.md` and neighboring application code.
3. Enable the required packages (`package:enable`). Do not fake APIs that are not enabled.
4. Run the canonical generator (`make:controller`, `make:request`, `make:model`, …).
5. Edit generated files to match STANDARD. Do not leave `Handle() error` stubs as the architecture. Do not invent layers.
6. Wire routes in the matching route file. Add middleware in route groups, not ad-hoc inside controllers.
7. Add tests next to the convention in STANDARD §Y.
8. If a rule is missing, stop and record a gap. Do not invent a second architecture.

---

## 7. Verification workflow

```text
go test ./...
go vet ./...
zatrano doctor
```

Architecture tests in this repository (`tests/architecture_test.go`, `tests/consumer_architecture_test.go`) protect kernel invariants. `zatrano doctor` enforces the high-confidence application STANDARD (exit 1 on errors). Catalog: [`docs/architecture/rules.md`](docs/architecture/rules.md). Semantic gaps (phase 3.5) are not doctor-proven — see [`docs/architecture/phase3.5.md`](docs/architecture/phase3.5.md).

---

## 8. Report conflicts

If neighboring code contradicts STANDARD:

1. Do not copy the violation.
2. Follow STANDARD.
3. Name the conflict (file + pattern vs rule).
4. Point to `docs/architecture/conflicts.md` and `docs/architecture/no-second-way.md`.
