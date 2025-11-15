package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountRepository_FindByAccountNumber(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)

	repo := NewAccountRepository(database)

	tests := []struct {
		name          string
		accountNumber string
		wantCVV       string
		wantErr       bool
	}{
		{
			name:          "existing account",
			accountNumber: "4532015112830366",
			wantErr:       false,
			wantCVV:       "123",
		},
		{
			name:          "non-existent account",
			accountNumber: "9999999999999999",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, err := repo.FindByAccountNumber(context.Background(), tt.accountNumber)

			if tt.wantErr {
				assert.Error(t, err, "expected error")
				assert.Nil(t, account, "expected nil account")
				return
			}

			require.NoError(t, err, "unexpected error")
			require.NotNil(t, account, "expected account")

			assert.Equal(t, tt.accountNumber, account.AccountNumber, "account number mismatch")
			assert.Equal(t, tt.wantCVV, account.CVV, "CVV mismatch")
			assert.NotEqual(t, uuid.Nil, account.ID, "account ID should not be nil")
		})
	}
}

func TestAccountRepository_FindByID(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)

	repo := NewAccountRepository(database)

	existingAccount, setupErr := repo.FindByAccountNumber(context.Background(), "4532015112830366")
	require.NoError(t, setupErr, "failed to get existing account")

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing account by ID",
			id:      existingAccount.ID,
			wantErr: false,
		},
		{
			name:    "non-existent account",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account, err := repo.FindByID(context.Background(), tt.id)

			if tt.wantErr {
				assert.Error(t, err, "expected error")
				return
			}

			require.NoError(t, err, "unexpected error")
			assert.Equal(t, tt.id, account.ID, "account ID mismatch")
		})
	}
}

func TestAccountRepository_AdjustBalances(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)

	repo := NewAccountRepository(database)

	account, setupErr := repo.FindByAccountNumber(context.Background(), "4532015112830366")
	require.NoError(t, setupErr, "failed to get existing account")

	initialBalance := account.BalanceCents
	initialAvailable := account.AvailableBalanceCents

	tests := []struct {
		name                  string
		accountID             uuid.UUID
		balanceDelta          int64
		availableBalanceDelta int64
		wantErr               bool
		checkBalance          bool
	}{
		{
			name:                  "decrease available balance (authorization)",
			accountID:             account.ID,
			balanceDelta:          0,
			availableBalanceDelta: -10000, // -$100
			wantErr:               false,
			checkBalance:          true,
		},
		{
			name:                  "decrease both balances (capture)",
			accountID:             account.ID,
			balanceDelta:          -10000, // -$100
			availableBalanceDelta: 0,
			wantErr:               false,
			checkBalance:          true,
		},
		{
			name:                  "increase balances (refund)",
			accountID:             account.ID,
			balanceDelta:          5000, // +$50
			availableBalanceDelta: 5000, // +$50
			wantErr:               false,
			checkBalance:          true,
		},
		{
			name:                  "non-existent account",
			accountID:             uuid.New(),
			balanceDelta:          0,
			availableBalanceDelta: -100,
			wantErr:               true,
			checkBalance:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var currentBalance, currentAvailable int64
			if tt.checkBalance {
				acc, err := repo.FindByID(context.Background(), tt.accountID)
				require.NoError(t, err, "failed to get account before adjustment")
				currentBalance = acc.BalanceCents
				currentAvailable = acc.AvailableBalanceCents
			}

			err := repo.AdjustBalances(
				context.Background(),
				tt.accountID,
				tt.balanceDelta,
				tt.availableBalanceDelta,
			)

			if tt.wantErr {
				assert.Error(t, err, "expected error")
				assert.Contains(t, err.Error(), "not found", "expected 'not found' error")
				return
			}

			require.NoError(t, err, "unexpected error")

			if tt.checkBalance {
				updatedAccount, err := repo.FindByID(context.Background(), tt.accountID)
				require.NoError(t, err, "failed to get account after adjustment")

				expectedBalance := currentBalance + tt.balanceDelta
				expectedAvailable := currentAvailable + tt.availableBalanceDelta

				assert.Equal(t, expectedBalance, updatedAccount.BalanceCents, "balance_cents mismatch")
				assert.Equal(t, expectedAvailable, updatedAccount.AvailableBalanceCents, "available_balance_cents mismatch")
			}
		})
	}

	finalAccount, err := repo.FindByID(context.Background(), account.ID)
	require.NoError(t, err, "failed to get final account state")

	// Sum of all deltas: 0, -10000, 5000 = -5000
	expectedFinalBalance := initialBalance - 5000
	assert.Equal(t, expectedFinalBalance, finalAccount.BalanceCents, "final balance_cents mismatch")

	// Sum of all available deltas: -10000, 0, 5000 = -5000
	expectedFinalAvailable := initialAvailable - 5000
	assert.Equal(t, expectedFinalAvailable, finalAccount.AvailableBalanceCents, "final available_balance_cents mismatch")
}

func TestAccountRepository_AdjustBalances_Concurrent(t *testing.T) {
	database := setupTestDB(t)
	defer cleanupTestDB(t, database)
	truncateTables(t, database)

	repo := NewAccountRepository(database)

	account, setupErr := repo.FindByAccountNumber(context.Background(), "4532015112830366")
	require.NoError(t, setupErr, "failed to get account")

	initialBalance := account.BalanceCents

	const numGoroutines = 10
	const delta = -1000

	errCh := make(chan error, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			errCh <- repo.AdjustBalances(context.Background(), account.ID, delta, 0)
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, <-errCh, "concurrent adjustment failed")
	}

	finalAccount, err := repo.FindByID(context.Background(), account.ID)
	require.NoError(t, err, "failed to get final account")

	expectedBalance := initialBalance + (numGoroutines * delta)
	assert.Equal(t, expectedBalance, finalAccount.BalanceCents, "concurrent updates lost update detected!")
}
