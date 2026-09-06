<p align="center">
  <strong>ZATRANO</strong>
</p>

<p align="center">
  <em>A Go application platform for building production-grade software.</em>
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
  <a href="VERSION"><img src="https://img.shields.io/badge/version-2.0.0-green.svg" alt="Version"></a>
  <a href=".github/SECURITY.md"><img src="https://img.shields.io/badge/security-policy-brightgreen.svg" alt="Security Policy"></a>
</p>

<p align="center">
  <a href="https://zatrano.com/docs">Documentation</a>
  ·
  <a href="https://zatrano.com/docs/installation">Installation</a>
  ·
  <a href="PACKAGES.md">Packages</a>
  ·
  <a href="https://github.com/zatrano/packages">Package ecosystem</a>
  ·
  <a href="https://github.com/zatrano/framework/releases">Releases</a>
</p>

---

## What is ZATRANO?

ZATRANO is a **Go application platform** for building, running, and extending production-grade software.

It is built around a small, dependency-neutral kernel and an opt-in package ecosystem. The kernel is the stable runtime foundation. Packages add capabilities such as databases, authentication, sessions, queues, notifications, AI, RAG, agents, billing, OAuth, and other application services.

You import what you run. The kernel has **zero third-party runtime dependencies**.

```text
                           ZATRANO
                    Go Application Platform
                              │
             ┌────────────────┼────────────────┐
             │                │                │
          Contracts         Kernel          Packages
             │                │                │
        Stable public      Runtime          Optional
            ABI            foundation      capabilities
             │                │                │
             │        ┌───────┼───────┐        │
             │        │       │       │        │
             │      HTTP   Router  Config     AI
             │        │       │       │        │
             │   Middleware Container …       RAG
             │                                Agent
             │                                Auth
             │                                Database
             │                                Queue
             │                                …
             │
             └──────────── Application ────────────┘
                              │
                         bootstrap.App()
                              │
                              ▼
                         Your application
```

ZATRANO is not an application skeleton, and it is not a monolith where every capability is built into the core. The platform is modular by design.

This repository (`github.com/zatrano/framework`) is the **platform runtime**: kernel, contracts, bootstrap, and CLI. It is not intended to be cloned and used as your application. Create applications with `zatrano new`.

## Architecture

ZATRANO separates the stable runtime foundation from optional application capabilities.

### Kernel

The kernel contains the primitives required to run an application:

- Application lifecycle
- HTTP request / response
- Routing
- Middleware
- Dependency container
- Configuration
- Environment
- Encryption
- Cookies
- Logging
- Exceptions
- Reports
- Trusted proxies
- Safe paths
- Core support utilities

The kernel does not depend on the packages module.

### Contracts

`contracts` is ZATRANO's dependency-neutral public ABI. It contains stable interfaces such as `App`, `Provider`, `LifecycleProvider`, `Container`, `Router`, and the HTTP bridge.

The contracts package does not import kernel implementation packages or the packages module. That keeps the ABI stable and prevents dependency cycles.

Typed developer APIs live next to their implementations:

```go
import (
    "github.com/zatrano/framework/kernel/http"
    "github.com/zatrano/framework/kernel/routing"
)

r := routing.From(app)

r.Get("/health", func(req *http.Request) *http.Response {
    return http.JSON(map[string]any{
        "ok": true,
    })
})
```

The distinction is intentional:

```text
contracts
    ↓
stable, dependency-neutral ABI

kernel/*
    ↓
strongly typed implementation

From(app)
    ↓
typed developer facade
```

### Packages

