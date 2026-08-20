package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/zatrano/framework/packages/http"
)

const rememberCookiePrefix = "remember_"

// RememberTokenProvider optionally supports long-lived remember-me tokens.
type RememberTokenProvider interface {
	RetrieveByToken(id, token string) (Authenticatable, error)
	UpdateRememberToken(user Authenticatable, token string) error
}

func rememberCookieName(guard string) string {
	if guard == "" {
		guard = "web"
	}
	return rememberCookiePrefix + guard
}

func wantsRemember(remember []bool) bool {
	return len(remember) > 0 && remember[0]
}

func generateRememberToken() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// HashRememberToken returns a SHA-256 hex digest for storage lookups.
func HashRememberToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RememberHashesEqual compares two remember-token digests in constant time.
func RememberHashesEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func encodeRememberCookie(id any, token string) string {
	return fmt.Sprintf("%s|%s", fmt.Sprint(id), token)
}

func decodeRememberCookie(value string) (id, token string, ok bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", false
	}
	parts := strings.SplitN(value, "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func (g *Guard) rememberProvider() RememberTokenProvider {
	if g == nil || g.provider == nil {
		return nil
	}
	rp, _ := g.provider.(RememberTokenProvider)
	return rp
}

func (g *Guard) queueRememberCookie(req *http.Request, user Authenticatable) error {
	rp := g.rememberProvider()
	if rp == nil || req == nil {
		return nil
	}
	token, err := generateRememberToken()
	if err != nil {
		return err
	}
	if err := rp.UpdateRememberToken(user, HashRememberToken(token)); err != nil {
		return err
	}
	req.Cookies().ForeverSecure(rememberCookieName(g.name), encodeRememberCookie(user.AuthID(), token), req.Secure())
	return nil
}

func (g *Guard) clearRememberCookie(req *http.Request, user Authenticatable) {
	if req == nil {
		return
	}
	req.Cookies().Forget(rememberCookieName(g.name))
	if user == nil {
		return
	}
	if rp := g.rememberProvider(); rp != nil {
		_ = rp.UpdateRememberToken(user, "")
	}
}

func (g *Guard) userFromRememberCookie(req *http.Request) Authenticatable {
	rp := g.rememberProvider()
	if rp == nil || req == nil {
		return nil
	}
	raw := strings.TrimSpace(req.Cookie(rememberCookieName(g.name)))
	id, token, ok := decodeRememberCookie(raw)
	if !ok {
		return nil
	}
	user, err := rp.RetrieveByToken(id, HashRememberToken(token))
	if err != nil || user == nil {
		req.Cookies().Forget(rememberCookieName(g.name))
		return nil
	}
	return user
}
