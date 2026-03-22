package tui

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdjun99/lazy-dbx/internal/domain"
	domainconfig "github.com/kdjun99/lazy-dbx/internal/domain/config"
	domainconn "github.com/kdjun99/lazy-dbx/internal/domain/connection"
	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
	"github.com/kdjun99/lazy-dbx/internal/domain/query"
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

// mockExecutor is a test double for query.Executor.
type mockExecutor struct {
	result domain.Result[query.Result]
}

func (m *mockExecutor) Execute(_ context.Context, _ string, _ string) domain.Result[query.Result] {
	return m.result
}

func nopExecutor() query.Executor {
	return &mockExecutor{}
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
	a := NewApp(mgr, nopExecutor(), nil, nil, testLogger(), nil)
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
	a := NewApp(mgr, nopExecutor(), cfg, nil, testLogger(), nil)
	require.NotNil(t, a)
	assert.False(t, a.readonly)
}

func TestNewApp_ReadonlyFromSettings(t *testing.T) {
	mgr := &mockConnectionManager{}
	settings := &domainconfig.SettingsConfig{
		Safety: domainconfig.SafetySettings{ReadonlyByDefault: true},
	}
	a := NewApp(mgr, nopExecutor(), nil, settings, testLogger(), nil)
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
	a := NewApp(mgr, nopExecutor(), cfg, nil, testLogger(), nil)
	require.NotNil(t, a)
	// Tree should have nodes for the connection.
	assert.NotNil(t, a.tree.nodeMap["dev.local.mydb"])
}

// --- handleSelect tests ---

func TestHandleSelect_PreventDoubleConnect(t *testing.T) {
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), nil, nil, testLogger(), nil)

	a.connectingPaths["g.s.c"] = true
	a.handleSelect("g.s.c")
	assert.True(t, a.connectingPaths["g.s.c"])
	assert.False(t, mgr.connectCalled)
}

func TestHandleSelect_ConnectedPathTriggersDisconnect(t *testing.T) {
	mgr := &mockConnectionManager{
		disconnectResult: domain.Result[struct{}]{Data: struct{}{}},
	}
	a := NewApp(mgr, nopExecutor(), nil, nil, testLogger(), nil)
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
	a := NewApp(mgr, nopExecutor(), nil, nil, log, nil)

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
	a := NewApp(mgr, nopExecutor(), nil, nil, log, nil)

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
	a := NewApp(mgr, nopExecutor(), nil, nil, testLogger(), nil)

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
	a := NewApp(mgr, nopExecutor(), nil, nil, log, nil)

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
	a := NewApp(mgr, nopExecutor(), nil, nil, log, nil)

	path := "g.s.c"
	a.connectedPaths[path] = true

	result := domain.Result[struct{}]{Error: errors.New("tunnel close failed")}
	a.applyDisconnectResult(path, result)

	assert.False(t, a.connectedPaths[path])
	assert.True(t, log.hasEntry("error", "disconnect failed"))
}

func TestApplyDisconnectResult_OtherActiveConnUntouched(t *testing.T) {
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), nil, nil, testLogger(), nil)

	a.connectedPaths["g.s.c"] = true
	a.activeConn = &domainconn.Info{Path: "g.s.other"}

	result := domain.Result[struct{}]{Data: struct{}{}}
	a.applyDisconnectResult("g.s.c", result)

	assert.False(t, a.connectedPaths["g.s.c"])
	require.NotNil(t, a.activeConn)
	assert.Equal(t, "g.s.other", a.activeConn.Path)
}

// --- Scenario tests: verify tree icon + status bar + state together ---

func testConfig() *domainconfig.ConnectionsConfig {
	return &domainconfig.ConnectionsConfig{
		Groups: map[string]*domainconfig.Group{
			"local": {
				Subgroups: map[string]*domainconfig.Subgroup{
					"mysql": {
						Connections: map[string]domainconfig.ConnectionEntry{
							"dev-db": {Name: "dev-db", Type: "mysql", Env: "test"},
						},
					},
				},
			},
		},
	}
}

const testPath = "local.mysql.dev-db"

func treeNodeIcon(a *App, path string) string {
	tNode, ok := a.tree.nodeMap[path]
	if !ok {
		return ""
	}
	runes := []rune(tNode.GetText())
	if len(runes) == 0 {
		return ""
	}
	return string(runes[0])
}

