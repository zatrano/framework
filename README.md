# ZATRANO

**ZATRANO** is an **enterprise-grade, opinionated modular monolith** Go framework: delivering **production-ready HTTP APIs (REST + OpenAPI)** and **robust server-rendered HTML form systems**, powered by a **first-class `zatrano` CLI** for scalable, standardized development workflows.

- **Module path:** `github.com/zatrano/framework`
- **Go:** 1.25+
- **Stack:** Fiber v3, PostgreSQL, Redis, GORM, Zap, golang-migrate (SQL migrations)

> **Status:** active development. Public Go APIs live under **`pkg/`** so generated apps can `import` the framework. This file is updated with every user-visible change.

---

## Table of Contents

- [Features](#features-roadmap)
- [Layout](#layout-pkg-vs-internal)
- [Requirements](#requirements)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [CLI Commands](#cli-commands)
- [HTTP Routes](#http-current)
- [Validation](#validation)
- [Authorization (RBAC & Gate/Policy)](#authorization-rbac--gatepolicy)
- [Cache System](#cache-system)
- [Queue / Job System](#queue--job-system)
- [Mail System](#mail-system)
- [Event / Listener System](#event--listener-system)
- [Internationalization (i18n)](#internationalization-i18n)
- [Configuration](#configuration)
- [Development](#development)

---

## Features (roadmap)

| Area | Plan |
|------|------|
| Architecture | Modular core + pluggable modules (modular monolith) |
| Layers | Handler → Service → Repository (mandatory bases) |
| Web | Fiber HTML templates, CSRF, **validation** (`go-playground/validator`), flash, **CORS**, **rate limit**, **i18n** (JSON locales), **cache** (Memory/Redis), security headers, gzip, static |
| API | REST + **OpenAPI 3** (`api/openapi.yaml`, `/docs`, `/openapi.yaml`) |
| Auth | **Session (Redis) + CSRF**; **JWT** for `/api/v1/private/*`; **OAuth2** (Google/GitHub) browser login; **RBAC** (role→permission, DB-backed); **Gate/Policy** (resource-based authorization) |
| Cache | **Memory / Redis** drivers, **Tag-based** invalidation, **Middleware** support |
| Queue | **Redis-backed** job queue, delayed jobs (ZADD), auto retry + exponential backoff, failed jobs (PostgreSQL) |
| Mail | **SMTP / Log** drivers, HTML templates with layouts, queue integration, attachments, Mailable pattern |
| Events | **Sync and async** event bus, `ShouldQueue` for queue-backed listeners, `gen event` + `gen listener` |
| Data | GORM + **`zatrano db migrate` / `rollback`** + **`db seed`** + **`db backup` / `restore`** (needs `pg_dump` / `pg_restore` / `psql` on PATH) |
| Ops | `/health`, `/ready`, `/status` |
| CLI | **`new`**, **`gen module`**, **`gen crud`**, **`gen request`**, **`gen policy`**, **`gen job`**, **`gen mail`**, **`gen event`**, **`gen listener`**, `serve`, `db`, **`cache`**, **`queue`**, **`mail`**, **`openapi export`**, `openapi validate`, **`jwt sign`**, … |

**Implemented now:** `serve`, `doctor`, **`routes`**, **`config print`**, **`config validate`**, **`verify`** (optional **`--race`**), `completion`, `version` / **`--version`**, **`new`**, **`gen module`** + **`gen crud`** + **`gen request`** + **`gen policy`** + **`gen job`** + **`gen mail`** + **`gen event`** + **`gen listener`** + **`gen wire`**, **`db`**, **`cache`** (Memory/Redis, Tags, middleware), **`queue`** (Redis FIFO, delayed jobs, retry, failed jobs, worker), **`mail`** (SMTP/log, templates, queue, attachments, preview), **`events`** (sync/async dispatch, ShouldQueue, queue-backed listeners), **`openapi validate`** + **`openapi export`**, **`jwt sign`**, **OAuth2**, **`http.*`** (CORS, rate limit, request timeout, body limit), **`i18n`** (JSON locales + Fiber helpers), **validation** (generic `Validate[T]`, i18n errors, custom rules, form requests), **authorization** (RBAC role→permission, Gate/Policy, `middleware.Can`, i18n 403), Redis session + CSRF, JWT, Scalar **`/docs`**, **Air** (`.air.toml`).

---

## Layout (`pkg/` vs `internal/`)

| Path | Purpose |
|------|---------|
| `pkg/config`, `pkg/core`, `pkg/server`, `pkg/health`, `pkg/middleware`, `pkg/security`, `pkg/auth`, `pkg/cache`, `pkg/queue`, `pkg/mail`, `pkg/events`, `pkg/oauth`, `pkg/openapi`, `pkg/i18n`, `pkg/validation`, `pkg/zatrano`, `pkg/meta` | **Public** — use from your apps |
| `internal/cli`, `internal/db`, `internal/gen` | **CLI & generators** — not imported by apps |

Generated apps use **`zatrano.Start`** with **`RegisterRoutes: routes.Register`** (see `internal/routes/register.go`) or **`zatrano.Run()`** when you do not inject routes.

---

## Requirements

- Go **1.25.0** or newer
- **PostgreSQL** for `db migrate` / GORM URLs
- **Redis** for session + CSRF (optional locally; required when you turn on `redis_url` / production sessions)
- **PostgreSQL client tools** (`pg_dump`, `pg_restore`, `psql`) on PATH for `zatrano db backup` and `db restore`

---

## Installation

Install the CLI globally:

```bash
go install github.com/zatrano/framework/cmd/zatrano@latest
```

---

## Quick start

Create a new app:

```bash
zatrano new app
cd app
zatrano serve
```

Or run the framework directly:

```bash
go run ./cmd/zatrano serve
```

Optional:

```bash
cp config/examples/dev.yaml config/dev.yaml
cp .env.example .env
```

Validate or export OpenAPI (export merges `api/openapi.yaml` with framework routes — same as live `/openapi.yaml`):

```bash
go run ./cmd/zatrano openapi validate api/openapi.yaml
go run ./cmd/zatrano openapi validate --merged
go run ./cmd/zatrano openapi export --output api/openapi.merged.yaml
```

---

## CLI commands

| Command | Purpose |
|---------|---------|
| `zatrano serve` | HTTP server (`--addr`, `--env`, `--config-dir`, `--no-dotenv`) |
| `zatrano doctor` | Config (incl. **HTTP** middleware summary) + Postgres/Redis checks |
| `zatrano routes` | Print routes (same config as `serve`; `--json`, `--all`, **`--group`**) |
| `zatrano config print` | Effective config, **masked** secrets; **`--paths-only`** short summary (default **lines**; `json` / `yaml`) |
| `zatrano config validate` | Load + **validate** only (no DB/Redis); **`--quiet`** / **`-q`** for CI exit code only |
| `zatrano new <name>` | Scaffold app (`--module`, `--output`, `--replace-zatrano` for local dev) |
| `zatrano db migrate` | Apply `migrations/*.up.sql` (golang-migrate) |
| `zatrano db rollback` | Roll back (`--steps`) |
| `zatrano db seed` | Run `db/seeds/*.sql` in one transaction (no-op if no `.sql` files) |
| `zatrano db backup` | `pg_dump` → file/dir (`--format`: custom, plain, or directory; `--output` or default under `backups/`) |
| `zatrano db restore` | `pg_restore` / `psql` (**requires `--yes`**, optional `--clean`) |
| `zatrano gen module <name>` | Scaffold `modules/<name>/`; **wires** + **`go fmt`** on wire file (`--skip-wire`, `--module-root`, `--out`, `--dry-run`) |
| `zatrano gen crud <name>` | Add CRUD stubs + **form request structs** (`requests/`); **wires** `RegisterCRUD` + **`go fmt`** (same flags) |
| `zatrano gen request <name>` | Generate form request structs only (`modules/<name>/requests/create_*.go`, `update_*.go`) |
| `zatrano gen policy <name>` | Generate authorization policy stub (`modules/<name>/policies/<name>_policy.go`) implementing `auth.Policy` with CRUD methods |
| `zatrano gen job <name>` | Generate queue job stub (`modules/jobs/<name>.go`) implementing `queue.Job` with Handle, Retries, Timeout |
| `zatrano gen mail <name>` | Generate Mailable struct + HTML template (`modules/mails/<name>_mail.go` + `views/mails/<name>.html`) |
| `zatrano gen event <name>` | Generate event struct (`modules/events/<name>_event.go`) implementing `events.Event` |
| `zatrano gen listener <name>` | Generate listener (`modules/listeners/<name>_listener.go`); use `--queued` for async |
| `zatrano gen wire <name>` | **Wire only** (no overwrite); picks `Register` / `RegisterCRUD` from existing files (`--register-only`, `--crud-only`) |
| `zatrano openapi validate [path]` | Validate one file, or **`--merged`** (same as live `/openapi.yaml`; `--base`, optional positional overrides base) |
| `zatrano openapi export` | Write merged YAML (`--base`, `--output` or `-` for stdout) |
| `zatrano jwt sign` | Print HS256 token (`--sub`, `--secret`, config flags) |
| `zatrano cache clear` | Clear all cache or specific tags (`--tag`) |
| `zatrano queue work` | Start queue worker process (`--queue`, `--tries`, `--timeout`, `--sleep`) |
| `zatrano queue failed` | List failed jobs |
| `zatrano queue retry [id]` | Retry a failed job or `--all` |
| `zatrano queue flush` | Delete all failed jobs |
| `zatrano mail preview [name]` | Preview email template in browser (`--port`, `--layout`) |
| `zatrano completion …` | Shell completions |
| `zatrano verify` | **`go vet` + `go test` + merged OpenAPI** (PR/CI; `--race` for data races; `--no-vet`, `--no-test`, `--no-openapi`, `--module-root`) |
| `zatrano version` | Version string (also **`zatrano --version`**) |

**Windows / paths with spaces:** use `--replace-zatrano` pointing at your checkout; the scaffold quotes the path in `go.mod` when needed.

---

## HTTP (current)

| Method | Path | Notes |
|--------|------|-------|
| GET | `/` | JSON index (`env`, `endpoints`, `http` flags for CORS/rate limit, `error_includes_request_id`) |
| GET | `/health`, `/ready`, `/status` | Liveness / readiness / aggregate (`/status` includes `env`) |
| GET | `/openapi.yaml` | **Merged** OpenAPI (your file + built-in ops; **`/`** and **`/status`** include JSON schemas) |
| GET | `/docs` | Scalar API reference (CDN) |
| GET | `/api/v1/public/ping` | Public JSON |
| GET | `/api/v1/private/me` | **Bearer JWT** required if `jwt_secret` set |
| POST | `/api/v1/auth/token` | **Only if** `security.demo_token_endpoint: true` (blocked when `env: prod`) |
| GET | `/auth/oauth/google/login`, `/auth/oauth/github/login` | Starts OAuth2 (requires `oauth.enabled` + provider keys) |
| GET | `/auth/oauth/google/callback`, `/auth/oauth/github/callback` | OAuth redirect handler |

**Session + CSRF:** enabled when `redis_url` is set and `security.session_enabled` / `csrf_enabled` are true. CSRF is skipped for `Authorization: Bearer …`, `csrf_skip_prefixes` (default includes `/api/`), and **`/auth/oauth/`** (OAuth callbacks).

**OAuth2:** set `oauth.enabled`, `oauth.base_url`, and `oauth.providers.google` / `github` client IDs. Redirect URLs in the provider console must be `{base_url}/auth/oauth/google/callback` (and the same for `github`). Session keys after login: `oauth_provider`, `oauth_subject`, `oauth_name`, `oauth_email`.

**Errors:** JSON responses use `{ "error": { "code", "message", "request_id"? } }`. `request_id` matches the **`X-Request-ID`** header when middleware runs (use it in logs and support tickets).

**HTTP middleware (`http` in YAML / `HTTP_*` env):**

- **CORS** — `http.cors_enabled`, `cors_allow_origins`, `cors_allow_methods`, `cors_allow_headers`, `cors_expose_headers`, `cors_allow_credentials`, `cors_max_age`. Default **off**. You cannot combine **`cors_allow_credentials: true`** with a wildcard origin **`*`** (browser rules); validation fails if you try.
- **Rate limit** — `http.rate_limit_enabled`, `rate_limit_max`, `rate_limit_window`, optional **`rate_limit_redis: true`** (uses **`redis_url`**; required if you enable Redis-backed limiting). Otherwise **in-memory** per process. Responses **under** the limit include **`X-RateLimit-*`** headers. When exceeded, **429** uses the same JSON `error` shape and Fiber sets **`Retry-After`** (RFC 6585).
- **Request timeout** — `http.request_timeout` (e.g. `60s`): Fiber **timeout** middleware; **408** JSON on overrun.
- **Body limit** — `http.body_limit` bytes (maps to Fiber **`BodyLimit`**; `0` = Fiber default **4 MiB**).

Order in the stack: **recover → request-id → i18n (if enabled) → CORS → request timeout → rate limit → helmet → compress → session/CSRF → routes**.

---

## Validation

ZATRANO provides a **generic, struct-tag based validation system** wrapping [`go-playground/validator/v10`](https://pkg.go.dev/github.com/go-playground/validator/v10) with automatic **422 JSON responses** and **i18n-translated error messages**.

### Quick Usage

The primary API is **`zatrano.Validate[T](c)`** — a single generic call that parses the request body, validates struct tags, and returns a structured 422 response on failure:

```go
import "github.com/zatrano/framework/pkg/zatrano"

func (h *ProductHandler) Create(c fiber.Ctx) error {
    req, err := zatrano.Validate[CreateProductRequest](c)
    if err != nil {
        return err // 422 JSON response already sent
    }
    // req is valid — use it
    return h.svc.Create(c.Context(), req.Name, req.Email)
}
```

### Form Request Structs

Define your request shapes as plain Go structs with `json` and `validate` tags:

```go
// requests/create_product.go
package requests

type CreateProductRequest struct {
    Name  string `json:"name"  validate:"required,min=2,max=255"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age"   validate:"gte=0,lte=150"`
}
```

```go
// requests/update_product.go
package requests

type UpdateProductRequest struct {
    Name  string `json:"name"  validate:"omitempty,min=2,max=255"`
    Email string `json:"email" validate:"omitempty,email"`
}
```

### Generating Request Structs

Use the CLI to scaffold request stubs automatically:

```bash
# Generate only request structs
zatrano gen request product
# → modules/product/requests/create_product.go
# → modules/product/requests/update_product.go

# gen crud also generates request structs automatically
zatrano gen crud product
# → modules/product/crud_handlers.go      (uses zatrano.Validate[T])
# → modules/product/crud_register.go
# → modules/product/requests/create_product.go
# → modules/product/requests/update_product.go
```

### 422 Error Response Format

When validation fails, the response body follows a consistent JSON structure:

```json
{
  "error": {
    "code": 422,
    "message": "validation failed",
    "details": [
      {
        "field": "Name",
        "tag": "required",
        "message": "This field is required"
      },
      {
        "field": "Email",
        "tag": "email",
        "value": "not-an-email",
        "message": "Must be a valid email address"
      }
    ]
  }
}
```

When i18n is enabled and the request locale is `tr`, messages are automatically translated:

```json
{
  "error": {
    "code": 422,
    "message": "validation failed",
    "details": [
      {
        "field": "Name",
        "tag": "required",
        "message": "Bu alan zorunludur"
      },
      {
        "field": "Email",
        "tag": "email",
        "value": "not-an-email",
        "message": "Geçerli bir e-posta adresi olmalıdır"
      }
    ]
  }
}
```

### i18n Validation Messages

Validation messages are stored in your locale files under the `validation.*` key namespace:

```json
// locales/en.json
{
  "validation": {
    "required": "This field is required",
    "email": "Must be a valid email address",
    "min": "Must be at least {{.Param}} characters",
    "max": "Must be at most {{.Param}} characters"
  }
}
```

The `{{.Param}}` placeholder is replaced with the constraint value (e.g. `min=5` → `"Must be at least 5 characters"`).

**Built-in translated tags:** `required`, `email`, `min`, `max`, `gte`, `lte`, `gt`, `lt`, `len`, `url`, `uri`, `uuid`, `oneof`, `numeric`, `number`, `alpha`, `alphanum`, `boolean`, `contains`, `excludes`, `startswith`, `endswith`, `ip`, `ipv4`, `ipv6`, `datetime`, `json`, `jwt`, `eqfield`, `nefield`.

### Custom Validation Rules

Register custom validation tags with optional i18n support:

```go
import (
    "github.com/go-playground/validator/v10"
    "github.com/zatrano/framework/pkg/zatrano"
)

// Register a custom rule
zatrano.RegisterRule("tc_no", func(fl validator.FieldLevel) bool {
    v := fl.Field().String()
    if len(v) != 11 {
        return false
    }
    // ... TC identity number algorithm
    return true
})

// With i18n message key (add "validation.tc_no" to your locale files)
zatrano.RegisterRuleWithMessage("tc_no", tcNoValidator, "validation.tc_no")
```

Then use it in struct tags:

```go
type CitizenRequest struct {
    TCNO string `json:"tc_no" validate:"required,tc_no"`
}
```

### Direct Engine Access

For advanced use cases, access the underlying validator engine:

```go
import "github.com/zatrano/framework/pkg/validation"

engine := validation.Default()
engine.Validator() // *validator.Validate from go-playground/validator

// Validate any struct programmatically (without Fiber context)
if verr := engine.ValidateStruct(myStruct, "en"); verr != nil {
    for _, fe := range verr.Errors {
        fmt.Printf("%s: %s\n", fe.Field, fe.Message)
    }
}
```

---

## Authorization (RBAC & Gate/Policy)

ZATRANO provides a **complete authorization system** with two complementary layers: **RBAC** (role-based, DB-backed) for permission checks and **Gate/Policy** (resource-based) for fine-grained instance-level authorization. Both integrate with the **i18n** system for localized 403 error messages.

### RBAC — Role-Based Access Control

Roles and permissions are stored in the database (`roles`, `permissions`, `role_permissions`, `zatrano_user_roles` tables). An in-memory cache avoids DB hits on hot-path permission checks. The `RBACManager` is initialized automatically during bootstrap (when DB is available) and accessible via `app.RBAC`.

```go
import "github.com/zatrano/framework/pkg/auth"

// Create roles and permissions
rbac := app.RBAC
rbac.CreateRole(ctx, "admin", "Administrator")
rbac.CreateRole(ctx, "editor", "Content editor")
rbac.CreatePermission(ctx, "posts.create", "Create posts")
rbac.CreatePermission(ctx, "posts.update", "Update posts")
rbac.CreatePermission(ctx, "posts.delete", "Delete posts")

// Assign permissions to roles
rbac.AssignPermissions(ctx, "admin", "posts.create", "posts.update", "posts.delete")
rbac.AssignPermissions(ctx, "editor", "posts.create", "posts.update")

// Assign roles to users
rbac.AssignRoleToUser(ctx, userID, "editor")

// Check permissions
ok, _ := rbac.UserHasPermission(ctx, userID, "posts.create") // true
ok, _ = rbac.UserHasPermission(ctx, userID, "posts.delete")  // false (editor can't delete)
```

**Database migration:** run `zatrano db migrate` — migration `000002_zatrano_rbac` creates the four required tables with proper indexes and foreign keys.

### Gate / Policy — Resource-Based Authorization

The `Gate` system (accessible via `app.Gate`) allows defining authorization checks for specific actions. Use `Define` for ad-hoc checks or `RegisterPolicy` for structured CRUD policies.

```go
import "github.com/zatrano/framework/pkg/auth"

// Ad-hoc gate definition
gate := app.Gate
gate.Define("edit-post", func(c fiber.Ctx, resource any) bool {
    post := resource.(*Post)
    userID, _ := c.Locals(middleware.LocalsUserID).(uint)
    return post.AuthorID == userID
})

// Super-admin bypass (runs before every gate check)
gate.Before(func(c fiber.Ctx, ability string, resource any) *bool {
    roles, _ := c.Locals(middleware.LocalsUserRoles).([]string)
    for _, r := range roles {
        if r == "super-admin" { t := true; return &t }
    }
    return nil // fall through to gate definition
})

// In handlers:
if err := gate.Authorize(c, "edit-post", post); err != nil {
    return err // 403 Forbidden
}
```

**Policy interface:** implement `auth.Policy` for structured CRUD authorization. Generate stubs with `zatrano gen policy`:

```bash
zatrano gen policy post
# → modules/post/policies/post_policy.go
```

The generated policy implements 7 methods: `ViewAny`, `View`, `Create`, `Update`, `Delete`, `ForceDelete`, `Restore`. Register it with the gate:

```go
import "myapp/modules/post/policies"

gate.RegisterPolicy("post", &policies.PostPolicy{})
// Creates: "post.viewAny", "post.view", "post.create", "post.update",
//          "post.delete", "post.forceDelete", "post.restore"
```

### Route-Level Authorization Middleware

The `pkg/middleware` package provides ready-to-use middleware for route-level permission and role checks. All return **403 JSON** with **i18n-aware** error messages.

| Middleware | Description |
|---|---|
| `middleware.Can(rbac, "perm")` | Requires the user to have a specific permission |
| `middleware.CanAny(rbac, "p1", "p2")` | Passes if the user has **any** of the listed permissions |
| `middleware.CanAll(rbac, "p1", "p2")` | Passes only if the user has **all** listed permissions |
| `middleware.HasRole("admin")` | Requires a specific role |
| `middleware.HasAnyRole("admin", "editor")` | Passes if the user has **any** of the listed roles |
| `middleware.GateAllows(gate, "ability")` | Checks a gate ability (without resource) |
| `middleware.InjectRoles(rbac)` | Loads user roles into Locals (place after auth middleware) |

```go
import "github.com/zatrano/framework/pkg/middleware"

// After authentication middleware:
app.Use(security.JWTMiddleware(cfg))
app.Use(middleware.InjectRoles(rbac))  // loads roles into context

// Permission-based
app.Get("/admin/users", middleware.Can(rbac, "users.view"), usersHandler)
app.Post("/posts", middleware.Can(rbac, "posts.create"), createPostHandler)
app.Delete("/system", middleware.CanAll(rbac, "system.admin", "system.delete"), handler)

// Role-based
app.Get("/dashboard", middleware.HasAnyRole("admin", "editor"), dashHandler)

// Gate-based
app.Get("/posts", middleware.GateAllows(gate, "post.viewAny"), listPostsHandler)
```

### 403 Error Response Format

When authorization fails, the response follows the standard JSON error shape:

```json
{
  "error": {
    "code": 403,
    "message": "You do not have permission to perform this action.",
    "permission": "posts.delete"
  }
}
```

When i18n is enabled and the request locale is `tr`:

```json
{
  "error": {
    "code": 403,
    "message": "Bu işlemi gerçekleştirme yetkiniz bulunmamaktadır.",
    "permission": "posts.delete"
  }
}
```

### i18n Authorization Messages

Authorization messages are stored under the `auth.*` key namespace in locale files:

```json
{
  "auth": {
    "forbidden": "You do not have permission to perform this action.",
    "unauthorized": "Authentication is required to access this resource.",
    "role_required": "You do not have the required role to access this resource.",
    "permission_required": "You do not have the required permission: {{.Permission}}."
  }
}
```

---

## Cache System

ZATRANO provides a **robust caching layer** with a unified API for **In-Memory** and **Redis** backends. It supports advanced patterns like `Remember`, JSON serialization, tag-based invalidation, and response middleware.

### Drivers

The system automatically chooses the best driver based on your configuration:
- **Redis:** Preferred when `redis_url` is configured. Supports distributed environments and tags.
- **Memory:** Fallback for local development or single-node deployments. Fast, but volatile.

### Basic Usage

Access the cache manager via `app.Cache`:

```go
import "context"

ctx := context.Background()

// Simple storage
app.Cache.Set(ctx, "key", "value", 10 * time.Minute)

// Retrieval
val, ok := app.Cache.Get(ctx, "key")

// Automatic JSON handling
type User struct { Name string }
app.Cache.SetJSON(ctx, "user:1", User{Name: "Alice"}, time.Hour)

var user User
ok, err := app.Cache.GetJSON(ctx, "user:1", &user)
```

### Advanced Patterns

#### `Remember` and `RememberJSON`

The most popular pattern (Laravl-style): returns the cached value if it exists, otherwise computes it via the provided function, caches it, and returns the result.

```go
// Fetch from DB only if not in cache
users, err := app.Cache.RememberJSON(ctx, "users:all", 30*time.Minute, &[]User{}, func() (any, error) {
    return db.FindAllUsers(ctx)
})
```

#### Tags (Redis Only)

Group related keys under tags for bulk invalidation.

```go
// Store under a tag
app.Cache.Tags("users").Set(ctx, "users:1", data, time.Hour)

// Invalidate all keys associated with a tag
app.Cache.Tags("users").Flush(ctx)
```

### Middleware

Cache the entire response of a route at the HTTP level.

```go
import "github.com/zatrano/framework/pkg/middleware"

// Cache for 5 minutes
app.Get("/api/v1/stats", middleware.Cache(app.Cache, 5*time.Minute), handler)

// With Tags
app.Get("/api/v1/users", middleware.CacheWithConfig(app.Cache, middleware.CacheConfig{
    TTL:  10 * time.Minute,
    Tags: []string{"users"},
}), handler)
```

### CLI Commands

Clear the cache from the terminal:

```bash
# Clear everything
zatrano cache clear

# Clear specific tags
zatrano cache clear --tag users --tag posts
```

---

## Queue / Job System

ZATRANO provides a **Redis-backed background job queue** with delayed scheduling, automatic retry with exponential backoff, and failed job persistence to PostgreSQL.

### Defining Jobs

Implement the `queue.Job` interface or embed `queue.BaseJob` for sensible defaults:

```go
package jobs

import (
    "context"
    "time"
    "github.com/zatrano/framework/pkg/queue"
)

type SendEmailJob struct {
    queue.BaseJob
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

func (j *SendEmailJob) Name() string            { return "send_email" }
func (j *SendEmailJob) Queue() string           { return "emails" }
func (j *SendEmailJob) Retries() int            { return 5 }
func (j *SendEmailJob) Timeout() time.Duration  { return 30 * time.Second }

func (j *SendEmailJob) Handle(ctx context.Context) error {
    // send the email...
    return mailer.Send(ctx, j.To, j.Subject, j.Body)
}
```

Generate job stubs with the CLI:

```bash
zatrano gen job send_email
# → modules/jobs/send_email.go
```

### Dispatching Jobs

```go
// Register job types at startup
app.Queue.Register("send_email", func() queue.Job { return &jobs.SendEmailJob{} })

// Dispatch immediately
app.Queue.Dispatch(ctx, &jobs.SendEmailJob{
    To:      "user@example.com",
    Subject: "Welcome!",
    Body:    "Hello world",
})

// Dispatch with delay (Redis ZADD sorted set)
app.Queue.Later(ctx, 5*time.Minute, &jobs.SendEmailJob{
    To:      "user@example.com",
    Subject: "Follow-up",
})
```

### Worker Process

Start a long-running worker that processes jobs from the queue:

```bash
zatrano queue work
zatrano queue work --queue emails --queue notifications
zatrano queue work --tries 5 --timeout 120s --sleep 5s
```

The worker automatically:
- Polls Redis using BRPOP (FIFO order)
- Migrates delayed jobs (ZADD → LPUSH) every second
- Retries failed jobs with **exponential backoff** (2^attempt seconds)
- Records permanently failed jobs in the `zatrano_failed_jobs` PostgreSQL table
- Recovers from panics inside `Handle()`
- Shuts down gracefully on SIGINT/SIGTERM

### Failed Jobs

Jobs that exceed their maximum retry count are saved to PostgreSQL with error message, stack trace, and original payload.

```bash
# List failed jobs
zatrano queue failed

# Retry a specific failed job
zatrano queue retry 42

# Retry all failed jobs
zatrano queue retry --all

# Delete all failed job records
zatrano queue flush
```

**Database migration:** run `zatrano db migrate` — migration `000003_zatrano_failed_jobs` creates the required table.

### Queue Architecture

| Component | Redis Structure | Purpose |
|---|---|---|
| Ready queue | `LIST` (LPUSH/BRPOP) | FIFO job processing |
| Delayed jobs | `SORTED SET` (ZADD) | Time-based scheduling |
| Failed jobs | PostgreSQL table | Persistent failure records |

---

## Mail System

ZATRANO provides a **multi-driver mail system** with HTML template support, queue integration for async sending, attachments, and a Mailable pattern for reusable email definitions.

### Configuration

```yaml
# config/dev.yaml
mail:
  driver: smtp          # smtp | log (log = dev/testing)
  from_name: "My App"
  from_email: "noreply@myapp.com"
  templates_dir: "views/mails"
  smtp:
    host: smtp.example.com
    port: 587
    username: user
    password: secret
    encryption: tls     # tls | starttls | ""
```

### Sending Emails

```go
import "github.com/zatrano/framework/pkg/mail"

// Simple message
app.Mail.Send(ctx, &mail.Message{
    To:      []mail.Address{{Email: "user@example.com", Name: "Alice"}},
    Subject: "Welcome!",
    HTMLBody: "<h1>Hello Alice!</h1>",
})

// With template
app.Mail.SendTemplate(ctx,
    []mail.Address{{Email: "user@example.com"}},
    "Welcome to Our App",
    "welcome",    // views/mails/welcome.html
    "default",    // views/mails/layouts/default.html
    map[string]any{"Name": "Alice"},
)

// Async via queue
app.Mail.Queue(ctx, &mail.Message{
    To:      []mail.Address{{Email: "user@example.com"}},
    Subject: "Newsletter",
    HTMLBody: body,
})
```

### Mailable Pattern

Generate structured, reusable email definitions:

```bash
zatrano gen mail welcome
# → modules/mails/welcome_mail.go
# → views/mails/welcome.html
```

```go
type WelcomeMail struct {
    Name  string
    Email string
}

func (m *WelcomeMail) Build(b *mail.MessageBuilder) error {
    b.To(m.Name, m.Email).
        Subject("Welcome!").
        View("welcome", "default", map[string]any{"Name": m.Name}).
        AttachData("guide.pdf", pdfBytes, "application/pdf")
    return nil
}

// Send synchronously
app.Mail.SendMailable(ctx, &mails.WelcomeMail{Name: "Alice", Email: "alice@example.com"})

// Or async via queue
app.Mail.QueueMailable(ctx, &mails.WelcomeMail{Name: "Alice", Email: "alice@example.com"})
```

### Attachments

```go
msg := &mail.Message{
    To:      []mail.Address{{Email: "user@example.com"}},
    Subject: "Invoice",
    HTMLBody: body,
    Attachments: []mail.Attachment{
        {Filename: "invoice.pdf", Content: pdfBytes},
        {Filename: "logo.png", Content: logoBytes, Inline: true},
    },
}
app.Mail.Send(ctx, msg)
```

### Template Preview

Preview email templates in the browser during development:

```bash
zatrano mail preview              # list templates
zatrano mail preview welcome      # preview welcome template
zatrano mail preview welcome --port 3001
```

For full local mail testing, use **Mailpit** or **MailHog** as the SMTP host.

---

## Event / Listener System

ZATRANO provides a **central event bus** (pub/sub) with support for synchronous and asynchronous listeners, queue-backed delivery via `ShouldQueue`, and generators for rapid development.

### Registering Listeners

```go
// In your service provider / bootstrap (e.g. events/event_service_provider.go):

// Sync listener (inline)
app.Events.ListenFunc("user.created", func(ctx context.Context, e events.Event) error {
    log.Println("user created", e)
    return nil
})

// Struct listener
app.Events.Listen("user.created", &listeners.SendWelcomeMailListener{})

// Multiple listeners for one event
app.Events.Subscribe("order.placed",
    &listeners.SendOrderConfirmationListener{},
    &listeners.UpdateInventoryListener{},
)
```

### Firing Events

```go
import "github.com/zatrano/framework/pkg/events"

// Define an event
type UserCreatedEvent struct {
    events.BaseEvent
    UserID uint
    Email  string
}
func (e *UserCreatedEvent) Name() string { return "user.created" }

// Fire synchronously (blocks until all sync listeners complete)
app.Events.Fire(ctx, &UserCreatedEvent{UserID: 1, Email: "alice@example.com"})

// Fire asynchronously (goroutines, errors only logged)
app.Events.FireAsync(ctx, &UserCreatedEvent{UserID: 1, Email: "alice@example.com"})
```

### Async Listeners via Queue (`ShouldQueue`)

Implement `ShouldQueue` to dispatch a listener as a queue job:

```go
type SendWelcomeMailListener struct{}

func (l *SendWelcomeMailListener) Handle(ctx context.Context, event events.Event) error {
    // runs in a background worker
    return nil
}

func (l *SendWelcomeMailListener) Queue() string { return "events" }   // queue name
func (l *SendWelcomeMailListener) Retries() int  { return 3 }
```

When `ShouldQueue` is implemented and a queue is configured (Redis), the listener is automatically dispatched via the Queue system instead of running inline.

### Generator

```bash
zatrano gen event user_created
# → modules/events/user_created_event.go

zatrano gen listener send_welcome_mail
# → modules/listeners/send_welcome_mail_listener.go  (sync)

zatrano gen listener send_welcome_mail --queued
# → modules/listeners/send_welcome_mail_listener.go  (ShouldQueue / async)
```

### Event Service Provider

Centralise all listener registrations in one place:

```go
// modules/events/event_service_provider.go
package myevents

import (
    "github.com/zatrano/framework/pkg/core"
    "myapp/modules/listeners"
)

// Register wires all event listeners. Call from main or bootstrap.
func Register(app *core.App) {
    app.Events.Listen("user.created", &listeners.SendWelcomeMailListener{})
    app.Events.Listen("order.placed", &listeners.SendOrderConfirmationListener{})
}
```

---

### Internationalization (`i18n`)

Application UI copy lives in **JSON** files under **`locales_dir`**, one file per locale: **`{locales_dir}/{tag}.json`** (e.g. `locales/en.json`). Nested objects are flattened to **dot keys** (`app.welcome`).

- **Config:** `i18n.enabled`, `i18n.default_locale`, `i18n.supported_locales`, `i18n.locales_dir`, optional `i18n.cookie_name` (default `zatrano_lang`), `i18n.query_key` (default `lang`). When **`i18n.enabled`** is true, **`locales_dir`** must exist on disk (validated at config load).
- **Resolution order:** query (`?lang=`), cookie, **`Accept-Language`**, then **`default_locale`**.
- **Handlers:** `import "github.com/zatrano/zatrano/pkg/i18n"` — **`i18n.T(c, "app.welcome")`** for static strings; **`i18n.Tf(c, "app.hello_user", map[string]any{"Name": userName})`** (or any struct) for **`text/template`** placeholders such as **`{{.Name}}`** in JSON. For **`map`** data, simple `{{.Field}}` segments are rewritten automatically; use **`Bundle.Format(locale, key, data)`** without Fiber. If i18n is off, **`T`** / **`Tf`** return the key unchanged ( **`Tf`** also returns **`nil`** error).
- **GET /** includes an **`i18n`** object (`enabled`, and when on: `default_locale`, `supported_locales`, **`active_locale`** for the current request).
- **Validation messages** are automatically resolved from `validation.*` keys when i18n is enabled (see [Validation](#validation)).

---

## Configuration

- **`.env`**, **`config/{env}.yaml`**, **environment variables** (nested keys use underscores, e.g. `SECURITY_JWT_SECRET`). For **lists** (e.g. multiple CORS origins or **`supported_locales`**), prefer **YAML**; env overrides for slices vary by shell.
- Key fields: `migrations_dir`, `seeds_dir`, `openapi_path`, **`http.*`**, **`i18n.*`**, `security.*`, `oauth.*` (see `config/examples/dev.yaml`).
- Debug: **`zatrano config print`** (full dump, redacted) or **`zatrano config print --paths-only`** (env, cwd, profile path, dirs — safe to paste in chat).
- CI: **`zatrano config validate -q`** (fast YAML/env checks), then **`zatrano openapi validate --merged`**, or **`zatrano verify`** for the full gate (see Development).

---

## Development

```bash
go test ./... -count=1
go fmt ./...
go vet ./...
golangci-lint run   # when installed
```

**One-shot gate:** `zatrano verify` (or **`make verify`** on POSIX) runs `vet`, `test`, and merged OpenAPI validation. **`make verify-race`** / **`zatrano verify --race`** before release builds (slower; catches data races). **`make config-validate`** mirrors **`zatrano config validate`**.

**Live reload:** install [Air](https://github.com/air-verse/air), then `air` (uses `.air.toml`). On Windows the binary is `./tmp/main.exe`.

**Merged OpenAPI file:** `make openapi-export` (POSIX Make) or `go run ./cmd/zatrano openapi export --output api/openapi.merged.yaml`.

**Environment check:** `zatrano doctor` prints config summary, **`http` middleware** (CORS, rate limit, timeout, body limit) and a pointer to **`config print --paths-only`**, **OAuth** when enabled, **`pg_dump` / `pg_restore` / `psql`** PATH resolution for backup/restore, then connectivity probes.

**Generate code:** `gen module` / `gen crud` patch **`zatrano:wire:*`** markers and run **`go fmt`** on the wire file. **`gen wire`** does the same patch without regenerating `modules/` (e.g. after **`--skip-wire`**). **`gen request`** generates form request struct stubs independently. **`gen policy`** generates an `auth.Policy` implementation with CRUD methods (ViewAny, View, Create, Update, Delete, ForceDelete, Restore). **Apps:** `internal/routes/register.go`. **Framework checkout:** `pkg/server/register_modules.go`.

**Embedding the server:** `server.Mount(app, fiberApp, server.MountOptions{RegisterRoutes: …})`; `zatrano.StartOptions.RegisterRoutes` passes through for generated apps.

---

## Documentation

- **English:** this file (`README.md`)
- **Türkçe:** [`README.tr.md`](README.tr.md)

Keep both in sync when adding or changing features.

---

## Contributing

Issues and PRs are welcome. For any behavior or CLI change, update **both** `README.md` and `README.tr.md` in the same change.

---

## License

To be determined.
