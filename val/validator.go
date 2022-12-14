package val

import (
	"fmt"
	"net/mail"
	"regexp"
)

var (
	// using regular expressions to validate the characters in a username
	// the values within [] are the characters allowed in the username i.e. a - z, 0 - 9, and underscore
	// the ^ symbol marks the beginning of the string, the + means the characters within the [] may appear more than once,
	// and the $ marks the end of the string - .MatchString turns isValidUsername into a function - MatchString() evaluates
	// if the input string contains any match of the regular expression object
	isValidUsername = regexp.MustCompile(`^[a-z0-9_]+$`).MatchString
	// the \\s means that spaces are allowed and the hyphen is allowed
	isValidFullName = regexp.MustCompile(`^[a-zA-Z\\s-]+$`).MatchString
)

// ValidateString validates that the input string meets the minimum and maximum length requirements
func ValidateString(value string, minLength int, maxLength int) error {
	strLen := len(value)
	if strLen < minLength || strLen > maxLength {
		return fmt.Errorf("must contain from %d-%d characters", minLength, maxLength)
	}
	return nil
}

// ValidateUsername validates that the input username is within the specified character limits
func ValidateUsername(username string) error {
	// arbitrarily chose 3 and 100 for minimum and maximum length
	if err := ValidateString(username, 3, 100); err != nil {
		return err
	}
	if !isValidUsername(username) {
		return fmt.Errorf("username must contain only lowercase letters, digits, or underscore")
	}
	return nil
}

// ValidateFullName validates that the input full name is within the specified character limits
func ValidateFullName(full_name string) error {
	// arbitrarily chose 3 and 100 for minimum and maximum length
	if err := ValidateString(full_name, 3, 100); err != nil {
		return err
	}
	if !isValidFullName(full_name) {
		return fmt.Errorf("full name must contain only letters, spaces, or hyphens")
	}
	return nil
}

// ValidatePassword validates that the input password is within the minimum and maximum length requirements
func ValidatePassword(password string) error {
	// password must be between 6 - 100 characters and can use any type of character
	return ValidateString(password, 6, 100)
}

// ValidateEmail validates that the input email is within the minimum and maximum length and is a valid email address
func ValidateEmail(email string) error {
	// arbitrarily chose 3 and 200 for minimum and maximum length
	if err := ValidateString(email, 3, 200); err != nil {
		return err
	}
	// confirming the input email is a valid email using Go's built-in mail package
	// ParseAddress() returns a parsed address and an error - only need the error
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("email is not a valid email address")
	}
	return nil
}
