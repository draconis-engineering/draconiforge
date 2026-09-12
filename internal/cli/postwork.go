package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var PostworkCmd = &cobra.Command{
	Use:   "postwork",
	Short: "Commit and push local work after finishing",
	Long: `Checks for local changes, optionally commits them, then pushes to remote.
By default aborts if dirty and no -m message is given (v1 safe mode).
Use -m to stage and commit in one go, or commit manually first.`,
	RunE: runPostwork,
}

func init() {
	PostworkCmd.Flags().StringP("message", "m", "", "commit message (stages all changes with -a and commits)")
	PostworkCmd.Flags().BoolP("all", "a", false, "with -m, stage all tracked+untracked changes before committing (git add -A)")
	PostworkCmd.Flags().Bool("dry-run", false, "show what would be committed/pushed without doing it")
	PostworkCmd.Flags().Bool("no-push", false, "commit but don't push")
	PostworkCmd.Flags().Bool("force", false, "push even if diverged (uses --force-with-lease, dangerous)")
}

func runPostwork(cmd *cobra.Command, args []string) error {
	msg, _ := cmd.Flags().GetString("message")
	addAll, _ := cmd.Flags().GetBool("all")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	noPush, _ := cmd.Flags().GetBool("no-push")
	force, _ := cmd.Flags().GetBool("force")

	if err := runGit(nil, "rev-parse", "--is-inside-work-tree"); err != nil {
		return fmt.Errorf("not inside a git repository")
	}

	dirty, statusOut, err := isDirty()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	branch, err := getBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Handle dirty worktree
	if dirty {
		if msg == "" {
			cmd.SetOut(cmd.ErrOrStderr())
			cmd.Println("✖ abort: local changes detected and no -m message given:")
			cmd.Println(statusOut)
			cmd.Println("")
			cmd.Println("  Commit manually, or run:")
			cmd.Println("    draconiforge postwork -m \"your message\"      # stage tracked, commit")
			cmd.Println("    draconiforge postwork -a -m \"your message\"   # stage all, commit")
			cmd.Println("  Or: git add -A && git commit -m \"msg\" then postwork again")
			return fmt.Errorf("dirty worktree, no commit message")
		}
		// Have message - commit it
		if dryRun {
			cmd.Printf("dry-run: would stage and commit with message %q\n", msg)
			if addAll {
				cmd.Println("dry-run: would run: git add -A")
			} else {
				cmd.Println("dry-run: would run: git add -u (tracked only, use -a for untracked)")
			}
			cmd.Printf("dry-run: would run: git commit -m %q\n", msg)
			cmd.Println("dry-run: would then push (skipping fetch in dry-run)")
			if noPush {
				cmd.Println("dry-run: --no-push, would not push")
			} else {
				// Show what push would look like without needing fetch
				if up, err := getUpstream(); err == nil {
					cmd.Printf("dry-run: would push 1 commit(s) to %s (after commit)\n", up)
				} else {
					cmd.Printf("dry-run: would push and set upstream: git push -u origin %s\n", branch)
				}
			}
			return nil
		} else {
			if addAll {
				cmd.Println("→ staging all changes (git add -A)...")
				if err := runGit(cmd, "add", "-A"); err != nil {
					return fmt.Errorf("git add failed: %w", err)
				}
			} else {
				// Default: stage tracked modifications/deletions, but not untracked
				// Check if there are tracked changes to stage
				cmd.Println("→ staging tracked changes (git add -u, use -a for untracked)...")
				if err := runGit(cmd, "add", "-u"); err != nil {
					return fmt.Errorf("git add failed: %w", err)
				}
				// If still dirty with untracked, warn but continue to commit tracked part
				if stillDirty, untrackedOut, _ := isDirty(); stillDirty {
					// Check if untracked remains (lines starting with ??)
					hasUntracked := false
					for _, line := range strings.Split(untrackedOut, "\n") {
						if strings.HasPrefix(line, "??") {
							hasUntracked = true
							break
						}
					}
					if hasUntracked {
						cmd.Println("! untracked files remain (use -a to include them):")
						cmd.Println(untrackedOut)
					}
				}
			}
			cmd.Printf("→ committing with message %q...\n", msg)
			if err := runGit(cmd, "commit", "-m", msg); err != nil {
				// Could be nothing to commit if user already committed
				if stillDirty, _, _ := isDirty(); !stillDirty {
					cmd.Println("! nothing to commit (maybe already committed)")
				} else {
					return fmt.Errorf("git commit failed: %w", err)
				}
			} else {
				cmd.Println("✓ committed")
			}
		}
	} else {
		cmd.Println("✓ working tree clean, nothing to commit")
	}

	if noPush {
		cmd.Println("→ --no-push, skipping push")
		return nil
	}

	// Check remote / upstream, then fetch to detect divergence
	upstream, upErr := getUpstream()
	if upErr != nil {
		// No upstream - try to set it on push
		cmd.Printf("! no upstream for branch %s\n", branch)
		// Check if any remote exists
		remotes, _ := gitOutput("remote")
		if strings.TrimSpace(remotes) == "" {
			cmd.Println("✖ no git remote configured. Add one:")
			cmd.Println("  git remote add origin <url>")
			return fmt.Errorf("no remote")
		}
		if dryRun {
			cmd.Printf("dry-run: would push and set upstream: git push -u origin %s\n", branch)
			return nil
		}
		cmd.Printf("→ pushing and setting upstream: git push -u origin %s...\n", branch)
		if err := runGit(cmd, "push", "-u", "origin", branch); err != nil {
			return fmt.Errorf("git push failed: %w", err)
		}
		cmd.Println("✓ pushed and set upstream")
		return nil
	}

	// Has upstream - fetch and check divergence before push
	cmd.Println("→ fetching origin --prune...")
	if err := runGit(cmd, "fetch", "origin", "--prune"); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}
	ahead, behind, err := aheadBehind()
	if err != nil {
		return fmt.Errorf("failed to compare with upstream: %w", err)
	}
	cmd.Printf("status: branch %s tracking %s, ahead %s, behind %s\n", branch, upstream, ahead, behind)

	if ahead == "0" && behind == "0" && !dirty {
		cmd.Println("✓ already up to date, nothing to push")
		return nil
	}
	if ahead != "0" && behind != "0" && !force {
		cmd.Println("✖ diverged: local and remote have different commits.")
		cmd.Println("  postwork will not force push without --force.")
		cmd.Println("  Resolve with: git pull --rebase, or draconiforge prework")
		if force {
			cmd.Println("  Or: draconiforge postwork --force (uses --force-with-lease)")
		}
		return fmt.Errorf("diverged history")
	}
	if behind != "0" && ahead == "0" {
		cmd.Println("✖ remote is ahead, you need to pull first:")
		cmd.Println("  draconiforge prework")
		return fmt.Errorf("behind remote")
	}
	if ahead == "0" {
		// Could be we just committed, now ahead should be 1, but if dry-run we skipped commit
		if dryRun {
			cmd.Println("dry-run: would push now")
			return nil
		}
		cmd.Println("! nothing to push (no commits ahead)")
		return nil
	}

	if dryRun {
		cmd.Printf("dry-run: would push %s commit(s) to %s\n", ahead, upstream)
		return nil
	}

	cmd.Printf("→ pushing %s commit(s) to %s...\n", ahead, upstream)
	args_ := []string{"push"}
	if force {
		args_ = []string{"push", "--force-with-lease"}
	}
	if err := runGit(cmd, args_...); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}
	cmd.Println("✓ pushed")
	return nil
}
