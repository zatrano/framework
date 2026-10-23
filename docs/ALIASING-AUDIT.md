# Aliasing audit

This file records where every `string`, `[]byte`, and pointer returned by the public request and response API came from in v3.2.4, how long it stayed valid, and what the `safe-by-construction` branch changed. Sources:

- **copy**: freshly allocated for the caller; never reused.
- **carrier**: points into the rawhttp connection read buffer (`*rawhttp.Ctx`). rawhttp reuses that buffer for the next request on the same keep-alive connection.
- **pool**: points into an object taken from a `sync.Pool`.
- **live ctx read**: the value was copied, but it was read through `r.ctx` at call time. A call made after the handler returned read whatever request the connection was serving by then.
- **arena**: owned by the request. Written once while the request is built, never written again, never reused, reclaimed by the GC.

## v3.2.4 (before)

### `core/kernel/http` `Request`

| API | Returns | Source in v3.2.4 | Lifetime in v3.2.4 | Finding |
| --- | --- | --- | --- | --- |
| `Method()` | `string` | constant for the seven common methods; otherwise `string(ctx.Method)` copy, cached in `methodView` | cached value safe; first call after the handler is a live ctx read | stale call returned the next request's method |
| `Path()` | `string` | `string(ctx.Path)` copy cached in `pathView` | first call after the handler is a live ctx read | stale call returned the next request's path |
| `PathBytes()` | `[]byte` | carrier (`ctx.Path`) | until the handler returns | aliased the connection buffer |
| `QueryString()`, `Query()`, `QueryAll()`, `HasQuery()`, `QueryInt()` … | `string` | `string(ctx.Query)` copy on every call | every call is a live ctx read | stale call returned the next request's query |
| `Header()`, `HeaderValues()`, `HasHeader()`, `BearerToken()`, `ContentType()` … | `string` / `[]string` | `string(ctx.Header(k))` copy | every call is a live ctx read | stale call returned the next request's header |
| `HeadersMap()` | `map[string]string` | `ctx.VisitHeader` copy | live ctx read | same |
| `Cookie()`, `HasCookie()`, `CookieMap()` | `string` / `bool` / map | `ctx.Cookie` / `ctx.VisitCookie` copy | live ctx read | stale call returned the next request's cookie, including a session id |
| `Host()` | `string` | `string(ctx.Host())` copy | live ctx read | same |
| `Secure()`, `Scheme()` | `bool` / `string` | `ctx.IsTLS()` | live ctx read; the `Ctx` may serve another connection | could report another connection's TLS state |
| `RemoteIP()`, `IP()` | `string` | `ctx.RemoteIP()` / `ctx.ClientIP()` (Go string held by the `Ctx`) | live ctx read | could report another connection's peer |
| `IP()` with `TRUSTED_PROXIES` | `string` | `ctx.ClientIP()` first, which takes the **leftmost** `X-Forwarded-For` hop | — | a client could prepend a forged hop; the trustedproxy middleware's rightmost resolution was only a fallback |
| `Body()`, `JSON()`, form values | `[]byte` / decoded | `append([]byte(nil), ctx.Body()...)` copy, cached | cached value safe; first read after the handler is a live ctx read | stale first read copied the next request's body |
| `BindJSON()` | decoded into `dest` | decoded in place from `ctx.Body()` (carrier) | during the call | stale call decoded the next request's body |
| `Route()`, `RouteInt()`, `RouteParams()` | `string` / map | substrings of `aliasBuf(req.PathBytes())`, an `unsafe.String` over the carrier path. This did not depend on the pool flag | until the handler returns | a retained value changed when the next request was read into the buffer |
| `File()`, `Files()` (multipart) | `*multipart.FileHeader` | `ctx.MultipartForm` (mime/multipart copies values; files are temp files removed at `Ctx.reset`) | live ctx read when first parsed after the handler | stale parse read the next request's body |
| `Ctx()` | `*rawhttp.Ctx` | the carrier itself | until the handler returns | every field and method of the raw context aliases the connection |
| `Context()` | `context.Context` | `context.Background()` or a value the app set | owned | none |
| `Get()` / `Set()` attributes | `any` | request map | owned (non-pooled) / **pool** (pooled) | with `HTTP_POOL_REQUESTS=true`, the map was reset and reused |
| the `*Request` itself | pointer | `new` (default) / **pool** (`HTTP_POOL_REQUESTS=true`) | `ReleaseRequest` zeroed it with `*r = Request{…}` and put it back | with the pool on, a kept pointer served another request |

