package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "draconiforge",
	Aliases: []string{"forge"},
	Short:   "draconiforge (forge) - streamline exhausting dev tasks",
	Long:    `Draconiforge (forge) is a CLI to automate tedious dev workflows.`,
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
	rootCmd.AddCommand(RunCmd)
	rootCmd.AddCommand(DoctorCmd)
	rootCmd.AddCommand(SyncCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
