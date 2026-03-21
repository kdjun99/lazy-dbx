package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/kdjun99/lazy-dbx/internal/app"
)

var pingPath string

func newConnectCmd(svc *app.ConnectionService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "connect [path]",
		Short: "Connect to a database or list configured connections",
		Long:  "Connect to a database by its dot-separated path (group.subgroup.name), list all connections, or ping a connection.",
	}

	cmd.Flags().StringVar(&pingPath, "ping", "", "Connect, ping, and disconnect from the specified path")
	listFlag := cmd.Flags().Bool("list", false, "List all configured connections")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if *listFlag {
			return runConnectList(svc, cmd)
		}

		if pingPath != "" {
			return runConnectPing(svc, cmd, pingPath)
		}

		if len(args) == 0 {
			return cmd.Help()
		}

		result := svc.Connect(ctx, args[0])
		if result.Error != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", result.Error)
			os.Exit(1)
		}

		fmt.Printf("Connected: %s (%s @ %s)\n", result.Data.Path, result.Data.Type, result.Data.Host)
		return nil
	}

	return cmd
}

func runConnectList(svc *app.ConnectionService, cmd *cobra.Command) error {
	result := svc.ListConnections(cmd.Context())
	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", result.Error)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PATH\tTYPE\tHOST\tENV")
	for _, info := range result.Data {
		env := info.Env
		if env == "" {
			env = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", info.Path, info.Type, info.Host, env)
	}
	return w.Flush()
}

func runConnectPing(svc *app.ConnectionService, cmd *cobra.Command, path string) error {
	ctx := cmd.Context()

	connectResult := svc.Connect(ctx, path)
	if connectResult.Error != nil {
		fmt.Fprintf(os.Stderr, "Error connecting: %v\n", connectResult.Error)
		os.Exit(1)
	}

	pingResult := svc.Ping(ctx, path)
	if pingResult.Error != nil {
		fmt.Fprintf(os.Stderr, "Error pinging: %v\n", pingResult.Error)
		os.Exit(1)
	}

	fmt.Printf("Connected: %s\n", path)
	if pingResult.Data.ServerVersion != "" {
		fmt.Printf("Server version: %s\n", pingResult.Data.ServerVersion)
	}

	disconnectResult := svc.Disconnect(ctx, path)
	if disconnectResult.Error != nil {
		fmt.Fprintf(os.Stderr, "Warning: disconnect error: %v\n", disconnectResult.Error)
	}

	return nil
}
