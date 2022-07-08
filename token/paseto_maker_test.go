package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/techschool/simplebank/db/util"
)

func TestPasetoMaker(t *testing.T) {
	// create a maker to test with
	maker, err := NewPasetoMaker(util.RandomString(32))
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

func TestExpiredPasetoToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
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
