package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type renewAccessTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type renewAccessTokenResponse struct {
	AccessToken          string    `json:"access_token"`
	AccessTokenExpiresAt time.Time `json:"access_token_expires_at"`
}

// renewAccessToken api handler
func (server *Server) renewAccessToken(ctx *gin.Context) {
	var req renewAccessTokenRequest
	// ShouldBindJSON will bind the data from the JSON body to the renewAccessTokenRequest object (req)
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// verify the refresh token is still valid
	refreshPayload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// if refresh token is still valid, get session
	session, err := server.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		// two reasons err may not be nil
		// first, the session doesn't exist
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		// second, internal issue with the GetSession api call
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// if refresh token is valid and session exists, check if the refresh token is blocked
	if session.IsBlocked {
		err := fmt.Errorf("blocked session")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// if refresh token is valid, session exists, and the token is not blocked, does the session username match the
	// refresh token username
	if session.Username != refreshPayload.Username {
		err := fmt.Errorf("incorrect session user")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// if refresh token is valid, session exists, the token isn't blocked, and session username matches the refresh token
	// username, does the session refresh token match the refresh token in the request
	if session.RefreshToken != req.RefreshToken {
		err := fmt.Errorf("mismatched session token")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// reconfirming the session isn't expired - in rare cases, we may want to force the session to expire early
	// checking if the current time is after the session.ExpiresAt value
	if time.Now().After(session.ExpiresAt) {
		err := fmt.Errorf("expired session")
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// create new access token
	accessToken, accessPayload, err := server.tokenMaker.CreateToken(refreshPayload.Username, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// create renewAccessTokenResponse
	rsp := renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}
	// send renewAccessTokenResponse to the client with 200 Status OK code
	ctx.JSON(http.StatusOK, rsp)
}
