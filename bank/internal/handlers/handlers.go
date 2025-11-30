// Package handlers implements HTTP handlers for the bank API.
package handlers

import (
	"log/slog"

	"github.com/benx421/payment-gateway/bank/internal/service"
)

// Handler implements the api.StrictServerInterface for all endpoints
type Handler struct {
	authService    service.Authorizer
	captureService service.Capturer
	voidService    service.Voider
	refundService  service.Refunder
	healthChecker  service.HealthChecker
	logger         *slog.Logger
}

// NewHandler creates a new Handler with injected service dependencies.
func NewHandler(
	authService service.Authorizer,
	captureService service.Capturer,
	voidService service.Voider,
	refundService service.Refunder,
	healthChecker service.HealthChecker,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		authService:    authService,
		captureService: captureService,
		voidService:    voidService,
		refundService:  refundService,
		healthChecker:  healthChecker,
		logger:         logger,
	}
}
