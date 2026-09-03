# ZATRANO V2 Stage B — rapor

Tarih: 2026-09-04  
Dallar: `framework` `v2-stage-b`, `packages` `v2-stage-b`  
Kapsam: fiziksel ayrım (addon’lar `github.com/zatrano/packages`), framework’ün uygulama iskeletini bırakması, `zatrano new`.

Intelligence katmanı (`packages/ai`, `packages/rag`, `packages/agent`) **taşınmadı**. Stage C CLI (`describe` / `doctor` / `agents:generate`) yok.

---

## 1. Ne kaldı / ne gitti

### Framework’te kalan

- `kernel/`, `contracts/`, `bootstrap/` (foundation + addon registry; addon **import’u yok**)
- LayerPrimitive / LayerFoundation / LayerIntelligence (`ai`, `rag`, `agent`)
- `packages/safepath`, `packages/database/driver/sqlite`
- `config/*.go` (foundation + addon provider’ların okuduğu namespace’ler)
- `cmd/zatrano` — yalnızca CLI (`new`, `make:*`, `package:*`, …)

Foundation hâlâ içe aktardığı için framework’te **kopyası duran** addon kütüphaneler (tam taşıma sonraki tur):  
`backup`, `cron`, `enums`, `export`, `factory`, `octane`, `openapi`, `pagination`, `testing`, `timing`, `totp`, `useragent`. Aynı isimler packages reposunda da var.

### `github.com/zatrano/packages`

LayerAddon servis + kütüphane ağaçları (katalog) ve nested sürücüler (`mysql` / `pgsql` / `mssql` / `oracle` / `mongo` / `webauthn` / `qr`).  
Kök `go.mod`: `require github.com/zatrano/framework v0.0.0` + `replace => ../framework`.  
**Not:** `v2.0.0-alpha` etiketi Go’da geçersizdir (`/v2` path gerekir). Yayın etiketi `v1.x` / `v0.x` olmalı.

### Starter (`zatrano new`)

Gömülü şablon: `packages/console/templates/` (`*.go.tmpl` — iç `go.mod` yok, `go test ./...` şablonları derlemez).  
Yer tutucular: `__MODULE__`, `__APP_NAME__`, `__FRAMEWORK_VERSION__`, `__REPLACE_LINE__`.  
Üretilen uygulama: `cmd/app`, `bootstrap.WithProviders(providers.All()...)`, `RegisterWeb` / `RegisterAPI`.

Framework kökünden silinen iskelet: `app/`, `routes/`, `views/`, `public/`, `lang/`, kök `storage/`, `database/migrations|seeders|factories`. `tests/storage` duruyor.

---

## 2. Faz özeti

| Faz | Durum | Commit / not |
| --- | --- | --- |
| 1 Audit | Tamam | `a6c2188` |
| 2 Packages split | Tamam | packages `a3a9c68`; framework `5777a7f` |
| 3–4 Skeleton + `zatrano new` | Tamam | framework `25d4ed2` |
| 5 CI | Tamam (etiket yok) | packages workflows; framework `v2-stage-*` + smoke job. **`v2.0.0-alpha` basılmadı.** |
| 6 E2E smoke | Tamam (CI job) | `.github/scripts/starter-smoke.sh`: `zatrano new`, billing blank-import, `go test`, `/health` |
| 7 Rapor | Bu dosya | |

---

## 3. Tüketici uygulaması nasıl bağlar

```go
import (
    _ "github.com/zatrano/packages/billing"
    "github.com/zatrano/framework/bootstrap"
)

app := bootstrap.App(
    bootstrap.WithAddons("billing"),
    bootstrap.WithProviders(providers.All()...),
)
```

Mongo / WebAuthn: uygulama blank-import eder (`github.com/zatrano/packages/database/driver/mongo`, `github.com/zatrano/packages/webauthn`). Framework kök `go.mod` bunları **require etmez** (modül döngüsü yok).

---

## 4. Doğrulama (bu makine)

- `go build ./...` ve `go test ./...` framework’te geçti (Faz 3–4 sonrası).
- `TestNewScaffoldsBuildableApp`: geçici dizinde `zatrano new` + `go build ./cmd/app`.
- Packages CI, framework dalı `v2-stage-b`’yi checkout eder — **önce framework `v2-stage-b` push** edilmeli; aksi halde GitHub `zatrano/framework@v2-stage-b` 404 verir.
- Smoke job packages `v2-stage-b` checkout eder (packages bu dalda zaten remote’da).

---

## 5. Bilinçli eksikler / sonraki tur

- 12 foundation-bağımlı kütüphane hâlâ framework kopyası; import kesilmeden tek kaynak packages olamaz.
- `make:*` iskelet string’leri hâlâ `github.com/zatrano/framework/app/...` yazabilir; üretilen uygulama kendi modül yolunu kullanmalı (ayrı iş).
- Sürüm etiketi / GitHub Release yok (Stage B dal çalışması; `framework-release` checklist’i main kesiminde).
- `mongo` / `webauthn` canlı CI ispatı smoke’da yok (billing + `/health` var). Nested sürücü `go get` + blank-import uygulama tarafında.
- Vanity import `zatrano.com/framework` V3.
