# Changelog

All notable changes to ZATRANO are documented in this file.

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
