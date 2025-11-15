package repository

import (
	"context"
	"testing"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionRepository_Create(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)
	truncateTables(t, database)

	repo := NewTransactionRepository(database)
	accountRepo := NewAccountRepository(database)

	account, err := accountRepo.FindByAccountNumber(context.Background(), "4532015112830366")
	require.NoError(t, err, "failed to get account")

	tests := []struct {
		tx      *models.Transaction
		name    string
		wantErr bool
	}{
		{
			name: "create AUTH_HOLD transaction",
			tx: &models.Transaction{
				AccountID:   account.ID,
				Type:        models.TransactionTypeAuthHold,
				AmountCents: 10000,
				Currency:    "USD",
				Status:      models.TransactionStatusActive,
				ExpiresAt:   timePtr(time.Now().Add(7 * 24 * time.Hour)),
			},
			wantErr: false,
		},
		{
			name: "create transaction with metadata",
			tx: &models.Transaction{
				AccountID:   account.ID,
				Type:        models.TransactionTypeCapture,
				AmountCents: 5000,
				Currency:    "USD",
				Status:      models.TransactionStatusCompleted,
				Metadata: map[string]any{
					"merchant_id": "test_merchant",
					"order_id":    "12345",
				},
			},
			wantErr: false,
		},
		{
			name: "create transaction with pre-set ID",
			tx: &models.Transaction{
				ID:          uuid.New(),
				AccountID:   account.ID,
				Type:        models.TransactionTypeVoid,
				AmountCents: 0,
				Currency:    "USD",
				Status:      models.TransactionStatusCompleted,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalID := tt.tx.ID

			err := repo.Create(context.Background(), tt.tx)

			if tt.wantErr {
				assert.Error(t, err, "expected error")
				return
			}

			require.NoError(t, err, "unexpected error")

			assert.NotEqual(t, uuid.Nil, tt.tx.ID, "transaction ID should not be nil UUID after create")

			if originalID != uuid.Nil {
				assert.Equal(t, originalID, tt.tx.ID, "transaction ID should be preserved")
			}

			retrieved, err := repo.FindByID(context.Background(), tt.tx.ID)
			require.NoError(t, err, "failed to retrieve created transaction")

			assert.Equal(t, tt.tx.Type, retrieved.Type, "type mismatch")
			assert.Equal(t, tt.tx.AmountCents, retrieved.AmountCents, "amount mismatch")

			if tt.tx.Metadata != nil {
				assert.NotNil(t, retrieved.Metadata, "metadata should not be nil")
			}
		})
	}
}

func TestTransactionRepository_FindByID(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)
	truncateTables(t, database)

	repo := NewTransactionRepository(database)
	accountRepo := NewAccountRepository(database)

	account, err := accountRepo.FindByAccountNumber(context.Background(), "4532015112830366")
	require.NoError(t, err, "failed to get account")

	tx := &models.Transaction{
		AccountID:   account.ID,
		Type:        models.TransactionTypeAuthHold,
		AmountCents: 10000,
		Currency:    "USD",
		Status:      models.TransactionStatusActive,
	}
	err = repo.Create(context.Background(), tx)
	require.NoError(t, err, "failed to create transaction")

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing transaction",
			id:      tx.ID,
			wantErr: false,
		},
		{
			name:    "non-existent transaction",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := repo.FindByID(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err, "expected error")
				return
			}

			require.NoError(t, err, "unexpected error")
			assert.Equal(t, tt.id, retrieved.ID, "ID mismatch")
		})
	}
}

func TestTransactionRepository_FindByReferenceID(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)
	truncateTables(t, database)

	repo := NewTransactionRepository(database)
	accountRepo := NewAccountRepository(database)

	account, err := accountRepo.FindByAccountNumber(context.Background(), "4532015112830366")
	require.NoError(t, err, "failed to get account")

	authTx := &models.Transaction{
		AccountID:   account.ID,
		Type:        models.TransactionTypeAuthHold,
		AmountCents: 10000,
		Currency:    "USD",
		Status:      models.TransactionStatusActive,
	}
	err = repo.Create(context.Background(), authTx)
	require.NoError(t, err, "failed to create auth transaction")

	captureTx := &models.Transaction{
		AccountID:   account.ID,
		Type:        models.TransactionTypeCapture,
		AmountCents: 10000,
		Currency:    "USD",
		Status:      models.TransactionStatusCompleted,
		ReferenceID: &authTx.ID,
	}
	err = repo.Create(context.Background(), captureTx)
	require.NoError(t, err, "failed to create capture transaction")

	tests := []struct {
		name      string
		txnType   models.TransactionType
		refID     uuid.UUID
		wantFound bool
	}{
		{
			name:      "find existing capture by reference",
			refID:     authTx.ID,
			txnType:   models.TransactionTypeCapture,
			wantFound: true,
		},
		{
			name:      "no void exists for this auth",
			refID:     authTx.ID,
			txnType:   models.TransactionTypeVoid,
			wantFound: false,
		},
		{
			name:      "non-existent reference",
			refID:     uuid.New(),
			txnType:   models.TransactionTypeCapture,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.FindByReferenceID(context.Background(), tt.refID, tt.txnType)
			require.NoError(t, err, "unexpected error")

			if tt.wantFound {
				require.NotNil(t, result, "expected to find transaction")
				require.NotNil(t, result.ReferenceID, "reference ID should not be nil")
				assert.Equal(t, tt.refID, *result.ReferenceID, "reference ID mismatch")
				assert.Equal(t, tt.txnType, result.Type, "type mismatch")
			} else {
				assert.Nil(t, result, "expected nil transaction")
			}
		})
	}
}

func TestTransactionRepository_UpdateStatus(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)
	truncateTables(t, database)

	repo := NewTransactionRepository(database)
	accountRepo := NewAccountRepository(database)

	account, err := accountRepo.FindByAccountNumber(context.Background(), "4532015112830366")
	require.NoError(t, err, "failed to get account")

	tx := &models.Transaction{
		AccountID:   account.ID,
		Type:        models.TransactionTypeAuthHold,
		AmountCents: 10000,
		Currency:    "USD",
		Status:      models.TransactionStatusActive,
	}
	err = repo.Create(context.Background(), tx)
	require.NoError(t, err, "failed to create transaction")

	tests := []struct {
		name      string
		newStatus models.TransactionStatus
		txID      uuid.UUID
		wantErr   bool
	}{
		{
			name:      "update to completed",
			txID:      tx.ID,
			newStatus: models.TransactionStatusCompleted,
			wantErr:   false,
		},
		{
			name:      "update non-existent transaction",
			txID:      uuid.New(),
			newStatus: models.TransactionStatusCompleted,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateStatus(context.Background(), tt.txID, tt.newStatus)

			if tt.wantErr {
				assert.Error(t, err, "expected error")
				return
			}

			require.NoError(t, err, "unexpected error")

			updated, err := repo.FindByID(context.Background(), tt.txID)
			require.NoError(t, err, "failed to retrieve updated transaction")

			assert.Equal(t, tt.newStatus, updated.Status, "status mismatch")
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
