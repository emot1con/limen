package database

import (
	"context"
	"database/sql"
	"fmt"
	"limen/internal/platform/config"
	"time"

	_ "github.com/lib/pq"
)

func NewPostgresConnection(ctx context.Context, cfg *config.Config) (*sql.DB, error) {
	var lastErr error
	backoff := cfg.PostgresInitialBackoff

	for attempt := 1; attempt <= cfg.PostgresRetryAttempts; attempt++ {
		db, err := sql.Open("postgres", cfg.PostgresDSN())
		if err == nil {
			db.SetMaxOpenConns(25)
			db.SetMaxIdleConns(5)
			db.SetConnMaxLifetime(5 * time.Minute)
			err = db.PingContext(ctx)
			if err == nil {
				return db, nil
			}
			_ = db.Close()
		}

		lastErr = err
		if attempt == cfg.PostgresRetryAttempts {
			break
		}
		if err := WaitForRetry(ctx, backoff); err != nil {
			return nil, fmt.Errorf("wait for postgres retry: %w", err)
		}
		backoff = NextBackoff(backoff, cfg.PostgresMaxBackoff)
	}

	return nil, fmt.Errorf("connect postgres after %d attempts: %w", cfg.PostgresRetryAttempts, lastErr)
}

func WaitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func NextBackoff(current, maximum time.Duration) time.Duration {
	if current >= maximum/2 {
		return maximum
	}
	return current * 2
}
