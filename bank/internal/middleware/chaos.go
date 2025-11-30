// Package middleware provides HTTP middleware components for the bank API.
package middleware

import (
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/config"
)

type chaosErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

var excludedPaths = []string{
	"/health",
	"/docs",
}

// FailureInjection creates middleware that injects latency and random failures
// for testing resilience of client applications.
func FailureInjection(cfg *config.AppConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isExcludedPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			injectLatency(cfg.MinLatencyMS, cfg.MaxLatencyMS)

			if shouldInjectFailure(cfg.FailureRate) {
				logger.Debug("injecting random failure",
					"path", r.URL.Path,
					"method", r.Method,
				)
				writeFailureResponse(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isExcludedPath(path string) bool {
	for _, excluded := range excludedPaths {
		if strings.HasPrefix(path, excluded) {
			return true
		}
	}
	return false
}

func injectLatency(minMS, maxMS int) {
	if minMS <= 0 && maxMS <= 0 {
		return
	}

	rangeMS := maxMS - minMS
	if rangeMS <= 0 {
		time.Sleep(time.Duration(minMS) * time.Millisecond)
		return
	}

	randomOffset, err := rand.Int(rand.Reader, big.NewInt(int64(rangeMS)))
	if err != nil {
		time.Sleep(time.Duration(minMS) * time.Millisecond)
		return
	}

	sleepMS := minMS + int(randomOffset.Int64())
	time.Sleep(time.Duration(sleepMS) * time.Millisecond)
}

func shouldInjectFailure(failureRate float64) bool {
	if failureRate <= 0 {
		return false
	}
	if failureRate >= 1 {
		return true
	}

	const precision = 1000000
	randomNum, err := rand.Int(rand.Reader, big.NewInt(precision))
	if err != nil {
		return false
	}

	threshold := int64(failureRate * precision)
	return randomNum.Int64() < threshold
}

func writeFailureResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)

	resp := chaosErrorResponse{
		Error:   "internal_error",
		Message: "Random failure injection",
	}

	//nolint:errcheck // Best effort response writing in chaos injection
	json.NewEncoder(w).Encode(resp)
}
