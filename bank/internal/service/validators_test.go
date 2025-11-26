package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateLuhn(t *testing.T) {
	tests := []struct {
		name       string
		cardNumber string
		wantErr    bool
	}{
		{
			name:       "valid card number",
			cardNumber: "4532015112830366",
			wantErr:    false,
		},
		{
			name:       "another valid card",
			cardNumber: "4556737586899855",
			wantErr:    false,
		},
		{
			name:       "invalid card number",
			cardNumber: "1234567890123456",
			wantErr:    true,
		},
		{
			name:       "empty card number",
			cardNumber: "",
			wantErr:    true,
		},
		{
			name:       "non-numeric card",
			cardNumber: "abcd1234efgh5678",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLuhn(tt.cardNumber)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateCVV(t *testing.T) {
	tests := []struct {
		name    string
		cvv     string
		wantErr bool
	}{
		{
			name:    "valid 3-digit CVV",
			cvv:     "123",
			wantErr: false,
		},
		{
			name:    "valid 4-digit CVV",
			cvv:     "1234",
			wantErr: false,
		},
		{
			name:    "too short",
			cvv:     "12",
			wantErr: true,
		},
		{
			name:    "too long",
			cvv:     "12345",
			wantErr: true,
		},
		{
			name:    "non-numeric",
			cvv:     "abc",
			wantErr: true,
		},
		{
			name:    "empty",
			cvv:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCVV(tt.cvv)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateExpiry(t *testing.T) {
	tests := []struct {
		name        string
		expiryMonth int
		expiryYear  int
		wantErr     bool
	}{
		{
			name:        "valid future date",
			expiryMonth: 12,
			expiryYear:  2030,
			wantErr:     false,
		},
		{
			name:        "invalid month - too low",
			expiryMonth: 0,
			expiryYear:  2025,
			wantErr:     true,
		},
		{
			name:        "invalid month - too high",
			expiryMonth: 13,
			expiryYear:  2025,
			wantErr:     true,
		},
		{
			name:        "expired card",
			expiryMonth: 1,
			expiryYear:  2020,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExpiry(tt.expiryMonth, tt.expiryYear)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAmount(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		wantErr bool
	}{
		{
			name:    "valid amount",
			amount:  1000,
			wantErr: false,
		},
		{
			name:    "zero amount invalid",
			amount:  0,
			wantErr: true,
		},
		{
			name:    "negative amount invalid",
			amount:  -100,
			wantErr: true,
		},
		{
			name:    "large valid amount",
			amount:  1000000,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAmount(tt.amount)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
