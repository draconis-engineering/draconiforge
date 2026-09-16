package runner

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Script represents a discoverable sh/ps1 script
type Script struct {
	Name        string // without extension, e.g. "file_sorter"
	FileName    string // e.g. "file_sorter.sh"
	Path        string // absolute path
	Ext         string // .sh .ps1 .bash
	SourceDir   string // which dir it was found in
	Description string // first comment line
}

// Discover finds all sh/ps1 scripts in known locations
func Discover() ([]Script, error) {
	dirs := scriptDirs()
	seen := map[string]bool{} // dedup by Name, first wins (higher priority first)
	var out []Script

	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			ext := strings.ToLower(filepath.Ext(name))
			if ext != ".sh" && ext != ".ps1" && ext != ".bash" {
				continue
			}
			base := strings.TrimSuffix(name, ext)
			// dedup: keep first occurrence (priority order matters)
			if seen[base] {
				continue
			}
			seen[base] = true
			abs := filepath.Join(d, name)
			desc := readDescription(abs)
			out = append(out, Script{
				Name:        base,
				FileName:    name,
				Path:        abs,
				Ext:         ext,
				SourceDir:   d,
				Description: desc,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Resolve finds a script by name (with or without extension)
func Resolve(name string) (*Script, error) {
	scripts, err := Discover()
	if err != nil {
		return nil, err
	}
	// allow name with ext
	clean := strings.TrimSpace(name)
	ext := strings.ToLower(filepath.Ext(clean))
	base := clean
	if ext == ".sh" || ext == ".ps1" || ext == ".bash" {
		base = strings.TrimSuffix(clean, ext)
	}
	for _, s := range scripts {
		if s.Name == base || s.FileName == clean {
			// copy to avoid loop var alias
			cp := s
			return &cp, nil
		}
	}
	// also try case-insensitive
	lower := strings.ToLower(base)
	for _, s := range scripts {
		if strings.ToLower(s.Name) == lower {
			cp := s
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("script %q not found (run `draconiforge run --list` to see available)", name)
}

// scriptDirs in priority order (first = highest priority for dedup)
func scriptDirs() []string {
	var dirs []string

	// 1. ./scripts in current working directory (project-local)
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, "scripts"))
		// also try git root/scripts if different
		if root := gitRoot(cwd); root != "" && root != cwd {
			p := filepath.Join(root, "scripts")
			if p != filepath.Join(cwd, "scripts") {
				dirs = append(dirs, p)
			}
		}
	}

	// 2. XDG config: $XDG_CONFIG_HOME/draconiforge/scripts or ~/.config/draconiforge/scripts
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		dirs = append(dirs, filepath.Join(xdg, "draconiforge", "scripts"))
	} else if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".config", "draconiforge", "scripts"))
	}

	// 3. XDG data: ~/.local/share/draconiforge/scripts (where `make install` copies)
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "draconiforge", "scripts"))
	}
	if xdgData := os.Getenv("XDG_DATA_HOME"); xdgData != "" {
		dirs = append(dirs, filepath.Join(xdgData, "draconiforge", "scripts"))
	}

	// 4. Next to executable: <bindir>/../share/draconiforge/scripts
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		dirs = append(dirs, filepath.Join(exeDir, "..", "share", "draconiforge", "scripts"))
		dirs = append(dirs, filepath.Join(exeDir, "scripts"))
	}

	// 5. Fallback: PREFIX/share if env set (for packaging)
	if prefix := os.Getenv("PREFIX"); prefix != "" {
		dirs = append(dirs, filepath.Join(prefix, "share", "draconiforge", "scripts"))
	}

	// dedup dirs
	seen := map[string]bool{}
	var uniq []string
	for _, d := range dirs {
		clean := filepath.Clean(d)
		if !seen[clean] {
			seen[clean] = true
			uniq = append(uniq, clean)
		}
	}
	return uniq
}

func gitRoot(start string) string {
	c := exec.Command("git", "rev-parse", "--show-toplevel")
	c.Dir = start
	out, err := c.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func readDescription(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	// read up to 5 lines, find first comment
	for i := 0; i < 5 && sc.Scan(); i++ {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "#!") {
			continue
		}
		if strings.HasPrefix(line, "#") {
			desc := strings.TrimSpace(strings.TrimPrefix(line, "#"))
			// skip empty or generic
			if desc != "" {
				if len(desc) > 80 {
					desc = desc[:77] + "..."
				}
				return desc
			}
		} else if line != "" {
			break
		}
	}
	return ""
}

// Executor decides shell for script

type RunOptions struct {
	Args   []string
	DryRun bool
	Shell  string // override, e.g. "bash" or "pwsh"
	Stdout *os.File
	Stderr *os.File
	Stdin  *os.File
}

// Run executes script with args, returns exit error if any
func Run(s *Script, opts RunOptions) error {
	if opts.DryRun {
		shell, shellArgs := shellFor(s, opts.Shell)
		fmt.Printf("dry-run: would run: %s %s %s\n", shell, strings.Join(shellArgs, " "), s.Path)
		if len(opts.Args) > 0 {
			fmt.Printf("  args: %v\n", opts.Args)
		}
		return nil
	}

	shell, shellArgs := shellFor(s, opts.Shell)
	// shellArgs already contains -File for ps1 etc.
	args := append(shellArgs, s.Path)
	args = append(args, opts.Args...)

	cmd := exec.Command(shell, args...)
	// inherit stdio by default
	if opts.Stdin != nil {
		cmd.Stdin = opts.Stdin
	} else {
		cmd.Stdin = os.Stdin
	}
	if opts.Stdout != nil {
		cmd.Stdout = opts.Stdout
	} else {
		cmd.Stdout = os.Stdout
	}
	if opts.Stderr != nil {
		cmd.Stderr = opts.Stderr
	} else {
		cmd.Stderr = os.Stderr
	}
	// run from script's dir? keep cwd as caller's cwd for file_sorter expectation
	return cmd.Run()
}

func shellFor(s *Script, override string) (string, []string) {
	if override != "" {
		return override, nil
	}
	switch s.Ext {
	case ".ps1":
		// prefer pwsh, fallback to powershell on Windows
		if _, err := exec.LookPath("pwsh"); err == nil {
			return "pwsh", []string{"-File"}
		}
		if runtime.GOOS == "windows" {
			if _, err := exec.LookPath("powershell"); err == nil {
				return "powershell", []string{"-File"}
			}
		}
		// fallback try pwsh anyway
		return "pwsh", []string{"-File"}
	default: // .sh .bash
		if _, err := exec.LookPath("bash"); err == nil {
			return "bash", nil
		}
		// fallback to sh
		return "sh", nil
	}
}

// ScriptDirs exposes for --verbose / debugging
func ScriptDirs() []string { return scriptDirs() }
