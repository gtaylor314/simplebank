package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	db "github.com/techschool/simplebank/db/sqlc"
	"github.com/techschool/simplebank/db/util"
	"github.com/techschool/simplebank/token"
)

// Implementing our HTTP API server

// Define server struct - serves HTTP requests for banking service
type Server struct {
	config     util.Config
	store      db.Store // Package db, Store interface - defined in store.go - for interacting with the db while processing api requests
	tokenMaker token.Maker
	router     *gin.Engine // Router helps send each api request to the correct handler
}

// NewServer creates a new HTTP server and sets up routing
func NewServer(config util.Config, store db.Store) (*Server, error) {
	// initialize tokenMaker, symmetric key will come from the environment variable
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}
	// Server struct, store property, initialized to store which we pass in
	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}

	// registering custom validator with gin
	// first obtain the current validator engine that gin is using
	// since Engine() returns a general interface type which, by default, is a pointer to the validator object of
	// the go-playground validator package, we need to convert to a pointer to validator.Validate type
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// registering the custom validator
		// currency is the name of the validation tag
		// validCurrency is the method in transfer.go
		v.RegisterValidation("currency", validCurrency)
	}
	//setup routes
	server.setupRouter()

	return server, nil
}

// Start runs the HTTP server on the input address to start listening for API requests
func (server *Server) Start(address string) error {
	// router is of type gin.Default(), gin provides the Run function
	// router field is private and cannot be accessed from outside of this package
	// this is one of the reasons we create the public Start function
	// this method is called on a variable of type *Server which means it returns that-variable.router.Run(address)
	// the use of server in server.router.Run(address) is not to be confused with the variable returned in NewServer
	return server.router.Run(address)
}

func (server *Server) setupRouter() {
	router := gin.Default()
	// adding routes to router
	// grouping routes that require the authMiddleware for authorization
	// the "/" is the path prefix for all routes in this group
	authRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker))

	// creating account
	// "/accounts" is the path, can pass 1+ handler functions
	// if you do, last function should be "real handler" and all other functions are middleware
	authRoutes.POST("/accounts", server.createAccount) // createAccount - method of the Server struct - handler
	// get account by id
	// "/accounts/:id" path to account with ID - the colon tells Gin that the ID is a URI parameter
	// URI (Unique Resource Identifier) is a resource identifier passed as a parameter in the URL
	authRoutes.GET("/accounts/:id", server.getAccount) // getAccount - method of the Server struct - handler
	// get a list of accounts with pagination
	// the path is left as /accounts since the query parameters will be obtained from the query itself
	authRoutes.GET("/accounts", server.listAccount) // listAccount - method of the Server struct - handler
	// update an account's balance
	// "/accounts" path to accounts
	authRoutes.PATCH("/accounts", server.updateAccount) // updateAccount - method of the Server struct - handler
	// delete an account
	// "/accounts/:id" path to account with ID - the colon tells Gin that the ID is a URI parameter
	authRoutes.DELETE("/accounts/:id", server.deleteAccount) //deleteAccount - method of the Server struct - handler
	// transfer money from FromAccountID to ToAccountID
	// "/transfers" path to the transfers table
	authRoutes.POST("/transfers", server.createTransfer) // createTransfer - method of the Server struct - handler

	// no authorization required:
	// create user account
	// "/users" path to the users table
	// no authorization needed as everyone should be able to create a user
	router.POST("/users", server.createUser) // createUser - method of the Server struct - handler
	// user login
	// "/users/login" path for login api
	// no authorization needed as everyone should be able to login
	router.POST("/users/login", server.loginUser) // loginUser - method of the Server struct - handler

	// update server.router with router object
	server.router = router
}

// errorResponse - converts error into a key-value object for JSON
// gin.H - shortcut to map[string]any - allowing for the creation of any key-value data
func errorResponse(err error) gin.H {
	// temporary - return map with one key ("error") and value (the error itself)
	return gin.H{"error": err.Error()}
}
