package honeypot

import (
	"strings"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Config configures honeypot spam protection.
type Config struct {
	Field     string
	Timestamp string
	MinDelay  time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Field:     "website",
		Timestamp: "_hp_ts",
		MinDelay:  2 * time.Second,
	}
}

// Middleware rejects requests that fill the honeypot field or submit too quickly.
func Middleware(cfg ...Config) routing.MiddlewareFunc {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
		if c.Field == "" {
			c.Field = "website"
		}
		if c.Timestamp == "" {
			c.Timestamp = "_hp_ts"
		}
	}
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if isUnsafe(req.Method()) {
				if strings.TrimSpace(req.Input(c.Field)) != "" {
					return http.Abort(422, "Spam detected")
				}
				if c.MinDelay > 0 {
					if ts := strings.TrimSpace(req.Input(c.Timestamp)); ts != "" {
						if started, err := time.Parse(time.RFC3339, ts); err == nil {
							if time.Since(started) < c.MinDelay {
								return http.Abort(422, "Form submitted too quickly")
							}
						}
					}
				}
			}
			return next(req)
		}
	}
}

// Fields returns hidden field HTML for forms.
func Fields(cfg ...Config) string {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
		if c.Field == "" {
			c.Field = "website"
		}
		if c.Timestamp == "" {
			c.Timestamp = "_hp_ts"
		}
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	return `<div style="display:none" aria-hidden="true">` +
		`<input type="text" name="` + c.Field + `" value="" tabindex="-1" autocomplete="off">` +
		`<input type="hidden" name="` + c.Timestamp + `" value="` + ts + `">` +
		`</div>`
}

func isUnsafe(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}
