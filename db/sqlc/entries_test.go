package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/techschool/simplebank/db/util" //provides random generator functions we defined in random.go
)

// createRandomEntry will create entries for the other test functions to use
func createRandomEntry(t *testing.T, account Account) Entry {
	arg := CreateEntryParams{
		AccountID: account.ID,
		Amount:    util.RandomMoney(),
	}

	entry, err := testQueries.CreateEntry(context.Background(), arg)
	require.NoError(t, err)    // for test to pass, there must be no error
	require.NotEmpty(t, entry) // for the test to pass, the entry object must not be empty

	require.Equal(t, arg.AccountID, entry.AccountID) // the account ID from arg must match the account ID of entry
	require.Equal(t, arg.Amount, entry.Amount)       // ensure the amount has remained the same from arg.amount to entry.amount

	require.NotZero(t, entry.ID) // postgres db should be auto generating non-zero IDs
	require.NotZero(t, entry.CreatedAt)

	return entry
}

func TestCreateEntry(t *testing.T) {
	account := createRandomAccount(t)
	createRandomEntry(t, account)
}

func TestGetEntry(t *testing.T) {
	account1 := createRandomAccount(t)       // create a random account to pass to CreateRandomEntry
	entry1 := createRandomEntry(t, account1) // create a random entry using account1.ID and a random amount (account1.amount)

	checkEntry, err := testQueries.GetEntry(context.Background(), entry1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, checkEntry)

	require.Equal(t, entry1.ID, checkEntry.ID)
	require.Equal(t, entry1.AccountID, checkEntry.AccountID)
	require.Equal(t, entry1.Amount, checkEntry.Amount)
	// require that the CreatedAt fields are within 1 second of each other
	require.WithinDuration(t, entry1.CreatedAt, checkEntry.CreatedAt, time.Second)
}

func TestUpdateEntry(t *testing.T) {
	account1 := createRandomAccount(t)
	entry1 := createRandomEntry(t, account1)

	arg := UpdateEntryParams{
		ID:     entry1.ID,
		Amount: util.RandomMoney(),
	}

	entry2, err := testQueries.UpdateEntry(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, entry2)

	// even though we only update the amount, we want to confirm the other properties haven't changed
	require.Equal(t, entry1.ID, entry2.ID)
	require.Equal(t, entry1.AccountID, entry2.AccountID)
	require.Equal(t, arg.Amount, entry2.Amount) // here we compare against the arg since it has the updated amount
	require.WithinDuration(t, entry1.CreatedAt, entry2.CreatedAt, time.Second)

}

func TestDeleteEntry(t *testing.T) {
	account1 := createRandomAccount(t)
	entry1 := createRandomEntry(t, account1)

	err := testQueries.DeleteEntry(context.Background(), entry1.ID)
	require.NoError(t, err) // to pass, err must be nil

	// can perform a get for entry1.ID to confirm it is deleted
	entry2, err := testQueries.GetEntry(context.Background(), entry1.ID)
	require.Error(t, err)                             // to pass, err must NOT be nil since entry1 should be deleted and cause an error
	require.EqualError(t, err, sql.ErrNoRows.Error()) // further check that the error is that there are no rows for the ID
	require.Empty(t, entry2)
}

func TestListEntries(t *testing.T) {
	account1 := createRandomAccount(t)
	// create 10 new entries all with the same AccountID to test with
	for i := 0; i < 10; i++ {
		createRandomEntry(t, account1)
	}

	arg := ListEntriesParams{
		AccountID: account1.ID,
		Limit:     5, // return five entry objects with AccountID = account1.ID
		Offset:    5, // skip the first five entry objects with AccountID = to account1.ID
	}

	entries, err := testQueries.ListEntries(context.Background(), arg)
	require.NoError(t, err) // to pass, err must be nil
	require.Len(t, entries, 5)

	for _, entry := range entries {
		require.NotEmpty(t, entry)                       // each entry in the slice of entries must not be empty
		require.Equal(t, arg.AccountID, entry.AccountID) // the list is of entries from a specific account ID and so they should match across all entries in the slice of entries
	}
}
