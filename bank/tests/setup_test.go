//nolint:errcheck // unchecked errors are acceptable in test files
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benx421/payment-gateway/bank/internal/config"
	"github.com/benx421/payment-gateway/bank/internal/db"
	"github.com/benx421/payment-gateway/bank/internal/handlers"
	"github.com/stretchr/testify/require"
)

// TestServer wraps the HTTP test server and database for integration tests.
type TestServer struct {
	Server   *httptest.Server
	Database *db.DB
	t        *testing.T
}

// SetupTest creates a new test server with a clean database state.
func SetupTest(t *testing.T) *TestServer {
	t.Helper()

	cfg, err := config.Load()
	require.NoError(t, err, "failed to load config")

	// Disable chaos for integration tests
	cfg.App.FailureRate = 0
	cfg.App.MinLatencyMS = 0
	cfg.App.MaxLatencyMS = 0

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	database, err := db.Connect(context.Background(), &cfg.Database, logger)
	require.NoError(t, err, "failed to connect to database")

	resetTestData(t, database)

	router := handlers.NewRouter(database, cfg, logger)
	server := httptest.NewServer(router)

	return &TestServer{
		Server:   server,
		Database: database,
		t:        t,
	}
}

// Close shuts down the test server and database connection.
func (ts *TestServer) Close() {
	ts.Server.Close()
	_ = ts.Database.Close()
}

// URL returns the full URL for a given path.
func (ts *TestServer) URL(path string) string {
	return ts.Server.URL + path
}

func resetTestData(t *testing.T, database *db.DB) {
	t.Helper()

	_, err := database.ExecContext(context.Background(), `
		TRUNCATE TABLE transactions CASCADE;
		TRUNCATE TABLE idempotency_keys CASCADE;
		DELETE FROM accounts;
		INSERT INTO accounts (account_number, cvv, expiry_month, expiry_year, balance_cents, available_balance_cents) VALUES
			('4111111111111111', '123', 12, 2030, 1000000, 1000000),
			('4242424242424242', '456', 6, 2030, 50000, 50000),
			('5555555555554444', '789', 9, 2030, 0, 0),
			('5105105105105100', '321', 3, 2020, 500000, 500000);
	`)
	require.NoError(t, err, "failed to reset test data")
}

// Authorize sends a POST request to create an authorization.
func (ts *TestServer) Authorize(t *testing.T, cardNumber, cvv string, amount int64, idempotencyKey string) *http.Response {
	t.Helper()

	body := map[string]any{
		"card_number": cardNumber,
		"cvv":         cvv,
		"amount":      amount,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, ts.URL("/api/v1/authorizations"), bytes.NewReader(jsonBody))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

// Capture sends a POST request to capture an authorization.
func (ts *TestServer) Capture(t *testing.T, authID string, amount int64, idempotencyKey string) *http.Response {
	t.Helper()

	body := map[string]any{
		"authorization_id": authID,
		"amount":           amount,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, ts.URL("/api/v1/captures"), bytes.NewReader(jsonBody))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

// Void sends a POST request to void an authorization.
func (ts *TestServer) Void(t *testing.T, authID string, idempotencyKey string) *http.Response {
	t.Helper()

	body := map[string]any{
		"authorization_id": authID,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, ts.URL("/api/v1/voids"), bytes.NewReader(jsonBody))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}

// Refund sends a POST request to refund a capture.
func (ts *TestServer) Refund(t *testing.T, captureID string, amount int64, idempotencyKey string) *http.Response {
	t.Helper()

	body := map[string]any{
		"capture_id": captureID,
		"amount":     amount,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, ts.URL("/api/v1/refunds"), bytes.NewReader(jsonBody))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	return resp
}
