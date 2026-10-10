# Changelog

All notable changes to ZATRANO are documented in this file.

## Unreleased

## 3.1.1 - 2026-10-09

### Added

- `zatrano new` runs `go mod tidy` after writing the project. This is a convenience: the command already printed `go mod tidy` as the next step. When tidy succeeds, the Next list is `cd`, `key:generate`, and `serve`. `--no-tidy` skips the step. If tidy fails, times out after 120 seconds, or `--no-tidy` skips it, the command still exits 0, the Next list keeps `go mod tidy`, and exactly one line is printed: Run `go mod tidy` in <dir> before building. `GOFLAGS` and `GOPROXY` are left unchanged. `--replace` uses the same path and does not fail the command when tidy fails.
- `zatrano new --framework-version vX.Y.Z` writes that version into the generated `go.mod`. The default is this CLI's own version. A missing `v` prefix, or a value that is not a version, is an error and the project is not written. `package:enable` stays `github.com/zatrano/packages@v1.14.0`.

## 3.1.0 - 2026-10-07

Minor release. New public API (`Route.BodyLimit`, `RegisterUpgradeProtocol`, `ListenOptions.AllowUpgrade`, `RegisterShutdownHook`) and changed defaults (`MaxHeaderBytes` 16 KiB, JSON body cap). `go get -u=patch` does not cross this boundary. Performance work (buffer pool, header list, JSON, request id, streaming multipart) is v3.2.0. Product `VERSION` matches the release tag. `v3.0.1` left `VERSION` at `3.0.0`; that split stays closed. Behavior changes and the way back are in [Upgrading from v3.0.1](UPGRADING.md).

### Fixed

- The production header ceiling is 16 KiB (`HTTP_MAX_HEADER_BYTES`, a unitless byte count; an invalid value aborts boot). v3.0.0–v3.0.1 set `MaxHeaderBytes` to 1 MiB, and rawhttp allocates the per-connection read buffer from that value. Clients that open a new connection per request lost about 60% of their throughput versus the rawhttp 8 KiB default (Linux, 64 concurrent clients, no keep-alive: about 10.1k rps at 8 KiB, about 4.0k rps at 1 MiB). A value above 64 KiB logs a warning until rawhttp v0.3.0 grows the buffer from 4 KiB. A header block over the limit is still 431 from the engine.
- v3.0.0–v3.0.1 rejected a chunked body larger than the read buffer with 431 (rawhttp 0.2.2). rawhttp 0.2.3 applies the body cap instead, so a chunked JSON body over 2 MiB and a chunked multipart body over 32 MiB are 413.
- Unmatched requests run the global middleware, then return 404. A matched route does not gain a wrapper. There is no `Allow` header and no 405: a method that does not match is 404, the same as v2.4.0.
- A CORS preflight (`OPTIONS` with `Origin` and `Access-Control-Request-Method`) is answered before later middleware and before the route. An allowed origin gets 204, `Access-Control-Allow-Origin`, `Access-Control-Allow-Methods`, `Access-Control-Allow-Headers`, `Access-Control-Max-Age`, and `Vary: Origin, Access-Control-Request-Method, Access-Control-Request-Headers`. Any other origin gets no CORS headers and 404. Auth registered after CORS does not see the preflight. An OPTIONS request that is not a preflight still runs a registered OPTIONS route. Wildcard origin and credentials are never sent together.
- rawhttp answers 400, 413, and 431 before the framework handler. Those responses do not carry security headers, `X-Request-ID`, or CORS headers.
- The header hook no longer calls `env.Get` per request. Bodyless requests return the carrier default with no allocation. Body ceilings are fixed at boot; changing `MAX_BODY_BYTES`, `MAX_UPLOAD_BYTES`, or `HTTP_MAX_INFLIGHT_BODY_BYTES` needs a restart. `Request.Body` and `Request.JSON` still read `MAX_BODY_BYTES` on each call.
- CORS `Vary: Origin` is set on every response whose request carries `Origin` when the allow list is not a wildcard, including 404 and a denied preflight. No configured origin adds no CORS headers and no `Vary`. Reflection is an exact origin match. Preflight `Access-Control-Allow-Headers` is the intersection with the configured list, not an echo of the request. `Access-Control-Max-Age` defaults to 600 and `CORS_MAX_AGE` overrides it. Credentials combined with a wildcard origin is a boot error.
- Header-time `BodyLimit` uses the same method and path resolver as `Dispatch`. A trailing slash is ignored. Case, percent-encoding, `/./`, and extra slashes are not rewritten. `X-HTTP-Method-Override` is applied before the lookup. An unread urlencoded `_method`, or more than one candidate route, keeps the tightest cap and never a larger one.
- `RequestConfig.MaxRequestBodySize` can exceed the server ceiling when `BodyLimit` does. A cap that also exceeds `HTTP_MAX_INFLIGHT_BODY_BYTES` is not granted: the request is cut so the engine returns 413, not 503.

### Added

- `http.RegisterShutdownHook` runs before rawhttp `Shutdown` on process exit. Hooks are concurrent, bounded by `min(5s, remaining/3)`, and a panic or error is logged without stopping the shutdown. A second call is a no-op. A hook registered after shutdown has started is ignored. `context.DeadlineExceeded` from `Shutdown` is followed by `Close`, which drops a handler still inside `Hijack`, then `Stop`. `packages/websocket` uses the hook to reject new upgrades and send close frame 1001.
- Per-client share of the in-flight body budget (`HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT`, default 25% of the budget; negative disables it). The share applies only when clients can be told apart: a trusted proxy plus a resolved client address, or a global-unicast peer. Loopback, RFC1918, link-local, CGNAT (`100.64.0.0/10`), and IPv6 unique-local peers with no trusted proxy do not take a share; only the global budget applies, and the first such request logs one warning with no address. `zatrano doctor` reports a missing `TRUSTED_PROXIES` as APP-HTTP-007. Behind a CDN, unset `TRUSTED_PROXIES` keys the share on the edge address, so many visitors share one quarter of the budget; set `TRUSTED_PROXIES` to the CDN ranges and the client address comes from `X-Forwarded-For`. IPv6 keys are a /64. Over the share is 503 with `Retry-After: 1`. A slow body can still occupy its share until the read timeout; a body-progress timer is rawhttp v0.3.0.
- In-flight body budget. `HTTP_MAX_INFLIGHT_BODY_BYTES` defaults to 256 MiB; a negative value turns it off. At header time a known `Content-Length` reserves that many bytes, and a chunked body reserves the worst case (JSON, urlencoded, and `text/*` use `MAX_BODY_BYTES`; everything else uses the effective cap). If the remainder cannot hold it, the engine returns 503 with `Retry-After: 1` and does not read the body. One request whose effective cap is larger than the whole budget is 413. The reservation is released once, from the handler or from connection close. A route `BodyLimit` above the budget logs a warning at boot, fails boot in production, and fails boot in every environment when `HTTP_STRICT_LIMITS` is set. `zatrano doctor` reports the same case as APP-HTTP-006.
- `Expect: 100-continue` is covered on the live server. A `Content-Length` over the effective cap, or a reservation that does not fit, is 413 or 503 with no `100 Continue`. The connection lingers so the client can read that response. rawhttp 0.2.3 caps lingering closes at 1024 (`MaxLingering`; negative is unlimited).

### Fixed

- Header-time body cap matches V2 for non-text bodies. `application/json` (and `+json`), `application/x-www-form-urlencoded`, and `text/*` are rejected at `MaxBodyBytes` (2 MiB) when the headers arrive. Multipart, `application/octet-stream`, `image/*`, an unknown type, and a missing `Content-Type` keep the server ceiling `MaxRequestBytes` (32 MiB). `Request.Body` and `Request.JSON` still stop at 2 MiB.
- `Route.BodyLimit(n)` overrides that cap for one route (`n` bytes; negative uses the server ceiling). The router is matched only when `Content-Length` is above the content-type default or the body is chunked. GET, HEAD, and smaller bodies do not pay for a lookup.
- WebSocket handshakes can reach `Hijack`. Admission order: `HTTP_ALLOW_UPGRADE=false` forces it off; then `ListenOptions.AllowUpgrade` and `serve --allow-upgrade`; then `HTTP_ALLOW_UPGRADE=true`; then `RegisterUpgradeProtocol`; then a linked `websocket` addon; then `EnabledAddons`. Default is off. `Upgrade: h2c` is still rejected. A websocket handshake that is not hijacked closes after the response.
- `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_IDLE_TIMEOUT`, and `HTTP_READ_HEADER_TIMEOUT` configure the server. Unset keeps 60s, 60s, 120s, and 10s. `0`, `0s`, and any negative value disable that deadline. A unitless integer is seconds. `30s` and `1m` are accepted. An invalid value aborts boot. The default read timeout can cut a slow upload: 32 MiB in 60s is about 4.4 Mbit/s.
- `Stream` and `StreamBody` re-arm `now+HTTP_WRITE_TIMEOUT` before each write and flush. A client that stops reading is closed within that timeout. A slow reader that keeps making progress is not cut. When the write timeout is off, the deadline is cleared. Only `Hijack` always clears it. `ClearWriteDeadline` opts out and accepts the slow-reader risk.

### Changed

- Require rawhttp v0.2.4. There is no `replace` directive.
- Retract `v3.0.0` (its `go.mod` contained local `replace` directives). Release CI rejects a module with `replace`, checks that `VERSION` equals the tag without the `v` prefix, installs `zatrano@<tag>`, runs `zatrano new`, and builds the generated app.

