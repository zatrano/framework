# ZATRANO V2 Stage B — Faz 1 denetim

Tarih: 2026-09-03  
Dal: `v2-stage-b` (base: `v2-stage-a` @ `8b594c2`)  
Kapsam: kod değişikliği yok. Stage A ön koşulu + ortam + katalog envanteri.  
**Durma noktası:** `github.com/zatrano/packages` boş olarak mevcut, ama bu ortamdan repo oluşturma/`gh` yok; Faz 2+ Serhan onayı bekliyor.

---

## 1. Stage A ön koşul (§0.1)

| Koşul | Durum |
| --- | --- |
| `contracts/` | Var (`assert.go`, `config.go`, `container.go`, `logger.go`, `router.go`, `services.go`) |
| `core/` → `kernel/` | Git yalnızca `kernel/` izliyor; `core/` dizini yok |
| `LayerPrimitive` / `LayerFoundation` / `LayerIntelligence` / `LayerAddon` | `kernel/catalog.go` satır 9–12 |
| `bootstrap/addons` self-registration | `registry.go` boş slice + `Register(Meta)`; addon import yok. `providers.go` diskte yok (Grep önbelleği yanıltıcı) |
| Tek `App(opts ...Option)` | `bootstrap/app.go`; `APIApp`/`WebApp`/… yok. `Minimal()`/`Kernel()` **Option** döner: `App(Minimal())` |
| `RegisterWeb` / `RegisterAPI` | `packages/routing/discovery.go` |

**Sonuç:** Stage A ön koşulları dosyada mevcut. Bu prompta devam *edilebilir* — durma nedeni Stage A eksikliği değil, GitHub/orkestrasyon yetkisi.

---

## 2. Katalogdan programatik LayerAddon listesi

Kaynak: `kernel/catalog.go` `Catalog` slice (`PackagesByLayer(LayerAddon)`). Elle yazılmadı.

**Addon servis (26)** — `ai` intelligence’da, burada yok:

`audit`, `backup`, `billing`, `bus`, `circuit`, `docs`, `enums`, `features`, `geo`, `graphql`, `hashid`, `inspector`, `lock`, `mongo`, `oauth`, `octane`, `otp`, `pulse`, `search`, `shorturl`, `sitemap`, `social`, `tenancy`, `webauthn`, `webhooks`, `wellknown`

**Addon kütüphane (31)** — `rag`/`agent` intelligence’da, `safepath` katalogda yok:

`api`, `archive`, `bloom`, `browser`, `collection`, `concurrency`, `consent`, `cron`, `debug`, `export`, `factory`, `fingerprint`, `honeypot`, `idempotency`, `image`, `jsonapi`, `jsonschema`, `markdown`, `negotiate`, `openapi`, `pages`, `pagination`, `pdf`, `process`, `qr`, `resources`, `testing`, `timing`, `totp`, `useragent`, `websocket`

**Toplam LayerAddon: 57** (26 + 31).

**Framework’te kalacak (taşınmaz):**

- Intelligence: `ai` (servis), `rag`, `agent` (kütüphane)
- `packages/safepath` — katalog dışı, kernel-internal. Public/addon sinyali **yok** (`PACKAGES.md` “not in catalog”; kernel/http/backup/filesystem/zipx kullanıyor). Karar değiştirilmedi.
- `packages/database/driver/sqlite` — foundation “kurulumsuz çalış” vaadi.

**Framework’te kalıp Stage B’de bağımlılığı kesilecek nested `require`/`replace` (kök `go.mod`):**

`mongo`, `webauthn`, `qr` + SQL sürücüleri `mysql`/`pgsql`/`mssql`/`oracle` (sqlite `replace` kalır).

**Prompt listesinde yok, karar bekleyen:** `packages/database/driver/mongo` (ayrı `go.mod`, foundation `database` belge-deposu sürücüsü; `packages/mongo` addon’undan farklı). Katalogda LayerAddon değil. Faz 2’de sqlite gibi foundation’da bırakmayı öneriyorum — aksi Serhan onayı.

---

## 3. Ortam — GitHub repo yetkisi

