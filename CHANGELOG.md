# Changelog

All notable changes to ZATRANO are documented in this file.

## 1.5.1 - 2026-08-30

### Added

- AI streaming tool-call deltas: `StreamToolCallDelta`, assembled `ToolCalls` on Done, `CollectStream` helper

## 1.5.0 - 2026-08-30

### Added

- AI capability registry: `Capability`, `Capabler`, `Capabilities` / `Supports` / `Describe` / `DescribeAll`, optional `SetModels` metadata

## 1.4.5 - 2026-08-30

### Added

- AI tool calling: `Tool` / `FunctionTool`, `ToolCall`, `ToolChoice`, `ToolResultMessage`, `HasToolCalls`

## 1.4.4 - 2026-08-30

### Added

- AI structured output: `ResponseFormat`, `JSONObject` / `JSONSchema`, `ChatJSON`, `ChatResponse.UnmarshalJSON`

## 1.4.3 - 2026-08-30

### Added

- AI `Observe` hook (`Observer` / `FuncObserver`) with latency, usage, attempts, fallbacks
- AI `Drivers()` / `Profiles()` return sorted names

## 1.4.2 - 2026-08-30

### Added

- AI streaming: optional `StreamDriver`, `ChatStream`, OpenAI-compatible SSE (`stream: true`)

## 1.4.1 - 2026-08-30

### Added

- AI error classification (`Kind`, `*Error`, `Classify`) with retry/backoff and smart profile fallback (auth/invalid stop; 429/5xx retry then fallback)
- AI retry config: `AI_RETRY_MAX`, `AI_RETRY_INITIAL_MS`, `AI_RETRY_MAX_MS`, `AI_FALLBACK_ON_TIMEOUT`

## 1.4.0 - 2026-08-30

### Added

- AI manager: named providers, `Using` / `Profile` clients, ordered profile fallback, `openai_compatible` driver, optional `EmbeddingDriver` (`Embed`)

### Changed

- AI package split into manager/client/profile/drivers; boot via `BootConfig` (flat `AI_*` config still supported)

## 1.3.4 - 2026-08-30

### Added

- CSRF `SkipAnonymousSeed` / `SetSessionCookieName`: skip token + `XSRF-TOKEN` cookie on safe methods without a session cookie (CDN-friendly)

## 1.3.3 - 2026-08-29

### Changed

- Skip `Set-Cookie` / session save for untouched anonymous sessions (CDN-friendly public pages)

## 1.3.2 - 2026-08-29

### Changed

- Default app no longer ships `make:auth` / `make:dashboard` generated files; use the CLI scaffolds and wire providers/routes yourself
- Empty application skeleton dirs (`events`, `listeners`, `jobs`, …) kept with `.gitkeep`

### Removed

- Route parameter constraints: `Where`, `WhereNumber`, `WhereUuid`, `WhereAlpha` (validate in handlers instead)

### Added

- Catch-all route params `{*slug}` / `{slug*}` (replaces docs `Where("slug", ".+")`)

## 1.3.1 - 2026-08-28

### Fixed

- Auth/dashboard scaffolds: remove forbidden dot imports (`http`/`routing`) for staticcheck ST1001; gofmt dashboard sources

## 1.3.0 - 2026-08-28

### Added

- `make:dashboard` CLI scaffold: modular admin shell (users, notifications, roles, RBAC matrix, settings, live analytics, impersonate, `/api/v1`)
- Dashboard stubs: CSS/JS shell (ZATRANO red), layouts, i18n (`en`/`tr`), migrations, `DashboardServiceProvider`
- Auth API routes versioned under `/api/v1/auth` via `api.Version`
- `DatabaseUserProvider.RetrieveByToken` uses hydrate (consistent with `RetrieveByID`)

### Docs

- README / PACKAGES links for dashboard scaffold; site guide at zatrano.com/docs/dashboard-scaffold

## 1.2.28 - 2026-08-24

### Added

