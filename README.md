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
- `/api/v1/auth/refresh` — refresh token ile yeni access token
- `/api/v1/auth/me` — oturumlu kullanıcının profili
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

- `GET /dashboard/home`
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
- `POST /api/v1/auth/refresh`
- `GET /api/v1/auth/me`
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

### Gerekli Adımlar

1. Reponun klonlanması

```bash
git clone https://github.com/zatrano/framework.git
cd zatrano
```

2. Ortam dosyasının oluşturulması

```bash
cp .env.example .env
```

3. `.env` içinde aşağıdaki temel değişkenleri düzenleyin:

- `APP_ENV`
- `APP_HOST`
- `APP_PORT`
- `APP_BASE_URL`
- `DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`, `DB_DATABASE`, `DB_SSL_MODE`
- `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `REDIS_DB`, `REDIS_TLS`
- `JWT_SECRET`
- `MAIL_HOST`, `MAIL_PORT`, `MAIL_USERNAME`, `MAIL_PASSWORD` (SMTP için)
- `TURNSTILE_SITE_KEY`, `TURNSTILE_SECRET_KEY` (Captcha için)
- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI`

4. Bağımlılıkları indirme ve doğrulama

```bash
make tidy
```

5. Veritabanı migrasyonları

```bash
make migrate
```

6. Seed verisi yükleme

```bash
make seed
```

7. Geliştirme sunucusunu çalıştırma

```bash
make dev
```

8. Tarayıcıdan erişim

- `http://127.0.0.1:3000`
- `http://127.0.0.1:3000/health`
- `http://127.0.0.1:3000/metrics`

---

## `Makefile` Komutları

- `make help` — tüm komutları gösterir
- `make tidy` — modülleri düzenler ve doğrular
- `make dev` — geliştirme modunda çalıştırır
- `make build` — production binary derler
- `make run` — derler ve çalıştırır
- `make migrate` — DB migrasyonlarını uygular
- `make seed` — seed verilerini yükler
- `make migrate-seed` — migrasyon + seed
- `make clean` — build artefaktlarını temizler
- `make test-unit` — unit test çalıştırır
- `make test-integration` — docker destekli integration test
- `make test` — unit test + coverage
- `make test-race` — race condition testi
- `make lint` — golangci-lint çalıştırır
- `make docker-build` — Docker image üretir

---

## Test ve Kalite Kontroller

### Birim Testler

`tests/unit/` klasöründe servis unit testleri bulunur. Aşağıdaki komutla run edilir:

```bash
make test-unit
```

### Entegrasyon Testler

`tests/integration/` içinde Docker destekli testler vardır. Aşağıdaki komut çalıştırılır:

```bash
make test-integration
```

### Kod Kalitesi

Linter ve race testleri:

```bash
make lint
make test-race
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
3. Başarılıysa session başlatılır ve yetkiye göre `/dashboard/home` veya `/panel/anasayfa` yönlendirilir

### JWT API Akışı

1. `POST /api/v1/auth/login` ile access ve refresh token alınır
2. `GET /api/v1/auth/me` ile tokenlı kullanıcı bilgileri alınır
3. `PUT /api/v1/user/password` ile şifre değiştirilebilir
4. `POST /api/v1/auth/refresh` ile yeni access token alınır

---

## İletişim ve Destek

- Geliştirici: **Serhan KARAKOÇ**
- GitHub: https://github.com/serhankarakoc
- LinkedIn: https://linkedin.com/in/serhankarakoc

---

## Lisans

Bu proje **MIT Lisansı** ile lisanslanmıştır.
