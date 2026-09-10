# Completeness matrix

Values: `YES` · `NO` · `PARTIAL` · `NOT APPLICABLE` · `NOT SUPPORTED`

Canonical Rule column points at STANDARD sections or ADRs. CLI / static / CI are current reality, not the roadmap.

| Concern | Exists | Canonical Rule | Example | CLI Generator | Static Check | CI Check | Documentation | Test |
|---|---|---|---|---|---|---|---|---|
| Application bootstrap | YES | STANDARD A | scaffold `cmd/app` | `zatrano new` | doctor layout | architecture_test | PACKAGES.md | boot_test |
| Enabled ∩ Imported | YES | STANDARD A | addons.go + enabled.go | package:enable | doctor imported.disabled | YES | PACKAGES.md | YES |
| contracts.App freeze | YES | kernel freeze | — | NOT APPLICABLE | architecture tests | YES | rules | YES |
| Directory layout | YES | STANDARD C | CanonicalConsumerDirs | `zatrano new` | doctor | PARTIAL | STANDARD C | dirs tests |
| Forbidden extra layers | NO | ADR-0001 | — | NO | NO | NO | STANDARD B | NO |
| Web vs API vs full | YES | STANDARD C | templates | `new --web/--api/--full` | NO | scaffold tests | README | YES |
| Controller shape | YES | STANDARD G | HomeController | `make:controller` | NO | NO | STANDARD G | PARTIAL |
| Controller JSON-vs-View | YES | ADR-0007 + 0009 | dual controllers | `make:controller` | NO | NO | STANDARD G | PARTIAL |
| FormRequest | YES | STANDARD E–F ADR-0002 | validation.FormRequest | `make:request` | NO | package tests | STANDARD E | YES (pkg) |
| Index/Filter request | YES | STANDARD E | golden Product/Post | `make:request --index` | NO | NO | YES | YES |
| Nested validation | PARTIAL | STANDARD F | dotted keys | NO | NO | PARTIAL | gaps | PARTIAL |
| unique/exists | PARTIAL | ADR-0010 | PresenceChecker fail-open | NO | NO | NO | STANDARD F | PARTIAL |
| Service layer | YES | STANDARD H ADR-0001 | OrderPlacementService | `make:service` | NO | NO | STANDARD H | NO |
| UseCase/Action/DTO | NOT SUPPORTED | ADR-0001 | — | NO | NO | NO | STANDARD B | NO |
| Domain Entity/VO | NOT SUPPORTED | STANDARD I | — | NO | NO | NO | STANDARD I | NO |
| Repository | PARTIAL | ADR-0003 | make:repository | `make:repository` | NO | NO | STANDARD J | NO |
| ORM models | YES | STANDARD K | orm.Model | `make:model` | NO | orm tests | STANDARD K | YES |
| ORM query API | YES | STANDARD K | Querier | NOT APPLICABLE | NO | YES | STANDARD K | YES |
| Cursor pagination | NOT SUPPORTED | STANDARD K | — | NO | NO | NO | STANDARD K | NO |
| Relationships | YES | STANDARD L | EagerHasMany | NO | NO | YES | STANDARD L | YES |
| String eager load | NOT SUPPORTED | STANDARD L | — | NO | NO | NO | STANDARD L | NO |
| Transactions | YES | STANDARD M ADR-0004 | orm.Transaction | NO | NO | YES | STANDARD M | YES |
| Nested TX | NOT SUPPORTED | STANDARD M | — | NO | NO | NO | STANDARD M | NO |
| Query context.Context | NOT SUPPORTED | G-M6 | — | NO | NO | NO | gaps | NO |
| Routes web/api | YES | STANDARD N | RegisterWeb | new templates | doctor | YES | STANDARD N | YES |
| routing.From vs contracts.Router | YES | conflicts C5 | — | NO | NO | NO | STANDARD N | NO |
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
| Architecture doctor | NO | enforcement.md | — | NO | NO | NO | enforcement | NO |
| Completeness matrix | YES | this file | yaml twin | NOT APPLICABLE | NO | NO | YES | NO |