| Variable | Default | Meaning |
| --- | --- | --- |
| `MAX_BODY_BYTES` | 2 MiB | JSON, urlencoded, and `text/*` header cap. Also `Request.Body` / `JSON`. |
| `MAX_UPLOAD_BYTES` | 32 MiB | Multipart ceiling. The server ceiling is the larger of this and `MAX_BODY_BYTES`. |
| `HTTP_READ_TIMEOUT` | 60s | Request read deadline. `0` or negative disables it. |
| `HTTP_WRITE_TIMEOUT` | 60s | Response write deadline, re-armed per stream chunk. `0` or negative disables it. |
| `HTTP_IDLE_TIMEOUT` | 120s | Keep-alive idle deadline. `0` or negative disables it. |
| `HTTP_READ_HEADER_TIMEOUT` | 10s | Header read deadline. `0` or negative disables it. |
| `HTTP_MAX_HEADER_BYTES` | 16 KiB | Header-block ceiling in bytes. rawhttp allocates the connection read buffer from it. Above 64 KiB logs a warning. Over the limit the engine returns 431. |
| `HTTP_ALLOW_UPGRADE` | unset (off) | `false` forces WebSocket admission off. `true` turns it on unless `AllowUpgrade` is explicitly false. |
| `HTTP_MAX_INFLIGHT_BODY_BYTES` | 256 MiB | In-flight body budget. Negative disables it. A known length reserves that length; chunked reserves the worst case. Not enough remaining budget is 503 with `Retry-After: 1`. One request whose cap exceeds the budget is 413. |
| `HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT` | 25% of the budget | Per-client share when the client can be told apart. Negative disables it. Loopback, private, link-local, CGNAT, and unique-local peers with no `TRUSTED_PROXIES` skip the share. IPv6 keys are a /64. Over the share is 503 with `Retry-After: 1`. |
| `HTTP_STRICT_LIMITS` | off | When set, a route `BodyLimit` above the in-flight budget fails boot outside production as well. |

## 3.0.1 - 2026-10-02

Patch after `v3.0.0`. Product `VERSION` stays `3.0.0`. Kernel ABI is unchanged.

### Changed

- Drop local `replace` directives for `canvas` and `rawhttp`. The module requires published `canvas v0.2.0` and `rawhttp v0.2.2`.
- First-time enablement pins `github.com/zatrano/packages@v1.14.0`.

Install with `go get github.com/zatrano/framework/v3@v3.0.1` and `go get github.com/zatrano/packages@v1.14.0`.

## 3.0.0 - 2026-10-02

V3 is stable. HTTP carrier is **rawhttp**; SSR is **Canvas** (`framework/v3/core/ssr`). Web apps require Canvas; API handlers stay JSON-capable.

### Breaking

- Transport is `rawhttp` only (`Application.Handle`); no `net/http` server path.
- Web scaffold uses `http.Template` + `templates/`; `http.HTML` in web handlers fails doctor APP-CTL-006.
- Module path remains `github.com/zatrano/framework/v3`; product VERSION `3.0.0`.

### Added

- `ssr.TryRender` for infrastructure error pages (exceptions, maintenance, docs 404).
- Starter Canvas stubs under `errors.*`.
- Doctor APP-SSR-001 / APP-HTTP-005 leak seals.

### Changed

- rawhttp pin **v0.2.2** (GA for application embedding); canvas pin **v0.2.0**.
- `WriteTo` production path fails loud (Commit-only).

## 3.0.0-rc.1 - 2026-10-02

First V3 release candidate. Superseded by `3.0.0`.

## 2.8.1 - 2026-09-20

Patch after `v2.8.0`. Kernel ABI (`contracts.App`) and ORM public API are unchanged.

### Changed

- First-time enablement pins `github.com/zatrano/packages@v1.13.1` (framework `v2.8.0` pin so `ratelimit` compiles `middleware.Throttle`).

Install with `go get github.com/zatrano/framework/v3@v2.8.1` and `go get github.com/zatrano/packages@v1.13.1`.

## 2.8.0 - 2026-09-20

HTTP protocol primitives (Accept negotiation, HTTP 429 throttle) live in the kernel. Kernel ABI (`contracts.App`) and ORM public API are unchanged.

### Breaking

- Content negotiation is a kernel HTTP primitive (`http.Negotiate`, `middleware.Negotiate`). The catalog no longer lists `negotiate` as a package.

### Added

- `http.Negotiate` / `http.NegotiatedFormat` / `middleware.Negotiate` (one Accept parser on `Request.Prefers`).
- `middleware.Throttle` behind atomic `AttemptLimiter.Take`. `packages/ratelimit` implements the limiter; the kernel does not import that package.

### Changed

- Accept negotiation honors RFC quality values (`q=0` is not acceptable; higher `q` wins). No match still falls back to the first offered format.
- First-time enablement pins `github.com/zatrano/packages@v1.13.0`.

Install with `go get github.com/zatrano/framework/v3@v2.8.0` and `go get github.com/zatrano/packages@v1.13.0`.

## 2.7.0 - 2026-09-19

Catalog `events` is now `facts`. Kernel ABI (`contracts.App`) and ORM public API are unchanged.

### Breaking

- Catalog `events` is now `facts`. `package:enable facts` writes the Fact/Reaction capability (`facts.From(app)`). The empty `EventServiceProvider` scaffold is gone.

### Changed

- First-time enablement pins `github.com/zatrano/packages@v1.12.0`.

Install with `go get github.com/zatrano/framework/v3@v2.7.0` and `go get github.com/zatrano/packages@v1.12.0`.

## 2.6.4 - 2026-09-19

Patch after `v2.6.3`. Kernel ABI (`contracts.App`) and ORM public API are unchanged.

### Changed

- Catalog copy presents `authorization`, `apitoken`, `oauth`, and `social` as the auth domain. `package:enable` writes nested imports (`auth/authorization`, `auth/token`, `auth/oauth`, `auth/social`) and strips leftover top-level blank-imports from `bootstrap/addons.go`. Enable names are unchanged. `webauthn` stays `github.com/zatrano/packages/webauthn`.
- First-time enablement pins `github.com/zatrano/packages@v1.11.0`.

Install with `go get github.com/zatrano/framework/v3@v2.6.4` and `go get github.com/zatrano/packages@v1.11.0`.

## 2.6.3 - 2026-09-19

Patch after `v2.6.2`. Kernel ABI (`contracts.App`) and ORM public API are unchanged.

### Performance

- Frozen routes compose the global+route middleware chain once at `Router.Freeze` instead of reallocating closures on every request.
- `NewRequest` no longer pre-allocates route params, attributes, or a cookie jar; `DrainCookies` skips the jar when nothing was queued. Unnamed static matches do not store an empty `_route` attribute.

Install with `go get github.com/zatrano/framework/v3@v2.6.3` and `go get github.com/zatrano/packages@v1.10.0`.

## 2.6.2 - 2026-09-19

Patch after `v2.6.1`. Kernel ABI (`contracts.App`) and ORM public API are unchanged.

### Fixed

- Production `public/` index no longer treats a regular nested directory as a symlink tree when `EvalSymlinks` canonicalizes the runner temp/checkout path (Windows CI).

Install with `go get github.com/zatrano/framework/v3@v2.6.2` and `go get github.com/zatrano/packages@v1.10.0`.

## 2.6.1 - 2026-09-19

Patch after `v2.6.0`. Kernel ABI (`contracts.App`) and ORM public API are unchanged.

### Performance

- Production `publicFile` builds a one-time index of `public/` so GET/HEAD requests that cannot be static files skip `EvalSymlinks`/`Stat`. Files added under non-symlink directories after boot are not served until restart. Symlink trees (for example `public/storage`) stay on the existing slow path.
- `TrimStrings` / `ConvertEmptyStringsToNull` queue input transforms and apply them on the first `Input`/`All` (or `Merge`/`Replace`/`Forget`) access, so `JSON()`/`Body()` handlers skip JSON-to-map parsing. `Raw().Form` is unchanged until those accessors run.

Install with `go get github.com/zatrano/framework/v3@v2.6.1` and `go get github.com/zatrano/packages@v1.10.0`.

## 2.6.0 - 2026-09-15

Why now: Put API path versioning in the kernel so applications and `make:auth` do not import a library addon for `/api/{version}`.

Minor release after `v2.5.1`. Kernel ABI (`contracts.App`) and ORM public API are unchanged.

### Breaking / DX

- API path versioning is kernel `routing.Version` / `FromRequest` / `RequireVersion`. The `api` library is gone from the CLI catalog; there is no `packages/api`. `RegisterAPI` still does not prefix `/api`.
- First-time enablement pins `github.com/zatrano/packages@v1.10.0`.

Install with `go get github.com/zatrano/framework/v3@v2.6.0` and `go get github.com/zatrano/packages@v1.10.0`.

## 2.5.1 - 2026-09-15

Patch after `v2.5.0`. Kernel ABI (`contracts.App`) and ORM public API are unchanged.

- Restored `docs` in the CLI catalog (markdown encyclopedia addon).
- Added `seo`: classic crawler SEO (sitemap, robots, OG/JSON-LD, security.txt) and LLM discovery (`llms.txt`, `ai-plugin.json`) as one addon. `sitemap` and `wellknown` stay removed; consume `seo`. Does not import `ai`.
- Starter smoke blank-imports `audit`, not the removed `billing` addon.
- First-time enablement pins `github.com/zatrano/packages@v1.9.1`.

Install with `go get github.com/zatrano/framework/v3@v2.5.1` and `go get github.com/zatrano/packages@v1.9.1`.

## 2.5.0 - 2026-09-15

