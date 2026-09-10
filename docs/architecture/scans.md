# Independent scans

## First pass

Kernel/contracts/bootstrap, CLI scaffolds, packages catalog, ORM querier, validation FormRequest, auth/authorization, views, testing.TestCase, acquire/enable freeze.

## Second pass (source patterns)

Searched Controller, Handler, Request, DTO, Validate, Service, UseCase, Action, Repository, Model, Entity, Route, Middleware, Guard, Policy, Auth, Session, CSRF, View, HTMX, Resource, Transaction, Job, Queue, Event, Notification, Mail, Storage, AI, Agent, Workflow, Provider.

**Newly incorporated:**

- `packages/bus` — optional command bus; rejected as CRUD default (ADR-0001).
- `packages/jsonapi` + `make:resource` — optional; default JSON (ADR-0008).
- `routing.Controller` / `RouteRegistrar` Get+Post only vs REST verbs (G-M4).
- `make:controller` JSON-only (ADR-0007).
- HTMX: zero hits (ADR-0005).
- `FindOrFail` formatted error vs `sql.ErrNoRows`.
- ORM events are dispatcher names, not struct methods.
- Gate bound in **auth** boot.
- Mail is notification channels.
- `factory` is not `app/database/factories` ORM.
- Tenancy: no first-class application tenancy API found — **out of scope** until a package exists.
- Workflow/orchestration: `agent` / `rag` / `ai` packages; no generic workflow engine in the app tree.

## Third pass — “what could I still decide?”

| Decision | Status |
|---|---|
| Where does StoreRequest live? | STANDARD E — `app/http/requests` |
| Service or not for CRUD? | Skip service for single-model; required for TX |
| Repository? | Optional |
| How to eager load? | Loader funcs, explicit FK |
| Where is TX? | Service |
| HTMX errors? | N/A — not supported |
| JSON:API? | Opt-in |
| Roles? | Not the API — Policy |
| Bus vs service? | Service |
| `app.Router` vs `routing.From` | From |
| Patch vs Update request type | One UpdateRequest |
| Job inside TX? | After commit |

Remaining **product** decisions (not silently invented):

- Doctor architecture: warn vs fail.
- Whether to widen `RouteRegistrar` (kernel API review).
- Whether `exists`/`unique` should fail closed without PresenceChecker (validation package behavior change).
- Official HTMX addon (would supersede ADR-0005).

Those are package/CLI reviews, not an excuse for a second application architecture.

## Quality gate (extraction)

- [x] Framework inspected (kernel, contracts, bootstrap, CLI, tests)
- [x] Packages inspected (ORM deep, auth, validation, queue, notification, filesystem, ai, jsonapi, bus, testing, authorization)
- [x] Generated applications inspected (empty/web/api/full templates)
- [x] Public APIs / generators mapped
- [x] ORM + relationships mapped with NOT SUPPORTED called out
- [x] Requests/validation/controllers/services/repos standardized (PROPOSED where code was silent)
- [x] Conflicts, gaps, matrix, ADRs, examples, enforcement, roadmap
- [x] Laravel similarity audited
- [x] Golden domains tested against the language
- [x] Second and third scans folded in
- [ ] Implementation of generators/doctor — **explicitly out of this phase**
