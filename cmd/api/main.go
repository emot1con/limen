package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"limen/internal/adapter/http/router"
	"limen/internal/platform/config"
	"limen/internal/platform/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, err := config.NewLogger(cfg)
	if err != nil {
		return fmt.Errorf("setup logger: %w", err)
	}

	postgresDB, err := database.NewPostgresConnection(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	sqlDB, err := postgresDB.DB()
	if err != nil {
		return fmt.Errorf("get postgres connection pool: %w", err)
	}
	defer sqlDB.Close()

	redisClient, err := config.NewRedisConnection(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}
	defer redisClient.Close()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	appRouter := router.New(postgresDB, redisClient, logger)

	appRouter.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		response := gin.H{}

		if err := sqlDB.PingContext(ctx); err != nil {
			response["postgres"] = "error"
		} else {
			response["postgres"] = "ok"
		}

		if err := redisClient.Ping(ctx).Err(); err != nil {
			response["redis"] = "error"
		} else {
			response["redis"] = "ok"
		}

		c.JSON(http.StatusOK, response)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.AppPort),
		Handler:           appRouter,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.WithField("address", server.Addr).Info("HTTP server started")
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("run HTTP server: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	logger.Info("HTTP server stopped")
	return nil
}