Why now: Pin the catalog freeze so applications can take toolkit ownership, nested pagination/TOTP/OTP, and dropped fake addons without tracking `main`.

Minor release after `v2.4.0`. Kernel ABI (`contracts.App`) and ORM public API are unchanged.

### Breaking / DX

- Boot membership is Requires-only (`Expand` / `Resolve` do not pull imported Optional addons into the enable-set). Optional still orders members already selected.
- HTTP surfaces: `web`, `api`, `auth/{web,api}`, and `make:panel {name}`. `zatrano doctor` APP-ROUTE-001 / APP-CTL-001 accept `app/routes/auth/{web,api}` and named panel folders. Auth JSON controllers under `handlers/auth/api` are APP-CTL-004. `package:enable template` creates `templates/web` and `templates/layouts`; the view package writes `layout/app.html` and `web/welcome.html` when missing and switches the starter home to `http.Template("web.welcome")`.
- Dropped `billing` from the CLI catalog. String enums and other import-only helpers live under `toolkit/` (`enums`, `collection`, `bloom`, `circuit`, `concurrency`, `cron`, `debug`, `process`, `timing`, `markdown`, `zip`, `jsonschema`). No `make:enum`, no enablement. User-Agent parsing stays in the kernel (`http.ParseUserAgent` / `req.Agent()`); there is no addon copy.
- Removed `octane`, `pulse`, and `inspector` from the CLI catalog. Metrics stay on `observability`. Application version is `app.Version()` / the kernel `version` CLI; there is no `packages/version` addon. Dropped site addons `search`, `shorturl`, `sitemap`, `wellknown`, `geo`, and `docs`. Page metadata lives in `orm/pagination`. TOTP lives in `auth/totp`. Numeric codes live in `notification/otp`. `hashid` and `lock` are toolkit libraries.
- Removed `bus`, `features`, and `tenancy`. Command dispatch stays on `events` and application services. Feature flags stay in config/env. Tenant resolution is application middleware, not a first-party package. `workflow` is catalogued as an experimental intelligence library (import-only; `agent.AsExecutor`).

Install with `go get github.com/zatrano/framework/v3@v2.5.0` and `go get github.com/zatrano/packages@v1.9.0`.

## 2.4.0 - 2026-09-15

Why now: Publish the kernel cleanup as a pinable minor so applications can take opt-in starter packages, flattened CSRF and HTTP helpers, and toolkit extraction without tracking `main`.

Minor release after `v2.3.1`. Kernel ABI (`contracts.App`) and ORM public API are unchanged. DX import paths and the default starter surface change.

### Breaking

- Removed `str.Random`. Randomness is only `support.RandomBytes`, `support.RandomHex`, `support.RandomBase64` (all return errors) and `support.MustRandomHex` (panics on entropy failure). `str` is deterministic string operations only.
- Moved `kernel/support/{date,arr,str,num,html,money,color}` to `github.com/zatrano/packages/toolkit/{date,arr,str,num,html,money,color}` (opt-in `LayerAddon` libraries). No re-export. Import `github.com/zatrano/packages/toolkit/str` (and siblings). Resource route parameter inflection is unexported in `kernel/routing`. Remaining `kernel/support` surface: entropy/`When` helpers, `uuid` (used by first-party packages), `files`, `fn`, `once`.
- Split `console` into `console/doctor`, `console/pkgmanager`, `console/scaffold`, `console/describe`, and `console/consolecore`. Root `console` keeps the CLI kernel (`console.go`, `cli_exit.go`, `version.go`, `service.go`, `agents.go`, `utility.go`, `env.go`, `env_seed.go`, `cache_commands.go`, `deploy.go`, `storage.go`, `exception.go`, `addon_cli.go`, `apppaths.go`). Import paths for those command groups change; `cmd/zatrano` still uses `console.New`.
- Removed `kernel/internal`. Resource name inflection lives unexported in `kernel/routing`.
- Flattened `kernel/middleware/csrf` into `kernel/middleware` (`CSRF`, `CSRFExcept`, `CSRFToken`). No re-export package.
- Flattened `kernel/http/useragent` into `kernel/http` (`Agent`, `ParseUserAgent`).
- `zatrano new` enables only `health` by default. `assets`, `localization`, `view`, and `validation` are opt-in (`package:enable`).

### Changed

- Split `kernel/http` large files without behavior change: `response.go` (core status/header) / `response_render.go` / `response_stream.go` / `response_redirect.go`; `input.go` / `input_form.go` / `input_json.go`; `request.go` / `request_headers.go` / `request_files.go`.
- `zatrano new` starter `app/` contains only `http`, `providers`, and `routes`. Opt-in package directories are created by `make:*` or `package:enable`.
- Framework repo root must not contain consumer trees (`storage/`, `app/`, `public/`, `views/`, `routes/`). `zatrano doctor` on this module runs `FW-ROOT-001`.
- `ai`, `rag`, and `agent` are catalogued as `Stability: experimental` until they complete the same security review as the rest of the ecosystem. `zatrano describe` prints a warning.
- README leads with `describe` / `doctor` / `agents:generate` before Learning ZATRANO. No Laravel-style marketing copy.

Install with `go get github.com/zatrano/framework/v3@v2.4.0` and `go get github.com/zatrano/packages@v1.8.0`.

## 2.3.1 - 2026-09-11

Patch release after `v2.3.0`. Kernel ABI, contracts, and ORM public API are unchanged.

### Breaking / DX

- Removed `zatrano add:web` and `zatrano add:api`. `zatrano new` already writes HTML `/` and JSON `/api` from a single `templates/web` tree (including `validation`). A framework upgrade still does not regenerate application source (G-001).

Install with `go get github.com/zatrano/framework/v3@v2.3.1` and `go get github.com/zatrano/packages@v1.7.2`.

## 2.3.0 - 2026-09-11

Minor release after `v2.2.1`. Kernel ABI, contracts, and ORM public API are unchanged.

### Breaking / DX

`zatrano new` generates one application: HTML at `/` and JSON at `/api`. Presentation packages `assets`, `health`, `localization`, `view`, and `validation` are enabled; other packages stay opt-in.

Canonical application directories live in `kernel/dirs` (`dirs.go`). HTML layouts stay in the `view` package.

Install with `go get github.com/zatrano/framework/v3@v2.3.0` and `go get github.com/zatrano/packages@v1.7.2`.

## 2.2.1 - 2026-09-11

Patch release. Kernel ABI, contracts, and ORM public API are unchanged.

### DX

