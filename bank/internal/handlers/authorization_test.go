package handlers

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/benx421/payment-gateway/bank/internal/api"
	"github.com/benx421/payment-gateway/bank/internal/models"
	"github.com/benx421/payment-gateway/bank/internal/service"
	"github.com/benx421/payment-gateway/bank/internal/service/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCreateAuthorization_Success(t *testing.T) {
	mockAuth := mocks.NewMockAuthorizer(t)
	handler := NewHandler(mockAuth, nil, nil, nil, nil, testLogger())

	txnID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	mockAuth.On("Authorize", mock.Anything, "4111111111111111", "123", int64(10000)).
		Return(&models.Transaction{
			ID:          txnID,
			AmountCents: 10000,
			Currency:    "USD",
			ExpiresAt:   &expiresAt,
			CreatedAt:   time.Now(),
		}, nil)

	req := api.CreateAuthorizationRequestObject{
		Body: &api.CreateAuthorizationJSONRequestBody{
			CardNumber: "4111111111111111",
			Cvv:        "123",
			Amount:     10000,
		},
	}

	resp, err := handler.CreateAuthorization(context.Background(), req)

	require.NoError(t, err)
	successResp, ok := resp.(api.CreateAuthorization200JSONResponse)
	require.True(t, ok, "expected 200 response")
	assert.Equal(t, api.Approved, successResp.Status)
	assert.Equal(t, int64(10000), successResp.Amount)
}

func TestCreateAuthorization_ServiceErrors(t *testing.T) {
	tests := []struct {
		serviceErr     *service.ServiceError
		name           string
		expectedCode   api.ErrorCode
		expectedStatus int
	}{
		{
			name:           "invalid card returns 400",
			serviceErr:     &service.ServiceError{Code: service.ErrCodeInvalidCard, Message: "invalid"},
			expectedStatus: 400,
			expectedCode:   api.ErrorCodeInvalidCard,
		},
		{
			name:           "insufficient funds returns 402",
			serviceErr:     &service.ServiceError{Code: service.ErrCodeInsufficientFunds, Message: "insufficient"},
			expectedStatus: 402,
			expectedCode:   api.ErrorCodeInsufficientFunds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAuth := mocks.NewMockAuthorizer(t)
			handler := NewHandler(mockAuth, nil, nil, nil, nil, testLogger())

			mockAuth.On("Authorize", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil, tt.serviceErr)

			req := api.CreateAuthorizationRequestObject{
				Body: &api.CreateAuthorizationJSONRequestBody{
					CardNumber: "4111111111111111",
					Cvv:        "123",
					Amount:     10000,
				},
			}

			resp, err := handler.CreateAuthorization(context.Background(), req)
			require.NoError(t, err)

			switch tt.expectedStatus {
			case 400:
				badResp, ok := resp.(api.CreateAuthorization400JSONResponse)
				require.True(t, ok)
				assert.Equal(t, tt.expectedCode, badResp.Error)
			case 402:
				payResp, ok := resp.(api.CreateAuthorization402JSONResponse)
				require.True(t, ok)
				assert.Equal(t, tt.expectedCode, payResp.Error)
			}
		})
	}
}

func TestGetAuthorization_Success(t *testing.T) {
	mockAuth := mocks.NewMockAuthorizer(t)
	handler := NewHandler(mockAuth, nil, nil, nil, nil, testLogger())

	txnID := uuid.New()
	expiresAt := time.Now().Add(24 * time.Hour)

	mockAuth.On("GetAuthorization", mock.Anything, txnID).
		Return(&models.Transaction{
			ID:          txnID,
			AmountCents: 10000,
			Currency:    "USD",
			ExpiresAt:   &expiresAt,
			CreatedAt:   time.Now(),
		}, nil)

	req := api.GetAuthorizationRequestObject{
		AuthorizationId: "auth_" + txnID.String(),
	}

	resp, err := handler.GetAuthorization(context.Background(), req)

	require.NoError(t, err)
	_, ok := resp.(api.GetAuthorization200JSONResponse)
	require.True(t, ok)
}

func TestGetAuthorization_NotFound(t *testing.T) {
	mockAuth := mocks.NewMockAuthorizer(t)
	handler := NewHandler(mockAuth, nil, nil, nil, nil, testLogger())

	txnID := uuid.New()
	mockAuth.On("GetAuthorization", mock.Anything, txnID).
		Return(nil, &service.ServiceError{Code: service.ErrCodeAuthNotFound})

	req := api.GetAuthorizationRequestObject{
		AuthorizationId: "auth_" + txnID.String(),
	}

	resp, err := handler.GetAuthorization(context.Background(), req)

	require.NoError(t, err)
	notFoundResp, ok := resp.(api.GetAuthorization404JSONResponse)
	require.True(t, ok)
	assert.Equal(t, api.ErrorCodeNotFound, notFoundResp.Error)
}

func TestGetAuthorization_InvalidIDFormat(t *testing.T) {
	handler := NewHandler(nil, nil, nil, nil, nil, testLogger())

	req := api.GetAuthorizationRequestObject{
		AuthorizationId: "invalid-format",
	}

	resp, err := handler.GetAuthorization(context.Background(), req)

	require.NoError(t, err)
	_, ok := resp.(api.GetAuthorization404JSONResponse)
	require.True(t, ok, "invalid ID format should return 404")
}
