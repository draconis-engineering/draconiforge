package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/draconis-engineering/draconiforge/internal/runner"
)

var RunCmd = &cobra.Command{
	Use:   "run [script] [-- script_args...]",
	Short: "Run a sh/ps1 script through draconiforge",
	Long: `Discover and run scripts from ./scripts, ~/.config/draconiforge/scripts,
or ~/.local/share/draconiforge/scripts (installed via make install).

Examples:
  draconiforge run --list
  draconiforge run file_sorter
  draconiforge run git_tracker -- --help
  draconiforge run git_pullall --dry-run`,
	Args:              cobra.ArbitraryArgs,
	DisableFlagParsing: false,
	RunE:              runScript,
}

func init() {
	RunCmd.Flags().Bool("list", false, "list available scripts")
	RunCmd.Flags().Bool("dry-run", false, "show what would run without executing")
	RunCmd.Flags().String("shell", "", "override shell (bash, sh, pwsh, powershell)")
	RunCmd.Flags().Bool("verbose", false, "show search dirs")
	// completion for script names
	RunCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		scripts, _ := runner.Discover()
		var out []string
		for _, s := range scripts {
			if strings.HasPrefix(s.Name, toComplete) {
				out = append(out, fmt.Sprintf("%s\t%s", s.Name, s.Description))
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}

func runScript(cmd *cobra.Command, args []string) error {
	list, _ := cmd.Flags().GetBool("list")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	shell, _ := cmd.Flags().GetString("shell")
	verbose, _ := cmd.Flags().GetBool("verbose")

	if verbose {
		fmt.Fprintln(cmd.OutOrStdout(), "Search dirs:")
		for _, d := range runner.ScriptDirs() {
			// only show existing?
			if _, err := os.Stat(d); err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %s\n", d)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "    %s (not found)\n", d)
			}
		}
		fmt.Fprintln(cmd.OutOrStdout(), "")
	}

	scripts, err := runner.Discover()
	if err != nil {
		return err
	}

	// --list or no args -> list
	if list || len(args) == 0 {
		if len(scripts) == 0 {
			cmd.Println("No scripts found.")
			cmd.Println("Searched:")
			for _, d := range runner.ScriptDirs() {
				cmd.Printf("  - %s\n", d)
			}
			cmd.Println("\nAdd scripts to ~/.config/draconiforge/scripts/ or ./scripts/")
			return nil
		}
		cmd.Printf("Available scripts (%d):\n", len(scripts))
		for _, s := range scripts {
			desc := s.Description
			if desc != "" {
				desc = " — " + desc
			}
			// show name, file, source
			cmd.Printf("  %-18s %-20s %s%s\n", s.Name, "("+s.FileName+")", s.SourceDir, desc)
		}
		cmd.Println("\nRun with: draconiforge run <name> [-- args...]")
		return nil
	}

	// first arg is script name, rest are script args
	scriptName := args[0]
	scriptArgs := args[1:]

	// handle `--` separator if user passed it: cobra already strips it but keep handling
	if len(scriptArgs) > 0 && scriptArgs[0] == "--" {
		scriptArgs = scriptArgs[1:]
	}

	s, err := runner.Resolve(scriptName)
	if err != nil {
		return err
	}

	cmd.Printf("→ running %s (%s) via %s ...\n", s.Name, s.Path, s.Ext)

	// delegate to runner, it prints dry-run details itself
	if err := runner.Run(s, runner.RunOptions{Args: scriptArgs, DryRun: dryRun, Shell: shell}); err != nil {
		// propagate exit code-ish error
		return fmt.Errorf("script %q failed: %w", s.Name, err)
	}
	return nil
}
