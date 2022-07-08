package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"github.com/techschool/simplebank/db/util"
)

func TestJWTMaker(t *testing.T) {
	// create a maker to test with
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	// create a random username for the token
	username := util.RandomOwner()
	// duration will be one minute
	duration := time.Minute

	issuedAt := time.Now()
	expiredAt := issuedAt.Add(duration)

	// create token
	token, err := maker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// get payload data
	payload, err := maker.VerifyToken(token)
	require.NoError(t, err)
	require.NotEmpty(t, payload)
	require.NotZero(t, payload.ID)
	require.Equal(t, username, payload.Username)
	// both issuedAt and expiredAt should be within one second of the times reported in the payload
	require.WithinDuration(t, issuedAt, payload.IssuedAt, time.Second)
	require.WithinDuration(t, expiredAt, payload.ExpiredAt, time.Second)
}

func TestExpiredJWTToken(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	// we create an expired token using a negative duration with CreateToken method
	token, err := maker.CreateToken(util.RandomOwner(), -time.Minute)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// grab payload data
	payload, err := maker.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrExpiredToken.Error())
	require.Nil(t, payload)
}

func TestInvalidJWTTokenAlgNone(t *testing.T) {
	// create a test payload with a random owner name for username and a duration of one minute
	payload, err := NewPayload(util.RandomOwner(), time.Minute)
	require.NoError(t, err)

	// create a test token using this test payload
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodNone, payload)
	// we need to sign the token; however, we cannot use a random secret key - the JWT Go package forbids signing with
	// the SigningMethodNone - we can use it for testing when passing in the special constant below
	token, err := jwtToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	// create a maker to call VerifyToken on this generated token, signed with SigningMethodNone
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	payload, err = maker.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrInvalidToken.Error())
	require.Nil(t, payload)
}
