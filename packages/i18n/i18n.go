// Package i18n — çoklu dil altyapısı.
// Fiber v3 context'inden dil tespiti yapar (Accept-Language başlığı,
// cookie veya query parametresi). Çeviriler embed edilmiş JSON dosyalarından yüklenir.
// Hiçbir harici bağımlılık gerektirmez.
package i18n

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/zatrano/framework/configs/logconfig"

	"github.com/gofiber/fiber/v3"
)

// Desteklenen diller
const (
	LangTR = "tr"
	LangEN = "en"

	DefaultLang    = LangTR
	CookieName     = "lang"
	QueryParamName = "lang"
	LocalsKey      = "lang"
)

var supportedLangs = map[string]bool{
	LangTR: true,
	LangEN: true,
}

// translations — dil → anahtar → çeviri metni
var (
	translations = map[string]map[string]string{}
	mu           sync.RWMutex
	initialized  bool
)

// builtinTranslations — kaynak kod içine gömülü çeviriler.
// Production'da ayrı JSON dosyaları yerine buradan okunur (deploy kolaylığı).
var builtinTranslations = map[string]map[string]string{
	LangTR: {
		// Genel
		"error.internal":     "Sunucu hatası oluştu. Lütfen daha sonra tekrar deneyin.",
		"error.not_found":    "Aradığınız sayfa veya kaynak bulunamadı.",
		"error.unauthorized": "Bu işlemi yapabilmek için giriş yapmanız gerekiyor.",
		"error.forbidden":    "Bu sayfaya erişim yetkiniz bulunmuyor.",
		"error.bad_request":  "Geçersiz istek. Lütfen girişlerinizi kontrol edin.",
		"error.rate_limit":   "Çok fazla istek gönderildi. Lütfen bekleyin.",
		"error.validation":   "Lütfen formdaki hataları düzeltin.",
		"error.csrf":         "Güvenlik doğrulaması başarısız. Lütfen sayfayı yenileyip tekrar deneyin.",
		// Auth
		"auth.login_success":          "Başarıyla giriş yapıldı.",
		"auth.login_failed":           "Kullanıcı adı veya şifre hatalı.",
		"auth.logout_success":         "Başarıyla çıkış yapıldı.",
		"auth.register_success":       "Kayıt başarılı. Lütfen e-postanızı doğrulayın.",
		"auth.email_exists":           "Bu e-posta adresi zaten kayıtlı.",
		"auth.user_inactive":          "Hesabınız aktif değil. Lütfen yöneticinizle iletişime geçin.",
		"auth.email_not_verified":     "Lütfen e-posta adresinizi doğrulayınız.",
		"auth.token_expired":          "Bağlantının süresi dolmuş. Lütfen tekrar talep edin.",
		"auth.password_reset_sent":    "Şifre sıfırlama bağlantısı e-posta adresinize gönderildi.",
		"auth.password_reset_success": "Şifreniz başarıyla sıfırlandı.",
		"auth.password_updated":       "Şifreniz başarıyla güncellendi.",
		"auth.email_verified":         "E-posta adresiniz başarıyla doğrulandı.",
		"auth.verification_sent":      "Doğrulama bağlantısı e-posta adresinize gönderildi.",
		"auth.profile_updated":        "Profil bilgileriniz güncellendi.",
		"auth.session_expired":        "Oturum süresi dolmuş. Lütfen tekrar giriş yapın.",
		// CRUD genel
		"crud.created":   "Kayıt başarıyla oluşturuldu.",
		"crud.updated":   "Kayıt başarıyla güncellendi.",
		"crud.deleted":   "Kayıt başarıyla silindi.",
		"crud.not_found": "Kayıt bulunamadı.",
		"crud.error":     "İşlem sırasında bir hata oluştu.",
		// Doğrulama
		"validation.required":   "Bu alan zorunludur.",
		"validation.email":      "Geçerli bir e-posta adresi giriniz.",
		"validation.min_length": "En az %d karakter olmalıdır.",
		"validation.max_length": "En fazla %d karakter olabilir.",
		"validation.mismatch":   "Alanlar eşleşmiyor.",
		// İletişim
		"contact.success": "Mesajınız alındı. En kısa sürede size dönüş yapacağız.",
		"contact.error":   "Mesaj gönderilemedi. Lütfen tekrar deneyin.",
		// Pagination
		"pagination.showing": "%d-%d arası gösteriliyor, toplam %d kayıt",
		"pagination.no_data": "Gösterilecek kayıt bulunamadı.",
	},
	LangEN: {
		// Genel
		"error.internal":     "An internal server error occurred. Please try again later.",
		"error.not_found":    "The page or resource you are looking for was not found.",
		"error.unauthorized": "You must be logged in to perform this action.",
		"error.forbidden":    "You do not have permission to access this page.",
		"error.bad_request":  "Invalid request. Please check your inputs.",
		"error.rate_limit":   "Too many requests. Please wait a moment.",
		"error.validation":   "Please correct the errors in the form.",
		"error.csrf":         "Security check failed. Please refresh the page and try again.",
		// Auth
		"auth.login_success":          "Successfully logged in.",
		"auth.login_failed":           "Invalid email or password.",
		"auth.logout_success":         "Successfully logged out.",
		"auth.register_success":       "Registration successful. Please verify your email.",
		"auth.email_exists":           "This email address is already registered.",
		"auth.user_inactive":          "Your account is not active. Please contact an administrator.",
		"auth.email_not_verified":     "Please verify your email address.",
		"auth.token_expired":          "The link has expired. Please request a new one.",
		"auth.password_reset_sent":    "Password reset link has been sent to your email.",
		"auth.password_reset_success": "Your password has been successfully reset.",
		"auth.password_updated":       "Your password has been successfully updated.",
		"auth.email_verified":         "Your email address has been successfully verified.",
		"auth.verification_sent":      "Verification link has been sent to your email.",
		"auth.profile_updated":        "Your profile information has been updated.",
		"auth.session_expired":        "Session expired. Please log in again.",
		// CRUD genel
		"crud.created":   "Record successfully created.",
		"crud.updated":   "Record successfully updated.",
		"crud.deleted":   "Record successfully deleted.",
		"crud.not_found": "Record not found.",
		"crud.error":     "An error occurred during the operation.",
		// Validation
		"validation.required":   "This field is required.",
		"validation.email":      "Please enter a valid email address.",
		"validation.min_length": "Must be at least %d characters.",
		"validation.max_length": "Must be at most %d characters.",
		"validation.mismatch":   "Fields do not match.",
		// İletişim
		"contact.success": "Your message has been received. We will get back to you shortly.",
		"contact.error":   "Message could not be sent. Please try again.",
		// Pagination
		"pagination.showing": "Showing %d-%d of %d records",
		"pagination.no_data": "No records found.",
	},
}

