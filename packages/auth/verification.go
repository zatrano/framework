package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// MustVerifyEmail is implemented by users that require email verification.
type MustVerifyEmail interface {
	HasVerifiedEmail() bool
	GetEmailForVerification() string
	AuthID() any
}

// HasVerifiedEmail reports whether the authenticatable user has a verified email.
// Users that do not implement MustVerifyEmail are treated as verified (opt-in verification).
func HasVerifiedEmail(user Authenticatable) bool {
	if user == nil {
		return false
	}
	if v, ok := user.(MustVerifyEmail); ok {
		return v.HasVerifiedEmail()
	}
	return true
}

// EmailForVerification returns the email used for verification links.
func EmailForVerification(user Authenticatable) string {
	if user == nil {
		return ""
	}
	if v, ok := user.(MustVerifyEmail); ok {
		return v.GetEmailForVerification()
	}
	if generic, ok := user.(*GenericUser); ok {
		return fmt.Sprint(generic.Get("email"))
	}
	return ""
}

// EmailHash returns a stable hash of the email for signed verification URLs.
func EmailHash(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(sum[:])
}

// VerifyEmailMiddleware blocks authenticated users who have not verified email.
func VerifyEmailMiddleware(manager *Manager, guards ...string) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			user, _ := firstAuthenticated(manager, req, guards...)
			if user == nil {
				if req.WantsJSON() {
					return http.JSON(map[string]any{"message": "Unauthenticated."}).Status(401)
				}
				return http.Redirect("/auth/login")
			}
			if HasVerifiedEmail(user) {
				return next(req)
			}
			if req.WantsJSON() {
				return http.JSON(map[string]any{"message": "Your email address is not verified."}).Status(403)
			}
			return http.Redirect("/auth/email/verify")
		}
	}
}

// MarkEmailVerified updates a GenericUser attribute map (caller persists).
func MarkEmailVerified(attrs map[string]any) {
	if attrs == nil {
		return
	}
	attrs["email_verified_at"] = time.Now().UTC()
}

// MarkEmailAsVerified persists verification and dispatches EventVerified.
func (m *Manager) MarkEmailAsVerified(req *http.Request, user Authenticatable) error {
	if m == nil || user == nil {
		return fmt.Errorf("user is required")
	}
	if HasVerifiedEmail(user) {
		return nil
	}
	guard := m.Guard()
	if guard == nil || guard.Provider() == nil {
		return fmt.Errorf("auth provider unavailable")
	}
	attrs := map[string]any{"email_verified_at": time.Now().UTC()}
	updater, ok := guard.Provider().(AttributeUpdater)
	if !ok || updater == nil {
		if generic, ok := user.(*GenericUser); ok {
			MarkEmailVerified(generic.Attributes)
		} else {
			return fmt.Errorf("verification provider unavailable")
		}
	} else if err := updater.UpdateAttributes(user.AuthID(), attrs); err != nil {
		return err
	} else if generic, ok := user.(*GenericUser); ok && generic.Attributes != nil {
		MarkEmailVerified(generic.Attributes)
	}
	m.dispatch(EventVerified, VerifiedEvent{
		Request: req,
		User:    user,
		Guard:   guard.guardName(),
		At:      time.Now().UTC(),
	})
	return nil
}

// HasVerifiedEmail implements MustVerifyEmail for GenericUser.
func (u *GenericUser) HasVerifiedEmail() bool {
	if u == nil {
		return false
	}
	raw := u.Get("email_verified_at")
	if raw == nil {
		return false
	}
	s := strings.TrimSpace(fmt.Sprint(raw))
	return s != "" && s != "<nil>"
}

// GetEmailForVerification implements MustVerifyEmail for GenericUser.
func (u *GenericUser) GetEmailForVerification() string {
	if u == nil {
		return ""
	}
	return fmt.Sprint(u.Get("email"))
}
