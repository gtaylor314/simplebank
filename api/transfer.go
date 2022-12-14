package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	db "SimpleBankProject/db/sqlc"
	"SimpleBankProject/token"

	"github.com/gin-gonic/gin"
)

type transferRequest struct {
	// example of binding tags
	FromAccountID int64 `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64 `json:"to_account_id" binding:"required,min=1"`
	// we are using int for simplicity but this could be a float e.g. $1.50
	Amount   int64  `json:"amount" binding:"required,gt=0"`       // gt=0 means greater than 0 - to allow for changes to float in the future
	Currency string `json:"currency" binding:"required,currency"` // we will need to validate both accounts use the same currency
}

// createAccount takes in gin.Context because it is a handler - the handler function is defined to take gin.Context
func (server *Server) createTransfer(ctx *gin.Context) {
	var req transferRequest

	// ShouldBindJSON will parse the input data from HTTP request body - "bind request body into a type"
	// Gin then validates the output object internally to confirm the binding tags are satisfied
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// if err is NOT nil, the customer has entered invalid data
		// first argument is a HTTP status code, the next is a JSON object that gets sent to the customer
		// to send the error, we need to convert it to a key-value object - Gin will serialize this to JSON and return to customer
		ctx.JSON(http.StatusBadRequest, errorResponse(err))

		return
	}

	// check if FromAccountID has the correct currency for the transfer
	// we also want fromAccount to ensure that the logged in user is really the owner of fromAccount
	// only the owner can transfer money from their account
	fromAccount, valid := server.validAccount(ctx, req.FromAccountID, req.Currency)
	if !valid {
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if fromAccount.Owner != authPayload.Username {
		err := errors.New("from account does not belong to authenticated user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// check if ToAccountID has the correct currency for the transfer
	// however we do not care who owns the toAccount
	_, valid = server.validAccount(ctx, req.ToAccountID, req.Currency)
	if !valid {
		return
	}

	// if no err, create account
	arg := db.TransferTxParams{
		FromAccountID: req.FromAccountID,
		ToAccountID:   req.ToAccountID,
		Amount:        req.Amount,
	}

	result, err := server.store.TransferTX(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// if no error, send a 200 OK status code and the created account object to the customer
	ctx.JSON(http.StatusOK, result)
}

// validAccount confirms the account exists and that its currency matches the input currency
func (server *Server) validAccount(ctx *gin.Context, accountID int64, currency string) (db.Account, bool) {
	// get account to confirm the account exists
	account, err := server.store.GetAccount(ctx, accountID)
	if err != nil {
		// account doesn't exist
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return account, false
		}
		// internal error
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return account, false
	}

	if account.Currency != currency {
		err = fmt.Errorf("account [%d] currency mismatch: %s vs %s", accountID, account.Currency, currency)
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return account, false
	}
	return account, true
}
