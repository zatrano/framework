package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore is a Redis-backed cache store.
type RedisStore struct {
	client *redis.Client
	prefix string
}

// NewRedisStore creates a Redis cache store.
func NewRedisStore(client *redis.Client, prefix string) *RedisStore {
	if prefix == "" {
		prefix = "zatrano_cache:"
	}
	return &RedisStore{client: client, prefix: prefix}
}

func (s *RedisStore) key(key string) string {
	return s.prefix + key
}

// Get returns a cached value.
func (s *RedisStore) Get(key string) (any, bool) {
	raw, err := s.client.Get(context.Background(), s.key(key)).Bytes()
	if err != nil {
		return nil, false
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return string(raw), true
	}
	return value, true
}

// Put stores a value with TTL.
func (s *RedisStore) Put(key string, value any, ttl time.Duration) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return s.client.Set(context.Background(), s.key(key), raw, 0).Err()
	}
	return s.client.Set(context.Background(), s.key(key), raw, ttl).Err()
}

// Forever stores a value indefinitely.
func (s *RedisStore) Forever(key string, value any) error {
	return s.Put(key, value, 0)
}

// Forget removes a key.
func (s *RedisStore) Forget(key string) error {
	return s.client.Del(context.Background(), s.key(key)).Err()
}

// Flush clears keys with the store prefix.
func (s *RedisStore) Flush() error {
	ctx := context.Background()
	iter := s.client.Scan(ctx, 0, s.prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		if err := s.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

// Has reports whether a key exists.
func (s *RedisStore) Has(key string) bool {
	n, err := s.client.Exists(context.Background(), s.key(key)).Result()
	return err == nil && n > 0
}

// Pull gets and deletes a key.
func (s *RedisStore) Pull(key string) (any, bool) {
	value, ok := s.Get(key)
	if ok {
		_ = s.Forget(key)
	}
	return value, ok
}

// Remember returns cached value or stores callback result.
func (s *RedisStore) Remember(key string, ttl time.Duration, callback func() (any, error)) (any, error) {
	if value, ok := s.Get(key); ok {
		return value, nil
	}
	value, err := callback()
	if err != nil {
		return nil, err
	}
	if err := s.Put(key, value, ttl); err != nil {
		return nil, err
	}
	return value, nil
}

// MemoryStore is an in-memory cache store.
type MemoryStore struct {
	items map[string]item
}

// NewMemoryStore creates an in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[string]item)}
}

// Get returns a cached value.
func (s *MemoryStore) Get(key string) (any, bool) {
	payload, ok := s.items[key]
	if !ok {
		return nil, false
	}
	if !payload.Forever && !payload.ExpiresAt.IsZero() && time.Now().After(payload.ExpiresAt) {
		delete(s.items, key)
		return nil, false
	}
	return payload.Value, true
}

// Put stores a value with TTL.
func (s *MemoryStore) Put(key string, value any, ttl time.Duration) error {
	payload := item{Value: value}
	if ttl > 0 {
		payload.ExpiresAt = time.Now().Add(ttl)
	} else {
		payload.Forever = true
	}
	s.items[key] = payload
	return nil
}

// Forever stores a value indefinitely.
func (s *MemoryStore) Forever(key string, value any) error { return s.Put(key, value, 0) }

// Forget removes a key.
func (s *MemoryStore) Forget(key string) error {
	delete(s.items, key)
	return nil
}

// Flush clears the store.
func (s *MemoryStore) Flush() error {
	s.items = make(map[string]item)
	return nil
}

// Has reports whether a key exists.
func (s *MemoryStore) Has(key string) bool {
	_, ok := s.Get(key)
	return ok
}

// Pull gets and deletes a key.
func (s *MemoryStore) Pull(key string) (any, bool) {
	value, ok := s.Get(key)
	if ok {
		_ = s.Forget(key)
	}
	return value, ok
}

// Remember returns cached value or stores callback result.
func (s *MemoryStore) Remember(key string, ttl time.Duration, callback func() (any, error)) (any, error) {
	if value, ok := s.Get(key); ok {
		return value, nil
	}
	value, err := callback()
	if err != nil {
		return nil, err
	}
	if err := s.Put(key, value, ttl); err != nil {
		return nil, err
	}
	return value, nil
}
