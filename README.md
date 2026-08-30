<p align="center">
  <strong>ZATRANO</strong>
</p>

<p align="center">
  <em>The Go framework for web artisans — thin kernel, opt-in packages, full-stack DX.</em>
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
  <a href="https://pkg.go.dev/github.com/zatrano/framework"><img src="https://img.shields.io/badge/go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="VERSION"><img src="https://img.shields.io/badge/version-1.4.2-green.svg" alt="Version"></a>
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

ZATRANO began in **February 2018** as the internal spine of real applications — not a public demo, not a portfolio piece. For years it lived where frameworks are judged hardest: behind logins, migrations, queues, mail, and the quiet pressure of production uptime. Pieces were added only when a shipped product needed them; pieces were rewritten when they failed that test.

What you see in **v1** is that long private maturity made explicit. In **August 2026** the same stack was opened under the MIT license so others can build on work that already survived contact with the real world — not a greenfield rewrite dressed up as history.

Eight years in the dark. Then the doors opened. Welcome aboard.

---



## What is ZATRANO?

ZATRANO is an opinionated Go web framework: routing, views, validation, ORM, auth, queues, mail, CLI, and dozens of first-party packages — without forcing every package into every binary.

**v1.0** is the thin-kernel release:


| Layer              | Location               | Role                                                         |
| ------------------ | ---------------------- | ------------------------------------------------------------ |
| **Kernel**         | `core/`                | `Application`, container, catalog, secure HTTP hooks         |
| **Foundation**     | `bootstrap/foundation` | DB, auth, mail, session, cache, queue, views                 |
| **Service addons** | `packages/`*           | Optional container services (`mongo`, `oauth`, `billing`, …) |
| **Library addons** | `packages/`*           | Import-only helpers (`collection`, `totp`, `support`, …)     |


Nothing optional is magic-loaded. You pick a **boot profile** and enable only the addons you need.

For what each package is for and how to use it, see **[PACKAGES.md](PACKAGES.md)**.

```text
┌─────────────────────────────────────────────────────────┐
│  Your app/  ·  routes/  ·  config/  ·  views/           │
├─────────────────────────────────────────────────────────┤
│  bootstrap/   APP_BOOT · EnabledAddons · presets        │
├──────────────────┬──────────────────────────────────────┤
│  foundation      │  service addons (opt-in)             │
│  auth db mail …  │  oauth billing ai webauthn …         │
├──────────────────┴──────────────────────────────────────┤
│  core/   thin kernel                                    │
└─────────────────────────────────────────────────────────┘
```



## Why teams choose it

