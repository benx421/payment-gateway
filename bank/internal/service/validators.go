package service

import (
	"fmt"
	"time"
)

// ValidateLuhn validates a card number using the Luhn algorithm
func ValidateLuhn(cardNumber string) error {
	var digits []int
	for _, r := range cardNumber {
		if r >= '0' && r <= '9' {
			digits = append(digits, int(r-'0'))
		}
	}

	if len(digits) < 13 || len(digits) > 19 {
		return fmt.Errorf("invalid card number length: must be 13-19 digits")
	}

	sum := 0
	isSecond := false

	for i := len(digits) - 1; i >= 0; i-- {
		digit := digits[i]

		if isSecond {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		isSecond = !isSecond
	}

	if sum%10 != 0 {
		return fmt.Errorf("invalid card number: failed Luhn check")
	}

	return nil
}

// ValidateExpiry checks if a card has expired
func ValidateExpiry(expiryMonth, expiryYear int) error {
	if expiryMonth < 1 || expiryMonth > 12 {
		return fmt.Errorf("invalid month: must be between 1 and 12")
	}

	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	if expiryYear < currentYear {
		return fmt.Errorf("card expired: year %d is in the past", expiryYear)
	}

	if expiryYear == currentYear && expiryMonth < currentMonth {
		return fmt.Errorf("card expired: %02d/%d", expiryMonth, expiryYear)
	}

	return nil
}

// ValidateCVV checks if CVV format is valid.
func ValidateCVV(cvv string) error {
	if len(cvv) < 3 || len(cvv) > 4 {
		return fmt.Errorf("invalid CVV: must be 3 or 4 digits")
	}

	for _, r := range cvv {
		if r < '0' || r > '9' {
			return fmt.Errorf("invalid CVV: must contain only digits")
		}
	}

	return nil
}

// ValidateAmount checks if amount is valid (positive)
func ValidateAmount(amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("invalid amount: must be greater than 0")
	}

	return nil
}
