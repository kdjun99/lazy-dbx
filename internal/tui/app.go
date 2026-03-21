package tui

import (
	"context"
	"fmt"

	"github.com/rivo/tview"

	"github.com/kdjun99/lazy-dbx/internal/domain"
	domainconfig "github.com/kdjun99/lazy-dbx/internal/domain/config"
	domainconn "github.com/kdjun99/lazy-dbx/internal/domain/connection"
	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
)

// App is the main TUI application wrapping tview.Application.
type App struct {
	tviewApp        *tview.Application
	tree            *ConnectionTree
	statusBar       *StatusBar
	focusManager    *FocusManager
	manager         domainconn.Manager
	logger          domainlogger.Logger
	connectedPaths  map[string]bool
	connectingPaths map[string]bool
	activeConn      *domainconn.Info
	readonly        bool
	cfg             *domainconfig.ConnectionsConfig
}

// NewApp constructs the full TUI application.
// manager handles connection lifecycle; cfg and settings are loaded once at startup.
func NewApp(
	manager domainconn.Manager,
	cfg *domainconfig.ConnectionsConfig,
	settings *domainconfig.SettingsConfig,
	log domainlogger.Logger,
) *App {
	a := &App{
		tviewApp:        tview.NewApplication(),
		manager:         manager,
		logger:          log,
		connectedPaths:  make(map[string]bool),
		connectingPaths: make(map[string]bool),
		cfg:             cfg,
	}

	if settings != nil {
		a.readonly = settings.Safety.ReadonlyByDefault
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
	a.statusBar.SetKeybindings("Ctrl+Q:Quit  Tab:Focus  Enter:Connect")
	a.statusBar.SetMode(a.readonly)

	welcomePanel := NewWelcomePanel()

	a.focusManager = NewFocusManager(a.tviewApp, a.tree.Widget(), welcomePanel)

	layout := BuildLayout(a.tree.Widget(), welcomePanel, a.statusBar.Widget())
	a.tviewApp.SetRoot(layout, true)

	RegisterKeybindings(a.tviewApp, a.focusManager, func() {
		a.tviewApp.Stop()
	})

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
