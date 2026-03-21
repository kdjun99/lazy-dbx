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
)

var (
	configDir string
	svc       *app.ConnectionService
)

var rootCmd = &cobra.Command{
	Use:   "lazy-dbx",
	Short: "Terminal-based Database IDE",
	Long:  "lazy-dbx is a terminal-based Database IDE with connection management, SSH tunneling, and production safety features.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initService(cmd *cobra.Command, args []string) error {
	dir, err := resolveConfigDir(configDir)
	if err != nil {
		return fmt.Errorf("resolving config dir: %w", err)
	}

	logPath := filepath.Join(dir, "debug.log")
	var log domainlogger.Logger
	jsonLog, logErr := infralogger.New(logPath)
	if logErr != nil {
		log = infralogger.NewNopLogger()
	} else {
		log = jsonLog
	}

	svc = buildService(dir, log)
	return nil
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config", "", "Config directory (default: ~/.config/lazy-dbx)")

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
