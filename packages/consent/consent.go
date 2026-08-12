package consent

import (
	"encoding/json"
	stdhttp "net/http"
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

const (
	// CookieName is the default consent cookie.
	CookieName = "zatrano_consent"
	requestKey = "consent"
)

// Preferences stores cookie consent choices.
type Preferences struct {
	Necessary bool `json:"necessary"`
	Analytics bool `json:"analytics"`
	Marketing bool `json:"marketing"`
}

// Default returns necessary-only consent.
func Default() Preferences {
	return Preferences{Necessary: true}
}

// Parse decodes a consent cookie value.
func Parse(raw string) Preferences {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Default()
	}
	var p Preferences
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return Default()
	}
	p.Necessary = true
	return p
}

// Encode serializes preferences for a cookie value.
func Encode(p Preferences) string {
	p.Necessary = true
	b, err := json.Marshal(p)
	if err != nil {
		return `{"necessary":true}`
	}
	return string(b)
}

// Allowed reports whether a category is consented.
func Allowed(p Preferences, category string) bool {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "necessary", "essential":
		return true
	case "analytics":
		return p.Analytics
	case "marketing":
		return p.Marketing
	default:
		return false
	}
}

// FromRequest reads consent preferences from the request context or cookie.
func FromRequest(req *http.Request) Preferences {
	if req == nil {
		return Default()
	}
	if v, ok := req.Get(requestKey).(Preferences); ok {
		return v
	}
	return Parse(req.Cookie(CookieName))
}

// Middleware loads consent preferences onto the request.
func Middleware() routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			prefs := Parse(req.Cookie(CookieName))
			req.Set(requestKey, prefs)
			return next(req)
		}
	}
}

// SetCookie writes the consent cookie onto a response.
func SetCookie(resp *http.Response, prefs Preferences, maxAge int) *http.Response {
	if resp == nil {
		resp = http.JSON(map[string]any{"ok": true})
	}
	if maxAge <= 0 {
		maxAge = 365 * 24 * 60 * 60
	}
	return resp.WithCookie(&stdhttp.Cookie{
		Name:     CookieName,
		Value:    Encode(prefs),
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: false,
		SameSite: stdhttp.SameSiteLaxMode,
	})
}

// BannerHTML returns a minimal consent banner snippet.
func BannerHTML() string {
	return `<aside id="zatrano-consent" role="dialog" aria-label="Cookie consent">
  <p>We use cookies for essential features. Analytics and marketing are optional.</p>
  <button type="button" data-consent="accept-all">Accept all</button>
  <button type="button" data-consent="necessary">Necessary only</button>
</aside>`
}
