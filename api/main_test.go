package api

import (
	"os"
	"testing"
	"time"

	db "SimpleBankProject/db/sqlc"
	"SimpleBankProject/db/util"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store) *Server {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}

	server, err := NewServer(config, store)
	require.NoError(t, err)

	return server
}

func TestMain(m *testing.M) {
	// places gin in test mode rather than debug mode to cut down on the amount of logs written by gin
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run()) // m.Run() returns an exit code to tell us if the tests pass or fail and we pass it to os. Exit()
}
