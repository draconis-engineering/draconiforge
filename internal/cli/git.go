package cli

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func isDirty() (bool, string, error) {
	out, err := gitOutput("status", "--porcelain")
	if err != nil {
		return false, "", err
	}
	trimmed := strings.TrimSpace(out)
	return trimmed != "", out, nil
}

func getBranch() (string, error) {
	out, err := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func getUpstream() (string, error) {
	out, err := gitOutput("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func hasUpstream() bool {
	_, err := getUpstream()
	return err == nil
}

// aheadBehind returns (ahead, behind) as strings, using HEAD...@{u}
// Caller must ensure upstream exists.
func aheadBehind() (string, string, error) {
	counts, err := gitOutput("rev-list", "--left-right", "--count", "HEAD...@{u}")
	if err != nil {
		return "", "", err
	}
	parts := strings.Fields(strings.TrimSpace(counts))
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected rev-list output: %q", counts)
	}
	return parts[0], parts[1], nil
}

func gitOutput(args ...string) (string, error) {
	c := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func runGit(cmd *cobra.Command, args ...string) error {
	c := exec.Command("git", args...)
	var stderr bytes.Buffer
	c.Stderr = &stderr
	if cmd != nil {
		c.Stdout = cmd.OutOrStdout()
	}
	if err := c.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return err
	}
	return nil
}

func gitAuthorName() string {
	out, err := gitOutput("config", "user.name")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
