# Reference application

A small production-shaped ZATRANO consumer. It exists to pressure the public
v2 APIs — not to extend the framework.

This directory is **not** a kernel, bootstrap, or console package. Treat it as
an external application that happens to live in the framework module so the
architecture suite can still walk it. It must not import `github.com/zatrano/packages`.

## Architecture

```text
cmd/reference          transport (HTTP) + process entry
internal/domain        items + sentinel errors
internal/service       application use-cases
internal/repository    in-memory persistence boundary
internal/worker        LifecycleProvider-owned ticker
internal/transport     HTTP mapping
config.go              kernel/env only
providers.go           Register / Boot / Start / Stop
```

Business logic stays small. The pressure is on lifecycle, configuration, HTTP,
readiness, shutdown, and errors.

## How do I start?

```bash
go run ./examples/reference/cmd/reference serve
```

That is the public path:

1. `bootstrap.App(bootstrap.WithProviders(...))` constructs the application
   (it does **not** call `Bootstrap`).
2. `console.New(app).Run` dispatches CLI commands.
3. `serve` calls `Application.Run`, which `Start`s lifecycle providers and
   listens. `Start` bootstraps if needed.

Tests that inject failures use `kernel.NewApplication` + `RegisterProviders` +
`Bootstrap` / `Start` / `Stop` because `bootstrap.App` panics only on
construction and is awkward for extra failing providers.

Exit codes for `serve` / `Run` come from `console.CodeFromError` (runtime 20–23).

## How do I configure?

Process environment via `kernel/env`:

| Variable | Role |
| --- | --- |
| `REFERENCE_NAME` | Default `reference` (`env.Get`) |
| `REFERENCE_POLL_MS` | Worker interval, default 50 (`env.IntOr`) |
| `REFERENCE_API_TOKEN` | Sensitive; never logged or returned |
| `REFERENCE_REQUIRE_TOKEN` | When true, empty token fails Register |
| `APP_PORT` | HTTP listen port (kernel `Run`) |

Invalid integers fail with a named type error. Secret-like keys do not echo
the received value. There is no `ConfigManager`.

## How do I register providers?

Return `[]kernel.Provider` from `Providers()` and pass them to
`bootstrap.WithProviders`. One ordinary `Provider` (`ConfigProvider`,
`CoreProvider`, `HTTPProvider`) plus one `LifecycleProvider` (`WorkerProvider`).

Bind instances with `app.Container().Instance` / `app.Make` during `Register`.
Do not invent a parallel container.

## How do I expose HTTP?

`routing.From(app)` in `Provider.Boot` (before the kernel freezes the router):

- `GET /up` — liveness after **Bootstrap** (kernel returns 503 until Booted)
- `GET /api/v1/status` — application info (name, whether the worker is running)
- `GET /api/v1/items/{id}` — sample resource

Do not treat `/up` as “worker running”. HTTP is ready at Booted; workers start at `Start`.

## How do I start a worker?

Implement `contracts.LifecycleProvider`. Launch goroutines in `Start`, not in
`Register` or `Boot`. `Stop` receives only `context.Context` (not `App`); keep
the worker pointer on the provider.

## How does shutdown work?

`Application.Run` listens for SIGINT/SIGTERM, shuts down the HTTP server, then
`Stop`s lifecycle providers in reverse order. The first `Stop` error remains
visible; later providers still run. `Stop` is a no-op unless the app is Running.

## How do I determine readiness?

- Before Bootstrap: any HTTP path is 503 (kernel).
- After Bootstrap: `/up` returns `{"status":"ok"}`.
- After Start: `/api/v1/status` reports `"worker": true`.

There is no kernel health subsystem. Optional `health` packages stay out of this app.

## How do I test it?

```bash
go test ./examples/reference/...
```

| Layer | What |
| --- | --- |
| Unit | `internal/domain`, `internal/service`, `internal/repository`, `internal/worker`, `internal/transport` |
| Integration | `http_test.go` — Bootstrap/Start → `ServeHTTP` |
| Lifecycle | `lifecycle_test.go` — Register/Boot/Start/Stop and injected failures |

## Persistence

The official packages repository has database support, but **this module must
not import it**. The in-memory repository shows that domain code depends on
`domain.Repository`, not on a driver. A real consumer would `package:acquire`
/ `package:enable` a database package in their own module.

## Architecture pressure

See the table at the end of this file. Friction is recorded; it is not an
automatic framework change.

| Area | Works | Friction | Required framework change |
| --- | --- | --- | --- |
| Configuration | `env.Get` / `IntOr` / `ConfigError` / `Sensitive` | App settings are not on `app.Config()` unless you copy them there | No |
| Provider lifecycle | Register → Boot → Start → Stop | `bootstrap.App` does not Bootstrap; tests use `NewApplication` | No |
| HTTP | `routing.From` + `kernel/http` | Typed router vs `App.Router()` is a known split | No |
| Workers | `LifecycleProvider` | `Stop` has no `App`; keep a field on the provider | No |
| Errors | `errors.Is` + HTTP map | No global mapper (by design) | No |
| Testing | `ServeHTTP` + temp dir | No exported `Running` state; infer from worker/HTTP | No |
| Package integration | N/A in-tree | Cannot enable `github.com/zatrano/packages` here | No (consumer module) |
| Shutdown | reverse Stop, first error kept | Start-failure cleanup vs Stop semantics differ (documented) | No |
| Readiness | kernel 503 until Booted; `/up` after | `/up` ≠ worker-running; easy to confuse | No |
