package handlers

import (
	"log/slog"
	"net/http"

	"github.com/benx421/payment-gateway/bank/internal/api"
	"github.com/benx421/payment-gateway/bank/internal/config"
	"github.com/benx421/payment-gateway/bank/internal/db"
	"github.com/benx421/payment-gateway/bank/internal/middleware"
	"github.com/benx421/payment-gateway/bank/internal/repository"
	"github.com/benx421/payment-gateway/bank/internal/service"
)

// NewRouter creates and configures the HTTP router with all routes and middleware.
func NewRouter(
	database *db.DB,
	cfg *config.Config,
	logger *slog.Logger,
) http.Handler {
	authService := service.NewAuthorizationService(database, cfg.App.AuthExpiryHours)
	captureService := service.NewCaptureService(database)
	voidService := service.NewVoidService(database)
	refundService := service.NewRefundService(database)

	handler := NewHandler(authService, captureService, voidService, refundService, database, logger)
	strictHandler := api.NewStrictHandler(handler, nil)

	mux := http.NewServeMux()
	api.RegisterDocsRoutes(mux)
	api.HandlerFromMux(strictHandler, mux)

	var finalHandler http.Handler = mux

	finalHandler = middleware.FailureInjection(&cfg.App, logger)(finalHandler)

	idempotencyRepo := repository.NewIdempotencyRepository(database)
	finalHandler = middleware.Idempotency(idempotencyRepo, logger)(finalHandler)

	return finalHandler
}
