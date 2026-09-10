# Completeness matrix

Frozen with [STANDARD.md](STANDARD.md) (ADR-0011). Machine twin: [completeness.yaml](completeness.yaml).

Values: `YES` · `NO` · `PARTIAL` · `n/a`

Do not mark **CI / doctor** as YES if the rule is documentation-only.

**Inventory:** 57 concerns defined · 48 with a canonical example · 30 generator-supported · 23 doctor/FW-static · 28 CI-enforced · 30 semantic-only · 12 not provided.

| Concern | Defined | Canonical example | Forbidden example | Generator | Doctor | CI | Runtime | Semantic only | Not provided |
|---|---|---|---|---|---|---|---|---|---|
| Bootstrap | YES | `bootstrap.App` | custom kernel boot | `zatrano new` | APP-LAY-004 | YES | YES | NO | NO |
| Enablement | YES | Enabled ∩ Imported | blank-import only | `package:enable` | — | YES | YES | NO | NO |
| Directory layout | YES | §C.1 | `app/usecases` | `zatrano new` | APP-LAY-001/004/005 | YES | NO | NO | NO |
| Extra layers | YES | absence | UseCase | none | APP-LAY-* | YES | NO | PARTIAL | NO |
| Routes | YES | `app/routes/web` | routes in services | `zatrano new` | APP-ROUTE-001 | YES | NO | NO | NO |
| REST verbs | YES | `routing.From` | `app.Router().Put` | none | APP-ROUTE-002 | YES | NO | PARTIAL | NO |
| Middleware | YES | kernel + `make:middleware` | authz-in-controller | `make:middleware` | — | PARTIAL | YES | YES | NO |
| Controllers | YES | `*Controller` | HTTP `*Handler` | `make:controller` | APP-CTL-001/002 | YES | NO | NO | NO |
| Web controllers | YES | View/Redirect | JSON as HTML CRUD | `make:controller` | APP-CTL-003 | YES | NO | PARTIAL | NO |
| API controllers | YES | JSON | View | `--api` | APP-CTL-004 | YES | NO | NO | NO |
| FormRequest | YES | Store/Update | `PostForm`, `Make` | `make:request` | APP-REQ-* | YES | NO | PARTIAL | NO |
| Request taxonomy | YES | `--store/--index` | `CreatePostRequest` | `make:request` | APP-REQ-001 | YES | NO | NO | NO |
| Validation | YES | `ValidateForm` | rules in ORM | `make:rule` | APP-REQ-002 | YES | YES | PARTIAL | NO |
| unique/exists | YES | + database | exists as IDOR; unique without completing lookup | none | APP-VAL-001 | YES | YES | PARTIAL | NO |
| Authorization | YES | Gate/Policy | `role ==` | `make:policy` | — | PARTIAL | YES | YES | NO |
| Models | YES | `orm.Model` | Entity layer | `make:model` | — | YES | YES | YES | NO |
| ORM access | YES | `Query[T]()` | sql in controller | none | — | YES | YES | YES | NO |
| Relationships | YES | typed `With` | `With("comments")` | none | APP-ORM-001 | YES | YES | PARTIAL | NO |
| Repositories | YES | optional concrete | interface / BaseRepository | `make:repository` | APP-REP-001 | YES | NO | PARTIAL | NO |
| Services | YES | §H table | ritual / UseCase | `make:service` | APP-LAY-003 | YES | NO | PARTIAL | NO |
| Transactions | YES | service TX | controller-file TX | none | APP-CTL-005 | YES | YES | PARTIAL | NO |
| Views | YES | `http.View` | fragment-view API | `make:view` | APP-CTL-004 | PARTIAL | YES | PARTIAL | NO |
| Auth | YES | `make:auth` | `app.Auth()` | `make:auth` | AUTH exception | PARTIAL | YES | PARTIAL | NO |
| Files | YES | golden File | unsanitized paths | none | — | PARTIAL | YES | YES | NO |
| Notifications / jobs | YES | From(app) after commit | mail package; TX dispatch | `make:notification` / `make:job` | — | PARTIAL | YES | YES | NO |
| Config / env | YES | boot `APP_ENV` snapshot | live re-parse for security | `zatrano new` | APP-PROV-002 | YES | YES | PARTIAL | NO |
| Testing | YES | `testing.TestCase` | E2E as platform | `make:test` | — | PARTIAL | NO | YES | NO |
| AI | YES | `ai.From(app)` | `App.AI()` | none | — | PARTIAL | YES | YES | NO |
| Package From(app) | YES | `pkg.From` | App package methods | `package:enable` | APP-CON-001 | YES | YES | PARTIAL | NO |
| FW ↛ packages | YES | architecture tests | packages import | none | FW-DEP-* | YES | NO | NO | NO |
| Fragments | YES | — | fragment architecture | none | — | NO | NO | YES | **YES** |
| Browser E2E | YES | — | browser E2E as platform | none | — | NO | NO | YES | **YES** |
| Mail package | YES | notification channel | `packages/mail` | none | — | NO | NO | YES | **YES** |
| Outbox / UoW | YES | — | Outbox types | none | — | NO | NO | YES | **YES** |
| Cursor pages | YES | — | keyset API | none | — | NO | NO | YES | **YES** |
| Nested TX | YES | — | nested Transaction | none | — | NO | YES | NO | **YES** |
| Query context | YES | — | fake ctx API | none | — | NO | NO | YES | **YES** |
| Typed not-found | YES | `sql.ErrNoRows` | `ErrModelNotFound` | none | — | NO | NO | YES | **YES** |
| Tenancy app layer | YES | `tenancy.From` | `app/tenants` | none | — | NO | YES | YES | **YES** |
| Reflection DI | YES | `NewX()` | autowire | none | — | NO | YES | YES | **YES** |
| RBAC package API | YES | — | dashboard as Gate | dashboard stubs | — | NO | NO | YES | **YES** |

Doctor-boundary six SEMANTIC stacks: [doctor-boundary.md](doctor-boundary.md) · [no-second-way.md](no-second-way.md).
