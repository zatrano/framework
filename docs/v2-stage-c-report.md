# ZATRANO V2 Stage C — Final rapor

Tarih: 2026-09-04  
Dal: `v2-stage-c` (base: `v2-stage-b` @ `bbf14b3`)

---

## 1. Faz 1 — ön koşul

Stage A varsayılan haliyle **uyumlu; durulmadı**. Ayrıntı: `docs/v2-stage-c-audit.md`.

| Varsayım | Gerçek |
| --- | --- |
| `contracts/` + `Router` / `ConfigRepository` | Var (`contracts/router.go`, `contracts/config.go`) |
| `kernel/` (eski `core/`) + katman sabitleri | `kernel/catalog.go` satır 8–16: `LayerPrimitive`, `LayerFoundation`, `LayerIntelligence`, `LayerAddon` |
| Self-registration | `bootstrap/addons/registry.go` `Register(Meta)`; addon `init()`; registry addon import etmez |

**Sapmalar (engelleyici değil):**

- Prompt hâlâ `core/catalog.go` diyor; diskte `kernel/catalog.go`.
- `Router` imzaları somut `packages/routing` tiplerine bağlı (Stage A sızıntısı). `doctor` somut-tip listesini `contracts/assert.go` bağlarından türetir.
- V2 uygulama ağacı Stage B şablonu: `app/http/controllers`, `app/routes/web`, `app/routes/api`. Promptun `app/controllers` / `app/config` örnekleri kullanılmadı.
- `zatrano package:doctor` addon sağlığı için kaldı; mimari komut `zatrano doctor`.

---

## 2. Faz özeti

### Faz 1 — denetim (kod yok)

- Commit: `026efeb`
- Dosya: `docs/v2-stage-c-audit.md`

### Faz 2 — `zatrano describe`

- Commit: `2b5eb38`
- Komut: `zatrano describe` (varsayılan insan-okunur özet; `--format=json`)
- Kaynak: `contracts/*.go` (go/ast), `kernel.Catalog` + `catalog.go` Layer sabitleri/yorumları, `packages/routing/discovery.go`, `kernel.Provider`, `bootstrap/addons/registry.go`
- Dosyalar: `packages/console/describe.go`, `describe_parse.go`, `describe_test.go`; `kernel/catalog.go` (Layer godoc); `cmd/zatrano/main.go`, `boot_test.go`; `packages/console/console.go`
- Testler: geçerli JSON + `contracts.Router`; yeni interface AST’de görünür; fixture `app/routes/web` sample route

### Faz 3 — `zatrano doctor`

- Commit: `eb7e274`
- Komut: `zatrano doctor [path]`
- Kontroller (ayrı fonksiyon; `--fix` yok; hepsi **uyarı**, çıkış kodu 0): `routes`, `concrete`, `layout`, `providers`
- Dosyalar: `packages/console/doctor.go`, `doctor_checks.go`, `doctor_test.go`, `packages/console/testdata/doctor/*`
- Testler: yanlış yerde `Get`; `packages/config` import; `application/` ağacı; `Boot` eksik Provider; temiz `zatrano new` uygulamasında route/concrete/provider uyarısı yok
- Muhafazakârlık: `version.Get()` route sayılmaz; `packages/routing` import yalnızca `app/routes/web`, `app/routes/api` ve `route_service_provider.go` içinde serbest

### Faz 4 — `zatrano agents:generate`

- Commit: `c6aa4f3`
- Komut: `zatrano agents:generate [path]`; `zatrano new` iskeletten sonra `AGENTS.md` yazar
- Dosyalar: `packages/console/agents.go`, `agents_test.go`; `new.go` / `new_test.go`
- İçerik `BuildDescribeDocument` + `DoctorChecks` + starter `app/routes*` dizinlerinden render edilir
- Testler: yeni interface/primitive markdown’a yansır; aynı sürümde idempotent; sürüm değişince fark

### Faz 5 — bu rapor

- Dosya: `docs/v2-stage-c-report.md`
- Tam JSON anlığı: `docs/v2-stage-c-describe.json` (`go run ./cmd/zatrano describe --format=json`)

---

