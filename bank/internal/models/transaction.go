package models

import (
	"time"

	"github.com/google/uuid"
)

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypeAuthHold TransactionType = "AUTH_HOLD"
	TransactionTypeCapture  TransactionType = "CAPTURE"
	TransactionTypeVoid     TransactionType = "VOID"
	TransactionTypeRefund   TransactionType = "REFUND"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusActive    TransactionStatus = "ACTIVE"
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
	TransactionStatusExpired   TransactionStatus = "EXPIRED"
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
