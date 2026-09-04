<p align="center">
  <strong>ZATRANO</strong>
</p>

<p align="center">
  <em>The Golang framework for web artisans — thin kernel, opt-in packages, full-stack DX.</em>
</p>

<p align="center">
  <a href="https://github.com/zatrano/framework/actions/workflows/tests.yml"><img src="https://github.com/zatrano/framework/actions/workflows/tests.yml/badge.svg" alt="Tests"></a>
  <a href="https://github.com/zatrano/framework/actions/workflows/static-analysis.yml"><img src="https://github.com/zatrano/framework/actions/workflows/static-analysis.yml/badge.svg" alt="Static Analysis"></a>
  <a href="https://github.com/zatrano/framework/actions/workflows/coding-style.yml"><img src="https://github.com/zatrano/framework/actions/workflows/coding-style.yml/badge.svg" alt="Coding Style"></a>
  <a href="https://github.com/zatrano/framework/actions/workflows/security.yml"><img src="https://github.com/zatrano/framework/actions/workflows/security.yml/badge.svg" alt="Security"></a>
</p>

<p align="center">
  <a href="https://github.com/zatrano/framework/actions/workflows/security.yml"><img src="https://img.shields.io/badge/gosec-SAST-222?logo=go" alt="gosec"></a>
  <a href="https://github.com/zatrano/framework/actions/workflows/security.yml"><img src="https://img.shields.io/badge/govulncheck-CVE-222?logo=go" alt="govulncheck"></a>
  <a href="https://github.com/zatrano/framework/actions/workflows/security.yml"><img src="https://img.shields.io/badge/Semgrep-rules-222?logo=semgrep" alt="Semgrep"></a>
  <a href="https://github.com/zatrano/framework/security"><img src="https://img.shields.io/badge/Trivy-FS-222?logo=aquasecurity" alt="Trivy"></a>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/zatrano/framework"><img src="https://img.shields.io/badge/golang-1.25+-00ADD8?logo=go&logoColor=white" alt="Golang"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="VERSION"><img src="https://img.shields.io/badge/version-2.0.0--dev-green.svg" alt="Version"></a>
  <a href=".github/SECURITY.md"><img src="https://img.shields.io/badge/security-policy-brightgreen.svg" alt="Security Policy"></a>
</p>

<p align="center">
  <a href="https://zatrano.com/docs">Documentation</a> ·
  <a href="https://zatrano.com/docs/installation">Installation</a> ·
  <a href="https://zatrano.com/docs/boot-profiles">Boot Profiles</a> ·
  <a href="https://zatrano.com/docs/package-ecosystem">Package Ecosystem</a> ·
  <a href="PACKAGES.md">Packages guide</a> ·
  <a href="https://github.com/zatrano/framework/releases">Releases</a>
</p>

---

## The story

ZATRANO began in **February 2018** as the internal spine of real applications — not a public demo, not a portfolio piece. For years it lived where frameworks are judged hardest: behind logins, migrations, queues, mail, and production uptime. Pieces were added only when a shipped product needed them; pieces were rewritten when they failed that test.

**v1** (August 2026) opened that stack under MIT. **v2** is the next cut of the same lineage: this repository is only the framework. Applications are created with `zatrano new`. Optional addons live in a separate module. Nothing optional is linked until you ask for it.

Eight years in the dark. Then the doors opened. Welcome aboard.

---

## What is ZATRANO?

ZATRANO is an opinionated Golang web framework: routing, views, validation, ORM, auth, queues, notifications, CLI, and a first-party AI layer — without forcing every package into every binary.

**v2** is two modules and a generated app:

