package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"

	"SimpleBankProject/api"
	db "SimpleBankProject/db/sqlc"
	"SimpleBankProject/db/util"
	_ "SimpleBankProject/doc/statik"
	"SimpleBankProject/gapi"
	"SimpleBankProject/pb"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/lib/pq" // without, code cannot talk to the database
	"github.com/rakyll/statik/fs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	// loading config from config file (provides DBDriver, DBSource, etc.)
	config, err := util.LoadConfig(".") // the dot means the path is the current folder - app.env is in the same folder as main.go
	if err != nil {
		log.Fatal("cannot load config:", err)
	}
	// to create a server, we first need to connect to the database and create a store
	// connect to the database
	conn, err := sql.Open(config.DBDriver, config.DBSource) // sql.Open() returns a sql db object and an error
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	// create store
	store := db.NewStore(conn)
	// uncomment runGinServer(config, store) if working with standard HTTP API
	// runGinServer(config, store)

	// we need to run the gRPC server and the gateway server in two different go routines
	// otherwise they will block each other
	go runGatewayServer(config, store)
	// start the gRPC server
	runGrpcServer(config, store)

}

func runGrpcServer(config util.Config, store db.Store) {
	// create our implementation of the Simple Bank server
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	// create a new gRPC server from auto-generated code - has no services registered
	grpcServer := grpc.NewServer()

	// register the new gRPC server
	pb.RegisterSimpleBankServer(grpcServer, server)
	// register a reflection for the gRPC server
	// allows the gRPC client to explore what RPCs are available on the server and how to call them
	reflection.Register(grpcServer)

	// create a listener to listen for traffic for the gRPC Server Address
	listener, err := net.Listen("tcp", config.GRPCServerAddress)
	if err != nil {
		log.Fatal("cannot create listener:", err)
	}

	log.Printf("start gRPC server at %s", listener.Addr().String())
	// start gRPC server
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal("cannot start gRPC server:", err)
	}
}

// setup gRPC gateway server using in-process translation method (limited to unary gRPC)
func runGatewayServer(config util.Config, store db.Store) {
	// create our implementation of the Simple Bank server
	server, err := gapi.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	// optional - the protocol buffer compiler generates camelCase JSON tags by default
	// here we make the response output match the case (camel case, etc.) of the properities defined in the proto files
	jsonOption := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})

	// create a ServeMux object whose internal mapping is empty
	grpcMux := runtime.NewServeMux(jsonOption)

	// create a context to pass to pb.RegisterSimpleBankHandlerServer
	// context.WithCancel(), creates a context using the background context and a cancel function to cancel the context
	ctx, cancel := context.WithCancel(context.Background())
	// defer cancel function until runGatewayServer() exits - canceling a context prevents unnecessary work
	defer cancel()

	// pb.RegisterSimpleBankHandlerServer registers HTTP handlers to the mux (grpcMux)
	err = pb.RegisterSimpleBankHandlerServer(ctx, grpcMux, server)
	if err != nil {
		log.Fatal("cannot register handler server:", err)
	}

	// create a HTTP serve mux - receives HTTP requests from clients
	mux := http.NewServeMux()

	// to convert the HTTP requests to gRPC format, the HTTP requests must be routed to the gRPC mux (grpcMux)
	// Handle() registers the handler for the given pattern (e.g "/" which covers all HTTP server routes and therefore
	// all handlers) - in other words, the HTTP serve mux now points to the grpcMux handlers
	mux.Handle("/", grpcMux)

	// optional - using Swagger UI in order to visually document our API
	// create file server
	// fileServer := http.FileServer(http.Dir("./doc/swagger"))

	// optional - using Statik to embed the front end files (doc/swagger) into the backend binary
	// fs is a subpackage of Statik
	statikFS, err := fs.New()
	if err != nil {
		log.Fatal("cannot create statik file server:", err)
	}

	// add a HTTP handler for the file server
	// documentation files should be served under the route that starts with /swagger/ - we need to remove this prefix from
	// the URL however prior to passing the request to the file server
	swaggerHandler := http.StripPrefix("/swagger/", http.FileServer(statikFS))
	mux.Handle("/swagger/", swaggerHandler)

	// create a listener to listen for traffic for the HTTP Server Address
	listener, err := net.Listen("tcp", config.HTTPServerAddress)
	if err != nil {
		log.Fatal("cannot create listener:", err)
	}

	log.Printf("start HTTP gateway server at %s", listener.Addr().String())
	// start HTTP server and pass in the listener and the HTTP mux object
	err = http.Serve(listener, mux)
	if err != nil {
		log.Fatal("cannot start HTTP server:", err)
	}
}

func runGinServer(config util.Config, store db.Store) {
	// create server
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	// start the server passing HTTPServerAddress 
	err = server.Start(config.HTTPServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}