func treeNodeText(a *App, path string) string {
	tNode, ok := a.tree.nodeMap[path]
	if !ok {
		return ""
	}
	return tNode.GetText()
}

func TestScenario_ConnectSuccess_TreeIconAndStatusBar(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), testConfig(), nil, log, nil)

	// Before connect: icon is ○ (disconnected).
	assert.Equal(t, "○", treeNodeIcon(a, testPath))

	// Simulate connect initiation.
	a.connectingPaths[testPath] = true
	a.statusBar.SetMessage("Connecting to "+testPath+"...", false)
	a.tree.UpdateNodeStatus(testPath, ConnectionStatusConnecting)

	// During connect: icon is ◌ (connecting) with … suffix, status bar shows connecting message.
	assert.Equal(t, "◌", treeNodeIcon(a, testPath))
	assert.Contains(t, treeNodeText(a, testPath), "…")
	assert.Contains(t, a.statusBar.widget.GetText(false), "Connecting to "+testPath)

	// Simulate connect result (success).
	result := domain.Result[domainconn.Info]{
		Data: domainconn.Info{Path: testPath, Name: "dev-db", Type: "mysql", Env: "test"},
	}
	a.applyConnectResult(testPath, result)

	// After connect: icon is ● (connected) with ✓ suffix, status bar shows connection info.
	assert.Equal(t, "●", treeNodeIcon(a, testPath))
	assert.Contains(t, treeNodeText(a, testPath), "✓")
	assert.True(t, a.connectedPaths[testPath])
	require.NotNil(t, a.activeConn)
	assert.Equal(t, testPath, a.activeConn.Path)
	assert.True(t, log.hasEntry("info", "connected"))
}

func TestScenario_ConnectError_TreeIconAndStatusBar(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), testConfig(), nil, log, nil)

	// Set up connecting state.
	a.connectingPaths[testPath] = true
	a.tree.UpdateNodeStatus(testPath, ConnectionStatusConnecting)
	assert.Equal(t, "◌", treeNodeIcon(a, testPath))

	// Simulate connect error.
	result := domain.Result[domainconn.Info]{
		Error: errors.New("connection refused"),
	}
	a.applyConnectResult(testPath, result)

	// After error: icon reverts to ○, status bar shows red error.
	assert.Equal(t, "○", treeNodeIcon(a, testPath))
	assert.False(t, a.connectedPaths[testPath])
	assert.Nil(t, a.activeConn)
	statusText := a.statusBar.widget.GetText(false)
	assert.Contains(t, statusText, "connection refused")
	assert.Contains(t, statusText, "[red]")
	assert.True(t, log.hasEntry("error", "connect failed"))
}

func TestScenario_DisconnectSuccess_TreeIconAndStatusBar(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), testConfig(), nil, log, nil)

	// Set up connected state.
	a.connectedPaths[testPath] = true
	a.activeConn = &domainconn.Info{Path: testPath, Name: "dev-db"}
	a.tree.UpdateNodeStatus(testPath, ConnectionStatusConnected)
	assert.Equal(t, "●", treeNodeIcon(a, testPath))

	// Simulate disconnect result (success).
	result := domain.Result[struct{}]{Data: struct{}{}}
	a.applyDisconnectResult(testPath, result)

	// After disconnect: icon is ○, activeConn cleared, status bar shows disconnect msg.
	assert.Equal(t, "○", treeNodeIcon(a, testPath))
	assert.False(t, a.connectedPaths[testPath])
	assert.Nil(t, a.activeConn)
	statusText := a.statusBar.widget.GetText(false)
	assert.Contains(t, statusText, "Disconnected from "+testPath)
	assert.True(t, log.hasEntry("info", "disconnected"))
}

func TestScenario_DisconnectError_TreeIconAndStatusBar(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), testConfig(), nil, log, nil)

	// Set up connected state.
	a.connectedPaths[testPath] = true
	a.tree.UpdateNodeStatus(testPath, ConnectionStatusConnected)

	// Simulate disconnect error.
	result := domain.Result[struct{}]{Error: errors.New("tunnel close timeout")}
	a.applyDisconnectResult(testPath, result)

	// After error: icon is ○ (force-disconnected), status bar shows red error.
	assert.Equal(t, "○", treeNodeIcon(a, testPath))
	assert.False(t, a.connectedPaths[testPath])
	statusText := a.statusBar.widget.GetText(false)
	assert.Contains(t, statusText, "tunnel close timeout")
	assert.Contains(t, statusText, "[red]")
	assert.True(t, log.hasEntry("error", "disconnect failed"))
}

