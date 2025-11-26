// Package main implements the mock bank API server.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/api"
	"github.com/benx421/payment-gateway/bank/internal/config"
	"github.com/benx421/payment-gateway/bank/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger := cfg.Logger.NewLogger()
	slog.SetDefault(logger)

	logger.Info("starting bank api",
		"port", cfg.Server.Port,
		"log_level", cfg.Logger.Level,
	)

	ctx := context.Background()
	database, err := db.Connect(ctx, &cfg.Database, logger)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err = database.Close(); err != nil {
			logger.Error("failed to close database connection", "error", err)
		}
	}()

	// Cleanup old idempotency keys (older than 24 hours)
	logger.Info("cleaning up old idempotency keys")
	cutoffTime := time.Now().Add(-24 * time.Hour)
	if _, err := database.ExecContext(ctx, "DELETE FROM idempotency_keys WHERE created_at < $1", cutoffTime); err != nil {
		logger.Warn("failed to cleanup old idempotency keys", "error", err)
	}

	mux := http.NewServeMux()

	api.RegisterDocsRoutes(mux)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		logger.Info("server listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server stopped")
}
