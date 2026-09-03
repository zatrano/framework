# ZATRANO V2 Stage A — Final rapor

Dal: `v2-stage-a`  
Base: `main` @ `90faf57` (VERSION `1.6.6`)  
Tarih: 2026-09-03  
Kapsam: 8 faz — kod içi bağımlılık yönü. Fiziksel repo bölünmesi yok. Modül path `github.com/zatrano/framework` değişmedi.

Toplam diff (`90faf57..HEAD`, bu rapor commiti hariç): **143 dosya, +2274 / −1255**.

---

## 1. Faz 1 denetimi — doğrulanan / düzeltilen bulgular

Kaynak: [`docs/v2-stage-a-audit.md`](v2-stage-a-audit.md). Prompt’taki “Bağlam” iddiaları dosyada doğrulandı; sapmalar:

| İddia | Gerçek |
| --- | --- |
| Addon kütüphane 36 / addon toplam 63 | **33 kütüphane / 60 toplam** |
| `Bootstrap()` 8 addon config | **9**: bağlamdaki 8 + **`appconfig.Database()`** |
| Altı boot fonksiyonu | `App` + `APIApp` + `WebApp` + `DemoApp` + `MinimalApp` + `CoreApp` + bağlamda olmayan **`KernelApp` alias** |
| 24 accessor | Hepsi var; ek: **`Version() string`** |
| `packages/mongo`, `packages/webauthn` | Ayrı `go.mod` (root `replace`) — self-register `init()` bu paketlerde root `bootstrap/addons` import edemez |
| `config/app.go` Bootstrap’ta çağrılıyor | **Hayır** — app anahtarı inline (`name/env/debug/url/key/locale/fallback`); `port`/`cors` yüklenmiyordu, sessizce doldurulmadı |

`Application` private field’ları (Faz 3 gizleme kararı buna dayandı): `basePath`, `container`, `config`, `router`, `logger`, `rateLimiter`, `ctx`, `urls`, `encrypter`, `hasher`, `metrics`, `health`, `maintenance`, `exceptions`, `reports`, `httpBridge`, `migrations`, `seeders`, `providers`, `booted`, `environment`.

`HTTPBridge` zaten interface — Faz 2/7 desen örneği.

---

## 2. Faz özetleri

### Faz 1 — Denetim (kod yok)

- **Commit:** `892dda5` — *Record Stage A Faz 1 audit of kernel accessors, catalog, and boot wiring.*
- **Dosyalar / satır:** `docs/v2-stage-a-audit.md` (+130)
- **Neden:** Bağlamı körü körüne kullanmamak; sonraki fazların field/import/catalog kararlarını sabitlemek.
- **Test:** yok (yalnızca doküman).

### Faz 2 — `contracts/` ve interface accessor’lar

- **Commit:** `75c94ed` — *Expose Application accessors through contracts interfaces.*
- **Dosyalar / satır:** 15 dosya, +240 / −45
- **Neden:** SemVer sözleşmesini somut struct’lardan ayırmak. Public dönüş tipleri `contracts.*`; private field’lar somut kaldı. Primitive’ler (`Make`/`Bound`/`Environment`/`IsProduction`/`IsDebug`/`BasePath`) ve `HTTPBridge` dokunulmadı.
- **Interface yüzeyi (fiilen kullanılan metodlar):**
  - `Container` (`Instance`), `ConfigRepository` (`Get`/`GetString`/`GetInt`; Faz 3’te `Load`/`All`), `Router` (`Get`/`Post`/`Use`/`Name`/`Group`/`Routes`/`Snapshot`/`SaveCache`)
  - `Logger` (`Debugf`/`Infof`/`Errorf`), `ContextStore` (`Put`), `URLGenerator` (`To`/`Route`/`Signed`)
  - `Encrypter`, `Hasher`, `Metrics` (`Snapshot`), `Health` (`Custom`/`Database`/`Handler`)
  - `Maintenance` (`Enable`/`Disable`), `Exceptions` (`Middleware`)
  - `RateLimiter` (`For`) ve `Reports` (`Recent`): dışarıdan çağrı yoktu; boş sözleşme yerine kernel boot’ta instance üzerinde kullanılan metod alındı.
- **Yan değişiklikler (derleme):** typed-nil koruması; `routing.Controller` → `RouteRegistrar`; `broadcasting`/`notification`/`pulse` dar logger/metrics interface; kernel Timing/Maintenance/Exceptions private field.
- **Test:** mevcut suite; `contracts/` compile-time `assert.go` (test dosyası yok).

### Faz 3 — Kernel `Bootstrap()` addon config’ten arındırma

