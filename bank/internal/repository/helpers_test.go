package repository

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/benx421/payment-gateway/bank/internal/config"
	"github.com/benx421/payment-gateway/bank/internal/db"
)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger := cfg.Logger.NewLogger()

	database, err := db.Connect(context.Background(), &cfg.Database, logger)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	runMigrations(t, database)

	return database
}

func runMigrations(t *testing.T, database *db.DB) {
	t.Helper()

	migrationPath := filepath.Join("..", "..", "internal", "db", "migrations", "000001_init.up.sql")
	sqlBytes, err := os.ReadFile(migrationPath) // #nosec G304
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}

	_, err = database.ExecContext(context.Background(), string(sqlBytes))
	if err != nil {
		if err.Error() != "pq: relation \"accounts\" already exists" {
			t.Logf("migration execution completed (tables may already exist)")
		}
	}
}

func cleanupTestDB(t *testing.T, database *db.DB) {
	t.Helper()
	if err := database.Close(); err != nil {
		log.Printf("failed to close test database: %v", err)
	}
}

func truncateTables(t *testing.T, database *db.DB) {
	t.Helper()

	tables := []string{"transactions", "idempotency_keys"}
	for _, table := range tables {
		_, err := database.ExecContext(context.Background(), "TRUNCATE TABLE "+table+" CASCADE")
		if err != nil {
			t.Fatalf("failed to truncate table %s: %v", table, err)
		}
	}

	_, err := database.ExecContext(context.Background(), `
		DELETE FROM accounts;
		INSERT INTO accounts (account_number, cvv, expiry_month, expiry_year, balance_cents, available_balance_cents) VALUES
			('4111111111111111', '123', 12, 2030, 1000000, 1000000),
			('4242424242424242', '456', 6, 2030, 50000, 50000),
			('5555555555554444', '789', 9, 2030, 0, 0),
			('5105105105105100', '321', 3, 2020, 500000, 500000);
	`)
	if err != nil {
		t.Fatalf("failed to reset accounts: %v", err)
	}
}
