package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisQueue is a Redis list-backed queue.
type RedisQueue struct {
	client   *redis.Client
	key      string
	handlers map[string]func(map[string]any) error
	mu       sync.RWMutex
}

// NewRedisQueue creates a Redis queue.
func NewRedisQueue(client *redis.Client, key string) *RedisQueue {
	if key == "" {
		key = "zatrano:queues:default"
	}
	return &RedisQueue{
		client:   client,
		key:      key,
		handlers: make(map[string]func(map[string]any) error),
	}
}

// Register registers a job handler.
func (q *RedisQueue) Register(name string, handler func(map[string]any) error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[name] = handler
}

// Handler returns a registered handler.
func (q *RedisQueue) Handler(name string) (func(map[string]any) error, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()
	handler, ok := q.handlers[name]
	return handler, ok
}

type redisPayload struct {
	Job         NamedJob  `json:"job"`
	AvailableAt time.Time `json:"available_at"`
	ID          string    `json:"id"`
}

// Push stores a job.
func (q *RedisQueue) Push(job NamedJob, delay ...time.Duration) error {
	wait := time.Duration(0)
	if len(delay) > 0 {
		wait = delay[0]
	}
	payload := redisPayload{
		Job:         job,
		AvailableAt: time.Now().Add(wait),
		ID:          strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return q.client.LPush(context.Background(), q.key, raw).Err()
}

// Pop reserves the next available job.
func (q *RedisQueue) Pop() (*ReservedJob, error) {
	ctx := context.Background()
	for {
		raw, err := q.client.RPop(ctx, q.key).Bytes()
		if err == redis.Nil {
			return nil, fmt.Errorf("sql: no rows in result set")
		}
		if err != nil {
			return nil, err
		}

		var payload redisPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			continue
		}
		if time.Now().Before(payload.AvailableAt) {
			_ = q.client.LPush(ctx, q.key, raw).Err()
			return nil, fmt.Errorf("sql: no rows in result set")
		}

		id, _ := strconv.ParseInt(payload.ID, 10, 64)
		return &ReservedJob{
			ID:  id,
			Job: payload.Job,
			delete: func() error {
				return nil
			},
			release: func(delay time.Duration) error {
				payload.AvailableAt = time.Now().Add(delay)
				raw, err := json.Marshal(payload)
				if err != nil {
					return err
				}
				return q.client.LPush(ctx, q.key, raw).Err()
			},
		}, nil
	}
}

// Size returns queue size.
func (q *RedisQueue) Size() (int, error) {
	n, err := q.client.LLen(context.Background(), q.key).Result()
	return int(n), err
}

// Clear deletes all jobs.
func (q *RedisQueue) Clear() error {
	return q.client.Del(context.Background(), q.key).Err()
}
