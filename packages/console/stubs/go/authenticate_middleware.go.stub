package middleware

import (
	"github.com/zatrano/framework/packages/apitoken"
	"github.com/zatrano/framework/packages/auth"
	. "github.com/zatrano/framework/packages/http"
	. "github.com/zatrano/framework/packages/routing"
)

// Authenticate is a legacy alias — prefer AuthenticateToken or AuthenticateSession.
func Authenticate(next HandlerFunc) HandlerFunc {
	return func(req *Request) *Response {
		return JSON(map[string]any{
			"message": "Unauthenticated. Use AuthenticateToken(apitoken.From(app)) or AuthenticateSession(auth.From(app)).",
		}).Status(401)
	}
}

// AuthenticateToken authenticates via personal access tokens.
func AuthenticateToken(manager *apitoken.Manager, abilities ...string) MiddlewareFunc {
	return manager.Middleware(abilities...)
}

// AuthenticateSession ensures a session-authenticated user is present.
func AuthenticateSession(manager *auth.Manager, guards ...string) MiddlewareFunc {
	return auth.Middleware(manager, guards...)
}
