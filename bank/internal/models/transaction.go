package models

import (
	"time"

	"github.com/google/uuid"
)

// TransactionType represents the type of transaction
type TransactionType string

// Transaction type constants
const (
	TransactionTypeAuthHold TransactionType = "AUTH_HOLD" // Authorization hold (funds reserved)
	TransactionTypeCapture  TransactionType = "CAPTURE"   // Capture authorized funds
	TransactionTypeVoid     TransactionType = "VOID"      // Void/cancel authorization
	TransactionTypeRefund   TransactionType = "REFUND"    // Refund captured funds
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

// Transaction status constants
const (
	TransactionStatusActive    TransactionStatus = "ACTIVE"    // Transaction is active (auth holds)
	TransactionStatusCompleted TransactionStatus = "COMPLETED" // Transaction completed successfully
	TransactionStatusExpired   TransactionStatus = "EXPIRED"   // Transaction expired (auth timeout)
)

// Transaction represents a ledger entry for account activity
type Transaction struct {
	CreatedAt   time.Time         `db:"created_at"`
	Metadata    map[string]any    `db:"metadata"`
	ReferenceID *uuid.UUID        `db:"reference_id"`
	ExpiresAt   *time.Time        `db:"expires_at"`
	Currency    string            `db:"currency"`
	Type        TransactionType   `db:"type"`
	Status      TransactionStatus `db:"status"`
	AmountCents int64             `db:"amount_cents"`
	ID          uuid.UUID         `db:"id"`
	AccountID   uuid.UUID         `db:"account_id"`
}

// IdempotencyKey tracks processed requests to prevent duplicate transactions
type IdempotencyKey struct {
	CreatedAt      time.Time `db:"created_at"`
	Key            string    `db:"key"`
	RequestPath    string    `db:"request_path"`
	ResponseBody   string    `db:"response_body"`
	ResponseStatus int       `db:"response_status"`
}
