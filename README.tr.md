# ZATRANO

**ZATRANO**, **kurumsal seviyede, güçlü görüşlere sahip bir modüler monolith** Go framework'üdür: **üretim hazır HTTP API'ler (REST + OpenAPI)** ve **güçlü sunucu tarafı HTML form sistemleri** sunar; ölçeklenebilir ve standartlaştırılmış geliştirme süreçleri için birinci sınıf **`zatrano` CLI** ile desteklenir.

- **Modül yolu:** `github.com/zatrano/framework`
- **Go:** 1.25+
- **Yığın:** Fiber v3, PostgreSQL, Redis, GORM, Zap, golang-migrate (SQL migration)

> **Durum:** aktif geliştirme. Üretilen uygulamaların import edebilmesi için genel API **`pkg/`** altında. Kullanıcıya görünen her değişiklikte bu dosya güncellenir.

---

## İçindekiler

- [Özellikler](#özellikler-yol-haritası)
- [Dizilim](#dizilim-pkg-ve-internal)
- [Gereksinimler](#gereksinimler)
- [Kurulum](#kurulum)
- [Hızlı Başlangıç](#hızlı-başlangıç)
- [CLI Komutları](#cli-komutları)
- [HTTP Rotaları](#http-şimdilik)
- [Validation (Doğrulama)](#validation-doğrulama)
- [Yetkilendirme (RBAC & Gate/Policy)](#yetkilendirme-rbac--gatepolicy)
- [Cache Sistemi (Önbellek)](#cache-sistemi-önbellek)
- [Kuyruk / Job Sistemi](#kuyruk--job-sistemi)
- [Mail Sistemi (E-posta)](#mail-sistemi-e-posta)
- [Uluslararasılaştırma (i18n)](#uluslararasılaştırma-i18n)
- [Yapılandırma](#yapılandırma)
- [Geliştirme](#geliştirme)

---

## Özellikler (yol haritası)

| Alan | Plan |
|------|------|
| Mimari | Modüler çekirdek + takılabilir modüller (modüler monolith) |
| Katmanlar | Handler → Service → Repository (zorunlu; hepsinde base) |
| Web | Fiber HTML şablonları, CSRF, **validation** (`go-playground/validator`), flash, **CORS**, **rate limit**, **i18n** (JSON çeviriler), **cache** (Memory/Redis), güvenlik başlıkları, gzip, static |
| API | REST + **OpenAPI 3** (`api/openapi.yaml`, `/docs`, `/openapi.yaml`) |
| Kimlik | **Oturum (Redis) + CSRF**; `/api/v1/private/*` için **JWT**; **OAuth2** (Google/GitHub) tarayıcı girişi; **RBAC** (rol→izin, DB destekli); **Gate/Policy** (kaynak bazlı yetkilendirme) |
| Veri | GORM + **`db migrate` / `rollback` / `seed`** + **`db backup` / `restore`** (`pg_dump` / `pg_restore` / `psql` PATH'te olmalı) |
| Kuyruk | **Redis tabanlı** job kuyruğu, geciktirilmiş joblar (ZADD), otomatik retry + üssel geri çekilme, başarısız joblar (PostgreSQL) |
| Mail | **SMTP / Log** sürücüleri, HTML şablon + layout desteği, kuyruk entegrasyonu, ek dosya, Mailable deseni |
| Operasyon | `/health`, `/ready`, `/status` |
| CLI | **`new`**, **`gen module`**, **`gen crud`**, **`gen request`**, **`gen policy`**, **`gen job`**, **`gen mail`**, `serve`, `db`, **`cache`**, **`queue`**, **`mail`**, **`openapi export`**, `openapi validate`, **`jwt sign`**, … |

**Şu an hazır:** `serve`, `doctor`, **`routes`**, **`config print`**, **`config validate`**, **`verify`** (isteğe **`--race`**), `completion`, `version` / **`--version`**, **`new`**, **`gen module`** + **`gen crud`** + **`gen request`** + **`gen policy`** + **`gen job`** + **`gen mail`** + **`gen wire`**, **`db`**, **`cache`** (Memory/Redis, Tags, middleware), **`queue`** (Redis FIFO, geciktirilmiş joblar, retry, failed jobs, worker), **`mail`** (SMTP/log, şablonlar, kuyruk, ek dosya, önizleme), **`openapi validate`** + **`openapi export`**, **`jwt sign`**, **OAuth2**, **`http.*`** (CORS, rate limit, istek süresi, gövde boyutu), **`i18n`** (JSON yereller + Fiber yardımcıları), **validation** (generic `Validate[T]`, i18n hata mesajları, özel kurallar, form request'ler), **yetkilendirme** (RBAC rol→izin, Gate/Policy, `middleware.Can`, i18n 403), Redis + CSRF, JWT, Scalar **`/docs`**, **Air** (`.air.toml`).

---

## Dizilim (`pkg/` ve `internal/`)

| Yol | Amaç |
|-----|------|
| `pkg/config`, `pkg/core`, `pkg/server`, `pkg/health`, `pkg/middleware`, `pkg/security`, `pkg/auth`, `pkg/cache`, `pkg/queue`, `pkg/mail`, `pkg/oauth`, `pkg/openapi`, `pkg/i18n`, `pkg/validation`, `pkg/zatrano`, `pkg/meta` | **Genel API** — uygulamalar import eder |
| `internal/cli`, `internal/db`, `internal/gen` | **CLI ve üreticiler** — uygulama import etmez |

Üretilen projeler **`zatrano.Start`** + **`RegisterRoutes: routes.Register`** (`internal/routes/register.go`) veya ek rota yoksa **`zatrano.Run()`** kullanır.

---

## Gereksinimler

- Go **1.25.0+**
- **`db migrate`** ve GORM için **PostgreSQL**
- **Redis** — oturum + CSRF için (yerelde isteğe bağlı; prod'da genelde zorunlu)
- **PostgreSQL istemci araçları** — `zatrano db backup` ve `db restore` için `pg_dump`, `pg_restore`, `psql` PATH'te olmalı

---

## Kurulum

CLI'yi global olarak yükleyin:

```bash
go install github.com/zatrano/framework/cmd/zatrano@latest
```

---

## Hızlı başlangıç

Yeni uygulama oluşturun:

```bash
zatrano new app
cd app
zatrano serve
```

Veya framework'ü doğrudan çalıştırın:

```bash
go run ./cmd/zatrano serve
```

İsteğe bağlı:

```bash
cp config/examples/dev.yaml config/dev.yaml
cp .env.example .env
```

OpenAPI doğrulama ve dışa aktarma:

```bash
go run ./cmd/zatrano openapi validate api/openapi.yaml
go run ./cmd/zatrano openapi validate --merged
go run ./cmd/zatrano openapi export --output api/openapi.merged.yaml
```

---

## CLI komutları

| Komut | Açıklama |
|-------|----------|
| `zatrano serve` | HTTP sunucusu (`--addr`, `--env`, `--config-dir`, `--no-dotenv`) |
| `zatrano doctor` | Yapılandırma (**HTTP** ara katman özeti dahil) + bağlantı kontrolleri |
| `zatrano routes` | Rotalar (`serve` ile aynı config; `--json`, `--all`, **`--group`**) |
| `zatrano config print` | Maskeli tam çıktı; **`--paths-only`** kısa özet (varsayılan **satırlar**; `json` / `yaml`) |
| `zatrano config validate` | Yükle + **doğrula** (DB/Redis yok); CI için **`--quiet`** / **`-q`** (yalnızca çıkış kodu) |
| `zatrano new <name>` | Yeni uygulama (`--module`, `--output`, yerel geliştirme için `--replace-zatrano`) |
| `zatrano db migrate` | SQL migration uygula |
| `zatrano db rollback` | Geri al (`--steps`) |
| `zatrano db seed` | `db/seeds/*.sql` (yoksa no-op) |
| `zatrano db backup` | `pg_dump` → dosya (`--format`, `--output` veya varsayılan `backups/`) |
| `zatrano db restore` | `pg_restore` / `psql` (**`--yes` zorunlu**, isteğe `--clean`) |
| `zatrano gen module <name>` | `modules/<name>/` + **wire** + wire dosyasında **`go fmt`** (`--skip-wire`, `--module-root`, `--out`, `--dry-run`) |
| `zatrano gen crud <name>` | CRUD + **form request struct'ları** (`requests/`) + **`RegisterCRUD`** wire + **`go fmt`** (aynı bayraklar) |
| `zatrano gen request <name>` | Yalnızca form request struct'ları üret (`modules/<name>/requests/create_*.go`, `update_*.go`) |
| `zatrano gen policy <name>` | Yetkilendirme policy stub'ı üret (`modules/<name>/policies/<name>_policy.go`) — `auth.Policy` arayüzünü CRUD metotlarıyla implemente eder |
| `zatrano gen job <name>` | Kuyruk job stub'ı üret (`modules/jobs/<name>.go`) — `queue.Job` arayüzünü Handle, Retries, Timeout ile implemente eder |
| `zatrano gen mail <name>` | Mailable struct + HTML şablon üret (`modules/mails/<name>_mail.go` + `views/mails/<name>.html`) |
| `zatrano gen wire <name>` | Sadece wire (dosya üretmez); `register.go` / `crud_register.go` varlığına göre (`--register-only`, `--crud-only`) |
| `zatrano openapi validate` | Tek dosya veya **`--merged`** (canlı `/openapi.yaml` ile aynı; `--base`, isteğe konumsal argüman) |
| `zatrano openapi export` | Birleşik YAML yaz (`--base`, `--output` veya `-` stdout) |
| `zatrano jwt sign` | Test JWT üret (`--sub`, `--secret`, config bayrakları) |
| `zatrano cache clear` | Önbelleği temizle veya belirli tag'leri sil (`--tag`) |
| `zatrano queue work` | Kuyruk worker süreci başlat (`--queue`, `--tries`, `--timeout`, `--sleep`) |
| `zatrano queue failed` | Başarısız jobları listele |
| `zatrano queue retry [id]` | Başarısız jobı yeniden gönder veya `--all` |
| `zatrano queue flush` | Tüm başarısız job kayıtlarını sil |
| `zatrano mail preview [name]` | E-posta şablonunu tarayıcıda önizle (`--port`, `--layout`) |
| `zatrano completion …` | Kabuk tamamlama |
| `zatrano verify` | **`go vet` + `go test` + birleşik OpenAPI** (PR/CI; yarış için **`--race`**; `--no-vet`, `--no-test`, `--no-openapi`, `--module-root`) |
| `zatrano version` | Sürüm (ayrıca **`zatrano --version`**) |

**Windows / boşluklu yol:** `--replace-zatrano` ile framework kökünü verin; gerekirse `go.mod` içinde yol tırnaklanır.

---

## HTTP (şimdilik)

| Metot | Yol | Not |
|-------|-----|-----|
| GET | `/` | JSON özet (`env`, `endpoints`, `http` CORS/rate-limit bayrakları, `error_includes_request_id`) |
| GET | `/health`, `/ready`, `/status` | Canlılık / hazırlık / özet (`/status` içinde `env`) |
| GET | `/openapi.yaml`, `/docs` | **Birleşik** OpenAPI (`/` ve `/status` için JSON şema) + Scalar |
| GET | `/api/v1/public/ping` | Herkese açık |
| GET | `/api/v1/private/me` | `jwt_secret` varsa **Bearer JWT** |
| POST | `/api/v1/auth/token` | Yalnızca `security.demo_token_endpoint: true` ve **`env: prod` değil** |
| GET | `/auth/oauth/google/login`, `/auth/oauth/github/login` | OAuth2 başlatır (`oauth.enabled` + anahtarlar gerekli) |
| GET | `/auth/oauth/google/callback`, `/auth/oauth/github/callback` | OAuth yönlendirme |

**Oturum + CSRF:** `redis_url` ve `security` uygunsa açılır. CSRF, `Bearer`, `csrf_skip_prefixes` (varsayılan `/api/`) ve **`/auth/oauth/`** için atlanır.

**OAuth2:** `oauth.enabled`, `oauth.base_url`, `oauth.providers.google` / `github` ayarlayın. Sağlayıcı konsolunda yönlendirme: `{base_url}/auth/oauth/google/callback` (GitHub için aynı kalıp). Oturum alanları: `oauth_provider`, `oauth_subject`, `oauth_name`, `oauth_email`.

**Hatalar:** JSON gövdesi `{ "error": { "code", "message", "request_id"? } }`. `request_id`, **`X-Request-ID`** başlığıyla aynıdır (log ve destek için).

**HTTP ara katmanı (`http` YAML / `HTTP_*` env):**

- **CORS** — `http.cors_enabled`, `cors_allow_origins`, `cors_allow_methods`, `cors_allow_headers`, `cors_expose_headers`, `cors_allow_credentials`, `cors_max_age`. Varsayılan **kapalı**. **`cors_allow_credentials: true`** ile köken **`*`** birlikte kullanılamaz (doğrulama hata verir).
- **Rate limit** — `rate_limit_enabled`, `rate_limit_max`, `rate_limit_window`, isteğe **`rate_limit_redis: true`** (`redis_url` gerekir). Aksi halde süreç başına **bellek içi**. Limit **altındaki** yanıtlarda **`X-RateLimit-*`** vardır. Limit aşımında **429** + aynı `error` JSON + **`Retry-After`** (RFC 6585).
- **İstek süresi** — `request_timeout` (ör. `60s`): Fiber **timeout**; aşımda **408** JSON.
- **Gövde boyutu** — `body_limit` bayt (`0` = Fiber varsayılanı **4 MiB**).

Yığın sırası: **recover → request-id → i18n (açıksa) → CORS → timeout → rate limit → helmet → compress → oturum/CSRF → rotalar**.

---

## Validation (Doğrulama)

ZATRANO, [`go-playground/validator/v10`](https://pkg.go.dev/github.com/go-playground/validator/v10) kütüphanesini sarmalayan **generic, struct-tag tabanlı bir doğrulama sistemi** sunar. Otomatik **422 JSON yanıtları** ve **i18n çeviri desteği** içerir.

### Temel Kullanım

Ana API **`zatrano.Validate[T](c)`** — tek bir generic çağrı ile istek gövdesi parse edilir, struct tag'leri doğrulanır ve hata durumunda yapılandırılmış 422 yanıtı döner:

```go
import "github.com/zatrano/framework/pkg/zatrano"

func (h *ProductHandler) Create(c fiber.Ctx) error {
    req, err := zatrano.Validate[CreateProductRequest](c)
    if err != nil {
        return err // 422 JSON yanıtı zaten gönderildi
    }
    // req geçerli — kullanabilirsiniz
    return h.svc.Create(c.Context(), req.Name, req.Email)
}
```

### Form Request Struct'ları

İstek yapılarınızı `json` ve `validate` tag'leri ile tanımlayın:

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

### Request Struct'larını Üretme

CLI ile request stub'larını otomatik oluşturabilirsiniz:

```bash
# Sadece request struct'larını üret
zatrano gen request product
# → modules/product/requests/create_product.go
# → modules/product/requests/update_product.go

# gen crud komutu request struct'larını da otomatik üretir
zatrano gen crud product
# → modules/product/crud_handlers.go      (zatrano.Validate[T] kullanır)
# → modules/product/crud_register.go
# → modules/product/requests/create_product.go
# → modules/product/requests/update_product.go
```

### 422 Hata Yanıt Formatı

Doğrulama başarısız olduğunda, tutarlı bir JSON yapısı döner:

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
        "value": "geçersiz-email",
        "message": "Must be a valid email address"
      }
    ]
  }
}
```

i18n açık ve istek dili `tr` olduğunda, mesajlar otomatik çevrilir:

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
        "value": "geçersiz-email",
        "message": "Geçerli bir e-posta adresi olmalıdır"
      }
    ]
  }
}
```

### i18n Doğrulama Mesajları

Doğrulama mesajları, locale dosyalarında `validation.*` anahtar alanı altında tutulur:

```json
// locales/tr.json
{
  "validation": {
    "required": "Bu alan zorunludur",
    "email": "Geçerli bir e-posta adresi olmalıdır",
    "min": "En az {{.Param}} karakter olmalıdır",
    "max": "En fazla {{.Param}} karakter olmalıdır"
  }
}
```

`{{.Param}}` yer tutucusu kısıtlamanın değeri ile değiştirilir (ör. `min=5` → `"En az 5 karakter olmalıdır"`).

**Hazır çevrilmiş tag'ler:** `required`, `email`, `min`, `max`, `gte`, `lte`, `gt`, `lt`, `len`, `url`, `uri`, `uuid`, `oneof`, `numeric`, `number`, `alpha`, `alphanum`, `boolean`, `contains`, `excludes`, `startswith`, `endswith`, `ip`, `ipv4`, `ipv6`, `datetime`, `json`, `jwt`, `eqfield`, `nefield`.

### Özel Doğrulama Kuralları

İsteğe bağlı i18n desteği ile özel doğrulama tag'leri kaydedin:

```go
import (
    "github.com/go-playground/validator/v10"
    "github.com/zatrano/framework/pkg/zatrano"
)

// Özel kural kaydet
zatrano.RegisterRule("tc_no", func(fl validator.FieldLevel) bool {
    v := fl.Field().String()
    if len(v) != 11 {
        return false
    }
    // ... TC kimlik numarası algoritması
    return true
})

// i18n mesaj anahtarı ile (locale dosyalarınıza "validation.tc_no" ekleyin)
zatrano.RegisterRuleWithMessage("tc_no", tcNoValidator, "validation.tc_no")
```

Sonra struct tag'lerinde kullanın:

```go
type VatandasRequest struct {
    TCNO string `json:"tc_no" validate:"required,tc_no"`
}
```

### Doğrudan Engine Erişimi

İleri düzey kullanım için alttaki validator engine'e erişebilirsiniz:

```go
import "github.com/zatrano/framework/pkg/validation"

engine := validation.Default()
engine.Validator() // go-playground/validator'dan *validator.Validate

// Herhangi bir struct'ı programatik olarak doğrulayın (Fiber context'i olmadan)
if verr := engine.ValidateStruct(myStruct, "tr"); verr != nil {
    for _, fe := range verr.Errors {
        fmt.Printf("%s: %s\n", fe.Field, fe.Message)
    }
}
```

---

## Yetkilendirme (RBAC & Gate/Policy)

ZATRANO, iki tamamlayıcı katmanlı **eksiksiz bir yetkilendirme sistemi** sunar: izin kontrolleri için **RBAC** (rol tabanlı, DB destekli) ve kaynak düzeyinde ince taneli yetkilendirme için **Gate/Policy** (kaynak bazlı). Her ikisi de yerelleştirilmiş 403 hata mesajları için **i18n** sistemiyle entegredir.

### RBAC — Rol Tabanlı Erişim Kontrolü

Roller ve izinler veritabanında saklanır (`roles`, `permissions`, `role_permissions`, `zatrano_user_roles` tabloları). Yoğun yolda DB çağrılarından kaçınmak için bellek içi önbellek kullanılır. `RBACManager`, bootstrap sırasında otomatik olarak başlatılır (DB varken) ve `app.RBAC` ile erişilebilir.

```go
import "github.com/zatrano/framework/pkg/auth"

// Rol ve izin oluşturma
rbac := app.RBAC
rbac.CreateRole(ctx, "admin", "Yönetici")
rbac.CreateRole(ctx, "editor", "İçerik editörü")
rbac.CreatePermission(ctx, "posts.create", "Yazı oluştur")
rbac.CreatePermission(ctx, "posts.update", "Yazı güncelle")
rbac.CreatePermission(ctx, "posts.delete", "Yazı sil")

// Rollere izin atama
rbac.AssignPermissions(ctx, "admin", "posts.create", "posts.update", "posts.delete")
rbac.AssignPermissions(ctx, "editor", "posts.create", "posts.update")

// Kullanıcıya rol atama
rbac.AssignRoleToUser(ctx, userID, "editor")

// İzin kontrolü
ok, _ := rbac.UserHasPermission(ctx, userID, "posts.create") // true
ok, _ = rbac.UserHasPermission(ctx, userID, "posts.delete")  // false (editör silemez)
```

**Veritabanı migration:** `zatrano db migrate` çalıştırın — `000002_zatrano_rbac` migration'ı dört tabloyu uygun indeks ve foreign key'lerle oluşturur.

### Gate / Policy — Kaynak Bazlı Yetkilendirme

`Gate` sistemi (`app.Gate` ile erişilir) belirli eylemler için yetkilendirme kontrolleri tanımlamaya olanak sağlar. Anlık kontroller için `Define`, yapılandırılmış CRUD policy'leri için `RegisterPolicy` kullanın.

```go
import "github.com/zatrano/framework/pkg/auth"

// Anlık gate tanımı
gate := app.Gate
gate.Define("edit-post", func(c fiber.Ctx, resource any) bool {
    post := resource.(*Post)
    userID, _ := c.Locals(middleware.LocalsUserID).(uint)
    return post.AuthorID == userID
})

// Süper-admin bypass (her gate kontrolünden önce çalışır)
gate.Before(func(c fiber.Ctx, ability string, resource any) *bool {
    roles, _ := c.Locals(middleware.LocalsUserRoles).([]string)
    for _, r := range roles {
        if r == "super-admin" { t := true; return &t }
    }
    return nil // gate tanımına düş
})

// Handler'larda:
if err := gate.Authorize(c, "edit-post", post); err != nil {
    return err // 403 Forbidden
}
```

**Policy arayüzü:** yapılandırılmış CRUD yetkilendirmesi için `auth.Policy` arayüzünü implemente edin. `zatrano gen policy` ile stub üretin:

```bash
zatrano gen policy post
# → modules/post/policies/post_policy.go
```

Üretilen policy 7 metot içerir: `ViewAny`, `View`, `Create`, `Update`, `Delete`, `ForceDelete`, `Restore`. Gate'e kaydedin:

```go
import "myapp/modules/post/policies"

gate.RegisterPolicy("post", &policies.PostPolicy{})
// Oluşturur: "post.viewAny", "post.view", "post.create", "post.update",
//            "post.delete", "post.forceDelete", "post.restore"
```

### Route Seviyesi Yetkilendirme Middleware

`pkg/middleware` paketi, rota seviyesinde izin ve rol kontrolü için kullanıma hazır middleware sağlar. Tümü **403 JSON** + **i18n destekli** hata mesajı döner.

| Middleware | Açıklama |
|---|---|
| `middleware.Can(rbac, "perm")` | Kullanıcının belirli bir izne sahip olmasını gerektirir |
| `middleware.CanAny(rbac, "p1", "p2")` | Kullanıcı listelenen izinlerden **herhangi birine** sahipse geçer |
| `middleware.CanAll(rbac, "p1", "p2")` | Kullanıcı **tüm** listelenen izinlere sahipse geçer |
| `middleware.HasRole("admin")` | Belirli bir rol gerektirir |
| `middleware.HasAnyRole("admin", "editor")` | Kullanıcı listelenen rollerden **herhangi birine** sahipse geçer |
| `middleware.GateAllows(gate, "ability")` | Gate ability kontrolü (kaynak olmadan) |
| `middleware.InjectRoles(rbac)` | Kullanıcı rollerini Locals'a yükler (auth middleware'den sonra yerleştirin) |

```go
import "github.com/zatrano/framework/pkg/middleware"

// Kimlik doğrulama middleware'inden sonra:
app.Use(security.JWTMiddleware(cfg))
app.Use(middleware.InjectRoles(rbac))  // rolleri context'e yükler

// İzin bazlı
app.Get("/admin/users", middleware.Can(rbac, "users.view"), usersHandler)
app.Post("/posts", middleware.Can(rbac, "posts.create"), createPostHandler)
app.Delete("/system", middleware.CanAll(rbac, "system.admin", "system.delete"), handler)

// Rol bazlı
app.Get("/dashboard", middleware.HasAnyRole("admin", "editor"), dashHandler)

// Gate bazlı
app.Get("/posts", middleware.GateAllows(gate, "post.viewAny"), listPostsHandler)
```

### 403 Hata Yanıt Formatı

Yetkilendirme başarısız olduğunda, standart JSON hata yapısı döner:

```json
{
  "error": {
    "code": 403,
    "message": "You do not have permission to perform this action.",
    "permission": "posts.delete"
  }
}
```

i18n açık ve istek dili `tr` olduğunda:

```json
{
  "error": {
    "code": 403,
    "message": "Bu işlemi gerçekleştirme yetkiniz bulunmamaktadır.",
    "permission": "posts.delete"
  }
}
```

### i18n Yetkilendirme Mesajları

Yetkilendirme mesajları, locale dosyalarında `auth.*` anahtar alanı altında tutulur:

```json
{
  "auth": {
    "forbidden": "Bu işlemi gerçekleştirme yetkiniz bulunmamaktadır.",
    "unauthorized": "Bu kaynağa erişmek için kimlik doğrulaması gereklidir.",
    "role_required": "Bu kaynağa erişmek için gerekli role sahip değilsiniz.",
    "permission_required": "Gerekli izne sahip değilsiniz: {{.Permission}}."
  }
}
```

---

## Cache Sistemi (Önbellek)

ZATRANO, **Bellek İçi (In-Memory)** ve **Redis** sürücüleri için ortak bir API sunan **güçlü bir önbellek katmanı** sağlar. `Remember`, JSON serileştirme, tag tabanlı geçersiz kılma ve response middleware gibi gelişmiş özellikleri destekler.

### Sürücüler (Drivers)

Sistem, yapılandırmanıza göre en iyi sürücüyü otomatik olarak seçer:
- **Redis:** `redis_url` ayarlandığında tercih edilir. Dağıtık ortamlar ve tag desteği için uygundur.
- **Memory:** Yerel geliştirme veya tek node'lu yapılar için geri dönüş (fallback) seçeneğidir. Hızlıdır ancak geçicidir.

### Temel Kullanım

Önbellek yöneticisine `app.Cache` üzerinden erişebilirsiniz:

```go
import "context"

ctx := context.Background()

// Basit veri saklama
app.Cache.Set(ctx, "anahtar", "değer", 10 * time.Minute)

// Veri okuma
val, ok := app.Cache.Get(ctx, "anahtar")

// Otomatik JSON işleme
type Kullanici struct { Ad string }
app.Cache.SetJSON(ctx, "user:1", Kullanici{Ad: "Deniz"}, time.Hour)

var user Kullanici
ok, err := app.Cache.GetJSON(ctx, "user:1", &user)
```

### Gelişmiş Kullanım

#### `Remember` ve `RememberJSON`

Laravel stili popüler desen: Veri önbellekte varsa döner, yoksa verilen fonksiyonu çalıştırıp sonucu önbelleğe yazar ve döner.

```go
// Veri yoksa DB'den çek ve önbelleğe yaz
var users []User
err := app.Cache.RememberJSON(ctx, "users:all", 30*time.Minute, &users, func() (any, error) {
    return db.FindAllUsers(ctx)
})
```

#### Tags (Sadece Redis)

İlişkili anahtarları tag'ler altında gruplayarak toplu silme yapmanızı sağlar.

```go
// Tag ile saklama
app.Cache.Tags("users").Set(ctx, "users:1", data, time.Hour)

// Bir tag'e ait tüm anahtarları temizleme
app.Cache.Tags("users").Flush(ctx)
```

### Middleware (Ara Katman)

Bir rotanın tüm HTTP yanıtını sunucu tarafında önbelleğe alabilirsiniz.

```go
import "github.com/zatrano/framework/pkg/middleware"

// 5 dakika boyunca önbelleğe al
app.Get("/api/v1/stats", middleware.Cache(app.Cache, 5*time.Minute), handler)

// Tag desteği ile
app.Get("/api/v1/users", middleware.CacheWithConfig(app.Cache, middleware.CacheConfig{
    TTL:  10 * time.Minute,
    Tags: []string{"users"},
}), handler)
```

### CLI Komutları

Terminal üzerinden önbelleği temizleyin:

```bash
# Tüm önbelleği sil
zatrano cache clear

# Sadece belirli tag'leri sil
zatrano cache clear --tag users --tag posts
```

---

## Kuyruk / Job Sistemi

ZATRANO, geciktirilmiş zamanlama, otomatik yeniden deneme ve üssel geri çekilme (exponential backoff) ve başarısız job’ların PostgreSQL’de saklanmasıyla **Redis tabanlı bir arkaplan job kuyruğu** sunar.

### Job Tanımlama

`queue.Job` arayüzünü implemente edin veya varsayılan değerler için `queue.BaseJob` gömün:

```go
package jobs

import (
    "context"
    "time"
    "github.com/zatrano/framework/pkg/queue"
)

type EpostaGonderJob struct {
    queue.BaseJob
    Kime    string `json:"kime"`
    Konu    string `json:"konu"`
    Icerik  string `json:"icerik"`
}

func (j *EpostaGonderJob) Name() string            { return "eposta_gonder" }
func (j *EpostaGonderJob) Queue() string           { return "epostalar" }
func (j *EpostaGonderJob) Retries() int            { return 5 }
func (j *EpostaGonderJob) Timeout() time.Duration  { return 30 * time.Second }

func (j *EpostaGonderJob) Handle(ctx context.Context) error {
    // e-postayı gönder...
    return mailer.Send(ctx, j.Kime, j.Konu, j.Icerik)
}
```

CLI ile job stub'ı üretin:

```bash
zatrano gen job eposta_gonder
# → modules/jobs/eposta_gonder.go
```

### Job Gönderme (Dispatch)

```go
// Uygulama başlangıcında job türlerini kaydedin
app.Queue.Register("eposta_gonder", func() queue.Job { return &jobs.EpostaGonderJob{} })

// Hemen gönder
app.Queue.Dispatch(ctx, &jobs.EpostaGonderJob{
    Kime:   "kullanici@example.com",
    Konu:   "Hoş geldiniz!",
    Icerik: "Merhaba dünya",
})

// Gecikmeli gönder (Redis ZADD sorted set)
app.Queue.Later(ctx, 5*time.Minute, &jobs.EpostaGonderJob{
    Kime: "kullanici@example.com",
    Konu: "Takip",
})
```

### Worker Süreci

Kuyruktan jobları işleyen uzun çalışan bir worker başlatın:

```bash
zatrano queue work
zatrano queue work --queue epostalar --queue bildirimler
zatrano queue work --tries 5 --timeout 120s --sleep 5s
```

Worker otomatik olarak:
- Redis BRPOP ile FIFO sırasında polllar
- Geciktirilmiş jobları her saniye taşır (ZADD → LPUSH)
- Başarısız jobları **üssel geri çekilme** ile yeniden dener (2^deneme saniye)
- Kalıcı olarak başarısız jobları `zatrano_failed_jobs` PostgreSQL tablosuna kaydeder
- `Handle()` içindeki panic’lerden kurtulur
- SIGINT/SIGTERM ile düzgün kapanır

### Başarısız Joblar

Maksimum yeniden deneme sayısını aşan joblar, hata mesajı, stack trace ve orijinal payload ile PostgreSQL’e kaydedilir.

```bash
# Başarısız jobları listele
zatrano queue failed

# Belirli bir başarısız jobı yeniden dene
zatrano queue retry 42

# Tüm başarısız jobları yeniden dene
zatrano queue retry --all

# Tüm başarısız job kayıtlarını sil
zatrano queue flush
```

**Veritabanı migration:** `zatrano db migrate` çalıştırın — `000003_zatrano_failed_jobs` migration’ı gerekli tabloyu oluşturur.

### Kuyruk Mimarisi

| Bileşen | Redis Yapısı | Amaç |
|---|---|---|
| Hazır kuyruk | `LIST` (LPUSH/BRPOP) | FIFO job işleme |
| Geciktirilmiş joblar | `SORTED SET` (ZADD) | Zaman bazlı zamanlama |
| Başarısız joblar | PostgreSQL tablosu | Kalıcı hata kayıtları |

---

## Mail Sistemi (E-posta)

ZATRANO, HTML şablon desteği, asenkron gönderim için kuyruk entegrasyonu, ek dosya desteği ve yeniden kullanılabilir e-posta tanımları için Mailable deseni sunan **çok sürücülü bir mail sistemi** sağlar.

### Yapılandırma

```yaml
# config/dev.yaml
mail:
  driver: smtp          # smtp | log (log = geliştirme/test)
  from_name: "Uygulamam"
  from_email: "noreply@uygulamam.com"
  templates_dir: "views/mails"
  smtp:
    host: smtp.example.com
    port: 587
    username: kullanici
    password: sifre
    encryption: tls     # tls | starttls | ""
```

### E-posta Gönderme

```go
import "github.com/zatrano/framework/pkg/mail"

// Basit mesaj
app.Mail.Send(ctx, &mail.Message{
    To:      []mail.Address{{Email: "kullanici@example.com", Name: "Deniz"}},
    Subject: "Hoş Geldiniz!",
    HTMLBody: "<h1>Merhaba Deniz!</h1>",
})

// Şablon ile
app.Mail.SendTemplate(ctx,
    []mail.Address{{Email: "kullanici@example.com"}},
    "Uygulamamıza Hoş Geldiniz",
    "welcome",    // views/mails/welcome.html
    "default",    // views/mails/layouts/default.html
    map[string]any{"Name": "Deniz"},
)

// Kuyruk ile asenkron
app.Mail.Queue(ctx, &mail.Message{
    To:      []mail.Address{{Email: "kullanici@example.com"}},
    Subject: "Bülten",
    HTMLBody: body,
})
```

### Mailable Deseni

Yapılandırılmış, yeniden kullanılabilir e-posta tanımları üretin:

```bash
zatrano gen mail hosgeldiniz
# → modules/mails/hosgeldiniz_mail.go
# → views/mails/hosgeldiniz.html
```

```go
type HosgeldinizMail struct {
    Ad    string
    Email string
}

func (m *HosgeldinizMail) Build(b *mail.MessageBuilder) error {
    b.To(m.Ad, m.Email).
        Subject("Hoş Geldiniz!").
        View("hosgeldiniz", "default", map[string]any{"Name": m.Ad}).
        AttachData("rehber.pdf", pdfBytes, "application/pdf")
    return nil
}

// Senkron gönder
app.Mail.SendMailable(ctx, &mails.HosgeldinizMail{Ad: "Deniz", Email: "deniz@example.com"})

// Kuyruk ile asenkron
app.Mail.QueueMailable(ctx, &mails.HosgeldinizMail{Ad: "Deniz", Email: "deniz@example.com"})
```

### Ek Dosya Desteği

```go
msg := &mail.Message{
    To:      []mail.Address{{Email: "kullanici@example.com"}},
    Subject: "Fatura",
    HTMLBody: body,
    Attachments: []mail.Attachment{
        {Filename: "fatura.pdf", Content: pdfBytes},
        {Filename: "logo.png", Content: logoBytes, Inline: true},
    },
}
app.Mail.Send(ctx, msg)
```

### Şablon Önizleme

Geliştirme sırasında e-posta şablonlarını tarayıcıda önizleyin:

```bash
zatrano mail preview              # şablonları listele
zatrano mail preview welcome      # welcome şablonunu önizle
zatrano mail preview welcome --port 3001
```

Tam yerel mail testi için **Mailpit** veya **MailHog**'u SMTP sunucusu olarak kullanın.

---

### Uluslararasılaştırma (`i18n`)

Uygulama metinleri **`locales_dir`** altında **JSON** dosyalarında tutulur: **`{locales_dir}/{etiket}.json`** (ör. `locales/tr.json`). İç içe nesneler **nokta anahtarlara** düzleştirilir (`app.welcome`).

- **Yapılandırma:** `i18n.enabled`, `i18n.default_locale`, `i18n.supported_locales`, `i18n.locales_dir`, isteğe `i18n.cookie_name` (varsayılan `zatrano_lang`), `i18n.query_key` (varsayılan `lang`). **`i18n.enabled: true`** iken **`locales_dir`** dizini diskte olmalıdır (yüklemede doğrulanır).
- **Çözüm sırası:** sorgu (`?lang=`), çerez, **`Accept-Language`**, ardından **`default_locale`**.
- **Handler:** `github.com/zatrano/zatrano/pkg/i18n` — sabit metin için **`i18n.T(c, "app.welcome")`**; değişkenli çeviri için **`i18n.Tf(c, "app.hello_user", map[string]any{"Name": ad})`** veya struct; JSON içinde **`{{.Name}}`** gibi **`text/template`** ifadeleri. **`map`** ile basit `{{.Alan}}` otomatik uyumludur; Fiber dışında **`Bundle.Format`**. i18n kapalıyken **`T`** / **`Tf`** anahtarı döner (**`Tf`** hata **`nil`**).
- **GET /** yanıtında **`i18n`** nesnesi (`enabled`; açıksa `default_locale`, `supported_locales`, **`active_locale`**).
- **Doğrulama mesajları**, i18n açık olduğunda `validation.*` anahtarlarından otomatik olarak çözülür (bkz. [Validation](#validation-doğrulama)).

---

## Yapılandırma

- **`.env`**, **`config/{env}.yaml`**, **ortam değişkenleri** (ör. `SECURITY_JWT_SECRET`). Çoklu köken veya **`supported_locales`** gibi **listeler** için **YAML** tercih edin.
- Ayrıntı: `migrations_dir`, `seeds_dir`, `openapi_path`, **`http.*`**, **`i18n.*`**, `security.*`, `oauth.*` — `config/examples/dev.yaml`.
- Hata ayıklama: **`zatrano config print`** (tam, maskeli) veya **`zatrano config print --paths-only`** (sohbete yapıştırmaya uygun özet).
- CI: önce **`zatrano config validate -q`** (hızlı YAML/ortam kontrolü), sonra **`zatrano openapi validate --merged`**, veya tam kapı için **`zatrano verify`** (Geliştirme bölümüne bakın).

---

## Geliştirme

```bash
go test ./... -count=1
go fmt ./...
go vet ./...
golangci-lint run
```

**Tek komut kontrol:** `zatrano verify` (veya POSIX **`make verify`**) — `vet`, `test`, birleşik OpenAPI. Yayın öncesi için **`make verify-race`** / **`zatrano verify --race`**. **`make config-validate`**, **`zatrano config validate`** ile aynıdır.

**Canlı yenileme:** [Air](https://github.com/air-verse/air) kurun, `air` çalıştırın (`.air.toml`). Windows'ta çıktı `./tmp/main.exe`.

**Birleşik OpenAPI dosyası:** `make openapi-export` veya `go run ./cmd/zatrano openapi export --output api/openapi.merged.yaml`.

**Ortam kontrolü:** `zatrano doctor` yapılandırma özeti, **`http`** ara katmanı (CORS, rate limit, timeout, gövde boyutu) ve **`config print --paths-only`** ipucu, açıksa **OAuth**, yedekleme için **`pg_dump` / `pg_restore` / `psql`** PATH bilgisi ve bağlantı testlerini gösterir.

**Kod üret:** `gen module` / `gen crud` wire dosyasını günceller ve **`go fmt`** çalıştırır. **`gen wire`** yalnızca patch (ör. **`--skip-wire`** sonrası). **`gen request`** bağımsız olarak form request struct stubs üretir. **`gen policy`** CRUD metotlarıyla (ViewAny, View, Create, Update, Delete, ForceDelete, Restore) `auth.Policy` implementasyonu üretir. Uygulamalarda **`internal/routes/register.go`**, framework deposunda **`pkg/server/register_modules.go`**.

**Sunucu gömme:** `server.Mount(..., server.MountOptions{RegisterRoutes: …})`; `zatrano.StartOptions.RegisterRoutes` üretilen uygulamalarda bu çağrıyı iletir.

---

## Dokümantasyon

- **İngilizce:** [`README.md`](README.md)
- **Türkçe:** bu dosya (`README.tr.md`)

İki dosyayı da aynı değişiklikte güncelleyin.

---

## Katkı

Öneri ve PR'lar memnuniyetle karşılanır. Davranış veya CLI değişikliklerinde **her iki** README'yi de güncelleyin.

---

## Lisans

Belirlenecek.