- ORM eager parity: `Then`/`Nested`, `LoadMissing`, constrain `*Fn` loaders, `EagerCount`/`EagerExists`/`EagerMax|Min|Avg|Sum`
- ORM relations: `MorphToMany`/`MorphedByMany`, `WhereHasMorph`/`WhereHasThrough`, `withPivot` via `Pivot()`/`MarkPivot`
- ORM model DX: `Hidden`/`Visible`, `ToMap`/`ToJSON`/`ToArray`, accessors/mutators/`Appends`, `Defaults()`, UUID/ULID keys
- ORM strict/lazy tooling: `PreventLazyLoading`, `Cursor`/`Lazy`, `Collection`, `Without`/`WithOnly`, `Load`, relation subqueries
- Model lifecycle events: `saving`/`saved`/`retrieved`/`replicating`; `events.LifecycleObserver`

## 1.2.27 - 2026-08-23

### Security

- Google OAuth now fail-closes on missing/false `email_verified` (never assumes verified)
- `Persist` uses `User.EmailVerified` for `email_verified_at`; unverified emails cannot link to an existing account by email
- Production rejects StubProvider / placeholder OAuth credentials (`SetAllowStubProviders`, register-time fail-fast)
- GitHub primary email selection requires an explicit `verified=true` claim

### Added

- `User.EmailVerified`, `SetAllowStubProviders` / `AllowStubProviders`, `IsPlaceholder`, `ErrStubNotAllowedInProduction`
- Social security tests (verified/unverified/missing claim, account-linking attack, dual `sub`, stub prod/dev)

## 1.2.26 - 2026-08-23

### Fixed

- `query.Insert` no longer requires an `id` column on PostgreSQL/SQL Server; `InsertGetID` returns the PK. Fixes password reset token persistence and forgot-password mail delivery on pgsql.

## 1.2.25 - 2026-08-23

### Changed

- Auth notification emails (password reset, verify email, password changed) use `auth.mail_*` locale keys (`APP_LOCALE` + fallback); reset body is link-only

### Added

- `notification.SetTranslator` / `SetMailDefaults` (and Manager wrappers) for localized built-in auth mails
- Locale keys: `mail_reset_*`, `mail_verify_*`, `mail_password_changed_*` (en/tr defaults + make:auth stubs)

## 1.2.24 - 2026-08-23

### Fixed

- SMTP `MAIL_ENCRYPTION=tls`/`starttls` no longer uses implicit TLS; only `ssl` or port `465` select SMTPS (Gmail `:587` STARTTLS works again)

## 1.2.23 - 2026-08-20

### Changed

- Social OAuth scaffold flash messages use localization (`auth.social_*`) with `:provider` placeholders (en/tr)

### Added

- Auth locale keys: `provider_google`, `provider_github`, and social OAuth flash strings in defaults + make:auth lang stubs

## 1.2.22 - 2026-08-20

### Added

- AI chat defaults from config (`model`, `base_url`, `timeout`, `temperature`, `max_tokens`)
- `context.Context` on AI `Driver.Chat` / `Manager.Chat` with request timeout
- Demo route `POST /demo/ai/chat`

### Changed

- `LogDriver` logs prompt/reply via app logger `Infof`
- OpenAI-compatible driver wired from config (`AI_BASE_URL`, model, HTTP timeout)

## 1.2.21 - 2026-08-20

### Added

- Billing gateway drivers (`memory` / `stripe`) with `Manager.Extend` / `Use`, `Billable` helpers, and `POST /billing/webhook`
- Async billing receipt notifications (`InvoicePaidNotification`, `SubscriptionStartedNotification`) via `notification.Send`
- SMS multi-driver manager (`SmsManager.Extend` / `Use`, channels `sms` and `sms.<driver>`)

### Changed

- Split monolithic `packages/billing` into gateway/manager/webhook modules; config adds `billing.default`, Stripe webhook secret, checkout URLs
- Auth user stubs include `stripe_customer_id` and Billable methods

## 1.2.20 - 2026-08-20

### Changed

- Mail folded into `packages/notification` (removed `packages/mail`, `mail.From`, `make:mail`, and container `mail`/`sms`/`push` bindings)
- Notifications always dispatch asynchronously via `notification.Send` / `SendMany` (`SendNow` for sync/tests)
- Auth password reset, email verification (register / resend / email change), and password-changed notices go through notification channels

