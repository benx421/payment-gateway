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

func TestCreateRefund_Success(t *testing.T) {
	mockRefund := mocks.NewMockRefunder(t)
	handler := NewHandler(nil, nil, nil, mockRefund, nil, testLogger())

	captureID := uuid.New()
	refundID := uuid.New()

	mockRefund.On("Refund", mock.Anything, captureID, int64(5000)).
		Return(&models.Transaction{
			ID:          refundID,
			ReferenceID: &captureID,
			AmountCents: 5000,
			Currency:    "USD",
			CreatedAt:   time.Now(),
		}, nil)

	req := api.CreateRefundRequestObject{
		Body: &api.CreateRefundJSONRequestBody{
			CaptureId: "cap_" + captureID.String(),
			Amount:    5000,
		},
	}

	resp, err := handler.CreateRefund(context.Background(), req)

	require.NoError(t, err)
	successResp, ok := resp.(api.CreateRefund200JSONResponse)
	require.True(t, ok)
	assert.Equal(t, api.Refunded, successResp.Status)
}

func TestCreateRefund_ServiceErrors(t *testing.T) {
	tests := []struct {
		name         string
		serviceErr   *service.ServiceError
		expectedCode api.ErrorCode
	}{
		{"capture not found", &service.ServiceError{Code: service.ErrCodeCaptureNotFound}, api.ErrorCodeCaptureNotFound},
		{"already refunded", &service.ServiceError{Code: service.ErrCodeAlreadyRefunded}, api.ErrorCodeAlreadyRefunded},
		{"amount mismatch", &service.ServiceError{Code: service.ErrCodeAmountMismatch}, api.ErrorCodeAmountMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRefund := mocks.NewMockRefunder(t)
			handler := NewHandler(nil, nil, nil, mockRefund, nil, testLogger())

			mockRefund.On("Refund", mock.Anything, mock.Anything, mock.Anything).
				Return(nil, tt.serviceErr)

			req := api.CreateRefundRequestObject{
				Body: &api.CreateRefundJSONRequestBody{
					CaptureId: "cap_" + uuid.New().String(),
					Amount:    5000,
				},
			}

			resp, err := handler.CreateRefund(context.Background(), req)
			require.NoError(t, err)

			badResp, ok := resp.(api.CreateRefund400JSONResponse)
			require.True(t, ok)
			assert.Equal(t, tt.expectedCode, badResp.Error)
		})
	}
}

func TestCreateRefund_InvalidIDFormat(t *testing.T) {
	handler := NewHandler(nil, nil, nil, nil, nil, testLogger())

	req := api.CreateRefundRequestObject{
		Body: &api.CreateRefundJSONRequestBody{CaptureId: "invalid", Amount: 5000},
	}

	resp, err := handler.CreateRefund(context.Background(), req)

	require.NoError(t, err)
	badResp, ok := resp.(api.CreateRefund400JSONResponse)
	require.True(t, ok)
	assert.Equal(t, api.ErrorCodeCaptureNotFound, badResp.Error)
}

func TestGetRefund_Success(t *testing.T) {
	mockRefund := mocks.NewMockRefunder(t)
	handler := NewHandler(nil, nil, nil, mockRefund, nil, testLogger())

	captureID := uuid.New()
	refundID := uuid.New()

	mockRefund.On("GetRefund", mock.Anything, refundID).
		Return(&models.Transaction{
			ID:          refundID,
			ReferenceID: &captureID,
			AmountCents: 5000,
			Currency:    "USD",
			CreatedAt:   time.Now(),
		}, nil)

	req := api.GetRefundRequestObject{RefundId: "ref_" + refundID.String()}
	resp, err := handler.GetRefund(context.Background(), req)

	require.NoError(t, err)
	_, ok := resp.(api.GetRefund200JSONResponse)
	require.True(t, ok)
}

func TestGetRefund_NotFound(t *testing.T) {
	mockRefund := mocks.NewMockRefunder(t)
	handler := NewHandler(nil, nil, nil, mockRefund, nil, testLogger())

	refundID := uuid.New()
	mockRefund.On("GetRefund", mock.Anything, refundID).
		Return(nil, &service.ServiceError{Code: service.ErrCodeCaptureNotFound})

	req := api.GetRefundRequestObject{RefundId: "ref_" + refundID.String()}
	resp, err := handler.GetRefund(context.Background(), req)

	require.NoError(t, err)
	_, ok := resp.(api.GetRefund404JSONResponse)
	require.True(t, ok)
}
