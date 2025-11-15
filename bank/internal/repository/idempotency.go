package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/db"
	"github.com/benx421/payment-gateway/bank/internal/models"
)

// IdempotencyRepository defines the interface for idempotency key data access
type IdempotencyRepository interface {
	Get(ctx context.Context, key, requestPath string) (*models.IdempotencyKey, error)
	Store(ctx context.Context, idemKey *models.IdempotencyKey) error
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

type idempotencyRepository struct {
	db *db.DB
}

// NewIdempotencyRepository creates a new IdempotencyRepository
func NewIdempotencyRepository(database *db.DB) IdempotencyRepository {
	return &idempotencyRepository{db: database}
}

// Get retrieves a cached idempotency key and its response
func (r *idempotencyRepository) Get(ctx context.Context, key, requestPath string) (*models.IdempotencyKey, error) {
	query := `
		SELECT key, request_path, response_status, response_body, created_at
		FROM idempotency_keys
		WHERE key = $1 AND request_path = $2
	`

	var idemKey models.IdempotencyKey
	err := r.db.QueryRowContext(ctx, query, key, requestPath).Scan(
		&idemKey.Key,
		&idemKey.RequestPath,
		&idemKey.ResponseStatus,
		&idemKey.ResponseBody,
		&idemKey.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error. This means this is a new request
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get idempotency key: %w", err)
	}

	return &idemKey, nil
}

// Store saves an idempotency key with its response
func (r *idempotencyRepository) Store(ctx context.Context, idemKey *models.IdempotencyKey) error {
	query := `
		INSERT INTO idempotency_keys (key, request_path, response_status, response_body, created_at)
		VALUES ($1, $2, $3, $4, COALESCE($5, NOW()))
		ON CONFLICT (key, request_path) DO NOTHING
	`

	_, err := r.db.ExecContext(
		ctx, query,
		idemKey.Key,
		idemKey.RequestPath,
		idemKey.ResponseStatus,
		idemKey.ResponseBody,
		idemKey.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to store idempotency key: %w", err)
	}

	return nil
}

// DeleteOlderThan removes idempotency keys created before the specified time
// This is used for cleanup of keys older than 24 hours
func (r *idempotencyRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := `
		DELETE FROM idempotency_keys
		WHERE created_at < $1
	`

	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old idempotency keys: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("no key deleted")
	}

	return rowsAffected, nil
}
