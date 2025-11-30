package handlers

import (
	"context"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/api"
)

// GetHealth handles GET /health
func (h *Handler) GetHealth(
	ctx context.Context,
	request api.GetHealthRequestObject,
) (api.GetHealthResponseObject, error) {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := h.healthChecker.PingContext(pingCtx); err != nil {
		h.logger.Error("health check failed: database unreachable", "error", err)
		return api.GetHealth503JSONResponse{
			Status: api.Unhealthy,
		}, nil
	}

	return api.GetHealth200JSONResponse{
		Status: api.Healthy,
	}, nil
}
