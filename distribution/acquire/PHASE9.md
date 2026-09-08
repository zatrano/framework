# Phase 9 — Sınırlandırıcı SPEC Taslağı

**Durum:** Draft — Implementation kapalı  
**Önkoşul:** Phase 8 (`v2.0.22`) frozen  
**Acceptance:** NOT ACCEPTED  
**Implementation:** LOCKED

Bu belge yalnızca SPEC taslağıdır. Bu aşamada kod yazılmayacaktır.

---

## 1. Kapsam

Phase 9 yalnızca üç bağımsız sözleşmeyi tanımlar:

1. **Contract A — Dry-run**
2. **Contract B — CLI Acquisition**
3. **Contract C — Acquisition ↔ Enablement**

Bu fazda sözleşme tanımlanır; implementation yapılmaz.

---

# Contract A — Dry-run

## Amaç

Acquisition işleminin ne yapacağını, herhangi bir acquisition mutation gerçekleştirmeden göstermek.

## MUST

Dry-run:

* mevcut concrete acquisition plan/target üzerinden çalışır.
* module path'i gösterir.
* concrete version/ref'i gösterir.
* `GoGetArg` değerini gösterir.
* hedef module root'u gösterir.
* çalıştırılacak acquisition komutunun biçimini gösterir.
* execution boundary'ini çağırmaz.
* filesystem mutation yapmaz.
* `go.mod` değiştirmez.
* `go.sum` değiştirmez.

## MUST NOT

Dry-run:

* `go get` çalıştıramaz.
* `go mod tidy` çalıştıramaz.
* recovery çalıştıramaz.
* enablement yapamaz.
* `package:install` çalıştıramaz.
* package registry resolution yapamaz.
* `latest` çözümleyemez.
* semver algoritması uygulayamaz.
* `GoGetArg` üretme algoritmasını yeniden uygulayamaz.
* acquisition sonucunu installed / acquired / applied olarak raporlayamaz.

## Temel ilke

Dry-run, execution'ın ikinci bir versiyonu değildir.

Aynı acquisition planı tüketir:

```text
Resolve
  ↓
FromResult
  ↓
Targets
  ↓
Plan
  ├── Dry-run
  └── Execute
```

Dry-run ile execution arasındaki fark plan değil, mutation boundary'sidir.

## Örnek

Concrete plan:

```text
module: github.com/zatrano/packages
version: main
arg: github.com/zatrano/packages@main
root: /project
```

Dry-run bunu raporlayabilir; ancak `go get` çalıştıramaz.

---

# Contract B — CLI Acquisition

## Amaç

Kullanıcı tarafından çağrılabilen acquisition workflow'unun mevcut acquisition API'lerini orchestrate etmesi.

CLI kendi acquisition motorunu oluşturmaz.

## CLI MUST

CLI:

* mevcut registry resolution sonucunu kullanır.
* mevcut `FromResult` / `Targets` / `Plan` akışını kullanır.
* mevcut `GoGetArg` değerini kullanır.
* mevcut execution boundary'sini kullanır.
* acquisition sonucunu kullanıcıya raporlar.
* partial acquisition sonucunu açıkça raporlar.
* recovery sonucunu açıkça raporlayabilir.
* dry-run desteği varsa aynı acquisition planını kullanır.

## CLI MUST NOT

CLI:

* ikinci resolver içeremez.
* semver resolution yapamaz.
* registry algorithm'ini kopyalayamaz.
* module normalization algoritması oluşturamaz.
* conflict resolution algoritması oluşturamaz.
* `go get` komutunu kendisi execute eden ikinci abstraction oluşturamaz.
* `go.mod` / `go.sum` dosyalarını doğrudan düzenleyemez.
* acquisition state modelinin ikinci bir versiyonunu oluşturamaz.

## Ayrım

```text
User
 ↓
CLI
 ↓
Existing Resolution / Planning APIs
 ↓
Existing Acquisition Execution Boundary
 ↓
Inspect / Result
```

CLI'nin görevi orchestration + presentation'dır.

---

# Contract C — Acquisition ↔ Enablement

Bu sözleşme iki state transition'ı birbirinden kesin olarak ayırır.

**Acquisition ≠ Enablement**

## Acquisition

Go module dependency state'ini değiştirir:

```text
module dependency
       ↓
go get
       ↓
go.mod / go.sum
```

## Enablement

Package runtime state'ini değiştirir:

```text
package
  ↓
import / registration / enabled state
```

## MUST

Başarılı acquisition otomatik olarak enablement yapmaz.

Acquisition failure enablement başlatmaz.

Enablement failure acquisition rollback anlamına gelmez.

Acquisition ile enablement arasında implicit transaction yoktur.

## `package:install`

`package:install` bu fazda acquisition command'i haline gelmez.

Anlamı: **enablement** olarak kalır.

## State isolation

| Adım | Acquisition | Enablement |
|------|-------------|------------|
| Başlangıç | Not acquired | Not enabled |
| Acquisition başarılı | Acquired | Not enabled |
| Enablement başarılı | Acquired | Enabled |
| Acquisition başarısız | Not acquired | Not enabled |
| Enablement başarısız | Acquired | Not enabled |

Son durumda acquisition'ın otomatik rollback edilmesi bu SPEC tarafından garanti edilmez.

## Gelecek birleşik workflow

İleride `Acquire + Enable` gibi tek bir kullanıcı workflow'u tasarlanabilir. Ancak bunun için ayrı bir sözleşme gerekir. Phase 9 bunu tanımlamaz.

---

# Phase 8 Freeze Boundary

Phase 9 aşağıdaki Phase 8 yüzeyine dokunmaz:

```text
FromResult
    ↓
Targets
    ↓
GoGetArg
    ↓
Execute / ExecuteTargets
    ↓
Inspect
    ↓
ApplyResult
    ↓
SnapshotFiles / RecoverFiles
```

Aşağıdakiler değiştirilemez:

* Phase 7 API'leri
* `GoGetArg`
* `Execute`
* `ExecuteTargets`
* `Inspect`
* recovery semantics
* fail-fast davranışı
* partial-result modeli
* concurrency / lock davranışı
* Go tooling mutation sınırı

Ve özellikle **`func Apply`** yasaktır.

Yeni bir Apply varyantı veya isim değiştirilmiş eşdeğeri de bu SPEC kapsamında kabul edilmez.

---

# Phase 9 Global Invariants

Phase 9 implementation'ı:

* Yeni resolver oluşturamaz.
* İkinci process abstraction oluşturamaz.
* `func Apply` ekleyemez.
* `package:install` semantiğini değiştiremez.
* `zatrano.lock` ekleyemez.
* Otomatik `go mod tidy` ekleyemez.
* Phase 7 acquisition planning API'sini değiştiremez.
* Phase 8 execution semantics'ini değiştiremez.
* Acquisition ile enablement'ı implicit transaction haline getiremez.
* A/B/C sözleşmelerini tek bir birleşik contract'a dönüştüremez.

---

# Implementation Gate

Implementation öncesi sıralama:

```text
Phase 9 Draft
     ↓
SPEC Acceptance
     ↓
Contract A/B/C implementation
```

Implementation aşamasında her sözleşme için:

```text
Contract
   ↓
Tests
   ↓
Implementation
   ↓
Verification
```

şeklinde ilerlenir.

## Mevcut durum

```text
Phase 9
Status: DRAFT
Implementation: LOCKED
Acceptance: NOT ACCEPTED
```

Bu aşamada kod yazılmayacaktır.
