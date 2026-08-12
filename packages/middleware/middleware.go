package middleware

import (
	"fmt"
	"log"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Stack builds a middleware pipeline around a final handler.
func Stack(handler routing.HandlerFunc, layers ...routing.MiddlewareFunc) routing.HandlerFunc {
	for i := len(layers) - 1; i >= 0; i-- {
		handler = layers[i](handler)
	}
	return handler
}

// Logger logs each request.
func Logger(next routing.HandlerFunc) routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		start := time.Now()
		resp := next(req)
		status := 200
		if resp != nil {
			status = resp.StatusCode()
		}
		log.Printf("%s %s -> %d (%s)", req.Method(), req.Path(), status, time.Since(start))
		return resp
	}
}

// Recover catches panics and returns a 500 response.
func Recover(next routing.HandlerFunc) routing.HandlerFunc {
	return func(req *http.Request) (resp *http.Response) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic recovered: %v", recovered)
				resp = http.Abort(500, fmt.Sprintf("Server Error: %v", recovered))
			}
		}()
		return next(req)
	}
}

// ForceJSON sets Accept to application/json.
func ForceJSON(next routing.HandlerFunc) routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		req.Raw().Header.Set("Accept", "application/json")
		return next(req)
	}
}

// RequestID assigns a simple request attribute timestamp ID.
func RequestID(next routing.HandlerFunc) routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		id := fmt.Sprintf("%d", time.Now().UnixNano())
		req.Set("request_id", id)
		resp := next(req)
		if resp != nil {
			resp.Header("X-Request-ID", id)
		}
		return resp
	}
}

// CORS adds permissive CORS headers for local development.
func CORS(next routing.HandlerFunc) routing.HandlerFunc {
	return CORSWith(DefaultCORSConfig())(next)
}
