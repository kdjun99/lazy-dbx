package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kdjun99/lazy-dbx/internal/app"
)

func newConfigCmd(svc *app.ConnectionService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and validate configuration",
	}

	cmd.AddCommand(newConfigValidateCmd(svc))
	cmd.AddCommand(newConfigShowCmd(svc))

	return cmd
}

func newConfigValidateCmd(svc *app.ConnectionService) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Parse and validate connections.toml without connecting",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			result := svc.ValidateConfig(ctx)
			if result.Error != nil {
				fmt.Fprintf(os.Stderr, "Config invalid: %v\n", result.Error)
				os.Exit(1)
			}

			fmt.Printf("Config OK — %d connection(s) configured\n", result.Data.ConnCount)
			return nil
		},
	}
}

func newConfigShowCmd(svc *app.ConnectionService) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show all configured connections",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			result := svc.ListConnections(ctx)
			if result.Error != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", result.Error)
				os.Exit(1)
			}

			if len(result.Data) == 0 {
				fmt.Println("No connections configured.")
				return nil
			}

			for _, info := range result.Data {
				env := info.Env
				if env == "" {
					env = "(no env)"
				}
				fmt.Printf("  [%s] %s  %s@%s  env=%s\n",
					info.Type, info.Path, info.Name, info.Host, env)
			}
			return nil
		},
	}
}
