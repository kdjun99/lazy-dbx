package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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
