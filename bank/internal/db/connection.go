// Package db provides database connection and management utilities.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/benx421/payment-gateway/bank/internal/config"
	"github.com/lib/pq"
)

// Executor defines the interface for executing database queries
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// DB wraps the database connection pool
type DB struct {
	*sql.DB
	logger *slog.Logger
}

// Tx wraps a database transaction
type Tx struct {
	*sql.Tx
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

// BeginTx starts a new database transaction with the specified isolation level
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		db.logger.Error("failed to begin transaction", "error", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	db.logger.Debug("transaction started")
	return &Tx{
		Tx:     tx,
		logger: db.logger,
	}, nil
}

// Commit commits the transaction
func (tx *Tx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		tx.logger.Error("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tx.logger.Debug("transaction committed")
	return nil
}

// Rollback rolls back the transaction
func (tx *Tx) Rollback() error {
	if err := tx.Tx.Rollback(); err != nil {
		if errors.Is(err, sql.ErrTxDone) {
			tx.logger.Debug("transaction already closed, ignoring rollback")
			return nil
		}
		tx.logger.Error("failed to rollback transaction", "error", err)
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	tx.logger.Debug("transaction rolled back")
	return nil
}

// IsUniqueViolation checks if the error is a PostgreSQL unique constraint violation
func IsUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// 23505 is the PostgreSQL error code for unique_violation
		return pqErr.Code == "23505"
	}
	return false
}
