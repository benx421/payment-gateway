package handlers

import (
	"context"

	"github.com/benx421/payment-gateway/bank/internal/api"
)

// CreateVoid handles POST /api/v1/voids
func (h *Handler) CreateVoid(
	ctx context.Context,
	request api.CreateVoidRequestObject,
) (api.CreateVoidResponseObject, error) {
	authID, err := parseAuthorizationID(request.Body.AuthorizationId)
	if err != nil {
		//nolint:nilerr // Returning 400 response object, not propagating error
		return api.CreateVoid400JSONResponse{
			BadRequestJSONResponse: api.BadRequestJSONResponse{
				Error:   api.ErrorCodeAuthorizationNotFound,
				Message: "invalid authorization ID format",
			},
		}, nil
	}

	txn, err := h.voidService.Void(ctx, authID)
	if err != nil {
		return h.handleVoidError(err)
	}

	return api.CreateVoid200JSONResponse{
		VoidId:          formatVoidID(txn.ID),
		AuthorizationId: formatAuthorizationID(*txn.ReferenceID),
		Status:          api.Voided,
		VoidedAt:        txn.CreatedAt,
	}, nil
}

func (h *Handler) handleVoidError(err error) (api.CreateVoidResponseObject, error) {
	svcErr := extractServiceError(err)
	if svcErr == nil {
		h.logger.Error("unexpected error during void", "error", err)
		return api.CreateVoid500JSONResponse{
			InternalErrorJSONResponse: api.InternalErrorJSONResponse{
				Error:   api.ErrorCodeInternalError,
				Message: "internal error",
			},
		}, nil
	}

	errorCode := mapServiceErrorToCode(svcErr.Code)

	return api.CreateVoid400JSONResponse{
		BadRequestJSONResponse: api.BadRequestJSONResponse{
			Error:   errorCode,
			Message: svcErr.Message,
		},
	}, nil
}
