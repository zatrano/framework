# Phase 9 — Sınırlandırıcı SPEC Taslağı

**Durum:** Draft — Implementation kapalı  
**Önkoşul:** Phase 8 frozen (`v2.0.22`)  
**Kapsam:** Dry-run, CLI acquisition, Acquisition ↔ Enablement

---

## 1. Amaç

Phase 9, Phase 8'de dondurulan module acquisition yüzeyinin üzerine üç ayrı kullanıcı/operasyon sözleşmesi eklemeyi değerlendirir:

1. **Dry-run** — acquisition'ın ne yapacağını mutation yapmadan göstermek.
2. **CLI acquisition** — acquisition yeteneğini açık ve deterministik CLI komutlarıyla sunmak.
3. **Acquisition ↔ enablement** — module acquisition ile ZATRANO package enablement arasındaki ilişkiyi açıkça tanımlamak.

Bu fazın amacı Phase 8'i genişletmek veya yeniden tasarlamak değildir.

---

# Contract A — Dry-run

## A1. Amaç

Dry-run, belirli bir acquisition isteğinin **ne yapacağını gösterebilir**, fakat herhangi bir module mutation gerçekleştiremez.

Dry-run yalnızca acquisition planının ve beklenen execution işlemlerinin görünürleştirilmesidir.

## A2. Mutation yasağı

Dry-run:

* `go get` çalıştırmaz.
* `go mod tidy` çalıştırmaz.
* `go.mod` yazmaz.
* `go.sum` yazmaz.
* `SnapshotFiles` / `RecoverFiles` çalıştırmaz.
* package enablement state değiştirmez.
* `package:install` çalıştırmaz.
* filesystem üzerinde acquisition mutation yapmaz.

Dry-run sonucunun oluşması, acquisition'ın uygulanmış olduğu anlamına gelmez.

## A3. Resolver sınırı

Dry-run kendi resolver'ını oluşturmaz.

Dry-run:

* kendi semver karşılaştırmasını yapmaz.
* `latest` için alternatif çözümleme algoritması uygulamaz.
* Registry resolution mantığını kopyalamaz.
* Phase 7 `GoGetArg` üretimini yeniden uygulamaz.

Dry-run mevcut acquisition planını tüketir.

## A4. Kaynak

Dry-run mümkün olan en erken aşamada **concrete acquisition target** üzerinden çalışmalıdır.

Örneğin:

```text
github.com/zatrano/packages@main
github.com/zatrano/packages@v1.0.0
github.com/zatrano/framework/v2@v2.0.22
```

`latest` dry-run tarafından bağımsız bir version-selection mekanizmasına dönüştürülmez.

## A5. Gösterilecek bilgi

Dry-run en azından:

* module path
* requested/concrete version
* Go acquisition argument
* hedef module root
* execution'ın yapılacağı komut biçimi

bilgisini gösterebilir.

Ancak dry-run çıktısı **"installed", "acquired", "applied"** gibi mutation gerçekleşmiş izlenimi veren durumlar kullanmamalıdır.

## A6. Gerçek execution ayrımı

Dry-run ile execution aynı planı tüketebilir; fakat execution boundary çağrılmaz.

```text
Plan
  ├── Dry-run → report only
  │
  └── Execute → mutation
```

Dry-run'ın amacı execution'ın simülasyonu değil, **execution planının güvenli şekilde gösterilmesidir**.

## A7. Phase 8 koruması

Dry-run eklemek:

* `Execute` semantiğini değiştirmez.
* `ExecuteTargets` semantiğini değiştirmez.
* `Inspect` semantiğini değiştirmez.
* `ApplyResult` semantiğini değiştirmez.
* Recovery davranışını değiştirmez.

---

# Contract B — CLI Acquisition

## B1. Amaç

CLI acquisition, kullanıcıya module acquisition işlemini açık bir komut yüzeyi olarak sunar.

CLI'nin görevi orchestration'dır; acquisition algoritmasının ikinci sahibi değildir.

