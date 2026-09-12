package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/draconis-engineering/draconiforge/internal/license"
	"github.com/draconis-engineering/draconiforge/internal/scaffold"
)

var NewCmd = &cobra.Command{
	Use:   "new [project-name]",
	Short: "Scaffold a new project from a language template",
	Long: `Create a new project directory from a template.

Each language has a minimal ready-to-run layout with README, .gitignore, and LICENSE.
Use -l/--license to pick a license (default gplv3 matching this repo).`,
	Example: `  draconiforge new myapp -L go -l mit
  draconiforge new myscript --language shell --license apache-2.0
  draconiforge new mypy -L python -l gplv3 --no-git --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runNew,
}

func init() {
	NewCmd.Flags().StringP("language", "L", "go", "project language (go, python, shell, julia, node)")
	NewCmd.Flags().StringP("license", "l", "gplv3", fmt.Sprintf("license (%s)", strings.Join(license.ValidLicenses, ", ")))
	NewCmd.Flags().String("author", "", "author name for LICENSE (default: git config user.name or $USER)")
	NewCmd.Flags().Bool("no-git", false, "skip git init and initial commit")
	NewCmd.Flags().Bool("dry-run", false, "show what would be created without writing")
	NewCmd.Flags().Bool("force", false, "overwrite existing directory if it exists")
}

func runNew(cmd *cobra.Command, args []string) error {
	arg := strings.TrimSpace(args[0])
	// Support paths like ./myapp or /tmp/myapp: dest is the full path, projectName is base
	cleanArg := filepath.Clean(arg)
	projectName := filepath.Base(cleanArg)
	lang, _ := cmd.Flags().GetString("language")
	lic, _ := cmd.Flags().GetString("license")
	authorFlag, _ := cmd.Flags().GetString("author")
	noGit, _ := cmd.Flags().GetBool("no-git")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	// Resolve author
	author := authorFlag
	if author == "" {
		author = gitAuthorName()
	}
	if author == "" {
		author = os.Getenv("USER")
		if author == "" {
			author = os.Getenv("USERNAME")
		}
	}
	if author == "" {
		author = "Author"
	}

	// Resolve dest path - allow both "myapp" and "path/to/myapp"
	dest := cleanArg
	if !filepath.IsAbs(dest) {
		cwd, _ := os.Getwd()
		dest = filepath.Join(cwd, cleanArg)
	}

	opts := scaffold.Options{
		ProjectName: projectName,
		Path:        dest,
		Language:    lang,
		License:     lic,
		Author:      author,
		Year:        time.Now().Year(),
		NoGit:       noGit,
		DryRun:      dryRun,
		Force:       force,
	}

	if err := scaffold.Scaffold(opts); err != nil {
		return err
	}
	return nil
}