### Added

- Built-in notifications: `PasswordResetNotification`, `VerifyEmailNotification`, `PasswordChangedNotification`
- Auth helpers: `SendEmailVerification`, verification URL / sender hooks on the auth manager

## 1.2.19 - 2026-08-20

### Fixed

- Security CI: Trivy action pin, Semgrep noise split, gosec `#nosec` config, `APP_KEY` for race tests
- Bump `golang.org/x/crypto` and `golang.org/x/text` (incl. nested driver/mongo modules) for Trivy HIGH CVEs

## 1.2.18 - 2026-08-20

### Added

- Security CI pipeline: parallel gosec (SARIF), govulncheck, Semgrep, Trivy FS, and Go fuzz jobs with a Security Summary
- `tests/fuzz` targets for router, validation/HTTP binding, session/CSRF, and query builder
- `.gosec.json` exclude config and `security/semgrep/zatrano-rules.yml` custom rules scaffold

### Fixed

- Router path compilation no longer panics on invalid UTF-8 / bad regex patterns

## 1.2.17 - 2026-08-20

### Security

- Session: directory `0700`, atomic writes, cross-process file locks; remember-me cookie Secure on HTTPS
- Password reset / remember hashes: constant-time digest comparison helpers
- CSRF: Origin / Referer / `Sec-Fetch-Site` defense-in-depth (token still required)
- CORS: no wildcard default in production; credentials never combine with `*`
- Backup: restore path containment, `0600` files, credentials via temp files (not argv), error redaction, connection-name validation
- Runtime dirs (`cache`, `log`, `audit`, OAuth store, schedule locks, SQLite parent): `0700` / file `0600`

## 1.2.16 - 2026-08-19

### Fixed

- PDF TTF parsing: gosec G115 integer conversions (FWORD, cmap delta, BMP/Unicode runes)

## 1.2.15 - 2026-08-19

### Fixed

- PDF embeds a system TrueType font (Arial on Windows) for non-ASCII text so Turkish (and other) glyphs render; ASCII still uses Helvetica

## 1.2.14 - 2026-08-19

### Fixed

- staticcheck SA4006/U1000 in `packages/backup` (unused assignment and dead helper)

## 1.2.13 - 2026-08-19

### Added

- Multi-driver `db:backup` / `db:restore` / `db:backup:list` via native CLI tools (`mysqldump`, `pg_dump`, `sqlpackage`, `exp`/`imp`, `mongodump`) plus SQLite file copy; `--connection` flag

## 1.2.12 - 2026-08-19

### Added

- XLSX export (`xlsx.FromMaps` / `Response`) with round-trip import
- CSV options (delimiter, UTF-8 BOM) and `export.ToMaps` format detection
- PDF multipage text, `FromMaps` tables, and `Inline` for browser viewing

## 1.2.11 - 2026-08-19

### Fixed

- CSRF token generation panics if `crypto/rand` fails instead of ignoring the error

## 1.2.10 - 2026-08-18

### Fixed

- View compiler emits backtick string literals for `dataGet` / comparison / CSRF / auth paths so `@if` inside HTML attributes (e.g. `class="@if($save_disabled)…@endif"`) no longer nests double quotes in the compiled Go template

## 1.2.9 - 2026-08-18

### Changed

- Social avatar is a provider snapshot on `social_accounts`; canonical display photo is `users.avatar` (`Persist` / `make:auth` stubs)
- README origin story rewritten; CI and meta badges restored on two rows
- Security docs live on zatrano.com (`/docs/security*`); repo keeps slim `SECURITY.md` reporting policy

### Fixed

- Zip-slip containment in `zipx.Extract` + per-member size cap
- WebSocket / TOTP / bloom integer-conversion hardening for gosec G115
- gosec CI excludes for nested modules (`qr`, `mongo`, `webauthn`, optional SQL drivers)

## 1.2.8 - 2026-08-17

### Fixed

