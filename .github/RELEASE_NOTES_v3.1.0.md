# v3.1.0

Minor release. `go get -u=patch` does not select it. Upgrade details and the way back to each v3.0.1 behavior are in [UPGRADING.md](../UPGRADING.md).

`v3.0.0` is retracted. `go install github.com/zatrano/framework/v3@v3.0.0` is rejected because that module was published with local `replace` directives. Install `v3.0.1` or `v3.1.0`.

## Breaking / Changed

- `MaxHeaderBytes` is 16 KiB (`HTTP_MAX_HEADER_BYTES`). v3.0.0 and v3.0.1 used 1 MiB. A header block over the limit is 431.
- JSON, urlencoded, and `text/*` bodies are capped at 2 MiB when the headers arrive. Other types use a 32 MiB server ceiling. `Route.BodyLimit` overrides one route. v3.0.1 left the engine ceiling at rawhttp's 4 MiB default.
- In-flight body budget defaults to 256 MiB. The per-client share defaults to 25% when clients can be told apart. Over the share or the remaining budget is 503 with `Retry-After: 1`. A negative value turns each one off.
- A chunked body over the cap is 413. rawhttp v0.2.2 could return 431.
- Unmatched paths run global middleware, then return 404. A CORS preflight is answered before later middleware. No `Origin` means no CORS headers.
- WebSocket upgrade is off unless `HTTP_ALLOW_UPGRADE`, `ListenOptions.AllowUpgrade`, `serve --allow-upgrade`, `RegisterUpgradeProtocol`, or the websocket addon turns it on.
- Require rawhttp v0.2.4. No `replace` directive.
- `VERSION` is `3.1.0` and matches the tag. v3.0.1 left `VERSION` at `3.0.0`.

## Fixed

- Header-time `BodyLimit` uses the same method and path resolver as `Dispatch`.
- A route cap above the in-flight budget is 413, not a retryable 503. Production refuses to boot. `HTTP_STRICT_LIMITS` does that in every environment.
- `Expect: 100-continue` over the cap or over the budget is 413 or 503 with no `100 Continue`.
- CORS `Vary: Origin` follows the request `Origin`. Preflight allow-headers are the intersection with the configured list. Credentials with a wildcard origin fail boot.

## Added

- `http.RegisterShutdownHook`. Hooks run before `Shutdown`, then `Stop`. `context.DeadlineExceeded` is followed by `Close`, which drops a handler still inside `Hijack`. `packages/websocket` uses the hook to reject new upgrades with 503 and send close frame 1001.
- `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_IDLE_TIMEOUT`, and `HTTP_READ_HEADER_TIMEOUT` (60s, 60s, 120s, 10s). `0` or negative disables one deadline. A unitless integer is seconds.
- `zatrano doctor` APP-HTTP-006 (body limit above the budget) and APP-HTTP-007 (`TRUSTED_PROXIES` unset, including behind a CDN).

## Upgrading

[Upgrading from v3.0.1](../UPGRADING.md)
