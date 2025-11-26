package service

import "fmt"

// ServiceError represents a business logic error with a code
type ServiceError struct {
	Err     error
	Message string
	Code    string
}

func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error for errors.Is/As support
func (e *ServiceError) Unwrap() error {
	return e.Err
}

// Common error codes
const (
	ErrCodeInvalidCard       = "invalid_card"
	ErrCodeInvalidCVV        = "invalid_cvv"
	ErrCodeInvalidAmount     = "invalid_amount"
	ErrCodeCardExpired       = "card_expired"
	ErrCodeInsufficientFunds = "insufficient_funds"
	ErrCodeAccountNotFound   = "account_not_found"
	ErrCodeAuthNotFound      = "authorization_not_found"
	ErrCodeAuthExpired       = "authorization_expired"
	ErrCodeAuthAlreadyUsed   = "authorization_already_used"
	ErrCodeAlreadyCaptured   = "already_captured"
	ErrCodeAlreadyVoided     = "already_voided"
	ErrCodeAlreadyRefunded   = "already_refunded"
	ErrCodeAmountMismatch    = "amount_mismatch"
	ErrCodeCaptureNotFound   = "capture_not_found"
	ErrCodeInternalError     = "internal_error"
)
