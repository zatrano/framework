package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisDriver is a cache driver backed by Redis. Suitable for multi-process and
// distributed deployments. Keys can optionally be namespaced with a prefix.
type RedisDriver struct {
	client *redis.Client
	prefix string
}

// RedisOption configures the Redis cache driver.
type RedisOption func(*RedisDriver)

// WithPrefix sets a key prefix for all cache operations (e.g. "cache:").
func WithPrefix(prefix string) RedisOption {
	return func(d *RedisDriver) { d.prefix = prefix }
}

// NewRedisDriver creates a Redis-backed cache driver.
func NewRedisDriver(client *redis.Client, opts ...RedisOption) *RedisDriver {
	d := &RedisDriver{client: client, prefix: "zatrano:cache:"}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Ensure RedisDriver implements Driver at compile time.
var _ Driver = (*RedisDriver)(nil)

func (r *RedisDriver) key(k string) string { return r.prefix + k }

func (r *RedisDriver) Get(ctx context.Context, key string) (string, bool) {
	val, err := r.client.Get(ctx, r.key(key)).Result()
	if errors.Is(err, redis.Nil) || err != nil {
		return "", false
	}
	return val, true
}

func (r *RedisDriver) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, r.key(key), value, ttl).Err()
}

func (r *RedisDriver) Has(ctx context.Context, key string) bool {
	n, err := r.client.Exists(ctx, r.key(key)).Result()
	return err == nil && n > 0
}

func (r *RedisDriver) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	prefixed := make([]string, len(keys))
	for i, k := range keys {
		prefixed[i] = r.key(k)
	}
	return r.client.Del(ctx, prefixed...).Err()
}

func (r *RedisDriver) TTL(ctx context.Context, key string) (time.Duration, bool) {
	d, err := r.client.TTL(ctx, r.key(key)).Result()
	if err != nil || d < 0 {
		return 0, false
	}
	return d, true
}

func (r *RedisDriver) Flush(ctx context.Context) error {
	iter := r.client.Scan(ctx, 0, r.prefix+"*", 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisDriver) Close() error {
	// Do not close the Redis client — it is shared with the application.
	return nil
}

// Client returns the underlying Redis client (used for tag operations).
func (r *RedisDriver) Client() *redis.Client { return r.client }

// Prefix returns the key prefix used by this driver.
func (r *RedisDriver) Prefix() string { return r.prefix }
