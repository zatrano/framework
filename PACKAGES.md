# ZATRANO packages guide

How to choose and use first-party packages.

This repository is the **framework**. `packages/` here is kernel, foundation, and intelligence code (`github.com/zatrano/framework/...`). It is not a copy of the addon repo.

- **Catalog source:** `kernel/catalog.go` (addon *names* are listed for discovery; addon *code* is not in this module)
- **Kernel / foundation / intelligence:** this repository, under `packages/`
- **Addon implementations:** [`github.com/zatrano/packages`](https://github.com/zatrano/packages) — `go get` + blank-import + `bootstrap.WithAddons` / `EnabledAddons`
- **Website docs:** [zatrano.com/docs](https://zatrano.com/docs)
- **CLI:** `go run ./cmd/zatrano package:list --all`

The two modules cannot be merged: `github.com/zatrano/packages` already requires this framework (Go import cycle).

This guide answers three questions per package: **what it is for**, **how to enable/resolve it**, and **how to use it** (minimal example). Deep API reference lives on the website.

---

## How packaging works

| Layer | Meaning | How to enable |
|-------|---------|----------------|
| **Kernel** | Secure HTTP boot surface | Always on with the app |
| **Foundation** | Typical web/API stack | `MinimalApp` / `App` / `APP_BOOT=app\|api\|web\|minimal` |
| **Addon (service)** | Optional container service | `package:enable NAME` → restart / same boot |
| **Addon (library)** | Import-only helper | `import` only — **never** put in `EnabledAddons` |

**Heavy** packages (`mongo`, `webauthn`, `qr`) use a separate Go module or heavy dependency — enable only when needed.

```bash
go run ./cmd/zatrano package:enable social billing
go run ./cmd/zatrano package:doctor
go run ./cmd/zatrano package:list --all
```

Resolve services with `From(app)` helpers — do not expect `app.Auth()` / `app.Mail()` on the kernel.

```go
auth.From(app)
session.From(app)
notification.From(app)
database.Migrator(app)
```

> **Mail:** There is no `packages/mail`. Send email through `notification` with `Channels: ["mail"]`.

---

## Quick map (need → package)

| Need | Package | Docs |
|------|---------|------|
| HTTP handlers / JSON | `http` + `routing` | [Requests](https://zatrano.com/docs/requests) · [Routing](https://zatrano.com/docs/routing) |
| Session login / MFA | `auth` | [Authentication](https://zatrano.com/docs/authentication) |
| Gates / policies | `authorization` | [Authorization](https://zatrano.com/docs/authorization) |
| Validate forms | `validation` | [Validation](https://zatrano.com/docs/validation) |
| SQL + models | `database` + `orm` | [Database](https://zatrano.com/docs/database) · [ORM](https://zatrano.com/docs/orm) |
| Send email / SMS | `notification` | [Notifications](https://zatrano.com/docs/notifications) · [Mail](https://zatrano.com/docs/mail) |
| Background jobs | `queue` | [Queues](https://zatrano.com/docs/queues) |
| Social login | `social` | [Socialite](https://zatrano.com/docs/socialite) |
| OAuth **server** | `oauth` | [OAuth](https://zatrano.com/docs/oauth) |
| API Bearer tokens | `apitoken` | [API Tokens](https://zatrano.com/docs/api-tokens) |
| Redis | `redisx` (+ cache/queue) | [Redis](https://zatrano.com/docs/redis) |
| CSRF | `middleware/csrf` | [CSRF](https://zatrano.com/docs/csrf) |

---

# Kernel

Always available. You rarely `package:enable` these.

### `container`

**For:** Dependency injection / service locator.  
**Use:** Prefer package `From(app)` helpers. Low-level:

```go
app.Container().Instance("my", svc)
v, err := app.Make("my")
```

### `config`

**For:** Typed config maps from `config/*.go` + `.env`.  
**Use:**

```go
app.Config().Get("app.name")
app.Config().Get("database.default")
```

Docs: [Configuration](https://zatrano.com/docs/configuration)

### `env`

**For:** Load `.env` into the process.  
**Use:** Boot calls this; raw access:

```go
import "github.com/zatrano/framework/env"
v := env.Get("APP_KEY", "")
```

### `context`

**For:** Request/app key-value context store used by the kernel.  
**Use:** Prefer `req.Set` / `req.Get` on `packages/http` for request-scoped data.

### `http`

**For:** Controllers receive `*http.Request` and return `*http.Response`.  
**Use:**

```go
import "github.com/zatrano/framework/http"

func (c *HomeController) Index(req *http.Request) *http.Response {
    email := req.Input("email")
    return http.JSON(map[string]any{"ok": true})
}
```

Docs: [Requests & Responses](https://zatrano.com/docs/requests)

### `routing`

**For:** Register routes, groups, named URLs, resources.  
**Use:**

```go
import "github.com/zatrano/framework/routing"

router.Get("/posts/{id}", c.Show).As("posts.show")
path, _ := router.URL("posts.show", map[string]string{"id": "1"})
```

Docs: [Routing](https://zatrano.com/docs/routing)

### `middleware`

**For:** Logger, Recover, CORS, SecurityHeaders, TrimStrings, Throttle, EncryptCookies, …  
**CSRF:** `packages/middleware/csrf`  
**Use:**

```go
import "github.com/zatrano/framework/middleware"
import "github.com/zatrano/framework/middleware/csrf"

router.Use(middleware.Logger, middleware.Recover, csrf.Except("/api"))
```

Docs: [Middleware](https://zatrano.com/docs/middleware) · [CSRF](https://zatrano.com/docs/csrf)

### `pipeline`

**For:** Compose middleware around a handler (used internally / via `middleware.Stack`).

### `exceptions` · `report`

**For:** Exception handling and reporting hooks.  
Docs: [Error Reporting](https://zatrano.com/docs/error-reporting)

### `log`

**For:** Application logger (boot-wired). Use the logger provided by foundation/boot in app code.

### `encryption`

**For:** AES-GCM encrypt/decrypt (`APP_KEY`). Used by cookies, auth 2FA secrets, etc.  
**Use:**

```go
enc := encryption.From(app) // or app.Encrypter()
s, err := enc.EncryptString("secret")
```

Docs: [Encryption](https://zatrano.com/docs/encryption)

### `trustedproxy`

**For:** Trust `X-Forwarded-*` only from known peers so `req.IP()` / `Secure()` are correct.  
**Use:**

```env
TRUSTED_PROXIES=10.0.0.0/8
```

Docs: [Trusted Proxies](https://zatrano.com/docs/trusted-proxies)

---

# Foundation

Present on `MinimalApp` and richer boot profiles. Resolve with `From(app)`.

### `session`

**For:** Per-visitor server-side session (default: file driver).  
**Use:**

```go
sess := req.Session()
sess.Put("locale", "en")
sess.Flash("status", "Saved")
_ = sess.Regenerate() // after login
```

Docs: [Session](https://zatrano.com/docs/session)

### `cookie`

**For:** Queue cookies onto responses (jar). Prefer `http.Response` cookie helpers for simple cases.  
Docs: [Cookies](https://zatrano.com/docs/cookies)

### `flash`

**For:** One-request success/error messages and old input.  
**Use:**

```go
return flash.WithSuccess(req, "Saved", "/posts")
```

Docs: [Flash](https://zatrano.com/docs/flash)

### `validation`

**For:** Validate `map[string]string` (usually `req.All()`) with pipe rules / FormRequest.  
**Use:**

```go
data, err := validation.ValidateRequest(req, map[string]string{
    "email": "required|email",
})
if err != nil {
    return validation.ResponseFor(req, err)
}
```

Docs: [Validation](https://zatrano.com/docs/validation) · [Rules](https://zatrano.com/docs/validation-rules)

### `auth`

**For:** Session authentication — login, register, password reset, email verify, lockout, MFA, remember-me.  
**Use:**

```go
mgr := auth.From(app)
ok, err := mgr.Attempt(req, map[string]string{
    "email": "ada@example.com", "password": "secret",
}, true)
router.Group("/account", routes, auth.Middleware(mgr))
```

Scaffold: `go run ./cmd/app make:auth` · `go run ./cmd/app make:dashboard` (auth paketi blank-import edildikten sonra)  
Docs: [Authentication](https://zatrano.com/docs/authentication) · [Dashboard Scaffold](https://zatrano.com/docs/dashboard-scaffold)

### `authorization`

**For:** Gates and policies after authentication.  
**Use:**

```go
gate := authorization.From(app)
gate.Define("edit-post", func(user authorization.Authenticatable, args ...any) bool {
    return true
})
if err := gate.Authorize(user, "edit-post", post); err != nil {
    return authorization.ResponseFor(err)
}
```

Docs: [Authorization](https://zatrano.com/docs/authorization)

### `hashing`

**For:** bcrypt password hashes.  
**Use:**

```go
h := hashing.From(app)
hash, _ := h.Make("secret")
ok := h.Check("secret", hash)
```

Docs: [Hashing](https://zatrano.com/docs/hashing)

### `cache`

**For:** Temporary key/value store (file / memory / redis).  
**Use:**

```go
c := cache.From(app)
_ = c.Put("k", 42, time.Hour)
v, ok := c.Get("k")
v, err := c.Remember("posts", time.Minute, loadPosts)
```

Docs: [Cache](https://zatrano.com/docs/cache)

### `redisx`

**For:** Shared Redis client used by cache/queue when configured.  
**Use:**

```env
REDIS_HOST=127.0.0.1
CACHE_STORE=redis
QUEUE_CONNECTION=redis
```

Docs: [Redis](https://zatrano.com/docs/redis)

### `database`

**For:** SQL connections, transactions, query builder, schema, migrations, seeders.  
**Use:**

```go
mgr := database.From(app)
users, err := mgr.Table("users")
err = mgr.Transaction(func(tx *sql.Tx) error { return nil })
mig := database.Migrator(app)
_ = mig.Migrate()
```

```bash
go run ./cmd/zatrano db:setup --drivers=sqlite,pgsql
go run ./cmd/zatrano migrate
```

Docs: [Database](https://zatrano.com/docs/database) · [Query Builder](https://zatrano.com/docs/queries) · [Schema](https://zatrano.com/docs/schema) · [Migrations](https://zatrano.com/docs/migrations)

### `orm`

**For:** Type-safe models, relations, eager loading, soft deletes.  
**Use:**

```go
user, err := orm.Query[User]().Where("email", email).First()
post, err := orm.Create[Post](map[string]any{"title": "Hi"})
```

Docs: [ORM](https://zatrano.com/docs/orm) (+ models / querying / relationships / eager / advanced)

### `view`

**For:** HTML templates (`views/`).  
**Use:**

```go
return http.View("dashboard", map[string]any{"name": "Ada"})
```

Docs: [Views](https://zatrano.com/docs/views)

### `queue`

**For:** Named background jobs (sync / database / redis).  
**Use:**

```go
q := queue.From(app)
q.Register("send-invoice", func(payload map[string]any) error { return nil })
_ = q.Push("send-invoice", map[string]any{"id": 1})
```

```bash
go run ./cmd/zatrano queue:work
```

Docs: [Queues](https://zatrano.com/docs/queues)

### `events`

**For:** Sync event dispatch and model observers.  
**Use:**

```go
d := events.From(app)
d.Listen("order.placed", func(e any) error { return nil })
_ = d.Dispatch("order.placed", payload)
```

Docs: [Events](https://zatrano.com/docs/events)

### `localization`

**For:** JSON translations under `lang/`.  
**Use:**

```go
t := localization.From(app)
msg := t.Get("auth.failed")
```

Docs: [Localization](https://zatrano.com/docs/localization)

### `schedule`

**For:** Cron-like tasks run via `schedule:run` (no long-running daemon).  
**Use:**

```go
s := schedule.From(app)
s.Call(job).Name("digest").DailyAt("08:00")
```

Docs: [Scheduling](https://zatrano.com/docs/scheduling)

### `filesystem`

**For:** Named disks (`local`, `public`, …).  
**Use:**

```go
files := filesystem.From(app)
_ = files.PutString("notes.txt", "hi")
```

Docs: [Filesystem](https://zatrano.com/docs/filesystem)

### `notification`

**For:** Async mail / SMS / push / database inbox / broadcast. **This is how you send email.**  
**Use:**

```go
n := notification.From(app)
_ = n.Send(
    notification.Recipient{Email: "ada@example.com"},
    notification.Message{
        Channels: []string{"mail"},
        Subject:  "Hello",
        Body:     "<p>Hi</p>",
    },
)
```

Docs: [Notifications](https://zatrano.com/docs/notifications) · [Mail](https://zatrano.com/docs/mail)

### `broadcasting`

**For:** Emit channel events to log/file/null drivers (not a WebSocket server).  
Docs: [Broadcasting](https://zatrano.com/docs/broadcasting) · [WebSockets](https://zatrano.com/docs/websockets)

### `httpclient`

**For:** Outbound HTTP with JSON, retries, fakes.  
**Use:**

```go
c := httpclient.From(app)
resp, err := c.BaseURL("https://api.example").Get("/ping")
```

Docs: [HTTP Client](https://zatrano.com/docs/http-client)

### `ratelimit`

**For:** In-memory named rate limiters (per process).  
**Use:**

```go
rl := ratelimit.From(app)
router.Use(ratelimit.Middleware(rl, "api"))
```

Docs: [Rate Limiting](https://zatrano.com/docs/rate-limiting)

### `url`

**For:** Absolute URLs, named routes, signed links.  
**Use:**

```go
u := url.From(app)
link, _ := u.Route("posts.show", map[string]string{"id": "1"})
signed, _ := u.Signed("/private")
```

Docs: [URL Generation](https://zatrano.com/docs/urls)

### `health`

**For:** `/health` style checks.  
Docs: [Health](https://zatrano.com/docs/health)

### `observability` · `maintenance` · `assets` · `console` · `support` · `version`

| Package | For | Docs / CLI |
|---------|-----|------------|
| `observability` | Metrics collection | [Observability](https://zatrano.com/docs/observability) |
| `maintenance` | Downtime page (`down` / `up`) | [Maintenance](https://zatrano.com/docs/maintenance-mode) |
| `assets` | Vite/Mix manifest URLs in views | [Assets](https://zatrano.com/docs/assets) |
| `console` | `cmd/zatrano` CLI | [Artisan](https://zatrano.com/docs/artisan) |
| `support` | Strings, arrays, UUID helpers | [Helpers](https://zatrano.com/docs/helpers) |
| `version` | Framework version helper | — |
| `apitoken` | Personal access tokens | [API Tokens](https://zatrano.com/docs/api-tokens) |

**`apitoken` usage:**

```go
tokens := apitoken.From(app)
plain, _, err := tokens.Create(userID, "cli", []string{"*"}, 24*time.Hour)
router.Use(apitoken.Middleware())
```

---

# Addon services (`package:enable`)

Enable, then resolve with `From(app)` (nil if not enabled).

```bash
go run ./cmd/zatrano package:enable NAME
```

### `social` (docs: Socialite)

**For:** GitHub/Google OAuth **client** login.  
**Use:**

```go
mgr := social.From(app)
url, _, err := mgr.Redirect("github")
user, err := mgr.User("github", code)
res, err := social.Persist(store, user)
```

Docs: [Socialite](https://zatrano.com/docs/socialite)

### `oauth`

**For:** Run an OAuth2 **authorization server** (not social login).  
Docs: [OAuth](https://zatrano.com/docs/oauth)

### `mongo` (heavy)

**For:** Document store client (not SQL ORM).  
**Use:** `db:setup --drivers=mongo` / `MONGO_URI`  
Docs: [MongoDB](https://zatrano.com/docs/mongodb)

### `webauthn` (heavy)

**For:** Passkey registration/login. Wire routes yourself.  
Docs: [WebAuthn](https://zatrano.com/docs/webauthn)

### `billing`

**For:** Subscriptions / Stripe-style billing manager + webhooks.  
Docs: [Billing](https://zatrano.com/docs/billing)

### `ai`

**For:** Chat / completion providers.  
Docs: [AI](https://zatrano.com/docs/ai)

### `backup`

**For:** DB backup/restore via native CLIs.  
```bash
go run ./cmd/zatrano db:backup
```
Docs: [Backup](https://zatrano.com/docs/backup)

### `bus`

**For:** Sync command bus (`Dispatch` → handler). Not a queue.  
Docs: [Command Bus](https://zatrano.com/docs/buses)

### `circuit`

**For:** Circuit breaker around flaky dependencies.  
Docs: [Circuit Breaker](https://zatrano.com/docs/circuit-breaker)

### `docs`

**For:** Markdown documentation repository (powers docs sites).  
Docs: [Documentation Engine](https://zatrano.com/docs/documentation-engine)

### `enums`

**For:** Register string-backed enums with labels.  
Docs: [Enums](https://zatrano.com/docs/enums)

### `features`

**For:** In-memory feature flags and % rollouts.  
Docs: [Feature Flags](https://zatrano.com/docs/feature-flags)

### `geo`

**For:** Resolve client geolocation.  
Docs: [Geolocation](https://zatrano.com/docs/geolocation)

### `graphql`

**For:** GraphQL schema/queries addon.  
Docs: [GraphQL](https://zatrano.com/docs/graphql)

### `hashid`

**For:** Obfuscate numeric IDs for URLs.  
Docs: [Hashids](https://zatrano.com/docs/hashids)

### `inspector`

**For:** Request inspector toolbar data.  
Docs: [Inspector](https://zatrano.com/docs/inspector)

### `lock`

**For:** Process-local atomic locks (not distributed).  
Docs: [Locks](https://zatrano.com/docs/locks)

### `octane`

**For:** Concurrent request metrics + `GOMAXPROCS` hint via `octane:start`. **Not** a multi-process app server.  
Docs: [Octane](https://zatrano.com/docs/octane)

### `otp`

**For:** Short numeric OTPs (you deliver via SMS/mail).  
Docs: [OTP](https://zatrano.com/docs/otp)

### `pulse`

**For:** Metrics pulse dashboard.  
Docs: [Pulse](https://zatrano.com/docs/pulse)

### `search`

**For:** In-memory search index.  
Docs: [Search](https://zatrano.com/docs/search)

### `shorturl`

**For:** Create and resolve short URLs.  
Docs: [Short URLs](https://zatrano.com/docs/short-urls)

### `sitemap`

**For:** Build XML sitemaps.  
Docs: [Sitemaps](https://zatrano.com/docs/sitemaps)

### `tenancy`

**For:** Resolve current tenant from header/query/host (no auto DB isolation).  
Docs: [Tenancy](https://zatrano.com/docs/tenancy)

### `webhooks`

**For:** Signed outbound webhook delivery.  
Docs: [Webhooks](https://zatrano.com/docs/webhooks)

### `wellknown`

**For:** `/.well-known` / `security.txt` helpers.  
Docs: [Well-Known](https://zatrano.com/docs/well-known)

### `audit`

**For:** Request/audit event logging.  
Docs: [Audit](https://zatrano.com/docs/audit)

---

# Addon libraries (import only)

Do **not** add these to `EnabledAddons`. Import and call.

| Package | For | How to use (sketch) | Docs |
|---------|-----|---------------------|------|
| `api` | API versioning | Version middleware / helpers | [API Versioning](https://zatrano.com/docs/api-versioning) |
| `archive` | ZIP create/extract | `zipx.Create` / `Extract` | [Archives](https://zatrano.com/docs/archives) |
| `bloom` | Bloom filter | `bloom.New` then Add/Test | [Bloom Filters](https://zatrano.com/docs/bloom-filters) |
| `browser` | Headless browser tests | Browser test helpers | [Browser Tests](https://zatrano.com/docs/browser-tests) |
| `collection` | In-memory collections | `collection.Make(...).Filter(...)` | [Collections](https://zatrano.com/docs/collections) |
| `concurrency` | Parallel tasks | `concurrency.Run` / `Map` / `Pool` | [Concurrency](https://zatrano.com/docs/concurrency) |
| `consent` | Cookie consent | Consent helpers | [Cookie Consent](https://zatrano.com/docs/cookie-consent) |
| `cron` | Cron parse/match | `cron.Parse("@hourly")` | [Cron](https://zatrano.com/docs/cron) |
| `debug` | Dump helpers | Debug dumps | [Debugging](https://zatrano.com/docs/debugging) |
| `export` | CSV/XLSX | `export.ToMaps` / `csv.Response` | [Exports](https://zatrano.com/docs/exports) |
| `factory` | Model factories | `factory.Register` / `Create` | [Factories](https://zatrano.com/docs/factories) |
| `fingerprint` | Device fingerprint | Fingerprint helpers | [Fingerprinting](https://zatrano.com/docs/fingerprinting) |
| `honeypot` | Spam traps | `honeypot.Middleware()` + `Fields()` | [Honeypot](https://zatrano.com/docs/honeypot) |
| `idempotency` | Idempotent POSTs | `idempotency.Middleware(cache, ttl)` | [Idempotency](https://zatrano.com/docs/idempotency) |
| `image` | Image processing | Resize/encode helpers | [Images](https://zatrano.com/docs/images) |
| `jsonapi` | JSON:API docs | `jsonapi.Response(doc)` | [JSON:API](https://zatrano.com/docs/json-api) |
| `jsonschema` | JSON Schema | Validate payloads | [JSON Schema](https://zatrano.com/docs/json-schema) |
| `markdown` | MD → HTML | `markdown.ToHTML(src)` | [Markdown](https://zatrano.com/docs/markdown) |
| `negotiate` | Accept negotiation | `negotiate.Middleware(...)` | [Content Negotiation](https://zatrano.com/docs/content-negotiation) |
| `openapi` | OpenAPI helpers | Generate/serve specs | [OpenAPI](https://zatrano.com/docs/openapi) |
| `pages` | File-based pages | `pages.New(...).Register(router)` | [Pages](https://zatrano.com/docs/pages) |
| `pagination` | Page metadata | `pagination.New(items, total, …)` | [Pagination](https://zatrano.com/docs/pagination) |
| `pdf` | PDF generate/view | PDF helpers | [PDF](https://zatrano.com/docs/pdf) |
| `process` | OS commands | `process.Command("git","status").Run()` | [Processes](https://zatrano.com/docs/processes) |
| `qr` (heavy) | QR codes | Generate QR images | [QR Codes](https://zatrano.com/docs/qr-codes) |
| `resources` | API transformers | `resources.JSON(UserResource{}, user)` | [API Resources](https://zatrano.com/docs/api-resources) |
| `testing` | Feature tests | `testkit.New(app).Get("/").AssertOK()` | [Testing](https://zatrano.com/docs/testing) |
| `timing` | Server-Timing | `timing.Measure(req, "db", fn)` | [Timing](https://zatrano.com/docs/timing) |
| `totp` | Authenticator codes | `totp.GenerateSecret` / `Verify` | [TOTP](https://zatrano.com/docs/totp) |
| `useragent` | UA parse | Parse browser/OS | [User Agent](https://zatrano.com/docs/user-agent) |
| `websocket` | WS upgrade | `websocket.Upgrade(handler)` | [WebSockets](https://zatrano.com/docs/websockets) |

---

## Internal helper (not in catalog)

### `safepath`

**For:** Block path traversal outside a root (filesystem, zip extract, uploads).  
Used internally; call if you write custom file APIs:

```go
if !safepath.Under(root, candidate) { /* reject */ }
```

---

## Recommended learning path

1. [Installation](https://zatrano.com/docs/installation) · [Boot Profiles](https://zatrano.com/docs/boot-profiles)  
2. `routing` · `http` · `validation` · `view`  
3. `database` · `orm` · `auth` · `authorization`  
4. `notification` · `queue` · `cache`  
5. Enable only the addons you need (`social`, `billing`, …)

---

## Related

- [README.md](README.md)  
- [Package Ecosystem](https://zatrano.com/docs/package-ecosystem)  
- [Resolving Services](https://zatrano.com/docs/accessors)  
- `core/catalog.go`  
