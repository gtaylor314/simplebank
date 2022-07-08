package token

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// declaring as a public const allows us to check for these errors outside of the package
var (
	ErrInvalidToken = errors.New("token is invalid")
	ErrExpiredToken = errors.New("token has expired")
)

// Payload will contain the payload data of the token
type Payload struct {
	ID        uuid.UUID `json:"id"` // can use ID to invalid tokens in the future if found to be leaked
	Username  string    `json:"username"`
	IssuedAt  time.Time `json:"issued_at"`  // when the token was created
	ExpiredAt time.Time `json:"expired_at"` //when the token will expire
}

// NewPayload creates a new token payload with a specific username/duration
func NewPayload(username string, duration time.Duration) (*Payload, error) {
	// create a new ID
	tokenID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	// create payload
	payload := &Payload{
		ID:        tokenID,
		Username:  username,
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(duration),
	}

	return payload, nil
}

// Valid method checks if the token payload is valid or not
// *Payload needs a Valid method in order to implement jwt.Claims interface
func (payload *Payload) Valid() error {
	if time.Now().After(payload.ExpiredAt) {
		return ErrExpiredToken
	}

	return nil
}
