package token

import "time"

// Maker is an interface to manage the creation and verification of tokens
// we will implement both a JWT struct and a PASETO struct to implement this interface and easily switch between the two
type Maker interface {
	// CreateToken creates a new token for a specific username and duration
	CreateToken(username string, duration time.Duration) (string, error)
	// VerifyToken will confirm if the token is valid or not
	// if valid, VerifyToken will return the payload data of the token
	VerifyToken(token string) (*Payload, error)
}
