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

// VoidService handles authorization void operations
type VoidService struct {
	db *db.DB
}

// NewVoidService creates a new VoidService
func NewVoidService(database *db.DB) *VoidService {
	return &VoidService{
		db: database,
	}
}

// Void cancels an authorization before it's captured
func (s *VoidService) Void(ctx context.Context, authorizationID uuid.UUID) (*models.Transaction, error) {
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

	voidTxn, err := s.performVoid(ctx, txTransactionRepo, txAccountRepo, authorizationID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to commit transaction: %v", err),
		}
	}

	return voidTxn, nil
}

// performVoid contains the core void business logic
func (s *VoidService) performVoid(
	ctx context.Context,
	transactionRepo repository.TransactionRepository,
	accountRepo repository.AccountRepository,
	authorizationID uuid.UUID,
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

	existingCapture, err := transactionRepo.FindByReferenceID(ctx, authorizationID, models.TransactionTypeCapture)
	if err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to check existing capture: %v", err),
		}
	}
	if existingCapture != nil {
		return nil, &ServiceError{
			Code:    ErrCodeAlreadyCaptured,
			Message: "cannot void an authorization that has been captured",
		}
	}

	voidID := uuid.New()
	voidedAt := time.Now()

	voidTxn := &models.Transaction{
		ID:          voidID,
		AccountID:   authTxn.AccountID,
		Type:        models.TransactionTypeVoid,
		AmountCents: authTxn.AmountCents,
		Currency:    authTxn.Currency,
		ReferenceID: &authorizationID,
		Status:      models.TransactionStatusCompleted,
		CreatedAt:   voidedAt,
	}

	if err := transactionRepo.Create(ctx, voidTxn); err != nil {
		if errors.Is(err, models.ErrDuplicateTransaction) {
			return nil, &ServiceError{
				Code:    ErrCodeAlreadyVoided,
				Message: "authorization has already been voided",
			}
		}
		return nil, fmt.Errorf("failed to create void: %w", err)
	}

	if err := transactionRepo.UpdateStatus(ctx, authorizationID, models.TransactionStatusCompleted); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to update authorization: %v", err),
		}
	}

	if err := accountRepo.AdjustBalances(ctx, authTxn.AccountID, 0, authTxn.AmountCents); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to adjust balance: %v", err),
		}
	}

	return voidTxn, nil
}
