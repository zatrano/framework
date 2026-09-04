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

ZATRANO is a Golang web framework with a **thin kernel**. This repository is the kernel: process boot, HTTP, routing, config, and the stable ABI. Everything else — session, auth, database, views, queues, AI — lives in [`github.com/zatrano/packages`](https://github.com/zatrano/packages) and is linked only when you import it.

The kernel has **zero third-party runtime dependencies**.

Create applications with `zatrano new`. This repository is the framework, not a project skeleton.

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
```

## Two modules

| | Module | Role |
| --- | --- | --- |
| **Framework** | [`github.com/zatrano/framework`](https://github.com/zatrano/framework) | Kernel, contracts, HTTP, CLI |
| **Packages** | [`github.com/zatrano/packages`](https://github.com/zatrano/packages) | Opt-in services and libraries |

`bootstrap.App()` starts the kernel, then every package this process blank-imported. Resolve services with `From(app)` helpers — they are not methods on `App`.

```go
import (
    _ "github.com/zatrano/packages/session"
    _ "github.com/zatrano/packages/auth"
    _ "github.com/zatrano/packages/database"
    _ "github.com/zatrano/packages/view"
)

app := bootstrap.App(bootstrap.WithProviders(providers.All()...))
_ = auth.From(app)
if err := app.Run(":8080"); err != nil {
    panic(err)
}
```

```bash
go run ./cmd/app package:list
go run ./cmd/app package:enable auth
```

Catalog: **[PACKAGES.md](PACKAGES.md)**. Guides: [zatrano.com/docs](https://zatrano.com/docs).

## Contracts and kernel

`contracts` is the public ABI (`App`, `Provider`, `Container`, `Router`). It does not import `kernel/*` or the packages module. Typed APIs live next to their implementation:

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

After bootstrap, registration is frozen. Optional long-running work uses `contracts.LifecycleProvider`; `Start` / `Stop` (and `Run`) own process lifetime.

```text
kernel/         Application + primitives
contracts/      Public ABI
bootstrap/      App() and the addon registry
console/        CLI (new, make:*, package:*, describe)
cmd/zatrano/    CLI entrypoint
tests/          Architecture, boot, compatibility, fuzz
```

Generated apps use `cmd/app`. This repo’s CLI is `cmd/zatrano`.

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

```bash
go get github.com/zatrano/framework@v2-dev
go get github.com/zatrano/packages@v2-dev
```

Kernel-only scaffold: `zatrano new lite --minimal --replace .`

## HTTP

```go
package web

import "github.com/zatrano/framework/kernel/http"

type HomeController struct{}

func (c *HomeController) Index(req *http.Request) *http.Response {
    return http.JSON(map[string]any{"ok": true})
}
```

Routes live in the generated app. Kernel middleware covers CSRF, CORS, trusted proxies, and exceptions.

## Line

v2 is developed on **`v2-dev`** (this branch and `zatrano/packages`). Version: `2.0.0-dev` ([`VERSION`](VERSION)).

## Security

CI runs **go vet**, **go test -race**, **gosec**, **govulncheck**, **Semgrep**, **Trivy**, and fuzz.

Report vulnerabilities privately to Serhan KARAKOÇ — [serhankarakoc@zatrano.com](mailto:serhankarakoc@zatrano.com).

## License

[MIT](LICENSE) · Copyright (c) 2026 Serhan KARAKOÇ

- [zatrano.com](https://zatrano.com/docs)
- [github.com/zatrano/framework](https://github.com/zatrano/framework)
- [github.com/zatrano/packages](https://github.com/zatrano/packages)
- [linkedin.com/company/zatrano](https://www.linkedin.com/company/zatrano)
