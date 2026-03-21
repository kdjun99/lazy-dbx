package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/rivo/tview"

	"github.com/kdjun99/lazy-dbx/internal/domain"
	domainconfig "github.com/kdjun99/lazy-dbx/internal/domain/config"
	domainconn "github.com/kdjun99/lazy-dbx/internal/domain/connection"
	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
	"github.com/kdjun99/lazy-dbx/internal/domain/query"
)

// App is the main TUI application wrapping tview.Application.
type App struct {
	tviewApp        *tview.Application
	tree            *ConnectionTree
	statusBar       *StatusBar
	focusManager    *FocusManager
	manager         domainconn.Manager
	executor        query.Executor
	editor          *QueryEditor
	results         *ResultsTable
	logger          domainlogger.Logger
	connectedPaths  map[string]bool
	connectingPaths map[string]bool
	activeConn      *domainconn.Info
	readonly        bool
	cfg             *domainconfig.ConnectionsConfig
	isExecuting     bool
	cancelQuery     context.CancelFunc
	pageSize        int
}

// NewApp constructs the full TUI application.
// manager handles connection lifecycle; executor runs SQL queries;
// cfg and settings are loaded once at startup.
func NewApp(
	manager domainconn.Manager,
	executor query.Executor,
	cfg *domainconfig.ConnectionsConfig,
	settings *domainconfig.SettingsConfig,
	log domainlogger.Logger,
) *App {
	a := &App{
		tviewApp:        tview.NewApplication(),
		manager:         manager,
		executor:        executor,
		logger:          log,
		connectedPaths:  make(map[string]bool),
		connectingPaths: make(map[string]bool),
		cfg:             cfg,
		pageSize:        100,
	}

	if settings != nil {
		a.readonly = settings.Safety.ReadonlyByDefault
		if settings.UI.ResultPageSize > 0 {
			a.pageSize = settings.UI.ResultPageSize
		}
	}

	// Build tree data from config (nil or empty config shows placeholder).
	var treeData []TreeNode
	if cfg != nil && len(cfg.Groups) > 0 {
		treeData = BuildTreeData(cfg, a.readonly, nil)
	}

	a.tree = NewConnectionTree(treeData, a.handleSelect)

	if len(treeData) == 0 {
		a.tree.SetEmptyMessage("No connections configured")
	}

	a.statusBar = NewStatusBar()
	a.statusBar.SetApp(a.tviewApp)
	a.statusBar.SetKeybindings("Ctrl+Q:Quit  Tab:Focus  Ctrl+E:Execute  Enter:Connect")
	a.statusBar.SetMode(a.readonly)

	a.editor = NewQueryEditor()
	a.results = NewResultsTable()

	a.focusManager = NewFocusManager(a.tviewApp, a.tree.Widget(), a.editor.Widget(), a.results.Widget())

	layout := BuildLayout(a.tree.Widget(), a.editor.Widget(), a.results.Widget(), a.statusBar.Widget())
	a.tviewApp.SetRoot(layout, true)

	RegisterKeybindings(a.tviewApp, a.focusManager, func() {
		a.tviewApp.Stop()
	}, a.handleExecute, a.handleCancel)

	log.Info(context.Background(), "App", "NewApp", "TUI initialized",
		domainlogger.F("readonly", fmt.Sprintf("%v", a.readonly)))

	return a
}

// Run starts the tview event loop and defers graceful shutdown.
func (a *App) Run() error {
	ctx := context.Background()
	defer a.manager.Shutdown(ctx)

	a.logger.Info(ctx, "App", "Run", "starting TUI")
	if err := a.tviewApp.Run(); err != nil {
		a.logger.Error(ctx, "App", "Run", "tview error", domainlogger.F("error", err.Error()))
		return err
	}
	a.logger.Info(ctx, "App", "Run", "TUI exited")
	return nil
}

// handleSelect is called when the user selects a leaf node in the connection tree.
func (a *App) handleSelect(path string) {
	ctx := context.Background()

	// Prevent double-connect while already connecting.
	if a.connectingPaths[path] {
		return
	}

	if a.connectedPaths[path] {
		a.handleDisconnect(ctx, path)
		return
	}

	// Connect flow.
	a.connectingPaths[path] = true
	a.statusBar.SetMessage("Connecting to "+path+"...", false)
	a.tree.UpdateNodeStatus(path, ConnectionStatusConnecting)

	a.logger.Info(ctx, "App", "handleSelect", "connecting",
		domainlogger.F("path", path))

	go func() {
		result := a.manager.Connect(ctx, path)
		a.tviewApp.QueueUpdateDraw(func() {
			a.applyConnectResult(path, result)
		})
	}()
}

