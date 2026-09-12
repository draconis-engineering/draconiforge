package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "draconiforge",
	Aliases: []string{"df"},
	Short: "draconiforge (df) - streamline exhausting dev tasks",
	Long:  `Draconiforge (df) is a CLI to automate tedious dev workflows.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default behavior when no subcommand is specified
		_ = cmd.Help()
	},
}

func init() {
	rootCmd.AddCommand(VersionCmd)
	rootCmd.AddCommand(PreworkCmd)
	rootCmd.AddCommand(PostworkCmd)
	rootCmd.AddCommand(NewCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
