package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var DoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your git setup for common issues",
	Long: `Run health checks on your git config, remotes, and repo.
Safe, read-only — explains failures with fixes, like prework/postwork.`,
	RunE: runDoctor,
}

func init() {
	DoctorCmd.Flags().Bool("check-ssh", false, "also test SSH to GitHub (may hang offline)")
	DoctorCmd.Flags().Bool("verbose", false, "show all checks including passing ones")
}

type checkResult struct {
	name   string
	status string // "ok", "warn", "fail"
	msg    string
	fix    string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	checkSSH, _ := cmd.Flags().GetBool("check-ssh")

	var results []checkResult
	warn := 0
	fail := 0

	add := func(r checkResult) {
		results = append(results, r)
		if r.status == "fail" {
			fail++
		} else if r.status == "warn" {
			warn++
		}
	}

	// 0. git installed
	if _, err := exec.LookPath("git"); err != nil {
		add(checkResult{"git installed", "fail", "git not found in PATH", "install git: https://git-scm.com/downloads"})
		printResults(cmd, results, verbose)
		return fmt.Errorf("git not installed")
	}
	add(checkResult{"git installed", "ok", "git found", ""})

	// 1. inside repo?
	inside := runGit(nil, "rev-parse", "--is-inside-work-tree") == nil
	if !inside {
		add(checkResult{"inside git repo", "warn", "not inside a git repository (some checks skipped)", "cd to a repo or run draconiforge new"})
		// still check global config
	} else {
		add(checkResult{"inside git repo", "ok", "inside repository", ""})
	}

	// 2. user.name / user.email
	if v, err := gitOutput("config", "user.name"); err != nil || strings.TrimSpace(v) == "" {
		add(checkResult{"git user.name", "fail", "not set", `git config --global user.name "Your Name"`})
	} else {
		add(checkResult{"git user.name", "ok", strings.TrimSpace(v), ""})
	}
	if v, err := gitOutput("config", "user.email"); err != nil || strings.TrimSpace(v) == "" {
		add(checkResult{"git user.email", "fail", "not set", `git config --global user.email "you@example.com"`})
	} else {
		add(checkResult{"git user.email", "ok", strings.TrimSpace(v), ""})
	}

	// 3. pull config
	if v, _ := gitOutput("config", "pull.rebase"); strings.TrimSpace(v) != "" {
		add(checkResult{"pull.rebase", "ok", "pull.rebase=" + strings.TrimSpace(v), ""})
	} else if v, _ := gitOutput("config", "pull.ff"); strings.TrimSpace(v) != "" {
		add(checkResult{"pull.ff", "ok", "pull.ff=" + strings.TrimSpace(v), ""})
	} else {
		add(checkResult{"pull strategy", "warn", "neither pull.rebase nor pull.ff set (git default is merge)", "set one: git config --global pull.ff only  # or: git config --global pull.rebase true"})
	}

	// 4. init.defaultBranch
	if v, _ := gitOutput("config", "init.defaultBranch"); strings.TrimSpace(v) == "" {
		add(checkResult{"init.defaultBranch", "warn", "not set (defaults to master, GitHub uses main)", `git config --global init.defaultBranch main`})
	} else {
		add(checkResult{"init.defaultBranch", "ok", strings.TrimSpace(v), ""})
	}

	// 5. remotes
	remotesOut, _ := gitOutput("remote")
	remotes := strings.Fields(strings.TrimSpace(remotesOut))
	if len(remotes) == 0 {
		if inside {
			add(checkResult{"git remote", "warn", "no remotes configured", "git remote add origin <url>"})
		} else {
			add(checkResult{"git remote", "warn", "no remotes (not in repo)", ""})
		}
	} else {
		add(checkResult{"git remote", "ok", "remotes: " + strings.Join(remotes, ", "), ""})
		// check each remote URL reachable via ls-remote? skip network unless --check-ssh, but try one fetch dry-run quick?
		for _, r := range remotes {
			url, _ := gitOutput("remote", "get-url", r)
			url = strings.TrimSpace(url)
			if url != "" {
				add(checkResult{"remote "+r+" url", "ok", url, ""})
			}
		}
		if inside {
			// upstream check
			if _, err := getUpstream(); err != nil {
				branch, _ := getBranch()
				if branch != "" && branch != "HEAD" {
					add(checkResult{"upstream", "warn", "no upstream for " + branch, "git push -u origin " + branch})
				}
			} else {
				if up, _ := getUpstream(); up != "" {
					add(checkResult{"upstream", "ok", up, ""})
				}
				// fetch check (quick, but may fail offline — warn not fail)
				if err := runGitWithTimeout(5*time.Second, "fetch", "--dry-run"); err != nil {
					add(checkResult{"fetch dry-run", "warn", "fetch failed (offline or auth?): " + err.Error(), "check network / ssh key: ssh -T git@github.com"})
				} else {
					add(checkResult{"fetch dry-run", "ok", "fetch works", ""})
				}
				// ahead/behind
				if ahead, behind, err := aheadBehind(); err == nil {
					if ahead == "0" && behind == "0" {
						add(checkResult{"sync status", "ok", "up to date", ""})
					} else if ahead != "0" && behind == "0" {
						add(checkResult{"sync status", "warn", "ahead by " + ahead + " (need push)", "draconiforge postwork"})
					} else if ahead == "0" && behind != "0" {
						add(checkResult{"sync status", "warn", "behind by " + behind + " (need pull)", "draconiforge prework"})
					} else {
						add(checkResult{"sync status", "fail", "diverged ahead " + ahead + " behind " + behind, "draconiforge prework or git rebase"})
					}
				}
			}
		}
	}

	// 6. .env tracked?
	if inside {
		if out, _ := gitOutput("ls-files", ".env"); strings.TrimSpace(out) != "" {
			add(checkResult{".env tracked", "fail", ".env is tracked by git (secret leak!)", "git rm --cached .env && echo .env >> .gitignore"})
		} else {
			add(checkResult{".env tracked", "ok", ".env not tracked", ""})
		}
		// large files > 10MB in index? quick check via ls-files -s and file size? simpler: check staged files size
		if warnMsg := checkLargeFiles(); warnMsg != "" {
			add(checkResult{"large files", "warn", warnMsg, "use git LFS or rm: git rm --cached <file>"})
		} else {
			add(checkResult{"large files", "ok", "no large files (>10MB) in repo", ""})
		}
		// .gitignore exists?
		if _, err := os.Stat(".gitignore"); err != nil {
			add(checkResult{".gitignore", "warn", ".gitignore not found", "create one: draconiforge new ignores it, or echo \"*.log\" > .gitignore"})
		} else {
			add(checkResult{".gitignore", "ok", ".gitignore exists", ""})
		}
		// dirty check
		if dirty, out, _ := isDirty(); dirty {
			lines := strings.Split(strings.TrimSpace(out), "\n")
			add(checkResult{"working tree", "warn", fmt.Sprintf("dirty (%d files)", len(lines)), "commit or stash: draconiforge postwork -m \"msg\""})
		} else {
			add(checkResult{"working tree", "ok", "clean", ""})
		}
		// branch is not detached?
		if b, _ := getBranch(); b == "HEAD" {
			add(checkResult{"branch", "warn", "detached HEAD", "git switch <branch>"})
		} else if b != "" {
			add(checkResult{"branch", "ok", b, ""})
		}
	}

	// 7. SSH check optional
	if checkSSH {
		if err := runGitWithTimeout(5*time.Second, "ls-remote", "--heads", "origin"); err != nil {
			if strings.Contains(err.Error(), "could not read") || strings.Contains(err.Error(), "Permission denied") {
				add(checkResult{"SSH to origin", "fail", "permission denied", "check ssh key: ssh -T git@github.com ; ssh-add"})
			} else {
				add(checkResult{"SSH to origin", "warn", "ls-remote failed: " + err.Error(), "check network / origin url"})
			}
		} else {
			add(checkResult{"SSH to origin", "ok", "ssh auth works", ""})
		}
	}

	// 8. draconiforge binary in PATH?
	if exe, err := os.Executable(); err == nil {
		add(checkResult{"draconiforge binary", "ok", exe, ""})
	}
	// check .local/bin in PATH
	inPath := false
	for _, p := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		if strings.Contains(p, ".local/bin") {
			inPath = true
			break
		}
	}
	if !inPath {
		add(checkResult{"PATH ~/.local/bin", "warn", "~/.local/bin not in PATH (df alias may not work)", `export PATH="$HOME/.local/bin:$PATH"`})
	}

	printResults(cmd, results, verbose)

	if fail > 0 {
		cmd.Printf("\n✖ %d failing, %d warnings — fix above.\n", fail, warn)
		return fmt.Errorf("doctor found %d failures", fail)
	}
	if warn > 0 {
		cmd.Printf("\n⚠ %d warnings, 0 failures — mostly ok.\n", warn)
		return nil
	}
	cmd.Println("\n✓ all checks passed")
	return nil
}

