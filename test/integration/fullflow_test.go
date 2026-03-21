//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	tcmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdjun99/lazy-dbx/internal/app"
	infraconfig "github.com/kdjun99/lazy-dbx/internal/infra/config"
	infraconn "github.com/kdjun99/lazy-dbx/internal/infra/connection"
	infralogger "github.com/kdjun99/lazy-dbx/internal/infra/logger"
	inframysql "github.com/kdjun99/lazy-dbx/internal/infra/mysql"
	infrapassword "github.com/kdjun99/lazy-dbx/internal/infra/password"
	infrapostgres "github.com/kdjun99/lazy-dbx/internal/infra/postgres"
	infratunnel "github.com/kdjun99/lazy-dbx/internal/infra/tunnel"
)

// startMySQL starts a MySQL testcontainer and returns host, port, and a cleanup func.
func startMySQL(t *testing.T) (host string, port int) {
	t.Helper()
	ctx := context.Background()
	container, err := tcmysql.Run(ctx, "mysql:8.0",
		tcmysql.WithDatabase("testdb"),
		tcmysql.WithUsername("testuser"),
		tcmysql.WithPassword("testpass"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	ep, err := container.Endpoint(ctx, "")
	require.NoError(t, err)

	h, p, err := net.SplitHostPort(ep)
	require.NoError(t, err)

	portNum, err := strconv.Atoi(p)
	require.NoError(t, err)

	return h, portNum
}

// startPostgres starts a PostgreSQL testcontainer and returns host, port, and a cleanup func.
func startPostgres(t *testing.T) (host string, port int) {
	t.Helper()
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:16",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	ep, err := container.Endpoint(ctx, "")
	require.NoError(t, err)

	h, p, err := net.SplitHostPort(ep)
	require.NoError(t, err)

	portNum, err := strconv.Atoi(p)
	require.NoError(t, err)

	return h, portNum
}

// buildService wires real infra components into a ConnectionService and returns it
// along with a temp config directory populated with the given connections.toml content.
func buildService(t *testing.T, connectionsToml string) (*app.ConnectionService, string) {
	t.Helper()

	dir := t.TempDir()

	err := os.WriteFile(filepath.Join(dir, "connections.toml"), []byte(connectionsToml), 0o600)
	require.NoError(t, err)

	settingsContent := `
[editor]
tab_size = 4

[safety]
confirm_dml = false
`
	err = os.WriteFile(filepath.Join(dir, "settings.toml"), []byte(settingsContent), 0o600)
	require.NoError(t, err)

	log := infralogger.NewNopLogger()

	svc := app.NewConnectionService(app.Config{
		ConfigDir:        dir,
		Loader:           infraconfig.NewTOMLLoader(),
		PasswordResolver: infrapassword.NewMultiResolver(log),
		TunnelManager:    infratunnel.NewTunnelManager(),
		MySQLConnector:   &inframysql.Connector{},
		PGConnector:      &infrapostgres.Connector{},
		Pool:             infraconn.NewInMemoryPool(),
		Logger:           log,
	})

	return svc, dir
}

func TestFullFlow_MySQL_DirectConnect(t *testing.T) {
	host, port := startMySQL(t)

	connectionsToml := fmt.Sprintf(`
[groups.test.write.mydb]
type = "mysql"
host = "%s"
port = %d
user = "testuser"
database = "testdb"
password = "testpass"
env = "test"
connect_timeout = "30s"
`, host, port)

	svc, _ := buildService(t, connectionsToml)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Connect
	connectResult := svc.Connect(ctx, "test.write.mydb")
	require.NoError(t, connectResult.Error)
	assert.Equal(t, "test.write.mydb", connectResult.Data.Path)
	assert.Equal(t, "test", connectResult.Data.Group)
	assert.Equal(t, "write", connectResult.Data.Subgroup)
	assert.Equal(t, "mydb", connectResult.Data.Name)
	assert.Equal(t, "mysql", connectResult.Data.Type)
	assert.Equal(t, host, connectResult.Data.Host)
	assert.Equal(t, "connected", connectResult.Data.Status)

	// Ping
	pingResult := svc.Ping(ctx, "test.write.mydb")
	require.NoError(t, pingResult.Error)
	assert.Equal(t, "test.write.mydb", pingResult.Data.Path)

	// Disconnect
	disconnectResult := svc.Disconnect(ctx, "test.write.mydb")
	require.NoError(t, disconnectResult.Error)
}

func TestFullFlow_Postgres_DirectConnect(t *testing.T) {
	host, port := startPostgres(t)

	connectionsToml := fmt.Sprintf(`
[groups.prod.read.pgdb]
type = "postgresql"
host = "%s"
port = %d
user = "testuser"
database = "testdb"
password = "testpass"
env = "test"
connect_timeout = "30s"
`, host, port)

	svc, _ := buildService(t, connectionsToml)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Connect
	connectResult := svc.Connect(ctx, "prod.read.pgdb")
	require.NoError(t, connectResult.Error)
	assert.Equal(t, "prod.read.pgdb", connectResult.Data.Path)
	assert.Equal(t, "prod", connectResult.Data.Group)
	assert.Equal(t, "read", connectResult.Data.Subgroup)
	assert.Equal(t, "pgdb", connectResult.Data.Name)
	assert.Equal(t, "postgresql", connectResult.Data.Type)
	assert.Equal(t, host, connectResult.Data.Host)
	assert.Equal(t, "connected", connectResult.Data.Status)

	// Ping
	pingResult := svc.Ping(ctx, "prod.read.pgdb")
	require.NoError(t, pingResult.Error)
	assert.Equal(t, "prod.read.pgdb", pingResult.Data.Path)

	// Disconnect
	disconnectResult := svc.Disconnect(ctx, "prod.read.pgdb")
	require.NoError(t, disconnectResult.Error)
}

func TestFullFlow_ValidateConfig(t *testing.T) {
	connectionsToml := `
[groups.staging.write.appdb]
type = "mysql"
host = "db.staging.internal"
port = 3306
user = "app"
database = "appdb"
password = "secret"
env = "staging"

[groups.staging.read.replica]
type = "postgresql"
host = "replica.staging.internal"
port = 5432
user = "readonly"
database = "appdb"
password = "readpass"
env = "staging"
`

	svc, _ := buildService(t, connectionsToml)

	ctx := context.Background()

	result := svc.ValidateConfig(ctx)
	require.NoError(t, result.Error)
	assert.True(t, result.Data.Valid)
	assert.Equal(t, 2, result.Data.ConnCount)
	assert.Empty(t, result.Data.Errors)
}

func TestFullFlow_ListConnections(t *testing.T) {
	connectionsToml := `
[groups.prod.write.primary]
type = "mysql"
host = "primary.prod.internal"
port = 3306
user = "admin"
database = "maindb"
password = "pw1"
env = "production"

[groups.prod.read.replica]
type = "mysql"
host = "replica.prod.internal"
port = 3306
user = "readonly"
database = "maindb"
password = "pw2"
env = "production"

[groups.dev.write.local]
type = "postgresql"
host = "localhost"
port = 5432
user = "dev"
database = "devdb"
password = "devpass"
env = "development"
`

	svc, _ := buildService(t, connectionsToml)

	ctx := context.Background()

	result := svc.ListConnections(ctx)
	require.NoError(t, result.Error)
	require.Len(t, result.Data, 3)

	// Index by path for order-independent assertions
	byPath := make(map[string]struct{})
	for _, info := range result.Data {
		byPath[info.Path] = struct{}{}
	}

	require.Contains(t, byPath, "prod.write.primary")
	require.Contains(t, byPath, "prod.read.replica")
	require.Contains(t, byPath, "dev.write.local")

	// Verify individual entries via direct range assertions
	for _, info := range result.Data {
		switch info.Path {
		case "prod.write.primary":
			assert.Equal(t, "prod", info.Group)
			assert.Equal(t, "write", info.Subgroup)
			assert.Equal(t, "primary", info.Name)
			assert.Equal(t, "mysql", info.Type)
			assert.Equal(t, "primary.prod.internal", info.Host)
			assert.Equal(t, "configured", info.Status)
		case "prod.read.replica":
			assert.Equal(t, "prod", info.Group)
			assert.Equal(t, "read", info.Subgroup)
			assert.Equal(t, "replica", info.Name)
			assert.Equal(t, "mysql", info.Type)
			assert.Equal(t, "replica.prod.internal", info.Host)
			assert.Equal(t, "configured", info.Status)
		case "dev.write.local":
			assert.Equal(t, "dev", info.Group)
			assert.Equal(t, "write", info.Subgroup)
			assert.Equal(t, "local", info.Name)
			assert.Equal(t, "postgresql", info.Type)
			assert.Equal(t, "localhost", info.Host)
			assert.Equal(t, "configured", info.Status)
		default:
			t.Errorf("unexpected connection path: %s", info.Path)
		}
	}
}
