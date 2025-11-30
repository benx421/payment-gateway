package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/db"
	"github.com/benx421/payment-gateway/bank/internal/models"
	"github.com/benx421/payment-gateway/bank/internal/repository"
	"github.com/google/uuid"
)

// RefundService handles refund operations
type RefundService struct {
	db *db.DB
}

// NewRefundService creates a new RefundService
func NewRefundService(database *db.DB) *RefundService {
	return &RefundService{
		db: database,
	}
}

// Refund refunds a captured payment
func (s *RefundService) Refund(ctx context.Context, captureID uuid.UUID, amount int64) (*models.Transaction, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to start transaction: %v", err),
		}
	}
	defer func() {
		_ = tx.Rollback() //nolint:errcheck // rollback error is not critical in defer
	}()

	txTransactionRepo := repository.NewTransactionRepository(tx)
	txAccountRepo := repository.NewAccountRepository(tx)

	refundTxn, err := s.performRefund(ctx, txTransactionRepo, txAccountRepo, captureID, amount)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to commit transaction: %v", err),
		}
	}

	return refundTxn, nil
}

// performRefund contains the core refund business logic
func (s *RefundService) performRefund(
	ctx context.Context,
	transactionRepo repository.TransactionRepository,
	accountRepo repository.AccountRepository,
	captureID uuid.UUID,
	amount int64,
) (*models.Transaction, error) {
	captureTxn, err := transactionRepo.FindByIDForUpdate(ctx, captureID)
	if err != nil || captureTxn.Type != models.TransactionTypeCapture {
		return nil, &ServiceError{
			Code:    ErrCodeCaptureNotFound,
			Message: "capture not found",
		}
	}

	if captureTxn.Status != models.TransactionStatusCompleted {
		return nil, &ServiceError{
			Code:    ErrCodeCaptureNotFound,
			Message: "capture is not in completed status",
		}
	}

	if amount != captureTxn.AmountCents {
		return nil, &ServiceError{
			Code: ErrCodeAmountMismatch,
			Message: fmt.Sprintf("refund amount (%d) must equal capture amount (%d)",
				amount, captureTxn.AmountCents),
		}
	}

	refundID := uuid.New()
	refundedAt := time.Now()

	refundTxn := &models.Transaction{
		ID:          refundID,
		AccountID:   captureTxn.AccountID,
		Type:        models.TransactionTypeRefund,
		AmountCents: amount,
		Currency:    captureTxn.Currency,
		ReferenceID: &captureID,
		Status:      models.TransactionStatusCompleted,
		CreatedAt:   refundedAt,
	}

	if err := transactionRepo.Create(ctx, refundTxn); err != nil {
		if errors.Is(err, models.ErrDuplicateTransaction) {
			return nil, &ServiceError{
				Code:    ErrCodeAlreadyRefunded,
				Message: "capture has already been refunded",
			}
		}
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	if err := accountRepo.AdjustBalances(ctx, captureTxn.AccountID, amount, amount); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to adjust balance: %v", err),
		}
	}

	return refundTxn, nil
}

// GetRefund retrieves a refund by ID
func (s *RefundService) GetRefund(ctx context.Context, refundID uuid.UUID) (*models.Transaction, error) {
	repo := repository.NewTransactionRepository(s.db)
	txn, err := repo.FindByID(ctx, refundID)
	if err != nil || txn.Type != models.TransactionTypeRefund {
		return nil, &ServiceError{
			Code:    "refund_not_found",
			Message: "refund not found",
		}
	}

	return txn, nil
}