## 3. `describe --format=json` (gerçek çalıştırma)

Komut (framework kökü, 2026-09-04):

```
go run ./cmd/zatrano describe --format=json
```

Framework reposunda `app/` olmadığı için `sample_routes` boş dizidir. Tam çıktı `docs/v2-stage-c-describe.json`. Aşağıda aynı koşudan kesitler.

### `contracts.Router`

```json
{
  "name": "Router",
  "file": "contracts/router.go",
  "methods": [
    {"name": "Get", "signature": "Get(path string, handler routing.HandlerFunc) *routing.Route"},
    {"name": "Post", "signature": "Post(path string, handler routing.HandlerFunc) *routing.Route"},
    {"name": "Use", "signature": "Use(middleware ...routing.MiddlewareFunc)"},
    {"name": "Group", "signature": "Group(prefix string, fn func(router *routing.Router), middleware ...routing.MiddlewareFunc)"},
    {"name": "Name", "signature": "Name(prefix string, fn func(router *routing.Router))"},
    {"name": "Routes", "signature": "Routes() []*routing.Route"},
    {"name": "Snapshot", "signature": "Snapshot() []routing.RouteInfo"},
    {"name": "SaveCache", "signature": "SaveCache(path string) error"}
  ]
}
```

### `catalog` — `LayerIntelligence` (ai / rag / agent; içeriklerine dokunulmadı)

```json
{
  "constant": "LayerIntelligence",
  "name": "intelligence",
  "role": "LayerIntelligence is the first-party AI identity layer (same activation as addons).",
  "packages": [
    {"name": "ai", "kind": "service", "heavy": false, "description": "AI chat providers"},
    {"name": "rag", "kind": "library", "heavy": false, "description": "RAG chunking, embed pipeline, vector store helpers"},
    {"name": "agent", "kind": "library", "heavy": false, "description": "AI agent loop, tools, conversation memory"}
  ]
}
```

### `routing` ve `providers`

```json
{
  "routing": {
    "primitives": [
      {"name": "RegisterWeb", "signature": "RegisterWeb(fn func(*Router))"},
      {"name": "RegisterAPI", "signature": "RegisterAPI(fn func(*Router))"},
      {"name": "ApplyWeb", "signature": "ApplyWeb(r *Router)"},
      {"name": "ApplyAPI", "signature": "ApplyAPI(r *Router)"}
    ],
    "sample_routes": []
  },
  "providers": {
    "interface": {
      "name": "Provider",
      "file": "kernel/application.go",
      "methods": [
        {"name": "Register", "signature": "Register(app *Application) error"},
        {"name": "Boot", "signature": "Boot(app *Application) error"}
      ]
    },
    "self_registration": {
      "package": "github.com/zatrano/framework/bootstrap/addons",
      "register": "Register(m Meta)",
      "select": "Select(names ...string) ([]kernel.Provider, error)",
      "lookup": "Lookup(name string) (Meta, bool)",
      "available": "Available() []Meta",
      "meta_type": "Meta",
      "meta_fields": ["Name", "Key", "Description", "Heavy", "Factory", "CLI"],
      "factory_field": "Factory",
      "factory_returns": "func() kernel.Provider",
      "register_called_from": "init",
      "registry_imports_addons": false,
      "consumer_blank_import": true
    }
  }
}
```

---

## 4. `doctor` örnek ihlaller (gerçek çalıştırma)

Fixture’lar kasıtlı olarak eksik starter ağacı taşıyor; hedef ihlalin yanında `layout` uyarıları da çıkar. Çıktılar `go run ./cmd/zatrano doctor <fixture>`.

### 4.1 Yanlış yerde route — `packages/console/testdata/doctor/misplaced_route`

