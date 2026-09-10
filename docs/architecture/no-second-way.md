# No second way

Phase 4 audit: for every major application concern, one canonical path. Alternatives are **intentional**, **forbidden**, or **semantic** (doctor may not see them). Target: **zero undocumented architectural ambiguity** — not zero theoretical bypasses.

Authoritative spec: [STANDARD.md](STANDARD.md). Doctor boundary: [phase3.5.md](phase3.5.md).

Status values:

| Status | Meaning |
|---|---|
| CANONICAL | The one way |
| INTENTIONAL ALTERNATIVE | STANDARD names when it is valid |
| FORBIDDEN | Non-compliant even if it compiles |
| SEMANTIC | FORBIDDEN as architecture; doctor may PASS |
| NOT PROVIDED | Do not invent |

---

## Application concerns

| Concern | Canonical way | Alternative | Status | Reason | Enforcement |
|---|---|---|---|---|---|
| Project tree | `dirs.CanonicalConsumerDirs()` + scaffold dirs in STANDARD §C.1 | Laravel `app/Http`, `internal/usecase`, Clean `app/application` | FORBIDDEN | ADR-0001 | APP-LAY-001/005 |
| Extra CA dirs with unbanned names | Do not use as layers | `app/core`, `workflows`, `processors`, `orchestration` | SEMANTIC | False-positive risk | Docs; doctor limitation |
| Routes | `app/routes/{web,api}` + `RegisterWeb`/`ApplyWeb` | `app/http/routes`, `app/router`, service `router.Get` | FORBIDDEN | STANDARD §N | APP-ROUTE-001 |
| REST verbs | `routing.From(app)` | `app.Router().Put` | FORBIDDEN | contracts.Router is narrow | APP-ROUTE-002 |
| Assigned router | `routing.From(app).Put` | `r := app.Router(); r.Put` | SEMANTIC | AST shape | phase 3.5 |
| Middleware | Kernel order + `app/http/middleware` on groups | Ad-hoc in controllers | FORBIDDEN | STANDARD §O | SEMANTIC / review |
| Controller placement | `controllers/{web,api,admin}` `*Controller` | `app/services/PostController`, HTTP `*Handler` | FORBIDDEN | §G | APP-CTL-001 |
| Controller signature | `(req *http.Request) *http.Response` | `Handle() error` as HTTP | FORBIDDEN | generators | APP-CTL-002 |
| Web response | View / Redirect | JSON on resource HTML CRUD | SEMANTIC (scaffold JSON home is intentional for `--api` web package) | ADR-0009 | APP-CTL-004 does not forbid web JSON |
| API response | JSON | `http.View` | FORBIDDEN | ADR-0009 | APP-CTL-004 |
| View+JSON mix | Never in one resource method | Mix in helper / other package | SEMANTIC | Same-function AST | APP-CTL-003 + limitation |
| Auth mix | `AuthController` / `auth_controller.go` only | `author_controller.go` | FORBIDDEN | doctor exact names | APP-CTL-003 |
| FormRequest writes | `{Resource}Store/UpdateRequest` + `ValidateForm` | `PostRequest`, `CreatePostRequest`, `PostForm`, `Make` | FORBIDDEN | ADR-0002 | APP-REQ-001/002 |
| Index filters | `{Resource}IndexRequest` | ad-hoc `req.Query` without request type when query exists | FORBIDDEN (SEMANTIC if no Rules()) | §E | Docs; APP-REQ-001 if misnamed Rules() |
| Show/Destroy input | `req.Param` + Policy | `DeleteRequest` / DTO | FORBIDDEN unless bulk body | §E | Docs |
| Validation package | `packages/validation` only | ORM/service re-validation of the same rules | FORBIDDEN | §F | APP-REQ-002 in service/model |
| unique/exists | Literal rules + `database` enabled | Concatenated strings; no database | Structural vs SEMANTIC | ADR-0010 | APP-VAL-001 literals only |
| Authorization | Gate/Policy | `role ==` dashboard stubs | FORBIDDEN as AuthZ | ADR-0006 | SEMANTIC (no doctor role scan) |
| Persistence | `orm.Query[T]()` | Concrete repository | INTENTIONAL ALTERNATIVE | ADR-0003 optional | APP-REP-001 forbids interfaces/generic |
| Repository interfaces | Do not create | `PostRepository interface`, `BaseRepository` | FORBIDDEN | Phase 3.5 | APP-REP-001 |
| Service | Only when §H MUST | Always / UseCase / Handler.Handle | MUST NOT / SEMANTIC | ADR-0001 | APP-LAY-003 suffixes; Handle() SEMANTIC |
| Transactions | Service + `orm.Transaction` + `QueryTx` | Controller file TX | FORBIDDEN | ADR-0004 | APP-CTL-005 |
| Cross-package TX | Service owns TX | `txutil` from controller | SEMANTIC | Whole-program | phase 3.5 |
| Nested TX | Not supported | Savepoints / inner Transaction | NOT PROVIDED | ORM | Docs |
| Relationships | Typed `With(loader)` + explicit FK | `With("comments")` | FORBIDDEN | §L | APP-ORM-001 (call chain) |
| Assigned `q.With("…")` | Typed loader on the chain | Variable then string With | SEMANTIC | AST | phase 3.5 |
| Related create | `orm.Create` + FK | Association `Create` | NOT PROVIDED | ORM has no assoc Create | Docs |
| Views | `http.View` + view package | HTMX fragments | NOT PROVIDED | ADR-0005 | Docs |
| JSON body | `http.JSON` | jsonapi / Resource required | INTENTIONAL ALTERNATIVE (opt-in) | ADR-0008 | Docs |
| Auth | `auth.From(app)` + `make:auth` | JWT invented in app; `app.Auth()` | FORBIDDEN | Kernel freeze | FW tests / docs |
| MFA | Generated auth APIs | Second MFA package in app | FORBIDDEN | §P | Docs |
| Files | `filesystem.From(app)` + UploadRequest | Raw `os` paths | FORBIDDEN | §V | SEMANTIC |
| Notifications | `notification.From(app)` after commit | Mail package / SMTP in app | NOT PROVIDED mail pkg | §U | Docs |
| Events/listeners | `make:event` / listener | Workflow engine dir | FORBIDDEN as layer | §U / §C.1 | APP-LAY if forbidden dir |
| Jobs/queue | `make:job` + queue package after commit | Dispatch inside TX | SEMANTIC correctness | ADR-0004 | Docs |
| Config | `.env` + `app.Config()`; bootstrapped `APP_ENV` | Live re-parse APP_ENV for security | FORBIDDEN | kernel snapshot | kernel tests |
| Logging / request ID | Kernel logger + RequestID MW | Ad-hoc ids | CANONICAL | §X | Runtime |
| Errors | Exception MW + Abort / ResponseFor | DomainError hierarchy | FORBIDDEN | §T | Docs |
| Pagination | `Paginate` / `SimplePaginate` | Cursor page API | NOT PROVIDED | ORM | Docs |
| Search/sort/filter | IndexRequest + Query Where | Second query DSL | FORBIDDEN | §E | Docs |
| Testing | `tests/` + `packages/testing.TestCase` | Browser E2E framework | NOT PROVIDED E2E | §Y | Docs |
| Factories | `packages/factory` | Invent fixtures framework | INTENTIONAL (factory package) | §Y | Docs |
| AI | `ai.From(app)` | `App.AI()`, AIService | FORBIDDEN / NOT PROVIDED | §U | Kernel freeze |
| Packages | `pkg.From(app)` + Enabled ∩ Imported | Blank-import only | FORBIDDEN | STANDARD A | package doctor |
| Framework ↛ packages | Never import packages module | Alias/dot import | FORBIDDEN | kernel freeze | FW-DEP-001 |
| Bootstrap | `bootstrap.App` | Custom kernel boot | FORBIDDEN | scaffold | Docs |
| Observability | Kernel logs, RequestID, optional metrics | Second APM architecture | INTENTIONAL optional MW | §X | Docs |
| Tenancy | `tenancy.From(app)` if enabled | `app/tenants` layer | NOT PROVIDED as app layer | G-M9 | Docs |
| Deployment | `APP_ENV=production`, secrets, `APP_DEBUG=false` | Debug in production | SEMANTIC / deploy cmd | §W | `deploy` warn; secrets fail-closed |

