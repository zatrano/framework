# ZATRANO V2 Stage A — Faz 1 denetim

Tarih: 2026-09-03  
Dal: `v2-stage-a` (base: `main` @ `90faf57`, VERSION `1.6.6`)  
Kapsam: kod değişikliği yok; bağlam iddialarının dosya doğrulaması.

## 1. `Application` accessor’ları

`core/application.go` + `core/services.go` + `core/http_bridge.go` doğrulandı.

### Private field listesi (`Application` struct)

| Field | Somut tip |
| --- | --- |
| `basePath` | `string` |
| `container` | `*container.Container` |
| `config` | `*config.Repository` |
| `router` | `*routing.Router` |
| `logger` | `*log.Logger` |
| `rateLimiter` | `*ratelimit.Limiter` |
| `ctx` | `*appcontext.Store` |
| `urls` | `*urlgen.Generator` |
| `encrypter` | `*encryption.Encrypter` |
| `hasher` | `*hashing.Manager` |
| `metrics` | `*observability.Metrics` |
| `health` | `*health.Manager` |
| `maintenance` | `*maintenance.Manager` |
| `exceptions` | `*exceptions.Handler` |
| `reports` | `*report.Manager` |
| `httpBridge` | `HTTPBridge` (interface) |
| `migrations` | `any` |
| `seeders` | `any` |
| `providers` | `[]Provider` |
| `booted` | `bool` |
| `environment` | `string` |

### Public accessor envanteri

Bağlamdaki 24 accessor mevcut. Ek (bağlamda yok): `Version() string`.

| Accessor | Dönüş | Faz 2 notu |
| --- | --- | --- |
| `Container()` | `*container.Container` | interface; kullanılan metod: `Instance` |
| `Config()` | `*config.Repository` | `Get`, `GetString`, `GetInt` |
| `Router()` | `*routing.Router` | `Get`, `Post`, `Use`, `Name`, `Group`, `Routes`, `Snapshot`, `SaveCache` |
| `Logger()` | `*log.Logger` | `Debugf`, `Infof`, `Errorf` |
| `RateLimiter()` | `*ratelimit.Limiter` | **hiç çağrı sitesi yok** |
| `Context()` | `*appcontext.Store` | `Put` |
| `URL()` | `*urlgen.Generator` | `Signed`, `To`, `Route` |
| `Encrypter()` | `*encryption.Encrypter` | `Encrypt`, `Decrypt` |
| `Hash()` | `*hashing.Manager` | `Make` |
| `Metrics()` | `*observability.Metrics` | `Snapshot`; kernel `Timing` private field’a çekilecek |
| `Health()` | `*health.Manager` | `Handler`, `Custom`, `Database` |
| `Maintenance()` | `*maintenance.Manager` | `Enable`, `Disable`; `Middleware` kernel’de private field |
| `Exceptions()` | `*exceptions.Handler` | `Middleware` + nil check |
| `Reports()` | `*report.Manager` | **hiç çağrı sitesi yok** |
| `HTTPBridge()` | `HTTPBridge` | zaten interface — dokunulmayacak |
| `Make` / `Bound` | primitive | dokunulmayacak |
| `Environment` / `IsProduction` / `IsDebug` | primitive | dokunulmayacak |
| `SetMigrations` / `Migrations` / `SetSeeders` / `Seeders` | `any` | dokunulmayacak |
| `BasePath` | `string` | dokunulmayacak |
| `RegisterProviders` | `...Provider` | dokunulmayacak |

`Provider` arayüzü zaten var (`Register`/`Boot` ile `*Application`).

## 2. Kernel `Bootstrap()` addon config

`core/application.go` `Bootstrap()` (satır 193–201) doğrulandı:

```
database, auth, notifications, oauth, mongo, webauthn, billing, ai, social
```

Bağlam 8 addon config saymıştı; **`database` da kernel’den yükleniyor** (foundation, ama `appconfig.` import’u). Faz 3 `grep appconfig. core/` sıfır şartı için `Database()` da kernel’den çıkmalı.

