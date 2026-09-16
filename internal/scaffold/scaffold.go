package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/draconis-engineering/draconiforge/internal/license"
)

var validLanguages = []string{"go", "python", "shell", "julia", "node"}

type Options struct {
	ProjectName string
	Path        string // absolute or relative dest dir
	Language    string
	License     string
	Author      string
	Year        int
	NoGit       bool
	DryRun      bool
	Force       bool
}

type templateData struct {
	ProjectName string
	ModulePath  string
	Author      string
	Year        int
	License     string
}

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

func IsValidLanguage(lang string) bool {
	l := strings.ToLower(strings.TrimSpace(lang))
	return slices.Contains(validLanguages, l)
}

func NormalizeLanguage(lang string) string {
	return strings.ToLower(strings.TrimSpace(lang))
}

func Validate(opts Options) error {
	if opts.ProjectName == "" {
		return fmt.Errorf("project name required")
	}
	if strings.Contains(opts.ProjectName, "/") || strings.Contains(opts.ProjectName, "\\") || strings.Contains(opts.ProjectName, "..") {
		return fmt.Errorf("invalid project name %q: must not contain path separators or ..", opts.ProjectName)
	}
	if !nameRe.MatchString(opts.ProjectName) {
		return fmt.Errorf("invalid project name %q: must match %s", opts.ProjectName, nameRe.String())
	}
	if !IsValidLanguage(opts.Language) {
		return fmt.Errorf("unknown language %q (valid: %s)", opts.Language, strings.Join(validLanguages, ", "))
	}
	if !license.IsValid(opts.License) {
		return fmt.Errorf("unknown license %q (valid: %s)", opts.License, strings.Join(license.ValidLicenses, ", "))
	}
	return nil
}

func Scaffold(opts Options) error {
	// Normalize
	opts.Language = NormalizeLanguage(opts.Language)
	opts.License = license.Normalize(opts.License)
	if opts.Year == 0 {
		opts.Year = time.Now().Year()
	}
	if err := Validate(opts); err != nil {
		return err
	}

	dest := opts.Path
	if dest == "" {
		dest = opts.ProjectName
	}
	// Safety: clean path
	clean := filepath.Clean(dest)
	if clean == "." || clean == "/" {
		return fmt.Errorf("refusing to scaffold into %q", dest)
	}

	// Check existing dir
	if _, err := os.Stat(dest); err == nil && !opts.Force {
		return fmt.Errorf("destination %q already exists (use --force to overwrite)", dest)
	}
	if _, err := os.Stat(dest); err == nil && opts.Force {
		// Allow overwrite - but ensure not to delete unintended? We'll merge/overwrite files
	}

	if opts.DryRun {
		fmt.Printf("dry-run: would create %q (%s, license=%s) at %s\n", opts.ProjectName, opts.Language, opts.License, dest)
		return dryRunList(opts)
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	data := templateData{
		ProjectName: opts.ProjectName,
		ModulePath:  strings.ToLower(opts.ProjectName),
		Author:      opts.Author,
		Year:        opts.Year,
		License:     opts.License,
	}
	// For Go, module path might want to be just project name; keep simple
	if opts.Language == "go" {
		// If author looks like github user, could be github.com/author/project - but keep simple
		data.ModulePath = strings.ToLower(opts.ProjectName)
	}

	if err := renderDir(opts.Language, dest, data); err != nil {
		return err
	}

	// License file
	if opts.License != "none" {
		licText, err := license.Render(opts.License, license.Data{
			Year:        opts.Year,
			Author:      opts.Author,
			ProjectName: opts.ProjectName,
		})
		if err != nil {
			return err
		}
		licPath := filepath.Join(dest, "LICENSE")
		if err := os.WriteFile(licPath, []byte(licText), 0644); err != nil {
			return fmt.Errorf("write LICENSE: %w", err)
		}
		fmt.Printf("  created LICENSE (%s)\n", opts.License)
	}

	if !opts.NoGit {
		if err := gitInit(dest); err != nil {
			// Non-fatal, warn
			fmt.Printf("! git init failed: %v (you can run git init manually)\n", err)
		} else {
			fmt.Printf("  initialized git repo\n")
		}
	}

	fmt.Printf("✓ created %s (%s, %s) at %s\n", opts.ProjectName, opts.Language, opts.License, dest)
	if !opts.NoGit {
		fmt.Printf("  next: cd %s && draconiforge prework\n", dest)
	}
	return nil
}

func dryRunList(opts Options) error {
	data := templateData{
		ProjectName: opts.ProjectName,
		ModulePath:  strings.ToLower(opts.ProjectName),
		Author:      opts.Author,
		Year:        opts.Year,
		License:     opts.License,
	}
	prefix := fmt.Sprintf("templates/%s", opts.Language)
	var count int
	err := fs.WalkDir(TemplateFS, prefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := strings.CutPrefix(path, prefix+"/")
		// Strip .tmpl suffix
		outRel := strings.TrimSuffix(rel, ".tmpl")
		// Render to check errors
		tmplBytes, err := fs.ReadFile(TemplateFS, path)
		if err != nil {
			return err
		}
		_, err = template.New(path).Parse(string(tmplBytes))
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		// quick render test
		var buf bytes.Buffer
		t := template.Must(template.New(path).Parse(string(tmplBytes)))
		if err := t.Execute(&buf, data); err != nil {
			return fmt.Errorf("render %s: %w", path, err)
		}
		fmt.Printf("  would create %s\n", filepath.Join(opts.Path, outRel))
		count++
		return nil
	})
	if err != nil {
		return err
	}
	if opts.License != "none" {
		fmt.Printf("  would create %s/LICENSE (%s)\n", opts.Path, opts.License)
		count++
	}
	if !opts.NoGit {
		fmt.Printf("  would run: git init + initial commit\n")
	}
	fmt.Printf("  total %d files\n", count)
	return nil
}

