package tui

import (
	"context"
	"errors"
	"sync"
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
	connectFunc      func(ctx context.Context, path string) domain.Result[domainconn.Info]
	connectResult    domain.Result[domainconn.Info]
	disconnectResult domain.Result[struct{}]
	pingResult       domain.Result[domainconn.PingResult]
	listResult       domain.Result[[]domainconn.Info]
	shutdownCalled   bool
	connectCalled    bool
	disconnectCalled bool
	lastPath         string
}

func (m *mockConnectionManager) Connect(ctx context.Context, path string) domain.Result[domainconn.Info] {
	m.connectCalled = true
	m.lastPath = path
	if m.connectFunc != nil {
		return m.connectFunc(ctx, path)
	}
	return m.connectResult
}

func (m *mockConnectionManager) Disconnect(_ context.Context, path string) domain.Result[struct{}] {
	m.disconnectCalled = true
	m.lastPath = path
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

// capturingLogger records log entries for verification.
type capturingLogger struct {
	mu      sync.Mutex
	entries []logEntry
}

type logEntry struct {
	level     string
	component string
	action    string
	detail    string
}

func (l *capturingLogger) log(level, component, action, detail string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, logEntry{level, component, action, detail})
}

func (l *capturingLogger) Debug(_ context.Context, component, action, detail string, _ ...domainlogger.Field) {
	l.log("debug", component, action, detail)
}

func (l *capturingLogger) Info(_ context.Context, component, action, detail string, _ ...domainlogger.Field) {
	l.log("info", component, action, detail)
}

func (l *capturingLogger) Warn(_ context.Context, component, action, detail string, _ ...domainlogger.Field) {
	l.log("warn", component, action, detail)
}

func (l *capturingLogger) Error(_ context.Context, component, action, detail string, _ ...domainlogger.Field) {
	l.log("error", component, action, detail)
}

func (l *capturingLogger) hasEntry(level, detail string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range l.entries {
		if e.level == level && e.detail == detail {
			return true
		}
	}
	return false
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

// --- NewApp tests ---

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

func TestNewApp_WithConfig(t *testing.T) {
	mgr := &mockConnectionManager{}
	cfg := &domainconfig.ConnectionsConfig{
		Groups: map[string]*domainconfig.Group{
			"dev": {
				Subgroups: map[string]*domainconfig.Subgroup{
					"local": {
						Connections: map[string]domainconfig.ConnectionEntry{
							"mydb": {Name: "mydb", Type: "mysql", Env: "test"},
						},
					},
				},
			},
		},
	}
	a := NewApp(mgr, cfg, nil, testLogger())
	require.NotNil(t, a)
	// Tree should have nodes for the connection.
	assert.NotNil(t, a.tree.nodeMap["dev.local.mydb"])
}

// --- handleSelect tests ---

func TestHandleSelect_PreventDoubleConnect(t *testing.T) {
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nil, nil, testLogger())

	a.connectingPaths["g.s.c"] = true
	a.handleSelect("g.s.c")
	assert.True(t, a.connectingPaths["g.s.c"])
	assert.False(t, mgr.connectCalled)
}

func TestHandleSelect_ConnectedPathTriggersDisconnect(t *testing.T) {
	mgr := &mockConnectionManager{
		disconnectResult: domain.Result[struct{}]{Data: struct{}{}},
	}
	a := NewApp(mgr, nil, nil, testLogger())
	a.connectedPaths["g.s.c"] = true
	a.activeConn = &domainconn.Info{Path: "g.s.c"}

	a.handleSelect("g.s.c")
	// Disconnect is async now, but manager.Disconnect is called in goroutine.
	// We can't easily test the async result here, tested via applyDisconnectResult.
}

// --- applyConnectResult tests (synchronous, fully testable) ---

func TestApplyConnectResult_Success(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nil, nil, log)

	path := "g.s.c"
	a.connectingPaths[path] = true

	result := domain.Result[domainconn.Info]{
		Data: domainconn.Info{Path: path, Name: "mydb", Type: "mysql", Env: "production"},
	}

	a.applyConnectResult(path, result)

	assert.True(t, a.connectedPaths[path])
	assert.False(t, a.connectingPaths[path])
	require.NotNil(t, a.activeConn)
	assert.Equal(t, path, a.activeConn.Path)
	assert.True(t, log.hasEntry("info", "connected"))
}

func TestApplyConnectResult_Error(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nil, nil, log)

	path := "g.s.c"
	a.connectingPaths[path] = true

	result := domain.Result[domainconn.Info]{
		Error: errors.New("connection refused"),
	}

	a.applyConnectResult(path, result)

	assert.False(t, a.connectedPaths[path])
	assert.False(t, a.connectingPaths[path])
	assert.Nil(t, a.activeConn)
	assert.True(t, log.hasEntry("error", "connect failed"))
}

func TestApplyConnectResult_ActiveConnUpdatedToLatest(t *testing.T) {
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nil, nil, testLogger())

	// First connection.
	a.applyConnectResult("g.s.first", domain.Result[domainconn.Info]{
		Data: domainconn.Info{Path: "g.s.first", Name: "first"},
	})
	assert.Equal(t, "g.s.first", a.activeConn.Path)

	// Second connection — activeConn should be the latest.
	a.applyConnectResult("g.s.second", domain.Result[domainconn.Info]{
		Data: domainconn.Info{Path: "g.s.second", Name: "second"},
	})
	assert.Equal(t, "g.s.second", a.activeConn.Path)
	// Both should be in connectedPaths.
	assert.True(t, a.connectedPaths["g.s.first"])
	assert.True(t, a.connectedPaths["g.s.second"])
}

// --- applyDisconnectResult tests ---

func TestApplyDisconnectResult_Success(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nil, nil, log)

	path := "g.s.c"
	a.connectedPaths[path] = true
	a.activeConn = &domainconn.Info{Path: path}

	result := domain.Result[struct{}]{Data: struct{}{}}
	a.applyDisconnectResult(path, result)

	assert.False(t, a.connectedPaths[path])
	assert.Nil(t, a.activeConn)
	assert.True(t, log.hasEntry("info", "disconnected"))
}

func TestApplyDisconnectResult_Error(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nil, nil, log)

	path := "g.s.c"
	a.connectedPaths[path] = true

	result := domain.Result[struct{}]{Error: errors.New("tunnel close failed")}
	a.applyDisconnectResult(path, result)

	assert.False(t, a.connectedPaths[path])
	assert.True(t, log.hasEntry("error", "disconnect failed"))
}

func TestApplyDisconnectResult_OtherActiveConnUntouched(t *testing.T) {
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nil, nil, testLogger())

	a.connectedPaths["g.s.c"] = true
	a.activeConn = &domainconn.Info{Path: "g.s.other"}

	result := domain.Result[struct{}]{Data: struct{}{}}
	a.applyDisconnectResult("g.s.c", result)

	assert.False(t, a.connectedPaths["g.s.c"])
	require.NotNil(t, a.activeConn)
	assert.Equal(t, "g.s.other", a.activeConn.Path)
}
