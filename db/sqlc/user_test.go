package db

import (
	"context"
	"testing"
	"time"

	"SimpleBankProject/db/util"

	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T) User {
	// create a random hashed password for user
	hashedPassword, err := util.HashPassword(util.RandomString(6))
	require.NoError(t, err)

	arg := CreateUserParams{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		FullName:       util.RandomOwner(),
		Email:          util.RandomEmail(),
	}

	user, err := testQueries.CreateUser(context.Background(), arg) // testQueries object defined in main_test.go - the *Queries methods are defined in user.sql.go
	require.NoError(t, err)                                        // require.NoError() comes from /stretchr/testify/require - requires no errors and fails if error
	require.NotEmpty(t, user)                                      // requires that the user object not be an empty object

	// comparing the username, hashed password, full name, and email properities within arg and user to be sure they match
	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.HashedPassword, user.HashedPassword)
	require.Equal(t, arg.FullName, user.FullName)
	require.Equal(t, arg.Email, user.Email)

	// confirm that the new user's password_change_at value is the default zero value
	require.True(t, user.PasswordChangeAt.IsZero())
	// postgres db should be auto generating the correct time stamp
	// require.NotZero() asserts that a value must not be a zero value of its type
	require.NotZero(t, user.CreatedAt)

	return user
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func TestGetUser(t *testing.T) {
	user1 := createRandomUser(t)                                            // creating user to test with
	user2, err := testQueries.GetUser(context.Background(), user1.Username) // testQueries is our global *Queries variable and GetUser is a method with a *Queries receiver

	require.NoError(t, err)    // there must not be an error to pass unit test - err must be nil
	require.NotEmpty(t, user2) // the user object must not be empty to pass unit test

	// user2 gets user1 and should therefore match across all properties
	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
	require.Equal(t, user1.FullName, user2.FullName)
	require.Equal(t, user1.Email, user2.Email)
	// require that user2's PasswordChangeAt field has a value within one second of user1's PasswordChangeAt field value
	require.WithinDuration(t, user1.PasswordChangeAt, user2.PasswordChangeAt, time.Second)
	// require that user2's CreatedAt field has a value within one second of user1's CreatedAt field value
	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
}