Optional capabilities live in the separate [`github.com/zatrano/packages`](https://github.com/zatrano/packages) module.

Examples include sessions, validation, authentication, database, ORM, views, queues, notifications, scheduler, localization, cache, AI, RAG, agents, OAuth, social login, WebAuthn, billing, backups, OpenAPI, GraphQL, database drivers, and other import-only libraries.

Packages are enabled only when an application needs them.

```text
Kernel
  │
  ├── always available
  │
  └── no optional application services

Packages
  │
  ├── session
  ├── auth
  ├── database
  ├── queue
  ├── notification
  ├── ai
  ├── rag
  ├── agent
  └── …
```

A minimal application does not pay for capabilities it does not use.

## Two modules

| Module | Role |
| --- | --- |
| [`github.com/zatrano/framework`](https://github.com/zatrano/framework) | Platform runtime: kernel, contracts, bootstrap, CLI |
| [`github.com/zatrano/packages`](https://github.com/zatrano/packages) | Optional application services and libraries |

The packages module depends on this module. This module does not import `github.com/zatrano/packages`.

```text
Application
    │
    ├── github.com/zatrano/framework
    │
    └── selected packages
             │
             ▼
        framework/contracts
```

There is no reverse dependency from the kernel into application capabilities.

## Application model

Applications are created with:

```bash
zatrano new myapp
```

Generated applications contain application-specific structure. A typical tree looks like:

```text
myapp/
├── app/
│   ├── http/controllers/
│   ├── routes/
│   ├── providers/
│   ├── views/
│   ├── database/
│   └── …
├── bootstrap/
├── cmd/
│   └── app/
├── public/
├── storage/
├── tests/
└── go.mod
```

The exact generated structure depends on the selected application capabilities.

## Quick start

Requires **Golang 1.25+**.

```bash
git clone https://github.com/zatrano/framework.git
cd framework

go run ./cmd/zatrano new myapp --replace .
cd myapp

cp .env.example .env
go run ./cmd/app key:generate
go run ./cmd/app serve
```

Open [http://localhost:8080](http://localhost:8080).

Use the modules directly:

```bash
go get github.com/zatrano/framework@main
go get github.com/zatrano/packages@main
```

Kernel-only scaffold (no package ecosystem):

```bash
go run ./cmd/zatrano new lite --minimal --replace .
```

## Enabling packages

Service packages register providers with the platform. An application enables them through blank imports:

```go
import (
    _ "github.com/zatrano/packages/session"
    _ "github.com/zatrano/packages/auth"
    _ "github.com/zatrano/packages/database"
    _ "github.com/zatrano/packages/view"
)
```

`bootstrap.App()` starts the kernel, then every package this process blank-imported. Resolve services with typed `From(app)` helpers — they are not methods on `App`:

```go
authService := auth.From(app)
```

Do not expect `app.Auth()`, `app.Database()`, or `app.AI()`. That would turn the central application object into a dependency-heavy service locator.

```text
Application
    │
    └── Container
          │
          ├── auth
          ├── database
          ├── session
          ├── ai
          └── …
```

Each package owns its typed developer API.

### Package management

```bash
go run ./cmd/app package:list
go run ./cmd/app package:enable auth
go run ./cmd/app package:doctor
```

Catalog: **[PACKAGES.md](PACKAGES.md)**. The package ecosystem is maintained separately from the kernel.

## HTTP

The kernel provides the HTTP runtime. Controllers use strongly typed kernel HTTP primitives:

```go
package web

import "github.com/zatrano/framework/kernel/http"

type HomeController struct{}

func (c *HomeController) Index(req *http.Request) *http.Response {
    return http.JSON(map[string]any{
        "ok": true,
    })
}
```

Routes are defined in the generated application:

```go
r.Get("/", controller.Index)
```

Kernel middleware provides the HTTP security and infrastructure layer, including CSRF, CORS, security headers, trusted proxies, request IDs, exception handling, method override, request limits, and safe static-file resolution.

## Routing

The router is mutable during application registration and immutable after bootstrap.

```text
Registration
     │
     ▼
 Mutable route graph
     │
     ▼
   Freeze()
     │
     ▼
 Immutable runtime graph
     │
     ├── dispatch
     ├── snapshot
     └── cache
```

After freezing, route registration and mutation are rejected. Typed routing APIs are provided by `routing.From(app)`.

## Application lifecycle

Providers can implement `contracts.LifecycleProvider`:

```go
Start(app contracts.App) error
Stop(ctx context.Context) error
```

`Start` / `Stop` (and `Run`) own process lifetime. Lifecycle transitions are serialized and protected against concurrent `Start` / `Stop` calls.

```text
Created
   │
   ▼
Bootstrapping
   │
   ├── fail → BootFailed (terminal)
   ▼
Booted
   │
   ▼
Starting
   │
   ▼
Running
   │
   ▼
Stopping
   │
   ▼
Stopped (terminal)
```

Failed bootstrap is terminal for that application instance. Stopped applications cannot restart.

## Dependency container

The kernel includes a concurrency-safe dependency container. It supports bindings, singletons, instances, aliases, lazy resolution, cycle detection, concurrent singleton initialization, and frozen registration.

The public contract exposes the essential resolver operations without exposing the implementation:

```go
value, err := app.Container().Make("service")
```

After application bootstrap, registration is frozen. Frozen means registration is immutable; it does not mean that runtime singleton instances can never be initialized.

## Configuration

Configuration is isolated from application ownership. The repository protects callers from accidental aliasing by recursively copying supported mutable configuration structures:

```text
primitive
map
  └── recursive
slice
  └── recursive
```

Pointer and arbitrary struct values are not cloned. Copy semantics cover the configuration value graph managed by the repository, not a universal Go object cloner.

After bootstrap, the configuration repository is frozen.

## CLI

This repository's CLI entrypoint is `cmd/zatrano`. Generated applications use `cmd/app`.

```bash
zatrano new
zatrano serve
zatrano make:*
zatrano package:list
zatrano package:enable
zatrano package:doctor
zatrano db:setup
zatrano describe
```

## Repository structure

```text
kernel/            Application + primitives
  http/            HTTP request / response
  routing/         Router
  middleware/      HTTP middleware
  config/          Configuration
  container/       Dependency container
  context/         Application context
  env/             Environment
  encryption/      Encryption primitives
  cookie/          Cookie handling
  exceptions/      Exception handling
  log/             Logging
  pipeline/        Pipelines
  report/          Reporting
  safepath/        Safe path resolution
  trustedproxy/    Trusted proxy handling
  support/         Support utilities
contracts/         Public dependency-neutral ABI
bootstrap/         Application boot and package registry
console/           Platform CLI commands
cmd/zatrano/       CLI entrypoint
tests/             Architecture, compatibility, boot, and fuzz tests
```

The packages ecosystem is maintained in the separate [packages](https://github.com/zatrano/packages) repository.

## Security

Security is a platform concern. CI runs **go vet**, **go test -race**, **gosec**, **govulncheck**, **Semgrep**, **Trivy**, and Go fuzzing.

The release gate:

```bash
bash .github/scripts/release-gate.sh
```

Optional extended checks:

```bash
RACE=1 bash .github/scripts/release-gate.sh
FUZZ=1 bash .github/scripts/release-gate.sh
```

Kernel security primitives include request size limits, safe path resolution, secure request IDs, security headers, trusted proxy handling, production secret validation, exception isolation, and cookie protection.

Report vulnerabilities privately to Serhan KARAKOÇ — [serhankarakoc@zatrano.com](mailto:serhankarakoc@zatrano.com). Do not open a public GitHub issue for security reports.

## Architecture principles

1. **Small kernel** — primitives, not every application feature.
2. **Dependency-neutral contracts** — the ABI does not import kernel implementation packages or optional packages.
3. **Opt-in capabilities** — application capabilities are packages.
4. **Typed developer APIs** — the public developer experience stays strongly typed even where the ABI must remain untyped.
5. **One-way dependency flow** — application → packages → framework contracts / kernel. The kernel never depends on the packages ecosystem.
6. **Immutable runtime configuration** — registration structures are mutable during boot and frozen before runtime.
7. **Explicit application lifecycle** — startup and shutdown are deterministic and concurrency-safe.
8. **Platform, not monolith** — the runtime foundation and the capability ecosystem without forcing every application to use every subsystem.

```text
Application
    ↓
Packages
    ↓
Framework contracts / kernel
```

## v2

**v2.0.0** is the current line on **`main`**. Version: `2.0.0` ([`VERSION`](VERSION)).

The v2 line is two independently maintained modules: `github.com/zatrano/framework` and `github.com/zatrano/packages`. Create applications with `zatrano new`. Do not clone this repository as your application.

```text
ZATRANO Platform
      │
      ├── Kernel
      ├── Contracts
      ├── Bootstrap
      ├── CLI
      └── Package ecosystem
```

| Line | Meaning |
| --- | --- |
| `v2.0.0` (`main`) | Current two-module application platform |
| `v1.x` | Previous tagged ZATRANO line |

ZATRANO follows semantic versioning: `vMAJOR.MINOR.PATCH`. The Go module path remains `github.com/zatrano/framework` (no `/v2` suffix). Generated apps use `v0.0.0` plus a `replace` until that path change.

## Documentation

- [zatrano.com/docs](https://zatrano.com/docs)
- [Installation](https://zatrano.com/docs/installation)
- [PACKAGES.md](PACKAGES.md)
- [github.com/zatrano/packages](https://github.com/zatrano/packages)
- [Releases](https://github.com/zatrano/framework/releases)

## Community

Issues and pull requests are welcome. Keep changes focused, preserve architectural boundaries, add tests for behavioral changes, and run `gofmt`.

## License

[MIT](LICENSE) · Copyright (c) 2026 Serhan KARAKOÇ

- [zatrano.com](https://zatrano.com/docs)
- [github.com/zatrano/framework](https://github.com/zatrano/framework)
- [github.com/zatrano/packages](https://github.com/zatrano/packages)
- [linkedin.com/company/zatrano](https://www.linkedin.com/company/zatrano)