- `zatrano new` seeds `.env` from `.env.example`. `key:generate` does the same when `.env` is missing, so the printed first-run steps no longer fail with "file not found".
- Generated `AGENTS.md` and the API starter README state that applications do not ship a `docs/` tree, and that `--api` keeps shared placeholder dirs (`templates`, …) without enabling the view package.
- Application engineering spec moved to [zatrano.com/docs/application-engineering](https://zatrano.com/docs/application-engineering). This repository no longer has a `docs/` tree. Root `AGENTS.md` and `zatrano doctor` point at the site.

Install with `go get github.com/zatrano/framework/v3@v2.2.1` and `go get github.com/zatrano/packages@v1.7.2`.

## 2.2.0 - 2026-09-10

Minor release after `v2.1.0`. Kernel ABI, contracts, and ORM public API are unchanged. The new work is application-architecture enforcement, documentation freeze, and pairing with Packages `v1.7.2`.

### New

- `zatrano doctor` enforces the high-confidence Application Engineering STANDARD: forbidden layers, controller transaction ownership, View/JSON mixing (with `make:auth` exception), string eager loads, FormRequest naming, `validation.Make` in handlers/services/models, persist-without-ValidateForm, repository *interfaces*, and `unique`/`exists` without `database`. Errors exit 1. `--json` and `--strict` are supported. Catalog: [application-engineering/rules](https://zatrano.com/docs/application-engineering/rules). Adversarial boundary: [doctor-boundary](https://zatrano.com/docs/application-engineering/doctor-boundary).

### Architecture

- Application Engineering Standard **frozen** (ADR-0011): one canonical path per concern, completeness matrix, [no-second-way](https://zatrano.com/docs/application-engineering/no-second-way). Kernel/ORM/API unchanged. Semantic doctor-PASS stacks remain documented, not silently closed.
- Fail-closed `unique` / `exists` (ADR-0010 amended). Runtime is Packages `v1.7.2`: those rules no longer silently succeed when the database checker or required infrastructure cannot determine the result.
- Platform conformance audit: [platform-audit](https://zatrano.com/docs/application-engineering/platform-audit).

Install with `go get github.com/zatrano/framework/v3@v2.2.0` and `go get github.com/zatrano/packages@v1.7.2`. Public module-proxy consumption of these tags is pending until they are pushed.

## 2.1.0 - 2026-09-09

### Breaking / DX

Default `zatrano new myapp` now generates an **empty** application: canonical layout, kernel HTTP/CLI, bootstrap, tests, and storage, with `EnabledAddons=[]` and no `github.com/zatrano/packages` dependency. Previous default was Web-oriented. This is a deliberate DX break and the reason for the 2.1.0 minor boundary under the pre-3.0 compatibility policy.

`--minimal` is removed as a user-facing scaffold. It fails loudly; it is not an alias to empty or API. `APP_BOOT=minimal` remains a legacy runtime boot alias only.

### New

- `zatrano new myapp --web` — full-capacity HTML presentation defaults (`assets`, `health`, `localization`, `view`). `GET /` is HTML.
- `zatrano new myapp --api` — full-capacity JSON/API presentation defaults (`health`, `validation`). `GET /` is JSON.
- `zatrano new myapp --full` — Web + API presentation composition (not database, auth, queue, notifications, AI, RAG, agents, or every package). Canonical starter: HTML `/` and JSON `/api`.
- `zatrano add:web` / `zatrano add:api` — composable presentation overlays. Idempotent, non-destructive, conflict-aware; exact empty-stub replacement via `bytes.Equal` only. They preserve the original root: `new --web` then `add:api` keeps HTML `/`; `new --api` then `add:web` keeps JSON `/`. That is intentional and different from `--full`.

`--web`, `--api`, and `--full` are mutually exclusive.

### Architecture

- Empty / Web / API / Full are explicit profiles. Web and API are full-capacity; Full is presentation composition (Web Apply + API Overlay).
- Generator remains Apply + Overlay. Overlay may write a missing file, treat an identical file as present, replace an exact known empty stub, or skip a user-modified file. No fuzzy matching.
- G-001 preserved: `zatrano upgrade` does not invoke `add:web` / `add:api` or regenerate application source. `add:*` does not rewrite `bootstrap/scaffold.go`.
- Framework does not depend on `github.com/zatrano/packages`. Empty generated applications also do not.
- `add:web` / `add:api` stay presentation composition. `package:enable` stays capability activation. `package:preset` stays empty and unwired.
- `package:doctor` distinguishes optional application `VERSION` (generated apps omit it) from framework identity (`go.mod` require, or `VERSION` in the framework tree). Generated API JSON no longer imports `packages/version` as an unenabled service.

Documentation and CLI catalog remain aligned with Packages `v1.7.1`. Install with `go get github.com/zatrano/framework/v3@v2.1.0`. Public module-proxy consumption of `v2.1.0` is pending until this tag is published.

## 2.0.28 - 2026-09-08

Acquisition production hardening: real CLI acquisition E2E, classified CLI exit codes, JSON inspection/recovery/target reporting, observable recovery failures, `--timeout` context propagation, enablement consistency checks, official/heavy/`framework_min`/tagged ecosystem validation, and CI acquisition E2E. Apply contract and Acquisition/enablement stay frozen. No `func Apply`. `package:install` stays enablement.

Runtime lifecycle hardening: deterministic boot order tests, lifecycle contract tests, Start-failure cleanup via `errors.Join`, `BootstrapContext` / `StartContext` (zero-arg methods remain), Enabled ∩ Imported and process-global registry contracts, `framework_min` agreement tests, isolated acquire→enable→Start/Stop E2E, and runtime CLI exit codes 20–23 (`serve` / `Run` never reuse acquisition 2–7). Apply/orchestration/hardening stay frozen. No `func Apply`. `package:install` stays enablement.

Enablement dependency-safe package lifecycle: `package:enable` writes the transitive `Requires` closure (Optional excluded) before mutating files; `package:disable` refuses when a remaining enabled addon requires the target, is a successful no-op when already disabled, and does not call Stop; enablement wiring preserves an existing `github.com/zatrano/packages` module pin instead of `go get @main`. Apply, orchestration, hardening, and runtime contracts stay frozen. No `func Apply`. No `package:upgrade` / `package:uninstall`. `package:install` stays enablement. Install with `go get github.com/zatrano/framework/v3@v2.0.28`.

## 2.0.27 - 2026-09-08

Open Contract C: `package:acquire --enable` reuses existing `enablePackage` after successful acquisition. Default acquire does not enable. Acquisition and enablement stay separate (no implicit transaction, no automatic rollback). `package:install` stays enablement. Contract A and Apply contract stay frozen. No `func Apply`. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.26 - 2026-09-08

Lock the only valid Contract C path: explicitly open C → inspect current boundaries → define/lock C tests → implement C. C must not be implemented, tested as an implementation, or wired into acquisition while closed. `package:acquire` stays orchestration; `package:install` stays enablement. Contract A and Apply contract stay frozen. No `func Apply`. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.25 - 2026-09-08

Freeze Contract B as implemented (`package:acquire` orchestration). Contract A stays complete / frozen. Contract C remains closed; the next step is not automatically C. Apply contract (`v2.0.22`) stays frozen. No `func Apply`. `package:install` stays enablement. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.24 - 2026-09-08

Contract A (`DryRun` / `DryRunTargets`) and Contract B (`package:acquire` CLI orchestration). No `func Apply`. `package:install` stays enablement. Contract C remains closed. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.23 - 2026-09-08

Record Acquisition/enablement bounding SPEC draft ([`ORCHESTRATION.md`](distribution/acquire/ORCHESTRATION.md)): dry-run, CLI acquisition, and acquisition ↔ enablement as three independent contracts. Implementation is not authorized. Apply contract stays frozen. `package:install` stays enablement. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.22 - 2026-09-07

Freeze Apply contract: steps 1–8 complete. Surface stays FromResult → Targets → GoGetArg → Execute / ExecuteTargets → Inspect → ApplyResult → SnapshotFiles / RecoverFiles. No `func Apply`, dry-run, CLI, tidy, `zatrano.lock`, or `package:install` change. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.21 - 2026-09-07

Apply contract integration tests: Plan → Targets → GoGetArg → Execute / ExecuteTargets → Inspect → RecoverFiles on a real module root. No new Apply API, resolver, tidy, `zatrano.lock`, or `package:install` change. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.20 - 2026-09-07

Apply contract file recovery: `RecoverFiles` restores a `go.mod` / `go.sum` snapshot (best-effort). It is not transactional rollback and does not undo the module cache. `ExecuteTargets` still does not restore files. No tidy, `func Rollback`, or `package:install` change. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.19 - 2026-09-07

Apply contract partial apply: `ExecuteTargets` reports successful, failed, and unattempted targets (fail-fast). Earlier successes are not rolled back. No tidy, `func Apply`, or `package:install` change. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.18 - 2026-09-07

Apply contract per-root mutation lock: same module root cannot run two `Execute` mutations at once; other roots proceed independently. `Inspect` is not locked. No tidy, `ApplyResult`, partial apply, or `package:install` change. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.17 - 2026-09-07

Apply contract `Inspect` reads go.mod / go.sum after `Execute`. `InvocationResult` is the process; `Inspection` is module state. No tidy, `ApplyResult`, concurrency, or `package:install` change. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.16 - 2026-09-07

Apply contract `go get` execution (`Execute` / `ExecRunner`). Result is `InvocationResult`; no go.mod / go.sum inspection, tidy, `ApplyResult`, or `package:install` change. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.15 - 2026-09-07

Apply contract process invocation boundary (`Invoke` / `Runner`). Fake process in tests; no `go get` execution, no go.mod mutation, `package:install` unchanged. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.14 - 2026-09-07

`deploy:build` compiles the generated application (`./cmd/app`), not the host CLI. Apply contract stays SPEC-only; contract tests lock [`APPLY.md`](distribution/acquire/APPLY.md). Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.13 - 2026-09-07

Nest the package distribution protocol under `distribution/` (`manifest`, `registry`, `acquire`). Import paths are `github.com/zatrano/framework/v3/core/distribution/...`. JSON schemas (`zatrano.package/v1`, `zatrano.registry/v1`) are unchanged. Apply contract stays SPEC-only. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.12 - 2026-09-07

Open Apply contract as Apply SPEC only ([`distribution/acquire/APPLY.md`](distribution/acquire/APPLY.md)): mutation boundary, `go get` invocation, fail-fast, no automatic `tidy`, rollback guaranteed vs unavailable. Implementation is not authorized. `package:install` stays enablement. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.11 - 2026-09-07

Record Apply contract entry: first artefact is the Apply contract (mutation, `go get` invocation, go.mod/go.sum failures, concurrency, partial apply, rollback). `package:install` stays enablement at Apply contract start. `tidy` is not an acquisition lockfile. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.10 - 2026-09-07

Close Acquisition plan: `Targets` deduplicates modules and does not resolve pin conflicts. Apply contract starts with a contract; `package:install` stays enablement; `go get` → `go mod tidy` is not an install assumption. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.9 - 2026-09-07

Freeze Acquisition Plan layer: `FromResult` is a pure translation; `Targets` collapses shared modules into a deterministic query list. Same Result → same Plan; `latest` never survives; unresolved → no Plan. Apply (`go get`) remains a later specification. `package:install` stays enablement. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.8 - 2026-09-07

Exhaustive `acquire.Plan` contract tests: shared vs heavy modules, tagged/`main`/`latest`→concrete, missing identity, incompatible Resolve (no plan). Apply still deferred. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.7 - 2026-09-07

Module acquisition contract (`acquire`): map `registry.Result` to a `go get` plan. No `zatrano.lock` — pins stay in go.mod / go.sum. No Apply / no change to `package:install`. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.6 - 2026-09-07

Freeze the registry CLI consumer (`package:search` / `info` / `resolve`). Module acquisition is a separate contract, not more resolution in the CLI. `package:install` stays enablement. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.5 - 2026-09-07

Harden registry Search/Resolve contracts (framework constraints, `main` fallback, heavy modules) and lock CLI output plus the invariant that console must call `registry.Resolve` rather than reimplement it. `package:install` stays enablement. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.4 - 2026-09-07

CLI registry consumers: `package:search` (discovery), `package:info` (identity), `package:resolve` (version selection). They call `registry.Search` / `Resolve`; they do not install, enable, or own the algorithm. Channel `main` is a source stream, not a published release. Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.3 - 2026-09-07

Package registry data model (`zatrano.registry/v1`): in-memory index, discovery, and version resolution. Not an HTTP service or marketplace. Artifact versions follow the Go module path (official packages: channel `main`). Install with `go get github.com/zatrano/framework/v3@latest`.

## 2.0.2 - 2026-09-07

Package contract freeze and the `zatrano.package/v1` distribution protocol. Install with `go get github.com/zatrano/framework/v3@latest`.

### Added

- Package manifest schema (`manifest`): name, import, kind, layer, optional requires. Not a second boot path; Enabled ∩ Imported is unchanged.
- Optional `addons.Meta.FrameworkMin`; `package:doctor` errors when the running kernel is older.

### Fixed

- `package:doctor` no longer treats every `addons.Register` name as KindService. Libraries may register for CLI or a no-op provider (`agent`, `rag`, `factory`, `openapi`).
- Duplicate addon registration stays a panic, now covered by tests.
- CLI catalog names stay aligned with packages `Register` names; foundation catalog kinds are explicit.

## 2.0.1 - 2026-09-06

First GOPROXY-valid `/v2` module version. `v2.0.0` was tagged before `go.mod` declared `module github.com/zatrano/framework/v3`; `proxy.golang.org` cached that as invalid. Install with `go get github.com/zatrano/framework/v3@v2.0.1`.

## 2.0.0 - 2026-09-06

v2 is the default line on `main`. The Go module path is `github.com/zatrano/framework/v3`. Use `@v2.0.1` (not `@v2.0.0`) so the public module proxy accepts the version.

### Breaking

- The framework module no longer ships an application skeleton. `app/`, `routes/`, `views/`, `public/`, `lang/`, and application `database/` migrations live in apps created by `zatrano new`. `cmd/zatrano` is the CLI; generated apps use `cmd/app`. Optional addons live in `github.com/zatrano/packages` and must be blank-imported by the consumer.
- `bootstrap.ApplicationProviders()` is empty; pass `bootstrap.WithProviders(...)` from the application.
- No database is linked by default. SQLite is installed with `db:setup --drivers=sqlite` like MySQL and PostgreSQL. Apps boot without `DB_CONNECTION`. The framework tree no longer has a placeholder `database/` directory.
- `App(WithDemo())`, `DemoAddons`, `APP_BOOT=demo`, and `package:preset demo` are removed. Enable addons explicitly with `EnabledAddons` or `App(WithAddons(...))`.
- Duplicate addon libraries (`backup`, `cron`, `enums`, `export`, `factory`, `octane`, `openapi`, `pagination`, `testing`, `timing`, `totp`, `useragent`) are no longer top-level under `packages/`. Foundation usage is nested (`auth/totp`, `schedule/cron`, `orm/pagination`, …); CLI-only copies live in `github.com/zatrano/packages`. `db:backup` registers when the backup addon is blank-imported.
- `contracts.App` no longer exposes package services (`RateLimiter`, `URL`, `Hash`, `Metrics`, `Health`, `Maintenance`). Resolve them with `From(app)` / `app.Make`. Kernel accessors remain on `*kernel.Application` for the CLI.
- Package config schemas (`auth`, `database`, `session`, `notifications`) moved out of `kernel/config` into the addon `DefaultConfig()` functions. Kernel config is the generic repository plus `app` defaults.
- `Meta.Requires` is now a hard dependency: missing names fail `Select`/`OrderMetas` instead of being skipped. Use `Meta.Optional` for “boot first if imported”. `WithAddons("auth")` expands imported requirements into the boot set. The default enable-set (`App()` without `WithAddons`) boots only imported addons whose Requires are present (`Bootable`); a helper import must not panic the process.

### Added

- Native `kernel/env` `.env` parser; `github.com/joho/godotenv` is removed. Process environment wins over file values. The framework module has zero third-party runtime dependencies.
- Factory container no longer holds its mutex while running factories (nested `Make` no longer deadlocks). Singletons use per-binding publish; alias chains and circular factories error instead of hanging. Cross-goroutine singleton cycles return an error instead of deadlocking. After bootstrap, `Bind`/`Singleton`/`Instance`/`Alias` panic (`Make` still publishes lazy singletons).
- Application bootstrap is idempotent and rejects `RegisterProviders` after boot. Bootstrap failure is terminal (`BootstrapFailed`); retrying the same instance is rejected. The router, config, and container registrations freeze at the end of bootstrap.
- Addon registry topological sort via `Meta.Requires` (missing requires are errors) and `Meta.Optional` (skipped when absent). `Select`/`Resolve` expand the dependency closure from the process registry. Duplicate addon names panic at `init`.
- After bootstrap the router compiles a per-method static map plus a segment trie for parameterized, optional, and catch-all routes (exact static paths win). Duplicate static paths and ambiguous parameterized shapes (`GET /users/{id}` vs `GET /users/{name}`) fail freeze. Config repository is frozen (Load/Set panic).
- JSON/`Body()` no longer close the raw body; bodies are capped (`MAX_BODY_BYTES`, default 2 MiB) and replayable. The HTTP server sets `MaxHeaderBytes` and wraps the body with an absolute ceiling (`MaxRequestBytes`, the larger of JSON and multipart limits).
- Optional `contracts.LifecycleProvider` (`Start`/`Stop`) for long-running workers. `Application.Start`/`Stop` run those hooks; `Run` starts them before listen and stops them on shutdown.
- Typed `ResolveOK` / `MustResolve` beside `Resolve`. Container nested `Make` uses an explicit resolution view (no goroutine-id parsing).
- Architecture tests freeze `contracts.App` methods and keep input helpers out of `kernel/http/request.go`. `tests/compatibility` golden-app: `zatrano new --minimal` must `go vet`/`go test` against this checkout.
- Production panic recovery no longer writes panic values into the response body. `X-Request-ID` is propagated when valid. Incoming `traceparent` supplies the request id when no `X-Request-ID` is present.
- Kernel `Catalog` lists primitive packages only. Foundation / intelligence / addon discovery lives in the CLI aggregator (`console`), so the kernel does not know names like `auth`, `billing`, or `agent`.
- Addon lifecycle snapshot: `addons.NewPlan` (`imported` vs `enabled`). `Application.EnabledAddons` records the enable-set.

### Changed

- README rewritten for v2: two-module layout, `zatrano new`, no default database, no demo boot profile. Language name shown as Golang.
- `contracts` no longer imports `packages/*`. `App`, `Provider`, and `Migrator` live in `contracts`. Addon providers take `contracts.App`.
- Addon config defaults live with the addon (`DefaultConfig()`); `kernel/config` is the generic repository plus `app` defaults.
- Kernel catalog no longer enumerates packages-module names; `zatrano describe` / `package:list` / doctor use the CLI catalog.
- Starter layout uses `templates`, `app/localization`, and `app/database` (old `views/`, `lang/`, `database/` still load if present).
- `make:*` scaffolds use the consumer `go.mod` module path. `package:enable` writes `bootstrap/addons.go` blank-imports and `go get github.com/zatrano/packages`.
- Product VERSION is `2.0.0` on `main`.
- `kernel/layout` renamed to `kernel/dirs` (application path helpers). HTML template layouts stay in the `view` package.
- CI checks out `zatrano/packages` as a sibling so `zatrano new --replace` tests can build; live SQL smoke lives in the packages module.

## 1.6.6 - 2026-08-30

### Added

- RAG `CrossEncoderReranker`, `LLMReranker` / `FromAIChat`, `FuncReranker` (already) documented
- Agent branching `Graph` with `RouteIf` / `RouteContains`

## 1.6.5 - 2026-08-30

### Added

- RAG rerank: `Reranker`, `KeywordReranker`, `QueryWith` / `QueryRerank`
- RAG `PGVectorStore` (Postgres + pgvector ANN, `EnsureIndex`)
- Agent sequential `Chain` / `CatalogChain`

## 1.6.4 - 2026-08-30

### Added

- Agent `ResultStore` (`MemoryResultStore`, `JSONFileResultStore`) wired into `Runner`
- RAG `SQLStore` durable vector store via `database/sql` (portable cosine search)

## 1.6.3 - 2026-08-30

### Added

- Agent queue integration: `Catalog`, `Runner`, `PushRun`, job name `agent.run`

## 1.6.2 - 2026-08-30

### Added

- RAG `JSONFileStore` durable vector store (JSON on disk)
- Agent built-in tools: `RegisterWebFetch`, `RegisterFileSearch`

## 1.6.1 - 2026-08-30

### Added

- `packages/agent` library: tool registry, `BufferMemory`, `Agent.Run` loop, `RAGRetrieve` / `FromManager`
- Catalog entry `agent` (KindLibrary)

## 1.6.0 - 2026-08-30

### Added

- `packages/rag` library: chunk → embed → `MemoryStore` / `VectorStore`, `Pipeline`, `FromAI`, `FormatContext`
- Catalog entry `rag` (KindLibrary)

## 1.5.7 - 2026-08-30

### Added

- AI speech: `SpeechDriver`, `TextToSpeech` / `SpeechToText`, `CapSpeech` (OpenAI audio + Fake)
- AI provider ranking: `SetPrices`, `PreferCheapest`, `PreferSmartest`
- AI image edit/variation: `EditImage`, `VaryImage` (`ImageEditor`)
- Anthropic tools + SSE stream; Gemini embed + SSE stream (+ CapVision)

## 1.5.6 - 2026-08-30

### Added

- AI image generation: `ImageDriver`, `GenerateImage`, `CapImage` (OpenAI Images API + Fake stub)
- AI `WatchHealth` background probes that reorder profile providers via `SetProfile`
- AI thin HTTP drivers: `Anthropic` / `Gemini` (`BuildDriver` ids `anthropic`/`claude`, `gemini`/`google`)

## 1.5.5 - 2026-08-30

### Added

- AI vision / multimodal messages: `ContentPart`, `UserVision`, `ImageURLPart`, `CapVision`

### Changed

- Rename `ChatResponse.UnmarshalJSON` → `DecodeJSON` (avoids `json.Unmarshaler` signature clash)

## 1.5.4 - 2026-08-30

### Added

- AI live routing: `ProfileLive` / `UsingLive`, `FilterHealthy` / `FilterCapable`, `FailIfNone` / `RequireCaps`

## 1.5.3 - 2026-08-30

### Added

- AI provider health probes: `Healthy`, `CheckHealth` / `CheckHealthAll`, `HealthyProviders` (OpenAI `GET /models`)

## 1.5.2 - 2026-08-30

### Added

- AI `UsageMeter` / `PriceTable` / `MeterSnapshot` for token and optional USD cost aggregation via `Observe`
- AI `MultiObserver` fan-out

## 1.5.1 - 2026-08-30

### Added

- AI streaming tool-call deltas: `StreamToolCallDelta`, assembled `ToolCalls` on Done, `CollectStream` helper

## 1.5.0 - 2026-08-30

### Added

- AI capability registry: `Capability`, `Capabler`, `Capabilities` / `Supports` / `Describe` / `DescribeAll`, optional `SetModels` metadata

## 1.4.5 - 2026-08-30

### Added

- AI tool calling: `Tool` / `FunctionTool`, `ToolCall`, `ToolChoice`, `ToolResultMessage`, `HasToolCalls`

## 1.4.4 - 2026-08-30

### Added

- AI structured output: `ResponseFormat`, `JSONObject` / `JSONSchema`, `ChatJSON`, `ChatResponse.UnmarshalJSON`

## 1.4.3 - 2026-08-30

### Added

- AI `Observe` hook (`Observer` / `FuncObserver`) with latency, usage, attempts, fallbacks
- AI `Drivers()` / `Profiles()` return sorted names

## 1.4.2 - 2026-08-30

### Added

- AI streaming: optional `StreamDriver`, `ChatStream`, OpenAI-compatible SSE (`stream: true`)

## 1.4.1 - 2026-08-30

### Added

- AI error classification (`Kind`, `*Error`, `Classify`) with retry/backoff and smart profile fallback (auth/invalid stop; 429/5xx retry then fallback)
- AI retry config: `AI_RETRY_MAX`, `AI_RETRY_INITIAL_MS`, `AI_RETRY_MAX_MS`, `AI_FALLBACK_ON_TIMEOUT`

## 1.4.0 - 2026-08-30

### Added

- AI manager: named providers, `Using` / `Profile` clients, ordered profile fallback, `openai_compatible` driver, optional `EmbeddingDriver` (`Embed`)

### Changed

- AI package split into manager/client/profile/drivers; boot via `BootConfig` (flat `AI_*` config still supported)

## 1.3.4 - 2026-08-30

### Added

- CSRF `SkipAnonymousSeed` / `SetSessionCookieName`: skip token + `XSRF-TOKEN` cookie on safe methods without a session cookie (CDN-friendly)

## 1.3.3 - 2026-08-29

### Changed

- Skip `Set-Cookie` / session save for untouched anonymous sessions (CDN-friendly public pages)

## 1.3.2 - 2026-08-29

### Changed

- Default app no longer ships `make:auth` / `make:dashboard` generated files; use the CLI scaffolds and wire providers/routes yourself
- Empty application skeleton dirs (`events`, `listeners`, `jobs`, …) kept with `.gitkeep`

### Removed

- Route parameter constraints: `Where`, `WhereNumber`, `WhereUuid`, `WhereAlpha` (validate in handlers instead)

### Added

- Catch-all route params `{*slug}` / `{slug*}` (replaces docs `Where("slug", ".+")`)

## 1.3.1 - 2026-08-28

### Fixed

- Auth/dashboard scaffolds: remove forbidden dot imports (`http`/`routing`) for staticcheck ST1001; gofmt dashboard sources

## 1.3.0 - 2026-08-28

### Added

- `make:dashboard` CLI scaffold: modular admin shell (users, notifications, roles, RBAC matrix, settings, live analytics, impersonate, `/api/v1`)
- Dashboard stubs: CSS/JS shell (ZATRANO red), layouts, i18n (`en`/`tr`), migrations, `DashboardServiceProvider`
- Auth API routes versioned under `/api/v1/auth` via `api.Version`
- `DatabaseUserProvider.RetrieveByToken` uses hydrate (consistent with `RetrieveByID`)

### Docs

- README / PACKAGES links for dashboard scaffold; site guide at zatrano.com/docs/dashboard-scaffold

## 1.2.28 - 2026-08-24

### Added

- ORM eager parity: `Then`/`Nested`, `LoadMissing`, constrain `*Fn` loaders, `EagerCount`/`EagerExists`/`EagerMax|Min|Avg|Sum`
- ORM relations: `MorphToMany`/`MorphedByMany`, `WhereHasMorph`/`WhereHasThrough`, `withPivot` via `Pivot()`/`MarkPivot`
- ORM model DX: `Hidden`/`Visible`, `ToMap`/`ToJSON`/`ToArray`, accessors/mutators/`Appends`, `Defaults()`, UUID/ULID keys
- ORM strict/lazy tooling: `PreventLazyLoading`, `Cursor`/`Lazy`, `Collection`, `Without`/`WithOnly`, `Load`, relation subqueries
- Model lifecycle events: `saving`/`saved`/`retrieved`/`replicating`; `events.LifecycleObserver`

## 1.2.27 - 2026-08-23

### Security

- Google OAuth now fail-closes on missing/false `email_verified` (never assumes verified)
- `Persist` uses `User.EmailVerified` for `email_verified_at`; unverified emails cannot link to an existing account by email
- Production rejects StubProvider / placeholder OAuth credentials (`SetAllowStubProviders`, register-time fail-fast)
- GitHub primary email selection requires an explicit `verified=true` claim

### Added

- `User.EmailVerified`, `SetAllowStubProviders` / `AllowStubProviders`, `IsPlaceholder`, `ErrStubNotAllowedInProduction`
- Social security tests (verified/unverified/missing claim, account-linking attack, dual `sub`, stub prod/dev)

## 1.2.26 - 2026-08-23

### Fixed

- `query.Insert` no longer requires an `id` column on PostgreSQL/SQL Server; `InsertGetID` returns the PK. Fixes password reset token persistence and forgot-password mail delivery on pgsql.

## 1.2.25 - 2026-08-23

### Changed

- Auth notification emails (password reset, verify email, password changed) use `auth.mail_*` locale keys (`APP_LOCALE` + fallback); reset body is link-only

### Added

- `notification.SetTranslator` / `SetMailDefaults` (and Manager wrappers) for localized built-in auth mails
- Locale keys: `mail_reset_*`, `mail_verify_*`, `mail_password_changed_*` (en/tr defaults + make:auth stubs)

## 1.2.24 - 2026-08-23

### Fixed

- SMTP `MAIL_ENCRYPTION=tls`/`starttls` no longer uses implicit TLS; only `ssl` or port `465` select SMTPS (Gmail `:587` STARTTLS works again)

## 1.2.23 - 2026-08-20

### Changed

- Social OAuth scaffold flash messages use localization (`auth.social_*`) with `:provider` placeholders (en/tr)

### Added

- Auth locale keys: `provider_google`, `provider_github`, and social OAuth flash strings in defaults + make:auth lang stubs

## 1.2.22 - 2026-08-20

### Added

- AI chat defaults from config (`model`, `base_url`, `timeout`, `temperature`, `max_tokens`)
- `context.Context` on AI `Driver.Chat` / `Manager.Chat` with request timeout
- Demo route `POST /demo/ai/chat`

### Changed

- `LogDriver` logs prompt/reply via app logger `Infof`
- OpenAI-compatible driver wired from config (`AI_BASE_URL`, model, HTTP timeout)

## 1.2.21 - 2026-08-20

### Added

- Billing gateway drivers (`memory` / `stripe`) with `Manager.Extend` / `Use`, `Billable` helpers, and `POST /billing/webhook`
- Async billing receipt notifications (`InvoicePaidNotification`, `SubscriptionStartedNotification`) via `notification.Send`
- SMS multi-driver manager (`SmsManager.Extend` / `Use`, channels `sms` and `sms.<driver>`)

### Changed

- Split monolithic `packages/billing` into gateway/manager/webhook modules; config adds `billing.default`, Stripe webhook secret, checkout URLs
- Auth user stubs include `stripe_customer_id` and Billable methods

## 1.2.20 - 2026-08-20

### Changed

- Mail folded into `packages/notification` (removed `packages/mail`, `mail.From`, `make:mail`, and container `mail`/`sms`/`push` bindings)
- Notifications always dispatch asynchronously via `notification.Send` / `SendMany` (`SendNow` for sync/tests)
- Auth password reset, email verification (register / resend / email change), and password-changed notices go through notification channels

### Added

- Built-in notifications: `PasswordResetNotification`, `VerifyEmailNotification`, `PasswordChangedNotification`
- Auth helpers: `SendEmailVerification`, verification URL / sender hooks on the auth manager

## 1.2.19 - 2026-08-20

### Fixed

- Security CI: Trivy action pin, Semgrep noise split, gosec `#nosec` config, `APP_KEY` for race tests
- Bump `golang.org/x/crypto` and `golang.org/x/text` (incl. nested driver/mongo modules) for Trivy HIGH CVEs

## 1.2.18 - 2026-08-20

### Added

- Security CI pipeline: parallel gosec (SARIF), govulncheck, Semgrep, Trivy FS, and Go fuzz jobs with a Security Summary
- `tests/fuzz` targets for router, validation/HTTP binding, session/CSRF, and query builder
- `.gosec.json` exclude config and `security/semgrep/zatrano-rules.yml` custom rules scaffold

### Fixed

- Router path compilation no longer panics on invalid UTF-8 / bad regex patterns

## 1.2.17 - 2026-08-20

### Security

- Session: directory `0700`, atomic writes, cross-process file locks; remember-me cookie Secure on HTTPS
- Password reset / remember hashes: constant-time digest comparison helpers
- CSRF: Origin / Referer / `Sec-Fetch-Site` defense-in-depth (token still required)
- CORS: no wildcard default in production; credentials never combine with `*`
- Backup: restore path containment, `0600` files, credentials via temp files (not argv), error redaction, connection-name validation
- Runtime dirs (`cache`, `log`, `audit`, OAuth store, schedule locks, SQLite parent): `0700` / file `0600`

## 1.2.16 - 2026-08-19

### Fixed

- PDF TTF parsing: gosec G115 integer conversions (FWORD, cmap delta, BMP/Unicode runes)

## 1.2.15 - 2026-08-19

### Fixed

- PDF embeds a system TrueType font (Arial on Windows) for non-ASCII text so Turkish (and other) glyphs render; ASCII still uses Helvetica

## 1.2.14 - 2026-08-19

### Fixed

- staticcheck SA4006/U1000 in `packages/backup` (unused assignment and dead helper)

## 1.2.13 - 2026-08-19

### Added

- Multi-driver `db:backup` / `db:restore` / `db:backup:list` via native CLI tools (`mysqldump`, `pg_dump`, `sqlpackage`, `exp`/`imp`, `mongodump`) plus SQLite file copy; `--connection` flag

## 1.2.12 - 2026-08-19

### Added

- XLSX export (`xlsx.FromMaps` / `Response`) with round-trip import
- CSV options (delimiter, UTF-8 BOM) and `export.ToMaps` format detection
- PDF multipage text, `FromMaps` tables, and `Inline` for browser viewing

## 1.2.11 - 2026-08-19

### Fixed

- CSRF token generation panics if `crypto/rand` fails instead of ignoring the error

## 1.2.10 - 2026-08-18

### Fixed

- View compiler emits backtick string literals for `dataGet` / comparison / CSRF / auth paths so `@if` inside HTML attributes (e.g. `class="@if($save_disabled)…@endif"`) no longer nests double quotes in the compiled Go template

## 1.2.9 - 2026-08-18

### Changed

- Social avatar is a provider snapshot on `social_accounts`; canonical display photo is `users.avatar` (`Persist` / `make:auth` stubs)
- README origin story rewritten; CI and meta badges restored on two rows
- Security docs live on zatrano.com (`/docs/security*`); repo keeps slim `SECURITY.md` reporting policy

### Fixed

- Zip-slip containment in `zipx.Extract` + per-member size cap
- WebSocket / TOTP / bloom integer-conversion hardening for gosec G115
- gosec CI excludes for nested modules (`qr`, `mongo`, `webauthn`, optional SQL drivers)

## 1.2.8 - 2026-08-17

### Fixed

- Path containment treats `\` as a separator on all OS (Linux CI rejected Windows-style traversal)
- Upload filename sanitize strips `\` directories cross-platform
- CI Go toolchain pinned to **1.25.13** (stdlib govulncheck fixes)
- `golang.org/x/text` → v0.39.0
- gofmt on `trustedproxy_test.go`

## 1.2.7 - 2026-08-17

### Security

- Session IDs restricted to hex; path traversal via cookie blocked
- Query builder sanitizes OrderBy/GroupBy/Having/Where/Join identifiers and operators
- Mongo equality filters reject `$` operator injection
- Password-reset / email verification hashes use SHA-256
- Session cookie `Secure` when HTTPS or `SESSION_SECURE=true`
- HTTP server Read/Write/Idle timeouts
- `packages/safepath`: shared path containment; static `public/` and `LocalDisk` reject traversal
- Upload `StoreAs` basenames filenames; `X-Max-Upload` cannot raise server cap (`MAX_UPLOAD_BYTES`)
- WebSocket `Upgrade` defaults to same-origin Origin checks (`UpgradeWithCheckOrigin` / `AllowAnyOrigin`)
- Production ignores `TRUSTED_PROXIES=*` unless `TRUST_PROXIES_ALLOW_STAR=true`
- CSRF `Middleware` protects all paths by default (scaffold still uses `csrf.Except("/api")`)
- Cookie jar / `Make` honor `COOKIE_SECURE` / `SESSION_SECURE`; XSRF cookie sets `Secure` on HTTPS
- CI: `.github/workflows/security.yml` (vet, race subset, staticcheck, gosec, govulncheck, fuzz smoke)
- Docs: `SECURITY_AUDIT.md`, `SECURITY_REPORT.md`, `docs/security/testing.md`, security demo app

## 1.2.6 - 2026-08-16

### Fixed

- Social OAuth stub authorize URL uses app origin (`/oauth/{provider}/authorize`), not `oauth.zatrano.test` (ZATRANO-033)
- `SocialServiceProvider.Boot` registers same-origin stub authorize handlers when credentials are placeholders

## 1.2.5 - 2026-08-16

### Fixed

- `@lang` inside `@foreach` reads locale from Execute root (`$`), not the loop item (ZATRANO-032)

## 1.2.4 - 2026-08-16

### Fixed

- `make:auth` profile / password / 2FA / logout-other-devices routes and stubs under `/auth/*` (ZATRANO-030)
- Foundation view `dict` returns `map[string]any` so `<x-*>` `mergeDict` works (ZATRANO-031)

## 1.2.3 - 2026-08-16

### Fixed

- Default `packages/version` string was still `1.2.1` after the 1.2.2 cut
- `gofmt` on `make:model` `Name()` alignment (`packages/console/database.go`)

## 1.2.2 - 2026-08-16

### Added

- ORM model `Connection() string` routes queries to a named SQL connection (`DB_CONNECTIONS`)
- `make:model --connection=pgsql` stubs `Connection()` on the model
- MongoDB as a first-class `db:setup` driver (`packages/database/driver/mongo`) alongside SQL engines
- Multi-DB env hints from `db:setup` (`DB_MONGO_URI`, `DB_MYSQL_*`, `DB_PGSQL_*`, …)
- README section: single/multi database + model connection selection

### Changed

- Mongo boots from `DB_CONNECTIONS` / `DB_MONGO_URI` (legacy `package:enable mongo` still works if not already bound)

## 1.2.1 - 2026-08-16

### Fixed

- Default SQLite driver module require uses `v1.0.0` (proxy-resolvable) instead of `v0.0.0`

## 1.2.0 - 2026-08-16

### Added

- Optional SQL driver modules: `sqlite`, `mysql`, `pgsql`, `mssql`, `oracle` (lean default: SQLite only)
- `zatrano db:setup` — interactive / `--drivers=` multi-select, writes `bootstrap/database_drivers.go`, `go get`s only selected drivers
- Multi-database config: `DB_CONNECTIONS` + per-connection `DB_<NAME>_HOST` overrides; `app.DB().Connection("pgsql")`
- Oracle dialector, schema types, query placeholders (`:n`), migrations table
- Docker Compose `oracle` profile

### Changed

- Root `go.mod` no longer pulls MySQL/PostgreSQL/MSSQL drivers unless installed via `db:setup`

## 1.1.2 - 2026-08-16

### Added

- SQL Server (`mssql` / `sqlserver`) connection, schema, migrations, query placeholders (`@pN`), `db:create`
- Docker Compose profiles: `postgres`, `mysql`, `mssql`, `mongo`
- Driver dialect tests + SQLite smoke; optional live smoke via `ZATRANO_LIVE_DB=1`
- `env.GetNonEmpty` so blank `.env` credentials fall back to driver defaults
- `DB_SSLMODE` for PostgreSQL

### Fixed

- Runtime database config now uses `config.Database()` (pgsql default user was wrongly `root`)
- MySQL `ForeignID` is `BIGINT UNSIGNED` (matches `ID()`)
- MySQL JSON columns use native `JSON` type
- SQLite alter uses `ADD COLUMN`
- Insert returns `LastInsertId` errors instead of silent `0`
- Mongo addon fails boot when a real URI cannot ping
- Removed `go.work` / `go.work.sum` (use `go.mod` `replace` only)

## 1.1.1 - 2026-08-16

### Added

- `make:model --translation` / `-t` → `name_tr` / `name_en` fields; `--translation=json` → JSON `translations` cast (ZATRANO-015)
- `make:model -m` writes a table-specific migration (correct table name + translation columns when requested)

### Fixed

- `make:*` CLI commands boot with `CoreApp()` (no DB/session); other commands still use `FromEnv("app")` (ZATRANO-019)

## 1.1.0 - 2026-08-16

Ecommerce shop integration fixes (ZATRANO-001…029 subset).

### Added

- `WebApp` / `APIApp` merge `Preset* ∪ EnabledAddons`
- `db:create` CLI (mysql/pgsql)
- Billing `Checkout` uses Stripe `mode=payment`; `CheckoutPayment` with `price_data` line items
- `schema.ForeignID(...).Constrained(...).CascadeOnDelete()`
- `make:handler --api` / `--admin`; `make:view --layout=`; `make:lang --group=`; `make:auth --social=`
- Auth provider `WithHydrate` for ORM `*models.User` mapping
- Validation `SetDefaultPresenceChecker` (foundation wires DB unique/exists)
- `@lang('key', ['name' => …])` replacements; request-locale-aware `trans`
- Docker Compose `postgres` profile

### Fixed

- CLI default `FromEnv("app")` (was `demo`)
- `make:migration` param shadowing (`s *schema.Builder`)
- Auth routes/middleware/wellknown under `/auth/*` and `/api/auth/*`
- Authorization middleware HTML redirect/abort (not JSON-only)
- Form validation `RedirectBack` fallback = current path
- pgsql default username `postgres`; schema `Rename` for pgsql
- `package:enable` preserves `enabled.go` header comments
- Policy stub uses `authorization.Authenticatable`
- Resource stub comment → `packages/resources`
- `lang/` no longer gitignored by default

## 1.0.8 - 2026-08-15

### Fixed

- Foreach root dotted collection vs range-alias head: `@foreach($pagination.Links as $link)` → `dataGet $ "pagination.Links"`; nested `@foreach($section.pages …)` still uses `$section` when the head is an in-scope alias

## 1.0.7 - 2026-08-15

### Fixed

- Foreach alias rewrite: `@unless` / `@isset` / `@empty` and form attrs (`@selected` / `@checked` / …) compile `__ZRV_*` and `__ZPARENT__.*` (same precedence as `@if`) so loops no longer leave raw `@unless` + early `{{ end }}` → `undefined variable "$inv"`
- `@elseif(__ZRV_alias__)` replacement uses `$$$1` (literal `$alias`), not `$$1`

## 1.0.6 - 2026-08-15

### Fixed

- Nested `@foreach($section.pages as $link)` (docs sidebar): compile inner loops before parent rewrite; dotted collections resolve via `$section`; `@if($link.active)` uses the range var correctly
- Parent lookups in loops use Go’s stable Execute root `$` (avoids nested `$__zparent` overwrite)

## 1.0.5 - 2026-08-15

### Fixed

- Remove unused `rewriteAlias` (staticcheck U1000) left after foreach parent-scope rewrite

## 1.0.4 - 2026-08-15

### Fixed

- Routing: trailing slash normalization on dispatch (`/dashboard/` → `/dashboard`; `/` unchanged; query string kept) — seen integrating davet.link
- Views: `@foreach` / `@forelse` / `@each` keep parent scope (`$var`, `$category.ID`, `@csrf` / `_token`) while row fields use `$item.*`
- Views: `@if` / `@elseif` compile `>`, `>=`, `<`, `<=` (numeric); nested `@if`/`@else` inside `@include` partials stay matched
- Views: unsupported `@if` operators fail fast with a clear compile error (no silent raw leftover / `{{else}}` drift)

### Migration

- Inside `@foreach($items as $item)`, prefer `$item.field` for row data. Bare `$field` now resolves against the **parent** view data (not the loop element). Templates that relied on the old “`.` = item” short names should switch to the alias prefix.

## 1.0.3 - 2026-08-14

### Added

- Auth user-facing errors as localization keys (`auth.email_taken`, `auth.lockout`, …)
- Built-in `localization/defaults/{en,tr}/auth.json`
- `make:auth` HTML stubs fully use `@lang('auth.*')`; controller stub uses `lang()` / `authMsg()`

### Changed

- Locale middleware priority: session → **APP_LOCALE** → Accept-Language (browser no longer overrides configured locale by default)

## 1.0.2 - 2026-08-13

### Fixed

- `gofmt` clean tree for CI coding-style checks
- Authorization middleware test uses per-guard request key (`auth.user.{guard}`)
- GitHub Actions: `actions/checkout@v5`, `actions/setup-go@v6` (Node 24)

## 1.0.1 - 2026-08-12

### Fixed

- Publish nested heavy-package modules (`packages/mongo`, `packages/webauthn`, `packages/qr`) as `v1.0.0` so consumers can resolve them without local `replace` directives

## 1.0.0 - 2026-08-12

### Added

- Thin-kernel architecture: first-party code under `packages/`, lean `core/`
- Boot profiles: `CoreApp`, `MinimalApp`, `App`, `APIApp`, `WebApp`, `DemoApp` + `APP_BOOT` / `bootstrap.FromEnv`
- Package ecosystem CLI: `package:list|enable|disable|preset|init|install|publish|status|doctor`
- Addon registry, presets (`api` / `web` / `demo`), config stubs, catalog (`KindService` / `KindLibrary`)
- Auth parity: per-guard session keys, guard-aware middleware, cache-backed lockout, password broker throttle + silent unknown emails, `MarkEmailAsVerified` + `auth.verified`, 2FA remember-device, challenge lockout, configurable issuer
- Resolve helpers (`auth.From(app)`, `mail.From(app)`, …) across packages
- Website is the sole documentation source (repo `docs/` removed)

### Changed

- Import paths `core/X` → `packages/X` for first-party packages
- Foundation accessors removed from `Application` in favor of package `From` helpers
- Starter notification demo routes/handlers/views removed
- README rewritten for the v1 architecture

### Migration

1. `go get github.com/zatrano/framework@v1.0.0`
2. Rewrite imports to `packages/...`
3. Replace `app.Auth()` / `app.Mail()` / … with `auth.From(app)` / `mail.From(app)` / …
4. Set `APP_BOOT` for production (`app|api|web|minimal`)
5. Enable optional services with `package:enable` / presets

## 0.2.5 - 2026-08-12

### Added

- Localization: `make:lang`, `@choice` view helper, `Accept-Language` negotiation
- Real drivers: scannable QR, ip-api geo lookup, OpenAI-compatible AI, report webhooks, Twilio/HTTP SMS, HTTP push, MongoDB URI mode, Stripe billing mode
- OAuth PKCE + refresh tokens + optional JSON store; WebAuthn via go-webauthn
- WebSocket binary / ping / pong / close frames

### Changed

- Expanded localization documentation and removed stub banners from packages that now run for real when configured

## 0.2.4 - 2026-08-12

### Added

- Real GitHub OAuth provider (token exchange, user + primary email); stub only for placeholder credentials

### Changed

- Auth scaffold views use ZATRANO brand layout (white / red) instead of unfinished teal demo styles
- Notification demo pages rebuilt as complete inbox / send / bulk forms

## 0.2.3 - 2026-08-12

### Added

- Social login persistence helpers (`social.Persist`) and `make:auth` stubs for Google/GitHub accounts
- Env keys for `GOOGLE_*` / `GITHUB_*` OAuth credentials in `.env.example`

## 0.2.2 - 2026-08-12

### Added

- Template `@if` / `@elseif` numeric comparisons (`$limit == 50`, `$n != 0`)

## 0.2.1 - 2026-08-12

### Fixed

- PostgreSQL inserts use `RETURNING id` (lib/pq does not support `LastInsertId`)

## 0.2.0 - 2026-08-12

### Added

- Central notifications: in-app (database), mail, and SMS channels for web + API
- Single and bulk send (`Send` / `SendMany`) with CSV and XLSX recipient import
- Notification inbox store (list, unread, mark read / mark all read)
- Demo web UI (`/notifications`) and REST endpoints under `/api/notifications`
- Lightweight `core/export/xlsx` reader for bulk imports

## 0.1.10 - 2026-08-12

### Fixed

- PostgreSQL migration repository uses `$1, $2, …` placeholders instead of `?`
- Boolean column defaults emit `TRUE`/`FALSE` (Postgres-compatible)
- Database queue `EnsureTable` / queries are dialect-aware (SQLite, MySQL, PostgreSQL)

### Added

- SMTP mailer implicit TLS (`MAIL_ENCRYPTION` / port 465)
- Real Google OAuth provider (stub when credentials are placeholders)

## 0.1.9 - 2026-08-12

### Added

- Short `@section('name', $var)` form for layout sections with view data variables

## 0.1.8 - 2026-08-11

### Fixed

- `Request.Input` / `All` now read multipart form field values

## 0.1.7 - 2026-08-11

### Changed

- View `dataGet` resolves dotted paths on structs (including embedded fields), not only maps

## 0.1.6 - 2026-08-11

### Fixed

- Jobs table migration skips create when the table already exists (queue bootstrap race)

## 0.1.5 - 2026-08-06

### Fixed

- `serve` now loads `.env` before resolving `APP_PORT` (was always defaulting to `8080`)

## 0.1.4 - 2026-08-06

### Added

- Configurable HTTP listen port via `APP_PORT` (config, `serve`, and `Application.Run`)

## 0.1.3 - 2026-08-06

### Fixed

- Nested `@foreach` / `@endforeach` compilation in the view engine

## 0.1.2 - 2026-08-06

### Added

- Fenced code blocks (` ```lang `) in `core/markdown`
- GFM-style pipe tables in `core/markdown`

## 0.1.1 - 2026-08-06

### Added

- Nested markdown discovery for the documentation engine (`core/docs`)
- Sidebar navigation via optional `navigation.json`
- Previous/next page neighbors and custom `ViewRenderer` support for docs routes

## 0.1.0 - 2026-08-05

Initial release.

### Added

- Go module `github.com/zatrano/framework`
- Application skeleton: `app/`, `bootstrap/`, `config/`, `routes/`, `views/`, `database/`, `storage/`, `cmd/zatrano`
- HTTP request/response layer and router
- Controller registration helpers
- View engine (layouts, components, directives)
- Config loading via `.env` and `config/`
- Middleware pipeline (CSRF, CORS, security headers, throttle, and related helpers)
- Database migrations, schema builder, and ORM (`core/orm`)
- Logging, session, cache, queue, mail, validation, and auth packages in `core/`
- CLI entrypoint (`serve`, `migrate`, `make:*`, `about`, …)
- Welcome web route and home controller
- GitHub Actions CI (tests, static analysis, coding style)
- MIT license (Serhan KARAKOÇ)
