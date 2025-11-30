package service

import (
	"context"

	"github.com/benx421/payment-gateway/bank/internal/models"
	"github.com/google/uuid"
)

// HealthChecker validates system health.
type HealthChecker interface {
	PingContext(ctx context.Context) error
}

// Authorizer handles payment authorization operations
type Authorizer interface {
	Authorize(ctx context.Context, cardNumber, cvv string, amount int64) (*models.Transaction, error)
	GetAuthorization(ctx context.Context, authID uuid.UUID) (*models.Transaction, error)
}

// Capturer handles payment capture operations
type Capturer interface {
	Capture(ctx context.Context, authorizationID uuid.UUID, amount int64) (*models.Transaction, error)
	GetCapture(ctx context.Context, captureID uuid.UUID) (*models.Transaction, error)
}

// Voider handles authorization void operations
type Voider interface {
	Void(ctx context.Context, authorizationID uuid.UUID) (*models.Transaction, error)
}

// Refunder handles refund operations
type Refunder interface {
	Refund(ctx context.Context, captureID uuid.UUID, amount int64) (*models.Transaction, error)
	GetRefund(ctx context.Context, refundID uuid.UUID) (*models.Transaction, error)
}

// Ensure concrete types implement interfaces
var (
	_ Authorizer = (*AuthorizationService)(nil)
	_ Capturer   = (*CaptureService)(nil)
	_ Voider     = (*VoidService)(nil)
	_ Refunder   = (*RefundService)(nil)
)
