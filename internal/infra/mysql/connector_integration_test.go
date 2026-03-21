//go:build integration

package mysql_test

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdjun99/lazy-dbx/internal/domain/config"
	"github.com/kdjun99/lazy-dbx/internal/domain/connection"
	"github.com/kdjun99/lazy-dbx/internal/infra/mysql"
)

func startMySQL(t *testing.T) (host string, port int, cleanup func()) {
	t.Helper()
	ctx := context.Background()
	container, err := tcmysql.Run(ctx, "mysql:8.0",
		tcmysql.WithDatabase("testdb"),
		tcmysql.WithUsername("testuser"),
		tcmysql.WithPassword("testpass"),
	)
	require.NoError(t, err)

	cleanup = func() {
		_ = container.Terminate(ctx)
	}
	t.Cleanup(cleanup)

	ep, err := container.Endpoint(ctx, "")
	require.NoError(t, err)

	h, p, err := net.SplitHostPort(ep)
	require.NoError(t, err)

	portNum, err := strconv.Atoi(p)
	require.NoError(t, err)

	return h, portNum, cleanup
}

func TestMySQLConnector_Connect_Integration(t *testing.T) {
	host, port, _ := startMySQL(t)

	entry := config.ConnectionEntry{
		Type:           "mysql",
		Host:           host,
		Port:           port,
		User:           "testuser",
		Database:       "testdb",
		ConnectTimeout: "30s",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := &mysql.Connector{}
	db, err := c.Connect(ctx, entry, "testpass", nil)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	err = db.PingContext(ctx)
	assert.NoError(t, err)

	row := db.QueryRowContext(ctx, "SELECT 1")
	var val int
	err = row.Scan(&val)
	assert.NoError(t, err)
	assert.Equal(t, 1, val)
}

func TestMySQLConnector_Connect_WrongPassword(t *testing.T) {
	host, port, _ := startMySQL(t)

	entry := config.ConnectionEntry{
		Type:           "mysql",
		Host:           host,
		Port:           port,
		User:           "testuser",
		Database:       "testdb",
		ConnectTimeout: "5s",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := &mysql.Connector{}
	db, err := c.Connect(ctx, entry, "wrongpassword", nil)
	if db != nil {
		db.Close()
	}

	require.Error(t, err)
	assert.ErrorIs(t, err, connection.ErrConnectionFailed)
}

func TestMySQLConnector_PoolConfig_Integration(t *testing.T) {
	host, port, _ := startMySQL(t)

	entry := config.ConnectionEntry{
		Type:            "mysql",
		Host:            host,
		Port:            port,
		User:            "testuser",
		Database:        "testdb",
		ConnectTimeout:  "30s",
		MaxOpenConns:    10,
		MaxIdleConns:    3,
		ConnMaxLifetime: "5m",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := &mysql.Connector{}
	db, err := c.Connect(ctx, entry, "testpass", nil)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	assert.Equal(t, 10, db.Stats().MaxOpenConnections)
}