| Piece | Module / location | Role |
| ----- | ----------------- | ---- |
| **Kernel** | `kernel/` | `Application`, container, catalog, secure HTTP hooks |
| **Implementations** | `packages/` | Kernel, foundation, and intelligence (`github.com/zatrano/framework/packages/...`) |
| **Foundation boot** | `bootstrap/foundation` | Wires DB, auth, session, cache, queue, views, notifications |
| **Intelligence** | `packages/ai`, `rag`, `agent` | First-party AI layer (stays in this module) |
| **Your app** | `zatrano new` | Routes, views, providers, `cmd/app` |
| **Service addons** | [`zatrano/packages`](https://github.com/zatrano/packages) | Optional services (`oauth`, `billing`, `social`, …) |
| **Library addons** | [`zatrano/packages`](https://github.com/zatrano/packages) | Import-only helpers (`collection`, `totp`, …) |

`packages/` in this repo is the framework implementation. It is not a copy of the addon repo. Addon **names** appear in `kernel/catalog.go` for discovery; addon **code** lives in [`github.com/zatrano/packages`](https://github.com/zatrano/packages). This module must not `require` that one (import cycle).

Nothing optional is magic-loaded. You pick a **boot profile** and enable only the addons you need.

For what each package is for and how to use it, see **[PACKAGES.md](PACKAGES.md)**.

```text
┌─────────────────────────────────────────────────────────┐
│  Your app (zatrano new)  ·  routes  ·  views            │
├─────────────────────────────────────────────────────────┤
│  bootstrap/   APP_BOOT · EnabledAddons · presets        │
├──────────────────┬──────────────────────────────────────┤
│  foundation      │  addons (github.com/zatrano/packages)│
│  auth db queue … │  oauth billing webauthn …            │
├──────────────────┴──────────────────────────────────────┤
│  kernel/   thin kernel                                  │
└─────────────────────────────────────────────────────────┘
```

## Why teams choose it

- **Golang speed** with a coherent application model — not 20 micro-libraries glued by hand
- **Opt-in weight** — kernel stays lean; API/web apps take foundation; addons are enabled explicitly
- **Batteries included** — auth (guards, 2FA, lockout, trusted devices), ORM, migrations, queues, notifications, localization
- **One CLI** — `new`, `serve`, `migrate`, `make:`*, `package:enable|preset|doctor`, `db:setup`, …
- **Docs that live with the product** — [zatrano.com/docs](https://zatrano.com/docs)

## Requirements

- Golang **1.25+**
- Optional database: SQLite, MySQL, PostgreSQL, SQL Server, Oracle, MongoDB via `db:setup` (none linked by default)
- Optional: Redis, Stripe, OpenAI — only if you enable those addons

## Quick start

This repository is the **framework**. Scaffold an application, then serve it:

```bash
git clone -b v2-dev https://github.com/zatrano/framework.git
cd framework
go run ./cmd/zatrano new myapp --replace .
cd myapp
cp .env.example .env
go run ./cmd/app key:generate
go run ./cmd/app serve
```

Open [http://localhost:8080](http://localhost:8080).

A new app has **no database** and **no addons** until you opt in.

### Docker

The image in this repo is the `zatrano` CLI. Application images are generated with `zatrano new` (`Dockerfile` in the new project).

Optional database profiles remain in `docker-compose.yml` (`postgres`, `mysql`, `mssql`, `mongo`, `oracle`).

As a module dependency (v2 development line):

```bash
go get github.com/zatrano/framework@v2-dev
```

Tagged v1 releases remain on [GitHub Releases](https://github.com/zatrano/framework/releases).

### First production-shaped boot

```bash
# Write EnabledAddons + config stubs for an API or web starter
go run ./cmd/app package:init api
# or: package:init web

# APP_BOOT=app     → foundation + EnabledAddons
# APP_BOOT=api     → foundation + API preset
# APP_BOOT=web     → foundation + web preset
# APP_BOOT=minimal → foundation only
# APP_BOOT=core    → kernel only

go run ./cmd/app package:doctor
go run ./cmd/app serve
```

`APP_BOOT` defaults to **app** when unset. Scaffold commands (`make:`*) always boot with `App(Kernel())` (no database).

## Databases (single & multi)

Drivers are **opt-in modules**. A new app has **no database** until you choose one:

```bash
go run ./cmd/app db:setup
# or non-interactive:
go run ./cmd/app db:setup --drivers=sqlite --yes
go run ./cmd/app db:setup --drivers=sqlite,mysql,pgsql,mongo --default=mysql --yes
```

Supported: `sqlite` · `mysql` · `pgsql` · `mssql` · `oracle` · `mongo`. SQLite is installed the same way as MySQL/PostgreSQL (not pre-linked).

SQLite’s driver lives in this module (`packages/database/driver/sqlite`). MySQL, PostgreSQL, SQL Server, Oracle, and Mongo drivers live in [`github.com/zatrano/packages`](https://github.com/zatrano/packages).

### Env

```env
DB_CONNECTION=mysql
DB_CONNECTIONS=mysql,pgsql,mongo

DB_HOST=127.0.0.1
DB_MYSQL_HOST=127.0.0.1
DB_MYSQL_DATABASE=shop
DB_PGSQL_HOST=127.0.0.1
DB_PGSQL_DATABASE=analytics
DB_PGSQL_USERNAME=postgres
DB_MONGO_URI=mongodb://localhost:27017
DB_MONGO_DATABASE=zatrano
```

Per-connection keys: `DB_<NAME>_HOST`, `_PORT`, `_DATABASE`, `_USERNAME`, `_PASSWORD`, `_SSLMODE`, `_SERVICE`, `_URI`.

### Models pick a connection

```golang
type Order struct {
    orm.Model
    Total int64 `db:"total"`
}

func (m *Order) TableName() string  { return "orders" }
func (m *Order) Connection() string { return "pgsql" } // must appear in DB_CONNECTIONS
```

```bash
go run ./cmd/app make:model Order --connection=pgsql
```

Runtime: `app.DB().Connection("pgsql")` for SQL; Mongo binds as container key `mongo` (and `mongo.<name>` when the connection name is not `mongo`).

Guide: [Database](https://zatrano.com/docs/database) · [ORM](https://zatrano.com/docs/orm) · [MongoDB](https://zatrano.com/docs/mongodb).

## Boot profiles

| API                          | `APP_BOOT`    | Boots                               |
| ---------------------------- | ------------- | ----------------------------------- |
| `App(Kernel())`              | `core`        | Kernel only                         |
| `App(Minimal())`             | `minimal`     | Foundation + your routes, no addons |
| `App()`                      | `app`         | Minimal + `EnabledAddons`           |
| `App(WithPresetAPI()/Web())` | `api` / `web` | Lean presets                        |
| `App(WithAddons(...))`       | —             | Foundation + an explicit addon list |

```golang
import (
    "github.com/zatrano/framework/bootstrap"
    "github.com/zatrano/framework/packages/auth"
)

func main() {
    app := bootstrap.FromEnv()       // reads APP_BOOT (default "app")
    // app := bootstrap.FromEnv("api")
    // app := bootstrap.App(bootstrap.WithPresetAPI())

    _ = auth.From(app)               // resolve services — not app.Auth()
    app.Run()
}
```

There is no demo boot profile. Enable addons with `EnabledAddons` or `App(WithAddons(...))`.

Full guide: [Boot Profiles](https://zatrano.com/docs/boot-profiles).

## Package ecosystem

Foundation packages (`auth`, `database`, `orm`, `queue`, …) ship in this module. Optional addons ship in [`github.com/zatrano/packages`](https://github.com/zatrano/packages): blank-import them in the app, then enable.

```bash
go get github.com/zatrano/packages@dev

# in cmd/app/main.go (or app/addons.go):
#   import _ "github.com/zatrano/packages/oauth"

go run ./cmd/app package:list
go run ./cmd/app package:list --libraries
go run ./cmd/app package:status
go run ./cmd/app package:enable oauth
go run ./cmd/app package:preset api --merge
go run ./cmd/app package:doctor
```

| Kind         | Enable?                        | Where                                                          | Examples                                                       |
| ------------ | ------------------------------ | -------------------------------------------------------------- | -------------------------------------------------------------- |
| **Service**  | Yes — `EnabledAddons` / preset | `github.com/zatrano/packages`                                  | `oauth`, `billing`, `social`                                   |
| **Library**  | No — just `import`             | `github.com/zatrano/packages`                                  | `collection`, `totp`                                           |
| **Intelligence** | Same as a service          | this module                                                    | `ai` (RAG / agent are libraries)                               |
| **Heavy**    | Only when needed               | separate modules                                               | `webauthn`, `qr`; DB engines via `db:setup`                    |
| **Database** | `db:setup`                     | sqlite here; others in `zatrano/packages`                      | `sqlite`, `mysql`, `pgsql`, `mssql`, `oracle`, `mongo`         |

Resolve from the container:

```golang
auth.From(app)
auth.Passwords(app)
notification.From(app)
session.From(app)
database.Migrator(app)
```

There is no `packages/mail`. Send email through `notification` with `Channels: ["mail"]`.

Guide: [PACKAGES.md](PACKAGES.md) (what + how) · [Package Ecosystem](https://zatrano.com/docs/package-ecosystem) · [Resolving Services](https://zatrano.com/docs/accessors).

## Authentication (highlights)

```bash
go run ./cmd/app make:auth
go run ./cmd/app make:dashboard
```

Session guards with **per-guard keys**, remember-me, password reset (no enumeration), email verification events, cache-backed lockout, TOTP MFA, **remember this device**, multi-device logout, intended URLs, password confirmation.

`make:dashboard` scaffolds a modular admin shell (`/dashboard`), optional users / notifications / roles / RBAC / settings / analytics / impersonate, and JSON under `/api/v1` (with auth at `/api/v1/auth`). Not pre-installed in the default app — run the CLI and follow its next steps.

```golang
ok, err := auth.From(app).Attempt(req, map[string]string{
    "email": "ada@example.com", "password": "secret",
}, true)
ok, err = auth.From(app).ChallengeTwoFactor(req, code, true) // trusted device
```

Guides: [Authentication](https://zatrano.com/docs/authentication) · [Dashboard Scaffold](https://zatrano.com/docs/dashboard-scaffold).

## What ships in the box

**This module (framework)**

| Area                | Capabilities                                                                                                           |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| **HTTP**            | Router, controllers, middleware (CSRF, CORS, throttle, security), requests/responses, cookies, flash, URLs             |
| **Views**           | Layouts, components, directives, markdown, file-based pages                                                            |
| **Validation**      | Rules, form requests, error bags                                                                                       |
| **Data**            | Query builder, schema, migrations, ORM (relations, scopes, eager load), seeders                                        |
| **Auth & security** | Guards, 2FA, lockout, API tokens, encryption, hashing, trusted proxies, `make:dashboard` admin shell                   |
| **Async**           | Queues (sync/DB/Redis), notifications (database/mail/SMS/push), broadcasting, scheduler                                |
| **Platform**        | Config + `.env`, sessions, cache, filesystem, localization, health, maintenance                                        |
| **Intelligence**    | AI chat, RAG, agents                                                                                                   |
| **Tooling**         | CLI (`zatrano new`, `make:*`, `package:*`, `db:setup`, `describe`, `doctor`)                                           |

**Addon module** ([`zatrano/packages`](https://github.com/zatrano/packages)): OAuth server, social login, WebAuthn, billing, Mongo client, backups, OpenAPI/GraphQL helpers, inspector, and import-only libraries.

## Project layout

```text
bootstrap/                     Boot profiles, foundation, EnabledAddons, presets
cmd/zatrano/                   Framework CLI (`new`, `make:*`, `package:*`, `db:setup`, …)
config/                        Default config maps
kernel/                        Thin kernel + catalog
packages/                      Kernel, foundation, and intelligence implementations
packages/console/templates/    Starter copied by `zatrano new`
```

There is no application skeleton in this tree (`app/`, `routes/`, `views/`, `database/` belong to apps created by `zatrano new`). Generated apps use `cmd/app`.

## Documentation

Product guides: **[https://zatrano.com/docs](https://zatrano.com/docs)**.

In-repo packages guide: **[PACKAGES.md](PACKAGES.md)** (purpose, enable, usage, docs links).

| Start here                                                      |                                  |
| --------------------------------------------------------------- | -------------------------------- |
| [PACKAGES.md](PACKAGES.md)                                      | Packages guide (this repo)       |
| [Installation](https://zatrano.com/docs/installation)           | Clone, key, serve                |
| [Boot Profiles](https://zatrano.com/docs/boot-profiles)         | `APP_BOOT`, Minimal / API / Web  |
| [Package Ecosystem](https://zatrano.com/docs/package-ecosystem) | enable, presets, doctor          |
| [Resolving Services](https://zatrano.com/docs/accessors)        | `From(app)` migration table      |
| [Authentication](https://zatrano.com/docs/authentication)       | Full auth surface                |
| [Release Notes](https://zatrano.com/docs/releases)              | Version history                  |

## Breaking changes in v2

Unreleased (`v2-dev`). Apps should be created with `zatrano new`, not by cloning this repo as a project.

- No application skeleton in the framework module. `app/`, `routes/`, `views/`, `public/`, `lang/`, and application `database/` live in generated apps. `cmd/zatrano` is the framework CLI; apps use `cmd/app`.
- Optional addons live in `github.com/zatrano/packages` and must be blank-imported by the consumer. This module does not `require` that one.
- `bootstrap.ApplicationProviders()` is empty; pass `bootstrap.WithProviders(...)` from the application.
- No database is linked by default. SQLite is installed with `db:setup --drivers=sqlite`.
- `App(WithDemo())`, `DemoAddons`, `APP_BOOT=demo`, and `package:preset demo` are removed. Enable addons explicitly.
- Duplicate top-level addon copies are gone from `packages/`. Foundation helpers are nested (`auth/totp`, `schedule/cron`, `orm/pagination`, …).

Upgrade from v1: generate a new app (`zatrano new`) or move your application tree out of the framework clone, then `go get github.com/zatrano/framework@v2-dev` and blank-import addons you still need. See [Release Notes](https://zatrano.com/docs/releases).

## Versioning

Semantic versioning. Tags: `vMAJOR.MINOR.PATCH` on [GitHub Releases](https://github.com/zatrano/framework/releases).

| Line       | Meaning                                      |
| ---------- | -------------------------------------------- |
| **v2-dev** | Two-module architecture (this branch)        |
| **v1.x**   | Last tagged line (`v1.6.6`)                  |
| **v0.2.x** | Pre-ecosystem line (historical)              |

## Community

- Docs: [zatrano.com/docs](https://zatrano.com/docs)
- Framework: [github.com/zatrano/framework](https://github.com/zatrano/framework)
- Addons: [github.com/zatrano/packages](https://github.com/zatrano/packages)
- LinkedIn: [linkedin.com/company/zatrano](https://www.linkedin.com/company/zatrano)

## Contributing

Issues and PRs welcome. Keep changes focused, add tests when practical, follow `gofmt` and existing package boundaries. Guide: [Contributing](https://zatrano.com/docs/contributions).

v2 work lands on **`v2-dev`** in this repository and **`dev`** in `zatrano/packages`.

## Security

CI (`.github/workflows/security.yml`) runs on `main` pushes, pull requests, and a weekly cron:

| Tool | Role |
|------|------|
| **go vet** / **go test -race** | Compiler checks and data-race detection |
| **gosec** | Golang SAST; SARIF uploaded to the GitHub Security tab (excludes in `.github/gosec.json`) |
| **govulncheck** | Known Golang module CVEs |
| **Semgrep** | `p/golang` + `p/security-audit` plus `.github/semgrep/zatrano-rules.yml` |
| **Trivy FS** | Dependency/secret/misconfig scan (fails on HIGH/CRITICAL only) |
| **Golang fuzz** | `tests/fuzz` targets (3m each in CI; longer locally) |

Report vulnerabilities privately to Serhan KARAKOÇ — [serhankarakoc@zatrano.com](mailto:serhankarakoc@zatrano.com).

## License

[MIT](LICENSE) · Copyright (c) 2026 Serhan KARAKOÇ
