package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/benx421/payment-gateway/bank/internal/models"
	"github.com/benx421/payment-gateway/bank/internal/repository/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestVoidService_PerformVoid(t *testing.T) {
	t.Run("successful void", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewVoidService(nil)
		ctx := context.Background()

		authID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		authTx := &models.Transaction{
			ID:          authID,
			AccountID:   accountID,
			Type:        models.TransactionTypeAuthHold,
			AmountCents: amount,
			Currency:    "USD",
			Status:      models.TransactionStatusActive,
		}

		mockTxRepo.On("FindByID", ctx, authID).Return(authTx, nil)
		mockTxRepo.On("FindByReferenceID", ctx, authID, models.TransactionTypeCapture).Return(nil, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).Return(nil)
		mockTxRepo.On("UpdateStatus", ctx, authID, models.TransactionStatusCompleted).Return(nil)
		mockAccountRepo.On("AdjustBalances", ctx, accountID, int64(0), int64(10000)).Return(nil)

		result, err := service.performVoid(ctx, mockTxRepo, mockAccountRepo, authID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, models.TransactionTypeVoid, result.Type)
		assert.Equal(t, amount, result.AmountCents)
		assert.Equal(t, authID, *result.ReferenceID)
		assert.Equal(t, models.TransactionStatusCompleted, result.Status)

		mockTxRepo.AssertExpectations(t)
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("authorization not found", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewVoidService(nil)
		ctx := context.Background()

		authID := uuid.New()

		mockTxRepo.On("FindByID", ctx, authID).Return(nil, sql.ErrNoRows)

		result, err := service.performVoid(ctx, mockTxRepo, mockAccountRepo, authID)

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
		service := NewVoidService(nil)
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

		mockTxRepo.On("FindByID", ctx, authID).Return(captureTx, nil)

		result, err := service.performVoid(ctx, mockTxRepo, mockAccountRepo, authID)

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
		service := NewVoidService(nil)
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

		mockTxRepo.On("FindByID", ctx, authID).Return(authTx, nil)

		result, err := service.performVoid(ctx, mockTxRepo, mockAccountRepo, authID)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeAuthAlreadyUsed, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("authorization already captured", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewVoidService(nil)
		ctx := context.Background()

		authID := uuid.New()
		accountID := uuid.New()
		captureID := uuid.New()
		var amount int64 = 10000

		authTx := &models.Transaction{
			ID:          authID,
			AccountID:   accountID,
			Type:        models.TransactionTypeAuthHold,
			AmountCents: amount,
			Status:      models.TransactionStatusActive,
		}

		existingCapture := &models.Transaction{
			ID:          captureID,
			AccountID:   accountID,
			Type:        models.TransactionTypeCapture,
			AmountCents: amount,
			ReferenceID: &authID,
			Status:      models.TransactionStatusCompleted,
		}

		mockTxRepo.On("FindByID", ctx, authID).Return(authTx, nil)
		mockTxRepo.On("FindByReferenceID", ctx, authID, models.TransactionTypeCapture).Return(existingCapture, nil)

		result, err := service.performVoid(ctx, mockTxRepo, mockAccountRepo, authID)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeAlreadyCaptured, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("check existing capture fails", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewVoidService(nil)
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

		mockTxRepo.On("FindByID", ctx, authID).Return(authTx, nil)
		mockTxRepo.On("FindByReferenceID", ctx, authID, models.TransactionTypeCapture).Return(nil, assert.AnError)

		result, err := service.performVoid(ctx, mockTxRepo, mockAccountRepo, authID)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeInternalError, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("already voided - duplicate error", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewVoidService(nil)
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

		mockTxRepo.On("FindByID", ctx, authID).Return(authTx, nil)
		mockTxRepo.On("FindByReferenceID", ctx, authID, models.TransactionTypeCapture).Return(nil, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).
			Return(models.ErrDuplicateTransaction)

		result, err := service.performVoid(ctx, mockTxRepo, mockAccountRepo, authID)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeAlreadyVoided, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("status update fails", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewVoidService(nil)
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

		mockTxRepo.On("FindByID", ctx, authID).Return(authTx, nil)
		mockTxRepo.On("FindByReferenceID", ctx, authID, models.TransactionTypeCapture).Return(nil, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).Return(nil)
		mockTxRepo.On("UpdateStatus", ctx, authID, models.TransactionStatusCompleted).
			Return(assert.AnError)

		result, err := service.performVoid(ctx, mockTxRepo, mockAccountRepo, authID)

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
		service := NewVoidService(nil)
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

		mockTxRepo.On("FindByID", ctx, authID).Return(authTx, nil)
		mockTxRepo.On("FindByReferenceID", ctx, authID, models.TransactionTypeCapture).Return(nil, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).Return(nil)
		mockTxRepo.On("UpdateStatus", ctx, authID, models.TransactionStatusCompleted).Return(nil)
		mockAccountRepo.On("AdjustBalances", ctx, accountID, int64(0), int64(10000)).
			Return(assert.AnError)

		result, err := service.performVoid(ctx, mockTxRepo, mockAccountRepo, authID)

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
