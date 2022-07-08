package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"      // stretchr/testify makes several packages available that provides testing tools
	"github.com/techschool/simplebank/db/util" // provides random generator functions that we defined in random.go
)

// since every unit test will need to create an account for testing the CRUD ops - we create a func which we can call to avoid code duplication
// this allows us to modify a unit test function without impacting every other unit test function - e.g. if we used TestCreateAccount to create accounts for all unit tests and then modified it
func createRandomAccount(t *testing.T) Account {
	// due to the foreign key constraint on account's owner (must tie back to a user), we generate a user first
	user := createRandomUser(t)
	arg := CreateAccountParams{
		Owner:    user.Username,
		Balance:  util.RandomMoney(),
		Currency: util.RandomCurrency(),
	}

	account, err := testQueries.CreateAccount(context.Background(), arg) // testQueries object defined in main_test.go - the *Queries methods are defined in account.sql.go
	require.NoError(t, err)                                              // require.NoError() comes from /stretchr/testify/require - requires no errors and fails if error
	require.NotEmpty(t, account)                                         // requires that the account object not be an empty object

	// comparing the owner, balance, and currency properities within arg and account to be sure they match
	require.Equal(t, arg.Owner, account.Owner)
	require.Equal(t, arg.Balance, account.Balance)
	require.Equal(t, arg.Currency, account.Currency)

	// postgres db should be auto generating non-zero IDs and the correct time stamp
	// require.NotZero() asserts that a value must not be a zero value of its type
	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	return account
}

func TestCreateAccount(t *testing.T) {
	createRandomAccount(t)
}

func TestGetAccount(t *testing.T) {
	account1 := createRandomAccount(t)                                         // creating account to test with
	account2, err := testQueries.GetAccount(context.Background(), account1.ID) // testQueries is our global *Queries variable and GetAccount is a method with a *Queries receiver

	require.NoError(t, err)       // there must not be an error to pass unit test - err must be nil
	require.NotEmpty(t, account2) // the account object must not be empty to pass unit test

	// account2 gets account1 and should therefore match across all properties
	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	// require that account2's CreatedAt field has a value within one second of account1's CreatedAt field value
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

func TestUpdateAccount(t *testing.T) {
	account1 := createRandomAccount(t)

	// declare the arguments - UpdateAccountParams object is defined in account.sql.go
	arg := UpdateAccountParams{
		ID:      account1.ID,        // ID provides which account we are updating
		Balance: util.RandomMoney(), // Balance is the new balance value which needs to replace the old balance value
	}

	account2, err := testQueries.UpdateAccount(context.Background(), arg)
	require.NoError(t, err)       // err must be nil, meaning no error occurred
	require.NotEmpty(t, account2) // account2 must not be empty

	// account2 should match account1 in all properties but balance
	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, arg.Balance, account2.Balance) // here we compare account2 against the UpdateAccountParams balance
	require.Equal(t, account1.Currency, account2.Currency)
	// require that account2's CreatedAt field have a value within one second of account1's CreatedAt field value
	require.WithinDuration(t, account1.CreatedAt, account2.CreatedAt, time.Second)
}

func TestDeleteAccount(t *testing.T) {
	account1 := createRandomAccount(t)
	err := testQueries.DeleteAccount(context.Background(), account1.ID)
	require.NoError(t, err) // err must be nil, meaning no error occurred

	// as another test to ensure account1 is deleted
	account2, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.Error(t, err)                             // in this case we would want an error as account1 has been deleted
	require.EqualError(t, err, sql.ErrNoRows.Error()) // we specifically check that the error matches SQL's no rows error
	require.Empty(t, account2)                        // in this case account2 should also be empty as account1 is deleted
}

func TestListAccounts(t *testing.T) {
	// always begin by creating accounts - since we need to return a slice of account objects, we need to create a few accounts to test with
	// retroactively adding filter by owner/username breaks test - to resolve, we grab the owner of the last randomly generated
	// account and use it in the owner property of ListAccountsParams
	var lastAccount Account
	for i := 0; i < 10; i++ {
		// update lastAccount until it finally has the last account
		lastAccount = createRandomAccount(t)
	}

	arg := ListAccountsParams{
		Owner:  lastAccount.Owner,
		Limit:  5, // return five account objects in the slice of account objects - even if the table starts out empty, 10 accounts are created for testing so five must be returned
		Offset: 0, // skip the first "x" account objects
	}

	accounts, err := testQueries.ListAccounts(context.Background(), arg)
	require.NoError(t, err) // err must be nil, meaning no errors
	require.NotEmpty(t, accounts)

	for _, account := range accounts {
		require.NotEmpty(t, account)                       // each account in the list must not be empty
		require.Equal(t, lastAccount.Owner, account.Owner) // each account must have an owner that matches the lastAccount.Username
	}
}
