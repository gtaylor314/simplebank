package util

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns the bcrypt hash of the password and an error
func HashPassword(password string) (string, error) {
	// GenerateFromPassword requires the password be a slice of bytes, bcrypt.DefaultCost is 10
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// return an empty string
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// CheckPassword checks the provided password against the hashedPassword to ensure it is correct
func CheckPassword(password string, hashedPassword string) error {
	// CompareHashAndPassword provided by the bcrypt package
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
