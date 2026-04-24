package sessionconfig

import (
	"encoding/gob"
	"strings"
	"time"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/configs/redisconfig"
	"github.com/zatrano/framework/models"

	"github.com/gofiber/fiber/v3"
	// v3: session paketi fiber/v3 altına taşındı
	"github.com/gofiber/fiber/v3/middleware/session"
	redisstorage "github.com/gofiber/storage/redis/v3"
)

// v3: session.Store artık *session.Store değil, session.Store interface döner
// Ancak session.New() hâlâ *session.Store pointer döner (v3 beta4)
var Store *session.Store

func isProd() bool { return envconfig.IsProd() }

func sessionSameSite() string {
	def := "Lax"
	v := strings.ToLower(strings.TrimSpace(envconfig.String("SESSION_COOKIE_SAMESITE", "")))
	switch v {
	case "strict":
		return "Strict"
	case "lax":
		return "Lax"
	case "none":
		return "None"
	default:
		return def
	}
}

func InitSession() {
	Store = createSessionStore()
	registerGobTypes()
	logconfig.SLog.Info("Oturum (session) sistemi başlatıldı.")
}

func SetupSession() *session.Store {
	if Store == nil {
		logconfig.SLog.Warn("Session store henüz başlatılmamış, şimdi başlatılıyor.")
		InitSession()
	}
	return Store
}

func createSessionStore() *session.Store {
	cookieName := envconfig.String("SESSION_COOKIE_NAME", "session_id")
	expirationHours := envconfig.Int("SESSION_EXPIRATION_HOURS", 0)
	if expirationHours <= 0 {
		expirationHours = envconfig.Int("SESSION_TTL_HOURS", 24)
	}
	sameSite := sessionSameSite()

	secure := isProd()
	cookieDomain := ""

	if !isProd() {
		secure = false
	}
	if isProd() {
		if sameSite == "None" {
			secure = true
		}
		cookieDomain = envconfig.String("COOKIE_DOMAIN", "")
	}

	redisStore := redisstorage.NewFromConnection(redisconfig.GetClient())

	// Fiber v3: session.NewStore + IdleTimeout / AbsoluteTimeout (Expiration alanı yok)
	exp := time.Duration(expirationHours) * time.Hour
	store := session.NewStore(session.Config{
		KeyLookup:      "cookie:" + cookieName,
		CookieHTTPOnly: true,
		CookieSecure:   secure,
		CookieSameSite: sameSite,
		CookieDomain:   cookieDomain,
		IdleTimeout:    exp,
		Storage:        redisStore,
	})

	logconfig.SLog.Infow("Session store Redis ile yapılandırıldı",
		"cookie_name", cookieName,
		"cookie_http_only", true,
		"cookie_secure", secure,
		"same_site", sameSite,
		"domain", cookieDomain,
		"expiration_hours", expirationHours,
	)
	return store
}

func registerGobTypes() {
	gob.Register(&models.User{})
	logconfig.SLog.Debug("Session için gob türleri kaydedildi: *models.User")
}

// v3: session.Store.Get(c fiber.Ctx) — fiber.Ctx değişti (artık interface değil concrete type)
func SessionStart(c fiber.Ctx) (*session.Session, error) {
	if Store == nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "session store not initialized")
	}
	return Store.Get(c)
}

func DestroySession(c fiber.Ctx) error {
	sess, err := SessionStart(c)
	if err != nil {
		return err
	}
	return sess.Destroy()
}

func GetUserID(c fiber.Ctx) (uint, error) {
	return GetUserIDFromSession(c)
}

func SetValue(c fiber.Ctx, key string, value interface{}) error {
	return SetSessionValue(c, key, value)
}

func GetUserIDFromSession(c fiber.Ctx) (uint, error) {
	sess, err := SessionStart(c)
	if err != nil {
		return 0, err
	}
	key := "user_id"
	switch v := sess.Get(key).(type) {
	case uint:
		return v, nil
	case int:
		return uint(v), nil
	case int64:
		return uint(v), nil
	case float64:
		if v < 0 {
			return 0, fiber.ErrUnauthorized
		}
		return uint(v), nil
	default:
		return 0, fiber.ErrUnauthorized
	}
}

func SetSessionValue(c fiber.Ctx, key string, value interface{}) error {
	sess, err := SessionStart(c)
	if err != nil {
		return err
	}
	sess.Set(key, value)
	return sess.Save()
}
