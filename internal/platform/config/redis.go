package config

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisConnection(ctx context.Context, cfg *Config) (*redis.Client, error) {
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
		if err := waitForRetry(ctx, backoff); err != nil {
			return nil, fmt.Errorf("wait for redis retry: %w", err)
		}
		backoff = nextBackoff(backoff, cfg.RedisMaxBackoff)
	}

	return nil, fmt.Errorf("connect redis after %d attempts: %w", cfg.RedisRetryAttempts, lastErr)
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func nextBackoff(current, maximum time.Duration) time.Duration {
	if current >= maximum/2 {
		return maximum
	}
	return current * 2
}
