package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	db "github.com/techschool/simplebank/db/sqlc"
	"github.com/techschool/simplebank/token"
)

// owner and currency will be specified by customer
// input parameters will come from the body of the HTTP request which is a JSON object
// gin provides internal validation of inputs - binding:"required" means the field is required
// the comma separates multiple binding conditions like required and oneof and currency
// currency is a custom validator that was registered with gin in server.go
type createAccountRequest struct {
	Currency string `json:"currency" binding:"required,currency"` // binding tags
}

// createAccount takes in gin.Context because it is a handler - the handler function is defined to take gin.Context
func (server *Server) createAccount(ctx *gin.Context) {
	var req createAccountRequest

	// ShouldBindJSON will parse the input data from HTTP request body - "bind request body into a type"
	// Gin then validates the output object internally to confirm the binding tags are satisfied
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// if err is NOT nil, the customer has entered invalid data
		// first argument is a HTTP status code, the next is a JSON object that gets sent to the customer
		// to send the error, we need to convert it to a key-value object - Gin will serialize this to JSON and return to customer
		ctx.JSON(http.StatusBadRequest, errorResponse(err))

		return
	}

	// the owner is only allowed to be the logged in user - this info is in the payload of the access token
	// the middleware will forward this information to the handler via context using ctx.Next()
	// MustGet returns a general interface so we cast it to be an object of type *token.Payload
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// if no err, create account
	arg := db.CreateAccountParams{
		Owner:    authPayload.Username,
		Currency: req.Currency,
		Balance:  0,
	}

	account, err := server.store.CreateAccount(ctx, arg)
	if err != nil {
		// try to convert err to type pq.Error
		// this is to provide a better error in the event someone attempts to create an account without a user or a second
		// account with a duplicate currency (users can only have one account per currency)
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "foreign_key_violation", "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// if no error, send a 200 OK status code and the created account object to the customer
	ctx.JSON(http.StatusOK, account)
}

type getAccountRequest struct {
	// cannot be obtained from the request body - must use uri:"id" to inform Gin of the URI parameter
	// min = 1 means the id can be no less than one (must be a positive number and not zero)
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getAccount(ctx *gin.Context) {
	var req getAccountRequest
	// since ID is a URI, we use ShouldBindUri to bind the data to the struct
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccount(ctx, req.ID)
	if err != nil {
		// there are two possible causes for error
		// there is no account with that id
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		// an internal error with querying data from the database
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// we can only return the account data if the account owner matches the logged in user
	// middleware passes the payload information to the handler via context using ctx.Next()
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	// if the account owner doesn't match the logged in user, we do not return the account
	if account.Owner != authPayload.Username {
		// create error
		err := errors.New("account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// if there are no errors
	ctx.JSON(http.StatusOK, account)
}

type listAccountRequest struct {
	// since we are using query parameters, we use form: "page_id"
	PageID int32 `form:"page_id" binding:"required,min=1"` // no spaces unless shown here
	// we use min and max to ensure the page size is neither too big or too small
	// the page size is the number of accounts per page
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"` //no spaces unless shown here
}

func (server *Server) listAccount(ctx *gin.Context) {
	var req listAccountRequest

	// since page id and page size are query parameters, we use ShouldBindQuery
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// the logged in user is only allowed to see their accounts - the username is in the payload of the access token
	// the middleware will forward this information to the handler via context using ctx.Next()
	// MustGet returns a general interface so we cast it to be an object of type *token.Payload
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	// Server.Store.ListAccounts requires passing ListAccountsParams
	arg := db.ListAccountsParams{
		Owner: authPayload.Username,
		Limit: req.PageSize,
		// what page your on times the number of entries on a page equals where to begin for the next set of accounts
		Offset: (req.PageID - 1) * req.PageSize,
	}

	accounts, err := server.store.ListAccounts(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
	}

	ctx.JSON(http.StatusOK, accounts)
}

type updateAccountRequest struct {
	// ID must be provided and have a value no less than 1
	// remember the capitalization exports the property
	ID int64 `json:"id" binding:"required,min=1"`
	// Balance must be provided and have a value no less than 0
	Balance int64 `json:"balance" binding:"required,min=0"`
}

func (server *Server) updateAccount(ctx *gin.Context) {
	var req updateAccountRequest

	// if err is not nil, something with the request is incorrect
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// server.store.UpdateAccount requires passing UpdateAccountParams
	arg := db.UpdateAccountParams{
		ID:      req.ID,
		Balance: req.Balance,
	}

	account, err := server.store.GetAccount(ctx, arg.ID)
	if err != nil {
		// two possible reasons
		// the id provided doesn't exist
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		// something failed internally - perhaps with the GetAccount method
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// the logged in user is only allowed to update an account they own - the username is in the payload of the access token
	// the middleware will forward this information to the handler via context using ctx.Next()
	// MustGet returns a general interface so we cast it to be an object of type *token.Payload
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	// if the account owner doesn't match the username of the access token, the account cannot be updated by the logged in user
	if account.Owner != authPayload.Username {
		// create error
		err := errors.New("account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	account, err = server.store.UpdateAccount(ctx, arg)
	if err != nil {
		// something failed internally - perhaps with the UpdateAccount method
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, account)

}

type deleteAccountRequest struct {
	// uri:"id" informs Gin that the ID is a URI parameter
	// the ID is required and must be no less than 1
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) deleteAccount(ctx *gin.Context) {
	var req deleteAccountRequest

	// if err is NOT nil, then the request is incorrect
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// test if the account actually exists
	// without this, deleteAccount returns StatusOK when deleting accounts that do not exist
	account, err := server.store.GetAccount(ctx, req.ID)
	if err != nil {
		// if no rows were found with the provided ID
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		// if an internal issue occurred with GetAccount
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// the logged in user is only allowed to delete an account they own - the username is in the payload of the access token
	// the middleware will forward this information to the handler via context using ctx.Next()
	// MustGet returns a general interface so we cast it to be an object of type *token.Payload
	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	// if the account owner doesn't match the username of the access token, the account cannot be updated by the logged in user
	if account.Owner != authPayload.Username {
		// create error
		err := errors.New("account doesn't belong to the authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	err = server.store.DeleteAccount(ctx, req.ID)
	if err != nil {
		// there is an internal issue, perhaps with the DeleteAccount method
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, req)
}