```
zatrano doctor
root: C:\Users\Pc\Desktop\ZATRANO\packages\console\testdata\doctor\misplaced_route
findings: 8 (warnings)

[layout] warning  app/console
  found: missing directory app/console
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/console (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/http/controllers/api
  found: missing directory app/http/controllers/api
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/http/controllers/api (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/http/controllers/web
  found: missing directory app/http/controllers/web
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/http/controllers/web (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/providers
  found: missing directory app/providers
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/providers (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes
  found: missing directory app/routes
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes/api
  found: missing directory app/routes/api
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes/api (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes/web
  found: missing directory app/routes/web
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes/web (or regenerate the app with zatrano new) and keep types in the starter locations.

[routes] warning  app/services/oops.go:4
  found: Get("/oops") outside app/routes/{web,api}
  why:   HTTP routes belong in self-registered web/api groups, not scattered through the app.
  how:   Move this call into app/routes/web or app/routes/api and register it with RegisterWeb/RegisterAPI.
```

### 4.2 Somut tip sızıntısı — `packages/console/testdata/doctor/concrete_leak`

```
zatrano doctor
root: C:\Users\Pc\Desktop\ZATRANO\packages\console\testdata\doctor\concrete_leak
findings: 7 (warnings)

[concrete] warning  app/http/controllers/web/home.go:3
  found: import github.com/zatrano/framework/packages/config (contracts.ConfigRepository concrete)
  why:   Depending on framework concrete types couples the app to implementation details and weakens contract stability.
  how:   Use contracts.ConfigRepository via kernel.Application accessors instead of importing github.com/zatrano/framework/packages/config.

[layout] warning  app/console
  found: missing directory app/console
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/console (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/http/controllers/api
  found: missing directory app/http/controllers/api
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/http/controllers/api (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/providers
  found: missing directory app/providers
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/providers (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes
  found: missing directory app/routes
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes/api
  found: missing directory app/routes/api
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes/api (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes/web
  found: missing directory app/routes/web
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes/web (or regenerate the app with zatrano new) and keep types in the starter locations.
```

### 4.3 Eski `application/` ağacı — `packages/console/testdata/doctor/legacy_layout`

```
zatrano doctor
root: C:\Users\Pc\Desktop\ZATRANO\packages\console\testdata\doctor\legacy_layout
findings: 8 (warnings)

[layout] warning  app/console
  found: missing directory app/console
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/console (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/http/controllers/api
  found: missing directory app/http/controllers/api
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/http/controllers/api (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/http/controllers/web
  found: missing directory app/http/controllers/web
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/http/controllers/web (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/providers
  found: missing directory app/providers
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/providers (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes
  found: missing directory app/routes
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes/api
  found: missing directory app/routes/api
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes/api (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes/web
  found: missing directory app/routes/web
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes/web (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  application
  found: unexpected path application
  why:   Legacy application/ trees are not the V2 consumer layout.
  how:   Move code into app/ (http/controllers, services, providers) and delete application/.
```

### 4.4 Provider `Boot` eksik — `packages/console/testdata/doctor/missing_boot`

```
zatrano doctor
root: C:\Users\Pc\Desktop\ZATRANO\packages\console\testdata\doctor\missing_boot
findings: 7 (warnings)

[layout] warning  app/console
  found: missing directory app/console
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/console (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/http/controllers/api
  found: missing directory app/http/controllers/api
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/http/controllers/api (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/http/controllers/web
  found: missing directory app/http/controllers/web
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/http/controllers/web (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes
  found: missing directory app/routes
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes/api
  found: missing directory app/routes/api
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes/api (or regenerate the app with zatrano new) and keep types in the starter locations.

[layout] warning  app/routes/web
  found: missing directory app/routes/web
  why:   zatrano new places application code in this tree; agents and doctor assume it.
  how:   Create app/routes/web (or regenerate the app with zatrano new) and keep types in the starter locations.

[providers] warning  app/providers/broken.go:5
  found: type BrokenProvider missing Boot
  why:   kernel.Provider requires both Register and Boot so bootstrap.WithProviders can load the type.
  how:   Add func (p *BrokenProvider) Boot(app *kernel.Application) error on this type.
```

---

## 5. Üretilen `AGENTS.md` (tam içerik)

`go run ./cmd/zatrano agents:generate` (framework 1.6.6, sample route yok):

