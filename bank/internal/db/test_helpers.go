package db

import (
	"database/sql"
	"io"
	"log/slog"
)

// NewTestDB creates a DB instance for testing with a no-op logger
// This is only for use in tests where logging output is not needed
func NewTestDB(sqlDB *sql.DB) *DB {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &DB{
		DB:     sqlDB,
		logger: logger,
	}
}
