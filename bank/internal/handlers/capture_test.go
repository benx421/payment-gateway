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

func TestCreateCapture_Success(t *testing.T) {
	mockCapture := mocks.NewMockCapturer(t)
	handler := NewHandler(nil, mockCapture, nil, nil, nil, testLogger())

	authID := uuid.New()
	captureID := uuid.New()

	mockCapture.On("Capture", mock.Anything, authID, int64(10000)).
		Return(&models.Transaction{
			ID:          captureID,
			ReferenceID: &authID,
			AmountCents: 10000,
			Currency:    "USD",
			CreatedAt:   time.Now(),
		}, nil)

	req := api.CreateCaptureRequestObject{
		Body: &api.CreateCaptureJSONRequestBody{
			AuthorizationId: "auth_" + authID.String(),
			Amount:          10000,
		},
	}

	resp, err := handler.CreateCapture(context.Background(), req)

	require.NoError(t, err)
	successResp, ok := resp.(api.CreateCapture200JSONResponse)
	require.True(t, ok)
	assert.Equal(t, api.Captured, successResp.Status)
}

func TestCreateCapture_ServiceErrors(t *testing.T) {
	tests := []struct {
		name         string
		serviceErr   *service.ServiceError
		expectedCode api.ErrorCode
	}{
		{"auth not found", &service.ServiceError{Code: service.ErrCodeAuthNotFound}, api.ErrorCodeAuthorizationNotFound},
		{"auth expired", &service.ServiceError{Code: service.ErrCodeAuthExpired}, api.ErrorCodeAuthorizationExpired},
		{"already captured", &service.ServiceError{Code: service.ErrCodeAlreadyCaptured}, api.ErrorCodeAlreadyCaptured},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCapture := mocks.NewMockCapturer(t)
			handler := NewHandler(nil, mockCapture, nil, nil, nil, testLogger())

			mockCapture.On("Capture", mock.Anything, mock.Anything, mock.Anything).
				Return(nil, tt.serviceErr)

			req := api.CreateCaptureRequestObject{
				Body: &api.CreateCaptureJSONRequestBody{
					AuthorizationId: "auth_" + uuid.New().String(),
					Amount:          10000,
				},
			}

			resp, err := handler.CreateCapture(context.Background(), req)
			require.NoError(t, err)

			badResp, ok := resp.(api.CreateCapture400JSONResponse)
			require.True(t, ok)
			assert.Equal(t, tt.expectedCode, badResp.Error)
		})
	}
}

func TestCreateCapture_InvalidIDFormat(t *testing.T) {
	handler := NewHandler(nil, nil, nil, nil, nil, testLogger())

	req := api.CreateCaptureRequestObject{
		Body: &api.CreateCaptureJSONRequestBody{
			AuthorizationId: "invalid",
			Amount:          10000,
		},
	}

	resp, err := handler.CreateCapture(context.Background(), req)

	require.NoError(t, err)
	badResp, ok := resp.(api.CreateCapture400JSONResponse)
	require.True(t, ok)
	assert.Equal(t, api.ErrorCodeAuthorizationNotFound, badResp.Error)
}

func TestGetCapture_Success(t *testing.T) {
	mockCapture := mocks.NewMockCapturer(t)
	handler := NewHandler(nil, mockCapture, nil, nil, nil, testLogger())

	authID := uuid.New()
	captureID := uuid.New()

	mockCapture.On("GetCapture", mock.Anything, captureID).
		Return(&models.Transaction{
			ID:          captureID,
			ReferenceID: &authID,
			AmountCents: 10000,
			Currency:    "USD",
			CreatedAt:   time.Now(),
		}, nil)

	req := api.GetCaptureRequestObject{CaptureId: "cap_" + captureID.String()}
	resp, err := handler.GetCapture(context.Background(), req)

	require.NoError(t, err)
	_, ok := resp.(api.GetCapture200JSONResponse)
	require.True(t, ok)
}

func TestGetCapture_NotFound(t *testing.T) {
	mockCapture := mocks.NewMockCapturer(t)
	handler := NewHandler(nil, mockCapture, nil, nil, nil, testLogger())

	captureID := uuid.New()
	mockCapture.On("GetCapture", mock.Anything, captureID).
		Return(nil, &service.ServiceError{Code: service.ErrCodeCaptureNotFound})

	req := api.GetCaptureRequestObject{CaptureId: "cap_" + captureID.String()}
	resp, err := handler.GetCapture(context.Background(), req)

	require.NoError(t, err)
	_, ok := resp.(api.GetCapture404JSONResponse)
	require.True(t, ok)
}
