package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq" // without, code cannot talk to the database
	"github.com/techschool/simplebank/api"
	db "github.com/techschool/simplebank/db/sqlc"
	"github.com/techschool/simplebank/db/util"
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
	// create server
	server, err := api.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server:", err)
	}

	// start the server passing ServerAddress (a constant for now)
	err = server.Start(config.ServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}
