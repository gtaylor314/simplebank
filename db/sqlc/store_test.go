package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTx(t *testing.T) {
	store := NewStore(testDB)          // testDB is a global variable declared in main_test.go
	account1 := createRandomAccount(t) // createRandomAccount defined in random.go
	account2 := createRandomAccount(t)

	// print out some log information to see what is happening
	fmt.Println(">> Balance for account1 and account2 before:", account1.Balance, account2.Balance)

	// need to run n concurrent transfer transactions to properly test (use Go routines)
	n := 5              // running five concurrent transfer transactions - creates five transfer records
	amount := int64(10) // each transfer transaction will transfer 10 dollars from account1 to account2

	// as explained below, we will need to use channels in order to send result and err back to the main Go routine
	// that TestTransferTx is running on to ensure that failures across concurrent Go routines, stop the entire test
	errs := make(chan error)               // channel of type error
	results := make(chan TransferTxResult) // channel of type TransferTxResult

	for i := 0; i < n; i++ {
		// testing for deadlock even though we are using GetAccountForUpdate in store.go which blocks concurrent access
		// essentially, due to the foreign key dependencies of the transfer table (from_account_id and to_account_id)
		// any "get account" query must wait for changes to the accounts table to complete
		// before it can lock the database, run its query, and commit changes
		// this creates a deadlock if two processes are waiting on each other to finish
		// txName will be passed to TransferTX as context
		txName := fmt.Sprintf("Transaction %d", i+1)
		// go routine
		go func() {
			// to pass txName as context to TransferTX
			// context.WithValue() requires a key value that should NOT be a built in type
			// txKey declared in store.go as an empty struct
			ctx := context.WithValue(context.Background(), txKey, txName)

			// returns a TransferTxResult struct and an error
			// we cannot use the same require statements from testify
			// these transfers are happening in separate Go routines and a failure on one may not stop the entire test
			// we must send result and err back to the main go routine that TestTransferTx is running on and verify there
			// we use channels for this
			result, err := store.TransferTX(ctx, TransferTxParams{
				FromAccountID: account1.ID,
				ToAccountID:   account2.ID,
				Amount:        amount,
			})

			errs <- err       // send err over the errs channel
			results <- result // send result over the results channel
		}() // remember the parentheses at the end - they execute the function
	}

	// check the errs and results channels in the main Go routine
	// map will be used to check that k is unique for each transaction as described below
	existed := make(map[int]bool) // the key is an int but the value is a bool

	for i := 0; i < n; i++ {
		err := <-errs // assign to err what is received from the errs channel
		require.NoError(t, err)

		result := <-results         // assign to result what is received from the results channel
		require.NotEmpty(t, result) // confirm result is not an empty object

		// check each object within result to confirm they are not empty as well
		// check the transfer record object
		require.NotEmpty(t, result.Transfer)
		require.Equal(t, account1.ID, result.Transfer.FromAccountID)
		require.Equal(t, account2.ID, result.Transfer.ToAccountID)
		require.Equal(t, amount, result.Transfer.Amount)
		require.NotZero(t, result.Transfer.ID)
		require.NotZero(t, result.Transfer.CreatedAt)

		// additional checking to confirm the Transfer record is in the Transfer table
		// note that because of composition (including Queries struct in Store struct)
		// the Queries methods are available to the Store
		_, err = store.GetTransfer(context.Background(), result.Transfer.ID)
		require.NoError(t, err)

		// check the FromEntry entry record
		require.NotEmpty(t, result.FromEntry)
		require.Equal(t, account1.ID, result.FromEntry.AccountID)
		require.Equal(t, -amount, result.FromEntry.Amount) // remember, this should be negative because money is transfering out
		require.NotZero(t, result.FromEntry.ID)
		require.NotZero(t, result.FromEntry.CreatedAt)

		// check if the entries record is really in the entries table
		_, err = store.GetEntry(context.Background(), result.FromEntry.ID)
		require.NoError(t, err)

		// check the ToEntry entry record
		require.NotEmpty(t, result.ToEntry)
		require.Equal(t, account2.ID, result.ToEntry.AccountID)
		require.Equal(t, amount, result.ToEntry.Amount)
		require.NotZero(t, result.ToEntry.ID)
		require.NotZero(t, result.ToEntry.CreatedAt)

		_, err = store.GetEntry(context.Background(), result.ToEntry.ID)
		require.NoError(t, err)

		// check account records
		// check the FromAccount
		require.NotEmpty(t, result.FromAccount)
		require.Equal(t, account1.ID, result.FromAccount.ID)

		// check the ToAccount
		require.NotEmpty(t, result.ToAccount)
		require.Equal(t, account2.ID, result.ToAccount.ID)

		// check the accounts' balance
		// print out the account balance after each transaction
		fmt.Println("Balance of account1 and account2 after each transaction:", result.FromAccount.Balance, result.ToAccount.Balance)
		// check the from account balance
		// remember the test makes five transactions - as such five times the amount is being transferred from account 1 to account 2
		diff1 := account1.Balance - result.FromAccount.Balance // once code is implemented, diff1 should equal the amount of money transfered to account 2
		diff2 := result.ToAccount.Balance - account2.Balance   // once code is implemented, diff2 should also equal the amount of money transfered to account 2
		require.Equal(t, diff1, diff2)
		require.True(t, diff1 > 0) // both diff1 and diff2 should be positive (since diff1 equals diff2 as required above, we can check diff1 and trust diff2 is also positive)
		// since our test makes n transactions, diff1/amount == n
		require.True(t, diff1%amount == 0) // Testing that the remainder is zero allows for n to change but the test to remain valid

		// since our test makes n transactions, we need to confirm that n transactions run
		// we also need to confirm that k is unique for each transaction
		// first transaction, k == 1, second transaction, k == 2 ... n transaction, k == n
		// remember this is only when comparing the original balance to that transaction's new balance since we are transfering the same amount of money each time
		k := int(diff1 / amount)
		require.True(t, k >= 1 && k <= n) // k must have run between 1 to n times

		// to test that k is unique for each transaction
		require.NotContains(t, existed, k) // remember this is in a for loop so we are confirming that k is unique and then adding it to the map
		existed[k] = true
	}

	// out of the for loop
	// check the final balances
	// first get the updated account record for account1
	updatedAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	// get the updated account record for account2
	updatedAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	// print out the updated balance
	fmt.Println("Balance of account1 and account2 after:", account1.Balance, account2.Balance)
	// check the final balance of the from account
	// account1 remains untouched since it was declared - in other words, account1.Balance is the original balance
	require.Equal(t, account1.Balance-int64(n)*amount, updatedAccount1.Balance)
	// account 2 remains untouched since it was declared
	require.Equal(t, account2.Balance+int64(n)*amount, updatedAccount2.Balance)
}

