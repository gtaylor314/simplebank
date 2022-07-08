package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const minSecretKeySize = 32 // the secret key should be no smaller than 32 characters

// JWTMaker is a JSON web token maker - will implement the token maker interface
type JWTMaker struct {
	// will use symmetric signing algorithm to sign tokens
	secretKey string
}

// NewJWTMaker creates a new JWTMaker - by returning the interface Maker, we make
// sure that the JWTMaker implements the token Maker interface - checked by Go compiler
// i.e. we cannot return &JWTMaker{secretKey} unless *JWTMaker implements the token Maker interface
func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < minSecretKeySize {
		return nil, fmt.Errorf("invalid key size: must be at least %d characters", minSecretKeySize)
	}

	return &JWTMaker{secretKey}, nil
}

// CreateToken method will create a token for a specific username and duration
func (maker *JWTMaker) CreateToken(username string, duration time.Duration) (string, error) {
	payload, err := NewPayload(username, duration)
	if err != nil {
		return "", err
	}

	// creating a new JWT
	// jwt.NewWithClaims takes in the signing method and claims, which is our payload
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	// creates a signed JWT, signed using the signing method specified in the token (done above via jwt.NewWithClaims)
	return jwtToken.SignedString([]byte(maker.secretKey))
}

// VerifyToken method will confirm the provided token is valid or not
func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	// need to verify header to make sure the signing algorithm matches what is normally used to sign tokens
	// if it matches, you return the key so JWT can verify the token
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		// obtain the signing algorithm via token.Method - will be of type jwt.SigningMethod (this is an interface)
		// we therefore try to convert token.Method to a specific implementation (in this case jwt.SigningMethodHMAC)
		// we use jwt.SigningMethodHS256 when creating a token and it is an instance of the SigningMethodHMAC struct
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ErrInvalidToken
		}
		return []byte(maker.secretKey), nil
	}
	// parsing token - ParseWithClaims takens in the token, an empty Payload, and a key function
	// the key function receives the parsed, unverified token
	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		// two possible reasons for an error that isn't nil
		// since ParseWithClaims calls Valid for us, it takes our token expiration error we defined in our Valid method
		// and hides it within a ValidationError type (converts a non-validation error to ValidationError object and
		// a generic ClaimsInvalid flag set)
		// ParseWithClaims maintains the original error in its ValidationError object
		// therefore, we convert error to type jwt.ValidationError to see the actual error in the Inner property
		verr, ok := err.(*jwt.ValidationError)
		// if the conversion went through without issue, and verr.Inner is in fact the token expiration error, we return an
		// empty payload and the ErrExpiredToken error
		if ok && errors.Is(verr.Inner, ErrExpiredToken) {
			return nil, ErrExpiredToken
		}
		// if the error is not expired token, the token must be invalid
		return nil, ErrInvalidToken
	}

	// if token is successfully parsed and verified, grab payload data by converting jwtToken.Claims to a Payload object
	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		// something must be wrong with the token - hence invalid token
		return nil, ErrInvalidToken
	}
	return payload, nil
}
