package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"SimpleBankProject/db/util"

	"github.com/stretchr/testify/require"
)

// needed for TestListTransfers - enables multiple transfers to be created using specific account IDs
func createMultiTransfers(t *testing.T, id1, id2 int64) Transfer {
	arg := CreateTransferParams{
		FromAccountID: id1,
		ToAccountID:   id2,
		Amount:        util.RandomMoney(),
	}

	transfer, err := testQueries.CreateTransfer(context.Background(), arg)
	require.NoError(t, err)       // err must be nil
	require.NotEmpty(t, transfer) // transfer must not be empty

	// ID and CreatedAt must not be zero
	require.NotZero(t, transfer.ID)
	require.NotZero(t, transfer.CreatedAt)

	return transfer
}

func createRandomTransfer(t *testing.T) Transfer {
	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	arg := CreateTransferParams{
		FromAccountID: account1.ID,
		ToAccountID:   account2.ID,
		Amount:        util.RandomMoney(),
	}

	transfer, err := testQueries.CreateTransfer(context.Background(), arg)
	require.NoError(t, err)       // to pass, err must be nil
	require.NotEmpty(t, transfer) // to pass, transfer must not be empty

	require.Equal(t, arg.FromAccountID, transfer.FromAccountID) // arg.FromAccountID (which is account1.ID) should match transfer.FromAccountID
	require.Equal(t, arg.ToAccountID, transfer.ToAccountID)     // arg.ToAccountID (which is account2.ID) should match transfer.ToAccountID
	require.Equal(t, arg.Amount, transfer.Amount)               // arg.Amount should match transfer.Amount

	// postgresql should be auto-generating non-zero IDs and time stamps with time zone
	require.NotZero(t, transfer.ID)
	require.NotZero(t, transfer.CreatedAt)

	return transfer
}

func TestCreateTransfer(t *testing.T) {
	createRandomTransfer(t)
}

func TestGetTransfer(t *testing.T) {
	// create a random transfer to test with
	transfer1 := createRandomTransfer(t)

	transfer2, err := testQueries.GetTransfer(context.Background(), transfer1.ID)
	require.NoError(t, err)        // to pass, err must be nil
	require.NotEmpty(t, transfer2) // to pass, transfer2 must not be empty

	require.Equal(t, transfer1.ID, transfer2.ID)
	require.Equal(t, transfer1.FromAccountID, transfer2.FromAccountID)
	require.Equal(t, transfer1.ToAccountID, transfer2.ToAccountID)
	require.Equal(t, transfer1.Amount, transfer2.Amount)

	require.WithinDuration(t, transfer1.CreatedAt, transfer2.CreatedAt, time.Second)
}

func TestUpdateTransfer(t *testing.T) {
	transfer1 := createRandomTransfer(t)

	arg := UpdateTransferParams{
		ID:     transfer1.ID,
		Amount: util.RandomMoney(),
	}

	transfer2, err := testQueries.UpdateTransfer(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transfer2)

	require.Equal(t, transfer1.ID, transfer2.ID)
	require.Equal(t, arg.Amount, transfer2.Amount) // this is the one property that is updated
	require.Equal(t, transfer1.FromAccountID, transfer2.FromAccountID)
	require.Equal(t, transfer1.ToAccountID, transfer2.ToAccountID)
	require.WithinDuration(t, transfer1.CreatedAt, transfer2.CreatedAt, time.Second)
}

func TestDeleteTransfer(t *testing.T) {
	transfer1 := createRandomTransfer(t)

	err := testQueries.DeleteTransfer(context.Background(), transfer1.ID)
	require.NoError(t, err)

	// for added testing
	transfer2, err := testQueries.GetTransfer(context.Background(), transfer1.ID)
	require.Error(t, err) // we expect an error as transfer1 should be deleted
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, transfer2) // transfer2 should therefore be empty
}

func TestListTransfers(t *testing.T) {
	transfer1 := createRandomTransfer(t)

	for i := 0; i < 5; i++ {
		createMultiTransfers(t, transfer1.FromAccountID, transfer1.ToAccountID) // create transfers using the FromAccountID as id1 and the ToAccountID as id2
		createMultiTransfers(t, transfer1.ToAccountID, transfer1.FromAccountID) // create transfers using the ToAccountID as id1 and the FromAccountID as id2
	}

	arg1 := ListTransfersParams{
		FromAccountID: transfer1.FromAccountID, // looking for table entries that have transfer1.FromAccountID as the FromAccountID OR (see below)
		ToAccountID:   transfer1.FromAccountID, // entries that have transfer1.FromAccountID as the ToAccountID
		Limit:         5,                       // select five transfer objects
		Offset:        5,                       // skip the first five transfer objects
	}

	transfers, err := testQueries.ListTransfers(context.Background(), arg1)
	require.NoError(t, err)
	require.Len(t, transfers, 5)

	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
		require.True(t, transfer.FromAccountID == transfer1.FromAccountID || transfer.ToAccountID == transfer1.FromAccountID)
	}
}
