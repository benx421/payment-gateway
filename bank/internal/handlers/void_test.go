package handlers

import (
	"context"
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

func TestCreateVoid_Success(t *testing.T) {
	mockVoid := mocks.NewMockVoider(t)
	handler := NewHandler(nil, nil, mockVoid, nil, nil, testLogger())

	authID := uuid.New()
	voidID := uuid.New()

	mockVoid.On("Void", mock.Anything, authID).
		Return(&models.Transaction{
			ID:          voidID,
			ReferenceID: &authID,
			CreatedAt:   time.Now(),
		}, nil)

	req := api.CreateVoidRequestObject{
		Body: &api.CreateVoidJSONRequestBody{AuthorizationId: "auth_" + authID.String()},
	}

	resp, err := handler.CreateVoid(context.Background(), req)

	require.NoError(t, err)
	successResp, ok := resp.(api.CreateVoid200JSONResponse)
	require.True(t, ok)
	assert.Equal(t, api.Voided, successResp.Status)
}

func TestCreateVoid_ServiceErrors(t *testing.T) {
	tests := []struct {
		name         string
		serviceErr   *service.ServiceError
		expectedCode api.ErrorCode
	}{
		{"auth not found", &service.ServiceError{Code: service.ErrCodeAuthNotFound}, api.ErrorCodeAuthorizationNotFound},
		{"already voided", &service.ServiceError{Code: service.ErrCodeAlreadyVoided}, api.ErrorCodeAlreadyVoided},
		{"already captured", &service.ServiceError{Code: service.ErrCodeAuthAlreadyUsed}, api.ErrorCodeAuthorizationAlreadyUsed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockVoid := mocks.NewMockVoider(t)
			handler := NewHandler(nil, nil, mockVoid, nil, nil, testLogger())

			mockVoid.On("Void", mock.Anything, mock.Anything).Return(nil, tt.serviceErr)

			req := api.CreateVoidRequestObject{
				Body: &api.CreateVoidJSONRequestBody{AuthorizationId: "auth_" + uuid.New().String()},
			}

			resp, err := handler.CreateVoid(context.Background(), req)
			require.NoError(t, err)

			badResp, ok := resp.(api.CreateVoid400JSONResponse)
			require.True(t, ok)
			assert.Equal(t, tt.expectedCode, badResp.Error)
		})
	}
}

func TestCreateVoid_InvalidIDFormat(t *testing.T) {
	handler := NewHandler(nil, nil, nil, nil, nil, testLogger())

	req := api.CreateVoidRequestObject{
		Body: &api.CreateVoidJSONRequestBody{AuthorizationId: "invalid"},
	}

	resp, err := handler.CreateVoid(context.Background(), req)

	require.NoError(t, err)
	badResp, ok := resp.(api.CreateVoid400JSONResponse)
	require.True(t, ok)
	assert.Equal(t, api.ErrorCodeAuthorizationNotFound, badResp.Error)
}
