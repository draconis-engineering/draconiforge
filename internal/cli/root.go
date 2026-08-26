package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "my-cli",
	Short: "my-cli is a powerful command-line tool",
	Long:  `A fast and flexible CLI built with Go.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior when no subcommand is specified
		_ = cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}
