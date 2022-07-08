package token

import (
	"fmt"
	"time"

	"github.com/o1egl/paseto"
	"golang.org/x/crypto/chacha20poly1305"
)

// PasetoMaker is a PASETO token maker
type PasetoMaker struct {
	// using the latest version of PASETO (version 2)
	paseto *paseto.V2
	// using only for internal backend API
	symmetricKey []byte
}

// NewPasetoMaker creates a new PasteoMaker
func NewPasetoMaker(symmetricKey string) (Maker, error) {
	// PASETO V2 uses the chacha poly algo. to encrypt the payload
	// we need to confirm the key size is correct
	if len(symmetricKey) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key size: must be exactly %d characters", chacha20poly1305.KeySize)
	}

	maker := &PasetoMaker{
		paseto:       paseto.NewV2(),       // returns a V2 implementation of PASETO tokens
		symmetricKey: []byte(symmetricKey), // symmetricKey converted to a slice of bytes
	}

	return maker, nil
}

// CreateToken creates a new token for a specific username and duration
func (maker *PasetoMaker) CreateToken(username string, duration time.Duration) (string, error) {
	payload, err := NewPayload(username, duration)
	if err != nil {
		return "", err
	}

	// encrypt the payload using the symmetricKey, the payload, and an optional footer (we use nil)
	return maker.paseto.Encrypt(maker.symmetricKey, payload, nil)
}

// VerifyToken checks if the token is valid or not
func (maker *PasetoMaker) VerifyToken(token string) (*Payload, error) {
	payload := &Payload{}

	err := maker.paseto.Decrypt(token, maker.symmetricKey, payload, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// check if payload is valid
	err = payload.Valid()
	if err != nil {
		return nil, err
	}

	return payload, nil
}