| Araç | Sonuç |
| --- | --- |
| `gh` CLI | **Yok** (`where gh` boş) |
| `GH_TOKEN` / `GITHUB_TOKEN` | **unset** |
| `git remote origin` | `https://github.com/zatrano/framework.git` |
| `GET https://api.github.com/repos/zatrano/packages` | **200** — repo **zaten var** |
| Repo durumu | public, `size=0`, `pushed_at=2026-09-03T20:04:33Z`, `git ls-remote` **boş** (henüz commit/branch yok) |
| Cursor `origin repo create` | Cursor-hosted repo üretir; hedef `github.com/zatrano/packages` değil — kullanılmadı |

Bu ajan **yeni GitHub reposu oluşturamaz** (`gh repo create` yok, token yok). Hedef repo Serhan tarafından boş açılmış görünüyor.

### Serhan’dan beklenen manuel adımlar (Faz 2 öncesi)

Prompt: yetki yoksa dur, **“repo oluşturuldu, devam et”** onayı bekle.

1. **Onay:** Boş `https://github.com/zatrano/packages` Stage B hedefi midir? (Evet ise “devam et”.)
2. **Push yetkisi:** Bu makineden `git push https://github.com/zatrano/packages.git` çalışmalı. `gh` yok; Git Credential Manager ile `zatrano/framework` push’u varsa aynı hesap `packages`’e de yazabilmeli. Güvenilir yol: [GitHub CLI](https://cli.github.com/) kur (`winget install GitHub.cli`) + `gh auth login` (org `zatrano`, `repo` scope).
3. **İlk branch:** Onay sonrası ajan `v2-stage-b` (veya `main`) üzerine history-preserving split push’lar. Branch protection **ilk push’tan sonra** GitHub UI’dan eklenebilir — şimdi gerekmez (repo boş).
4. **`git-filter-repo` (isteğe bağlı):** Kurulu değil. `git subtree split` **var** (git 2.46). Geçmiş koruma subtree ile yapılabilir; daha temiz path rewrite için: `pip install git-filter-repo`.
5. **`gh` yokken CI/release:** Faz 5 GitHub Release/`gh` isterse aynı kurulum gerekir; `git tag` + `git push origin tag` credential ile mümkün olabilir.

**Faz 2+ bu onay olmadan başlamadı.**

---

## 4. Git geçmişi araçları

| Araç | Durum |
| --- | --- |
| `git filter-repo` | Yok; `python -m git_filter_repo` yok; pip paketi kurulu değil |
| `git subtree split` | **Var** — geçmiş koruma planı B |
| Düz kopyala-yapıştır | Yapılmayacak; araç yoksa raporlanır (subtree mevcut) |

---

## 5. Stage B’yi etkileyecek Stage A artıkları (Faz 2/3 işi — şimdi dokunulmadı)

Self-registration `bootstrap/addons` içinde temiz; **demo/kök hâlâ addon’ları import ediyor:**

- `bootstrap/register_addons.go` — 25 addon blank-import (`ai` dahil; `ai` framework’te kalır)
- `bootstrap/nested_addons.go` — `mongo` + `webauthn` kök kaydı (silinecek)
- `bootstrap/foundation/boot.go` — **`packages/mongo` import** (`bootMongoConnection`). Split sonrası kernel/foundation mongo bilmemeli; belge-deposu boot’u addon/uygulama tarafına taşınmalı
- Kök `go.mod` `require`+`replace`: mongo, webauthn, qr, mysql, pgsql, mssql, oracle, sqlite

`config/` addon-özel: `ai.go`, `billing.go`, `mongo.go`, `oauth.go`, `social.go`, `webauthn.go`. Foundation-ish kalan: `app.go`, `auth.go`, `database.go`, `notifications.go`, `session.go` — Faz 3/4 starter’a neyin gideceği orada ayrılacak.

---

## 6. Karar: dur

Stage A tamam. Katalog listesi yukarıda. `zatrano/packages` boş duruyor ama oluşturma/push orkestrasyonu bu oturumda doğrulanmış değil.

**Sonraki fazlara geçilmedi.** Serhan: (1) packages repo onayı, (2) push/`gh auth`, (3) isteğe bağlı `git-filter-repo` kurulumu.