// TestTransferTxDeadlock - tests what happens when multiple concurrent transactions do different things. Previous test runs five
// identical transactions but deadlock could occur if two transactions need to update each other's balance
func TestTransferTxDeadlock(t *testing.T) {
	store := NewStore(testDB)          // testDB is a global variable declared in main_test.go
	account1 := createRandomAccount(t) // createRandomAccount defined in random.go
	account2 := createRandomAccount(t)

	// need to run n concurrent transfer transactions to properly test (use Go routines)
	n := 10             // running ten concurrent transfer transactions - creates five transfer records from account1 to account2 and five transfer records from account2 to account1
	amount := int64(10) // each transfer transaction will transfer 10 dollars

	// as explained below, we will need to use channels in order to send err back to the main Go routine
	// that TestTransferTxDeadlock is running on to ensure that failures across concurrent Go routines, stop the entire test
	errs := make(chan error) // channel of type error

	for i := 0; i < n; i++ {

		fromAccountID := account1.ID
		toAccountID := account2.ID

		// half of the n transactions need to be from account2 to account1
		// check if i is an odd number
		if i%2 == 1 {
			fromAccountID = account2.ID
			toAccountID = account1.ID
		}

		// go routine
		go func() {

			// returns a TransferTxResult struct and an error
			// we cannot use the same require statements from testify
			// these transfers are happening in separate Go routines and a failure on one may not stop the entire test
			// we must send result and err back to the main go routine that TestTransferTx is running on and verify there
			// we use channels for this
			_, err := store.TransferTX(context.Background(), TransferTxParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        amount,
			})

			errs <- err // send err over the errs channel
		}() // remember the parentheses at the end - they execute the function
	}

	// check the errs channels in the main Go routine

	for i := 0; i < n; i++ {
		err := <-errs // assign to err what is received from the errs channel
		require.NoError(t, err)

	}

	// out of the for loop
	// check the final balances
	// first get the updated account record for account1
	updatedAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	// get the updated account record for account2
	updatedAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)

	// check the final balance of account1 and account2
	// since half the transactions move x amount of money from account1 to account2 and the other half move x amount from
	// account2 to account1, the final balance before and after all of the transactions, should be the same
	// account1 remains untouched since it was declared - in other words, account1.Balance is the original balance
	require.Equal(t, account1.Balance, updatedAccount1.Balance)
	// account 2 remains untouched since it was declared
	require.Equal(t, account2.Balance, updatedAccount2.Balance)
}