func printResults(cmd *cobra.Command, results []checkResult, verbose bool) {
	for _, r := range results {
		if !verbose && r.status == "ok" {
			continue
		}
		var icon string
		switch r.status {
		case "ok":
			icon = "✓"
		case "warn":
			icon = "⚠"
		case "fail":
			icon = "✖"
		}
		cmd.Printf("%s %-20s %s\n", icon, r.name, r.msg)
		if r.fix != "" && r.status != "ok" {
			cmd.Printf("  → fix: %s\n", r.fix)
		}
	}
	if !verbose {
		okCount := 0
		for _, r := range results {
			if r.status == "ok" {
				okCount++
			}
		}
		if okCount > 0 {
			cmd.Printf("  (%d passing checks hidden, use --verbose to see)\n", okCount)
		}
	}
}

func runGitWithTimeout(timeout time.Duration, args ...string) error {
	c := exec.Command("git", args...)
	// timeout via channel
	done := make(chan error, 1)
	go func() { done <- c.Run() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		if c.Process != nil {
			_ = c.Process.Kill()
		}
		return fmt.Errorf("timeout after %s", timeout)
	}
}

func checkLargeFiles() string {
	// list tracked files, check size >10MB
	out, err := gitOutput("ls-files")
	if err != nil {
		return ""
	}
	var big []string
	for _, f := range strings.Split(strings.TrimSpace(out), "\n") {
		if f == "" {
			continue
		}
		if info, err := os.Stat(f); err == nil {
			if info.Size() > 10*1024*1024 {
				big = append(big, fmt.Sprintf("%s (%.1fMB)", f, float64(info.Size())/1024/1024))
			}
		}
		// also check path exists via filepath? handle subdirs
		_ = filepath.Base(f)
		if len(big) >= 3 {
			break
		}
	}
	if len(big) > 0 {
		return "large tracked files: " + strings.Join(big, ", ")
	}
	return ""
}
