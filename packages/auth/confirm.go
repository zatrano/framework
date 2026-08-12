package auth

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

const passwordConfirmedKey = "auth.password_confirmed_at"

// ConfirmPassword verifies the given password against the authenticated user.
func (m *Manager) ConfirmPassword(req *http.Request, password string) bool {
	user := m.User(req)
	if user == nil {
		return false
	}
	return hashing.Check(password, user.AuthPassword())
}

// MarkPasswordConfirmed stores the confirmation timestamp in the session.
func MarkPasswordConfirmed(req *http.Request) {
	sess := req.Session()
	if sess == nil {
		return
	}
	sess.Put(passwordConfirmedKey, time.Now().Unix())
}

// PasswordConfirmed reports whether the password was confirmed within the duration.
func PasswordConfirmed(req *http.Request, within time.Duration) bool {
	if within <= 0 {
		within = 15 * time.Minute
	}
	sess := req.Session()
	if sess == nil {
		return false
	}
	raw := sess.Get(passwordConfirmedKey)
	if raw == nil {
		return false
	}
	var ts int64
	switch v := raw.(type) {
	case int64:
		ts = v
	case int:
		ts = int64(v)
	case float64:
		ts = int64(v)
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return false
		}
		ts = n
	default:
		n, err := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		if err != nil {
			return false
		}
		ts = n
	}
	confirmedAt := time.Unix(ts, 0)
	return time.Since(confirmedAt) <= within
}

// ConfirmPasswordMiddleware blocks requests until the password was recently confirmed.
func ConfirmPasswordMiddleware(manager *Manager, within ...time.Duration) routing.MiddlewareFunc {
	ttl := 15 * time.Minute
	if len(within) > 0 && within[0] > 0 {
		ttl = within[0]
	}
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if !manager.Check(req) {
				if req.WantsJSON() {
					return http.JSON(map[string]any{"message": "Unauthenticated."}).Status(401)
				}
				return http.Redirect("/login")
			}
			if PasswordConfirmed(req, ttl) {
				return next(req)
			}
			if req.WantsJSON() {
				return http.JSON(map[string]any{
					"message": "Password confirmation required.",
					"confirm": "/api/auth/confirm-password",
				}).Status(423)
			}
			return http.Redirect("/confirm-password")
		}
	}
}

// ClearPasswordConfirmation removes the confirmation timestamp.
func ClearPasswordConfirmation(req *http.Request) {
	sess := req.Session()
	if sess == nil {
		return
	}
	sess.Forget(passwordConfirmedKey)
}

// NormalizePasswordField reads password from common input keys.
func NormalizePasswordField(req *http.Request) string {
	for _, key := range []string{"password", "current_password"} {
		if v := strings.TrimSpace(req.Input(key)); v != "" {
			return v
		}
	}
	return ""
}
