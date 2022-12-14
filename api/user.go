package api

import (
	"database/sql"
	"net/http"
	"time"

	db "SimpleBankProject/db/sqlc"
	"SimpleBankProject/db/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"` // alphanum means it must be alphanumeric chars only
	Password string `json:"password" binding:"required,min=6"`    // min=6 means the password must be at least 6 chars
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"` // email validates that the string is a valid email
}

// by returning the user, we return the hashed password, creating a new struct to return in place of the user
// this allows us to remove sensitive data like the hashed password
type userResponse struct {
	Username         string    `json:"username"`
	FullName         string    `json:"full_name"`
	Email            string    `json:"email"`
	PasswordChangeAt time.Time `json:"password_change_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// newUserResponse will convert a user object, which contains the hashed password, to a userResponse object which
// doesn't have the hashed password
func newUserResponse(user db.User) userResponse {
	return userResponse{
		Username:         user.Username,
		FullName:         user.FullName,
		Email:            user.Email,
		PasswordChangeAt: user.PasswordChangeAt,
		CreatedAt:        user.CreatedAt,
	}
}

// createUser takes in gin.Context because it is a handler - the handler function is defined to take gin.Context
func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest

	// ShouldBindJSON will parse the input data from HTTP request body - "bind request body into a type"
	// Gin then validates the output object internally to confirm the binding tags are satisfied
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// if err is NOT nil, the customer has entered invalid data
		// first argument is a HTTP status code, the next is a JSON object that gets sent to the customer
		// to send the error, we need to convert it to a key-value object - Gin will serialize this to JSON and return to customer
		ctx.JSON(http.StatusBadRequest, errorResponse(err))

		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// if no err, begin user creation
	arg := db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashedPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	}

	user, err := server.store.CreateUser(ctx, arg)
	if err != nil {
		// try to convert err to type pq.Error
		// this is to provide a better error in the event someone attempts to create a user with a username or email that
		// already exists
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusForbidden, errorResponse(err))
				return
			}
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// create a response to return instead of the user which contains the hashed password
	rsp := newUserResponse(user)

	// if no error, send a 200 OK status code
	ctx.JSON(http.StatusOK, rsp)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"` // alphanum means it must be alphanumeric chars only
	Password string `json:"password" binding:"required,min=6"`    // min=6 means the password must be at least 6 chars
}

type loginUserResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}

// loginUser api handler
func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	// ShouldBindJSON will bind the data from the JSON body to the loginUserRequest object (req)
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// get user requested if it exists
	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		// two reasons err may not be nil
		// first, the user doesn't exist
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		// second, internal issue with the GetUser api call
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// check if the password provided is correct
	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		// if err isn't nil, the password provided was incorrect
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	// user exists and password provided is correct, create access token
	accessToken, accessPayload, err := server.tokenMaker.CreateToken(user.Username, server.config.AccessTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// create refresh token with a longer valid duration than the access token - will use to create session
	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(user.Username, server.config.RefreshTokenDuration)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// create session
	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(), // client type
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// create loginUserResponse
	rsp := loginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  newUserResponse(user),
	}
	// send loginUserResponse to the client with 200 Status OK code
	ctx.JSON(http.StatusOK, rsp)
}
