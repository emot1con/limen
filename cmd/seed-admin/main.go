package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"limen/internal/platform/config"
	"limen/internal/platform/database"
	"limen/internal/usecase/auth"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL")))
	fullName := strings.TrimSpace(os.Getenv("ADMIN_FULL_NAME"))
	password := os.Getenv("ADMIN_PASSWORD")
	if email == "" || fullName == "" || password == "" {
		return errors.New("ADMIN_EMAIL, ADMIN_FULL_NAME, and ADMIN_PASSWORD are required")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := database.NewPostgresConnection(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get postgres connection pool: %w", err)
	}
	defer sqlDB.Close()

	hash, err := auth.HashArgon2idPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	tx, err := sqlDB.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(context.Background(), `
		INSERT INTO roles (code, name)
		VALUES ('admin', 'Admin')
		ON CONFLICT (code) DO NOTHING
	`); err != nil {
		return fmt.Errorf("seed admin role: %w", err)
	}

	var userID string
	if err := tx.QueryRowContext(context.Background(), `
		INSERT INTO users (email, password_hash, full_name, status)
		VALUES ($1, $2, $3, 'active')
		ON CONFLICT (email) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
			full_name = EXCLUDED.full_name,
			status = 'active',
			updated_at = now(),
			deleted_at = NULL
		RETURNING id
	`, email, hash, fullName).Scan(&userID); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}

	if _, err := tx.ExecContext(context.Background(), `
		INSERT INTO user_roles (user_id, role_code)
		VALUES ($1, 'admin')
		ON CONFLICT (user_id, role_code) DO NOTHING
	`, userID); err != nil {
		return fmt.Errorf("seed admin role assignment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit admin seed: %w", err)
	}
	fmt.Printf("admin seeded: %s\n", email)
	return nil
}
