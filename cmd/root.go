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
	"github.com/kdjun99/lazy-dbx/internal/infra/mysql"
	infrapassword "github.com/kdjun99/lazy-dbx/internal/infra/password"
	"github.com/kdjun99/lazy-dbx/internal/infra/postgres"
	infratunnel "github.com/kdjun99/lazy-dbx/internal/infra/tunnel"
)

var configDir string

var rootCmd = &cobra.Command{
	Use:   "lazy-dbx",
	Short: "Terminal-based Database IDE",
	Long:  "lazy-dbx is a terminal-based Database IDE with connection management, SSH tunneling, and production safety features.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("lazy-dbx — Terminal Database IDE")
		fmt.Println("Run 'lazy-dbx --help' for usage.")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configDir, "config", "", "Config directory (default: ~/.config/lazy-dbx)")

	cobra.OnInitialize(func() {
		dir, err := resolveConfigDir(configDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving config dir: %v\n", err)
			os.Exit(1)
		}

		logPath := filepath.Join(dir, "debug.log")
		log, err := infralogger.New(logPath)
		if err != nil {
			// Fall back to nop logger if log file cannot be opened
			nop := infralogger.NewNopLogger()
			svc := buildService(dir, nop)
			rootCmd.AddCommand(newConnectCmd(svc))
			rootCmd.AddCommand(newConfigCmd(svc))
			return
		}

		svc := buildService(dir, log)
		rootCmd.AddCommand(newConnectCmd(svc))
		rootCmd.AddCommand(newConfigCmd(svc))
	})
}

func buildService(dir string, log domainlogger.Logger) *app.ConnectionService {
	return app.NewConnectionService(app.Config{
		ConfigDir:        dir,
		Loader:           infraconfig.NewTOMLLoader(),
		PasswordResolver: infrapassword.NewMultiResolver(log),
		TunnelManager:    infratunnel.NewTunnelManager(),
		MySQLConnector:   &mysql.Connector{},
		PGConnector:      &postgres.Connector{},
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
