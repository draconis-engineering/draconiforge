package cli

import "github.com/spf13/cobra"

// VersionCmd represents the version command
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Prints the version number of the Draconiforge CLI developed by Draconis Engineering.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("v1.0.0")
	},
}
