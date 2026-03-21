package app

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdjun99/lazy-dbx/internal/domain"
	domainconfig "github.com/kdjun99/lazy-dbx/internal/domain/config"
	domainconn "github.com/kdjun99/lazy-dbx/internal/domain/connection"
	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
	domainpw "github.com/kdjun99/lazy-dbx/internal/domain/password"
	domaintunnel "github.com/kdjun99/lazy-dbx/internal/domain/tunnel"
)

// --- mock implementations ---

type mockConfigLoader struct {
	connectionsResult domain.Result[*domainconfig.ConnectionsConfig]
	settingsResult    domain.Result[*domainconfig.SettingsConfig]
}

func (m *mockConfigLoader) LoadConnections(_ string) (domain.Result[*domainconfig.ConnectionsConfig], error) {
	return m.connectionsResult, nil
}

func (m *mockConfigLoader) LoadSettings(_ string) (domain.Result[*domainconfig.SettingsConfig], error) {
	return m.settingsResult, nil
}

type mockPasswordResolver struct {
	password string
	err      error
}

func (m *mockPasswordResolver) Resolve(_ context.Context, _ domainpw.Config) (string, error) {
	return m.password, m.err
}

type mockTunnelManager struct {
	state    *domaintunnel.State
	err      error
	closed   bool
	released []string
}

func (m *mockTunnelManager) GetOrOpen(_ context.Context, tunnelName string, _ domaintunnel.Config, _ string, _ int) (*domaintunnel.State, error) {
	return m.state, m.err
}

func (m *mockTunnelManager) Release(tunnelName string) error {
	m.released = append(m.released, tunnelName)
	return nil
}

func (m *mockTunnelManager) CloseAll() error {
	m.closed = true
	return nil
}

type mockConnector struct {
	db  *sql.DB
	err error
}

func (m *mockConnector) Connect(_ context.Context, _ domainconfig.ConnectionEntry, _ string, _ *domaintunnel.State) (*sql.DB, error) {
	return m.db, m.err
}

type mockPool struct {
	dbs    map[string]*sql.DB
	pinged []string
	closed []string
}

func newMockPool() *mockPool {
	return &mockPool{dbs: make(map[string]*sql.DB)}
}

func (m *mockPool) Get(path string) (*sql.DB, error) {
	db, ok := m.dbs[path]
	if !ok {
		return nil, domainconn.ErrConnectionNotInPool
	}
	return db, nil
}

func (m *mockPool) Put(path string, db *sql.DB) error {
	m.dbs[path] = db
	return nil
}

func (m *mockPool) Close(path string) error {
	m.closed = append(m.closed, path)
	delete(m.dbs, path)
	return nil
}

func (m *mockPool) CloseAll() error {
	m.dbs = make(map[string]*sql.DB)
	return nil
}

func (m *mockPool) Ping(_ context.Context, path string) error {
	m.pinged = append(m.pinged, path)
	if _, ok := m.dbs[path]; !ok {
		return domainconn.ErrConnectionNotInPool
	}
	return nil
}

type mockLogger struct{}

func (l *mockLogger) Debug(_ context.Context, _, _, _ string, _ ...domainlogger.Field) {}
func (l *mockLogger) Info(_ context.Context, _, _, _ string, _ ...domainlogger.Field)  {}
func (l *mockLogger) Warn(_ context.Context, _, _, _ string, _ ...domainlogger.Field)  {}
func (l *mockLogger) Error(_ context.Context, _, _, _ string, _ ...domainlogger.Field) {}

// --- helpers ---

func makeSimpleConfig() *domainconfig.ConnectionsConfig {
	return &domainconfig.ConnectionsConfig{
		Groups: map[string]*domainconfig.Group{
			"mygroup": {
				Subgroups: map[string]*domainconfig.Subgroup{
					"write": {
						Connections: map[string]domainconfig.ConnectionEntry{
							"main-db": {
								Name:     "main-db",
								Type:     "mysql",
								Host:     "db.example.com",
								Port:     3306,
								User:     "admin",
								Database: "mydb",
								Password: "secret",
							},
						},
					},
				},
			},
		},
		SSHTunnels: map[string]domainconfig.SSHTunnelEntry{},
	}
}

func makeService(loader *mockConfigLoader, pwResolver *mockPasswordResolver, tunnelMgr *mockTunnelManager, mysqlConnector *mockConnector, pgConnector *mockConnector, pool *mockPool) *ConnectionService {
	return NewConnectionService(Config{
		ConfigDir:        "/fake/config",
		Loader:           loader,
		PasswordResolver: pwResolver,
		TunnelManager:    tunnelMgr,
		MySQLConnector:   mysqlConnector,
		PGConnector:      pgConnector,
		Pool:             pool,
		Logger:           &mockLogger{},
	})
}

// --- tests ---

func TestConnectionService_ListConnections_Empty(t *testing.T) {
	loader := &mockConfigLoader{
		connectionsResult: domain.Result[*domainconfig.ConnectionsConfig]{
			Data: &domainconfig.ConnectionsConfig{
				Groups:     map[string]*domainconfig.Group{},
				SSHTunnels: map[string]domainconfig.SSHTunnelEntry{},
			},
		},
	}

	svc := makeService(loader, &mockPasswordResolver{}, &mockTunnelManager{}, &mockConnector{}, &mockConnector{}, newMockPool())
	result := svc.ListConnections(context.Background())
	require.NoError(t, result.Error)
	assert.Empty(t, result.Data)
}

