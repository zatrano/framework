# ZATRANO packages

What each first-party package under `packages/` is for.

Source of truth: `core/catalog.go` plus the `packages/` directory inventory.  
Full guides: [zatrano.com/docs](https://zatrano.com/docs) · CLI: `go run ./cmd/zatrano package:list --all`

## Layers

| Layer | Meaning | How to enable |
|-------|---------|----------------|
| **Kernel** | Secure HTTP boot surface | Always on |
| **Foundation** | Typical web/API stack | `MinimalApp` / `App` / `APP_BOOT=…` |
| **Addon (service)** | Optional container-bound service | `package:enable` + `EnabledAddons` |
| **Addon (library)** | Import-only helper | `import` only — **never** list in `EnabledAddons` |

Heavy = separate Go module / heavy dependency (`mongo`, `webauthn`, `qr`).

---

## Kernel

| Package | Purpose |
|---------|---------|
| `container` | Service container (DI) |
| `config` | Configuration repository |
| `env` | `.env` environment loader |
| `context` | Request / app context store |
| `http` | HTTP request and response helpers |
| `routing` | HTTP router |
| `middleware` | HTTP middleware primitives (includes `csrf` subpackage) |
| `pipeline` | Middleware pipeline |
| `exceptions` | Exception handler |
| `log` | Application logger |
| `encryption` | Symmetric encryption (AES-GCM) |
| `trustedproxy` | Trusted proxy headers |
| `report` | Exception reporting |

---

## Foundation

| Package | Purpose |
|---------|---------|
| `session` | HTTP sessions |
| `cookie` | Cookie jar helpers |
| `flash` | Flash / toast messages |
| `validation` | Input validation (rules, FormRequest) |
| `auth` | Authentication (guards, MFA, reset, remember-me) |
| `authorization` | Authorization (gates / policies) |
| `hashing` | Password hashing (bcrypt) |
| `cache` | Cache manager (file / memory / redis) |
| `redisx` | Redis client helper |
| `database` | DB manager; subpackages: `query`, `schema`, `migration`, `seeder`, `driver/*` |
| `orm` | Active-record ORM (relations, eager load, soft deletes, …) |
| `view` | HTML view engine |
| `queue` | Background job queues |
| `events` | Event dispatcher / observers |
| `localization` | Translator / locales |
| `schedule` | Task scheduler (`schedule:run`) |
| `filesystem` | Filesystem disks |
| `notification` | Async notifications (mail, SMS, push, database, broadcast) |
| `broadcasting` | Event broadcasting (log / file / null) |
| `httpclient` | Outbound HTTP client |
| `ratelimit` | Request rate limiter |
| `url` | URL and signed URL generator |
| `health` | Health checks |
| `observability` | Metrics collector |
| `maintenance` | Maintenance mode |
| `apitoken` | Personal access tokens (Bearer) |
| `assets` | Asset manifest (Vite / Mix) |
| `console` | CLI (`cmd/zatrano` commands) |
| `support` | General support helpers |
| `version` | Version helper |

> **Mail:** There is no separate `packages/mail`. Send email through `notification` with the `mail` channel.

---

## Addon — service (`package:enable`)

| Package | Purpose | Heavy |
|---------|---------|:-----:|
| `ai` | AI chat providers | |
| `audit` | Request / audit event log | |
| `backup` | Database backup / restore | |
| `billing` | Billing (memory / Stripe, webhooks) | |
| `bus` | Synchronous command bus | |
| `circuit` | Circuit breaker | |
| `docs` | Markdown docs repository | |
| `enums` | String enum registry | |
| `features` | Feature flags / gradual rollouts | |
| `geo` | Geolocation resolver | |
| `graphql` | GraphQL schema and queries | |
| `hashid` | Obfuscated public IDs | |
| `inspector` | Request inspector toolbar data | |
| `lock` | Atomic locks (process-local) | |
| `mongo` | MongoDB client | ✓ |
| `oauth` | OAuth2 **authorization server** | |
| `octane` | Concurrent runtime metrics | |
| `otp` | One-time passwords (OTP) | |
| `pulse` | Metrics pulse dashboard | |
| `search` | In-memory search engine | |
| `shorturl` | Short URL manager | |
| `sitemap` | XML sitemap builder | |
| `social` | Social OAuth login (GitHub / Google) — docs: Socialite | |
| `tenancy` | Multi-tenant resolution | |
| `webauthn` | WebAuthn / passkeys | ✓ |
| `webhooks` | Outbound signed webhooks | |
| `wellknown` | `security.txt` / `.well-known` | |

---

## Addon — library (import only)

| Package | Purpose | Heavy |
|---------|---------|:-----:|
| `api` | API versioning helpers | |
| `archive` | ZIP archive helpers | |
| `bloom` | Bloom filter | |
| `browser` | Headless browser testing | |
| `collection` | Collection helpers | |
| `concurrency` | Concurrency primitives | |
| `consent` | Cookie / consent helpers | |
| `cron` | Cron expression parser | |
| `debug` | Debug dump helpers | |
| `export` | CSV / XLSX import and export | |
| `factory` | Model factories | |
| `fingerprint` | Device fingerprinting | |
| `honeypot` | Spam honeypot fields | |
| `idempotency` | Idempotency-Key middleware | |
| `image` | Image processing | |
| `jsonapi` | JSON:API document helpers | |
| `jsonschema` | JSON Schema validation | |
| `markdown` | Markdown → HTML | |
| `negotiate` | Content negotiation | |
| `openapi` | OpenAPI generator helpers | |
| `pages` | File-based page routes | |
| `pagination` | Paginator helpers | |
| `pdf` | PDF generation / inline viewing | |
| `process` | OS process runner | |
| `qr` | QR code generation | ✓ |
| `resources` | API resource transformers | |
| `testing` | Feature test helpers | |
| `timing` | Server-Timing / stopwatch | |
| `totp` | TOTP (authenticator) codes | |
| `useragent` | User-Agent parser | |
| `websocket` | WebSocket upgrade helpers | |

---

## Not in catalog / internal helper

Present under `packages/` but not listed in `core/catalog.go`:

| Package | Purpose |
|---------|---------|
| `safepath` | Path traversal / zip-slip protection (used by filesystem, archive, …) |

---

## `database` subpackages

Catalog entry is `database`; on disk it includes:

| Path | Purpose |
|------|---------|
| `packages/database/query` | SQL query builder |
| `packages/database/schema` | Schema builder |
| `packages/database/migration` | Migrations |
| `packages/database/seeder` | Seeder runner |
| `packages/database/driver/*` | Opt-in drivers: sqlite, mysql, pgsql, mssql, oracle, mongo |

---

## Common mix-ups

| Need | Package |
|------|---------|
| Session login / MFA | `auth` |
| Authorization (policy / gate) | `authorization` |
| Send email | `notification` (`Channels: ["mail"]`) |
| SQL / models | `database` + `orm` |
| Social login | `social` |
| OAuth **server** | `oauth` |
| API Bearer tokens | `apitoken` |
| Redis | `redisx` (+ `cache` / `queue`) |
| Authenticator apps | `totp` (+ `auth` MFA) |
| Form spam | `honeypot` (+ `middleware/csrf`) |

---

## CLI

```bash
go run ./cmd/zatrano package:list
go run ./cmd/zatrano package:list --libraries
go run ./cmd/zatrano package:list --all
go run ./cmd/zatrano package:status
go run ./cmd/zatrano package:enable billing social
go run ./cmd/zatrano package:doctor
```

## Related

- [README.md](README.md) — install and overview  
- [Package Ecosystem](https://zatrano.com/docs/package-ecosystem)  
- [Boot Profiles](https://zatrano.com/docs/boot-profiles)  
- `core/catalog.go` — machine-readable catalog  
