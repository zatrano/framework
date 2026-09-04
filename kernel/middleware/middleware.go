package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9._\-:]{1,128}$`)

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

// Recover catches panics and returns a generic 500. Panic details stay in logs.
func Recover(next routing.HandlerFunc) routing.HandlerFunc {
	return RecoverWith(false)(next)
}

// RecoverDebug catches panics and includes the panic value in the response body.
func RecoverDebug(next routing.HandlerFunc) routing.HandlerFunc {
	return RecoverWith(true)(next)
}

// RecoverWith catches panics. When debug is false the response body is generic.
func RecoverWith(debug bool) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) (resp *http.Response) {
			defer func() {
				if recovered := recover(); recovered != nil {
					log.Printf("panic recovered: %v", recovered)
					if debug {
						resp = http.Abort(500, fmt.Sprintf("Server Error: %v", recovered))
					} else {
						resp = http.Abort(500, "Internal Server Error")
					}
				}
			}()
			return next(req)
		}
	}
}

// ForceJSON sets Accept to application/json.
func ForceJSON(next routing.HandlerFunc) routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		req.Raw().Header.Set("Accept", "application/json")
		return next(req)
	}
}

// RequestID assigns or propagates a request id (X-Request-ID / X-Correlation-ID).
func RequestID(next routing.HandlerFunc) routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		id := incomingRequestID(req)
		if id == "" {
			id = newRequestID()
		}
		req.Set("request_id", id)
		resp := next(req)
		if resp != nil {
			resp.Header("X-Request-ID", id)
			if tp := strings.TrimSpace(req.Header("Traceparent")); tp != "" {
				resp.Header("Traceparent", tp)
			}
		}
		return resp
	}
}

func incomingRequestID(req *http.Request) string {
	if req == nil {
		return ""
	}
	for _, key := range []string{"X-Request-ID", "X-Correlation-ID"} {
		if id := strings.TrimSpace(req.Header(key)); requestIDPattern.MatchString(id) {
			return id
		}
	}
	if id := traceIDFromTraceparent(req.Header("Traceparent")); id != "" {
		return id
	}
	return ""
}

func traceIDFromTraceparent(header string) string {
	header = strings.TrimSpace(header)
	parts := strings.Split(header, "-")
	if len(parts) < 4 {
		return ""
	}
	traceID := parts[1]
	if len(traceID) == 32 && requestIDPattern.MatchString(traceID) {
		return traceID
	}
	return ""
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// CORS adds CORS headers (wildcard in non-production; set CORS_ALLOWED_ORIGINS in production).
func CORS(next routing.HandlerFunc) routing.HandlerFunc {
	return CORSWith(DefaultCORSConfig())(next)
}