func renderDir(lang, dest string, data templateData) error {
	prefix := fmt.Sprintf("templates/%s", lang)
	return fs.WalkDir(TemplateFS, prefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := strings.CutPrefix(path, prefix+"/")
		outRel := strings.TrimSuffix(rel, ".tmpl")
		outPath := filepath.Join(dest, outRel)

		tmplBytes, err := fs.ReadFile(TemplateFS, path)
		if err != nil {
			return err
		}
		tmpl, err := template.New(path).Parse(string(tmplBytes))
		if err != nil {
			return fmt.Errorf("parse template %s: %w", path, err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("render template %s: %w", path, err)
		}

		// Ensure dir exists
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return err
		}
		perm := os.FileMode(0644)
		if strings.HasSuffix(outRel, ".sh") {
			perm = 0755
		}
		if err := os.WriteFile(outPath, buf.Bytes(), perm); err != nil {
			return fmt.Errorf("write %s: %w", outPath, err)
		}
		fmt.Printf("  created %s\n", outRel)
		return nil
	})
}

func gitInit(dest string) error {
	// Check git available
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH")
	}
	cmds := [][]string{
		{"init"},
		{"add", "."},
		{"commit", "-m", "Initial commit from draconiforge"},
	}
	for _, args := range cmds {
		c := exec.Command("git", args...)
		c.Dir = dest
		var stderr bytes.Buffer
		c.Stderr = &stderr
		if err := c.Run(); err != nil {
			// commit may fail if no user config - try to set local config
			if args[0] == "commit" && strings.Contains(stderr.String(), "Author identity unknown") {
				// Set fallback and retry
				exec.Command("git", "-C", dest, "config", "user.email", "draconiforge@localhost").Run()
				exec.Command("git", "-C", dest, "config", "user.name", "draconiforge").Run()
				c2 := exec.Command("git", args...)
				c2.Dir = dest
				var stderr2 bytes.Buffer
				c2.Stderr = &stderr2
				if err2 := c2.Run(); err2 != nil {
					return fmt.Errorf("git %v: %s: %w", args, strings.TrimSpace(stderr2.String()), err2)
				}
				continue
			}
			return fmt.Errorf("git %v: %s: %w", args, strings.TrimSpace(stderr.String()), err)
		}
	}
	return nil
}
