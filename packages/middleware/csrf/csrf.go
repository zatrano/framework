package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	stdhttp "net/http"
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

const sessionKey = "_csrf_token"

// Middleware verifies CSRF tokens on unsafe HTTP methods for all paths.
// To exempt prefixes (e.g. token-authenticated /api), use Except explicitly in the app.
func Middleware(next routing.HandlerFunc) routing.HandlerFunc {
	return Except()(next)
}

// Except skips CSRF verification for matching path prefixes.
func Except(prefixes ...string) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			for _, prefix := range prefixes {
				if strings.HasPrefix(req.Path(), prefix) {
					return next(req)
				}
			}

			token := ensureToken(req)

			if isReading(req.Method()) {
				resp := next(req)
				return withXSRFCookie(resp, token, req)
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
	secure := false
	if req != nil {
		secure = req.Secure()
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
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func tokensMatch(expected, provided string) bool {
	if expected == "" || provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(provided)) == 1
}

func isReading(method string) bool {
	switch strings.ToUpper(method) {
	case stdhttp.MethodGet, stdhttp.MethodHead, stdhttp.MethodOptions, stdhttp.MethodTrace:
		return true
	default:
		return false
	}
}

// Token returns the CSRF token from the request session.
func Token(req *http.Request) string {
	return ensureToken(req)
}