- **Commit:** `ccb521c` — *Move addon and foundation config loading out of kernel Bootstrap.*
- **Dosyalar / satır:** 7 dosya, +63 / −12 (o anki `bootstrap/addons/providers.go` +8; Faz 4’te dosya silindi, yükleme her paketin `Register()`’ına taşındı)
- **Neden:** Kernel hangi addon’ların var olduğunu bilmesin. `kernel/` artık `appconfig` import etmiyor (`grep appconfig. kernel/` → sıfır).
- **Nereye taşındı:** foundation `database`/`auth`/`notifications`; addon `oauth`/`mongo`/`webauthn`/`billing`/`ai`/`social` kendi Provider `Register()` içinde. Kernel yalnızca inline genel `app` config.
- **Yardımcı:** `config.LoadIfAbsent` — cache varsa overwrite yok.
- **Test:** `packages/config/cache_test.go` (+13); `tests/package_boot_test.go` — MinimalApp foundation config, addon config yalnızca demo profili.

### Faz 4 — Statik registry → self-registration

- **Commit:** `b97eb03` — *Replace static addon registry with package self-registration.*
- **Dosyalar / satır:** 31 dosya, +1122 / −674
- **Neden:** `bootstrap/addons` hiçbir `packages/{ai,billing,...}` import etmesin. `providers.go` (~625 satır) silindi; `registry.go` boş slice + `Register(Meta)` (mutex).
- **25 root-modül addon:** `packages/{name}/provider.go` `init()` → `addons.Register`. `packages/ai` için **yalnızca yeni `provider.go`**; mevcut AI logic dosyalarına dokunulmadı. `rag`/`agent` kütüphane — registry’de yok.
- **Blank import:** `bootstrap/register_addons.go` (25 paket). Tüketici uygulamalar yalnızca etkinleştirdiklerini import etmeli; bu demo repo tam seti import eder.
- **İstisna:** `mongo`/`webauthn` ayrı modül — `bootstrap/nested_addons.go` kök modülde kayıt (paket `init()` ile `bootstrap/addons` import edemez).
- **Test:** `bootstrap/addons/registry_test.go`, `catalog_test.go` → `package addons_test` + `_ "github.com/zatrano/framework/bootstrap"`; `tests/package_boot_test.go` davranış korunarak güncellendi.

### Faz 5 — Tek `App(opts ...Option)`

- **Commit:** `c4453e5` — *Collapse boot profiles into App with functional options.*
- **Dosyalar / satır:** 11 dosya, +99 / −72
- **Neden:** `APIApp`/`WebApp`/`DemoApp`/`MinimalApp`/`CoreApp`/`KernelApp` ayrı giriş noktalarını kaldırmak.
- **API:** `App()` = foundation + `EnabledAddons`; `WithAddons`, `Minimal()`, `Kernel()`, `WithDemo()`, `WithPresetAPI()`, `WithPresetWeb()`. `APP_BOOT` aynı değerlerle option’a map edilir.
- **Çağrı yerleri:** `cmd/zatrano/main.go`, testler, `package:list`/`package:doctor`, README tablosu; `make:*` / `db:setup` → `App(Kernel())`.
- **Test:** `tests/package_boot_test.go`, `packages/console/package_cmd_test.go`.

### Faz 6 — `core` → `kernel` + catalog taksonomi

- **Commit:** `341add8` — *Rename core to kernel and add the intelligence catalog layer.*
- **Dosyalar / satır:** 118 dosya, +544 / −514
- **Neden:** İsim çakışması (`core` hem paket hem “kernel katmanı”) ve AI kimliğini “opsiyonel addon”dan ayırmak.
- **Catalog:** `LayerKernel` → `LayerPrimitive` (`"primitive"`). Yeni `LayerIntelligence`: `ai` (KindService), `rag`/`agent` (KindLibrary). Fiziksel konum ve paket içeriği değişmedi. Aktivasyon hâlâ self-register + `EnabledAddons` (`ai` için).
- **Import:** `"github.com/zatrano/framework/core"` → `"github.com/zatrano/framework/kernel"`.
- **CLI:** `package:list --libraries` `kernel.LibraryCatalog()` kullanır (intelligence + addon kütüphaneleri).
- **Test:** `kernel/catalog_test.go` (3 intelligence paketi); `bootstrap/addons/catalog_test.go` LayerIntelligence KindService.

### Faz 7 — Route self-registration primitifleri

