package redisx

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis connection settings.
type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// Connect opens a Redis client and pings it.
func Connect(cfg Config) (*redis.Client, error) {
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port == "" {
		cfg.Port = "6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Host + ":" + cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis unavailable: %w", err)
	}
	return client, nil
}

// ClientFrom resolves a Redis client from an any value.
func ClientFrom(v any) *redis.Client {
	c, _ := v.(*redis.Client)
	return c
}

// ParseDB converts a string DB index.
func ParseDB(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}
