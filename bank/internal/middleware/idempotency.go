// Package middleware provides HTTP middleware components for the bank API.
package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/models"
)

const idempotencyKeyHeader = "Idempotency-Key"

// idempotentPaths defines which paths require idempotency handling
//
// Only mutating operations (POST) need idempotency
var idempotentPaths = []string{
	"/api/v1/authorizations",
	"/api/v1/captures",
	"/api/v1/voids",
	"/api/v1/refunds",
}

// IdempotencyRepository defines the interface for idempotency storage
type IdempotencyRepository interface {
	Get(ctx context.Context, key, requestPath string) (*models.IdempotencyKey, error)
	Store(ctx context.Context, idemKey *models.IdempotencyKey) error
}

type responseCapture struct {
	http.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

func newResponseCapture(w http.ResponseWriter) *responseCapture {
	return &responseCapture{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // Default if WriteHeader not called
	}
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.statusCode = code
	rc.ResponseWriter.WriteHeader(code)
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	rc.body.Write(b) // Capture for caching
	return rc.ResponseWriter.Write(b)
}

// Idempotency creates middleware that handles idempotent request caching.
func Idempotency(repo IdempotencyRepository, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !requiresIdempotency(r) {
				next.ServeHTTP(w, r)
				return
			}

			idempotencyKey := r.Header.Get(idempotencyKeyHeader)
			if idempotencyKey == "" {
				// Let the generated handler return the proper error
				next.ServeHTTP(w, r)
				return
			}

			requestPath := normalizeRequestPath(r.URL.Path)
			ctx := r.Context()

			cached, err := repo.Get(ctx, idempotencyKey, requestPath)
			if err != nil {
				logger.Error("failed to check idempotency cache", "error", err)
				next.ServeHTTP(w, r)
				return
			}

			if cached != nil {
				logger.Debug("returning cached idempotent response",
					"key", idempotencyKey,
					"path", requestPath,
					"status", cached.ResponseStatus,
				)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Idempotent-Replayed", "true")
				w.WriteHeader(cached.ResponseStatus)
				//nolint:errcheck // Best effort response writing
				w.Write([]byte(cached.ResponseBody))
				return
			}

			capture := newResponseCapture(w)
			next.ServeHTTP(capture, r)

			if shouldCacheResponse(capture.statusCode) {
				idemKey := &models.IdempotencyKey{
					Key:            idempotencyKey,
					RequestPath:    requestPath,
					ResponseStatus: capture.statusCode,
					ResponseBody:   capture.body.String(),
					CreatedAt:      time.Now(),
				}

				if err := repo.Store(ctx, idemKey); err != nil {
					logger.Error("failed to store idempotency key",
						"error", err,
						"key", idempotencyKey,
					)
				}
			}
		})
	}
}

func requiresIdempotency(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	for _, path := range idempotentPaths {
		if r.URL.Path == path {
			return true
		}
	}
	return false
}

func normalizeRequestPath(urlPath string) string {
	return strings.TrimSuffix(urlPath, "/")
}

func shouldCacheResponse(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}
