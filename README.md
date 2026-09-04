<p align="center">
  <strong>ZATRANO</strong>
</p>

<p align="center">
  <em>A Golang web framework with a thin kernel. You import what you run.</em>
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
  <a href="https://zatrano.com/docs">Docs</a>
  ·
  <a href="https://zatrano.com/docs/installation">Install</a>
  ·
  <a href="PACKAGES.md">Package guide</a>
  ·
  <a href="https://github.com/zatrano/packages">Addons</a>
  ·
  <a href="https://github.com/zatrano/framework/releases">Releases</a>
</p>

---

ZATRANO started in **February 2018** as the internal spine of real products — logins, migrations, queues, mail, production uptime. **v1** opened that stack under MIT in August 2026. **v2** is the same lineage, cut clean: this repository is only the kernel. Applications are created with `zatrano new`. Everything optional lives in [`github.com/zatrano/packages`](https://github.com/zatrano/packages) and is linked only when you import it.

## Two modules

| | Module | What it is |
| --- | --- | --- |
| **Framework** | [`github.com/zatrano/framework`](https://github.com/zatrano/framework) | Kernel, contracts, HTTP, CLI, `zatrano new` |
| **Addons** | [`github.com/zatrano/packages`](https://github.com/zatrano/packages) | Auth, database, views, queues, AI, OAuth, billing, … |

They cannot be merged: the addon module already requires the framework (Go import cycle). Addon **names** are listed in [`kernel/catalog.go`](kernel/catalog.go) for discovery. Addon **code** is not in this tree.

```text
  your app  (zatrano new)
       │
       │  blank-import  github.com/zatrano/packages/<name>
       ▼
  bootstrap.App()
       │
       ├─ kernel     always on
       │              Application, http, routing, middleware,
       │              config, env, encryption, cookie, support
       │
       └─ addons     only what this process imported
                      session, auth, database, view, queue, ai, …
```

A new app has **no session, no database, no views, no auth** until you import those packages. `bootstrap.App()` boots the kernel plus every package the process blank-imported. That is the whole rule.

## This repository

```text
kernel/         Application + primitives
  http/         Request / response
  routing/      Router
  middleware/   CSRF, CORS, security, …
  config/ env/  Configuration and .env
  cookie/       Cookie jar
  support/      Strings, UUID, helpers
  layout/       app/views vs views/ path helpers
contracts/      Public interfaces (App, Provider, …)
bootstrap/      App() and the addon registry
console/        Framework CLI (new, make:*, package:*, describe)
cmd/zatrano/    CLI entrypoint
tests/          Integration fixtures
```

Import kernel types from under `kernel/`:

```go
import "github.com/zatrano/framework/kernel/http"
import "github.com/zatrano/framework/kernel/routing"
import "github.com/zatrano/framework/contracts"
```

There is no application skeleton here. `app/`, `app/routes`, `app/views`, and `app/database` belong to apps created by `zatrano new`. Generated apps use `cmd/app`; this repo’s CLI is `cmd/zatrano`.

## Requirements

- Golang **1.25+**
- A database only if you opt in: SQLite, MySQL, PostgreSQL, SQL Server, Oracle, MongoDB via `db:setup`
- Redis, Stripe, OpenAI — only if you import those addons

## Quick start

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

Kernel-only scaffold (no addon module in `go.mod`):

```bash
go run ./cmd/zatrano new lite --minimal --replace .
```

As a module (development line):

```bash
go get github.com/zatrano/framework@v2-dev
```

Tagged v1 releases stay on [GitHub Releases](https://github.com/zatrano/framework/releases).

The Docker image in this repo is the `zatrano` CLI. Application images are generated with `zatrano new`. Optional database services are in `docker-compose.yml`.

## Turning packages on

In the **application**, blank-import the addon. Its `init()` registers a provider. `App()` loads every registered provider.

```go
// bootstrap/addons.go  (package:enable writes this)
import (
    _ "github.com/zatrano/packages/session"
    _ "github.com/zatrano/packages/auth"
    _ "github.com/zatrano/packages/database"
    _ "github.com/zatrano/packages/view"
)

func main() {
    app := bootstrap.App(bootstrap.WithProviders(providers.All()...))
    _ = auth.From(app) // resolve from the container — not app.Auth()
    app.Run(":8080")
}
```

```bash
go get github.com/zatrano/packages@v2-dev
go run ./cmd/app package:list
go run ./cmd/app package:enable auth
go run ./cmd/app package:doctor
```

| Kind | How you use it | Examples |
| --- | --- | --- |
| **Kernel** | Always on | `http`, `routing`, `middleware`, `config` |
| **Service addon** | Blank-import (or `package:enable`) | `auth`, `database`, `queue`, `ai`, `oauth` |
| **Library addon** | Normal `import` — never enable | `collection`, `totp`, `resources` |
| **Heavy** | Own Go module, only when needed | `webauthn`, `mongo`, `qr` |

There is no `mail` package. Send email through `notification` with `Channels: ["mail"]`.

What each package is for: **[PACKAGES.md](PACKAGES.md)**. Addon module: **[zatrano/packages](https://github.com/zatrano/packages)**.

## Databases

No driver is linked by default. SQLite is not special — install it the same way as MySQL:

```bash
go run ./cmd/app db:setup
go run ./cmd/app db:setup --drivers=sqlite --yes
go run ./cmd/app db:setup --drivers=sqlite,mysql,pgsql,mongo --default=mysql --yes
```

Drivers live in [`github.com/zatrano/packages/database/driver/...`](https://github.com/zatrano/packages).

```env
DB_CONNECTION=mysql
DB_CONNECTIONS=mysql,pgsql,mongo
DB_MYSQL_HOST=127.0.0.1
DB_MYSQL_DATABASE=shop
DB_PGSQL_HOST=127.0.0.1
DB_MONGO_URI=mongodb://localhost:27017
```

A model can pin a connection:

```go
func (m *Order) Connection() string { return "pgsql" }
```

```bash
go run ./cmd/app make:model Order --connection=pgsql
```

Guides: [Database](https://zatrano.com/docs/database) · [ORM](https://zatrano.com/docs/orm) · [MongoDB](https://zatrano.com/docs/mongodb).

## HTTP

Controllers talk to kernel types:

```go
package web

import "github.com/zatrano/framework/kernel/http"

type HomeController struct{}

func (c *HomeController) Index(req *http.Request) *http.Response {
    return http.JSON(map[string]any{"ok": true})
}
```

Routes live under `app/routes/{web,api}` in the generated app. CSRF, CORS, trusted proxies, and exception handling are kernel middleware.

## Authentication

```bash
go run ./cmd/app make:auth
go run ./cmd/app make:dashboard
```

Session guards, remember-me, password reset, email verification, lockout, TOTP, trusted devices, multi-device logout. `make:dashboard` scaffolds `/dashboard` and optional JSON under `/api/v1` — not preinstalled.

```go
ok, err := auth.From(app).Attempt(req, map[string]string{
    "email": "ada@example.com", "password": "secret",
}, true)
```

Guides: [Authentication](https://zatrano.com/docs/authentication) · [Dashboard](https://zatrano.com/docs/dashboard-scaffold).

## What you get

**Kernel (this module)** — process start and a secure HTTP surface: container, config, env, router, request/response, middleware, encryption, cookies, logging, exceptions.

**Addons (packages module)** — sessions, validation, views, auth, ORM, queues, notifications, scheduler, localization, cache, AI (`ai`, `rag`, `agent`), plus opt-in services (OAuth, social login, WebAuthn, billing, backups, OpenAPI, GraphQL) and import-only libraries.

**CLI** — `zatrano new`, `serve`, `make:*`, `package:enable|list|doctor`, `db:setup`, `describe`.

## v2

Unreleased (`v2-dev`). Create apps with `zatrano new`; do not clone this repo as a project.

- Primitives live under `kernel/` (`kernel/http`, `kernel/routing`, …). The module root is `kernel`, `contracts`, `bootstrap`, `console`, `cmd`.
- Auth, database, views, queues, AI, and the rest are addons. Blank-import them.
- `bootstrap.ApplicationProviders()` is empty; pass `WithProviders` from the app.
- No database until `db:setup`.
- Demo boot profiles are gone. Import what you want.

Upgrade: generate a new app, or move your tree out of a framework clone, then `go get github.com/zatrano/framework@v2-dev` and blank-import the addons you still need. [Release notes](https://zatrano.com/docs/releases).

## Versioning

Semantic versioning. Tags: `vMAJOR.MINOR.PATCH` on [GitHub Releases](https://github.com/zatrano/framework/releases).

| Line | Meaning |
| --- | --- |
| **v2-dev** | Two-module architecture (this branch) |
| **v1.x** | Last tagged line (`v1.6.6`) |

v2 work lands on **`v2-dev`** here and **`v2-dev`** in `zatrano/packages`.

## Documentation

| | |
| --- | --- |
| [zatrano.com/docs](https://zatrano.com/docs) | Product guides |
| [PACKAGES.md](PACKAGES.md) | What each package is for |
| [Installation](https://zatrano.com/docs/installation) | Clone, key, serve |
| [Package ecosystem](https://zatrano.com/docs/package-ecosystem) | Enable, doctor |
| [Resolving services](https://zatrano.com/docs/accessors) | `From(app)` |
| [Contributing](https://zatrano.com/docs/contributions) | PRs and tests |

## Community

- [zatrano.com/docs](https://zatrano.com/docs)
- [github.com/zatrano/framework](https://github.com/zatrano/framework)
- [github.com/zatrano/packages](https://github.com/zatrano/packages)
- [linkedin.com/company/zatrano](https://www.linkedin.com/company/zatrano)

Issues and PRs welcome. Keep changes focused, add tests when practical, follow `gofmt`.

## Security

CI (`.github/workflows/security.yml`) runs on `main` pushes, pull requests, and a weekly cron: **go vet**, **go test -race**, **gosec**, **govulncheck**, **Semgrep**, **Trivy FS**, and Golang fuzz in `tests/fuzz`.

Report vulnerabilities privately to Serhan KARAKOÇ — [serhankarakoc@zatrano.com](mailto:serhankarakoc@zatrano.com).

## License

[MIT](LICENSE) · Copyright (c) 2026 Serhan KARAKOÇ
