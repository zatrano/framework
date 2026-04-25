<div align="center">

# ZATRANO

**Enterprise Seviye Go + Fiber Web Uygulama Şablonu**

[![Go](https://img.shields.io/badge/Go-1.26.1-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![Fiber](https://img.shields.io/badge/Fiber-v3-00ACD7?style=flat-square&logo=gofiber)](https://gofiber.io)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791?style=flat-square&logo=postgresql)](https://postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-6+-DC382D?style=flat-square&logo=redis)](https://redis.io)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

</div>

---

## Genel Bakış

ZATRANO, Go dilinde ve **Fiber v3** üzerinde inşa edilmiş, kurumsal web uygulamalarını hızlıca ayağa kaldırmak için tasarlanmış bir şablon projedir. Bu repo, hem server-rendered web/panel tarafını hem de JWT tabanlı REST API katmanını aynı uygulama içinde sunar.

Bu proje, aşağıdaki ihtiyaçlara cevap verir:

- Kurumsal kullanıcı yönetimi
- JWT tabanlı API kimlik doğrulama
- Google OAuth entegrasyonu
- PostgreSQL + GORM tabanlı veri modeli
- Redis destekli session, rate limiting ve cache
- Prometheus observability metrikleri
- Sentry hata izleme
- CSRF koruması, güvenlik başlıkları ve input sanitization
- Çoklu dil (TR/EN) destekli web arayüzü
- Async mail queue ve e-posta gönderimi
- Admin paneli ve genel kullanıcı paneli
- Hem unit hem integration test altyapısı

---

## Mimari Özeti

Uygulamanın çalışması şu adımlarla ilerler:

1. `.env` veya çevresel değişkenler okunur
2. Logger ve log seviyeleri konfigüre edilir
3. Config doğrulamaları yapılır (`validateconfig`)
4. Sentry hata raporlama başlatılır
5. Prometheus metrikleri kaydedilir
6. i18n çoklu dil altyapısı başlatılır
7. PostgreSQL ve Redis bağlantıları açılır
8. Session yönetimi başlatılır
9. Dosya yükleme ayarları yapılandırılır
10. Fiber uygulaması oluşturulur, template motoru bağlanır
11. Dependency Injection container oluşturulur
12. Asenkron mail kuyruğu başlatılır
13. Global middleware zinciri uygulanır
14. Statik içerik, route grupları ve error handler eklenir

### Uygulama Katmanları

- `main.go` — uygulama başlangıcı ve tüm altyapının hattını döşer
- `app/container.go` — servis/repository bağımlılıklarını oluşturur
- `configs/` — ortam, logging, database, session, Redis, CSRF, dosya, validasyon yapılandırmaları
- `handlers/` — web ve API handler'ları
- `middlewares/` — güvenlik ve operasyonel middleware'ler
- `models/` — GORM veri modelleri
- `repositories/` — database erişim katmanı
- `services/` — iş mantığı, erişim kontrolleri, JWT, mail
- `routes/` — web/panel/auth/website route kayıtları
- `api/v1/` — REST API route ve handlerları
- `observability/` — Prometheus metrikleri
- `packages/` — destek servisleri (flash, renderer, requestid, mail queue, i18n, jwtclaims, vb.)

---

## Öne Çıkan Özellikler

### Güvenlik ve Kimlik Doğrulama

- Oturum tabanlı web auth ve admin/panel ayrımı
- `authMiddleware`, `guestMiddleware`, `userTypeMiddleware`
- JWT tabanlı API auth
- Token yenileme (refresh token) akışı
- Google OAuth login/callback altyapısı
- CSRF koruması web formları için
- Güvenlik başlıkları (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, vb.)
- CORS yapılandırması production ve development için ayrı
- Rate limiting: global, form POST, login, API

### Web & Panel Uç Noktaları

- `/auth` — giriş, kayıt, şifre sıfırlama, e-posta doğrulama
- `/dashboard` — admin paneli, kullanıcı tipi 1 için yönetim ekranları
- `/panel` — normal kullanıcı paneli
- `/iletisim` — iletişim formu

### REST API

- `/api/v1/auth/login` — JWT erişim ve refresh token üretimi
- `/api/v1/auth/register` — kullanıcı kaydı
- `/api/v1/auth/verify-email` — e-posta doğrulama
- `/api/v1/auth/resend-verification` — doğrulama e-postasını yeniden gönderme
- `/api/v1/auth/forgot-password` — şifre sıfırlama bağlantısı üretimi
- `/api/v1/auth/reset-password` — token ile şifre sıfırlama
- `/api/v1/auth/refresh` — refresh token ile yeni access token
- `/api/v1/auth/me` — oturumlu kullanıcının profili
- `/api/v1/auth/logout` — access/refresh token iptali (revoke)
- `/api/v1/user/profile` — kullanıcı profili güncelleme
- `/api/v1/user/password` — şifre değiştirme
- `/api/v1/admin/users` — admin kullanıcı CRUD

### Altyapı ve Operasyonellik

- PostgreSQL + GORM
- Redis session ve rate limiter
- Prometheus metrikleri `/metrics`
- Sağlık kontrolleri `/health`, `/readyz`
- Async mail queue + retry/backoff
- Template engine ile server-side render
- Dosya servisleri ve yükleme klasörü yönetimi
- Mutlak URL, port, TLS ve host yapılandırması

---

## Uygulama Akışı Detayı

### 1. Ortam ve Yapılandırma

`main.go` içinde `envconfig.Load()` çağrısı ile `.env` dosyası okunur. `APP_ENV` production değilse ek olarak `LoadIfDev()` dev modda çevreyi yükler.

`validateconfig.ValidateAll()` ve production için `validateconfig.ValidateProduction()` ile ortam değişkenleri fail-fast olarak doğrulanır.

### 2. Logger ve Sentry

`logconfig.InitLogger()` ve `sentrylog.Init(version)` uygulama içi loglama ve hata izleme altyapısını başlatır.

### 3. Veritabanı ve Redis

- `databaseconfig.InitDB()` PostgreSQL bağlantısını GORM ile başlatır
- `redisconfig.InitRedis()` Redis istemcisini hazırlar
- `sessionconfig.InitSession()` Fiber oturum yönetimini konfigüre eder

### 4. Mail Kuyruğu

`mailqueue.Init(container.Mail, 3, 200)` ile bellek içi asenkron e-posta kuyruğu başlatılır. Her işçi (`worker`) başarılı gönderim, hata ve yeniden deneme (backoff) yönetir.

### 5. Fiber ve Template Motoru

- Fiber config: `IdleTimeout=60s`, `ReadTimeout=30s`, `BodyLimit=10MB`
- Ters proxy desteği: `TrustProxy=true` ve `CF-Connecting-IP`
- Template engine: `html.New("./views", ".html")`
- `t()` template helper ile çeviri fonksiyonu, `getFlashMessages()` ile flash mesajları destekler
- `Reload(true)` development modunda anlık template reload sağlar

### 6. Error Handler

- API isteklerinde JSON hata cevabı
- HTML isteklerinde `website/error` template'i
- 5xx hatalar Sentry'ye raporlanır
- Sağlık, readiness ve metrics endpoint'leri özel olarak ele alınır

### 7. Global Middleware Zinciri

Sıralama:

1. `recover.New()` — panik yakalama
2. `requestid.Middleware()` — request ID
3. `sentrylog.Middleware()` — Sentry context entegrasyonu
4. `observability.Middleware()` — Prometheus ölçümleri
5. `middlewares.ZapLogger()` — structured request logging
6. `i18n.Middleware()` — dil tespiti
7. `middlewares.SecurityHeaders()` — güvenlik başlıkları
8. `middlewares.InputSanitizer()` — form input tarama
9. `middlewares.Compress()` — gzip/brotli sıkıştırma

### 8. Statik çerik ve Upload

- `public/` altı statik dosyalar için
- `/uploads/*` route'u `fileconfig.Config.BasePath` üzerinden dosya servisler

### 9. Method Override

`_method` form alanı ile POST istekleri PUT/DELETE/PATCH olarak işlendirilebilir.

### 10. Rotasyon

`routes.SetupRoutes` uygulama web rotalarını kaydeder:

- `registerAuthRoutes`
- `registerDashboardRoutes`
- `registerPanelRoutes`
- `registerWebsiteRoutes`

`api/v1/routes.SetupAPIRoutes` REST API rotalarını aşağıdaki ek middleware'lerle kaydeder:

- `middlewares.CORS()`
- `middlewares.APIRateLimit()`
- `middlewares.ContentTypeEnforcer("application/json")`
- `middlewares.RequestSizeLimiter(10 * 1024 * 1024)`

---

## Route Haritası

### Web / Auth

- `GET /auth/login`
- `POST /auth/login`
- `GET /auth/logout`
- `GET /auth/profile`
- `POST /auth/profile/update-password`
- `POST /auth/profile/update-info`
- `GET /auth/register`
- `POST /auth/register`
- `GET /auth/forgot-password`
- `POST /auth/forgot-password`
- `GET /auth/reset-password`
- `POST /auth/reset-password`
- `GET /auth/verify-email`
- `GET /auth/resend-verification`
- `POST /auth/resend-verification`
- `GET /auth/oauth/:provider/login`
- `GET /auth/oauth/:provider/callback`

### Yönetici Paneli

- `GET /dashboard` (ana sayfa)
- `GET /dashboard/user-types`
- `GET /dashboard/user-types/create`
- `POST /dashboard/user-types/create`
- `GET /dashboard/user-types/update/:id`
- `POST /dashboard/user-types/update/:id`
- `DELETE /dashboard/user-types/delete/:id`
- `GET /dashboard/countries`
- `GET /dashboard/countries/create`
- `POST /dashboard/countries/create`
- `GET /dashboard/countries/update/:id`
- `POST /dashboard/countries/update/:id`
- `DELETE /dashboard/countries/delete/:id`
- `GET /dashboard/cities`
- `GET /dashboard/cities/create`
- `POST /dashboard/cities/create`
- `GET /dashboard/cities/update/:id`
- `POST /dashboard/cities/update/:id`
- `DELETE /dashboard/cities/delete/:id`
- `GET /dashboard/districts`
- `GET /dashboard/districts/create`
- `POST /dashboard/districts/create`
- `GET /dashboard/districts/update/:id`
- `POST /dashboard/districts/update/:id`
- `DELETE /dashboard/districts/delete/:id`
- `GET /dashboard/addresses`
- `GET /dashboard/addresses/create`
- `POST /dashboard/addresses/create`
- `GET /dashboard/addresses/update/:id`
- `POST /dashboard/addresses/update/:id`
- `DELETE /dashboard/addresses/delete/:id`
- `GET /dashboard/contact-messages`
- `GET /dashboard/contact-messages/:id`
- `GET /dashboard/users`
- `GET /dashboard/users/create`
- `POST /dashboard/users/create`
- `GET /dashboard/users/update/:id`
- `POST /dashboard/users/update/:id`
- `DELETE /dashboard/users/delete/:id`

### Kullanıcı Paneli

- `GET /panel/anasayfa`

### Website

- `GET /iletisim`
- `POST /iletisim`

### API

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/verify-email`
- `POST /api/v1/auth/resend-verification`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`
- `POST /api/v1/auth/refresh`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/logout`
- `GET /api/v1/user/profile`
- `PUT /api/v1/user/profile`
- `PUT /api/v1/user/password`
- `GET /api/v1/admin/users`
- `GET /api/v1/admin/users/:id`
- `POST /api/v1/admin/users`
- `PUT /api/v1/admin/users/:id`
- `DELETE /api/v1/admin/users/:id`

---

## Core Servisler ve Bağımlılıklar

### Dependency Injection Container

`app/container.go` şu servisleri sunar:

- `Mail` — mail servisi
- `JWT` — JWT token servisleri
- `Auth` — auth servisleri
- `Address`, `City`, `Contact`, `Country`, `Definition`, `District`, `User`, `UserType` — domain servisleri
- `UserRepo`, `CountryRepo`, `CityRepo`, `DistrictRepo`, `AddressRepo` — repositoryler

### Servis Katmanı

Her domain servis için bir interface vardır:

- `IAuthService`
- `IUserService`
- `IUserTypeService`
- `ICountryService`
- `ICityService`
- `IDistrictService`
- `IAddressService`
- `IContactService`
- `IDefinitionService`
- `IJWTService`
- `IMailService`

Bu sayede unit testlerde mock ve bağımlılık yerleştirme kolaydır.

### Repository Katmanı

Tüm repositoryler `repositories/` altında tanımlıdır ve GORM kullanır:

- `IAuthRepository`
- `IUserRepository`
- `IUserTypeRepository`
- `ICountryRepository`
- `ICityRepository`
- `IDistrictRepository`
- `IAddressRepository`
- `IContactRepository`
- `IDefinitionRepository`

`repositories/base_repository.go` generic bir base repository sağlar.

---

## Başlangıç Rehberi

### Ön koşullar

| Gereksinim | Açıklama |
|------------|----------|
| **Go** | `go.mod` ile uyumlu sürüm (ör. 1.25+); `go version` ile kontrol edin. |
| **PostgreSQL** | Çalışan bir instance; `DB_*` değişkenleri buna göre. |
| **Redis** | Oturum, rate limit ve cache için; `REDIS_*` değişkenleri. |
| **Make** (isteğe bağlı) | `Makefile` kısayolları için; yoksa aşağıdaki `go run` komutlarını kullanın. |

### 1. Repoyu klonlayın

```bash
git clone https://github.com/zatrano/framework.git
cd framework
```

### 2. Ortam dosyasını oluşturun

Kök dizinde `env.example` şablonu vardır; kopyalayıp `.env` yapın:

```bash
# Linux / macOS
cp env.example .env

# Windows (PowerShell)
Copy-Item env.example .env
```

> **Not:** `FILE_BASE_PATH` gibi alanlarda aynı satıra `# yorum` yazmayın; değer ile yorumu ayrı satırlara bölün. Boş bırakılırsa uygulama varsayılan olarak `./uploads` kullanır.

### 3. `.env` içinde zorunlu / kritik alanlar

Aşağıdakileri kendi ortamınıza göre doldurun (tam liste `env.example` içinde):

- **Uygulama:** `APP_ENV`, `APP_HOST`, `APP_PORT`, `APP_BASE_URL` (üretimde `APP_BASE_URL` zorunlu doğrulamaya girer)
- **Veritabanı:** `DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`, `DB_DATABASE`, `DB_SSL_MODE`, `DB_TIMEZONE`
- **Redis:** `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`
- **Güvenlik:** `JWT_SECRET` (yeterince uzun, rastgele)
- **E-posta (opsiyonel):** `MAIL_HOST`, `MAIL_PORT`, `MAIL_USERNAME`, `MAIL_PASSWORD`
- **Turnstile / OAuth (kullanacaksanız):** `TURNSTILE_*`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI`

PostgreSQL ve Redis’in **çalışıyor** ve `.env`’deki host/port/şifrelerin **doğru** olduğundan emin olun.

### 4. Go bağımlılıkları

```bash
go mod download
# veya
make tidy
```

### 5. Migrasyon (şema)

Şema [GORM AutoMigrate](https://gorm.io/docs/migration.html) ile `database/migrations/schema.go` üzerinden güncellenir; tablolar yoksa oluşturulur.

**Make kullanıyorsanız:**

```bash
make migrate
```

**Düz `go` ile:**

```bash
go run ./database/cmd/main.go -migrate
```

> Bayrak yokken `go run ./database/cmd/main.go` sadece DB’ye bağlanır; `-migrate` ve/veya `-seed` verilmedikçe şema/seed çalışmaz (logda bilgi mesajı yazar).

### 6. Seed (örnek / başlangıç verisi)

Kullanıcı tipleri, admin kullanıcı, ülke/şehir/ilçe örnekleri, `definitions` seeder’ı vb. `database/seeders` içinde tanımlıdır; detay `seed_all.go` sırasıyla çalışır.

**Make:**

```bash
make seed
```

**Düz `go` ile:**

```bash
go run ./database/cmd/main.go -seed
```

### 7. İlk kurulum: migrasyon + seed birlikte

**Make:**

```bash
make migrate-seed
```

**Düz `go` ile:**

```bash
go run ./database/cmd/main.go -migrate -seed
```

Bu komut **tek transaction** içinde önce migrasyon, sonra tüm seeder’ları çalıştırır. Üretimde seed’i yalnızca bilinçli olarak çalıştırın.

### 8. Uygulamayı çalıştırma

**Make (Linux/macOS; `APP_ENV` ayarlanır):**

```bash
make dev
```

**Düz `go` (tüm platformlar):**

```bash
go run .
```

Geliştirme için genelde `APP_ENV=development` kullanılır. **Windows PowerShell** örneği:

```powershell
$env:APP_ENV="development"; go run .
```

**Üretim** için önce derleyip binary çalıştırın; örnek:

```bash
make build
./bin/zatrano
# veya: go build -o zatrano . && ./zatrano
```

### 9. Hızlı kontrol listesi (sıra)

1. PostgreSQL + Redis ayakta  
2. `env.example` → `.env`, değerler dolduruldu  
3. `go mod download` veya `make tidy`  
4. `make migrate` veya `go run ./database/cmd/main.go -migrate`  
5. `make seed` veya `go run ./database/cmd/main.go -seed` (veya 4–5 için `make migrate-seed`)  
6. `make dev` veya `go run .`  
7. Tarayıcı: `http://127.0.0.1:3000` (port `APP_PORT` ile aynı olmalı)

### 10. Uç noktalar (çalıştırdıktan sonra)

- Ana uygulama: `http://127.0.0.1:3000` (veya `APP_BASE_URL`)  
- Sağlık: `http://127.0.0.1:3000/health`  
- Hazırlık: `http://127.0.0.1:3000/readyz`  
- Metrik: `http://127.0.0.1:3000/metrics`  

`APP_PORT` varsayılanı `3000`’dir; `.env` ile değişir.

---

## `Makefile` ve eşdeğer `go` / CLI komutları

İlk kurulum adımları için **Başlangıç Rehberi** bölümüne bakın. Aşağıda `make` hedeflerinin **Make kullanmadan** karşılıkları verilmiştir. Tam `LDFLAGS`, çapraz derleme (`GOOS`/`GOARCH`) ve sürüm enjeksiyonu için `Makefile` içine bakın.

| Make hedefi | Go / komut satırı eşdeğeri |
|-------------|----------------------------|
| `make help` | `Makefile` içinde `grep -E '^## '` (Windows’ta hedef adlarını `Makefile`’dan okuyun) |
| `make tidy` | `go mod tidy` ve ardından `go mod verify` |
| `make dev` | Linux/macOS: `APP_ENV=development go run .` — **Windows (PowerShell):** `$env:APP_ENV="development"; go run .` |
| `make build` | `mkdir -p bin` (Windows: `New-Item -ItemType Directory -Force bin`) sonra `go build -trimpath -o bin/zatrano .` (Windows çıktı: `bin\zatrano.exe`) — üretimde `Makefile` ayrıca `-ldflags` ve `linux/amd64` sabitler |
| `make run` | `make build` + `./bin/zatrano` (Windows: `.\bin\zatrano.exe`) veya: `go run .` |
| `make migrate` | `go run ./database/cmd/main.go -migrate` |
| `make seed` | `go run ./database/cmd/main.go -seed` |
| `make migrate-seed` | `go run ./database/cmd/main.go -migrate -seed` |
| `make clean` | `rm -rf ./bin` — Windows: `Remove-Item -Recurse -Force bin -ErrorAction SilentlyContinue` (varsa `coverage.out`, `coverage.html` da silinir) |
| `make test-unit` | `go test -v -race -count=1 ./tests/unit/...` |
| `make test` | `go test -race -coverprofile=coverage.out -covermode=atomic ./tests/unit/...` sonra `go tool cover -func=coverage.out` / `-html=coverage.html` |
| `make test-race` | `go test -race -count=3 ./tests/unit/...` |
| `make test-integration` | Aynı: `docker-compose -f docker-compose.test.yml up --abort-on-container-exit --build` (Go ile değil, Docker) |
| `make vet` | `go vet ./...` |
| `make lint` | `golangci-lint` kuruluysa: `golangci-lint run --timeout=5m ./...` (Make’e bağlı) |
| `make docker-build` | `docker build` (Makefile’daki `VERSION` / `--build-arg` için `Makefile`’a bakın) |

**Özet (sık kullanılan `go` komutları):**

```bash
# Modül
go mod tidy
go mod verify
go mod download

# Uygulama
go run .
go build -o zatrano .

# Veritabanı aracı (migrate / seed)
go run ./database/cmd/main.go -migrate
go run ./database/cmd/main.go -seed
go run ./database/cmd/main.go -migrate -seed

# Test
go test -v -race -count=1 ./tests/unit/...
go test -race -coverprofile=coverage.out ./tests/unit/...
go vet ./...
```

---

## Test ve Kalite Kontroller

### Birim Testler

`tests/unit/` servis testlerini barındırır.

**Make:**

```bash
make test-unit
```

**Sadece `go`:**

```bash
go test -v -race -count=1 ./tests/unit/...
```

### Entegrasyon Testler

`tests/integration/` Docker ile çalışır; `go test` yerine `docker-compose` kullanılır (Make de aynısını çağırır).

```bash
make test-integration
# veya
docker-compose -f docker-compose.test.yml up --abort-on-container-exit --build
docker-compose -f docker-compose.test.yml down -v
```

### Coverage (Make `test` hedefi eşdeğeri)

**Make:** `make test` (rapor `coverage.html`).

**Sadece `go`:**

```bash
go test -race -coverprofile=coverage.out -covermode=atomic ./tests/unit/...
go tool cover -func=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Kod Kalitesi

| İş | Make | `go` / araç |
|----|------|-------------|
| Race tekrar | `make test-race` | `go test -race -count=3 ./tests/unit/...` |
| Statik analiz (Go yerleşik) | `make vet` | `go vet ./...` |
| Linter | `make lint` | `golangci-lint run --timeout=5m ./...` (ayrı kurulum gerekir) |

```bash
make test-race
# veya
go test -race -count=3 ./tests/unit/...

go vet ./...
```

---

## Üretime Hazırlık

Önerilen üretim adımları:

1. `APP_ENV=production` olarak ayarlayın
2. `DB_SSL_MODE=require` veya `verify-full` kullanın
3. `JWT_SECRET` değerini güçlü bir anahtarla değiştirin
4. `CORS_ALLOWED_ORIGINS` ile domain filtreleme uygulayın
5. `REDIS_TLS=true` gerekiyorsa TLS yapılandırmasını tamamlayın
6. `COOKIE_DOMAIN` değerini doğru ayarlayın
7. `LOG_LEVEL=info` veya `warn` seçin
8. `SENTRY_DSN` ile hata izlemeyi etkinleştirin
9. `EMAIL` ve `MAIL` ayarlarını doldurun

---

## Operasyonel Gösterge Panelleri

- `GET /health` — uygulama ve DB/Redis sağlık durumu
- `GET /readyz` — readiness kontrolü
- `GET /metrics` — Prometheus toplama

### Prometheus Metrikleri

Toplanan metrikler:

- `http_requests_total`
- `http_request_duration_seconds`
- `http_requests_in_flight`
- `mail_queue_size`
- `db_query_duration_seconds`

---

## Genişletme Rehberi

### Yeni Web Route Ekleme

1. `handlers/` içinde yeni bir handler oluşturun
2. `services/` ve `repositories/` içinde gerekiyorsa interface ve implementasyon ekleyin
3. `app/container.go` içinde yeni servisi container'a kayıt edin
4. `routes/` içinde route grubunu ekleyin

### Yeni API Endpoint

1. `api/v1/handlers/` altına handler ekleyin
2. `api/v1/routes/api_routes.go` içinde route kaydını yapın
3. Gerekirse yeni middleware ekleyin (`JWTAuth`, `JWTTypeMiddleware`, `ContentTypeEnforcer`)

### Yeni Ortam Değişkeni

1. `env.example` içine ekleyin
2. `configs/` içinde ilgili config modülüne bağlayın
3. `validateconfig` içine doğrulama kuralı ekleyin

---

## Dosya ve Klasör Yapısı

- `api/v1/` — REST API handler ve route yapılandırmaları
- `app/` — dependency injection container
- `configs/` — çalışma zamanı yapılandırmaları
- `database/` — migrasyonlar ve seed verileri
- `handlers/` — web, panel ve auth handler'ları
- `middlewares/` — uygulama ara katmanları
- `models/` — GORM veri modelleri
- `observability/` — metrik ve monitoring desteği
- `packages/` — yardımcı paketler ve ortak altyapı
- `public/` — statik içerikler
- `repositories/` — veri erişim katmanı
- `requests/` — form/JSON request parsing ve validasyon
- `routes/` — route kayıtları
- `services/` — iş mantığı ve domain servisleri
- `tests/` — test senaryoları
- `views/` — HTML şablonları

---

## En Önemli Çalışma Akışları

### Kayıt Akışı

1. Kullanıcı `/auth/register` formunu doldurur
2. `AuthService.RegisterUser` hesap oluşturur ve e-posta doğrulama maili kuyruğa eklenir
3. Kullanıcı gelen link ile `/auth/verify-email` çalıştırır
4. Doğrulama başarılıysa kullanıcı login olabilir

### Giriş Akışı

1. Kullanıcı `/auth/login` formunu gönderir
2. `AuthService.Authenticate` kimlik bilgilerini doğrular
3. Başarılıysa session başlatılır ve yetkiye göre `/dashboard` veya `/panel/anasayfa` yönlendirilir

### JWT API Akışı

1. `POST /api/v1/auth/login` ile access ve refresh token alınır
2. `GET /api/v1/auth/me` ile tokenlı kullanıcı bilgileri alınır
3. `PUT /api/v1/user/password` ile şifre değiştirilebilir
4. `POST /api/v1/auth/refresh` ile yeni access token alınır
5. `POST /api/v1/auth/logout` ile access (ve opsiyonel refresh) token revoke edilir

### Mobil Auth Örnekleri

#### Login Request

```json
{
  "email": "user@example.com",
  "password": "secret123"
}
```

#### Login Response (200)

```json
{
  "data": {
    "access_token": "<ACCESS_TOKEN>",
    "refresh_token": "<REFRESH_TOKEN>",
    "token_type": "Bearer",
    "user": {
      "id": 12,
      "name": "Demo User",
      "email": "user@example.com",
      "user_type_id": 2
    }
  }
}
```

#### Logout Request

Header:

```text
Authorization: Bearer <ACCESS_TOKEN>
Content-Type: application/json
```

Body (opsiyonel refresh revoke):

```json
{
  "refresh_token": "<REFRESH_TOKEN>"
}
```

#### Logout Response (200)

```json
{
  "data": {
    "message": "Çıkış başarılı. Access ve varsa refresh token iptal edildi."
  }
}
```

## Auth Dokümantasyonu (Web + Mobil/API)

Bu bölüm, projedeki auth ile ilişkili tüm akışı ve fonksiyon envanterini tek yerde toplar. Web tarafı **session/cookie**, mobil/API tarafı **JWT (Bearer)** kullanır.

### Auth Modları

- **Web/Auth:** Form tabanlı, session ile oturum yönetimi, CSRF korumalı.
- **Mobil/API:** JSON + Bearer token, refresh token akışı, revoke/blacklist destekli.

### Web Auth Endpointleri (Session)

- `GET /auth/login` (`ShowLogin`) — giriş sayfası.
- `POST /auth/login` (`Login`) — kimlik doğrulama, session açma.
- `GET /auth/logout` (`Logout`) — session yok etme.
- `GET /auth/profile` (`Profile`) — oturum kullanıcısı profil ekranı.
- `POST /auth/profile/update-password` (`UpdatePassword`) — şifre güncelleme.
- `POST /auth/profile/update-info` (`UpdateInfo`) — ad/e-posta güncelleme.
- `GET /auth/register` (`ShowRegister`) — kayıt sayfası.
- `POST /auth/register` (`Register`) — kullanıcı kaydı + doğrulama maili kuyruğa ekleme.
- `GET /auth/forgot-password` (`ShowForgotPassword`) — şifre unuttum ekranı.
- `POST /auth/forgot-password` (`ForgotPassword`) — reset linki üretme/gönderme.
- `GET /auth/reset-password` (`ShowResetPassword`) — token ile reset ekranı.
- `POST /auth/reset-password` (`ResetPassword`) — yeni şifreyi kaydetme.
- `GET /auth/verify-email` (`VerifyEmail`) — e-posta doğrulama.
- `GET /auth/resend-verification` (`ShowResendVerification`) — doğrulama linki yeniden gönder formu.
- `POST /auth/resend-verification` (`ResendVerification`) — doğrulama mailini tekrar gönderme.
- `GET /auth/oauth/:provider/login` (`OAuthLogin`) — OAuth provider login yönlendirmesi.
- `GET /auth/oauth/:provider/callback` (`OAuthCallback`) — OAuth callback işleme.

### Mobil/API Auth Endpointleri (JWT)

- `POST /api/v1/auth/login` (`AuthAPIHandler.Login`) — `access_token` + `refresh_token`.
- `POST /api/v1/auth/register` (`AuthAPIHandler.Register`) — kayıt.
- `POST /api/v1/auth/verify-email` (`AuthAPIHandler.VerifyEmail`) — token ile e-posta doğrulama.
- `POST /api/v1/auth/resend-verification` (`AuthAPIHandler.ResendVerification`) — doğrulama maili tekrar gönderme.
- `POST /api/v1/auth/forgot-password` (`AuthAPIHandler.ForgotPassword`) — reset linki talebi.
- `POST /api/v1/auth/reset-password` (`AuthAPIHandler.ResetPassword`) — token ile şifre sıfırlama.
- `POST /api/v1/auth/refresh` (`AuthAPIHandler.Refresh`) — refresh ile yeni access token.
- `GET /api/v1/auth/me` (`AuthAPIHandler.Me`) — geçerli kullanıcının özeti.
- `POST /api/v1/auth/logout` (`AuthAPIHandler.Logout`) — access + opsiyonel refresh revoke.

İlgili kullanıcı endpointleri (JWT gerekli):

- `GET /api/v1/user/profile` (`UserAPIHandler.Profile`)
- `PUT /api/v1/user/profile` (`UserAPIHandler.UpdateProfile`)
- `PUT /api/v1/user/password` (`UserAPIHandler.ChangePassword`)

### Auth Function Envanteri (Atlanmayan Tam Liste)

`handlers/auth/auth_handler.go`:

- `NewAuthHandler`
- `getSessionUser`
- `destroySession`
- `ShowLogin`
- `Login`
- `ShowRegister`
- `Register`
- `Profile`
- `UpdatePassword`
- `ShowForgotPassword`
- `ForgotPassword`
- `ShowResetPassword`
- `ResetPassword`
- `UpdateInfo`
- `VerifyEmail`
- `ShowResendVerification`
- `ResendVerification`
- `Logout`
- `OAuthLogin`
- `OAuthCallback`
- `GoogleLogin`
- `GoogleCallback`

`api/v1/handlers/auth_api_handler.go`:

- `NewAuthAPIHandler`
- `Login`
- `Register`
- `Refresh`
- `VerifyEmail`
- `ResendVerification`
- `ForgotPassword`
- `ResetPassword`
- `Me`
- `Logout`

`api/v1/handlers/user_api_handler.go`:

- `NewUserAPIHandler`
- `Profile`
- `UpdateProfile`
- `ChangePassword`

`services/auth_service.go` (`IAuthService` + implementasyon):

- `Authenticate`
- `RegisterUser`
- `VerifyEmail`
- `ResendVerificationLink`
- `SendPasswordResetLink`
- `ResetPassword`
- `UpdatePassword`
- `GetUserProfile`
- `UpdateUserInfo`
- `FindOrCreateOAuthUser`
- Yardımcılar: `logAuthSuccess`, `logDBError`, `logWarn`, `generateToken`, `getUserByEmail`, `getUserByID`, `comparePasswords`, `hashPassword`, `enqueueVerificationEmail`, `enqueuePasswordResetEmail`

`services/jwt_service.go` (`IJWTService` + implementasyon):

- `GenerateToken`
- `GenerateRefreshToken`
- `ValidateToken`
- `RefreshAccessToken`
- `RevokeToken`
- `IsTokenRevoked`

`middlewares/jwt_auth.go`:

- `JWTAuth`
- `JWTClaimsFromFiber`
- `JWTTypeMiddleware`

`middlewares/auth.go`:

- `AuthMiddleware`

`packages/jwtrevoke/jwtrevoke.go`:

- `RevokeToken`
- `IsRevoked`

### Middleware ve Güvenlik Davranışı

- Web auth formları CSRF ile korunur (`csrfconfig.SetupCSRF`).
- API auth grubu CORS + API rate limit + JSON content-type zorlaması + body size limit ile çalışır.
- `JWTAuth` akışı: Authorization parse -> revoke blacklist kontrolü -> JWT imza/expiry doğrulama -> `Locals(userID)`.
- Logout sonrası tokenlar Redis blacklist’e yazılır; süresi dolana kadar tekrar kullanılamaz.

### Geliştirici Notları (Web vs Mobil)

- Web istemcisi `session cookie` taşır; API token kullanmaz.
- Mobil istemci `Authorization: Bearer <access_token>` taşır; session cookie kullanmaz.
- Refresh token güvenli depoda tutulmalı (Keychain/Keystore).
- Logout çağrısında `refresh_token` body ile verilirse iki token da revoke edilir.
- Refresh sonrası eski access tokenın kullanımını istemci tarafında bırakın; yeni access tokenı kullanın.

### Hata Modeli

- API hata gövdesi `packages/apierrors` standardına göre döner: `status`, `code`, `message`, opsiyonel `details`.
- Web tarafı hata/başarı geri bildirimleri `flashmessages` ile render edilir.

### Postman Collection

Mobil/API auth akışını doğrudan import etmek için koleksiyon:

- `docs/postman/zatrano-auth.postman_collection.json`

---

## İletişim ve Destek

- Geliştirici: **Serhan KARAKOÇ**
- GitHub: https://github.com/serhankarakoc
- LinkedIn: https://linkedin.com/in/serhankarakoc

---

## Lisans

Bu proje **MIT Lisansı** ile lisanslanmıştır.