### `core/kernel/http` `Response`

| API | Source in v3.2.4 | Finding |
| --- | --- | --- |
| `Text()`, `JSON()` result | `new` (default) / **pool** (`HTTP_POOL_REQUESTS=true`) | with the pool on, `ReleaseResponse` zeroed and reused the object after `Commit` |
| response passed through middleware | the handler's pointer, mutated in place by `SecurityHeaders`, `CORS`, `CSRF` (`X-CSRF-TOKEN` header and token cookie), `Throttle`, `RequestID`, `httpcache`, version headers, the HTTP bridge `Finalize`, and `Application.Handle` (queued cookies, stream deadline) | a handler that returned one cached `*Response` for many requests had it mutated concurrently. Cookies were appended to the shared object, so one visitor's CSRF cookie could be sent to the next |
| `Commit()` | `Headers()` called `flushInlineHeaders`, which moved inline headers into the cold map, so the write path mutated the response | concurrent commits of one response raced |
| body bytes | `ctx.SetBody(resp.content)` keeps a reference until rawhttp writes | read-only; rawhttp never writes into it |

### `core/kernel/routing`

| Item | Source | Finding |
| --- | --- | --- |
| `aliasBuf(b)` | `unsafe.String(&b[0], len(b))` over `req.PathBytes()` (carrier) | used for the static lookup key **and** as the string the trie split into route parameters |
| `routeHit.k` | route definition strings | owned, immutable |
| `routeHit.v` | substrings of the aliased path | carrier (see `Route()` above) |
| `normalizeDispatchPath` | `string(req.PathBytes())` copy | owned |

### `Request.Ctx()`

Returns the raw `*rawhttp.Ctx`. Every `[]byte` it returns (`Method`, `Path`, `Query`, `Header`, `Cookie`, `Body`, `Host`, …) aliases the connection buffer and is valid only while the handler runs. rawhttp resets the same `Ctx` for the next request on the connection. A stored `*rawhttp.Ctx` or any slice taken from it reads another request after the handler returns.

## `safe-by-construction` (after)

| API | Source now | Lifetime now |
| --- | --- | --- |
| `Method()`, `Path()`, `QueryString()` and all query helpers | arena | the life of the `*Request` |
| `Header()`, `HeaderValues()`, `HeadersMap()` and helpers | arena | same |
| `Cookie()`, `HasCookie()`, `CookieMap()` | parsed from the arena copy of the `Cookie` header | same |
| `Host()` | arena | same |
| `RemoteIP()`, `Secure()` | copied from the carrier when the request is built | same |
| `IP()` | the trustedproxy middleware's rightmost resolution, else the copied peer address. `rawhttp.Ctx.ClientIP` is no longer consulted | same |
| `Route()`, `RouteInt()`, `RouteParams()` | substrings of the arena path | same |
| `PathBytes()` | deprecated; returns a fresh copy | owned by the caller |
| `Body()`, `JSON()`, form values, `File()` | the body is copied into the request when it is built, before any handler runs; every read uses that copy | the life of the `*Request` |
| `BindJSON()` | decodes from the request's body copy | the decode finishes before the call returns; the copy is never written |
| `Ctx()` | deprecated and unsafe. Returns the carrier while the handler runs and `nil` after `Detach` | until the handler returns |
| the `*Request` | `new`; never pooled. `AcquireRequest` is `NewRequest`; `ReleaseRequest` is a no-op | GC |
| the handler's `*Response` | never modified by the engine. Each middleware boundary and `Application.Handle` take a new request-owned copy (`Request.Own`) before anything adds headers or cookies, also when the response already belongs to the request, because a middleware may keep the response its inner handler returned. `Commit` only reads | GC |

The arena is one `string` built with `strings.Builder` and sliced. No `unsafe` code remains in the framework module. See `docs/SECURITY-GUARANTEES.md` for the invariants and the tests that check them.
