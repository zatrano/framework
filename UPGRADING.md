# Upgrading

## Upgrading from v3.0.1

v3.1.0 is a minor release. `go get -u=patch` does not cross it. The last published tag before this release is v3.0.1. v3.0.0 is retracted and cannot be installed.

| Behavior | v3.0.1 | v3.1.0 | Restore the old behavior |
| --- | --- | --- | --- |
| Header ceiling | `MaxHeaderBytes` 1 MiB. A larger block is 431. | 16 KiB (`HTTP_MAX_HEADER_BYTES`, a unitless byte count). Invalid, zero, and negative values abort boot. Above 64 KiB logs a warning. Over the limit is still 431. rawhttp sizes the per-connection read buffer from this value. | `HTTP_MAX_HEADER_BYTES=1048576`. The warning above 64 KiB still logs until rawhttp grows the buffer. |
| Body limits | The server left `MaxRequestBodySize` unset, so the rawhttp engine ceiling was 4 MiB. | JSON (`application/json` and `+json`), `application/x-www-form-urlencoded`, and `text/*` are rejected when the headers arrive if `Content-Length` exceeds `MAX_BODY_BYTES` (2 MiB). Multipart, octet-stream, an unknown type, and a missing `Content-Type` use the server ceiling, the larger of `MAX_BODY_BYTES` and `MAX_UPLOAD_BYTES` (32 MiB). `Route.BodyLimit(n)` overrides one route (`n` bytes; negative uses the server ceiling). `Request.Body` and `Request.JSON` still stop at 2 MiB. | `MAX_UPLOAD_BYTES=4194304` brings the non-text ceiling back to 4 MiB. JSON, form, and text stay at `MAX_BODY_BYTES` unless that is raised too. |
| In-flight body budget and per-client share | No in-flight budget. | `HTTP_MAX_INFLIGHT_BODY_BYTES` defaults to 256 MiB. A known `Content-Length` reserves that length. A chunked body reserves the worst case (JSON, urlencoded, and `text/*` reserve `MAX_BODY_BYTES`; everything else reserves its cap). Not enough room is 503 with `Retry-After: 1` and the body is not read. One request whose cap exceeds the whole budget is 413. `HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT` defaults to 25% of the budget when the client can be told apart (trusted proxy plus a resolved address, or a global-unicast peer). Over the share is the same 503. | A negative `HTTP_MAX_INFLIGHT_BODY_BYTES` turns the budget off. A negative `HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT` turns the share off. |
| Chunked body over the cap | rawhttp v0.2.2 could answer 431 when the body exceeded the read buffer. | rawhttp v0.2.4 applies the body cap. Chunked JSON over 2 MiB and chunked multipart over 32 MiB are 413. | No setting. The status follows rawhttp v0.2.4. |
| HTTP timeouts | 60s read, 60s write, 120s idle, 10s header, compiled in. | The same defaults, from `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_IDLE_TIMEOUT`, and `HTTP_READ_HEADER_TIMEOUT`. `0`, `0s`, and any negative value disable that deadline. A unitless integer is seconds. `30s` and `1m` are accepted. An invalid value aborts boot. `Stream` re-arms the write deadline before each chunk. | Leave the four variables unset. The numbers match v3.0.1. Set `0` to disable one deadline. |
| WebSocket upgrade | No admission switch on the server. | Default off. First match wins: `HTTP_ALLOW_UPGRADE=false`; then `ListenOptions.AllowUpgrade` and `serve --allow-upgrade`; then `HTTP_ALLOW_UPGRADE=true`; then `RegisterUpgradeProtocol`; then a linked `websocket` addon; then `EnabledAddons` contains `websocket`. `Upgrade: h2c` stays rejected. | Leave admission unset. Nothing turns it on unless one of the sources above is set. |
| Shutdown | `Shutdown`, then `Stop`. | Hooks registered with `http.RegisterShutdownHook`, then rawhttp `Shutdown`, then `Stop`. Hooks are concurrent, bounded by `min(5s, remaining/3)`. A panic or error is logged and does not stop shutdown. A second call is a no-op. A hook registered after shutdown has started is ignored. `context.DeadlineExceeded` from `Shutdown` is followed by `Close` (this drops a handler still inside `Hijack`), then `Stop`. | Do not register a hook. An application that never calls `RegisterShutdownHook` keeps the previous order. |
| Unmatched path and CORS preflight | A miss could skip global middleware. | An unmatched path runs global middleware, then returns 404. There is no `Allow` header and no 405. An `OPTIONS` request with `Origin` and `Access-Control-Request-Method` is answered before later middleware: an allowed origin is 204 with the allow headers; any other origin is 404 with no CORS headers. | No setting. |
| CORS headers without `Origin` | A response could still carry CORS headers. | No configured origin, and a request with no `Origin`, write no CORS headers and no `Vary`. `Vary: Origin` is set when the allow list is not a wildcard and the request has `Origin`, including 404. | No setting. Credentials plus a wildcard origin fails boot. |
| `Expect: 100-continue` | Not covered as a framework guarantee. | A `Content-Length` over the effective cap, or a reservation that does not fit, is 413 or 503 with no `100 Continue`. The connection lingers so the client can read that response (`MaxLingering` 1024; negative is unlimited). | No setting. |
| Minimum Go | `go 1.25.0` | `go 1.25.0` | Unchanged. |
| rawhttp | v0.2.2 | v0.2.4. The module has no `replace` directive. | No supported downgrade. v0.2.2 is the release that answered an oversize chunked body with 431. |
| Retract | v3.0.0 was installable and contained local `replace` directives. | `retract v3.0.0`. `go install github.com/zatrano/framework/v3@v3.0.0` is rejected. v3.0.1 remains installable. | Install v3.0.1, not v3.0.0. |
| `zatrano doctor` | No in-flight body rules. | APP-HTTP-006: a route `BodyLimit` above `HTTP_MAX_INFLIGHT_BODY_BYTES`. APP-HTTP-007: `TRUSTED_PROXIES` is empty, so clients behind one address share one key. Behind a CDN that key is the edge address (64 MiB at the default 25% share). | APP-HTTP-006: lower `BodyLimit`, or raise the budget. Production already refuses to boot; `HTTP_STRICT_LIMITS` does that in every environment. APP-HTTP-007: set `TRUSTED_PROXIES`, or set `HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT` negative. |
| `VERSION` and `zatrano version` | The v3.0.1 tag left the `VERSION` file at `3.0.0`. | `VERSION` is `3.1.0`. `zatrano version` reads that file and falls back to the same number. Release CI rejects a tag whose name, without the leading `v`, differs from `VERSION`. | None. The product version and the module tag are the same string. |

