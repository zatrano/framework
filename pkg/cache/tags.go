package cache

import (
	"context"
	"fmt"
	"time"
)

// tagSetKey returns the Redis SET key that tracks all cache keys for a given tag.
func tagSetKey(prefix, tag string) string {
	return prefix + "tag:" + tag
}

// TaggedCache wraps a Manager with tag-based invalidation support.
// Each Set tracks the key under all configured tags so that Flush can
// remove every key belonging to one or more tags in a single pass.
//
// Tag tracking uses Redis SETs (SADD / SMEMBERS). When the underlying
// driver is not Redis, tags fall back to a no-op (keys are stored but
// Flush only calls the driver's Flush).
type TaggedCache struct {
	manager *Manager
	tags    []string
}

// Set stores value and registers the key under all configured tags.
func (t *TaggedCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := t.manager.Set(ctx, key, value, ttl); err != nil {
		return err
	}
	t.trackKey(ctx, key)
	return nil
}

// SetJSON JSON-encodes value and registers the key under all configured tags.
func (t *TaggedCache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	if err := t.manager.SetJSON(ctx, key, value, ttl); err != nil {
		return err
	}
	t.trackKey(ctx, key)
	return nil
}

// Get retrieves a value (passthrough — no tag awareness needed on read).
func (t *TaggedCache) Get(ctx context.Context, key string) (string, bool) {
	return t.manager.Get(ctx, key)
}

// GetJSON retrieves and JSON-decodes a value.
func (t *TaggedCache) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	return t.manager.GetJSON(ctx, key, dest)
}

// Remember computes or returns cached value and tracks the key under tags.
func (t *TaggedCache) Remember(ctx context.Context, key string, ttl time.Duration, fn func() (string, error)) (string, error) {
	val, err := t.manager.Remember(ctx, key, ttl, fn)
	if err == nil {
		t.trackKey(ctx, key)
	}
	return val, err
}

// RememberJSON computes or returns cached JSON value and tracks the key under tags.
func (t *TaggedCache) RememberJSON(ctx context.Context, key string, ttl time.Duration, dest any, fn func() (any, error)) error {
	err := t.manager.RememberJSON(ctx, key, ttl, dest, fn)
	if err == nil {
		t.trackKey(ctx, key)
	}
	return err
}

// Flush removes all keys that were tagged with any of the configured tags.
// For Redis: reads members from the tag SETs (SMEMBERS), deletes them, then
// removes the tag SETs themselves.
// For non-Redis drivers: falls back to deleting all cache entries.
func (t *TaggedCache) Flush(ctx context.Context) error {
	rd, ok := t.manager.driver.(*RedisDriver)
	if !ok {
		// Fallback for non-Redis drivers: full flush.
		return t.manager.Flush(ctx)
	}

	client := rd.Client()
	prefix := rd.Prefix()

	// Collect all keys from all tag sets.
	keySet := make(map[string]bool)
	var tagKeys []string
	for _, tag := range t.tags {
		tk := tagSetKey(prefix, tag)
		tagKeys = append(tagKeys, tk)
		members, err := client.SMembers(ctx, tk).Result()
		if err != nil {
			return fmt.Errorf("cache: tag smembers %q: %w", tag, err)
		}
		for _, m := range members {
			keySet[m] = true
		}
	}

	// Delete all tagged cache keys.
	if len(keySet) > 0 {
		keys := make([]string, 0, len(keySet))
		for k := range keySet {
			keys = append(keys, k)
		}
		if err := client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("cache: tag delete keys: %w", err)
		}
	}

	// Delete the tag sets themselves.
	if len(tagKeys) > 0 {
		if err := client.Del(ctx, tagKeys...).Err(); err != nil {
			return fmt.Errorf("cache: tag delete sets: %w", err)
		}
	}

	return nil
}

// trackKey registers a prefixed key in all tag SETs (Redis SADD).
func (t *TaggedCache) trackKey(ctx context.Context, key string) {
	rd, ok := t.manager.driver.(*RedisDriver)
	if !ok {
		return // Tags are a no-op for non-Redis drivers.
	}
	client := rd.Client()
	prefix := rd.Prefix()
	prefixedKey := prefix + key
	for _, tag := range t.tags {
		_ = client.SAdd(ctx, tagSetKey(prefix, tag), prefixedKey).Err()
	}
}

// FlushTag is a convenience function to flush a single tag without creating a TaggedCache.
//
// Usage:
//
//	cache.FlushTag(ctx, manager, "users")
func FlushTag(ctx context.Context, m *Manager, tag string) error {
	tc := m.Tags(tag)
	return tc.Flush(ctx)
}
