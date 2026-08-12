# Changelog

All notable changes to ZATRANO are documented in this file.

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
