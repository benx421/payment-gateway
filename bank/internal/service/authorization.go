package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/db"
	"github.com/benx421/payment-gateway/bank/internal/models"
	"github.com/benx421/payment-gateway/bank/internal/repository"
	"github.com/google/uuid"
)

// AuthorizationService handles payment authorization operations
type AuthorizationService struct {
	db              *db.DB
	authExpiryHours int
}

// NewAuthorizationService creates a new AuthorizationService
func NewAuthorizationService(
	database *db.DB,
	authExpiryHours int,
) *AuthorizationService {
	return &AuthorizationService{
		db:              database,
		authExpiryHours: authExpiryHours,
	}
}

// Authorize creates an authorization hold on a customer's account
func (s *AuthorizationService) Authorize(ctx context.Context, cardNumber, cvv string, amount int64) (*models.Transaction, error) {
	if err := s.validateAuthorizationRequest(cardNumber, cvv, amount); err != nil {
		return nil, err
	}

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

	txAccountRepo := repository.NewAccountRepository(tx)
	txTransactionRepo := repository.NewTransactionRepository(tx)

	authTx, err := s.performAuthorization(ctx, txAccountRepo, txTransactionRepo, cardNumber, cvv, amount)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to commit transaction: %v", err),
		}
	}

	return authTx, nil
}

// performAuthorization contains the core authorization business logic
func (s *AuthorizationService) performAuthorization(
	ctx context.Context,
	accountRepo repository.AccountRepository,
	transactionRepo repository.TransactionRepository,
	cardNumber, cvv string,
	amount int64,
) (*models.Transaction, error) {
	account, err := accountRepo.FindByAccountNumberForUpdate(ctx, cardNumber)
	if err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInvalidCard,
			Message: "card not found or invalid",
		}
	}

	if account.CVV != cvv {
		return nil, &ServiceError{
			Code:    ErrCodeInvalidCVV,
			Message: "CVV does not match",
		}
	}

	if err := ValidateExpiry(account.ExpiryMonth, account.ExpiryYear); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeCardExpired,
			Message: err.Error(),
		}
	}

	if account.AvailableBalanceCents < amount {
		return nil, &ServiceError{
			Code:    ErrCodeInsufficientFunds,
			Message: "insufficient funds",
		}
	}

	authID := uuid.New()
	expiresAt := time.Now().Add(time.Duration(s.authExpiryHours) * time.Hour)
	createdAt := time.Now()

	authTx := &models.Transaction{
		ID:          authID,
		AccountID:   account.ID,
		Type:        models.TransactionTypeAuthHold,
		AmountCents: amount,
		Currency:    "USD",
		Status:      models.TransactionStatusActive,
		ExpiresAt:   &expiresAt,
		CreatedAt:   createdAt,
	}

	if err := transactionRepo.Create(ctx, authTx); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to create authorization: %v", err),
		}
	}

	if err := accountRepo.AdjustBalances(ctx, account.ID, 0, -amount); err != nil {
		return nil, &ServiceError{
			Code:    ErrCodeInternalError,
			Message: fmt.Sprintf("failed to adjust balance: %v", err),
		}
	}

	return authTx, nil
}

// GetAuthorization retrieves an authorization by ID
func (s *AuthorizationService) GetAuthorization(ctx context.Context, authID uuid.UUID) (*models.Transaction, error) {
	repo := repository.NewTransactionRepository(s.db)
	txn, err := repo.FindByID(ctx, authID)
	if err != nil || txn.Type != models.TransactionTypeAuthHold {
		return nil, &ServiceError{
			Code:    ErrCodeAuthNotFound,
			Message: "authorization not found",
		}
	}

	return txn, nil
}

func (s *AuthorizationService) validateAuthorizationRequest(cardNumber, cvv string, amount int64) error {
	if err := ValidateLuhn(cardNumber); err != nil {
		return &ServiceError{
			Code:    ErrCodeInvalidCard,
			Message: err.Error(),
		}
	}

	if err := ValidateCVV(cvv); err != nil {
		return &ServiceError{
			Code:    ErrCodeInvalidCVV,
			Message: err.Error(),
		}
	}

	if err := ValidateAmount(amount); err != nil {
		return &ServiceError{
			Code:    ErrCodeInvalidAmount,
			Message: err.Error(),
		}
	}

	return nil
}
