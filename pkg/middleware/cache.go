package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/cache"
)

// cachedResponse is the serialised form stored in the cache.
type cachedResponse struct {
	Status      int               `json:"status"`
	ContentType string            `json:"content_type"`
	Body        string            `json:"body"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// CacheConfig configures the response cache middleware.
type CacheConfig struct {
	// TTL is the default time-to-live for cached responses.
	TTL time.Duration

	// KeyFunc generates the cache key from the request.
	// Defaults to method + path + query string.
	KeyFunc func(c fiber.Ctx) string

	// Next defines a function to skip caching for certain requests.
	// Return true to skip caching.
	Next func(c fiber.Ctx) bool

	// Tags are optional cache tags applied to every stored response.
	// Use cache.Tags("tag").Flush(ctx) to invalidate.
	Tags []string

	// StoreHeader when true, caches response headers (Content-Type is always cached).
	StoreHeader bool

	// Methods lists HTTP methods to cache. Defaults to GET only.
	Methods []string
}

// Cache returns middleware that caches full HTTP responses for the given duration.
// Only GET requests are cached by default. Responses with status >= 400 are not cached.
//
// Usage:
//
//	app.Get("/api/v1/products", middleware.Cache(mgr, 5*time.Minute), handler)
//
// With options:
//
//	app.Get("/api/v1/products", middleware.CacheWithConfig(mgr, middleware.CacheConfig{
//	    TTL:  10 * time.Minute,
//	    Tags: []string{"products"},
//	}), handler)
func Cache(mgr *cache.Manager, ttl time.Duration) fiber.Handler {
	return CacheWithConfig(mgr, CacheConfig{TTL: ttl})
}

// CacheWithConfig returns response cache middleware with full configuration.
func CacheWithConfig(mgr *cache.Manager, cfg CacheConfig) fiber.Handler {
	if cfg.TTL <= 0 {
		cfg.TTL = 5 * time.Minute
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = defaultCacheKey
	}
	methods := map[string]bool{"GET": true}
	if len(cfg.Methods) > 0 {
		methods = make(map[string]bool, len(cfg.Methods))
		for _, m := range cfg.Methods {
			methods[m] = true
		}
	}

	return func(c fiber.Ctx) error {
		// Skip conditions.
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}
		if !methods[c.Method()] {
			return c.Next()
		}

		key := "resp:" + cfg.KeyFunc(c)

		// Try cache hit.
		var cached cachedResponse
		if ok, _ := mgr.GetJSON(c.Context(), key, &cached); ok {
			c.Set("X-Cache", "HIT")
			c.Set("Content-Type", cached.ContentType)
			for k, v := range cached.Headers {
				c.Set(k, v)
			}
			return c.Status(cached.Status).SendString(cached.Body)
		}

		// Execute handler.
		if err := c.Next(); err != nil {
			return err
		}

		// Only cache successful responses.
		status := c.Response().StatusCode()
		if status == 0 {
			status = fiber.StatusOK
		}
		if status >= 400 {
			return nil
		}

		// Build cached response.
		entry := cachedResponse{
			Status:      status,
			ContentType: string(c.Response().Header.ContentType()),
			Body:        string(c.Response().Body()),
		}
		if cfg.StoreHeader {
			entry.Headers = make(map[string]string)
			c.Response().Header.VisitAll(func(key, val []byte) {
				k := string(key)
				if k != "Content-Type" && k != "Content-Length" {
					entry.Headers[k] = string(val)
				}
			})
		}

		// Store in cache (with optional tags).
		c.Set("X-Cache", "MISS")
		if len(cfg.Tags) > 0 {
			_ = mgr.Tags(cfg.Tags...).SetJSON(c.Context(), key, entry, cfg.TTL)
		} else {
			_ = mgr.SetJSON(c.Context(), key, entry, cfg.TTL)
		}

		return nil
	}
}

// defaultCacheKey generates a cache key from method + path + sorted query.
func defaultCacheKey(c fiber.Ctx) string {
	raw := c.Method() + ":" + c.Path() + "?" + string(c.Request().URI().QueryString())
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:8]) // 16-char hex
}

// CacheControl sets standard HTTP cache control headers.
// This is a lightweight alternative to full response caching — it instructs
// clients and CDNs to cache responses without server-side storage.
//
// Usage:
//
//	app.Get("/static/data.json", middleware.CacheControl(1*time.Hour), handler)
func CacheControl(maxAge time.Duration) fiber.Handler {
	val := "public, max-age=" + strconv.Itoa(int(maxAge.Seconds()))
	return func(c fiber.Ctx) error {
		c.Set("Cache-Control", val)
		return c.Next()
	}
}
