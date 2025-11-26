package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ServiceError
		expected string
	}{
		{
			name: "error without underlying cause",
			err: &ServiceError{
				Code:    "test_error",
				Message: "test message",
			},
			expected: "test message",
		},
		{
			name: "error with underlying cause",
			err: &ServiceError{
				Code:    "test_error",
				Message: "test message",
				Err:     errors.New("underlying error"),
			},
			expected: "test message: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &ServiceError{
		Code:    "test_error",
		Message: "test message",
		Err:     underlying,
	}

	assert.Equal(t, underlying, err.Unwrap())
	assert.True(t, errors.Is(err, underlying))
}

func TestServiceError_NoUnwrap(t *testing.T) {
	err := &ServiceError{
		Code:    "test_error",
		Message: "test message",
	}

	assert.Nil(t, err.Unwrap())
}
