package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/techschool/simplebank/token"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

// authMiddleware will return the actual authentication middleware function - it isn't middleware in and of itself
// it is a higher order function
func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	// anonymous function which takes in the same context input as gin.HandlerFunc
	// this anonymous function is in fact, the authentication middleware
	return func(ctx *gin.Context) {
		// to authorize user to perform request, we need to extract the authorization header from the request
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)
		// if length is zero, the authorization header is empty
		if len(authorizationHeader) == 0 {
			// create error
			err := errors.New("authorization header is not provided")
			// send status and error to the client
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		// if the authorization header is provided, it should have the prefix Bearer followed by a space
		// e.g. Bearer v2.local... - Bearer tells the server what type of authorization it is as server may support
		// multiple types of authorization schemes
		fields := strings.Fields(authorizationHeader) // strings.Fields() splits authorization header by space
		// we expect fields to have at least two elements
		if len(fields) < 2 {
			// create error
			err := errors.New("invalid authorization header format")
			// send status and error to the client
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		// authorization type should be the first element of the fields slice
		// strings.ToLower converts it to lower case - easier to compare if we know the data is all lower case
		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			// create the error
			err := fmt.Errorf("unsupported authorization type %s", authorizationType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		// access token should be the second element of the fields slice
		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken)
		// if err is not nil, something is wrong with the access token
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}
		// storing the payload in the gin context using authorizationPayloadKey
		// allows us to retrieve the payload data from the context using the same key
		ctx.Set(authorizationPayloadKey, payload)
		// ctx.Next forwards the context to the next handler
		ctx.Next()
	}
}
