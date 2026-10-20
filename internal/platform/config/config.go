package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                 string
	LogLevel               string
	AppPort                int
	PostgresHost           string
	PostgresPort           int
	PostgresUser           string
	PostgresPassword       string
	PostgresDB             string
	PostgresSSLMode        string
	PostgresRetryAttempts  int
	PostgresInitialBackoff time.Duration
	PostgresMaxBackoff     time.Duration
	RedisHost              string
	RedisPort              int
	RedisPassword          string
	RedisDB                int
	RedisRetryAttempts     int
	RedisInitialBackoff    time.Duration
	RedisMaxBackoff        time.Duration
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		PostgresHost:           getEnv("POSTGRES_HOST", "localhost"),
		PostgresUser:           getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword:       getEnv("POSTGRES_PASSWORD", ""),
		PostgresDB:             getEnv("POSTGRES_DB", "limen"),
		PostgresSSLMode:        getEnv("POSTGRES_SSLMODE", "disable"),
		RedisHost:              getEnv("REDIS_HOST", "localhost"),
		RedisPassword:          getEnv("REDIS_PASSWORD", ""),
		PostgresInitialBackoff: getEnvDuration("POSTGRES_RETRY_INITIAL_BACKOFF", time.Second),
		PostgresMaxBackoff:     getEnvDuration("POSTGRES_RETRY_MAX_BACKOFF", 10*time.Second),
		RedisInitialBackoff:    getEnvDuration("REDIS_RETRY_INITIAL_BACKOFF", time.Second),
		RedisMaxBackoff:        getEnvDuration("REDIS_RETRY_MAX_BACKOFF", 10*time.Second),
	}

	var err error
	if cfg.AppPort, err = getEnvInt("APP_PORT", 8000); err != nil {
		return nil, fmt.Errorf("invalid APP_PORT: %w", err)
	}
	if cfg.PostgresPort, err = getEnvInt("POSTGRES_PORT", 5432); err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_PORT: %w", err)
	}
	if cfg.RedisPort, err = getEnvInt("REDIS_PORT", 6379); err != nil {
		return nil, fmt.Errorf("invalid REDIS_PORT: %w", err)
	}
	if cfg.RedisDB, err = getEnvInt("REDIS_DB", 0); err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}
	if cfg.PostgresRetryAttempts, err = getEnvInt("POSTGRES_RETRY_ATTEMPTS", 5); err != nil {
		return nil, fmt.Errorf("invalid POSTGRES_RETRY_ATTEMPTS: %w", err)
	}
	if cfg.RedisRetryAttempts, err = getEnvInt("REDIS_RETRY_ATTEMPTS", 5); err != nil {
		return nil, fmt.Errorf("invalid REDIS_RETRY_ATTEMPTS: %w", err)
	}

	if cfg.PostgresPassword == "" {
		return nil, fmt.Errorf("POSTGRES_PASSWORD is required")
	}
	if cfg.PostgresRetryAttempts < 1 || cfg.RedisRetryAttempts < 1 {
		return nil, fmt.Errorf("retry attempts must be at least 1")
	}
	if cfg.PostgresInitialBackoff <= 0 || cfg.PostgresMaxBackoff <= 0 || cfg.RedisInitialBackoff <= 0 || cfg.RedisMaxBackoff <= 0 {
		return nil, fmt.Errorf("retry backoff durations must be positive")
	}

	return cfg, nil
}

func (c *Config) PostgresDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		quoteDSN(c.PostgresHost), c.PostgresPort, quoteDSN(c.PostgresUser), quoteDSN(c.PostgresPassword), quoteDSN(c.PostgresDB), quoteDSN(c.PostgresSSLMode))
}

func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.RedisHost, c.RedisPort)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("must be a non-negative integer")
	}
	return parsed, nil
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func quoteDSN(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
}