`config/app.go` Bootstrap’ta **çağrılmıyor**; app anahtarı inline yükleniyor (`name/env/debug/url/key/locale/fallback` — `port` ve `cors` yok). Bu mevcut davranış; sessizce doldurulmayacak.

`config/session.go` mevcut ama Bootstrap’ta `Load` edilmiyor.

## 3. Addon registry

`bootstrap/addons/providers.go` (~625 satır) 27 servis paketini statik import ediyor.  
`bootstrap/addons/registry.go` `Meta{Name, Key, Description, Heavy, Factory}` tablosu — 27 kayıt, bağlamla örtüşüyor.

**Nested module istisnası (Faz 4):** `packages/mongo` ve `packages/webauthn` ayrı `go.mod` (root `replace`). Bu paketler root modülü (`bootstrap/addons`) import edemez. Self-registration `init()` 25 root-modül addon’da paketin içinde; mongo/webauthn için kök modülde (addons paketi dışında) kayıt.

## 4. Boot profilleri

`bootstrap/app.go`: `App`, `APIApp`, `WebApp`, `DemoApp`, `MinimalApp`, `CoreApp` + bağlamda olmayan `KernelApp` alias.

Fark: foundation seti + addon listesi (`EnabledAddons` / `PresetAPI` / `PresetWeb` / `DemoAddons` / boş / `KernelProviders`).

`APP_BOOT`: `bootstrap/boot_env.go` — `app|api|web|minimal|core|kernel|demo`.

## 5. Catalog taksonomi

`core/catalog.go` üç katman: `LayerKernel` (13), `LayerFoundation` (30), `LayerAddon`.

| | Bağlam | Gerçek |
| --- | --- | --- |
| Kernel | 13 | 13 (doğru) |
| Foundation | 30 | 30 (doğru) |
| Addon servis | 27 | 27 (doğru) |
| Addon kütüphane | 36 | **33** |
| Addon toplam | 63 | **60** |

Kütüphane sapması: bağlam 36 varsaymış; katalogda 33 `KindLibrary` var (`agent`…`websocket`). `ai`/`rag`/`agent` şu an `LayerAddon`.

## 6. Routes

`routes/{web,api,health}.go` — `routes.Web/API/Health(app)` `app/providers/route_service_provider.go` `Boot()` içinde elle çağrılıyor. Doğrulandı.

## 7. `HTTPBridge`

`core/http_bridge.go` — `Middleware()` / `Finalize(...)`. Foundation `bootstrap/foundation/http_bridge.go` `installHTTPBridge` ile kuruyor. Faz 2–3 için desen örneği.

## 8. Faz 2 tasarım kararları (denetimden)

1. `contracts` interface imzaları somut paket tiplerini kullanır (`*routing.Route`, `routing.HandlerFunc`) — `HTTPBridge` ile aynı; `*Router` zaten interface’i implement eder.
2. Pointer accessor’lar typed-nil koruması ile döner (`if p == nil { return nil }`).
3. `routing.Controller` `*Router` yerine yerel bir interface alacak (contracts↔routing çevrimi yok).
4. `broadcasting.NewLogBroadcaster` / `notification.NewLogMailer` / `pulse.New` somut `*Logger`/`*Metrics` yerine dar interface alacak — davranış aynı, `app.Logger()`/`app.Metrics()` derlenir.
5. Kernel içi `Timing`/`Maintenance.Middleware`/`Exceptions.Middleware` private field kullanacak.
6. `RateLimiter()` / `Reports()` çağrı sitesi yok; named boş-olmayan sözleşme için boot’ta instance üzerinde fiilen çağrılan metodlar alınacak (`For` / `Recent`+`Reporter` gerekçesi Faz 2 notunda).

## 9. Faz 4–6 ön notları

- `packages/ai` içeriğine dokunulmayacak; yalnızca self-registration için yeni `provider.go`/`register.go` eklenecek (mevcut `AIServiceProvider` taşınması). `rag`/`agent` kütüphane — registry’de yok, `init()` yok.
- Fiziksel repo bölünmesi yok; import path `github.com/zatrano/framework` kalır.
- `core` → `kernel` Faz 6.
