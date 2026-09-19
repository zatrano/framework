<p align="center">
  <strong>ZATRANO</strong>
</p>

<p align="center">
  <em>A small, typed, stable Go kernel with machine-readable contracts for humans and AI agents.</em>
</p>

<p align="center">
  <a href="https://github.com/zatrano/framework/actions/workflows/tests.yml"><img src="https://github.com/zatrano/framework/actions/workflows/tests.yml/badge.svg?branch=main" alt="Tests"></a>
  <a href="https://github.com/zatrano/framework/actions/workflows/static-analysis.yml"><img src="https://github.com/zatrano/framework/actions/workflows/static-analysis.yml/badge.svg?branch=main" alt="Static Analysis"></a>
  <a href="https://github.com/zatrano/framework/actions/workflows/coding-style.yml"><img src="https://github.com/zatrano/framework/actions/workflows/coding-style.yml/badge.svg?branch=main" alt="Coding Style"></a>
  <a href="https://github.com/zatrano/framework/actions/workflows/security.yml"><img src="https://github.com/zatrano/framework/actions/workflows/security.yml/badge.svg?branch=main" alt="Security"></a>
</p>

<p align="center">
  <a href="https://github.com/zatrano/framework/actions/workflows/security.yml"><img src="https://img.shields.io/badge/gosec-SAST-222?logo=go" alt="gosec"></a>
  <a href="https://github.com/zatrano/framework/actions/workflows/security.yml"><img src="https://img.shields.io/badge/govulncheck-CVE-222?logo=go" alt="govulncheck"></a>
  <a href="https://github.com/zatrano/framework/actions/workflows/security.yml"><img src="https://img.shields.io/badge/Semgrep-rules-222?logo=semgrep" alt="Semgrep"></a>
  <a href="https://github.com/zatrano/framework/security"><img src="https://img.shields.io/badge/Trivy-FS-222?logo=aquasecurity" alt="Trivy"></a>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/zatrano/framework/v2"><img src="https://img.shields.io/badge/golang-1.25+-00ADD8?logo=go&logoColor=white" alt="Golang"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="VERSION"><img src="https://img.shields.io/badge/version-2.6.3-green.svg" alt="Version"></a>
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

## Benchmarks

Kernel `v2.6.0` under concurrent load — full HTTP request path: routing, middleware, and AES-GCM encrypted session cookies.

<p align="center">
  <img src="https://img.shields.io/badge/requests-304K%2B-2ea44f?style=for-the-badge" alt="Total requests">
  <img src="https://img.shields.io/badge/errors-0-2ea44f?style=for-the-badge" alt="Errors">
  <img src="https://img.shields.io/badge/data_races-0-2ea44f?style=for-the-badge" alt="Data races">
  <img src="https://img.shields.io/badge/panics-0-2ea44f?style=for-the-badge" alt="Panics">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/peak-9%2C514_RPS-3498db?style=for-the-badge" alt="Peak RPS">
  <img src="https://img.shields.io/badge/burst-2%2C000%2F2%2C000_%40_397ms-3498db?style=for-the-badge" alt="Burst test">
  <img src="https://img.shields.io/badge/session_collisions-0-2ea44f?style=for-the-badge" alt="Session collisions">
  <img src="https://img.shields.io/badge/cross--user_leaks-0-2ea44f?style=for-the-badge" alt="Cross-user leaks">
</p>

<table align="center">
<tr>
<td align="center" valign="top">

**Throughput by concurrency**

| Workers | RPS |
|:---:|:---:|
| 10 | 9,514 |
| 50 | 7,837 |
| 200 | 7,138 |
| 500 | 5,704 |
| 1,000 | 5,230 |

</td>
<td align="center" valign="top">

**Sustained and burst load**

| Test | p99 | Errors |
|:---:|:---:|:---:|
| 500 workers, 100K req | 243.40 ms | ✅ 0 |
| 1,000 workers, 100K req | 505.21 ms | ✅ 0 |
| Sustained, 300w / 15s, 91K req | 120.69 ms | ✅ 0 |
| Burst, 2,000 concurrent | 360 ms | ✅ 0 |

</td>
</tr>
</table>

**Concurrency safety** — session handling (login → encrypted cookie → profile → user verification) run under `go test -race` and under every load test above: 100,000 requests through the race detector with **0 data races**, **0 session collisions**, **0 cross-user leaks**.