// applyConnectResult updates UI state after a connect attempt completes.
// Extracted from the goroutine for testability.
func (a *App) applyConnectResult(path string, result domain.Result[domainconn.Info]) {
	ctx := context.Background()
	delete(a.connectingPaths, path)

	if result.Error != nil {
		a.tree.UpdateNodeStatus(path, ConnectionStatusDisconnected)
		a.statusBar.SetMessage(result.Error.Error(), true)
		a.logger.Error(ctx, "App", "handleSelect", "connect failed",
			domainlogger.F("path", path),
			domainlogger.F("error", result.Error.Error()))
		return
	}

	a.connectedPaths[path] = true
	a.tree.UpdateNodeStatus(path, ConnectionStatusConnected)
	info := result.Data
	a.activeConn = &info
	a.statusBar.SetConnectionInfo(FormatConnectionInfo(info.Name, info.Type, info.Env))
	a.statusBar.SetMessage("Connected to "+path, false)
	a.logger.Info(ctx, "App", "handleSelect", "connected",
		domainlogger.F("path", path))
}

// handleDisconnect disconnects an active connection asynchronously.
func (a *App) handleDisconnect(ctx context.Context, path string) {
	a.statusBar.SetMessage("Disconnecting from "+path+"...", false)

	a.logger.Info(ctx, "App", "handleDisconnect", "disconnecting",
		domainlogger.F("path", path))

	go func() {
		result := a.manager.Disconnect(ctx, path)
		a.tviewApp.QueueUpdateDraw(func() {
			a.applyDisconnectResult(path, result)
		})
	}()
}

// applyDisconnectResult updates UI state after a disconnect attempt completes.
// Extracted from the goroutine for testability.
func (a *App) applyDisconnectResult(path string, result domain.Result[struct{}]) {
	ctx := context.Background()
	delete(a.connectedPaths, path)
	a.tree.UpdateNodeStatus(path, ConnectionStatusDisconnected)

	if a.activeConn != nil && a.activeConn.Path == path {
		a.activeConn = nil
		a.statusBar.SetConnectionInfo("")
	}

	if result.Error != nil {
		a.logger.Error(ctx, "App", "handleDisconnect", "disconnect failed",
			domainlogger.F("path", path),
			domainlogger.F("error", result.Error.Error()))
		a.statusBar.SetMessage(result.Error.Error(), true)
		return
	}

	a.logger.Info(ctx, "App", "handleDisconnect", "disconnected",
		domainlogger.F("path", path))
	a.statusBar.SetMessage("Disconnected from "+path, false)
}

// handleExecute runs the SQL from the editor against the active connection.
func (a *App) handleExecute() {
	if a.isExecuting {
		return
	}
	if a.activeConn == nil {
		a.statusBar.SetMessage("No active connection. Connect first.", true)
		return
	}
	sql := a.editor.GetText()
	if strings.TrimSpace(sql) == "" {
		a.statusBar.SetMessage("Empty query", true)
		return
	}
	a.isExecuting = true
	a.statusBar.SetMessage("Executing query...", false)
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelQuery = cancel

	// Capture path before goroutine to avoid race with disconnect.
	connPath := a.activeConn.Path

	a.logger.Info(ctx, "App", "handleExecute", "executing",
		domainlogger.F("path", connPath))

	go func() {
		defer cancel()
		result := a.executor.Execute(ctx, connPath, sql)
		a.tviewApp.QueueUpdateDraw(func() {
			a.applyResult(result)
		})
	}()
}

// handleCancel cancels a running query. Returns true if a query was cancelled.
func (a *App) handleCancel() bool {
	if a.isExecuting && a.cancelQuery != nil {
		a.cancelQuery()
		a.statusBar.SetMessage("Query cancelled", false)
		a.logger.Info(context.Background(), "App", "handleCancel", "query cancelled")
		return true
	}
	return false
}

// applyResult updates UI state after a query execution completes.
// Extracted from the goroutine for testability.
func (a *App) applyResult(result domain.Result[query.Result]) {
	a.isExecuting = false
	a.cancelQuery = nil
	ctx := context.Background()

	if result.Error != nil {
		a.results.ShowError(result.Error.Error())
		a.statusBar.SetMessage(result.Error.Error(), true)
		a.logger.Error(ctx, "App", "handleExecute", "query failed",
			domainlogger.F("error", result.Error.Error()))
		return
	}

	data := result.Data
	if data.Type == query.StatementExec {
		a.results.ShowExecResult(data.RowsAffected, data.Duration)
		a.statusBar.SetMessage(fmt.Sprintf("%d rows affected (%s)", data.RowsAffected, data.Duration), false)
	} else {
		a.results.SetData(&data, a.pageSize)
		a.statusBar.SetMessage(fmt.Sprintf("Query completed: %d rows in %s", data.TotalRows, data.Duration), false)
	}
	a.logger.Info(ctx, "App", "handleExecute", "completed",
		domainlogger.F("type", data.Type),
		domainlogger.F("duration_ms", data.Duration.Milliseconds()))
}