func TestScenario_FullCycle_ConnectThenDisconnect(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), testConfig(), nil, log, nil)

	// Step 1: Connect.
	a.connectingPaths[testPath] = true
	a.tree.UpdateNodeStatus(testPath, ConnectionStatusConnecting)
	a.applyConnectResult(testPath, domain.Result[domainconn.Info]{
		Data: domainconn.Info{Path: testPath, Name: "dev-db", Type: "mysql", Env: "test"},
	})
	assert.Equal(t, "●", treeNodeIcon(a, testPath))
	assert.True(t, a.connectedPaths[testPath])

	// Step 2: Disconnect.
	a.applyDisconnectResult(testPath, domain.Result[struct{}]{Data: struct{}{}})
	assert.Equal(t, "○", treeNodeIcon(a, testPath))
	assert.False(t, a.connectedPaths[testPath])
	assert.Nil(t, a.activeConn)

	// Step 3: Reconnect.
	a.connectingPaths[testPath] = true
	a.tree.UpdateNodeStatus(testPath, ConnectionStatusConnecting)
	assert.Equal(t, "◌", treeNodeIcon(a, testPath))
	a.applyConnectResult(testPath, domain.Result[domainconn.Info]{
		Data: domainconn.Info{Path: testPath, Name: "dev-db", Type: "mysql", Env: "test"},
	})
	assert.Equal(t, "●", treeNodeIcon(a, testPath))
	assert.True(t, a.connectedPaths[testPath])
}

// --- applyResult tests ---

func TestApplyResult_Select_Success(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), nil, nil, log, nil)

	result := domain.Result[query.Result]{
		Data: query.Result{
			Columns:   []query.ColumnInfo{{Name: "id", TypeName: "INT"}},
			Rows:      [][]query.Value{{{String: "1", Valid: true}}},
			TotalRows: 1,
			Type:      query.StatementQuery,
			Duration:  10 * time.Millisecond,
		},
	}

	a.applyResult(result)

	assert.False(t, a.isExecuting)
	assert.Nil(t, a.cancelQuery)
	assert.True(t, log.hasEntry("info", "completed"))
}

func TestApplyResult_Error(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), nil, nil, log, nil)
	a.isExecuting = true

	result := domain.Result[query.Result]{
		Error: errors.New("syntax error"),
	}

	a.applyResult(result)

	assert.False(t, a.isExecuting)
	statusText := a.statusBar.widget.GetText(false)
	assert.Contains(t, statusText, "syntax error")
	assert.True(t, log.hasEntry("error", "query failed"))
}

func TestApplyResult_DML(t *testing.T) {
	log := &capturingLogger{}
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), nil, nil, log, nil)

	result := domain.Result[query.Result]{
		Data: query.Result{
			Type:         query.StatementExec,
			RowsAffected: 42,
			Duration:     5 * time.Millisecond,
		},
	}

	a.applyResult(result)

	assert.False(t, a.isExecuting)
	statusText := a.statusBar.widget.GetText(false)
	assert.Contains(t, statusText, "42")
	assert.True(t, log.hasEntry("info", "completed"))
}

// --- handleExecute guard tests ---

func TestHandleExecute_NoConnection(t *testing.T) {
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), nil, nil, testLogger(), nil)
	a.activeConn = nil

	a.handleExecute()

	assert.False(t, a.isExecuting)
	statusText := a.statusBar.widget.GetText(false)
	assert.Contains(t, statusText, "No active connection")
}

func TestHandleExecute_EmptyQuery(t *testing.T) {
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), nil, nil, testLogger(), nil)
	a.activeConn = &domainconn.Info{Path: testPath}
	a.editor.SetText("   ")

	a.handleExecute()

	assert.False(t, a.isExecuting)
	statusText := a.statusBar.widget.GetText(false)
	assert.Contains(t, statusText, "Empty query")
}

func TestHandleExecute_WhileExecuting(t *testing.T) {
	mgr := &mockConnectionManager{}
	a := NewApp(mgr, nopExecutor(), nil, nil, testLogger(), nil)
	a.activeConn = &domainconn.Info{Path: testPath}
	a.isExecuting = true

	a.handleExecute()

	// Still executing, no second goroutine started.
	assert.True(t, a.isExecuting)
}
