package handlers

import (
	"errors"
	"fmt"
	"strings"

	"github.com/benx421/payment-gateway/bank/internal/api"
	"github.com/benx421/payment-gateway/bank/internal/service"
	"github.com/google/uuid"
)

// ID prefixes for API responses
const (
	PrefixAuthorization = "auth_"
	PrefixCapture       = "cap_"
	PrefixVoid          = "void_"
	PrefixRefund        = "ref_"
)

func formatAuthorizationID(id uuid.UUID) string {
	return PrefixAuthorization + id.String()
}

func formatCaptureID(id uuid.UUID) string {
	return PrefixCapture + id.String()
}

func formatVoidID(id uuid.UUID) string {
	return PrefixVoid + id.String()
}

func formatRefundID(id uuid.UUID) string {
	return PrefixRefund + id.String()
}

func parseAuthorizationID(id string) (uuid.UUID, error) {
	return parseIDWithPrefix(id, PrefixAuthorization, "authorization")
}

func parseCaptureID(id string) (uuid.UUID, error) {
	return parseIDWithPrefix(id, PrefixCapture, "capture")
}

func parseRefundID(id string) (uuid.UUID, error) {
	return parseIDWithPrefix(id, PrefixRefund, "refund")
}

func parseIDWithPrefix(id, prefix, typeName string) (uuid.UUID, error) {
	if !strings.HasPrefix(id, prefix) {
		return uuid.Nil, fmt.Errorf("invalid %s ID format: missing %s prefix", typeName, prefix)
	}

	uuidStr := strings.TrimPrefix(id, prefix)
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s ID format: %w", typeName, err)
	}

	return parsed, nil
}

func mapServiceErrorToCode(code string) api.ErrorCode {
	switch code {
	case service.ErrCodeInvalidCard:
		return api.ErrorCodeInvalidCard
	case service.ErrCodeInvalidCVV:
		return api.ErrorCodeInvalidCvv
	case service.ErrCodeInvalidAmount:
		return api.ErrorCodeInvalidAmount
	case service.ErrCodeCardExpired:
		return api.ErrorCodeCardExpired
	case service.ErrCodeInsufficientFunds:
		return api.ErrorCodeInsufficientFunds
	case service.ErrCodeAuthNotFound:
		return api.ErrorCodeAuthorizationNotFound
	case service.ErrCodeAuthExpired:
		return api.ErrorCodeAuthorizationExpired
	case service.ErrCodeAuthAlreadyUsed:
		return api.ErrorCodeAuthorizationAlreadyUsed
	case service.ErrCodeAlreadyCaptured:
		return api.ErrorCodeAlreadyCaptured
	case service.ErrCodeAlreadyVoided:
		return api.ErrorCodeAlreadyVoided
	case service.ErrCodeAlreadyRefunded:
		return api.ErrorCodeAlreadyRefunded
	case service.ErrCodeAmountMismatch:
		return api.ErrorCodeAmountMismatch
	case service.ErrCodeCaptureNotFound:
		return api.ErrorCodeCaptureNotFound
	default:
		return api.ErrorCodeInternalError
	}
}

func isPaymentRequiredError(code string) bool {
	return code == service.ErrCodeInsufficientFunds
}

func extractServiceError(err error) *service.ServiceError {
	var svcErr *service.ServiceError
	if errors.As(err, &svcErr) {
		return svcErr
	}
	return nil
}
