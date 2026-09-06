package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	stdhttp "net/http"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/zatrano/framework/v2/kernel/cookie"
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
)

const sessionKey = "_csrf_token"

// DefaultSessionCookie is the framework session cookie name used to detect an existing visit.
const DefaultSessionCookie = "zatrano_session"

var (
	csrfMu            sync.RWMutex
	sessionCookieName = DefaultSessionCookie
	skipAnonymousSeed func(*http.Request) bool
)

// SetSessionCookieName overrides the cookie checked by SkipAnonymousSeed (default zatrano_session).
func SetSessionCookieName(name string) {
	csrfMu.Lock()
	defer csrfMu.Unlock()
	name = strings.TrimSpace(name)
	if name == "" {
		sessionCookieName = DefaultSessionCookie
		return
	}
	sessionCookieName = name
}

// SkipAnonymousSeed registers a path matcher for CDN-friendly public GETs.
// When the matcher returns true and the request has no session cookie, reading
// methods skip token seed and XSRF-TOKEN Set-Cookie. Pass nil to disable.
func SkipAnonymousSeed(match func(*http.Request) bool) {
	csrfMu.Lock()
	defer csrfMu.Unlock()
	skipAnonymousSeed = match
}

func shouldSkipAnonymousSeed(req *http.Request) bool {
	csrfMu.RLock()
	match := skipAnonymousSeed
	cookieName := sessionCookieName
	csrfMu.RUnlock()
	if match == nil || req == nil {
		return false
	}
	if strings.TrimSpace(req.Cookie(cookieName)) != "" {
		return false
	}
	return match(req)
}

// Middleware verifies CSRF tokens on unsafe HTTP methods for all paths.
// Browser same-origin signals (Origin / Referer / Sec-Fetch-Site) are also enforced.
// To exempt prefixes (e.g. token-authenticated /api), use Except explicitly in the app.
func Middleware(next routing.HandlerFunc) routing.HandlerFunc {
	return Except()(next)
}

// Except skips CSRF verification for matching path prefixes.
func Except(prefixes ...string) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if exceptedPath(req, prefixes) {
				return next(req)
			}

			// Public cacheable GETs: no token mint, no XSRF-TOKEN cookie when anonymous.
			if isReading(req.Method()) && shouldSkipAnonymousSeed(req) {
				return next(req)
			}

			token := ensureToken(req)

			if isReading(req.Method()) {
				resp := next(req)
				return withXSRFCookie(resp, token, req)
			}

			if ok, reason := sameOriginOK(req); !ok {
				return http.Abort(stdhttp.StatusForbidden, "CSRF "+reason)
			}

			provided := req.Header("X-CSRF-TOKEN")
			if provided == "" {
				provided = req.Header("X-XSRF-TOKEN")
			}
			if provided == "" {
				provided = req.Cookie("XSRF-TOKEN")
			}
			if provided == "" {
				provided = req.Input("_token")
			}

			if !tokensMatch(token, provided) {
				return http.Abort(stdhttp.StatusForbidden, "CSRF token mismatch")
			}

			resp := next(req)
			return withXSRFCookie(resp, token, req)
		}
	}
}

func withXSRFCookie(resp *http.Response, token string, req *http.Request) *http.Response {
	if resp == nil {
		resp = http.Text("")
	}
	if token == "" {
		return resp
	}
	resp.Header("X-CSRF-TOKEN", token)
	secure := cookie.SecureByDefault()
	if req != nil && req.Secure() {
		secure = true
	}
	return resp.WithCookie(&stdhttp.Cookie{
		Name:     "XSRF-TOKEN",
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: stdhttp.SameSiteLaxMode,
	})
}

func ensureToken(req *http.Request) string {
	sess := req.Session()
	if sess == nil {
		return ""
	}
	if existing, ok := sess.Get(sessionKey).(string); ok && existing != "" {
		return existing
	}
	token := generateToken()
	sess.Put(sessionKey, token)
	return token
}

func generateToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("csrf: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(buf)
}

func tokensMatch(expected, provided string) bool {
	if expected == "" || provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

func exceptedPath(req *http.Request, prefixes []string) bool {
	requestPath := "/"
	if req != nil {
		if cleaned := canonicalExceptPath(req.Path()); cleaned != "" {
			requestPath = cleaned
		}
	}
	for _, prefix := range prefixes {
		prefix = canonicalExceptPath(prefix)
		if prefix == "" {
			continue
		}
		if requestPath == prefix {
			return true
		}
		if prefix != "/" && strings.HasPrefix(requestPath, prefix+"/") {
			return true
		}
	}
	return false
}

func canonicalExceptPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	cleaned := path.Clean(p)
	if cleaned == "." {
		return "/"
	}
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned
}

func isReading(method string) bool {
	switch strings.ToUpper(method) {
	case stdhttp.MethodGet, stdhttp.MethodHead, stdhttp.MethodOptions, stdhttp.MethodTrace:
		return true
	default:
		return false
	}
}

// sameOriginOK applies defense-in-depth for browser cross-site requests.
// Clients without Origin/Referer/Sec-Fetch-Site (curl, native apps) pass this layer
// and still require a valid CSRF token.
func sameOriginOK(req *http.Request) (bool, string) {
	sfs := strings.ToLower(strings.TrimSpace(req.Header("Sec-Fetch-Site")))
	if sfs == "cross-site" {
		return false, "cross-site request blocked"
	}

	origin := strings.TrimSpace(req.Header("Origin"))
	if origin != "" {
		if strings.EqualFold(origin, "null") {
			return false, "null origin blocked"
		}
		if !originMatchesRequest(req, origin) {
			return false, "origin mismatch"
		}
		return true, ""
	}

	referer := strings.TrimSpace(req.Header("Referer"))
	if referer != "" {
		if !refererMatchesRequest(req, referer) {
			return false, "referer mismatch"
		}
		return true, ""
	}

	return true, ""
}

func originMatchesRequest(req *http.Request, origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	host := req.Host()
	if host == "" {
		return false
	}
	if !strings.EqualFold(u.Host, host) {
		return false
	}
	return strings.EqualFold(u.Scheme, req.Scheme())
}

func refererMatchesRequest(req *http.Request, referer string) bool {
	u, err := url.Parse(referer)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	host := req.Host()
	if host == "" {
		return false
	}
	if !strings.EqualFold(u.Host, host) {
		return false
	}
	return strings.EqualFold(u.Scheme, req.Scheme())
}

// Token returns the CSRF token from the request session.
func Token(req *http.Request) string {
	return ensureToken(req)
}