- **Go speed** with a coherent application skeleton — not 20 micro-libraries glued by hand
- **Opt-in weight** — health binary stays lean; API/web apps take foundation; demos can load everything
- **Batteries included** — auth (guards, 2FA, lockout, trusted devices), ORM, migrations, queues, mail, notifications, localization
- **One CLI** — `serve`, `migrate`, `make:`*, `package:enable|preset|doctor`, …
- **Docs that live with the product** — [zatrano.com/docs](https://zatrano.com/docs)



## Requirements

- Go **1.25+**
- SQLite (default); MySQL, PostgreSQL, SQL Server, Oracle, MongoDB via `db:setup`
- Optional: Redis, Stripe, OpenAI — only if you enable those addons



## Quick start

```bash
git clone https://github.com/zatrano/framework.git
cd framework
cp .env.example .env
go mod tidy
go run ./cmd/zatrano key:generate
go run ./cmd/zatrano serve
```

Open [http://localhost:8080](http://localhost:8080).

### Docker

Default stack uses SQLite inside the `app` service:

```bash
docker compose up --build
```

Optional PostgreSQL (profile `postgres`):

```bash
docker compose --profile postgres up --build
```

Point the app at Postgres by setting `DB_CONNECTION=pgsql`, `DB_HOST=postgres`, and matching `DB_*` credentials (see commented env keys in `docker-compose.yml`). Then run `go run ./cmd/zatrano db:create` (or the binary) before migrate if the database does not exist yet.

As a module dependency:

```bash
go get github.com/zatrano/framework@latest
```



### First production-shaped boot

```bash
# Write EnabledAddons + config stubs for an API or web starter
go run ./cmd/zatrano package:init api
# or: package:init web

# Prefer lean profiles in production (never demo)
# APP_BOOT=app   → foundation + EnabledAddons
# APP_BOOT=api   → foundation + API preset
# APP_BOOT=web   → foundation + web preset
# APP_BOOT=minimal → foundation only

go run ./cmd/zatrano package:doctor
go run ./cmd/zatrano migrate
go run ./cmd/zatrano serve
```

`cmd/zatrano` defaults to **app** when `APP_BOOT` is unset. Set `APP_BOOT` explicitly before shipping. Scaffold commands (`make:`*) always boot with `CoreApp()` (no database).

## Databases (single & multi)

Drivers are **opt-in modules**. Default checkout links **SQLite** only. Install engines with:

```bash
go run ./cmd/zatrano db:setup
# or non-interactive:
go run ./cmd/zatrano db:setup --drivers=sqlite,mysql,pgsql,mongo --default=mysql --yes
```

Supported: `sqlite` · `mysql` · `pgsql` · `mssql` · `oracle` · `mongo` (same list as SQL — Mongo is not a special-case-only addon path).

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

```go
type Order struct {
    orm.Model
    Total int64 `db:"total"`
}

func (m *Order) TableName() string  { return "orders" }
func (m *Order) Connection() string { return "pgsql" } // must appear in DB_CONNECTIONS
```

```bash
go run ./cmd/zatrano make:model Order --connection=pgsql
```

Runtime: `app.DB().Connection("pgsql")` for SQL; Mongo binds as container key `mongo` (and `mongo.<name>` when the connection name is not `mongo`).

Guide: [Database](https://zatrano.com/docs/database) · [ORM](https://zatrano.com/docs/orm) · [MongoDB](https://zatrano.com/docs/mongodb).

## Boot profiles


| API                     | `APP_BOOT`    | Boots                               |
| ----------------------- | ------------- | ----------------------------------- |
| `CoreApp()`             | `core`        | Kernel only                         |
| `MinimalApp()`          | `minimal`     | Foundation + your routes, no addons |
| `App()`                 | `app`         | Minimal + `EnabledAddons`           |
| `APIApp()` / `WebApp()` | `api` / `web` | Lean presets                        |
| `DemoApp()`             | `demo`        | Full demo addon set                 |


```go
import (
    "github.com/zatrano/framework/bootstrap"
    "github.com/zatrano/framework/packages/auth"
)

func main() {
    app := bootstrap.FromEnv()       // reads APP_BOOT (default "app")
    // app := bootstrap.FromEnv("demo")  // CLI default when unset
    // app := bootstrap.APIApp()

    _ = auth.From(app)               // resolve services — not app.Auth()
    app.Run()
}
```

Full guide: [Boot Profiles](https://zatrano.com/docs/boot-profiles).

## Package ecosystem

```bash
go run ./cmd/zatrano package:list
go run ./cmd/zatrano package:list --libraries
go run ./cmd/zatrano package:status
go run ./cmd/zatrano package:enable oauth
go run ./cmd/zatrano package:preset api --merge
go run ./cmd/zatrano package:doctor
```


| Kind         | Enable?                        | Examples                                                       |
| ------------ | ------------------------------ | -------------------------------------------------------------- |
| **Service**  | Yes — `EnabledAddons` / preset | `oauth`, `billing`, `ai`, `social`                             |
| **Library**  | No — just `import`             | `collection`, `totp`, `support`                                |
| **Heavy**    | Only when needed               | `webauthn`, `qr` (separate modules); DB engines via `db:setup` |
| **Database** | `db:setup`                     | `sqlite`, `mysql`, `pgsql`, `mssql`, `oracle`, `mongo`         |


Resolve from the container:

```go
auth.From(app)
auth.Passwords(app)
notification.From(app)
session.From(app)
database.Migrator(app)
mongo.From(app) // nil unless mongo is in DB_CONNECTIONS / db:setup (or legacy addon)
```

Guide: [PACKAGES.md](PACKAGES.md) (what + how) · [Package Ecosystem](https://zatrano.com/docs/package-ecosystem) · [Resolving Services](https://zatrano.com/docs/accessors).

## Authentication (highlights)

```bash
go run ./cmd/zatrano make:auth
go run ./cmd/zatrano make:dashboard
```

Session guards with **per-guard keys**, remember-me, password reset (no enumeration), email verification events, cache-backed lockout, TOTP MFA, **remember this device**, multi-device logout, intended URLs, password confirmation.

`make:dashboard` scaffolds a modular admin shell (`/dashboard`), optional users / notifications / roles / RBAC / settings / analytics / impersonate, and JSON under `/api/v1` (with auth at `/api/v1/auth`). Not pre-installed in the default app — run the CLI and follow its next steps.

Reset / verify / password-changed emails use `auth.mail_*` keys under `APP_LOCALE` (see `lang/*/auth.json` and framework defaults).

```go
ok, err := auth.From(app).Attempt(req, map[string]string{
    "email": "ada@example.com", "password": "secret",
}, true)
ok, err = auth.From(app).ChallengeTwoFactor(req, code, true) // trusted device
```

Guides: [Authentication](https://zatrano.com/docs/authentication) · [Dashboard Scaffold](https://zatrano.com/docs/dashboard-scaffold).

## What ships in the box


| Area                | Capabilities                                                                                                           |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| **HTTP**            | Router, controllers, middleware (CSRF, CORS, throttle, security), requests/responses, cookies, flash, URLs             |
| **Views**           | Layouts, components, Blade-like directives, markdown, file-based pages                                                 |
| **Validation**      | Rules, form requests, error bags                                                                                       |
| **Data**            | Query builder, schema, migrations, ORM (relations, scopes, eager load), factories, seeders                             |
| **Auth & security** | Guards, 2FA, lockout, API tokens, encryption, hashing, honeypot, trusted proxies, OAuth server, social login, WebAuthn, `make:dashboard` admin shell |
| **Async**           | Queues (sync/DB/Redis), notifications (database/mail/SMS/push), broadcasting, scheduler                                |
| **Platform**        | Config + `.env`, sessions, cache, filesystem, localization, health, maintenance, backups                               |
| **Tooling**         | CLI, OpenAPI, GraphQL helpers, docs engine, inspector, observability hooks                                             |
| **Addons**          | MongoDB, billing (Stripe), AI chat, search, webhooks, sitemaps, short URLs, …                                          |




## Project layout

```text
app/                 Controllers, providers, models
bootstrap/           Boot profiles, foundation, EnabledAddons, presets
cmd/zatrano/         CLI entrypoint
config/              Config maps (+ published addon stubs)
database/            Migrations, seeders, factories
routes/              web, api, …
views/               Templates
packages/            First-party packages (framework module)
core/                Thin kernel
```



## Documentation

All website guides are on **[https://zatrano.com/docs](https://zatrano.com/docs)** — this repository has no `docs/` tree.

In-repo packages guide: **[PACKAGES.md](PACKAGES.md)** (purpose, enable, usage, docs links).


| Start here                                                      |                                  |
| --------------------------------------------------------------- | -------------------------------- |
| [PACKAGES.md](PACKAGES.md)                                      | Packages guide (this repo)       |
| [Installation](https://zatrano.com/docs/installation)           | Clone, key, serve                |
| [Boot Profiles](https://zatrano.com/docs/boot-profiles)         | `APP_BOOT`, Minimal / API / Demo |
| [Package Ecosystem](https://zatrano.com/docs/package-ecosystem) | enable, presets, doctor          |
| [Resolving Services](https://zatrano.com/docs/accessors)        | `From(app)` migration table      |
| [Authentication](https://zatrano.com/docs/authentication)       | Full auth surface                |
| [Release Notes](https://zatrano.com/docs/releases)              | Version history                  |




## Breaking changes in 1.0

- First-party imports: `core/X` → `packages/X` (kernel stays `core`)
- Prefer `auth.From(app)`, `notification.From(app)`, … — foundation accessors removed from `Application`
- Boot is profile-driven (`APP_BOOT` / `bootstrap.*App()`); optional services via addons

Upgrade: `go get github.com/zatrano/framework@v1.0.0` then fix imports and resolvers. See [Release Notes](https://zatrano.com/docs/releases).

## Versioning

Semantic versioning. Tags: `vMAJOR.MINOR.PATCH` on [GitHub Releases](https://github.com/zatrano/framework/releases).


| Line       | Meaning                                   |
| ---------- | ----------------------------------------- |
| **v1.x**   | Thin kernel + package ecosystem (current) |
| **v0.2.x** | Pre-ecosystem line (historical)           |


## Community

- Docs: [zatrano.com/docs](https://zatrano.com/docs)
- Source: [github.com/zatrano/framework](https://github.com/zatrano/framework)
- LinkedIn: [linkedin.com/company/zatrano](https://www.linkedin.com/company/zatrano)



## Contributing

Issues and PRs welcome. Keep changes focused, add tests when practical, follow `gofmt` and existing package boundaries. Guide: [Contributing](https://zatrano.com/docs/contributions).

## Security

CI (`.github/workflows/security.yml`) runs on `main` pushes, pull requests, and a weekly cron:

| Tool | Role |
|------|------|
| **go vet** / **go test -race** | Compiler checks and data-race detection |
| **gosec** | Go SAST; SARIF uploaded to the GitHub Security tab (excludes in `.github/gosec.json`) |
| **govulncheck** | Known Go module CVEs |
| **Semgrep** | `p/golang` + `p/security-audit` plus `.github/semgrep/zatrano-rules.yml` |
| **Trivy FS** | Dependency/secret/misconfig scan (fails on HIGH/CRITICAL only) |
| **Go fuzz** | `tests/fuzz` targets (3m each in CI; longer locally) |

Report vulnerabilities privately to Serhan KARAKOÇ — [serhankarakoc@zatrano.com](mailto:serhankarakoc@zatrano.com).

## License

[MIT](LICENSE) · Copyright (c) 2026 Serhan KARAKOÇ