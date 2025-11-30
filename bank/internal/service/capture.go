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

// CaptureService handles payment capture operations
type CaptureService struct {
	db *db.DB
}

// NewCaptureService creates a new CaptureService
func NewCaptureService(database *db.DB) *CaptureService {
	return &CaptureService{
		db: database,
	}
}

// Capture captures an authorized payment
func (s *CaptureService) Capture(ctx context.Context, authorizationID uuid.UUID, amount int64) (*models.Transaction, error) {
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

	captureTxn, err := s.performCapture(ctx, txTransactionRepo, txAccountRepo, authorizationID, amount)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to commit transaction: %v", err),
		}
	}

	return captureTxn, nil
}

// performCapture contains the core capture business logic
func (s *CaptureService) performCapture(
	ctx context.Context,
	transactionRepo repository.TransactionRepository,
	accountRepo repository.AccountRepository,
	authorizationID uuid.UUID,
	amount int64,
) (*models.Transaction, error) {
	authTxn, err := transactionRepo.FindByIDForUpdate(ctx, authorizationID)
	if err != nil || authTxn.Type != models.TransactionTypeAuthHold {
		return nil, &ServiceError{
			Code:    ErrCodeAuthNotFound,
			Message: "authorization not found",
		}
	}

	if authTxn.Status != models.TransactionStatusActive {
		return nil, &ServiceError{
			Code:    ErrCodeAuthAlreadyUsed,
			Message: "authorization has already been completed or cancelled",
		}
	}

	if authTxn.ExpiresAt != nil && time.Now().After(*authTxn.ExpiresAt) {
		return nil, &ServiceError{
			Code:    ErrCodeAuthExpired,
			Message: "authorization has expired",
		}
	}

	if amount != authTxn.AmountCents {
		return nil, &ServiceError{
			Code:    ErrCodeAmountMismatch,
			Message: "capture amount does not match authorized amount",
		}
	}

	captureID := uuid.New()
	capturedAt := time.Now()

	captureTxn := &models.Transaction{
		ID:          captureID,
		AccountID:   authTxn.AccountID,
		Type:        models.TransactionTypeCapture,
		AmountCents: amount,
		Currency:    authTxn.Currency,
		ReferenceID: &authorizationID,
		Status:      models.TransactionStatusCompleted,
		CreatedAt:   capturedAt,
	}

	if err := transactionRepo.Create(ctx, captureTxn); err != nil {
		if errors.Is(err, models.ErrDuplicateTransaction) {
			return nil, &ServiceError{
				Code:    ErrCodeAlreadyCaptured,
				Message: "authorization has already been captured",
			}
		}
		return nil, fmt.Errorf("failed to create capture: %w", err)
	}

	if err := transactionRepo.UpdateStatus(ctx, authorizationID, models.TransactionStatusCompleted); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to update authorization: %v", err),
		}
	}

	if err := accountRepo.AdjustBalances(ctx, authTxn.AccountID, -amount, 0); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to adjust balance: %v", err),
		}
	}

	return captureTxn, nil
}

// GetCapture retrieves a capture by ID
func (s *CaptureService) GetCapture(ctx context.Context, captureID uuid.UUID) (*models.Transaction, error) {
	repo := repository.NewTransactionRepository(s.db)
	txn, err := repo.FindByID(ctx, captureID)
	if err != nil || txn.Type != models.TransactionTypeCapture {
		return nil, &ServiceError{
			Code:    ErrCodeCaptureNotFound,
			Message: "capture not found",
		}
	}

	return txn, nil
}
