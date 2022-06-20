package db

import (
	"database/sql" // provides generic interface around SQL db
	"log"
	"os"
	"testing"

	// lib/pq provides postgres driver support
	_ "github.com/lib/pq" // the underscore is a blank identifier - it tells the Go formatter to leave this import even though we do not directly call any functions from lib/pq
)

// eventually, this will be done via pulling from environment variables and not constants
const (
	dbDriver = "postgres"
	dbSource = "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable"
)

// queries struct defined in db.go - contains DBTX variable - either a db connection or a transaction
var testQueries *Queries // global variable to test CRUD ops - you need a queries object to test the defined methods
var testDB *sql.DB       // global variable to use in testing db transactions

func TestMain(m *testing.M) {
	var err error

	testDB, err = sql.Open(dbDriver, dbSource) // sql.Open() returns a sql db object and an error
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	testQueries = New(testDB) // testQueries is the global variable defined above - New() comes from db.go

	os.Exit(m.Run()) // m.Run() returns an exit code to tell us if the tests pass or fail and we pass it to os. Exit()
}
