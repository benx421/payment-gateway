// Package db provides database connection and management utilities.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/benx421/payment-gateway/bank/internal/config"

	// Import postgres driver for registration with database/sql)
	_ "github.com/lib/pq"
)

// DB wraps the database connection pool
type DB struct {
	*sql.DB
	logger *slog.Logger
}

// Connect establishes a connection to the database
func Connect(ctx context.Context, cfg *config.DatabaseConfig, logger *slog.Logger) (*DB, error) {
	logger.Info("connecting to database",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.DBName,
	)

	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		logger.Error("failed to open database connection", "error", err)
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		logger.Error("failed to ping database", "error", err)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("successfully connected to database",
		"max_open_conns", cfg.MaxOpenConns,
		"max_idle_conns", cfg.MaxIdleConns,
		"conn_max_lifetime", cfg.ConnMaxLifetime,
	)

	return &DB{
		DB:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection and logs the closure.
func (db *DB) Close() error {
	db.logger.Info("closing database connection")
	return db.DB.Close()
}
