package idempotency

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

const (
	HeaderKey = "Idempotency-Key"
	AttrKey   = "idempotency_key"
)

// Cache is the minimal cache contract used by the middleware.
type Cache interface {
	Get(key string) (any, bool)
	Put(key string, value any, ttl time.Duration) error
}

type cachedResponse struct {
	Status  int               `json:"status"`
	Body    json.RawMessage   `json:"body"`
	Headers map[string]string `json:"headers"`
}

// Middleware replays identical responses for the same Idempotency-Key.
// Only applies to unsafe methods when the header is present.
func Middleware(cache Cache, ttl time.Duration) routing.MiddlewareFunc {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if !isUnsafe(req.Method()) {
				return next(req)
			}
			key := req.Header(HeaderKey)
			if key == "" {
				return next(req)
			}
			req.Set(AttrKey, key)
			cacheKey := "idempotency:" + hashKey(key, req.Method(), req.Path())

			if raw, ok := cache.Get(cacheKey); ok {
				if payload, err := decodeCached(raw); err == nil {
					resp := http.NoContent().Status(payload.Status)
					ct := payload.Headers["Content-Type"]
					if ct == "" {
						ct = "application/json"
					}
					resp.SetContent([]byte(payload.Body), ct)
					resp.Header("Idempotent-Replayed", "true")
					for k, v := range payload.Headers {
						if k == "Content-Type" {
							continue
						}
						resp.Header(k, v)
					}
					return resp
				}
			}

			resp := next(req)
			if resp == nil {
				return resp
			}
			// Only cache successful JSON-ish responses.
			if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
				payload := cachedResponse{
					Status:  resp.StatusCode(),
					Body:    json.RawMessage(resp.Content()),
					Headers: map[string]string{},
				}
				if ct := resp.ContentType(); ct != "" {
					payload.Headers["Content-Type"] = ct
				}
				_ = cache.Put(cacheKey, mustJSON(payload), ttl)
			}
			resp.Header("Idempotency-Key", key)
			return resp
		}
	}
}

func isUnsafe(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func hashKey(parts ...string) string {
	h := sha1.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func mustJSON(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func decodeCached(raw any) (*cachedResponse, error) {
	switch v := raw.(type) {
	case string:
		var payload cachedResponse
		if err := json.Unmarshal([]byte(v), &payload); err != nil {
			return nil, err
		}
		return &payload, nil
	case []byte:
		var payload cachedResponse
		if err := json.Unmarshal(v, &payload); err != nil {
			return nil, err
		}
		return &payload, nil
	default:
		rawBytes, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var payload cachedResponse
		if err := json.Unmarshal(rawBytes, &payload); err != nil {
			return nil, err
		}
		return &payload, nil
	}
}