// Init — çevirileri belleğe yükler. main() içinde çağrılmalıdır.
func Init() {
	mu.Lock()
	defer mu.Unlock()
	for lang, msgs := range builtinTranslations {
		translations[lang] = make(map[string]string, len(msgs))
		for k, v := range msgs {
			translations[lang][k] = v
		}
	}
	initialized = true
	logconfig.SLog.Infow("i18n başlatıldı",
		"languages", []string{LangTR, LangEN},
		"default", DefaultLang)
}

// LoadJSON — harici JSON dosyasından ek çeviriler yükler (opsiyonel).
// Format: {"key": "value", ...}
func LoadJSON(lang string, data []byte) error {
	var msgs map[string]string
	if err := json.Unmarshal(data, &msgs); err != nil {
		return fmt.Errorf("i18n JSON parse hatası (%s): %w", lang, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if translations[lang] == nil {
		translations[lang] = make(map[string]string)
	}
	for k, v := range msgs {
		translations[lang][k] = v
	}
	return nil
}

// Middleware — her isteğin dilini tespit eder ve Locals'a yazar.
// Öncelik: query parametresi > cookie > Accept-Language başlığı > default
func Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		lang := detectLang(c)
		c.Locals(LocalsKey, lang)
		// Response header'a da ekle
		c.Set("Content-Language", lang)
		return c.Next()
	}
}

// T — fiber.Ctx'den dil alarak çeviri döner. Anahtar bulunamazsa key döner.
func T(c fiber.Ctx, key string, args ...interface{}) string {
	lang, _ := c.Locals(LocalsKey).(string)
	return Translate(lang, key, args...)
}

// Translate — dil ve anahtar ile çeviri döner. Context gerektirmez.
func Translate(lang, key string, args ...interface{}) string {
	if lang == "" {
		lang = DefaultLang
	}
	mu.RLock()
	defer mu.RUnlock()

	msgs, ok := translations[lang]
	if !ok {
		msgs = translations[DefaultLang]
	}
	val, exists := msgs[key]
	if !exists {
		// Yedek: DefaultLang
		if lang != DefaultLang {
			if fallback, ok2 := translations[DefaultLang][key]; ok2 {
				val = fallback
				exists = true
			}
		}
		if !exists {
			return key // anahtar bulunamazsa key'i döndür
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(val, args...)
	}
	return val
}

// LangFromFiber — fiber context'ten aktif dili döner.
func LangFromFiber(c fiber.Ctx) string {
	if lang, ok := c.Locals(LocalsKey).(string); ok && lang != "" {
		return lang
	}
	return DefaultLang
}

// detectLang — isteğin dilini tespit eder.
func detectLang(c fiber.Ctx) string {
	// 1. Query parametresi: ?lang=en
	if q := c.Query(QueryParamName); q != "" && supportedLangs[q] {
		// cookie'ye yaz (sonraki istekler için)
		c.Cookie(&fiber.Cookie{
			Name:     CookieName,
			Value:    q,
			MaxAge:   365 * 24 * 60 * 60,
			HTTPOnly: false, // JS okuyabilmeli
			SameSite: "Lax",
		})
		return q
	}

	// 2. Cookie
	if cookie := c.Cookies(CookieName); cookie != "" && supportedLangs[cookie] {
		return cookie
	}

	// 3. Accept-Language başlığı
	if accept := c.Get("Accept-Language"); accept != "" {
		for _, part := range strings.Split(accept, ",") {
			tag := strings.TrimSpace(strings.Split(part, ";")[0])
			// "tr-TR" → "tr"
			lang := strings.ToLower(strings.Split(tag, "-")[0])
			if supportedLangs[lang] {
				return lang
			}
		}
	}

	return DefaultLang
}
