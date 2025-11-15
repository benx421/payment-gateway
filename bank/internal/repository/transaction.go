package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/benx421/payment-gateway/bank/internal/db"
	"github.com/benx421/payment-gateway/bank/internal/models"
	"github.com/google/uuid"
)

// TransactionRepository defines the interface for transaction data access
type TransactionRepository interface {
	Create(ctx context.Context, tx *models.Transaction) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error)
	FindByReferenceID(ctx context.Context, refID uuid.UUID, txnType models.TransactionType) (*models.Transaction, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.TransactionStatus) error
}

type transactionRepository struct {
	db *db.DB
}

// NewTransactionRepository creates a new TransactionRepository
func NewTransactionRepository(database *db.DB) TransactionRepository {
	return &transactionRepository{db: database}
}

// Create inserts a new transaction into the database
func (r *transactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}

	var metadataJSON *[]byte
	if tx.Metadata != nil {
		jsonBytes, err := json.Marshal(tx.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = &jsonBytes
	}

	query := `
		INSERT INTO transactions (
			id, account_id, type, amount_cents, currency,
			reference_id, status, expires_at, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, COALESCE($10, NOW()))
	`

	_, err := r.db.ExecContext(
		ctx, query,
		tx.ID,
		tx.AccountID,
		tx.Type,
		tx.AmountCents,
		tx.Currency,
		tx.ReferenceID,
		tx.Status,
		tx.ExpiresAt,
		metadataJSON,
		tx.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

// FindByID retrieves a transaction by its ID
func (r *transactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	query := `
		SELECT id, account_id, type, amount_cents, currency,
		       reference_id, status, expires_at, metadata, created_at
		FROM transactions
		WHERE id = $1
	`

	var tx models.Transaction
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tx.ID,
		&tx.AccountID,
		&tx.Type,
		&tx.AmountCents,
		&tx.Currency,
		&tx.ReferenceID,
		&tx.Status,
		&tx.ExpiresAt,
		&metadataJSON,
		&tx.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction: %w", err)
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &tx.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &tx, nil
}

// FindByReferenceID finds a transaction by its reference_id and type
// This is used to check if a capture/void/refund already exists for an authorization/capture
func (r *transactionRepository) FindByReferenceID(ctx context.Context, refID uuid.UUID, txnType models.TransactionType) (*models.Transaction, error) {
	query := `
		SELECT id, account_id, type, amount_cents, currency,
		       reference_id, status, expires_at, metadata, created_at
		FROM transactions
		WHERE reference_id = $1 AND type = $2
		LIMIT 1
	`

	var tx models.Transaction
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, refID, txnType).Scan(
		&tx.ID,
		&tx.AccountID,
		&tx.Type,
		&tx.AmountCents,
		&tx.Currency,
		&tx.ReferenceID,
		&tx.Status,
		&tx.ExpiresAt,
		&metadataJSON,
		&tx.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error for this use case
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find transaction by reference: %w", err)
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &tx.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &tx, nil
}

// UpdateStatus updates the status of a transaction
func (r *transactionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TransactionStatus) error {
	query := `
		UPDATE transactions
		SET status = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found")
	}

	return nil
}
