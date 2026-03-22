package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kdjun99/lazy-dbx/internal/app"
	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
	infraconfig "github.com/kdjun99/lazy-dbx/internal/infra/config"
	infraconn "github.com/kdjun99/lazy-dbx/internal/infra/connection"
	infralogger "github.com/kdjun99/lazy-dbx/internal/infra/logger"
	inframysql "github.com/kdjun99/lazy-dbx/internal/infra/mysql"
	infrapassword "github.com/kdjun99/lazy-dbx/internal/infra/password"
	infrapg "github.com/kdjun99/lazy-dbx/internal/infra/postgres"
	infratunnel "github.com/kdjun99/lazy-dbx/internal/infra/tunnel"
	"github.com/kdjun99/lazy-dbx/internal/tui"
)

var (
	configDir string
	noTUI     bool
	svc       *app.ConnectionService
)

var rootCmd = &cobra.Command{
	Use:   "lazy-dbx",
	Short: "Terminal-based Database IDE",
	Long:  "lazy-dbx is a terminal-based Database IDE with connection management, SSH tunneling, and production safety features.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if noTUI {
			return cmd.Help()
		}
		return runTUI(cmd)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initService(_ *cobra.Command, _ []string) error {
	dir, log := initLogger(configDir)
	if dir == "" {
		return fmt.Errorf("resolving config dir")
	}
	svc = buildService(dir, log)
	return nil
}

func runTUI(_ *cobra.Command) error {
	dir, log := initLogger(configDir)

	loader := infraconfig.NewTOMLLoader()

	connResult, err := loader.LoadConnections(filepath.Join(dir, "connections.toml"))
	if err != nil {
		return fmt.Errorf("loading connections config: %w", err)
	}
	if connResult.Error != nil {
		return fmt.Errorf("loading connections config: %w", connResult.Error)
	}

	settingsResult, err := loader.LoadSettings(filepath.Join(dir, "settings.toml"))
	if err != nil {
		return fmt.Errorf("loading settings config: %w", err)
	}
	if settingsResult.Error != nil {
		return fmt.Errorf("loading settings config: %w", settingsResult.Error)
	}

	service := buildService(dir, log)

	maxRows := 10000
	if settingsResult.Data != nil && settingsResult.Data.UI.MaxResultRows > 0 {
		maxRows = settingsResult.Data.UI.MaxResultRows
	}
	querySvc := app.NewQueryService(service.GetPool(), log, maxRows)

	// driverTypeFunc looks up the DB type ("mysql"/"postgresql") for a connection path.
	connCfg := connResult.Data
	driverTypeFunc := func(path string) string {
		if connCfg == nil {
			return "mysql"
		}
		entry, err := connCfg.FindConnection(path)
		if err != nil || entry == nil {
			return "mysql"
		}
		return entry.Type
	}

	catalogSvc := app.NewCatalogService(service.GetPool(), log, driverTypeFunc)

	tuiApp := tui.NewApp(service, querySvc, connResult.Data, settingsResult.Data, log, catalogSvc)
	return tuiApp.Run()
}

func initLogger(cfgDir string) (string, domainlogger.Logger) {
	dir, err := resolveConfigDir(cfgDir)
	if err != nil {
		return "", infralogger.NewNopLogger()
	}

	logPath := filepath.Join(dir, "debug.log")
	jsonLog, logErr := infralogger.New(logPath)
	if logErr != nil {
		return dir, infralogger.NewNopLogger()
	}
	return dir, jsonLog
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config", "", "Config directory (default: ~/.config/lazy-dbx)")
	rootCmd.Flags().BoolVar(&noTUI, "no-tui", false, "Print help instead of launching TUI")

	rootCmd.AddCommand(newConnectCmd())
	rootCmd.AddCommand(newConfigCmd())
}

func buildService(dir string, log domainlogger.Logger) *app.ConnectionService {
	return app.NewConnectionService(app.Config{
		ConfigDir:        dir,
		Loader:           infraconfig.NewTOMLLoader(),
		PasswordResolver: infrapassword.NewMultiResolver(log),
		TunnelManager:    infratunnel.NewTunnelManager(),
		MySQLConnector:   &inframysql.Connector{},
		PGConnector:      &infrapg.Connector{},
		Pool:             infraconn.NewInMemoryPool(),
		Logger:           log,
	})
}

func resolveConfigDir(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if env := os.Getenv("LAZY_DBX_CONFIG_DIR"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lazy-dbx"), nil
}
