# Completeness matrix

Values: `YES` · `NO` · `PARTIAL` · `NOT APPLICABLE` · `NOT SUPPORTED`

Canonical Rule column points at STANDARD sections or ADRs. CLI / static / CI are current reality, not the roadmap.

| Concern | Exists | Canonical Rule | Example | CLI Generator | Static Check | CI Check | Documentation | Test |
|---|---|---|---|---|---|---|---|---|
| Application bootstrap | YES | STANDARD A | scaffold `cmd/app` | `zatrano new` | doctor layout | architecture_test | PACKAGES.md | boot_test |
| Enabled ∩ Imported | YES | STANDARD A | addons.go + enabled.go | package:enable | doctor imported.disabled | YES | PACKAGES.md | YES |
| contracts.App freeze | YES | kernel freeze | — | NOT APPLICABLE | architecture tests | YES | rules | YES |
| Directory layout | YES | STANDARD C | CanonicalConsumerDirs | `zatrano new` | doctor | PARTIAL | STANDARD C | dirs tests |
| Forbidden extra layers | NO | ADR-0001 | — | NO | doctor APP-LAY-* | YES | STANDARD B | YES |
| Controller shape | YES | STANDARD G | HomeController | `make:controller` | doctor APP-CTL-* | YES | STANDARD G | YES |
| Controller JSON-vs-View | YES | ADR-0007 + 0009 | dual controllers | `make:controller` | doctor APP-CTL-003/004 | YES | STANDARD G | YES |
| FormRequest | YES | STANDARD E–F ADR-0002 | validation.FormRequest | `make:request` | doctor APP-REQ-* | YES | STANDARD E | YES |
| unique/exists | PARTIAL | ADR-0010 | PresenceChecker fail-open | NO | doctor APP-VAL-001 | YES | STANDARD F | YES |
| Service layer | YES | STANDARD H ADR-0001 | OrderPlacementService | `make:service` | doctor APP-CTL-005 (TX location) | YES | STANDARD H | YES |
| UseCase/Action/DTO | NOT SUPPORTED | ADR-0001 | — | NO | doctor APP-LAY-003 | YES | STANDARD B | YES |
| String eager load | NOT SUPPORTED | STANDARD L | — | NO | doctor APP-ORM-001 | YES | STANDARD L | YES |
| Transactions | YES | STANDARD M ADR-0004 | orm.Transaction | NO | doctor APP-CTL-005 | YES | STANDARD M | YES |
| Routes web/api | YES | STANDARD N | RegisterWeb | new templates | doctor APP-ROUTE-* | YES | STANDARD N | YES |
| Architecture doctor | YES | enforcement.md | zatrano doctor | doctor | YES | YES | rules.md | YES |
| Web vs API vs full | YES | STANDARD C | templates | `new --web/--api/--full` | NO | scaffold tests | README | YES |
| Index/Filter request | YES | STANDARD E | golden Product/Post | `make:request --index` | NO | NO | YES | YES |
| Nested validation | PARTIAL | STANDARD F | dotted keys | NO | NO | PARTIAL | gaps | PARTIAL |
| Domain Entity/VO | NOT SUPPORTED | STANDARD I | — | NO | NO | NO | STANDARD I | NO |
| Repository | PARTIAL | ADR-0003 | make:repository | `make:repository` | NO | NO | STANDARD J | NO |
| ORM models | YES | STANDARD K | orm.Model | `make:model` | NO | orm tests | STANDARD K | YES |
| ORM query API | YES | STANDARD K | Querier | NOT APPLICABLE | NO | YES | STANDARD K | YES |
| Cursor pagination | NOT SUPPORTED | STANDARD K | — | NO | NO | NO | STANDARD K | NO |
| Relationships | YES | STANDARD L | EagerHasMany | NO | NO | YES | STANDARD L | YES |
| Nested TX | NOT SUPPORTED | STANDARD M | — | NO | NO | NO | STANDARD M | NO |
| Query context.Context | NOT SUPPORTED | G-M6 | — | NO | NO | NO | gaps | NO |
| routing.From vs contracts.Router | YES | conflicts C5 | — | NO | doctor APP-ROUTE-002 | YES | STANDARD N | NO |
| Kernel middleware order | YES | STANDARD O | application.go | NOT APPLICABLE | NO | YES | STANDARD O | YES |
| CSRF | YES | STANDARD O,R | Except /api | generated provider | NO | YES | STANDARD R | YES |
| Authentication | YES | STANDARD P | make:auth | `make:auth` | NO | pkg tests | STANDARD P | YES |
| MFA | YES | STANDARD P | auth twofactor | via auth | NO | pkg tests | STANDARD P | YES |
| API tokens | YES | STANDARD P | apitoken | package:enable | NO | pkg | STANDARD P | YES |
| Gate/Policy | YES | STANDARD Q ADR-0006 | make:policy | `make:policy` | NO | pkg | STANDARD Q | YES |
| Roles/permissions package API | NOT SUPPORTED | ADR-0006 | dashboard stubs only | dashboard | NO | NO | STANDARD Q | NO |
| Views | YES | STANDARD S | http.View | `make:view` | NO | pkg | STANDARD S | PARTIAL |
| HTMX | NOT SUPPORTED | ADR-0005 | — | NO | NO | NO | STANDARD S | NO |
| Validation errors web/API | YES | STANDARD F,T | ResponseFor | NOT APPLICABLE | NO | YES | STANDARD F | YES |
| Find/not-found | PARTIAL | STANDARD T | sql.ErrNoRows | NO | NO | PARTIAL | STANDARD K | YES |
| Events/listeners | YES | STANDARD U | make:event | make:event/listener | NO | PARTIAL | STANDARD U | PARTIAL |
| Jobs/queue | YES | STANDARD U | make:job | make:job | NO | PARTIAL | STANDARD U | PARTIAL |
| Notifications/mail | YES | STANDARD U | Channels mail | make:notification | NO | PARTIAL | STANDARD U | PARTIAL |
| Command bus | YES | ADR-0001 optional | packages/bus | NO | NO | PARTIAL | STANDARD H | PARTIAL |
| Filesystem/uploads | YES | STANDARD V | filesystem | package:enable | NO | PARTIAL | STANDARD V | PARTIAL |
| Image processing | NOT SUPPORTED | STANDARD V | — | NO | NO | NO | STANDARD V | NO |
| Config/env | YES | STANDARD W | .env.example | new + package:enable | doctor | YES | STANDARD W | YES |
| Logging/request id | YES | STANDARD X | RequestID | kernel | NO | YES | STANDARD X | YES |
| Health | YES | STANDARD X | /up + health pkg | new --web/--api | doctor | YES | STANDARD X | YES |
| Metrics/tracing | PARTIAL | STANDARD X | optional MW | package | NO | PARTIAL | STANDARD X | PARTIAL |
| HTTP tests | YES | STANDARD Y | packages/testing | make:test | NO | PARTIAL | STANDARD Y | YES |
| E2E browser tests | NOT SUPPORTED | STANDARD Y | — | NO | NO | NO | STANDARD Y | NO |
| AI constitution | YES | AGENTS.md + golden.md | phase2 | agents:generate | NO | NO | AGENTS.md | NO |
| Completeness matrix | YES | this file | yaml twin | NOT APPLICABLE | NO | NO | YES | NO |

## Phase 3 — Defined / Golden / Generator / Machine / CI

| Concern | Defined | Golden | Generator | Machine | CI |
|---|---|---|---|---|---|
| Controller structure | YES | YES | YES | YES | YES |
| Forbidden layers | YES | YES | YES (absence) | YES | YES |
| FormRequest writes | YES | YES | YES | YES (persist heuristic) | YES |
| Transactions in controllers | YES | YES | YES | YES | YES |
| Web/API mix in one method | YES | YES | YES | YES | YES |
| String eager load | YES | YES | YES | YES | YES |
| unique/exists vs database | YES | YES | n/a | YES | YES |
| Framework ↛ packages | YES | n/a | n/a | YES (test) | YES |
| Authorization semantics | YES | YES | PARTIAL | NO | tests only |
| Repository optional | YES | YES | YES | NO (must not require) | NO |
| HTMX | YES (unsupported) | n/a | n/a | NO (intentionally) | NO |
