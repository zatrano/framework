package queue

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisQueuePrefix   = "zatrano:queue:"
	redisDelayedPrefix = "zatrano:queue:delayed:"
)

// RedisDriver is a queue backend that uses Redis lists (LPUSH/BRPOP) for
// FIFO processing and sorted sets (ZADD) for delayed jobs.
type RedisDriver struct {
	client  *redis.Client
	timeout time.Duration // BRPOP timeout
}

// RedisDriverOption configures the Redis queue driver.
type RedisDriverOption func(*RedisDriver)

// WithPopTimeout sets the BRPOP blocking timeout (default 5s).
func WithPopTimeout(d time.Duration) RedisDriverOption {
	return func(r *RedisDriver) { r.timeout = d }
}

// NewRedisDriver creates a Redis-backed queue driver.
func NewRedisDriver(client *redis.Client, opts ...RedisDriverOption) *RedisDriver {
	d := &RedisDriver{client: client, timeout: 5 * time.Second}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Ensure RedisDriver implements Driver at compile time.
var _ Driver = (*RedisDriver)(nil)

func queueKey(name string) string   { return redisQueuePrefix + name }
func delayedKey(name string) string { return redisDelayedPrefix + name }

// Push adds a payload to the ready queue using LPUSH (tail of BRPOP FIFO).
func (r *RedisDriver) Push(ctx context.Context, queue string, payload []byte) error {
	return r.client.LPush(ctx, queueKey(queue), payload).Err()
}

// Pop blocks until a payload is available using BRPOP (head of FIFO).
// Returns (nil, nil) when context is cancelled.
func (r *RedisDriver) Pop(ctx context.Context, queues ...string) ([]byte, error) {
	keys := make([]string, len(queues))
	for i, q := range queues {
		keys[i] = queueKey(q)
	}

	result, err := r.client.BRPop(ctx, r.timeout, keys...).Result()
	if err != nil {
		if err == redis.Nil || ctx.Err() != nil {
			return nil, nil
		}
		return nil, fmt.Errorf("queue pop: %w", err)
	}
	if len(result) < 2 {
		return nil, nil
	}
	return []byte(result[1]), nil
}



// LaterAt schedules a payload at the given Unix timestamp.
func (r *RedisDriver) LaterAt(ctx context.Context, queue string, at time.Time, payload []byte) error {
	score := float64(at.UnixMilli())
	return r.client.ZAdd(ctx, delayedKey(queue), redis.Z{
		Score:  score,
		Member: string(payload),
	}).Err()
}

// MigrateDelayed moves delayed payloads whose score <= now into the ready queue.
func (r *RedisDriver) MigrateDelayed(ctx context.Context, queue string) error {
	now := strconv.FormatFloat(float64(time.Now().UnixMilli()), 'f', 0, 64)
	dk := delayedKey(queue)

	members, err := r.client.ZRangeByScore(ctx, dk, &redis.ZRangeBy{
		Min: "-inf",
		Max: now,
	}).Result()
	if err != nil {
		return fmt.Errorf("queue migrate scan: %w", err)
	}
	if len(members) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for _, m := range members {
		pipe.LPush(ctx, queueKey(queue), m)
	}
	// Remove migrated entries from the sorted set.
	zMembers := make([]any, len(members))
	for i, m := range members {
		zMembers[i] = m
	}
	pipe.ZRem(ctx, dk, zMembers...)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("queue migrate exec: %w", err)
	}
	return nil
}

// Size returns the number of pending payloads.
func (r *RedisDriver) Size(ctx context.Context, queue string) (int64, error) {
	return r.client.LLen(ctx, queueKey(queue)).Result()
}

// Flush removes all payloads from ready and delayed queues.
func (r *RedisDriver) Flush(ctx context.Context, queue string) error {
	pipe := r.client.Pipeline()
	pipe.Del(ctx, queueKey(queue))
	pipe.Del(ctx, delayedKey(queue))
	_, err := pipe.Exec(ctx)
	return err
}

// Close does not close the shared Redis client.
func (r *RedisDriver) Close() error { return nil }
