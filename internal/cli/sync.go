package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var SyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Run prework + commit + push in one go",
	Long: `Convenience for lazy days: fetch/pull, then commit and push.

If dirty and -m given, stages and commits first (like postwork), then pulls (fast-forward) and pushes.
If dirty and no -m, aborts like prework — commit manually or use -m.
Use --rebase to pull with rebase, --dry-run to preview, --no-push to stop before push.`,
	RunE: runSync,
}

func init() {
	SyncCmd.Flags().StringP("message", "m", "", "commit message (stages changes and commits before pull/push)")
	SyncCmd.Flags().BoolP("all", "a", false, "with -m, stage all tracked+untracked (git add -A), else tracked only (-u)")
	SyncCmd.Flags().Bool("dry-run", false, "show what would happen without doing it")
	SyncCmd.Flags().Bool("rebase", false, "pull with --rebase instead of --ff-only")
	SyncCmd.Flags().Bool("no-push", false, "commit/pull but don't push")
	SyncCmd.Flags().Bool("force", false, "push even if diverged (--force-with-lease, dangerous)")
}

func runSync(cmd *cobra.Command, args []string) error {
	msg, _ := cmd.Flags().GetString("message")
	addAll, _ := cmd.Flags().GetBool("all")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	useRebase, _ := cmd.Flags().GetBool("rebase")
	noPush, _ := cmd.Flags().GetBool("no-push")
	force, _ := cmd.Flags().GetBool("force")

	if err := runGit(nil, "rev-parse", "--is-inside-work-tree"); err != nil {
		return fmt.Errorf("not inside a git repository")
	}

	branch, err := getBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// 1. Handle dirty worktree (like postwork) — commit before pull to avoid losing work
	dirty, statusOut, err := isDirty()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}
	if dirty {
		if msg == "" {
			cmd.SetOut(cmd.ErrOrStderr())
			cmd.Println("✖ abort: local changes detected and no -m message given:")
			cmd.Println(statusOut)
			cmd.Println("")
			cmd.Println("  draconiforge sync -m \"your message\"      # stage tracked, commit, then sync")
			cmd.Println("  draconiforge sync -a -m \"your message\"   # stage all, commit, then sync")
			return fmt.Errorf("dirty worktree, no commit message")
		}
		if dryRun {
			cmd.Printf("dry-run: would stage and commit with message %q\n", msg)
			if addAll {
				cmd.Println("dry-run: would run: git add -A")
			} else {
				cmd.Println("dry-run: would run: git add -u")
			}
			cmd.Printf("dry-run: would run: git commit -m %q\n", msg)
		} else {
			if addAll {
				cmd.Println("→ staging all changes (git add -A)...")
				if err := runGit(cmd, "add", "-A"); err != nil {
					return fmt.Errorf("git add failed: %w", err)
				}
			} else {
				cmd.Println("→ staging tracked changes (git add -u, use -a for untracked)...")
				if err := runGit(cmd, "add", "-u"); err != nil {
					return fmt.Errorf("git add failed: %w", err)
				}
				if stillDirty, out, _ := isDirty(); stillDirty {
					hasUntracked := false
					for _, line := range strings.Split(out, "\n") {
						if strings.HasPrefix(line, "??") {
							hasUntracked = true
							break
						}
					}
					if hasUntracked {
						cmd.Println("! untracked files remain (use -a to include them):")
						cmd.Println(out)
					}
				}
			}
			cmd.Printf("→ committing with message %q...\n", msg)
			if err := runGit(cmd, "commit", "-m", msg); err != nil {
				if stillDirty, _, _ := isDirty(); !stillDirty {
					cmd.Println("! nothing to commit (maybe already committed)")
				} else {
					return fmt.Errorf("git commit failed: %w", err)
				}
			} else {
				cmd.Println("✓ committed")
			}
		}
		// after commit, working tree may be clean or still has untracked if -a not used; re-check
		dirty, _, _ = isDirty()
	} else {
		cmd.Println("✓ working tree clean, nothing to commit")
	}

	// 2. Check upstream, then fetch (like prework)
	upstream, upErr := getUpstream()
	if upErr != nil {
		cmd.Printf("! no upstream for branch %s\n", branch)
		remotes, _ := gitOutput("remote")
		if strings.TrimSpace(remotes) == "" {
			cmd.Println("✖ no git remote configured. Add one:")
			cmd.Println("  git remote add origin <url>")
			return fmt.Errorf("no remote")
		}
		if noPush {
			cmd.Println("→ --no-push, skipping push even without upstream")
			return nil
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
	cmd.Printf("branch %s tracking %s\n", branch, upstream)

	cmd.Println("→ fetching origin --prune ...")
	if err := runGit(cmd, "fetch", "origin", "--prune"); err != nil {
		return fmt.Errorf("git fetch failed (offline?): %w", err)
	}

	// after fetch, check ahead/behind — note: ahead may have changed if we committed
	ahead, behind, err := aheadBehind()
	if err != nil {
		return fmt.Errorf("failed to compare HEAD with upstream: %w", err)
	}
	cmd.Printf("status: ahead %s, behind %s\n", ahead, behind)

	if ahead != "0" && behind != "0" && !force {
		cmd.Println("✖ diverged: local and remote have different commits.")
		cmd.Println("  sync will not merge/rebase automatically without --force.")
		cmd.Println("  resolve with: draconiforge prework or git pull --rebase")
		return fmt.Errorf("diverged history")
	}
	if behind != "0" && ahead == "0" {
		// need pull
		if dryRun {
			cmd.Printf("dry-run: would pull %s commit(s) from %s\n", behind, upstream)
		} else {
			cmd.Printf("→ pulling %s commit(s) from %s ...\n", behind, upstream)
			var pullArgs []string
			if useRebase {
				pullArgs = []string{"pull", "--rebase"}
			} else {
				pullArgs = []string{"pull", "--ff-only"}
			}
			if err := runGit(cmd, pullArgs...); err != nil {
				cmd.Println("✖ pull failed.")
				cmd.Println("  try: git pull --rebase or draconiforge prework --rebase")
				return fmt.Errorf("git pull failed: %w", err)
			}
			if head, err := gitOutput("log", "--oneline", "-1"); err == nil {
				cmd.Println("✓ pulled to:", strings.TrimSpace(head))
			} else {
				cmd.Println("✓ pull complete")
			}
			// refresh ahead/behind after pull
			ahead, behind, _ = aheadBehind()
		}
	} else if ahead != "0" && behind == "0" {
		cmd.Printf("! local ahead by %s, will push\n", ahead)
	} else if ahead == "0" && behind == "0" {
		if !dirty {
			cmd.Println("✓ already up to date")
			if noPush {
				return nil
			}
			// if we committed, ahead may be 0 in dry-run (since commit skipped), handle below
		}
	}

	if noPush {
		cmd.Println("→ --no-push, skipping push")
		return nil
	}

	// refresh ahead if we were in dry-run and committed is skipped, simulate 1 ahead
	if dryRun {
		// if we had a message and were dirty, dry-run commit didn't happen, so show what push would do
		if msg != "" && dirty {
			// Actually dirty is now maybe still true in dry-run, but we treat as would be 1 ahead
			cmd.Printf("dry-run: would push 1 commit(s) to %s (after commit)\n", upstream)
			return nil
		}
		if ahead == "0" {
			cmd.Println("dry-run: would check push (nothing to push)")
			return nil
		}
		cmd.Printf("dry-run: would push %s commit(s) to %s\n", ahead, upstream)
		return nil
	}

	// real push if ahead >0
	if ahead == "0" {
		// re-check after commit/pull
		if a, b, err := aheadBehind(); err == nil {
			ahead, behind = a, b
		}
		if ahead == "0" {
			cmd.Println("✓ nothing to push")
			return nil
		}
	}

	cmd.Printf("→ pushing %s commit(s) to %s...\n", ahead, upstream)
	pushArgs := []string{"push"}
	if force {
		pushArgs = []string{"push", "--force-with-lease"}
	}
	if err := runGit(cmd, pushArgs...); err != nil {
		return fmt.Errorf("git push failed: %w", err)
	}
	cmd.Println("✓ pushed")
	return nil
}