- Path containment treats `\` as a separator on all OS (Linux CI rejected Windows-style traversal)
- Upload filename sanitize strips `\` directories cross-platform
- CI Go toolchain pinned to **1.25.13** (stdlib govulncheck fixes)
- `golang.org/x/text` → v0.39.0
- gofmt on `trustedproxy_test.go`

## 1.2.7 - 2026-08-17

### Security

- Session IDs restricted to hex; path traversal via cookie blocked
- Query builder sanitizes OrderBy/GroupBy/Having/Where/Join identifiers and operators
- Mongo equality filters reject `$` operator injection
- Password-reset / email verification hashes use SHA-256
- Session cookie `Secure` when HTTPS or `SESSION_SECURE=true`
- HTTP server Read/Write/Idle timeouts
- `packages/safepath`: shared path containment; static `public/` and `LocalDisk` reject traversal
- Upload `StoreAs` basenames filenames; `X-Max-Upload` cannot raise server cap (`MAX_UPLOAD_BYTES`)
- WebSocket `Upgrade` defaults to same-origin Origin checks (`UpgradeWithCheckOrigin` / `AllowAnyOrigin`)
- Production ignores `TRUSTED_PROXIES=*` unless `TRUST_PROXIES_ALLOW_STAR=true`
- CSRF `Middleware` protects all paths by default (scaffold still uses `csrf.Except("/api")`)
- Cookie jar / `Make` honor `COOKIE_SECURE` / `SESSION_SECURE`; XSRF cookie sets `Secure` on HTTPS
- CI: `.github/workflows/security.yml` (vet, race subset, staticcheck, gosec, govulncheck, fuzz smoke)
- Docs: `SECURITY_AUDIT.md`, `SECURITY_REPORT.md`, `docs/security/testing.md`, security demo app

## 1.2.6 - 2026-08-16

### Fixed

- Social OAuth stub authorize URL uses app origin (`/oauth/{provider}/authorize`), not `oauth.zatrano.test` (ZATRANO-033)
- `SocialServiceProvider.Boot` registers same-origin stub authorize handlers when credentials are placeholders

## 1.2.5 - 2026-08-16

### Fixed

- `@lang` inside `@foreach` reads locale from Execute root (`$`), not the loop item (ZATRANO-032)

## 1.2.4 - 2026-08-16

### Fixed

- `make:auth` profile / password / 2FA / logout-other-devices routes and stubs under `/auth/*` (ZATRANO-030)
- Foundation view `dict` returns `map[string]any` so `<x-*>` `mergeDict` works (ZATRANO-031)

## 1.2.3 - 2026-08-16

### Fixed

- Default `packages/version` string was still `1.2.1` after the 1.2.2 cut
- `gofmt` on `make:model` `Name()` alignment (`packages/console/database.go`)

## 1.2.2 - 2026-08-16

### Added

- ORM model `Connection() string` routes queries to a named SQL connection (`DB_CONNECTIONS`)
- `make:model --connection=pgsql` stubs `Connection()` on the model
- MongoDB as a first-class `db:setup` driver (`packages/database/driver/mongo`) alongside SQL engines
- Multi-DB env hints from `db:setup` (`DB_MONGO_URI`, `DB_MYSQL_*`, `DB_PGSQL_*`, …)
- README section: single/multi database + model connection selection

### Changed

- Mongo boots from `DB_CONNECTIONS` / `DB_MONGO_URI` (legacy `package:enable mongo` still works if not already bound)

## 1.2.1 - 2026-08-16

### Fixed

- Default SQLite driver module require uses `v1.0.0` (proxy-resolvable) instead of `v0.0.0`

## 1.2.0 - 2026-08-16

### Added

- Optional SQL driver modules: `sqlite`, `mysql`, `pgsql`, `mssql`, `oracle` (lean default: SQLite only)
- `zatrano db:setup` — interactive / `--drivers=` multi-select, writes `bootstrap/database_drivers.go`, `go get`s only selected drivers
- Multi-database config: `DB_CONNECTIONS` + per-connection `DB_<NAME>_HOST` overrides; `app.DB().Connection("pgsql")`
- Oracle dialector, schema types, query placeholders (`:n`), migrations table
- Docker Compose `oracle` profile

### Changed

- Root `go.mod` no longer pulls MySQL/PostgreSQL/MSSQL drivers unless installed via `db:setup`

## 1.1.2 - 2026-08-16

### Added

- SQL Server (`mssql` / `sqlserver`) connection, schema, migrations, query placeholders (`@pN`), `db:create`
- Docker Compose profiles: `postgres`, `mysql`, `mssql`, `mongo`
- Driver dialect tests + SQLite smoke; optional live smoke via `ZATRANO_LIVE_DB=1`
- `env.GetNonEmpty` so blank `.env` credentials fall back to driver defaults
- `DB_SSLMODE` for PostgreSQL

### Fixed

- Runtime database config now uses `config.Database()` (pgsql default user was wrongly `root`)
- MySQL `ForeignID` is `BIGINT UNSIGNED` (matches `ID()`)
- MySQL JSON columns use native `JSON` type
- SQLite alter uses `ADD COLUMN`
- Insert returns `LastInsertId` errors instead of silent `0`
- Mongo addon fails boot when a real URI cannot ping
- Removed `go.work` / `go.work.sum` (use `go.mod` `replace` only)

## 1.1.1 - 2026-08-16

### Added

- `make:model --translation` / `-t` → `name_tr` / `name_en` fields; `--translation=json` → JSON `translations` cast (ZATRANO-015)
- `make:model -m` writes a table-specific migration (correct table name + translation columns when requested)

### Fixed

- `make:*` CLI commands boot with `CoreApp()` (no DB/session); other commands still use `FromEnv("app")` (ZATRANO-019)

## 1.1.0 - 2026-08-16

Ecommerce shop integration fixes (ZATRANO-001…029 subset).

### Added

- `WebApp` / `APIApp` merge `Preset* ∪ EnabledAddons`
- `db:create` CLI (mysql/pgsql)
- Billing `Checkout` uses Stripe `mode=payment`; `CheckoutPayment` with `price_data` line items
- `schema.ForeignID(...).Constrained(...).CascadeOnDelete()`
- `make:controller --api` / `--admin`; `make:view --layout=`; `make:lang --group=`; `make:auth --social=`
- Auth provider `WithHydrate` for ORM `*models.User` mapping
- Validation `SetDefaultPresenceChecker` (foundation wires DB unique/exists)
- `@lang('key', ['name' => …])` replacements; request-locale-aware `trans`
- Docker Compose `postgres` profile

### Fixed

- CLI default `FromEnv("app")` (was `demo`)
- `make:migration` param shadowing (`s *schema.Builder`)
- Auth routes/middleware/wellknown under `/auth/*` and `/api/auth/*`
- Authorization middleware HTML redirect/abort (not JSON-only)
- Form validation `RedirectBack` fallback = current path
- pgsql default username `postgres`; schema `Rename` for pgsql
- `package:enable` preserves `enabled.go` header comments
- Policy stub uses `authorization.Authenticatable`
- Resource stub comment → `packages/resources`
- `lang/` no longer gitignored by default

## 1.0.8 - 2026-08-15

### Fixed

- Foreach root dotted collection vs range-alias head: `@foreach($pagination.Links as $link)` → `dataGet $ "pagination.Links"`; nested `@foreach($section.pages …)` still uses `$section` when the head is an in-scope alias

## 1.0.7 - 2026-08-15

### Fixed

- Foreach alias rewrite: `@unless` / `@isset` / `@empty` and form attrs (`@selected` / `@checked` / …) compile `__ZRV_*` and `__ZPARENT__.*` (same precedence as `@if`) so loops no longer leave raw `@unless` + early `{{ end }}` → `undefined variable "$inv"`
- `@elseif(__ZRV_alias__)` replacement uses `$$$1` (literal `$alias`), not `$$1`

## 1.0.6 - 2026-08-15

### Fixed

- Nested `@foreach($section.pages as $link)` (docs sidebar): compile inner loops before parent rewrite; dotted collections resolve via `$section`; `@if($link.active)` uses the range var correctly
- Parent lookups in loops use Go’s stable Execute root `$` (avoids nested `$__zparent` overwrite)

## 1.0.5 - 2026-08-15

### Fixed

- Remove unused `rewriteAlias` (staticcheck U1000) left after foreach parent-scope rewrite

## 1.0.4 - 2026-08-15

### Fixed

- Routing: trailing slash normalization on dispatch (`/dashboard/` → `/dashboard`; `/` unchanged; query string kept) — seen integrating davet.link
- Views: `@foreach` / `@forelse` / `@each` keep parent scope (`$var`, `$category.ID`, `@csrf` / `_token`) while row fields use `$item.*`
- Views: `@if` / `@elseif` compile `>`, `>=`, `<`, `<=` (numeric); nested `@if`/`@else` inside `@include` partials stay matched
- Views: unsupported `@if` operators fail fast with a clear compile error (no silent raw leftover / `{{else}}` drift)

### Migration

- Inside `@foreach($items as $item)`, prefer `$item.field` for row data. Bare `$field` now resolves against the **parent** view data (not the loop element). Templates that relied on the old “`.` = item” short names should switch to the alias prefix.

## 1.0.3 - 2026-08-14

### Added

- Auth user-facing errors as localization keys (`auth.email_taken`, `auth.lockout`, …)
- Built-in `localization/defaults/{en,tr}/auth.json`
- `make:auth` HTML stubs fully use `@lang('auth.*')`; controller stub uses `lang()` / `authMsg()`

### Changed

- Locale middleware priority: session → **APP_LOCALE** → Accept-Language (browser no longer overrides configured locale by default)

## 1.0.2 - 2026-08-13

### Fixed

- `gofmt` clean tree for CI coding-style checks
- Authorization middleware test uses per-guard request key (`auth.user.{guard}`)
- GitHub Actions: `actions/checkout@v5`, `actions/setup-go@v6` (Node 24)

## 1.0.1 - 2026-08-12

### Fixed

- Publish nested heavy-package modules (`packages/mongo`, `packages/webauthn`, `packages/qr`) as `v1.0.0` so consumers can resolve them without local `replace` directives

## 1.0.0 - 2026-08-12

### Added

- Thin-kernel architecture: first-party code under `packages/`, lean `core/`
- Boot profiles: `CoreApp`, `MinimalApp`, `App`, `APIApp`, `WebApp`, `DemoApp` + `APP_BOOT` / `bootstrap.FromEnv`
- Package ecosystem CLI: `package:list|enable|disable|preset|init|install|publish|status|doctor`
- Addon registry, presets (`api` / `web` / `demo`), config stubs, catalog (`KindService` / `KindLibrary`)
- Auth parity: per-guard session keys, guard-aware middleware, cache-backed lockout, password broker throttle + silent unknown emails, `MarkEmailAsVerified` + `auth.verified`, 2FA remember-device, challenge lockout, configurable issuer
- Resolve helpers (`auth.From(app)`, `mail.From(app)`, …) across packages
- Website is the sole documentation source (repo `docs/` removed)

### Changed

- Import paths `core/X` → `packages/X` for first-party packages
- Foundation accessors removed from `Application` in favor of package `From` helpers
- Starter notification demo routes/controllers/views removed
- README rewritten for the v1 architecture

### Migration

1. `go get github.com/zatrano/framework@v1.0.0`
2. Rewrite imports to `packages/...`
3. Replace `app.Auth()` / `app.Mail()` / … with `auth.From(app)` / `mail.From(app)` / …
4. Set `APP_BOOT` for production (`app|api|web|minimal`)
5. Enable optional services with `package:enable` / presets

## 0.2.5 - 2026-08-12

### Added

- Localization: `make:lang`, `@choice` view helper, `Accept-Language` negotiation
- Real drivers: scannable QR, ip-api geo lookup, OpenAI-compatible AI, report webhooks, Twilio/HTTP SMS, HTTP push, MongoDB URI mode, Stripe billing mode
- OAuth PKCE + refresh tokens + optional JSON store; WebAuthn via go-webauthn
- WebSocket binary / ping / pong / close frames

### Changed

- Expanded localization documentation and removed stub banners from packages that now run for real when configured

## 0.2.4 - 2026-08-12

### Added

- Real GitHub OAuth provider (token exchange, user + primary email); stub only for placeholder credentials

### Changed

- Auth scaffold views use ZATRANO brand layout (white / red) instead of unfinished teal demo styles
- Notification demo pages rebuilt as complete inbox / send / bulk forms

## 0.2.3 - 2026-08-12

### Added

- Social login persistence helpers (`social.Persist`) and `make:auth` stubs for Google/GitHub accounts
- Env keys for `GOOGLE_*` / `GITHUB_*` OAuth credentials in `.env.example`

## 0.2.2 - 2026-08-12

### Added

- Template `@if` / `@elseif` numeric comparisons (`$limit == 50`, `$n != 0`)

## 0.2.1 - 2026-08-12

### Fixed

- PostgreSQL inserts use `RETURNING id` (lib/pq does not support `LastInsertId`)

## 0.2.0 - 2026-08-12

### Added

- Central notifications: in-app (database), mail, and SMS channels for web + API
- Single and bulk send (`Send` / `SendMany`) with CSV and XLSX recipient import
- Notification inbox store (list, unread, mark read / mark all read)
- Demo web UI (`/notifications`) and REST endpoints under `/api/notifications`
- Lightweight `core/export/xlsx` reader for bulk imports

## 0.1.10 - 2026-08-12

### Fixed

- PostgreSQL migration repository uses `$1, $2, …` placeholders instead of `?`
- Boolean column defaults emit `TRUE`/`FALSE` (Postgres-compatible)
- Database queue `EnsureTable` / queries are dialect-aware (SQLite, MySQL, PostgreSQL)

### Added

- SMTP mailer implicit TLS (`MAIL_ENCRYPTION` / port 465)
- Real Google OAuth provider (stub when credentials are placeholders)

## 0.1.9 - 2026-08-12

### Added

- Short `@section('name', $var)` form for layout sections with view data variables

## 0.1.8 - 2026-08-11

### Fixed

- `Request.Input` / `All` now read multipart form field values

## 0.1.7 - 2026-08-11

### Changed

- View `dataGet` resolves dotted paths on structs (including embedded fields), not only maps

## 0.1.6 - 2026-08-11

### Fixed

- Jobs table migration skips create when the table already exists (queue bootstrap race)

## 0.1.5 - 2026-08-06

### Fixed

- `serve` now loads `.env` before resolving `APP_PORT` (was always defaulting to `8080`)

## 0.1.4 - 2026-08-06

### Added

- Configurable HTTP listen port via `APP_PORT` (config, `serve`, and `Application.Run`)

## 0.1.3 - 2026-08-06

### Fixed

- Nested `@foreach` / `@endforeach` compilation in the view engine

## 0.1.2 - 2026-08-06

### Added

- Fenced code blocks (` ```lang `) in `core/markdown`
- GFM-style pipe tables in `core/markdown`

## 0.1.1 - 2026-08-06

### Added

- Nested markdown discovery for the documentation engine (`core/docs`)
- Sidebar navigation via optional `navigation.json`
- Previous/next page neighbors and custom `ViewRenderer` support for docs routes

## 0.1.0 - 2026-08-05

Initial release.

### Added

- Go module `github.com/zatrano/framework`
- Application skeleton: `app/`, `bootstrap/`, `config/`, `routes/`, `views/`, `database/`, `storage/`, `cmd/zatrano`
- HTTP request/response layer and router
- Controller registration helpers
- View engine (layouts, components, directives)
- Config loading via `.env` and `config/`
- Middleware pipeline (CSRF, CORS, security headers, throttle, and related helpers)
- Database migrations, schema builder, and ORM (`core/orm`)
- Logging, session, cache, queue, mail, validation, and auth packages in `core/`
- CLI entrypoint (`serve`, `migrate`, `make:*`, `about`, …)
- Welcome web route and home controller
- GitHub Actions CI (tests, static analysis, coding style)
- MIT license (Serhan KARAKOÇ)
