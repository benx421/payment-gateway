package models

import (
	"time"

	"github.com/google/uuid"
)

// Account represents a customer account with card details and balance
type Account struct {
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
	AccountNumber         string    `db:"account_number"`
	CVV                   string    `db:"cvv"`
	BalanceCents          int64     `db:"balance_cents"`
	AvailableBalanceCents int64     `db:"available_balance_cents"`
	ExpiryMonth           int       `db:"expiry_month"`
	ExpiryYear            int       `db:"expiry_year"`
	ID                    uuid.UUID `db:"id"`
}