**Undocumented ambiguity remaining:** **0** (every row is classified).

**Intentional alternatives:** optional concrete repository; optional jsonapi/resources; optional bus; optional tenancy package; `--api` web home JSON; `make:auth` View+JSON; service vs controller when §H says so.

**Semantic bypasses:** the six phase 3.5 stacks plus assigned router/With, unused FormRequest, dynamic unique strings, role-string AuthZ, jobs inside TX.

---

## Phase 3.5 six stacks (preserved)

| Bypass | Why doctor can PASS | Static/semantic | Should doctor catch it? | Canonical prevention |
|---|---|---|---|---|
| 1 Unbanned CA directories | Names not in APP-LAY-001 | SEMANTIC | **No** (false positives) | STANDARD §C.1 — not a layer |
| 2 UseCase-shaped types in `app/services` | No suffix / no HTTP signature | SEMANTIC | **No** (Handler/Action names are legal business words) | Verb methods; no `Handle()` |
| 3 Ownership hiding | No whole-program attribution | SEMANTIC | **No** | TX in service; mix in the method; Make on FormRequest |
| 4 Transport collapse | No route graph; web JSON allowed for scaffold | SEMANTIC + scaffold exception | **No** for wiring; APP-CTL-004 only API View | ADR-0009 two controllers |
| 5 FormRequest theater | Unused types; concat strings; `internal` wrappers | SEMANTIC | Only same-fn persist (APP-REQ-003) and literal unique | ValidateForm; enable database |
| 6 DDD-lite in legal folders | Entity names and concrete repos allowed | SEMANTIC | Interfaces/generic FAIL (APP-REP-001) | ORM models; optional repo only as seam |

Do not add AST heuristics to force this table to zero.
