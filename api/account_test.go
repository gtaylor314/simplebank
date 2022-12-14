package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mockdb "SimpleBankProject/db/mock"
	db "SimpleBankProject/db/sqlc"
	"SimpleBankProject/db/util"
	"SimpleBankProject/token"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCreateAccountAPI(t *testing.T) {
	// create random account for testing
	// this requires a random user as we've added authentication and authorization logic to the handlers
	// we do not care about the password returned in this case
	user, _ := randomUser(t)
	account := randomAccount(user.Username)
	// structs for passing to CreateAccount with a valid or invalid owner and currency and balance of zero
	validCreateAccount := db.CreateAccountParams{Owner: user.Username, Currency: account.Currency, Balance: 0}
	invalidCreateAccount := db.CreateAccountParams{Owner: "", Currency: "CREDITS", Balance: 0}

	// test cases - slice of structs
	testCases := []struct {
		name               string
		createAccountInput db.CreateAccountParams
		// since adding authentication and authorization logic, we need to setup the authentication for each case
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			// must have an owner and a currency of USD or EUR
			createAccountInput: validCreateAccount,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// addAuthorization defined in middleware_test
				// addAuthorization creates the token, creates the authentication header, and adds header to request
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// EXPECT() returns an object of *MockStoreMockRecorder and indicates expected use
				// CreateAccount expects two generic inputs since CreateAccount defined in account.sql.go takes two inputs
				// the context argument (first argument) could be any value so we use gomock.Any() matcher
				// .Times(n) means the expected method should run n times
				// .Return(account, nil) means that we expect the method to return the account object and a nil error
				// expect CreateAccount to be called once and to return a valid account with no error
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Eq(validCreateAccount)).Times(1).Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				// the created account should match the random account generated above after gomock simulates the creation
				requireBodyMatchAccount(t, recorder.Body, account)
			},
		},
		{
			name:               "Internal Error",
			createAccountInput: validCreateAccount,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// expect CreateAccount to run one time and fail thus returning an empty account and an internal error
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Eq(validCreateAccount)).Times(1).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:               "Invalid Params",
			createAccountInput: invalidCreateAccount, // invalid owner and currency
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// for any invalid parameters, we do not expect CreateAccount to run
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		// tc will store the data of the current test case
		tc := testCases[i]

		// running each case as a separate sub-test of this unit test
		t.Run(tc.name, func(t *testing.T) {
			// to call NewMockStore, we need a *gomock.Controller object - here we create one
			ctrl := gomock.NewController(t)
			// we defer calling ctrl.Finish() - Finish() checks to see if all the methods that were expected to be called, were called
			// Finish() should be invoked for each controller - checks store.EXPECT was satisfied
			defer ctrl.Finish()

			// declare a new store
			store := mockdb.NewMockStore(ctrl)
			// build stub relevant to test case
			tc.buildStubs(store)

			// start test HTTP server and send request - this is not an actual server
			// will use recorder to record the response of the api request
			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := "/accounts"
			jsonText := fmt.Sprintf("{\"owner\":\"%v\", \"currency\":\"%v\"}", tc.createAccountInput.Owner, tc.createAccountInput.Currency)
			jsonBody := strings.NewReader(jsonText)
			// generate a new HTTP request with MethodPost to the url
			request, err := http.NewRequest(http.MethodPost, url, jsonBody)
			require.NoError(t, err)

			// creating token and then the authentication header - adding header to request
			tc.setupAuth(t, request, server.tokenMaker)

			// sends our api request through the server router and records response in the recorder
			server.router.ServeHTTP(recorder, request)

			// check response
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetAccountAPI(t *testing.T) {
	user, _ := randomUser(t)
	account := randomAccount(user.Username)

	// declare a slice of test cases - uses an anonymous class to store the test data
	testCases := []struct {
		name string
		// accountID that we want to get
		accountID  int64
		setupAuth  func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs func(store *mockdb.MockStore)
		// check the output of the API
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		// adding scenarios
		{
			name:      "OK",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// building stub for this MockStore
				// in this situation, we only care about the GetAccount method
				// it is the only method that should be called by the GetAccount API handler
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check the response
				require.Equal(t, http.StatusOK, recorder.Code)
				// requireBodyMatchAccount - defined below - compares recorder.Body with account to ensure they are the same
				requireBodyMatchAccount(t, recorder.Body, account)
			},
		},
		{
			name:      "Unauthorized User",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, "unauthorized_user", time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// building stub for this MockStore
				// in this situation, we only care about the GetAccount method
				// it is the only method that should be called by the GetAccount API handler
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check the response
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:      "No Authorization",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// client did not complete the token and authentication header creation
			},
			buildStubs: func(store *mockdb.MockStore) {
				// building stub for this MockStore
				// in this situation, we only care about the GetAccount method
				// it is the only method that should be called by the GetAccount API handler
				// the middleware will abort the request due to no authorization
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check the response
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:      "Not Found",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// here we do not expect to find the account
				// we expect an empty account and the no rows found error
				// this is simulated with the acutal GetAccount method to confirm the StatusNotFound is returned
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check the response
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "Internal Error",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// here we expect an internal error
				// we expect an empty account and errconndone (this is one example of an internal error we can use to mock the
				// http.StatusInternalServerError) - this is simulated with the acutal GetAccount method to confirm the
				// StatusInternalServerError is returned
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check the response
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Invalid ID",
			// use an invalid ID to cause a bad request
			accountID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// here we expect a bad request error
				// for any context with any invalid id, GetAccount should never be called
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				// check the response
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	// for loop to iterate through the test cases
	for i := range testCases {
		// tc will store the data of the current test case
		tc := testCases[i]

		// running each case as a separate sub-test of this unit test
		t.Run(tc.name, func(t *testing.T) {
			// to call NewMockStore, we need a *gomock.Controller object - here we create one
			ctrl := gomock.NewController(t)
			// we defer calling ctrl.Finish() - Finish() checks to see if all the methods that were expected to be called, were called
			// Finish() should be invoked for each controller - checks store.EXPECT was satisfied
			defer ctrl.Finish()

			// declare a new store
			store := mockdb.NewMockStore(ctrl)
			// build stub relevant to test case
			tc.buildStubs(store)

			// start test HTTP server and send request - this is not an actual server
			// will use recorder to record the response of the api request
			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			// Sprintf - formats according to a format specifier and then returns the resulting string
			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			// generate a new HTTP request with MethodGet to the url - since it is a Get, we can use nil for the request body
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// create the token and then the authentication header - update request with header
			tc.setupAuth(t, request, server.tokenMaker)

			// sends our api request through the server router and records response in the recorder
			server.router.ServeHTTP(recorder, request)

			// check response
			tc.checkResponse(t, recorder)
		})
	}
}

type getQueryParams struct {
	pageID   int32
	pageSize int32
}

func TestListAccountAPI(t *testing.T) {
	user, _ := randomUser(t)
	accounts := make([]db.Account, 10) // slice with a length of 10
	for i := 0; i < 10; i++ {
		accounts[i] = randomAccount(user.Username)
	}

	// valid pageID and pageSize
	validQuery := getQueryParams{pageID: 1, pageSize: 5}
	// invalid pageID and pageSize
	invalidQuery := getQueryParams{pageID: 0, pageSize: 0}
	// limit is set to pageSize and offset is set to (pageID - 1) * pageSize
	validListAccounts := db.ListAccountsParams{Owner: user.Username, Limit: validQuery.pageSize, Offset: (validQuery.pageID - 1) * validQuery.pageSize}
	// since pagesize is restricted to between five and ten, limit cannot be zero
	invalidListAccounts := db.ListAccountsParams{Owner: user.Username, Limit: invalidQuery.pageSize, Offset: (invalidQuery.pageID - 1) * invalidQuery.pageSize}

	// create test cases for testing
	testCases := []struct {
		name              string
		queryInput        getQueryParams
		listAccountsInput db.ListAccountsParams
		setupAuth         func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs        func(store *mockdb.MockStore)
		checkResponse     func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:              "OK",
			queryInput:        validQuery,
			listAccountsInput: validListAccounts,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListAccounts(gomock.Any(), gomock.Eq(validListAccounts)).Times(1).Return(accounts[:5], nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchAccounts(t, recorder.Body, accounts)
			},
		},
		{
			name:              "Internal Error",
			queryInput:        validQuery,
			listAccountsInput: validListAccounts,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListAccounts(gomock.Any(), gomock.Eq(validListAccounts)).Times(1).Return([]db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:              "Invalid Params",
			queryInput:        invalidQuery,
			listAccountsInput: invalidListAccounts,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	// for loop to iterate through test cases
	for i := range testCases {
		// tc will store the current test case data
		tc := testCases[i]
		// running each case as a separate sub-test of this unit test
		t.Run(tc.name, func(t *testing.T) {
			// to call NewMockStore, we need a *gomock.Controller object - here we create one
			ctrl := gomock.NewController(t)
			// we defer calling ctrl.Finish() - Finish() checks to see if all the methods that were expected to be called, were called
			// Finish() should be invoked for each controller - checks store.EXPECT was satisfied
			defer ctrl.Finish()

			// declare a new store
			store := mockdb.NewMockStore(ctrl)
			// build stub relevant to test case using tc to call it
			tc.buildStubs(store)

			// start test HTTP server and send request - this is not an actual server
			// will use recorder to record the response of the api request
			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			// Sprintf - formats according to a format specifier and then returns the resulting string
			// page_id and page_size are the json tags used in account.go
			url := fmt.Sprintf("/accounts?page_id=%d&page_size=%d", tc.queryInput.pageID, tc.queryInput.pageSize)

			// generate a new HTTP request with MethodGet to the url
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// create the token and then the authorization header - update request with header
			tc.setupAuth(t, request, server.tokenMaker)

			// sends our api request through the server router and records response in the recorder
			server.router.ServeHTTP(recorder, request)

			// check response
			tc.checkResponse(t, recorder)
		})
	}
}

func TestUpdateAccountAPI(t *testing.T) {
	user, _ := randomUser(t)
	account := randomAccount(user.Username)
	// copy made to update balance without impacting other tests
	copyAccount := account
	// valid ID and balance for updating account at ID with the balance
	validUpdateAccount := db.UpdateAccountParams{ID: account.ID, Balance: 500}
	// invalid ID as ID cannot be zero
	invalidUpdateAccount := db.UpdateAccountParams{ID: 0, Balance: 100}

	// create test cases for testing
	testCases := []struct {
		name               string
		updateAccountInput db.UpdateAccountParams
		setupAuth          func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:               "OK",
			updateAccountInput: validUpdateAccount,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// update the balance
				copyAccount.Balance = validUpdateAccount.Balance

				first := store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(validUpdateAccount.ID)).Times(1).Return(account, nil)
				second := store.EXPECT().UpdateAccount(gomock.Any(), gomock.Eq(validUpdateAccount)).Times(1).Return(copyAccount, nil)
				// ensure the Get Account is called first and the Delete Account is called second
				gomock.InOrder(first, second)

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchAccount(t, recorder.Body, copyAccount)
			},
		},
		{
			name:               "Not Found",
			updateAccountInput: validUpdateAccount,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				first := store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(validUpdateAccount.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
				second := store.EXPECT().UpdateAccount(gomock.Any(), gomock.Eq(validUpdateAccount)).Times(0)

				gomock.InOrder(first, second)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:               "Internal Error Get Account",
			updateAccountInput: validUpdateAccount,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				first := store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(validUpdateAccount.ID)).Times(1).Return(db.Account{}, sql.ErrConnDone)
				second := store.EXPECT().UpdateAccount(gomock.Any(), gomock.Eq(validUpdateAccount)).Times(0)
				gomock.InOrder(first, second)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:               "Internal Error Update Account",
			updateAccountInput: validUpdateAccount,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				first := store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(validUpdateAccount.ID)).Times(1).Return(account, nil)
				second := store.EXPECT().UpdateAccount(gomock.Any(), gomock.Eq(validUpdateAccount)).Times(1).Return(db.Account{}, sql.ErrConnDone)
				gomock.InOrder(first, second)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:               "Invalid Params",
			updateAccountInput: invalidUpdateAccount,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// with any invalid update parameters, we expect GetAccount and UpdateAccount not to run
				first := store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				second := store.EXPECT().UpdateAccount(gomock.Any(), gomock.Any()).Times(0)
				gomock.InOrder(first, second)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for i := range testCases {
		// tc will store the data of the current test case
		tc := testCases[i]

		// running each case as a separate sub-test of this unit test
		t.Run(tc.name, func(t *testing.T) {
			// to call NewMockStore, we need a *gomock.Controller object - here we create one
			ctrl := gomock.NewController(t)
			// we defer calling ctrl.Finish() - Finish() checks to see if all the methods that were expected to be called, were called
			// Finish() should be invoked for each controller - checks store.EXPECT was satisfied
			defer ctrl.Finish()

			// declare a new store
			store := mockdb.NewMockStore(ctrl)
			// build stub relevant to test case
			tc.buildStubs(store)

			// start test HTTP server and send request - this is not an actual server
			// will use recorder to record the response of the api request
			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := "/accounts"
			jsonText := fmt.Sprintf("{\"id\":%d, \"balance\":%d}", tc.updateAccountInput.ID, tc.updateAccountInput.Balance)
			jsonBody := strings.NewReader(jsonText)
			// generate a new HTTP request with MethodPatch to the url
			request, err := http.NewRequest(http.MethodPatch, url, jsonBody)
			require.NoError(t, err)

			// create token and then the authentication header - update request with header
			tc.setupAuth(t, request, server.tokenMaker)

			// sends our api request through the server router and records response in the recorder
			server.router.ServeHTTP(recorder, request)

			// check response
			tc.checkResponse(t, recorder)
		})
	}
}

func TestDeleteAccountAPI(t *testing.T) {
	user, _ := randomUser(t)
	// create a random account for testing
	account := randomAccount(user.Username)

	// create test cases for testing
	testCases := []struct {
		// each test case will have a unique name
		name string
		// accountID that we want to delete
		accountID  int64
		setupAuth  func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs func(store *mockdb.MockStore)
		// check the output of the API
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		// adding scenarios
		{
			// each test case will have a unique name
			name: "OK",
			// accountID that we want to delete
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			// we expect to first get the account to see if it is there
			// we then expect DeleteAccount to run once and return nil
			buildStubs: func(store *mockdb.MockStore) {
				first := store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
				second := store.EXPECT().DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(nil)
				// ensure the Get Account is called first and the Delete Account is called second
				gomock.InOrder(first, second)
			},
			// check the output of the API
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:      "Not Found",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// if the ID is valid but doesn't exist, only GetAccount will run
				// we expect it to run once and return an empty account with the SQL error no rows
				first := store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
				second := store.EXPECT().DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).Times(0)
				gomock.InOrder(first, second)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "Internal Get Error",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// for an error with the GetAccount method, we expect it to return an empty account and an internal sever error
				// we expect DeleteAccount to not run at all
				first := store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrConnDone)
				second := store.EXPECT().DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).Times(0)
				gomock.InOrder(first, second)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "Internal Delete Error",
			accountID: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// we expect GetAccount to return the account, proving that the account exists
				// we expect DeleteAccount however, to return an internal server error
				first := store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
				second := store.EXPECT().DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(sql.ErrConnDone)
				gomock.InOrder(first, second)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "Invalid ID",
			accountID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// for any invalid ID, we do not expect GetAccount or DeleteAccount to run
				first := store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				second := store.EXPECT().DeleteAccount(gomock.Any(), gomock.Any()).Times(0)
				gomock.InOrder(first, second)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	// for loop to iterate through test cases
	for i := range testCases {
		// tc will store the current test case data
		tc := testCases[i]
		// running each case as a separate sub-test of this unit test
		t.Run(tc.name, func(t *testing.T) {
			// to call NewMockStore, we need a *gomock.Controller object - here we create one
			ctrl := gomock.NewController(t)
			// we defer calling ctrl.Finish() - Finish() checks to see if all the methods that were expected to be called, were called
			// Finish() should be invoked for each controller - checks store.EXPECT was satisfied
			defer ctrl.Finish()

			// declare a new store
			store := mockdb.NewMockStore(ctrl)
			// build stub relevant to test case using tc to call it
			tc.buildStubs(store)

			// start test HTTP server and send request - this is not an actual server
			// will use recorder to record the response of the api request
			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			// Sprintf - formats according to a format specifier and then returns the resulting string
			url := fmt.Sprintf("/accounts/%d", tc.accountID)

			// generate a new HTTP request with MethodDelete to the url
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			// create the token and then the authentication header - update request with authentication header
			tc.setupAuth(t, request, server.tokenMaker)

			// sends our api request through the server router and records response in the recorder
			server.router.ServeHTTP(recorder, request)

			// check response
			tc.checkResponse(t, recorder)
		})
	}
}

// to test get account, we need a test account to retrieve
func randomAccount(owner string) db.Account {
	return db.Account{
		// create a random ID between 1 and 1000
		ID:    util.RandomInt(1, 1000),
		Owner: owner,
		// we could use util.RandomMoney() but CreateAccount requires a balance of zero
		// and a zero balance will not impact the other methods
		Balance: 0,
		// cannot use util.RandomCurrency() since it includes CAD as an option and we only allow for USD or EUR
		// since we handle the invalid currency manually, we can hard set a valid one here.
		Currency: "USD",
	}
}

// requireBodyMatchAccount will compare the account properties against the recorder body (which is a bytes buffer)
// to ensure they match
func requireBodyMatchAccount(t *testing.T, body *bytes.Buffer, account db.Account) {
	// ioutil reads all the data from the bytes buffer
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	// declaring gotAccount variable to store the data read from the bytes buffer
	var gotAccount db.Account
	// using json.Unmarshal to unmarshal the data into the gotAccount object - must be a pointer
	err = json.Unmarshal(data, &gotAccount)
	require.NoError(t, err)
	// test that the gotAccount and the account we passed in, are equal
	require.Equal(t, account, gotAccount)
}

// requireBodyMatchAccounts will compare the properties of each account against the recorder body to ensure they match
func requireBodyMatchAccounts(t *testing.T, body *bytes.Buffer, accounts []db.Account) {
	// ioutil reads all the data from the bytes buffer
	data, err := ioutil.ReadAll(body)
	require.NoError(t, err)

	// declaring slice gotAccounts to store the data read from the bytes buffer
	// since our pageSize is five, we create a slice of length five
	gotAccounts := make([]db.Account, 5)
	err = json.Unmarshal(data, &gotAccounts)
	require.NoError(t, err)
	require.Equal(t, accounts[:5], gotAccounts)
}