## B2. CLI sorumlulukları

CLI:

1. kullanıcı girdisini alır,
2. gerekli resolution/plan akışını mevcut API'ler üzerinden çağırır,
3. acquisition sonucunu kullanıcıya aktarır.

CLI kendi:

* semver resolver'ını,
* registry resolver'ını,
* module normalization algoritmasını,
* conflict resolution algoritmasını

oluşturmaz.

## B3. Resolution sınırı

CLI:

```text
user input
    ↓
existing resolution/plan APIs
    ↓
concrete acquisition target
    ↓
execution
```

şeklinde çalışmalıdır.

CLI içinde:

* `compareSemver`
* `latestCompatible`
* alternatif `Resolve`
* module pin seçimi

gibi ikinci resolution mekanizmaları bulunamaz.

## B4. Dry-run ilişkisi

CLI acquisition dry-run destekleyebilir.

Ancak:

```text
acquisition --dry-run
```

ile gerçek acquisition birbirinden davranışsal olarak ayrılmalıdır.

Dry-run:

* process çalıştırmaz,
* mutation yapmaz.

Normal acquisition:

* mevcut Phase 8 execution yüzeyini kullanır.

CLI, dry-run için ayrı bir acquisition implementation oluşturmaz.

## B5. `package:install` ayrımı

CLI acquisition ile mevcut:

```text
package:install
```

aynı operasyon değildir.

`package:install` mevcut anlamıyla:

> **package enablement**

olarak kalır.

CLI acquisition ise:

> **Go module acquisition**

işlemidir.

İki komutun kullanıcı açısından benzer görünmesi, semantik olarak birleştirilmelerini gerektirmez.

## B6. CLI'nin mutation sınırı

CLI kendi başına:

* `go.mod` düzenlemez,
* `go.sum` düzenlemez,
* shell komutu oluşturmaz,
* `go get` invocation implementation'ı yazmaz.

Acquisition mevcut process/execution boundary üzerinden gerçekleştirilir.

## B7. Çıktı

CLI:

* selected/concrete target
* execution sonucu
* success/failure
* partial result
* recovery durumu

gibi acquisition API'lerinin sağladığı bilgileri gösterebilir.

CLI bu sonuçları yeniden yorumlayıp farklı bir acquisition state modeli yaratmamalıdır.

---

# Contract C — Acquisition ↔ Enablement

## C1. Temel ayrım

Bu fazın en önemli sözleşmesi:

> **Acquisition ≠ Enablement**

Bir Go module'ünün başarılı şekilde acquire edilmesi, package'ın otomatik olarak enabled olduğu anlamına gelmez.

## C2. Acquisition

Acquisition'ın sorumluluğu:

```text
Go module
    ↓
go.mod / go.sum dependency state
```

ile sınırlıdır.

Acquisition:

* package enablement state değiştirmez.
* `bootstrap/enabled.go` değiştirmez.
* package registration state değiştirmez.

## C3. Enablement

Enablement'ın sorumluluğu:

```text
package capability
    ↓
enabled/imported state
    ↓
runtime registration
```

alanındadır.

Bu mevcut `package:install` semantiğinin parçasıdır.

## C4. Otomatik enablement yasağı

Başarılı acquisition sonrasında:

```text
go get package
        ↓
automatic enablement
```

olmayacaktır.

Özellikle:

* acquisition success → enablement success değildir.
* acquisition failure → enablement state'i değiştirmez.
* partial acquisition → enablement işlemi başlatmaz.

## C5. `package:install`

`package:install`:

**enablement command** olarak kalır.

Phase 9:

* `package:install` anlamını değiştirmez.
* acquisition command'a dönüştürmez.
* `go get` çalıştırmasını zorunlu hale getirmez.
* acquisition ile enablement'ı tek command altında gizlice birleştirmez.

## C6. Gelecekteki bağlantı

Eğer ileride acquisition ve enablement tek kullanıcı workflow'unda birleştirilmek istenirse, bunun ayrıca açık bir contract'a ihtiyacı vardır.