> Environment: 1 vCPU / 3.9 GB RAM, single core. Structural/relative measurements, not production capacity figures — multi-core benchmarking with production-like network, TLS, and database workloads is next. Full methodology and raw output: [releases](https://github.com/zatrano/framework/releases).

---

## What is ZATRANO?

ZATRANO is a **small, typed, stable kernel** plus opt-in packages. The kernel is HTTP, container, config, routing, lifecycle, and CLI. Packages bind with `From(app)` / `app.Make`. `contracts.App` does not grow package methods.

The same surface is for people and for AI agents: frozen `contracts`, plus three machine-readable CLI tools.

```bash
zatrano describe
zatrano doctor
zatrano agents:generate
```

- `describe` prints the live catalog, routing primitives, and package inventory.
- `doctor` checks the application (and this repository) against the frozen architecture rules.
- `agents:generate` writes an application-root `AGENTS.md` from `describe` so agents read the same facts the CLI does.

`ai`, `rag`, `agent`, and `workflow` are **experimental**: they have not completed the same security review as the rest of the ecosystem.

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

This repository (`github.com/zatrano/framework/v2`) is the **platform runtime**: kernel, contracts, bootstrap, CLI, generator engine, kernel `make:*` commands, and the embedded application starter. It is not intended to be cloned and used as your application. Create applications with `zatrano new`. Production-shaped consumers live in [`github.com/zatrano/examples`](https://github.com/zatrano/examples).

ZATRANO is an application platform, not merely a web toolkit.

```text
framework   runtime, contracts, CLI, generator engine, first-party scaffolds
packages    optional capabilities, package-owned make:*, stubs, config, .env fragments
examples    runnable reference applications — not templates
```

`zatrano new myapp` generates one application with HTML at `/` and JSON at `/api`. Use `http.HTML` or `http.JSON` per controller; `make:controller` and `make:controller --api` pick the tree. `health` is enabled by default. `assets`, `localization`, `view`, `validation`, and other capabilities stay opt-in via `package:enable`. First-party templates are embedded in the CLI release; `zatrano new` does not fetch templates from the network.

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
    "github.com/zatrano/framework/v2/kernel/http"
    "github.com/zatrano/framework/v2/kernel/routing"
)

r := routing.From(app)

r.Get("/health", func(req *http.Request) *http.Response {
    return http.JSON(map[string]any{
        "ok": true,
    })
})

routing.Version(r, "v1", func(api *routing.Router) {
    api.Get("/ping", ping)
})
```

`routing.Version` mounts `/api/{version}` and sets `X-API-Version`. There is no `packages/api`.

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

Examples include sessions, validation, authentication, database, ORM, views, queues, notifications, scheduler, localization, cache, AI, RAG, agents, OAuth, social login, WebAuthn, backups, OpenAPI, GraphQL, database drivers, and other import-only libraries.

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

An application does not pay for capabilities it does not enable. Database, auth, queue, and similar packages stay off until `package:enable`.

## Two modules

| Module | Role |
| --- | --- |
| [`github.com/zatrano/framework/v2`](https://github.com/zatrano/framework) | Platform runtime: kernel, contracts, bootstrap, CLI |
| [`github.com/zatrano/packages`](https://github.com/zatrano/packages) | Optional application services and libraries |

The packages module depends on this module. This module does not import `github.com/zatrano/packages`.

```text
Application
    │
    ├── github.com/zatrano/framework/v2
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
│   ├── http/
│   ├── routes/
│   └── providers/
├── bootstrap/
│   ├── addons.go      # blank-imports (process registry)
│   └── enabled.go     # enablement manifest
├── cmd/
│   └── app/
├── public/
├── storage/
├── tests/
└── go.mod
```

Default `zatrano new` generates one application: HTML at `/`, JSON at `/api`, and `health` enabled. `assets`, `localization`, `view`, `validation`, database, auth, queue, and other capabilities stay opt-in.

A framework upgrade does not regenerate application source (G-001). Generated apps record scaffold name, version, and digest in `bootstrap/scaffold.go` at `zatrano new` time only.

## Quick start

Requires **Golang 1.25+**.

Create an application from the published modules:

```bash
go install github.com/zatrano/framework/v2/cmd/zatrano@v2.6.3
zatrano new myapp
cd myapp
go mod tidy
go run ./cmd/app key:generate
go run ./cmd/app serve
```

Open [http://localhost:8080](http://localhost:8080). Default listen port is `APP_PORT` (8080).

Use the modules in an existing `go.mod`:

```bash
go get github.com/zatrano/framework/v2@v2.6.3
go get github.com/zatrano/packages@v1.10.0
```

These are the **current stable public releases**. The two modules version independently; later applications may pin newer compatible tags. There is no monolithic `zatrano@x.y.z` version.

To generate against this checkout, clone **framework** and **packages** as siblings (CI does the same), then pass the **absolute** framework path to `--replace`. `.` is wrong when the app is a subdirectory: Go resolves replace paths relative to the new module.

```bash
git clone https://github.com/zatrano/framework.git
git clone https://github.com/zatrano/packages.git
cd framework

go run ./cmd/zatrano new myapp --replace "$PWD"
cd myapp
go run ./cmd/app key:generate
go run ./cmd/app serve
```

`zatrano new` does not accept `..` in the project name. Put the app next to this clone with an absolute destination, or create it as a subdirectory as above.

## Enabling packages

Acquisition, import, and enablement are separate. `package:enable` does not acquire a module as a resolver, and `package:acquire` does not enable unless you pass `--enable`.

```text
Acquire                 go get (package:acquire) — Go modules own resolution
   ↓
Imported                blank-import in the application
   ↓
Enabled ∩ Imported      bootstrap.App() registers that intersection
   ↓
Expand Requires         declared addon metadata only (not go get)
   ↓
Bootstrap               Provider.Register + Provider.Boot
```

`bootstrap.App()` constructs the application and **registers** the intersection of the enablement list and imported packages (see `bootstrap/app.go`). It does not call `Application.Bootstrap()`:

1. `App(WithAddons(names...))` → names ∩ imported
2. else consumer `RegisterEnablement` (`bootstrap/enabled.go`) → Enabled ∩ imported
3. else no manifest (legacy) → all imported

A blank-import (or any import) only registers the package in the process. Generated apps also have `bootstrap/enabled.go`; a name that is imported but not listed there is not registered. `auth.From(app)` is nil unless `auth` is both imported and enabled. Providers run at `Bootstrap` (or the first `Start`/`Run`). Resolve services with `From(app)` — the kernel has no `app.Auth()` / `app.Queue()` / `app.AI()`.

Prefer the CLI, which writes both sides:

```bash
go run ./cmd/app package:enable auth
```

That updates `bootstrap/enabled.go`, writes a blank-import in `bootstrap/addons.go`, `go get`s `github.com/zatrano/packages@v1.10.0` when that module is not yet required, and merges env keys into `.env.example`. Then rebuild/restart.

To add a module that is not yet in `go.mod`, acquire first (enablement is separate; default acquire does not enable):

```bash
go run ./cmd/app package:acquire auth --enable
```

Manual equivalent:

```go
import (
    _ "github.com/zatrano/packages/session"
    _ "github.com/zatrano/packages/auth"
    _ "github.com/zatrano/packages/database"
)
```

…and add those names to `EnabledAddons` in `bootstrap/enabled.go`.

Resolve services with typed `From(app)` helpers — they are not methods on `App`:

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

`package:doctor` reports framework version, per-imported package state (`imported` / `enabled` / compatibility), and the transitive `Requires` closure. These are distinct states — there is no collapsed `"installed"` flag. Catalog: **[PACKAGES.md](PACKAGES.md)**. The package ecosystem is maintained separately from the kernel.

## HTTP

The kernel provides the HTTP runtime. Controllers use strongly typed kernel HTTP primitives.

Generated web home (`app/http/controllers/web`) returns kernel HTML (`view` is opt-in):

```go
func (c *HomeController) Index(req *http.Request) *http.Response {
    return http.HTML("<h1>__APP_NAME__</h1>")
}
```

JSON is the same `*http.Response` type (API home and `/up` use it):

```go
return http.JSON(map[string]any{"ok": true})
```

Routes are registered in the generated application (`r.Get("/", c.Index).As("home")`).

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

Construction is not Bootstrap:

```text
bootstrap.App / bootstrap.Boot
    → Created (providers registered, HTTP 503)

Application.Bootstrap
    → Booted (HTTP ready; workers not started)

Application.Start
    → Running (bootstraps first if needed; LifecycleProvider.Start)

Application.Run
    → Start + listen + SIGINT/SIGTERM → Stop

Application.Stop
    → Stopped (terminal; no-op unless Running)
```

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

HTTP is not dispatched until `Booted` (Created/Bootstrapping/BootFailed return `503`). `/up` on a generated app is process liveness after Bootstrap. Optional `package:enable health` adds `/health`. `LifecycleProvider.Start` (workers) runs only in `Start()` / `Running`. `Run()` handles `SIGINT`/`SIGTERM` with a 15s bounded HTTP shutdown, then `Stop` in reverse provider order.

Failed bootstrap is terminal for that application instance. Stopped applications cannot restart.

`contracts.App` also exposes context-bearing entry points:

```go
BootstrapContext(ctx context.Context) error
StartContext(ctx context.Context) error
```

`Bootstrap()` and `Start()` remain. They call the context methods with `context.Background()`. A nil `ctx` is treated as `context.Background()`. Cancellation is checked between provider operations (`Register`, `Boot`, `LifecycleProvider.Start`). An in-flight provider call is not forcibly killed. Provider method signatures are unchanged. If `StartContext` fails after some lifecycle providers started, those that returned nil from `Start` are stopped; the failed provider is not stopped; cleanup errors are retained with the start error; the application returns to Booted and may retry.

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

After bootstrap, the configuration repository is frozen. Startup-critical integers such as `APP_PORT` use `env.IntOr`: unset falls back, invalid values fail boot with a named type error and do not echo credential-like keys. `env.GetInt` remains a silent-fallback helper (invalid → default); do not use it where an unparsable value must stop startup.

## Package lifecycle

These operations are separate. Do not collapse them into “install”.

```text
Acquire   = modify Go module dependencies (go get via package:acquire)
Enable    = write consumer enablement + blank-imports (Requires closure)
Boot      = Provider.Register + Provider.Boot for Enabled ∩ Imported
Start     = LifecycleProvider.Start
Stop      = LifecycleProvider.Stop
Disable   = remove persistent enablement + that package’s blank-import
```

`package:enable` expands transitive `Requires` before writing files. Optional dependencies are not enabled. `package:disable` refuses if a remaining enabled addon requires the target (directly or through that Requires graph). Already-disabled is a successful no-op. Disable is not Stop: it does not shut down a running process.

Disable does not remove Go modules, config stubs, `.env` keys, or database state. Unused module cleanup is Go/user-owned (`go get` / `go mod tidy` are not run automatically).

Enablement does not overwrite an existing `github.com/zatrano/packages` requirement. A tagged `package:acquire` pin stays in go.mod. First-time wiring may `go get github.com/zatrano/packages@v1.10.0` (current stable tag) when that module is not yet required. That `go get` is a wiring convenience, not registry Resolve and not automatic enablement. `addons.Expand` closes declared `Requires` only.

Upgrade is `package:acquire name@version`. There is no `package:upgrade` or `package:uninstall` command.

## CLI

This repository's CLI entrypoint is `cmd/zatrano`. Generated applications use `cmd/app` (`go run ./cmd/app …`).

Always on the kernel CLI:

```bash
zatrano new
zatrano serve
zatrano doctor
zatrano describe
zatrano agents:generate
zatrano key:generate
zatrano package:list
zatrano package:search
zatrano package:info
zatrano package:resolve
zatrano package:enable
zatrano package:disable
zatrano package:doctor
zatrano make:controller
zatrano make:middleware
zatrano make:provider
zatrano make:command
zatrano make:service
zatrano make:exception
zatrano make:test
```

`make:request` and `make:rule` register when the validation package is imported. `db:setup`, `migrate`, `queue:work`, `make:auth`, and similar commands register only when their package is imported. They are not part of a kernel-only CLI.

Process exit codes are classified at the CLI boundary (`cmd/zatrano` via `console.CodeFromError`). Acquisition and runtime use different tables.

Acquisition (`package:search` / `info` / `resolve` / `acquire`, enablement on acquire `--enable` / `package:enable` / `package:install`):

```text
0  success
1  general
2  usage
3  resolution
4  planning
5  acquisition
6  enablement
7  canceled
```

Runtime (`zatrano serve` / `Application.Run`):

```text
0   success
20  ExitRuntimeBoot
21  ExitRuntimeShutdown
22  ExitRuntimeCanceled
23  ExitRuntimeTimeout
```

Runtime `serve` / `Run` failures use 20–23. They never reuse acquisition codes 2–7. Runtime cancellation is `ExitRuntimeCanceled` (22), not acquisition `ExitCanceled` (7). Acquisition JSON is unchanged. There is no runtime `serve --format=json` contract.

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
  dirs/            Application path helpers
  support/         Support utilities
contracts/         Public dependency-neutral ABI
bootstrap/         Application boot and package registry
console/           Platform CLI commands
console/doctor/    Application architecture doctor
console/pkgmanager/ Package enablement, registry, and acquire CLI
console/scaffold/  zatrano new / make:* and embedded starter templates
console/describe/  describe, catalog, AGENTS.md
console/consolecore/ Shared CLI types (no import cycles)
console/generator/ Generator engine (templates, placeholders, filesystem)
cmd/zatrano/       CLI entrypoint
distribution/      Package manifest, registry index, acquisition plan
tests/             Architecture, compatibility, boot, and fuzz tests
```

The packages ecosystem is maintained in the separate [packages](https://github.com/zatrano/packages) repository.

## Security

Security is a platform concern. Tests CI runs **go test -race**. Security CI runs **gosec**, **govulncheck**, **Semgrep**, **Trivy**, and Go fuzzing. Static analysis runs **go vet**.

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

### Frozen invariants

These are the public architectural baseline after Framework `v2.1.0` and Packages `v1.7.1`. Do not reopen them for convenience.

- Framework does not depend on Packages. Packages depend on Framework. Applications may import both.
- `contracts` remains dependency-neutral. Kernel owns runtime lifecycle. Capabilities are not `App` methods; typed facades (`From(app)`) live beside implementations.
- Packages are opt-in. `Enabled ∩ Imported` controls participation. `Expand` resolves declared package metadata dependencies only.
- Go modules own dependency resolution. There is no second PackageManager, resolver, ServiceLocator, or lifecycle manager.
- There is no automatic package enablement, implicit rollback, automatic `go mod tidy`, or `zatrano.lock`.
- Resource ownership is singular. `redisx`, `rag`, and `agent` are libraries. Cache owns the Redis connection.
- Framework remains `github.com/zatrano/framework/v2`. Packages remains `github.com/zatrano/packages` (v1.x, no `/v2` suffix).

## v2

**Framework `v2.6.3`** is the current public kernel. **Packages `v1.10.0`** is the current public official-packages release. Create applications with `zatrano new`. Do not clone this repository as your application.

```text
Framework
  module: github.com/zatrano/framework/v2
  major:  v2
  current: v2.6.3

Packages
  module: github.com/zatrano/packages
  major:  v1
  current: v1.10.0
```

The two modules release independently. Framework does not import Packages. Packages pins a compatible Framework release. Applications do not need a single ZATRANO-wide version.

Nested modules (SQL drivers, `mongo`, `webauthn`, `qr`) are separately versioned Go modules. Their tags follow the nested path (`database/driver/sqlite/v1.0.0`), not a `packages/` prefix. Nested publication is a separate release operation. Root `packages@v1.10.0` does not require those drivers.

Historical `packages@v1.7.0` required an unpublished nested SQLite module. Do not retag it. New apps use `v1.10.0`.

| Line | Meaning |
| --- | --- |
| Framework `v2.6.3` | Current kernel / CLI / contracts |
| Packages `v1.10.0` | Current official package ecosystem |
| Framework `v1.x` | Previous tagged kernel line |

Each module follows Go semantic versioning on its own path. Install current stables with the `go get` commands above.

## Learning ZATRANO

- [zatrano.com/docs](https://zatrano.com/docs)
- [Installation](https://zatrano.com/docs/installation)
- [PACKAGES.md](PACKAGES.md)
- [github.com/zatrano/packages](https://github.com/zatrano/packages)
- [github.com/zatrano/examples](https://github.com/zatrano/examples)
- [Releases](https://github.com/zatrano/framework/releases)

## Community

Issues and pull requests are welcome. Keep changes focused, preserve architectural boundaries, add tests for behavioral changes, and run `gofmt`.

## License

[MIT](LICENSE) · Copyright (c) 2026 Serhan KARAKOÇ

- [zatrano.com](https://zatrano.com/docs)
- [github.com/zatrano/framework/v2](https://github.com/zatrano/framework)
- [github.com/zatrano/packages](https://github.com/zatrano/packages)
- [github.com/zatrano/examples](https://github.com/zatrano/examples)
- [linkedin.com/company/zatrano](https://www.linkedin.com/company/zatrano)