func TestConnectionService_ListConnections_ReturnsAllConnections(t *testing.T) {
	loader := &mockConfigLoader{
		connectionsResult: domain.Result[*domainconfig.ConnectionsConfig]{
			Data: makeSimpleConfig(),
		},
	}

	svc := makeService(loader, &mockPasswordResolver{}, &mockTunnelManager{}, &mockConnector{}, &mockConnector{}, newMockPool())
	result := svc.ListConnections(context.Background())
	require.NoError(t, result.Error)
	assert.Len(t, result.Data, 1)
	assert.Equal(t, "mygroup.write.main-db", result.Data[0].Path)
}

func TestConnectionService_ListConnections_ConfigError(t *testing.T) {
	loader := &mockConfigLoader{
		connectionsResult: domain.Result[*domainconfig.ConnectionsConfig]{
			Error: errors.New("config not found"),
		},
	}

	svc := makeService(loader, &mockPasswordResolver{}, &mockTunnelManager{}, &mockConnector{}, &mockConnector{}, newMockPool())
	result := svc.ListConnections(context.Background())
	assert.Error(t, result.Error)
}

func TestConnectionService_Connect_Success(t *testing.T) {
	cfg := makeSimpleConfig()
	loader := &mockConfigLoader{
		connectionsResult: domain.Result[*domainconfig.ConnectionsConfig]{Data: cfg},
	}
	pool := newMockPool()

	svc := makeService(
		loader,
		&mockPasswordResolver{password: "secret"},
		&mockTunnelManager{},
		&mockConnector{db: &sql.DB{}},
		&mockConnector{},
		pool,
	)

	result := svc.Connect(context.Background(), "mygroup.write.main-db")
	require.NoError(t, result.Error)
	assert.Equal(t, "mygroup.write.main-db", result.Data.Path)
}

func TestConnectionService_Connect_NotFound(t *testing.T) {
	loader := &mockConfigLoader{
		connectionsResult: domain.Result[*domainconfig.ConnectionsConfig]{
			Data: makeSimpleConfig(),
		},
	}

	svc := makeService(loader, &mockPasswordResolver{}, &mockTunnelManager{}, &mockConnector{}, &mockConnector{}, newMockPool())
	result := svc.Connect(context.Background(), "nonexistent.path.conn")
	assert.Error(t, result.Error)
}

func TestConnectionService_Connect_ConnectorError(t *testing.T) {
	loader := &mockConfigLoader{
		connectionsResult: domain.Result[*domainconfig.ConnectionsConfig]{
			Data: makeSimpleConfig(),
		},
	}
	connErr := errors.New("connection refused")

	svc := makeService(
		loader,
		&mockPasswordResolver{password: "pass"},
		&mockTunnelManager{},
		&mockConnector{err: connErr},
		&mockConnector{},
		newMockPool(),
	)

	result := svc.Connect(context.Background(), "mygroup.write.main-db")
	assert.Error(t, result.Error)
}

func TestConnectionService_Disconnect_Success(t *testing.T) {
	pool := newMockPool()
	pool.dbs["mygroup.write.main-db"] = &sql.DB{}

	loader := &mockConfigLoader{
		connectionsResult: domain.Result[*domainconfig.ConnectionsConfig]{
			Data: makeSimpleConfig(),
		},
	}

	tunnelMgr := &mockTunnelManager{}
	svc := makeService(loader, &mockPasswordResolver{}, tunnelMgr, &mockConnector{}, &mockConnector{}, pool)

	result := svc.Disconnect(context.Background(), "mygroup.write.main-db")
	require.NoError(t, result.Error)
}

func TestConnectionService_Ping_NotConnected(t *testing.T) {
	pool := newMockPool()

	loader := &mockConfigLoader{
		connectionsResult: domain.Result[*domainconfig.ConnectionsConfig]{
			Data: makeSimpleConfig(),
		},
	}

	svc := makeService(loader, &mockPasswordResolver{}, &mockTunnelManager{}, &mockConnector{}, &mockConnector{}, pool)
	result := svc.Ping(context.Background(), "mygroup.write.main-db")
	assert.Error(t, result.Error)
}

func TestConnectionService_ValidateConfig_Success(t *testing.T) {
	loader := &mockConfigLoader{
		connectionsResult: domain.Result[*domainconfig.ConnectionsConfig]{
			Data: makeSimpleConfig(),
		},
	}

	svc := makeService(loader, &mockPasswordResolver{}, &mockTunnelManager{}, &mockConnector{}, &mockConnector{}, newMockPool())
	result := svc.ValidateConfig(context.Background())
	require.NoError(t, result.Error)
	assert.True(t, result.Data.Valid)
}

func TestConnectionService_ValidateConfig_Error(t *testing.T) {
	loader := &mockConfigLoader{
		connectionsResult: domain.Result[*domainconfig.ConnectionsConfig]{
			Error: errors.New("parse error"),
		},
	}

	svc := makeService(loader, &mockPasswordResolver{}, &mockTunnelManager{}, &mockConnector{}, &mockConnector{}, newMockPool())
	result := svc.ValidateConfig(context.Background())
	assert.Error(t, result.Error)
}

func TestConnectionService_ResultWrapping(t *testing.T) {
	// Verify all methods return Result[T] (compilation check)
	loader := &mockConfigLoader{
		connectionsResult: domain.Result[*domainconfig.ConnectionsConfig]{
			Data: makeSimpleConfig(),
		},
	}
	svc := makeService(loader, &mockPasswordResolver{}, &mockTunnelManager{}, &mockConnector{}, &mockConnector{}, newMockPool())

	var _ domain.Result[domainconn.Info] = svc.Connect(context.Background(), "x")
	var _ domain.Result[struct{}] = svc.Disconnect(context.Background(), "x")
	var _ domain.Result[domainconn.PingResult] = svc.Ping(context.Background(), "x")
	var _ domain.Result[[]domainconn.Info] = svc.ListConnections(context.Background())
	var _ domain.Result[ValidationReport] = svc.ValidateConfig(context.Background())
}