## Environment

Names the kernel reads with `env.Get`, `env.Lookup`, `env.IntOr`, `env.GetBool`, or `os.Getenv`. `TestEnvironmentVariablesAreDocumented` fails when code reads an `HTTP_*`, `MAX_*`, `CORS_*`, or `TRUSTED_PROXIES` variable that is missing from this table.

| Variable | Default | Meaning |
| --- | --- | --- |
| `HTTP_READ_TIMEOUT` | 60s | Request read deadline. `0` or negative disables it. A unitless integer is seconds. |
| `HTTP_WRITE_TIMEOUT` | 60s | Response write deadline, re-armed per stream chunk. `0` or negative disables it. |
| `HTTP_IDLE_TIMEOUT` | 120s | Keep-alive idle deadline. `0` or negative disables it. |
| `HTTP_READ_HEADER_TIMEOUT` | 10s | Header read deadline. `0` or negative disables it. |
| `HTTP_MAX_HEADER_BYTES` | 16 KiB | Header-block ceiling in bytes. Over the limit the engine returns 431. |
| `HTTP_ALLOW_UPGRADE` | unset (off) | `false` forces WebSocket admission off. `true` turns it on unless `AllowUpgrade` is explicitly false. |
| `HTTP_MAX_INFLIGHT_BODY_BYTES` | 256 MiB | In-flight body budget. Negative disables it. Not enough room is 503 with `Retry-After: 1`. |
| `HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT` | 25% of the budget | Per-client share when the client can be told apart. Negative disables it. |
| `HTTP_STRICT_LIMITS` | off | A route `BodyLimit` above the in-flight budget fails boot outside production as well. |
| `MAX_BODY_BYTES` | 2 MiB | JSON, urlencoded, and `text/*` header cap. Also `Request.Body` and `Request.JSON`. Ignored unless positive. |
| `MAX_UPLOAD_BYTES` | 32 MiB | Multipart ceiling. The server ceiling is the larger of this and `MAX_BODY_BYTES`. Ignored unless positive. |
| `TRUSTED_PROXIES` | empty | Proxy or CDN addresses. Empty is APP-HTTP-007. |
| `CORS_ENABLED` | true | `false` does not install CORS middleware. |
| `CORS_ALLOWED_ORIGINS` | empty | Comma-separated exact origins. Empty writes no CORS headers. Development may still default to `*` until this is set. |
| `CORS_ALLOWED_METHODS` | `GET, POST, PUT, PATCH, DELETE, OPTIONS` | Preflight methods. |
| `CORS_ALLOWED_HEADERS` | `Content-Type, Authorization, X-Requested-With, X-CSRF-TOKEN, X-Idempotency-Key` | Preflight headers. The response is the intersection with the request. |
| `CORS_EXPOSE_HEADERS` | empty | Sent as `Access-Control-Expose-Headers` when an origin matches. |
| `CORS_ALLOW_CREDENTIALS` | false | `true` together with a wildcard origin fails boot. |
| `CORS_MAX_AGE` | 600 | Preflight `Access-Control-Max-Age`. A non-integer is ignored. |
