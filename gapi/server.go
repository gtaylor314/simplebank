package gapi

import (
	"fmt"

	db "SimpleBankProject/db/sqlc"
	"SimpleBankProject/db/util"
	"SimpleBankProject/pb"
	"SimpleBankProject/token"
)

// Define server struct - serves gRPC requests for banking service
type Server struct {
	// pb.UnimplementedSimpleBankServer must be embeded per SimpleBankServer interface comments
	// in service_simple_bank_grpc.pb.go - enables forward compatibility in that, the Server object can accept calls to
	// CreateUser and LoginUser even before implementing them - simply gives an unimplemented error
	pb.UnimplementedSimpleBankServer
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
}

// NewServer creates a new gRPC server - Server object must implement CreateUser and LoginUser to implement
// the SimpleBankServer interface
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

	return server, nil
}
