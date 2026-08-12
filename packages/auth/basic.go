package auth

import (
	"encoding/base64"
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// BasicAuth authenticates the request with HTTP Basic credentials without writing a session.
func BasicAuth(manager *Manager, realm string) routing.MiddlewareFunc {
	if strings.TrimSpace(realm) == "" {
		realm = "Restricted"
	}
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if !manager.OnceBasic(req) {
				return basicUnauthorized(realm)
			}
			return next(req)
		}
	}
}

// OnceBasic authenticates via Authorization Basic and binds the user for this request only.
func (m *Manager) OnceBasic(req *http.Request) bool {
	if m == nil || req == nil || m.Guard() == nil || m.Guard().Provider() == nil {
		return false
	}
	header := req.Header("Authorization")
	encoded, ok := strings.CutPrefix(header, "Basic ")
	if !ok {
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return false
	}
	email, password, ok := strings.Cut(string(raw), ":")
	if !ok {
		return false
	}
	guard := m.Guard()
	user, err := guard.Provider().RetrieveByCredentials(map[string]string{"email": email})
	if err != nil || user == nil {
		return false
	}
	if !guard.Provider().ValidateCredentials(user, map[string]string{"email": email, "password": password}) {
		return false
	}
	return guard.Once(req, user) == nil
}

func basicUnauthorized(realm string) *http.Response {
	return http.JSON(map[string]any{"message": "Unauthenticated."}).
		Status(401).
		Header("WWW-Authenticate", `Basic realm="`+realm+`"`)
}
