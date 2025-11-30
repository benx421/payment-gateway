package handlers

import (
	"context"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/api"
)

// CreateAuthorization handles POST /api/v1/authorizations
func (h *Handler) CreateAuthorization(
	ctx context.Context,
	request api.CreateAuthorizationRequestObject,
) (api.CreateAuthorizationResponseObject, error) {
	txn, err := h.authService.Authorize(
		ctx,
		request.Body.CardNumber,
		request.Body.Cvv,
		request.Body.Amount,
	)

	if err != nil {
		return h.handleAuthorizationError(err)
	}

	return api.CreateAuthorization200JSONResponse{
		AuthorizationId: formatAuthorizationID(txn.ID),
		Status:          api.Approved,
		Amount:          txn.AmountCents,
		Currency:        txn.Currency,
		ExpiresAt:       *txn.ExpiresAt,
		CreatedAt:       txn.CreatedAt,
	}, nil
}

// GetAuthorization handles GET /api/v1/authorizations/{authorizationId}
func (h *Handler) GetAuthorization(
	ctx context.Context,
	request api.GetAuthorizationRequestObject,
) (api.GetAuthorizationResponseObject, error) {
	authID, err := parseAuthorizationID(request.AuthorizationId)
	if err != nil {
		//nolint:nilerr // Returning 404 response object, not propagating error
		return api.GetAuthorization404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse{
				Error:   api.ErrorCodeNotFound,
				Message: "authorization not found",
			},
		}, nil
	}

	txn, err := h.authService.GetAuthorization(ctx, authID)
	if err != nil {
		//nolint:nilerr // Returning 404 response object, not propagating error
		return api.GetAuthorization404JSONResponse{
			NotFoundJSONResponse: api.NotFoundJSONResponse{
				Error:   api.ErrorCodeNotFound,
				Message: "authorization not found",
			},
		}, nil
	}

	expiresAt := time.Time{}
	if txn.ExpiresAt != nil {
		expiresAt = *txn.ExpiresAt
	}

	return api.GetAuthorization200JSONResponse{
		AuthorizationId: formatAuthorizationID(txn.ID),
		Status:          api.Approved,
		Amount:          txn.AmountCents,
		Currency:        txn.Currency,
		ExpiresAt:       expiresAt,
		CreatedAt:       txn.CreatedAt,
	}, nil
}

// handleAuthorizationError maps service errors to appropriate HTTP responses
func (h *Handler) handleAuthorizationError(
	err error,
) (api.CreateAuthorizationResponseObject, error) {
	svcErr := extractServiceError(err)
	if svcErr == nil {
		h.logger.Error("unexpected error during authorization", "error", err)
		return api.CreateAuthorization500JSONResponse{
			InternalErrorJSONResponse: api.InternalErrorJSONResponse{
				Error:   api.ErrorCodeInternalError,
				Message: "internal error",
			},
		}, nil
	}

	errorCode := mapServiceErrorToCode(svcErr.Code)

	if isPaymentRequiredError(svcErr.Code) {
		return api.CreateAuthorization402JSONResponse{
			PaymentRequiredJSONResponse: api.PaymentRequiredJSONResponse{
				Error:   errorCode,
				Message: svcErr.Message,
			},
		}, nil
	}

	return api.CreateAuthorization400JSONResponse{
		BadRequestJSONResponse: api.BadRequestJSONResponse{
			Error:   errorCode,
			Message: svcErr.Message,
		},
	}, nil
}
