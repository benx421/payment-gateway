package handlers

import (
	"context"

	"github.com/benx421/payment-gateway/bank/internal/api"
)

// CreateCapture handles POST /api/v1/captures
func (h *Handler) CreateCapture(
	ctx context.Context,
	request api.CreateCaptureRequestObject,
) (api.CreateCaptureResponseObject, error) {
	authID, err := parseAuthorizationID(request.Body.AuthorizationId)
	if err != nil {
		//nolint:nilerr // Returning 400 response object, not propagating error
		return api.CreateCapture400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   api.ErrorCodeAuthorizationNotFound,
				Message: "invalid authorization ID format",
			},
		}, nil
	}

	txn, err := h.captureService.Capture(ctx, authID, request.Body.Amount)
	if err != nil {
		return h.handleCaptureError(err)
	}

	return api.CreateCapture200JSONResponse{
		CaptureId:       formatCaptureID(txn.ID),
		AuthorizationId: formatAuthorizationID(*txn.ReferenceID),
		Status:          api.Captured,
		Amount:          txn.AmountCents,
		Currency:        txn.Currency,
		CapturedAt:      txn.CreatedAt,
	}, nil
}

// GetCapture handles GET /api/v1/captures/{captureId}
func (h *Handler) GetCapture(
	ctx context.Context,
	request api.GetCaptureRequestObject,
) (api.GetCaptureResponseObject, error) {
	captureID, err := parseCaptureID(request.CaptureId)
	if err != nil {
		//nolint:nilerr // Returning 404 response object, not propagating error
		return api.GetCapture404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse{
				Error:   api.ErrorCodeNotFound,
				Message: "capture not found",
			},
		}, nil
	}

	txn, err := h.captureService.GetCapture(ctx, captureID)
	if err != nil {
		//nolint:nilerr // Returning 404 response object, not propagating error
		return api.GetCapture404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse{
				Error:   api.ErrorCodeNotFound,
				Message: "capture not found",
			},
		}, nil
	}

	return api.GetCapture200JSONResponse{
		CaptureId:       formatCaptureID(txn.ID),
		AuthorizationId: formatAuthorizationID(*txn.ReferenceID),
		Status:          api.Captured,
		Amount:          txn.AmountCents,
		Currency:        txn.Currency,
		CapturedAt:      txn.CreatedAt,
	}, nil
}

// handleCaptureError maps service errors to appropriate HTTP responses
func (h *Handler) handleCaptureError(err error) (api.CreateCaptureResponseObject, error) {
	svcErr := extractServiceError(err)
	if svcErr == nil {
		h.logger.Error("unexpected error during capture", "error", err)
		return api.CreateCapture500JSONResponse{
			InternalErrorJSONResponse: api.InternalErrorJSONResponse{
				Error:   api.ErrorCodeInternalError,
				Message: "internal error",
			},
		}, nil
	}

	errorCode := mapServiceErrorToCode(svcErr.Code)

	return api.CreateCapture400JSONResponse{
		BadRequestJSONResponse: api.BadRequestJSONResponse{
			Error:   errorCode,
			Message: svcErr.Message,
		},
	}, nil
}
