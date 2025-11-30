package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/models"
	"github.com/benx421/payment-gateway/bank/internal/repository/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCaptureService_PerformCapture(t *testing.T) {
	t.Run("successful capture", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewCaptureService(nil)
		ctx := context.Background()

		authID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000
		expiresAt := time.Now().Add(24 * time.Hour)

		authTx := &models.Transaction{
			ID:          authID,
			AccountID:   accountID,
			Type:        models.TransactionTypeAuthHold,
			AmountCents: amount,
			Currency:    "USD",
			Status:      models.TransactionStatusActive,
			ExpiresAt:   &expiresAt,
		}

		mockTxRepo.On("FindByIDForUpdate", ctx, authID).Return(authTx, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).Return(nil)
		mockTxRepo.On("UpdateStatus", ctx, authID, models.TransactionStatusCompleted).Return(nil)
		mockAccountRepo.On("AdjustBalances", ctx, accountID, int64(-10000), int64(0)).Return(nil)

		result, err := service.performCapture(ctx, mockTxRepo, mockAccountRepo, authID, amount)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, models.TransactionTypeCapture, result.Type)
		assert.Equal(t, amount, result.AmountCents)
		assert.Equal(t, authID, *result.ReferenceID)
		assert.Equal(t, models.TransactionStatusCompleted, result.Status)

		mockTxRepo.AssertExpectations(t)
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("authorization not found", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewCaptureService(nil)
		ctx := context.Background()

		authID := uuid.New()
		var amount int64 = 10000

		mockTxRepo.On("FindByIDForUpdate", ctx, authID).Return(nil, sql.ErrNoRows)

		result, err := service.performCapture(ctx, mockTxRepo, mockAccountRepo, authID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeAuthNotFound, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("wrong transaction type", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewCaptureService(nil)
		ctx := context.Background()

		authID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		// Return a CAPTURE instead of AUTH_HOLD
		captureTx := &models.Transaction{
			ID:          authID,
			AccountID:   accountID,
			Type:        models.TransactionTypeCapture,
			AmountCents: amount,
			Status:      models.TransactionStatusCompleted,
		}

		mockTxRepo.On("FindByIDForUpdate", ctx, authID).Return(captureTx, nil)

		result, err := service.performCapture(ctx, mockTxRepo, mockAccountRepo, authID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeAuthNotFound, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("authorization already used", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewCaptureService(nil)
		ctx := context.Background()

		authID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		authTx := &models.Transaction{
			ID:          authID,
			AccountID:   accountID,
			Type:        models.TransactionTypeAuthHold,
			AmountCents: amount,
			Status:      models.TransactionStatusCompleted, // Already used
		}

		mockTxRepo.On("FindByIDForUpdate", ctx, authID).Return(authTx, nil)

		result, err := service.performCapture(ctx, mockTxRepo, mockAccountRepo, authID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeAuthAlreadyUsed, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("authorization expired", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewCaptureService(nil)
		ctx := context.Background()

		authID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000
		expiresAt := time.Now().Add(-1 * time.Hour) // Expired

		authTx := &models.Transaction{
			ID:          authID,
			AccountID:   accountID,
			Type:        models.TransactionTypeAuthHold,
			AmountCents: amount,
			Status:      models.TransactionStatusActive,
			ExpiresAt:   &expiresAt,
		}

		mockTxRepo.On("FindByIDForUpdate", ctx, authID).Return(authTx, nil)

		result, err := service.performCapture(ctx, mockTxRepo, mockAccountRepo, authID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeAuthExpired, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("amount mismatch", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewCaptureService(nil)
		ctx := context.Background()

		authID := uuid.New()
		accountID := uuid.New()
		var authAmount int64 = 10000
		var captureAmount int64 = 5000 // Different amount

		authTx := &models.Transaction{
			ID:          authID,
			AccountID:   accountID,
			Type:        models.TransactionTypeAuthHold,
			AmountCents: authAmount,
			Status:      models.TransactionStatusActive,
		}

		mockTxRepo.On("FindByIDForUpdate", ctx, authID).Return(authTx, nil)

		result, err := service.performCapture(ctx, mockTxRepo, mockAccountRepo, authID, captureAmount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeAmountMismatch, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("already captured - duplicate error", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewCaptureService(nil)
		ctx := context.Background()

		authID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		authTx := &models.Transaction{
			ID:          authID,
			AccountID:   accountID,
			Type:        models.TransactionTypeAuthHold,
			AmountCents: amount,
			Status:      models.TransactionStatusActive,
		}

		mockTxRepo.On("FindByIDForUpdate", ctx, authID).Return(authTx, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).
			Return(models.ErrDuplicateTransaction)

		result, err := service.performCapture(ctx, mockTxRepo, mockAccountRepo, authID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeAlreadyCaptured, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("status update fails", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewCaptureService(nil)
		ctx := context.Background()

		authID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		authTx := &models.Transaction{
			ID:          authID,
			AccountID:   accountID,
			Type:        models.TransactionTypeAuthHold,
			AmountCents: amount,
			Status:      models.TransactionStatusActive,
		}

		mockTxRepo.On("FindByIDForUpdate", ctx, authID).Return(authTx, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).Return(nil)
		mockTxRepo.On("UpdateStatus", ctx, authID, models.TransactionStatusCompleted).
			Return(assert.AnError)

		result, err := service.performCapture(ctx, mockTxRepo, mockAccountRepo, authID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeInternalError, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("balance adjustment fails", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewCaptureService(nil)
		ctx := context.Background()

		authID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		authTx := &models.Transaction{
			ID:          authID,
			AccountID:   accountID,
			Type:        models.TransactionTypeAuthHold,
			AmountCents: amount,
			Status:      models.TransactionStatusActive,
		}

		mockTxRepo.On("FindByIDForUpdate", ctx, authID).Return(authTx, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).Return(nil)
		mockTxRepo.On("UpdateStatus", ctx, authID, models.TransactionStatusCompleted).Return(nil)
		mockAccountRepo.On("AdjustBalances", ctx, accountID, int64(-10000), int64(0)).
			Return(assert.AnError)

		result, err := service.performCapture(ctx, mockTxRepo, mockAccountRepo, authID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeInternalError, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
		mockAccountRepo.AssertExpectations(t)
	})
}