Örneğin potansiyel workflow:

```text
Acquire module
      ↓
Verify acquisition
      ↓
Enable package
```

ancak bu üç adımın aynı transaction olduğu anlamına gelmez.

Her adımın kendi başarısızlık ve recovery semantiği ayrıca tanımlanmalıdır.

## C7. Failure isolation

Acquisition başarısız olduğunda:

* enablement otomatik başlamaz.

Enablement başarısız olduğunda:

* daha önce başarılı olmuş module acquisition otomatik olarak rollback edilmiş sayılmaz.

Bu iki state birbirinden bağımsızdır.

---

# 4. Phase 8 Freeze Boundary

Phase 9 aşağıdaki Phase 8 yüzeyini değiştiremez:

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

Aşağıdakiler özellikle yasaktır:

* `func Apply`
* Phase 7 API değişikliği
* `GoGetArg` değişikliği
* yeni resolver
* ikinci process abstraction
* `Execute` semantiğini değiştirme
* `ExecuteTargets` semantiğini değiştirme
* `Inspect` semantiğini değiştirme
* recovery semantics değiştirme
* `package:install` semantiğini değiştirme.

---

# 5. Lockfile ve Tidy Boundary

Phase 9:

* `zatrano.lock` oluşturmaz.
* `go.mod` / `go.sum` dışında acquisition state dosyası oluşturmaz.
* `go mod tidy`yi acquisition workflow'una otomatik olarak eklemez.
* `tidy`yi dependency resolver olarak kullanmaz.

---

# 6. Contract Independence

Üç sözleşme birbirine bağımlı özellikler olarak tasarlanmayacaktır.

```text
Dry-run
   │
   └── acquisition plan visibility

CLI acquisition
   │
   └── user-facing acquisition orchestration

Acquisition ↔ Enablement
   │
   └── state/semantic boundary
```

Bir contract'ın implementation'ı diğer contract'ın semantiğini kendiliğinden değiştiremez.

Özellikle:

* Dry-run → enablement yapmaz.
* CLI acquisition → `package:install` anlamını değiştirmez.
* Enablement → Phase 8 acquisition implementation'ını değiştirmez.

---

# 7. Implementation Gate

Bu SPEC kabul edilmeden:

* dry-run implementation yok,
* yeni CLI acquisition command yok,
* acquisition ↔ enablement bağlantısı yok,
* yeni API freeze yok,
* mevcut Phase 8 API değişikliği yok.

SPEC kabulünden sonra implementation sırası ayrıca belirlenecektir.

Önerilen sıra:

```text
Contract A — Dry-run
        ↓
Contract tests
        ↓
Implementation

Contract B — CLI acquisition
        ↓
Contract tests
        ↓
Implementation

Contract C — Acquisition ↔ enablement
        ↓
Contract tests
        ↓
Implementation
```

Bir contract tamamlanmadan diğerinin implementation'ı onun davranışına varsayılan bağımlılık eklememelidir.

---

# 8. Acceptance Criteria

Phase 9 SPEC'i ancak şu sınırlar açıkça kabul edildiğinde implementation'a açılır:

* [ ] Dry-run mutation yapmaz.
* [ ] Dry-run resolver değildir.
* [ ] CLI acquisition resolver değildir.
* [ ] CLI acquisition process boundary'yi yeniden implement etmez.
* [ ] Acquisition ve enablement ayrı state'lerdir.
* [ ] Acquisition success otomatik enablement değildir.
* [ ] `package:install` enablement olarak kalır.
* [ ] Phase 8 API'leri değişmez.
* [ ] `func Apply` eklenmez.
* [ ] Phase 7 API'leri değişmez.
* [ ] `GoGetArg` değişmez.
* [ ] `go mod tidy` otomatik değildir.
* [ ] `zatrano.lock` yoktur.
* [ ] Üç contract birbirinden bağımsızdır.

**Phase 9 implementation kapısı: SPEC kabulü.**

SPEC kabul edilene kadar **kod yoktur**.