```markdown
# AGENTS.md

Generated by `zatrano agents:generate` from `zatrano describe` (framework 1.6.6). Re-run after upgrading the framework. Do not edit by hand.

## Routing

HTTP routes are registered with these primitives (parsed from `packages/routing/discovery.go`):

- `RegisterWeb(fn func(*Router))`
- `RegisterAPI(fn func(*Router))`
- `ApplyWeb(r *Router)`
- `ApplyAPI(r *Router)`

Put `RegisterWeb` / `RegisterAPI` in:

- `app/routes/`
- `app/routes/api/`
- `app/routes/web/`

`ApplyWeb` and `ApplyAPI` run from an application `Provider` under `app/providers` (typically `RouteServiceProvider`).

## Config and providers

Application providers implement `Provider` (`kernel/application.go`):

- `Register(app *Application) error`
- `Boot(app *Application) error`

Addon self-registration (parsed from `bootstrap/addons`):

- package: `github.com/zatrano/framework/bootstrap/addons`
- Register: `Register(m Meta)`
- Select: `Select(names ...string) ([]kernel.Provider, error)`
- Lookup: `Lookup(name string) (Meta, bool)`
- Available: `Available() []Meta`
- Meta: `Meta` fields `Name, Key, Description, Heavy, Factory, CLI`
- Meta.Factory: `func() kernel.Provider`
- register_called_from: `init`
- registry_imports_addons: `false`
- consumer_blank_import: `true`

Addon config maps belong in the addon's own Provider, not in kernel boot. Load application settings through `contracts.ConfigRepository` (`Application.Config()`).

## Catalog

### `LayerPrimitive` (`primitive`)

LayerPrimitive is always on the boot path: process must start; secure HTTP surface.

- `container` [service] — Service container
- `config` [service] — Configuration repository
- `env` [service] — Environment loader
- `context` [service] — Request/app context store
- `http` [service] — HTTP request/response helpers
- `routing` [service] — HTTP router
- `middleware` [service] — HTTP middleware primitives
- `pipeline` [service] — Middleware pipeline
- `exceptions` [service] — Exception handler
- `log` [service] — Application logger
- `encryption` [service] — Symmetric encryption
- `trustedproxy` [service] — Trusted proxy headers
- `report` [service] — Exception reporting

### `LayerFoundation` (`foundation`)

LayerFoundation is opt-in but common: typical web/API application services.

- `session` [service] — HTTP sessions
- `cookie` [service] — Cookie jar helpers
- `flash` [service] — Flash / toast messages
- `validation` [service] — Input validation
- `auth` [service] — Authentication guards
- `authorization` [service] — Gates and policies
- `hashing` [service] — Password hashing
- `cache` [service] — Cache manager
- `redisx` [service] — Redis client helper
- `database` [service] — Database manager
- `orm` [service] — Active-record ORM
- `view` [service] — HTML view engine
- `queue` [service] — Job queues
- `events` [service] — Event dispatcher
- `localization` [service] — Translator / locales
- `schedule` [service] — Task scheduler
- `filesystem` [service] — Filesystem disks
- `notification` [service] — Async multi-channel notifications (mail, SMS, push, database, broadcast)
- `broadcasting` [service] — Event broadcasting
- `httpclient` [service] — Outbound HTTP client
- `ratelimit` [service] — Rate limiter
- `url` [service] — URL generator
- `health` [service] — Health checks
- `observability` [service] — Metrics collector
- `maintenance` [service] — Maintenance mode
- `apitoken` [service] — Personal access tokens
- `assets` [service] — Asset manifest / Vite mix
- `console` [service] — CLI application
- `support` [service] — Support helpers
- `version` [service] — Version helper

### `LayerIntelligence` (`intelligence`)

LayerIntelligence is the first-party AI identity layer (same activation as addons).

- `ai` [service] — AI chat providers
- `rag` [library] — RAG chunking, embed pipeline, vector store helpers
- `agent` [library] — AI agent loop, tools, conversation memory

### `LayerAddon` (`addon`)

LayerAddon is project opt-in: services and libraries enabled by the consumer.

- `audit` [service] — Request/audit event log
- `backup` [service] — Database backup/restore (SQLite + native dump tools)
- `billing` [service] — Central billing manager (memory/stripe gateways, webhooks)
- `bus` [service] — Synchronous command bus
- `circuit` [service] — Circuit breaker
- `docs` [service] — Markdown docs repository
- `enums` [service] — String enum registry
- `features` [service] — Feature flags and gradual rollouts
- `geo` [service] — Geolocation resolver
- `graphql` [service] — GraphQL schema and queries
- `hashid` [service] — Obfuscated public IDs
- `inspector` [service] — Request inspector toolbar data
- `lock` [service] — Atomic locks
- `mongo` [service] heavy — MongoDB client (separate module)
- `oauth` [service] — OAuth2 authorization server
- `octane` [service] — Concurrent runtime metrics
- `otp` [service] — One-time passwords
- `pulse` [service] — Metrics pulse dashboard
- `search` [service] — In-memory search engine
- `shorturl` [service] — Short URL manager
- `sitemap` [service] — XML sitemap builder
- `social` [service] — Social OAuth login (GitHub/Google)
- `tenancy` [service] — Multi-tenant resolution
- `webauthn` [service] heavy — WebAuthn/passkeys (separate module)
- `webhooks` [service] — Outbound signed webhooks
- `wellknown` [service] — security.txt / well-known
- `api` [library] — API versioning helpers
- `archive` [library] — ZIP archive helpers
- `bloom` [library] — Bloom filter
- `browser` [library] — Headless browser testing
- `collection` [library] — Collection helpers
- `concurrency` [library] — Concurrency primitives
- `consent` [library] — Cookie/consent helpers
- `cron` [library] — Cron expression parser
- `debug` [library] — Debug dump helpers
- `export` [library] — CSV/XLSX import and export
- `factory` [library] — Model factories
- `fingerprint` [library] — Device fingerprinting
- `honeypot` [library] — Spam honeypot fields
- `idempotency` [library] — Idempotency keys
- `image` [library] — Image processing helpers
- `jsonapi` [library] — JSON:API document helpers
- `jsonschema` [library] — JSON Schema validation
- `markdown` [library] — Markdown renderer
- `negotiate` [library] — Content negotiation
- `openapi` [library] — OpenAPI generator helpers
- `pages` [library] — Static page helpers
- `pagination` [library] — Paginator helpers
- `pdf` [library] — PDF generation and inline viewing
- `process` [library] — OS process runner
- `qr` [library] heavy — QR code generation
- `resources` [library] — API resource transformers
- `testing` [library] — Test helpers
- `timing` [library] — Timing / stopwatch helpers
- `totp` [library] — TOTP codes
- `useragent` [library] — User-Agent parser
- `websocket` [library] — WebSocket helpers

## Contracts

Prefer these interfaces over concrete `packages/` types. Direct imports of contract concretes are reported by `zatrano doctor` (check `concrete`).

### `ConfigRepository` (`contracts/config.go`)

- `Get(key string, fallback ...any) any`
- `GetString(key string, fallback ...string) string`
- `GetInt(key string, fallback ...int) int`
- `All() map[string]any`
- `Load(name string, values map[string]any)`

### `Container` (`contracts/container.go`)

- `Instance(abstract string, instance any)`

### `ContextStore` (`contracts/services.go`)

- `Put(key string, value any)`

### `Encrypter` (`contracts/services.go`)

- `Encrypt(plaintext string) (string, error)`
- `Decrypt(payload string) (string, error)`

### `Exceptions` (`contracts/services.go`)

- `Middleware() routing.MiddlewareFunc`

### `Hasher` (`contracts/services.go`)

- `Make(value string) (string, error)`

### `Health` (`contracts/services.go`)

- `Custom(name string, check func(ctx context.Context) error)`
- `Database(db *sql.DB)`
- `Handler() routing.HandlerFunc`

### `Logger` (`contracts/logger.go`)

- `Debugf(format string, args ...any)`
- `Infof(format string, args ...any)`
- `Errorf(format string, args ...any)`

### `Maintenance` (`contracts/services.go`)

- `Enable(payload maintenance.Payload) error`
- `Disable() error`

### `Metrics` (`contracts/services.go`)

- `Snapshot() map[string]any`

### `RateLimiter` (`contracts/services.go`)

- `For(name string, limit ratelimit.Limit)`

### `Reports` (`contracts/services.go`)

- `Recent(limit int) []report.Event`

### `Router` (`contracts/router.go`)

- `Get(path string, handler routing.HandlerFunc) *routing.Route`
- `Post(path string, handler routing.HandlerFunc) *routing.Route`
- `Use(middleware ...routing.MiddlewareFunc)`
- `Group(prefix string, fn func(router *routing.Router), middleware ...routing.MiddlewareFunc)`
- `Name(prefix string, fn func(router *routing.Router))`
- `Routes() []*routing.Route`
- `Snapshot() []routing.RouteInfo`
- `SaveCache(path string) error`

### `URLGenerator` (`contracts/services.go`)

- `To(path string) string`
- `Route(name string, params ...map[string]string) (string, error)`
- `Signed(path string, expiresIn time.Duration, query ...map[string]string) (string, error)`

## Doctor

Run `zatrano doctor` from an application root (`app/` present). It reports warnings (not build errors) for:

- `routes`
- `concrete`
- `layout`
- `providers`

Each finding includes **found**, **why**, and **how** to fix. There is no `--fix` flag.
```