- **Commit:** `94cbdd6` — *Add route group self-registration primitives for web and API.*
- **Dosyalar / satır:** 7 dosya, +154 / −16
- **Neden:** Framework tarafında web/api grup + register/apply çifti; tüketicilerin izleyeceği desen bu repoda `routes/` ile örneklenir.
- **API (`packages/routing/discovery.go`):** `RouteGroup`, `GroupWeb`/`GroupAPI`, `RegisterWeb`/`RegisterAPI`, `ApplyWeb`/`ApplyAPI`. Apply kopya slice üzerinde çalışır (kayıt sırasında kilit tutmaz).
- **Örnek kullanım:** `routes/web.go` ve `routes/health.go` `init()` → `RegisterWeb`; `routes/api.go` → `RegisterAPI`. Health hâlâ web grubunda (`/up`, `/health`, `/api/health`) — eski `Boot()` sırası korunur. `routes.Use(app)` çünkü `init()`’te app yok; `Boot()` önce bağlar sonra apply eder.
- **Provider:** `routing.ApplyWeb`/`ApplyAPI` somut `*routing.Router` ister; `app.Router()` `contracts.Router` döndüğü için type assert (`HTTPBridge` deseninin tersi: burada apply hedefi somut router).
- **Test (yeni):** `packages/routing/discovery_test.go` — iki ayrı `RegisterWeb` çağrısı, `ApplyWeb` sonrası `/from-a` ve `/from-b`. Global registry save/restore ile izole.

### Faz 8 — Bu rapor

- **Commit:** bu dosya.
- **Dosyalar:** `docs/v2-stage-a-report.md`

---

## 3. Kalan riskler / kapsam dışı işler

Bu promptun **bilinçli olarak dışarıda bıraktığı** işler (ayrı aşamada):

1. **Fiziksel repo bölünmesi** (kernel / foundation / addon / intelligence ayrı git modülleri). Stage A yalnızca bağımlılık yönünü düzeltti.
2. **Vanity import** `zatrano.com/framework` — V3.
3. **`packages/ai|rag|agent` içeriği** (hardening). Yalnızca `ai/provider.go` eklendi; rag/agent’a dokunulmadı.
4. **mongo/webauthn nested module:** self-register `init()` hâlâ kök `bootstrap/nested_addons.go`’da. Ayrı modül `bootstrap/addons` import edemez; fiziksel split’te bu iki paket kendi modüllerinde kayıt için `contracts` veya ince bir registry modülü gerekir.
5. **Bu demo app tüm 27 servis addon’u blank-import eder** (`register_addons.go` + nested). Tüketici uygulamalar yalnızca kullandıklarını import etmeli — aksi halde binary’ye çekilirler. Dokümante edilmeli ama bu aşamada demo davranışı korundu.
6. **`contracts` somut paket tipleri kullanır** (`*routing.Route`, `ratelimit.Limit`, `maintenance.Payload`). Tamamen kernel-agnostic bir sözleşme değil; SemVer yüzeyi küçüldü ama paket import’u duruyor. İleride DTO’lara çekilebilir.
7. **`ApplyWeb`/`ApplyAPI` somut `*Router` ister** — `contracts.Router` üzerinden çağrılamaz. Tüketici `type assert` veya routing paketini import eder. Route discovery tüketici uygulamasına yaymak bu promptun dışında.
8. **`routes.Use(app)` process-global** — test paralelliğinde çakışma riski düşük (tek Boot) ama çoklu `Application` aynı process’te route apply ederse son `Use` kazanır.
9. **Davranış kayması (bilinçli, testle kilitli):** Minimal/Kernel boot artık oauth/mongo/billing addon config taşımaz — kernel `Bootstrap()` onları yüklemiyor. Demo/full `App()` aynı config’i ilgili provider `Register()` ile yükler.
10. **CRLF:** Faz 6 toplu replace bazı dosyalarda CRLF üretmiş olabilir; git LF’ye normalize eder.

---

## 4. Build / test kanıtı

Komutlar Faz 7 değişikliklerinden sonra (commit `94cbdd6` içeriğiyle, bu rapor commiti öncesi) kök modülde çalıştırıldı. Çıkış kodu **0**.

```
go build ./...
```

Çıktı: boş (başarılı).

```
go test ./...
```

