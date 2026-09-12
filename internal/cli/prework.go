package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var PreworkCmd = &cobra.Command{
	Use:   "prework",
	Short: "Sync local branch with remote before starting work",
	Long: `Checks git status, fetches from remote, and fast-forward pulls if behind.
Aborts if there are local changes, no upstream, or diverged history.`,
	RunE: runPrework,
}

func init() {
	PreworkCmd.Flags().Bool("dry-run", false, "only check status, don't pull")
	PreworkCmd.Flags().Bool("rebase", false, "use --rebase instead of --ff-only when pulling (default is --ff-only)")
}

func runPrework(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	useRebase, _ := cmd.Flags().GetBool("rebase")

	// 1. Must be inside a git repo
	if err := runGit(nil, "rev-parse", "--is-inside-work-tree"); err != nil {
		return fmt.Errorf("not inside a git repository")
	}

	// 2. Abort if dirty (includes untracked files, unstaged, staged)
	if dirty, statusOut, err := isDirty(); err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	} else if dirty {
		cmd.SetOut(cmd.ErrOrStderr())
		cmd.Println("✖ abort: local changes detected. Commit or stash before prework:")
		cmd.Println(statusOut)
		return fmt.Errorf("dirty worktree")
	}

	// 3. Check upstream exists
	branch, err := getBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	upstream, err := getUpstream()
	if err != nil {
		cmd.Println("✖ no upstream configured for branch", branch)
		cmd.Println("  fix with: git push -u origin", branch)
		return fmt.Errorf("no upstream for branch %s", branch)
	}
	cmd.Printf("branch %s tracking %s\n", branch, upstream)

	// 4. Fetch
	cmd.Println("→ fetching origin --prune ...")
	if err := runGit(cmd, "fetch", "origin", "--prune"); err != nil {
		return fmt.Errorf("git fetch failed (offline?): %w", err)
	}

	// 5. Compare HEAD vs @{u}
	ahead, behind, err := aheadBehind()
	if err != nil {
		return fmt.Errorf("failed to compare HEAD with upstream: %w", err)
	}

	cmd.Printf("status: ahead %s, behind %s\n", ahead, behind)

	if ahead == "0" && behind == "0" {
		cmd.Println("✓ already up to date.")
		return nil
	}

	if ahead != "0" && behind == "0" {
		cmd.Printf("! local branch is ahead by %s commit(s). Nothing to pull — you need to push.\n", ahead)
		return nil
	}

	if ahead != "0" && behind != "0" {
		cmd.Println("✖ diverged: local and remote have different commits.")
		cmd.Println("  prework will not merge/rebase automatically (v1).")
		cmd.Println("  resolve manually, e.g.:")
		cmd.Println("    git status")
		cmd.Println("    git log --oneline --graph --left-right HEAD...@{u}")
		cmd.Println("    git rebase origin/"+branch+"  # or: git merge origin/"+branch)
		return fmt.Errorf("diverged history")
	}

	// behind > 0, ahead == 0 => fast-forward pull
	if dryRun {
		cmd.Printf("dry-run: would pull %s commit(s) from %s\n", behind, upstream)
		return nil
	}

	cmd.Printf("→ pulling %s commit(s) from %s ...\n", behind, upstream)
	var pullArgs []string
	if useRebase {
		pullArgs = []string{"pull", "--rebase"}
	} else {
		pullArgs = []string{"pull", "--ff-only"}
	}
	if err := runGit(cmd, pullArgs...); err != nil {
		cmd.Println("✖ pull failed. History may have diverged or fast-forward not possible.")
		cmd.Println("  try: git pull --rebase  or  git status")
		return fmt.Errorf("git pull failed: %w", err)
	}

	// Show new HEAD
	if head, err := gitOutput("log", "--oneline", "-1"); err == nil {
		cmd.Println("✓ synced to:", strings.TrimSpace(head))
	} else {
		cmd.Println("✓ pull complete.")
	}
	return nil
}