---

## 6. Kapsam dışı bırakılan işler

- Gerçek MCP server, wire protokolü, transport, tool-calling şeması
- `zatrano doctor --fix` otomatik düzeltme (kontroller `DoctorCheck.Run` olarak ayrıldı; `Fix` yok)
- `davet.link` veya başka belirli bir uygulamada deneme
- `packages/ai`, `packages/rag`, `packages/agent` içerik değişikliği (yalnızca katalogda `LayerIntelligence` olarak raporlandı)
- `package:doctor` ile birleştirme (addon sağlığı ayrı komut)

---

## 7. Build / test kanıtı

Aşağıdaki `go test ./...` Faz 5 sırasında alındı (`exit 0`). Rapor commit’inden hemen önce `sample_routes` için boş dilim (`[]`, `null` değil) düzeltmesi `go test ./packages/console` ve `go build ./...` ile yeniden doğrulandı.

### go build ./...
Exit code 0. Compiler stdout/stderr empty (success).

### go test ./...
Exit code 0.

```
ok  	github.com/zatrano/framework/bootstrap	0.618s
ok  	github.com/zatrano/framework/bootstrap/addons	(cached)
?   	github.com/zatrano/framework/bootstrap/foundation	[no test files]
?   	github.com/zatrano/framework/bootstrap/stubs	[no test files]
ok  	github.com/zatrano/framework/cmd/zatrano	(cached)
?   	github.com/zatrano/framework/config	[no test files]
?   	github.com/zatrano/framework/contracts	[no test files]
ok  	github.com/zatrano/framework/kernel	(cached)
ok  	github.com/zatrano/framework/packages/agent	(cached)
ok  	github.com/zatrano/framework/packages/ai	(cached)
ok  	github.com/zatrano/framework/packages/apitoken	(cached)
ok  	github.com/zatrano/framework/packages/assets	(cached)
ok  	github.com/zatrano/framework/packages/auth	(cached)
ok  	github.com/zatrano/framework/packages/auth/totp	(cached)
ok  	github.com/zatrano/framework/packages/authorization	(cached)
ok  	github.com/zatrano/framework/packages/broadcasting	(cached)
ok  	github.com/zatrano/framework/packages/cache	(cached)
ok  	github.com/zatrano/framework/packages/config	(cached)
ok  	github.com/zatrano/framework/packages/console	12.424s
?   	github.com/zatrano/framework/packages/container	[no test files]
?   	github.com/zatrano/framework/packages/context	[no test files]
?   	github.com/zatrano/framework/packages/cookie	[no test files]
ok  	github.com/zatrano/framework/packages/database	(cached)
?   	github.com/zatrano/framework/packages/database/migration	[no test files]
ok  	github.com/zatrano/framework/packages/database/query	(cached)
ok  	github.com/zatrano/framework/packages/database/schema	(cached)
?   	github.com/zatrano/framework/packages/database/seeder	[no test files]
?   	github.com/zatrano/framework/packages/encryption	[no test files]
ok  	github.com/zatrano/framework/packages/env	(cached)
ok  	github.com/zatrano/framework/packages/events	(cached)
ok  	github.com/zatrano/framework/packages/exceptions	(cached)
ok  	github.com/zatrano/framework/packages/filesystem	(cached)
ok  	github.com/zatrano/framework/packages/flash	(cached)
?   	github.com/zatrano/framework/packages/hashing	[no test files]
?   	github.com/zatrano/framework/packages/health	[no test files]
ok  	github.com/zatrano/framework/packages/http	(cached)
ok  	github.com/zatrano/framework/packages/http/useragent	(cached)
ok  	github.com/zatrano/framework/packages/httpclient	(cached)
ok  	github.com/zatrano/framework/packages/localization	(cached)
?   	github.com/zatrano/framework/packages/localization/defaults	[no test files]
ok  	github.com/zatrano/framework/packages/log	(cached)
ok  	github.com/zatrano/framework/packages/maintenance	(cached)
ok  	github.com/zatrano/framework/packages/middleware	(cached)
ok  	github.com/zatrano/framework/packages/middleware/csrf	(cached)
ok  	github.com/zatrano/framework/packages/notification	(cached)
ok  	github.com/zatrano/framework/packages/notification/export	(cached)
ok  	github.com/zatrano/framework/packages/notification/export/csv	(cached)
ok  	github.com/zatrano/framework/packages/notification/export/xlsx	(cached)
ok  	github.com/zatrano/framework/packages/observability	(cached)
ok  	github.com/zatrano/framework/packages/observability/timing	(cached)
ok  	github.com/zatrano/framework/packages/orm	(cached)
ok  	github.com/zatrano/framework/packages/orm/pagination	(cached)
ok  	github.com/zatrano/framework/packages/pipeline	(cached)
ok  	github.com/zatrano/framework/packages/queue	(cached)
ok  	github.com/zatrano/framework/packages/rag	(cached)
ok  	github.com/zatrano/framework/packages/ratelimit	(cached)
?   	github.com/zatrano/framework/packages/redisx	[no test files]
ok  	github.com/zatrano/framework/packages/report	(cached)
ok  	github.com/zatrano/framework/packages/routing	(cached)
ok  	github.com/zatrano/framework/packages/safepath	(cached)
ok  	github.com/zatrano/framework/packages/schedule	(cached)
ok  	github.com/zatrano/framework/packages/schedule/cron	(cached)
ok  	github.com/zatrano/framework/packages/session	(cached)
ok  	github.com/zatrano/framework/packages/support	(cached)
ok  	github.com/zatrano/framework/packages/support/arr	(cached)
ok  	github.com/zatrano/framework/packages/support/color	(cached)
ok  	github.com/zatrano/framework/packages/support/date	(cached)
ok  	github.com/zatrano/framework/packages/support/files	(cached)
ok  	github.com/zatrano/framework/packages/support/fn	(cached)
ok  	github.com/zatrano/framework/packages/support/html	(cached)
ok  	github.com/zatrano/framework/packages/support/money	(cached)
ok  	github.com/zatrano/framework/packages/support/num	(cached)
ok  	github.com/zatrano/framework/packages/support/once	(cached)
ok  	github.com/zatrano/framework/packages/support/str	(cached)
ok  	github.com/zatrano/framework/packages/support/uuid	(cached)
ok  	github.com/zatrano/framework/packages/trustedproxy	(cached)
ok  	github.com/zatrano/framework/packages/url	(cached)
ok  	github.com/zatrano/framework/packages/validation	(cached)
ok  	github.com/zatrano/framework/packages/version	(cached)
ok  	github.com/zatrano/framework/packages/view	(cached)
ok  	github.com/zatrano/framework/tests	1.114s
ok  	github.com/zatrano/framework/tests/fuzz	(cached)
?   	github.com/zatrano/framework/tests/securitydemo	[no test files]
```

---

## Muhafazakâr kararlar

- Doctor bulguları **uyarı**; süreç çıkış kodu 0 (CI’yi kırmamak için; `--strict` yok).
- Beklenen `app/` ağacı prompt metnindeki `app/controllers` değil, Stage B `zatrano new` şablonundaki gerçek dosyalardan (`.gitkeep` hariç) türetilir.
