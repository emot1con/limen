package config

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

func NewLogger(cfg *Config) (*logrus.Logger, error) {
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid LOG_LEVEL: %w", err)
	}

	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(level)
	logger.SetReportCaller(false)
	if cfg.AppEnv == "production" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	}
	return logger, nil
}