```
?   	github.com/zatrano/framework/app/console	[no test files]
?   	github.com/zatrano/framework/app/http/controllers/api	[no test files]
?   	github.com/zatrano/framework/app/http/controllers/web	[no test files]
?   	github.com/zatrano/framework/app/providers	[no test files]
ok  	github.com/zatrano/framework/bootstrap	0.781s
ok  	github.com/zatrano/framework/bootstrap/addons	0.452s
?   	github.com/zatrano/framework/bootstrap/foundation	[no test files]
?   	github.com/zatrano/framework/bootstrap/stubs	[no test files]
ok  	github.com/zatrano/framework/cmd/zatrano	0.530s
?   	github.com/zatrano/framework/config	[no test files]
?   	github.com/zatrano/framework/contracts	[no test files]
?   	github.com/zatrano/framework/database/migrations	[no test files]
?   	github.com/zatrano/framework/database/seeders	[no test files]
ok  	github.com/zatrano/framework/kernel	2.583s
ok  	github.com/zatrano/framework/packages/agent	4.787s
ok  	github.com/zatrano/framework/packages/ai	5.028s
ok  	github.com/zatrano/framework/packages/api	(cached)
ok  	github.com/zatrano/framework/packages/apitoken	(cached)
ok  	github.com/zatrano/framework/packages/archive/zipx	(cached)
ok  	github.com/zatrano/framework/packages/assets	(cached)
ok  	github.com/zatrano/framework/packages/audit	2.263s
ok  	github.com/zatrano/framework/packages/auth	(cached)
ok  	github.com/zatrano/framework/packages/authorization	(cached)
ok  	github.com/zatrano/framework/packages/backup	4.895s
ok  	github.com/zatrano/framework/packages/billing	3.532s
ok  	github.com/zatrano/framework/packages/bloom	(cached)
ok  	github.com/zatrano/framework/packages/broadcasting	(cached)
ok  	github.com/zatrano/framework/packages/browser	0.859s
ok  	github.com/zatrano/framework/packages/bus	2.981s
ok  	github.com/zatrano/framework/packages/cache	(cached)
ok  	github.com/zatrano/framework/packages/circuit	3.264s
ok  	github.com/zatrano/framework/packages/collection	(cached)
ok  	github.com/zatrano/framework/packages/concurrency	(cached)
ok  	github.com/zatrano/framework/packages/config	(cached)
ok  	github.com/zatrano/framework/packages/consent	(cached)
ok  	github.com/zatrano/framework/packages/console	0.820s
?   	github.com/zatrano/framework/packages/container	[no test files]
?   	github.com/zatrano/framework/packages/context	[no test files]
?   	github.com/zatrano/framework/packages/cookie	[no test files]
ok  	github.com/zatrano/framework/packages/cron	(cached)
ok  	github.com/zatrano/framework/packages/database	(cached)
?   	github.com/zatrano/framework/packages/database/migration	[no test files]
ok  	github.com/zatrano/framework/packages/database/query	(cached)
ok  	github.com/zatrano/framework/packages/database/schema	(cached)
?   	github.com/zatrano/framework/packages/database/seeder	[no test files]
ok  	github.com/zatrano/framework/packages/debug	(cached)
ok  	github.com/zatrano/framework/packages/docs	2.366s
?   	github.com/zatrano/framework/packages/encryption	[no test files]
ok  	github.com/zatrano/framework/packages/enums	2.099s
ok  	github.com/zatrano/framework/packages/env	(cached)
ok  	github.com/zatrano/framework/packages/events	(cached)
ok  	github.com/zatrano/framework/packages/exceptions	(cached)
ok  	github.com/zatrano/framework/packages/export	(cached)
ok  	github.com/zatrano/framework/packages/export/csv	(cached)
ok  	github.com/zatrano/framework/packages/export/xlsx	(cached)
ok  	github.com/zatrano/framework/packages/factory	(cached)
ok  	github.com/zatrano/framework/packages/features	2.258s
ok  	github.com/zatrano/framework/packages/filesystem	(cached)
ok  	github.com/zatrano/framework/packages/fingerprint	(cached)
ok  	github.com/zatrano/framework/packages/flash	(cached)
ok  	github.com/zatrano/framework/packages/geo	4.687s
ok  	github.com/zatrano/framework/packages/graphql	2.592s
ok  	github.com/zatrano/framework/packages/hashid	2.282s
?   	github.com/zatrano/framework/packages/hashing	[no test files]
?   	github.com/zatrano/framework/packages/health	[no test files]
ok  	github.com/zatrano/framework/packages/honeypot	(cached)
ok  	github.com/zatrano/framework/packages/http	(cached)
ok  	github.com/zatrano/framework/packages/httpclient	(cached)
ok  	github.com/zatrano/framework/packages/idempotency	(cached)
ok  	github.com/zatrano/framework/packages/image	(cached)
ok  	github.com/zatrano/framework/packages/inspector	2.246s
ok  	github.com/zatrano/framework/packages/jsonapi	(cached)
ok  	github.com/zatrano/framework/packages/jsonschema	(cached)
ok  	github.com/zatrano/framework/packages/localization	(cached)
?   	github.com/zatrano/framework/packages/localization/defaults	[no test files]
ok  	github.com/zatrano/framework/packages/lock	2.309s
ok  	github.com/zatrano/framework/packages/log	(cached)
ok  	github.com/zatrano/framework/packages/maintenance	(cached)
ok  	github.com/zatrano/framework/packages/markdown	(cached)
ok  	github.com/zatrano/framework/packages/middleware	(cached)
ok  	github.com/zatrano/framework/packages/middleware/csrf	(cached)
ok  	github.com/zatrano/framework/packages/negotiate	(cached)
ok  	github.com/zatrano/framework/packages/notification	(cached)
ok  	github.com/zatrano/framework/packages/oauth	2.655s
ok  	github.com/zatrano/framework/packages/observability	(cached)
ok  	github.com/zatrano/framework/packages/octane	2.432s
ok  	github.com/zatrano/framework/packages/openapi	2.177s
ok  	github.com/zatrano/framework/packages/orm	(cached)
ok  	github.com/zatrano/framework/packages/otp	2.002s
ok  	github.com/zatrano/framework/packages/pages	2.166s
ok  	github.com/zatrano/framework/packages/pagination	(cached)
ok  	github.com/zatrano/framework/packages/pdf	(cached)
ok  	github.com/zatrano/framework/packages/pipeline	(cached)
ok  	github.com/zatrano/framework/packages/process	(cached)
ok  	github.com/zatrano/framework/packages/pulse	1.995s
ok  	github.com/zatrano/framework/packages/queue	(cached)
ok  	github.com/zatrano/framework/packages/rag	3.180s
ok  	github.com/zatrano/framework/packages/ratelimit	(cached)
?   	github.com/zatrano/framework/packages/redisx	[no test files]
ok  	github.com/zatrano/framework/packages/report	(cached)
ok  	github.com/zatrano/framework/packages/resources	(cached)
ok  	github.com/zatrano/framework/packages/routing	(cached)
ok  	github.com/zatrano/framework/packages/safepath	(cached)
ok  	github.com/zatrano/framework/packages/schedule	(cached)
ok  	github.com/zatrano/framework/packages/search	2.353s
ok  	github.com/zatrano/framework/packages/session	(cached)
ok  	github.com/zatrano/framework/packages/shorturl	2.435s
ok  	github.com/zatrano/framework/packages/sitemap	2.189s
ok  	github.com/zatrano/framework/packages/social	4.065s
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
ok  	github.com/zatrano/framework/packages/tenancy	2.411s
?   	github.com/zatrano/framework/packages/testing	[no test files]
ok  	github.com/zatrano/framework/packages/timing	(cached)
ok  	github.com/zatrano/framework/packages/totp	(cached)
ok  	github.com/zatrano/framework/packages/trustedproxy	(cached)
ok  	github.com/zatrano/framework/packages/url	2.051s
ok  	github.com/zatrano/framework/packages/useragent	(cached)
ok  	github.com/zatrano/framework/packages/validation	(cached)
ok  	github.com/zatrano/framework/packages/version	(cached)
ok  	github.com/zatrano/framework/packages/view	(cached)
ok  	github.com/zatrano/framework/packages/webhooks	2.060s
ok  	github.com/zatrano/framework/packages/websocket	(cached)
ok  	github.com/zatrano/framework/packages/wellknown	2.094s
?   	github.com/zatrano/framework/routes	[no test files]
ok  	github.com/zatrano/framework/tests	1.490s
ok  	github.com/zatrano/framework/tests/fuzz	3.062s
?   	github.com/zatrano/framework/tests/securitydemo	[no test files]
```

Fail yok. `packages/mongo` ve `packages/webauthn` ayrı `go.mod` olduğu için kök `./...` kapsamına girmez (önceki `main` davranışı).

Her faz commiti de kendi turunda `go build ./...` + `go test ./...` ile geçirildi.

---

## Commit zinciri

| Hash | Faz | Konu |
| --- | --- | --- |
| `892dda5` | 1 | Audit dokümanı |
| `75c94ed` | 2 | contracts accessor’lar |
| `ccb521c` | 3 | Kernel Bootstrap config arındırma |
| `b97eb03` | 4 | Addon self-registration |
| `c4453e5` | 5 | Tek `App(opts...)` |
| `341add8` | 6 | `core` → `kernel` + LayerIntelligence |
| `94cbdd6` | 7 | Route discovery primitifleri |
| (bu commit) | 8 | Final rapor |
