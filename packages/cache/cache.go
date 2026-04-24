// Package cache — Redis tabanlı okuma-önbellekleme katmanı.
// Servis katmanında şeffaf olarak kullanılır; DB her seferinde sorgulanmaz.
// TTL, prefix ve invalidation desteği vardır.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/configs/redisconfig"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ErrCacheMiss — cache'de kayıt yoksa döner.
var ErrCacheMiss = errors.New("cache miss")

const defaultTTL = 5 * time.Minute

// Client — cache istemcisi.
type Client struct {
	redis  *redis.Client
	prefix string
}

// New — yeni cache client oluşturur.
// prefix: uygulama adı (farklı uygulamalar aynı Redis'i paylaşabilir).
func New(prefix string) *Client {
	return &Client{
		redis:  redisconfig.GetClient(),
		prefix: prefix,
	}
}

// key — prefix + key kombinasyonu.
func (c *Client) key(k string) string {
	if c.prefix == "" {
		return k
	}
	return fmt.Sprintf("%s:%s", c.prefix, k)
}

// Get — cache'den JSON deserialize ederek değer döner.
// Cache miss'te ErrCacheMiss döner.
func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := c.redis.Get(ctx, c.key(key)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		logconfig.Log.Warn("Cache get hatası", zap.String("key", key), zap.Error(err))
		return ErrCacheMiss // Redis hatasında cache miss gibi davran
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		logconfig.Log.Warn("Cache deserialize hatası", zap.String("key", key), zap.Error(err))
		return ErrCacheMiss
	}
	return nil
}

// Set — değeri JSON serialize ederek cache'e yazar.
func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) error {
	d := defaultTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		d = ttl[0]
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache serialize hatası: %w", err)
	}
	if err := c.redis.Set(ctx, c.key(key), data, d).Err(); err != nil {
		logconfig.Log.Warn("Cache set hatası", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}

// Delete — tek bir anahtarı siler.
func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.redis.Del(ctx, c.key(key)).Err(); err != nil && !errors.Is(err, redis.Nil) {
		logconfig.Log.Warn("Cache delete hatası", zap.String("key", key), zap.Error(err))
		return err
	}
	return nil
}

// DeleteByPattern — pattern'e uyan tüm anahtarları siler (wildcard destekli).
// Dikkat: production'da büyük key seti varsa SCAN kullanılır, KEYS değil.
func (c *Client) DeleteByPattern(ctx context.Context, pattern string) error {
	fullPattern := c.key(pattern)
	var cursor uint64
	var deleted int64
	for {
		keys, nextCursor, err := c.redis.Scan(ctx, cursor, fullPattern, 100).Result()
		if err != nil {
			logconfig.Log.Warn("Cache pattern scan hatası",
				zap.String("pattern", fullPattern), zap.Error(err))
			return err
		}
		if len(keys) > 0 {
			n, err := c.redis.Del(ctx, keys...).Result()
			if err != nil {
				logconfig.Log.Warn("Cache pattern delete hatası", zap.Error(err))
			}
			deleted += n
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	if deleted > 0 {
		logconfig.Log.Debug("Cache pattern temizlendi",
			zap.String("pattern", fullPattern),
			zap.Int64("deleted", deleted))
	}
	return nil
}

// GetOrSet — okuma-önbellekleme: cache'de varsa döner, yoksa fn() çağırır, cache'e yazar, döner.
// fn() bir hata döndürürse cache'e yazılmaz ve hata iletilir.
func (c *Client) GetOrSet(ctx context.Context, key string, dest interface{}, fn func() (interface{}, error), ttl ...time.Duration) error {
	if err := c.Get(ctx, key, dest); err == nil {
		return nil // cache hit
	}
	val, err := fn()
	if err != nil {
		return err
	}
	// Arka planda yaz (hata varsa sadece logla)
	go func() {
		bgCtx := context.Background()
		if setErr := c.Set(bgCtx, key, val, ttl...); setErr != nil {
			logconfig.Log.Warn("Cache arka plan yazma hatası",
				zap.String("key", key), zap.Error(setErr))
		}
	}()
	// dest'e yaz
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Exists — anahtarın cache'de olup olmadığını kontrol eder.
func (c *Client) Exists(ctx context.Context, key string) bool {
	n, err := c.redis.Exists(ctx, c.key(key)).Result()
	return err == nil && n > 0
}

// FlushPrefix — prefix'e ait tüm anahtarları siler (test veya admin amaçlı).
func (c *Client) FlushPrefix(ctx context.Context) error {
	return c.DeleteByPattern(ctx, "*")
}

// TTL seçenekleri — servis katmanında kolayca kullanılır
var (
	TTL1Min   = time.Minute
	TTL5Min   = 5 * time.Minute
	TTL15Min  = 15 * time.Minute
	TTL1Hour  = time.Hour
	TTL6Hours = 6 * time.Hour
	TTL1Day   = 24 * time.Hour
)
