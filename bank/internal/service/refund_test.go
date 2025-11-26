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

func TestRefundService_PerformRefund(t *testing.T) {
	t.Run("successful refund", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewRefundService(nil)
		ctx := context.Background()

		captureID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		captureTx := &models.Transaction{
			ID:          captureID,
			AccountID:   accountID,
			Type:        models.TransactionTypeCapture,
			AmountCents: amount,
			Currency:    "USD",
			Status:      models.TransactionStatusCompleted,
		}

		mockTxRepo.On("FindByID", ctx, captureID).Return(captureTx, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).Return(nil)
		mockAccountRepo.On("AdjustBalances", ctx, accountID, int64(10000), int64(10000)).Return(nil)

		result, err := service.performRefund(ctx, mockTxRepo, mockAccountRepo, captureID, amount)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, models.TransactionTypeRefund, result.Type)
		assert.Equal(t, amount, result.AmountCents)
		assert.Equal(t, captureID, *result.ReferenceID)
		assert.Equal(t, models.TransactionStatusCompleted, result.Status)

		mockTxRepo.AssertExpectations(t)
		mockAccountRepo.AssertExpectations(t)
	})

	t.Run("capture not found", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewRefundService(nil)
		ctx := context.Background()

		captureID := uuid.New()
		var amount int64 = 10000

		mockTxRepo.On("FindByID", ctx, captureID).Return(nil, sql.ErrNoRows)

		result, err := service.performRefund(ctx, mockTxRepo, mockAccountRepo, captureID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeCaptureNotFound, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("wrong transaction type", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewRefundService(nil)
		ctx := context.Background()

		captureID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		// Return an AUTH_HOLD instead of CAPTURE
		authTx := &models.Transaction{
			ID:          captureID,
			AccountID:   accountID,
			Type:        models.TransactionTypeAuthHold,
			AmountCents: amount,
			Status:      models.TransactionStatusActive,
		}

		mockTxRepo.On("FindByID", ctx, captureID).Return(authTx, nil)

		result, err := service.performRefund(ctx, mockTxRepo, mockAccountRepo, captureID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeCaptureNotFound, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("capture not completed", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewRefundService(nil)
		ctx := context.Background()

		captureID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		captureTx := &models.Transaction{
			ID:          captureID,
			AccountID:   accountID,
			Type:        models.TransactionTypeCapture,
			AmountCents: amount,
			Status:      models.TransactionStatusActive, // Not completed
		}

		mockTxRepo.On("FindByID", ctx, captureID).Return(captureTx, nil)

		result, err := service.performRefund(ctx, mockTxRepo, mockAccountRepo, captureID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeCaptureNotFound, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("amount mismatch", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewRefundService(nil)
		ctx := context.Background()

		captureID := uuid.New()
		accountID := uuid.New()
		var captureAmount int64 = 10000
		var refundAmount int64 = 5000 // Different amount

		captureTx := &models.Transaction{
			ID:          captureID,
			AccountID:   accountID,
			Type:        models.TransactionTypeCapture,
			AmountCents: captureAmount,
			Status:      models.TransactionStatusCompleted,
		}

		mockTxRepo.On("FindByID", ctx, captureID).Return(captureTx, nil)

		result, err := service.performRefund(ctx, mockTxRepo, mockAccountRepo, captureID, refundAmount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeAmountMismatch, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("already refunded - duplicate error", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewRefundService(nil)
		ctx := context.Background()

		captureID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		captureTx := &models.Transaction{
			ID:          captureID,
			AccountID:   accountID,
			Type:        models.TransactionTypeCapture,
			AmountCents: amount,
			Status:      models.TransactionStatusCompleted,
		}

		mockTxRepo.On("FindByID", ctx, captureID).Return(captureTx, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).
			Return(models.ErrDuplicateTransaction)

		result, err := service.performRefund(ctx, mockTxRepo, mockAccountRepo, captureID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)

		var svcErr *ServiceError
		if assert.ErrorAs(t, err, &svcErr) {
			assert.Equal(t, ErrCodeAlreadyRefunded, svcErr.Code)
		}

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("transaction creation fails", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewRefundService(nil)
		ctx := context.Background()

		captureID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		captureTx := &models.Transaction{
			ID:          captureID,
			AccountID:   accountID,
			Type:        models.TransactionTypeCapture,
			AmountCents: amount,
			Status:      models.TransactionStatusCompleted,
		}

		mockTxRepo.On("FindByID", ctx, captureID).Return(captureTx, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).
			Return(assert.AnError)

		result, err := service.performRefund(ctx, mockTxRepo, mockAccountRepo, captureID, amount)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.NotErrorIs(t, err, models.ErrDuplicateTransaction)

		mockTxRepo.AssertExpectations(t)
	})

	t.Run("balance adjustment fails", func(t *testing.T) {
		mockTxRepo := mocks.NewMockTransactionRepository(t)
		mockAccountRepo := mocks.NewMockAccountRepository(t)
		service := NewRefundService(nil)
		ctx := context.Background()

		captureID := uuid.New()
		accountID := uuid.New()
		var amount int64 = 10000

		captureTx := &models.Transaction{
			ID:          captureID,
			AccountID:   accountID,
			Type:        models.TransactionTypeCapture,
			AmountCents: amount,
			Status:      models.TransactionStatusCompleted,
		}

		mockTxRepo.On("FindByID", ctx, captureID).Return(captureTx, nil)
		mockTxRepo.On("Create", ctx, mock.AnythingOfType("*models.Transaction")).Return(nil)
		mockAccountRepo.On("AdjustBalances", ctx, accountID, int64(10000), int64(10000)).
			Return(assert.AnError)

		result, err := service.performRefund(ctx, mockTxRepo, mockAccountRepo, captureID, amount)

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
