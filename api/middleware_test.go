package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/techschool/simplebank/token"
)

// addAuthorization creates an access token and adds it to the authorization header
func addAuthorization(
	t *testing.T,
	request *http.Request,
	tokenMaker token.Maker,
	authorizationType string,
	username string,
	duration time.Duration,
) {
	// create token
	token, err := tokenMaker.CreateToken(username, duration)
	require.NoError(t, err)
	// create authorization header - remember, it should be two strings separated by a space
	// first the authorizationType (bearer) and the token itself
	authorizationHeader := fmt.Sprintf("%s %s", authorizationType, token)
	// set the header of the request
	request.Header.Set(authorizationHeaderKey, authorizationHeader)
}

func TestAuthMiddleware(t *testing.T) {
	// generate test cases
	testCases := []struct {
		name string
		// setupAuth function sets up the authorization header of the request
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// create an access token and add it to the authorization header
				// we use user as the username and give the token a duration of one minute
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, "user", time.Minute)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "No Authorization",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// in this case, the client doesn't provide an authorization header

			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Unsupported Authorization",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// create an access token and add it to the authorization header
				// we use user as the username and give the token a duration of one minute
				addAuthorization(t, request, tokenMaker, "unsupported", "user", time.Minute)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Invalid Authorization Format",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// create an access token and add it to the authorization header
				// we use user as the username and give the token a duration of one minute
				addAuthorization(t, request, tokenMaker, "", "user", time.Minute)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Expired Token",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// create an access token and add it to the authorization header
				// we use user as the username and give the token a duration of one minute
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, "user", -time.Minute)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for i := range testCases {
		// current test case
		tc := testCases[i]

		// run as a subtest
		t.Run(tc.name, func(t *testing.T) {
			// create a test server, since we are testing middleware, we do not need to access a store object (thus we pass nil)
			server := newTestServer(t, nil)

			// adding a simple route for the sake of testing only
			authPath := "/auth"
			server.router.GET(authPath,
				authMiddleware(server.tokenMaker),
				// for testing purposes, we write a simple handler
				func(ctx *gin.Context) {
					// for testing purposes, we simply return Status OK 200
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)
			// setup HTTP test recorder
			recorder := httptest.NewRecorder()
			// setup HTTP request with nil body
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)

			// need to add authentication header to the request
			tc.setupAuth(t, request, server.tokenMaker)
			// send HTTP request
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
