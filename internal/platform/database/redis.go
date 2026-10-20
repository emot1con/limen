package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"limen/internal/platform/config"
)

func NewRedisConnection(ctx context.Context, cfg *config.Config) (*redis.Client, error) {
	var lastErr error
	backoff := cfg.RedisInitialBackoff

	for attempt := 1; attempt <= cfg.RedisRetryAttempts; attempt++ {
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr(),
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})
		if err := client.Ping(ctx).Err(); err == nil {
			return client, nil
		} else {
			lastErr = err
			_ = client.Close()
		}

		if attempt == cfg.RedisRetryAttempts {
			break
		}
		if err := WaitForRetry(ctx, backoff); err != nil {
			return nil, fmt.Errorf("wait for redis retry: %w", err)
		}
		backoff = NextBackoff(backoff, cfg.RedisMaxBackoff)
	}

	return nil, fmt.Errorf("connect redis after %d attempts: %w", cfg.RedisRetryAttempts, lastErr)
}
