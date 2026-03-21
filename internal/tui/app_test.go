package tui

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdjun99/lazy-dbx/internal/domain"
	domainconfig "github.com/kdjun99/lazy-dbx/internal/domain/config"
	domainconn "github.com/kdjun99/lazy-dbx/internal/domain/connection"
	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
)

// mockConnectionManager is a test double for domainconn.Manager.
type mockConnectionManager struct {
	connectResult    domain.Result[domainconn.Info]
	disconnectResult domain.Result[struct{}]
	pingResult       domain.Result[domainconn.PingResult]
	listResult       domain.Result[[]domainconn.Info]
	shutdownCalled   bool
}

func (m *mockConnectionManager) Connect(_ context.Context, _ string) domain.Result[domainconn.Info] {
	return m.connectResult
}

func (m *mockConnectionManager) Disconnect(_ context.Context, _ string) domain.Result[struct{}] {
	return m.disconnectResult
}

func (m *mockConnectionManager) Ping(_ context.Context, _ string) domain.Result[domainconn.PingResult] {
	return m.pingResult
}

func (m *mockConnectionManager) ListConnections(_ context.Context) domain.Result[[]domainconn.Info] {
	return m.listResult
}

func (m *mockConnectionManager) Shutdown(_ context.Context) {
	m.shutdownCalled = true
}

// nopLogger satisfies domainlogger.Logger for testing.
type nopLogger struct{}

func (n *nopLogger) Debug(_ context.Context, _, _, _ string, _ ...domainlogger.Field) {}
func (n *nopLogger) Info(_ context.Context, _, _, _ string, _ ...domainlogger.Field)  {}
func (n *nopLogger) Warn(_ context.Context, _, _, _ string, _ ...domainlogger.Field)  {}
func (n *nopLogger) Error(_ context.Context, _, _, _ string, _ ...domainlogger.Field) {}

func testLogger() domainlogger.Logger {
	return &nopLogger{}
}

func TestNewApp_NilConfig(t *testing.T) {
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nil, nil, testLogger())
	require.NotNil(t, a)
	assert.NotNil(t, a.tviewApp)
	assert.NotNil(t, a.tree)
	assert.NotNil(t, a.statusBar)
	assert.NotNil(t, a.focusManager)
}

func TestNewApp_EmptyConfig(t *testing.T) {
	mgr := &mockConnectionManager{}
	cfg := &domainconfig.ConnectionsConfig{
		Groups: map[string]*domainconfig.Group{},
	}
	a := NewApp(mgr, cfg, nil, testLogger())
	require.NotNil(t, a)
	assert.False(t, a.readonly)
}

func TestNewApp_ReadonlyFromSettings(t *testing.T) {
	mgr := &mockConnectionManager{}
	settings := &domainconfig.SettingsConfig{
		Safety: domainconfig.SafetySettings{ReadonlyByDefault: true},
	}
	a := NewApp(mgr, nil, settings, testLogger())
	require.NotNil(t, a)
	assert.True(t, a.readonly)
}

func TestHandleSelect_PreventDoubleConnect(t *testing.T) {
	mgr := &mockConnectionManager{
		connectResult: domain.Result[domainconn.Info]{
			Data: domainconn.Info{Path: "g.s.c", Name: "test", Type: "mysql", Env: "test"},
		},
	}
	a := NewApp(mgr, nil, nil, testLogger())

	// Simulate already connecting.
	a.connectingPaths["g.s.c"] = true
	// handleSelect should return early without changing state.
	a.handleSelect("g.s.c")
	// connectingPaths should remain true (not cleared).
	assert.True(t, a.connectingPaths["g.s.c"])
}

func TestHandleDisconnect_ClearsActiveConn(t *testing.T) {
	mgr := &mockConnectionManager{
		disconnectResult: domain.Result[struct{}]{Data: struct{}{}},
	}
	a := NewApp(mgr, nil, nil, testLogger())

	path := "g.s.c"
	a.connectedPaths[path] = true
	a.activeConn = &domainconn.Info{Path: path}

	a.handleDisconnect(context.Background(), path)

	assert.False(t, a.connectedPaths[path])
	assert.Nil(t, a.activeConn)
}

func TestHandleDisconnect_OtherActiveConnUntouched(t *testing.T) {
	mgr := &mockConnectionManager{
		disconnectResult: domain.Result[struct{}]{Data: struct{}{}},
	}
	a := NewApp(mgr, nil, nil, testLogger())

	path := "g.s.c"
	otherPath := "g.s.other"
	a.connectedPaths[path] = true
	a.activeConn = &domainconn.Info{Path: otherPath}

	a.handleDisconnect(context.Background(), path)

	assert.False(t, a.connectedPaths[path])
	// activeConn for a different path is unchanged.
	assert.NotNil(t, a.activeConn)
	assert.Equal(t, otherPath, a.activeConn.Path)
}
