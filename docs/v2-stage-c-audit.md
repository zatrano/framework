# ZATRANO V2 Stage C — Faz 1 ön koşul denetimi

Tarih: 2026-09-04  
Dal: `v2-stage-c` (base: `v2-stage-b` @ `bbf14b3`)  
Kapsam: kod değişikliği yok. Stage A varsayımları diskte doğrulandı.

**Sonuç: devam edilebilir.** Üç ön koşul da mevcut. İsim sapmaları aşağıda; Stage C bunlara göre inşa edilmeli.

---

## 1. `contracts/`

Dizin var. Git izlenen dosyalar:

| Dosya | İçerik |
| --- | --- |
| `contracts/router.go` | `Router` — `Get`/`Post`/`Use`/`Group`/`Name`/`Routes`/`Snapshot`/`SaveCache` |
| `contracts/config.go` | `ConfigRepository` |
| `contracts/container.go` | `Container` |
| `contracts/logger.go` | `Logger` |
| `contracts/services.go` | `RateLimiter`, `ContextStore`, `URLGenerator`, `Encrypter`, `Hasher`, `Metrics`, `Health`, `Maintenance`, `Exceptions`, `Reports` |
| `contracts/assert.go` | Derleme-zamanı `_ Interface = (*concrete)(nil)` bağları |

Promptun örneklediği `Router` ve `ConfigRepository` mevcut (`contracts/router.go` satır 6, `contracts/config.go` satır 4).

**Sapma (Stage A gerçeği):** `Router` imzaları hâlâ `packages/routing` somut tiplerine (`routing.HandlerFunc`, `*routing.Route`, `*routing.Router`) bağlı. Stage C `doctor` somut-tip kontrolü bunu resmi şablonda da görecek (`RouteServiceProvider` `*routing.Router` type-assert kullanır).

---

## 2. `kernel/` ve katalog katmanları

- `core/` **yok** (disk `Test-Path core` = false; git `core/` = 0 dosya). Yeniden adlandırma tamam.
- `kernel/catalog.go` satır 8–12:

```go
LayerPrimitive    Layer = "primitive"
LayerFoundation   Layer = "foundation"
LayerIntelligence Layer = "intelligence"
LayerAddon        Layer = "addon"
```

- Intelligence paketleri katalogda (satır 96–98): `ai` (service), `rag` (library), `agent` (library). Bu prompt bunların **içeriğine dokunmaz**.
- `kernel.Provider` (`kernel/application.go` satır 37–40): `Register` / `Boot`.

---

## 3. Self-registration

- `bootstrap/addons/registry.go`: boş `registry` dilimi; `Register(Meta)` (`init` çağrısı için); `Select` / `Available`.
- `bootstrap/addons` **addon paketlerini import etmez** (kayıt tüketicinin blank-import’u ile gelir).
- Framework `bootstrap/register_addons.go` yalnızca `_ "…/packages/ai"` (intelligence).
- Stage B sonrası `backup` gibi addon’lar `github.com/zatrano/packages` içinde `init()` → `addons.Register`.

Desen çalışıyor; Stage A’daki “registry.go addon import etmez” kuralı duruyor.

---

## 4. Stage C’nin uyması gereken Stage B şablonu

Prompt Faz 3’te `app/controllers`, `app/config` örnekliyor. **Üretilen V2 uygulama** (`packages/console/templates/`):

- `app/routes/web/*.go`, `app/routes/api/*.go` + `routing.RegisterWeb` / `RegisterAPI`
- denetleyiciler: `app/http/controllers/{web,api}` (üst düzey `app/controllers` yok)
- `app/providers`, `app/services`
- uygulama `config/` taşımaz; config framework `config/*.go` + isteğe bağlı publish

`doctor` beklenen ağacı bu şablona göre kontrol edecek, prompt metnindeki eski yollara göre değil.

---

## 5. İsim çakışması

`zatrano package:doctor` zaten var (`packages/console/package_doctor.go`, addon listesi / stub denetimi). Stage C komutu **`zatrano doctor`** (mimari). İkisi ayrı kalacak.

---

**Faz 2+ bu denetimle başlayabilir.**
