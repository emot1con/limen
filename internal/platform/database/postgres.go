package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"limen/internal/platform/config"
)

func NewPostgresConnection(ctx context.Context, cfg *config.Config) (*gorm.DB, error) {
	var lastErr error
	backoff := cfg.PostgresInitialBackoff

	gormCfg := &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)}

	for attempt := 1; attempt <= cfg.PostgresRetryAttempts; attempt++ {
		db, err := gorm.Open(postgres.Open(cfg.PostgresDSN()), gormCfg)
		if err == nil {
			sqlDB, dbErr := db.DB()
			if dbErr == nil {
				sqlDB.SetMaxOpenConns(25)
				sqlDB.SetMaxIdleConns(5)
				sqlDB.SetConnMaxLifetime(5 * time.Minute)
				err = sqlDB.PingContext(ctx)
			} else {
				err = dbErr
			}
			if err == nil {
				return db, nil
			}
			if sqlDB, closeErr := db.DB(); closeErr == nil {
				_ = sqlDB.Close()
			}
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
