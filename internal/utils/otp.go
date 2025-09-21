package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

func GenerateOTP() (string, error) {
	// Generate 6-digit OTP
	max := big.NewInt(999999)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	// Format as 6-digit string with leading zeros
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func IsOTPExpired(createdAt time.Time, expiryDuration time.Duration) bool {
	return time.Since(createdAt) > expiryDuration
}

func FormatPhoneNumber(phone string) string {
	// Remove all non-digit characters
	cleaned := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			cleaned += string(char)
		}
	}

	// Add Ethiopian country code if not present
	if len(cleaned) == 9 && cleaned[0] == '9' {
		return "+251" + cleaned
	}

	if len(cleaned) == 10 && cleaned[0] == '0' {
		return "+251" + cleaned[1:]
	}

	if len(cleaned) == 12 && cleaned[:3] == "251" {
		return "+" + cleaned
	}

	return "+" + cleaned
}
