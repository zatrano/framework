<p align="center">
  <strong>ZATRANO</strong>
</p>

<p align="center">
  <em>A small, typed Go application kernel. V3 carries HTTP on rawhttp and renders web pages with Canvas.</em>
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
  <a href="https://pkg.go.dev/github.com/zatrano/framework/v3"><img src="https://img.shields.io/badge/golang-1.25+-00ADD8?logo=go&logoColor=white" alt="Golang"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="VERSION"><img src="https://img.shields.io/badge/version-3.1.0-green.svg" alt="Version"></a>
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
  ·
  <a href="UPGRADING.md">Upgrading from v3.0.1</a>
</p>

---

## V3

This repository is the **V3 kernel**: `github.com/zatrano/framework/v3`.

| | |
| --- | --- |
| Current module tag | `v3.1.0` |
| Release branch | `main` |
| HTTP carrier | [`github.com/zatrano/rawhttp`](https://github.com/zatrano/rawhttp) `v0.2.4` |
| HTML SSR | [`github.com/zatrano/canvas`](https://github.com/zatrano/canvas) `v0.2.0`, wired at `framework/v3/core/ssr` |
| Packages | [`github.com/zatrano/packages`](https://github.com/zatrano/packages) `v1.14.0` |

The kernel requires **rawhttp** and **canvas**. It does not import `github.com/zatrano/packages`. Web handlers return `http.Template`. API handlers return `http.JSON`. `routing.Version` mounts `/api/{version}`.

Older lines stay on their own branches:

| Branch | Line |
| --- | --- |
| `main` | V3. Current tag `v3.1.0` |
| `3.x` | V3 through `v3.0.1` |
| `2.x` | V2. Tag `v2.8.1`, packages `v1.13.1` |
| `1.x` | V1. Tag `v1.6.6` |

Create applications with `zatrano new`. Do not clone this repository and treat it as your application.

## What is ZATRANO?

ZATRANO is a **small, typed kernel** plus opt-in packages. The kernel is HTTP, container, config, routing, lifecycle, and CLI. Packages bind with `From(app)`. `contracts.App` does not grow package methods.

The same surface is for people and for AI agents:

```bash
zatrano describe
zatrano doctor
zatrano agents:generate
```

- `describe` prints the live catalog, routing primitives, and package inventory.
- `doctor` checks the application against the frozen architecture rules.
- `agents:generate` writes an application-root `AGENTS.md` from `describe`.

`ai`, `rag`, `agent`, and `workflow` are **experimental**.

You import what you run.

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
             │      HTTP   Router  Config     AI
             │   Middleware Container         RAG
             │      rawhttp   Canvas         Auth
             │                                Database
             │                                Queue
             │
             └──────────── Application ────────────┘
                              │
                         bootstrap.App()
                              ▼
                         Your application
```

```text
framework   runtime, contracts, CLI, generator, Canvas SSR wiring
packages    optional capabilities, package-owned make:*, stubs, config
rawhttp     HTTP/1.1 carrier
canvas      HTML template engine
```

`zatrano new myapp` generates one application: Canvas HTML at `/` (`http.Template`) and JSON at `/api` (`http.JSON`). Default enabled addons are `health` and `template` (`framework/v3/core/ssr`). `assets`, `localization`, `validation`, database, auth, and queue stay opt-in via `package:enable`.

## Quick start

Requires **Go 1.25+**.

```bash
go install github.com/zatrano/framework/v3/cmd/zatrano@v3.1.0
zatrano new myapp
cd myapp
go mod tidy
go run ./cmd/app key:generate
go run ./cmd/app serve
```

Open [http://localhost:8080](http://localhost:8080). The listen port is `APP_PORT` (default 8080).

Pin the modules in an existing `go.mod`:

```bash
go get github.com/zatrano/framework/v3@v3.1.0
go get github.com/zatrano/packages@v1.14.0
```

These are the current stable public releases. The two modules version independently. There is no monolithic `zatrano@x.y.z` version. `v3.0.0` is retracted and cannot be installed. Moving from v3.0.1 is described in [Upgrading](UPGRADING.md).

## Two modules

| Module | Role |
| --- | --- |
| [`github.com/zatrano/framework/v3`](https://github.com/zatrano/framework) | Kernel, contracts, bootstrap, CLI, Canvas SSR wiring |
| [`github.com/zatrano/packages`](https://github.com/zatrano/packages) | Optional services and libraries |

Packages depend on this module. This module does not import packages.

Nested modules (`db/*`, `mongo`, `webauthn`, `qr`) have their own tags. Root `packages@v1.14.0` does not require them.

## Architecture

### Kernel

`core/kernel` holds the primitives required to run an application: lifecycle, HTTP, routing, middleware, container, configuration, environment, encryption, cookies, logging, exceptions, reports, trusted proxies, and safe paths.

### Contracts

`core/contracts` is the dependency-neutral public ABI: `App`, `Provider`, `LifecycleProvider`, `Container`, `Router`, and the HTTP bridge. It does not import kernel implementation packages or the packages module.

Typed APIs live next to their implementations:

```go
import (
    "github.com/zatrano/framework/v3/core/kernel/http"
    "github.com/zatrano/framework/v3/core/kernel/routing"
)

r := routing.From(app)

r.Get("/health", func(req *http.Request) *http.Response {
    return http.JSON(map[string]any{"ok": true})
})

routing.Version(r, "v1", func(api *routing.Router) {
    api.Get("/ping", ping)
})
```

`routing.Version` mounts `/api/{version}` and sets `X-API-Version`.

### Packages

Optional capabilities live in [`github.com/zatrano/packages`](https://github.com/zatrano/packages): session, validation, authentication, SQL (`db`), queues, notifications, scheduler, localization, cache, AI, RAG, agents, OAuth, social login, WebAuthn, backups, OpenAPI, GraphQL, and import-only libraries.

Database, auth, queue, and similar packages stay off until `package:enable`. Catalog: **[PACKAGES.md](PACKAGES.md)**.

## Application model

```bash
zatrano new myapp
```

```text
myapp/
├── app/
│   ├── http/
│   ├── routes/
│   └── providers/
├── bootstrap/
│   ├── addons.go      # blank-imports
│   └── enabled.go     # enablement manifest
├── cmd/app/
├── templates/
├── public/
├── storage/
├── tests/
└── go.mod
```

A framework upgrade does not regenerate application source. `zatrano new` records the scaffold name, version, and digest in `bootstrap/scaffold.go`.

## Enabling packages

Acquisition, import, and enablement are separate.

```text
Acquire                 go get (package:acquire)
   ↓
Imported                blank-import in the application
   ↓
Enabled ∩ Imported      bootstrap.App() registers that intersection
   ↓
Expand Requires         declared addon metadata only
   ↓
Bootstrap               Provider.Register + Provider.Boot
```

`bootstrap.App()` registers **Enabled ∩ Imported**. It does not call `Application.Bootstrap()`. A name that is imported but absent from `bootstrap/enabled.go` is not registered. `auth.From(app)` is nil unless `auth` is both imported and enabled. Resolve services with `From(app)`. The kernel has no `app.Auth()` / `app.Queue()` / `app.AI()`.

```bash
go run ./cmd/app package:enable auth
```

That updates `bootstrap/enabled.go`, writes a blank-import in `bootstrap/addons.go`, `go get`s `github.com/zatrano/packages@v1.14.0` when that module is not yet required, and merges env keys into `.env.example`. Rebuild or restart after enablement.

```bash
go run ./cmd/app package:acquire auth --enable
go run ./cmd/app package:list
go run ./cmd/app package:doctor
```

`package:doctor` reports framework version, per-imported package state (`imported` / `enabled` / compatibility), and the transitive `Requires` closure.

```text
Acquire   = go get via package:acquire
Enable    = enablement files + blank-imports (Requires closure)
Boot      = Provider.Register + Provider.Boot for Enabled ∩ Imported
Start     = LifecycleProvider.Start
Stop      = LifecycleProvider.Stop
Disable   = remove persistent enablement and that package’s blank-import
```

`package:enable` expands transitive `Requires` before writing files. `package:disable` refuses when a remaining enabled addon requires the target. Disable does not stop a running process and does not run `go mod tidy`. First-time wiring may `go get github.com/zatrano/packages@v1.14.0`. An existing packages requirement is left alone. Upgrade is `package:acquire name@version`.

## HTTP

Web home returns a Canvas template:

```go
func (c *HomeHandler) Index(req *http.Request) *http.Response {
    return http.Template("web.welcome")
}
```

JSON uses the same `*http.Response` type:

```go
return http.JSON(map[string]any{"ok": true})
```

The carrier is rawhttp (`Application.Handle`). Kernel middleware covers CSRF, CORS, security headers, trusted proxies, request IDs, exception handling, method override, request limits, and safe static files.

`MAX_BODY_BYTES`, `MAX_UPLOAD_BYTES`, and `HTTP_MAX_INFLIGHT_BODY_BYTES` are read once at boot for the header hook. Changing them requires a process restart. `Request.Body` and `Request.JSON` still read `MAX_BODY_BYTES` on each call.

A path with no route and no fallback runs that global middleware and then returns 404. The matched path is unchanged. A method that does not match a route is 404 with no `Allow` header. A browser CORS preflight (`OPTIONS` plus `Origin` plus `Access-Control-Request-Method`) is answered by CORS before later middleware, whether or not a route exists: an allowed origin is 204 with the allow headers and `Vary: Origin, Access-Control-Request-Method, Access-Control-Request-Headers`; any other origin gets no allow headers and 404. An OPTIONS request that is not a preflight still runs a registered OPTIONS route. Only an exact allowed origin is reflected. When the allow list is not a wildcard, every response to a request that carries `Origin` includes `Vary: Origin`, including 404. No configured origin means no CORS headers and no `Vary`. Preflight `Access-Control-Allow-Headers` is the intersection of `Access-Control-Request-Headers` with the configured list. `Access-Control-Max-Age` defaults to 600. Credentials combined with a wildcard origin, including the implicit development wildcard, fails boot. `CORSWith` still will not emit both.

rawhttp writes 400, 413, and 431 before `Handle`. Those responses do not include framework headers. `HTTP_MAX_HEADER_BYTES` defaults to 16 KiB. v3.0.0 and v3.0.1 used 1 MiB, and rawhttp allocates the connection read buffer from that ceiling, so a new connection per request paid for a megabyte. A header block over the limit is 431. A configured value above 64 KiB logs a warning.

The router is mutable during registration and immutable after `Freeze()`. Typed routing is `routing.From(app)`. `Route.BodyLimit(n)` on that concrete route sets one route's body cap in bytes. A negative `n` uses the server ceiling. A positive `n` is sent as `RequestConfig.MaxRequestBodySize` and can be larger than the server ceiling; rawhttp then allows that request up to `n`. If `n` is also larger than `HTTP_MAX_INFLIGHT_BODY_BYTES`, the request cannot be reserved and is rejected with 413 rather than a retryable 503.

```go
func init() {
    routing.RegisterAPI(func(r *routing.Router) {
        r.Post("/hook", receive).BodyLimit(8 << 20)
    })
}

routing.From(app).Post("/hook", receive).BodyLimit(8 << 20)
```

Header-time matching uses the same resolver as `Dispatch`: method override from `X-HTTP-Method-Override`, and a trailing slash is ignored. Percent-encoding, letter case, `/./`, and extra slashes are not rewritten. When the method can still change because `_method` is in an unread urlencoded body, or more than one route could match, the cap stays at the tightest candidate and is never raised.

Header-time caps: JSON (`application/json` and `+json`), `application/x-www-form-urlencoded`, and `text/*` stop at `MAX_BODY_BYTES` (2 MiB). Multipart and every other type, including a missing `Content-Type`, stop at `MaxRequestBytes` (32 MiB unless `MAX_UPLOAD_BYTES` is higher). `Request.Body` and `Request.JSON` still stop at 2 MiB. The router is consulted for a `BodyLimit` only when `Content-Length` is above that default or the body is chunked.

| Variable | Unset | `0` or negative |
| --- | --- | --- |
| `HTTP_READ_TIMEOUT` | 60s | deadline off |
| `HTTP_WRITE_TIMEOUT` | 60s | deadline off |
| `HTTP_IDLE_TIMEOUT` | 120s | deadline off |
| `HTTP_READ_HEADER_TIMEOUT` | 10s | deadline off |
| `HTTP_MAX_HEADER_BYTES` | 16 KiB | positive byte count. Invalid values abort boot. Above 64 KiB logs a per-connection buffer warning |
| `HTTP_ALLOW_UPGRADE` | unset (off) | `false` forces admission off; `true` turns it on |
| `HTTP_MAX_INFLIGHT_BODY_BYTES` | 256 MiB | negative disables the budget. A known length reserves that length. A chunked body reserves the worst case. Not enough room is 503 with `Retry-After: 1` and the body is not read. One request whose cap is larger than the whole budget is 413. |
| `HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT` | 25% of the budget | Negative disables the per-client share. Over the share is 503 with `Retry-After: 1`. The share is skipped when the peer is not a distinct client. |
| `HTTP_STRICT_LIMITS` | off | A route `BodyLimit` above the in-flight budget fails boot in every environment. Production fails boot either way. |
| `MAX_BODY_BYTES` | 2 MiB | ignored unless positive |
| `MAX_UPLOAD_BYTES` | 32 MiB | ignored unless positive |
| `TRUSTED_PROXIES` | empty | Comma-separated proxy or CDN addresses. Empty skips the per-client body share for private and loopback peers, and keys a CDN edge address as the client. `zatrano doctor` reports that as APP-HTTP-007. |
| `CORS_ENABLED` | true | `false` does not install CORS middleware. |
| `CORS_ALLOWED_ORIGINS` | empty (development may use `*`) | Comma-separated exact origins. No configured origin writes no CORS headers. |
| `CORS_ALLOWED_METHODS` | `GET, POST, PUT, PATCH, DELETE, OPTIONS` | Preflight allow-methods. |
| `CORS_ALLOWED_HEADERS` | `Content-Type, Authorization, X-Requested-With, X-CSRF-TOKEN, X-Idempotency-Key` | Preflight allow-headers are the intersection with the request, not an echo. |
| `CORS_EXPOSE_HEADERS` | empty | `Access-Control-Expose-Headers` when an origin matches. |
| `CORS_ALLOW_CREDENTIALS` | false | `true` with a wildcard origin, including the implicit development wildcard, fails boot. |
| `CORS_MAX_AGE` | 600 | Preflight `Access-Control-Max-Age`. A non-integer is ignored. |

A unitless integer is seconds (`30`). Go durations (`30s`, `1m`) are accepted. An invalid value (`abc`, `1x`) aborts boot. The 60s read timeout can cut a slow upload: 32 MiB in 60s is about 4.4 Mbit/s. Raise `HTTP_READ_TIMEOUT`, or set it to `0`, for large uploads on a slow link.

`HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT` defaults to a quarter of the in-flight budget. A negative value turns the per-client share off. The share applies only when clients can be told apart: a configured `TRUSTED_PROXIES` list and a resolved client address (IPv6 grouped by /64), or, with no proxy, a global-unicast peer that is not loopback, RFC1918, link-local, CGNAT (`100.64.0.0/10`), or IPv6 unique-local. Otherwise only the global budget applies, and the first such request logs one warning with no address. `zatrano doctor` reports the missing proxy list as APP-HTTP-007. Set `TRUSTED_PROXIES` to the reverse proxy when the process listens on a private address. Behind a CDN the edge addresses are global unicast, so with `TRUSTED_PROXIES` unset the share keys on that edge address: visitors that arrive through the same address share one quarter of the budget (64 MiB at the default). APP-HTTP-007 fires in that case too. Set `TRUSTED_PROXIES` to the CDN ranges so the client address from `X-Forwarded-For` is the key. One distinguishable client that dribbles a body can hold only its share, not the whole budget. A slow body can still hold that share until `HTTP_READ_TIMEOUT`. A body-progress timer is rawhttp v0.3.0. Rate-limit at the edge (`limit_conn`, a WAF) as well.

The in-flight budget is checked when the headers arrive. A known `Content-Length` reserves that many bytes. A chunked body has no length yet, so JSON, urlencoded, and `text/*` reserve `MAX_BODY_BYTES` and every other type reserves its effective cap. If that reservation does not fit what is already in flight, the response is 503 with `Retry-After: 1` and the body is not read. The hold is released once: when the handler returns, or when the connection closes, including after a panic or a client that disconnects mid-body. A route `BodyLimit` larger than the budget logs a warning and is rejected with 413; production refuses to boot, and `HTTP_STRICT_LIMITS` does the same in every environment. `zatrano doctor` reports it as APP-HTTP-006.

An early 413 or 503 does not drop the TCP connection before the client can read it. rawhttp 0.2.3 lingers on that close. `MaxLingering` defaults to 1024 connections; a negative value is unlimited. Connections past the cap are closed immediately.

| Body | V2 | V3 |
| --- | --- | --- |
| JSON, urlencoded, `text/*` | 2 MiB | 2 MiB at header time (`MAX_BODY_BYTES`). `Request.JSON` still stops at 2 MiB. |
| Multipart, octet-stream, other, missing type | route / php limit | 32 MiB server ceiling unless `BodyLimit` or `MAX_UPLOAD_BYTES` says otherwise |
| Chunked over the cap | 413 | 413. v3.0.0–v3.0.1 could return 431 when the body exceeded the read buffer (rawhttp 0.2.2). |
| Over the in-flight budget | not applied | 503 and `Retry-After: 1`. One request that cannot fit the budget at all is 413. |
| Header block | bufio 4 KiB | 16 KiB (`HTTP_MAX_HEADER_BYTES`). Over the limit is 431. v3.0.0–v3.0.1 used 1 MiB. |

`Stream` and `StreamBody` re-arm the write deadline before each chunk. `ClearWriteDeadline` removes it and accepts a peer that stops reading. `Hijack` always clears it.

WebSocket admission, first match wins:

| Order | Source |
| --- | --- |
| 1 | `HTTP_ALLOW_UPGRADE=false` forces admission off |
| 2 | `ListenOptions.AllowUpgrade`, including `serve --allow-upgrade` |
| 3 | `HTTP_ALLOW_UPGRADE=true` |
| 4 | `RegisterUpgradeProtocol` |
| 5 | linked `websocket` addon |
| 6 | `EnabledAddons` contains `websocket` |

Default is off. `Upgrade: h2c` is rejected even when admission is open. A valid websocket handshake to a route that does not hijack the connection is answered, then the connection is closed.

## Application lifecycle

```text
bootstrap.App
    → Created (providers registered, HTTP 503)

Application.Bootstrap
    → Booted (HTTP ready)

Application.Start
    → Running (LifecycleProvider.Start)

Application.Run
    → Start + listen + SIGINT/SIGTERM → Stop

Application.Stop
    → Stopped
```

HTTP is not dispatched until `Booted`. `/up` is process liveness after Bootstrap. `package:enable health` adds `/health`. `Run()` shuts down within 15s on `SIGINT`/`SIGTERM`. The order is shutdown hooks, then rawhttp `Shutdown`, then `Stop` in reverse provider order. Hooks run concurrently. Each one gets `min(5s, remaining/3)`. A panic or a returned error is logged and does not stop the shutdown. A second pass over the same shutdown does nothing, and a hook registered after shutdown has started is ignored. If `Shutdown` returns `context.DeadlineExceeded` because a handler is still inside `Hijack`, `Close` drops that connection so the process can leave.

```go
http.RegisterShutdownHook("jobs", func(ctx context.Context) error {
    return jobs.Shutdown(ctx)
})
```

`packages/websocket` registers its own hook. On shutdown it refuses new upgrades with 503, writes a close frame `1001 Going Away` (write deadline 1s), waits for the peer close or 1s, then closes the connection. An application author does not register a second hook for those sockets.

```go
BootstrapContext(ctx context.Context) error
StartContext(ctx context.Context) error
```

`Bootstrap()` and `Start()` call those with `context.Background()`.

## CLI

This repository's entrypoint is `cmd/zatrano`. Generated applications use `cmd/app`.

Kernel commands include `new`, `serve`, `doctor`, `describe`, `agents:generate`, `key:generate`, `package:list`, `package:search`, `package:info`, `package:resolve`, `package:enable`, `package:disable`, `package:doctor`, and `make:handler`, `make:middleware`, `make:provider`, `make:command`, `make:service`, `make:exception`, `make:test`.

`make:request` and `make:rule` register when validation is imported. `db:setup`, `migrate`, `queue:work`, and `make:auth` register when their package is imported.

Acquisition exit codes are 0–7 (`success`, `general`, `usage`, `resolution`, `planning`, `acquisition`, `enablement`, `canceled`). Runtime `serve` / `Run` uses 20–23 and does not reuse acquisition codes 2–7.

## Repository structure

```text
core/kernel/          HTTP, routing, middleware, config, container, env
core/contracts/       Public dependency-neutral ABI
core/bootstrap/       Application boot and package registry
core/ssr/             Canvas engine binding
core/console/         CLI, doctor, package manager, scaffold, describe
cmd/zatrano/          CLI entrypoint
core/distribution/    Manifest, registry index, acquisition plan
tests/                Architecture, compatibility, boot, and fuzz tests
```

The package ecosystem is [github.com/zatrano/packages](https://github.com/zatrano/packages).

## Security

Tests CI runs **go test -race**. Security CI runs **gosec**, **govulncheck**, **Semgrep**, **Trivy**, and Go fuzzing. Static analysis runs **go vet**.

```bash
bash .github/scripts/release-gate.sh
```

Kernel security primitives include request size limits, safe path resolution, secure request IDs, security headers, trusted proxy handling, production secret validation, exception isolation, and cookie protection.

Report vulnerabilities privately to Serhan KARAKOÇ — [serhankarakoc@zatrano.com](mailto:serhankarakoc@zatrano.com). Do not open a public GitHub issue for security reports.

## Architecture principles

1. **Small kernel** — primitives, not every application feature.
2. **Dependency-neutral contracts** — the ABI does not import kernel implementations or optional packages.
3. **Opt-in capabilities** — application capabilities are packages.
4. **Typed developer APIs** — `From(app)` beside each implementation.
5. **One-way dependency flow** — application → packages → framework. The kernel does not import packages.
6. **Immutable runtime configuration** — registration is mutable during boot and frozen before runtime.
7. **Explicit lifecycle** — startup and shutdown are deterministic.
8. **Platform** — runtime and capability ecosystem stay separate modules.

V3 baseline:

- Framework does not depend on Packages. Packages depend on Framework.
- `contracts` stays dependency-neutral. Capabilities are not `App` methods.
- **Enabled ∩ Imported** controls participation. **Expand Requires** resolves declared package metadata only.
- Go modules own dependency resolution. There is no second resolver, no automatic enablement, and no `zatrano.lock`.
- Framework remains `github.com/zatrano/framework/v3`. Packages remains `github.com/zatrano/packages` (v1 module path).

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
- [github.com/zatrano/framework](https://github.com/zatrano/framework)
- [github.com/zatrano/packages](https://github.com/zatrano/packages)
- [github.com/zatrano/examples](https://github.com/zatrano/examples)
- [linkedin.com/company/zatrano](https://www.linkedin.com/company/zatrano)
