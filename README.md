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

ZATRANO is a Golang application platform with a **thin kernel**. This repository boots a process and a secure HTTP surface. Session, auth, database, views, queues, and AI are not in the kernel — they live in [`github.com/zatrano/packages`](https://github.com/zatrano/packages) and exist in *your* process only when you import them.

Create applications with `zatrano new`. Do not clone this repo as a project.

```text
                    your app
                       │
              bootstrap.App()
                       │
              ┌────────┴────────┐
              │                 │
           CONTRACTS          KERNEL
              │                 │
         stable ABI         primitives
                                │
                     ┌──────────┼──────────┐
                     │          │          │
                 Foundation  Intelligence  Addons
                     │          │          │
                  session       AI        database
                  auth          RAG       queue
                  validation    agent     notification
```

## Two modules

| | Module | Role |
| --- | --- | --- |
| **Framework** | [`github.com/zatrano/framework`](https://github.com/zatrano/framework) | Kernel, contracts, HTTP, CLI, `zatrano new` |
| **Packages** | [`github.com/zatrano/packages`](https://github.com/zatrano/packages) | Foundation, intelligence, and other addons |

They stay separate because the packages module already requires the framework (Go would cycle). Addon **names** are listed here for discovery ([`kernel/catalog.go`](kernel/catalog.go) is primitives only). Addon **code** is not in this tree.

## How it boots

`bootstrap.App()` always starts the kernel. Then it registers every addon this process **blank-imported**. A new app has no session, no database, no views, no auth until those imports exist.

```go
import (
    _ "github.com/zatrano/packages/session"
    _ "github.com/zatrano/packages/auth"
    _ "github.com/zatrano/packages/database"
    _ "github.com/zatrano/packages/view"
)

app := bootstrap.App(bootstrap.WithProviders(providers.All()...))
_ = auth.From(app) // container — not app.Auth()
if err := app.Run(":8080"); err != nil {
    panic(err)
}
```

`package:enable` writes the blank-import (and `go get` when needed). `App(WithAddons("auth"))` selects a subset of what is already imported. Missing hard dependencies fail boot; optional ones are skipped.

| Kind | How it gets in | Examples |
| --- | --- | --- |
| **Kernel** | Always on | HTTP, router, config, container, middleware |
| **Service addon** | Blank-import | `auth`, `database`, `queue`, `ai` |
| **Library addon** | Normal `import` — never enable | `collection`, `totp`, `resources` |
| **Heavy** | Own module, only when needed | `webauthn`, `mongo`, `qr` |

What each package is for: **[PACKAGES.md](PACKAGES.md)**.

## Contracts vs kernel

`contracts` is the stable ABI: `App`, `Provider`, `Container`, `Router`. It does not import `kernel/*` or `github.com/zatrano/packages`. Handler and middleware values stay untyped (`any`) so contracts never pull `kernel/http`.

Typed APIs sit next to the implementation:

```go
import (
    "github.com/zatrano/framework/kernel/http"
    "github.com/zatrano/framework/kernel/routing"
)

r := routing.From(app)
r.Get("/health", func(req *http.Request) *http.Response {
    return http.JSON(map[string]any{"ok": true})
})
```

After bootstrap the container, config, and router **freeze**. Route registration, config writes, and new bindings panic. `Make` may still publish lazy singletons. Optional workers implement `contracts.LifecycleProvider`; `Start` / `Stop` run them, and `Run` ties them to process lifetime.

## This repository

```text
kernel/         Application + primitives
  http/         Request / response
  routing/      Router (use routing.From)
  middleware/   CSRF, CORS, security, …
  config/ env/  Configuration and .env
  cookie/       Cookie jar
  support/      Strings, UUID, helpers
  dirs/         Application path helpers
contracts/      Public ABI
bootstrap/      App() and the addon registry
console/        Framework CLI (new, make:*, package:*, describe)
cmd/zatrano/    CLI entrypoint
tests/          Architecture, boot, compatibility, fuzz
```

Application code (`app/`, routes, views, migrations) lives in apps created by `zatrano new`. Generated apps use `cmd/app`. This repo’s CLI is `cmd/zatrano`.

## Quick start

Requires **Golang 1.25+**.

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

Kernel-only scaffold (no packages module in `go.mod`):

```bash
go run ./cmd/zatrano new lite --minimal --replace .
```

Development line as a module:

```bash
go get github.com/zatrano/framework@v2-dev
go get github.com/zatrano/packages@v2-dev
```

```bash
go run ./cmd/app package:list
go run ./cmd/app package:enable auth
go run ./cmd/app package:doctor
```

The Docker image in this repo is the `zatrano` CLI. Application images come from `zatrano new`. Optional database services are in `docker-compose.yml`.

## HTTP

Controllers use kernel types. Routes are registered on the typed router in the generated app (`app/routes/{web,api}`).

```go
package web

import "github.com/zatrano/framework/kernel/http"

type HomeController struct{}

func (c *HomeController) Index(req *http.Request) *http.Response {
    return http.JSON(map[string]any{"ok": true})
}
```

CSRF, CORS, trusted proxies, request-id, and exception handling are kernel middleware. Bodies are size-capped and replayable.

## Data and auth

Nothing is linked until you opt in.

```bash
go run ./cmd/app db:setup --drivers=sqlite --yes
go run ./cmd/app make:auth
```

SQLite is not special — install it the same way as MySQL or PostgreSQL. Drivers live under [`github.com/zatrano/packages/database/driver`](https://github.com/zatrano/packages). Resolve auth with `auth.From(app)`, not a method on `App`.

Guides: [Database](https://zatrano.com/docs/database) · [ORM](https://zatrano.com/docs/orm) · [Authentication](https://zatrano.com/docs/authentication).

## Line

Work for v2 lands on **`v2-dev`** here and **`v2-dev`** in `zatrano/packages`. Current version is `2.0.0-dev` ([`VERSION`](VERSION)). Semantic tags (`vMAJOR.MINOR.PATCH`) are published on [GitHub Releases](https://github.com/zatrano/framework/releases) when a line is cut.

v1 remains available as tagged releases. New apps should follow this branch.

## Documentation

| | |
| --- | --- |
| [zatrano.com/docs](https://zatrano.com/docs) | Product guides |
| [PACKAGES.md](PACKAGES.md) | What each package is for |
| [Installation](https://zatrano.com/docs/installation) | Clone, key, serve |
| [Resolving services](https://zatrano.com/docs/accessors) | `From(app)` |
| [Contributing](CONTRIBUTING.md) | PRs and tests |

## Security

CI (`.github/workflows/security.yml`) runs **go vet**, **go test -race**, **gosec**, **govulncheck**, **Semgrep**, **Trivy FS**, and fuzz in `tests/fuzz`.

Report vulnerabilities privately to Serhan KARAKOÇ — [serhankarakoc@zatrano.com](mailto:serhankarakoc@zatrano.com). Do not open a public issue.

## License

[MIT](LICENSE) · Copyright (c) 2026 Serhan KARAKOÇ

- [zatrano.com](https://zatrano.com/docs)
- [github.com/zatrano/framework](https://github.com/zatrano/framework)
- [github.com/zatrano/packages](https://github.com/zatrano/packages)
- [linkedin.com/company/zatrano](https://www.linkedin.com/company/zatrano)
