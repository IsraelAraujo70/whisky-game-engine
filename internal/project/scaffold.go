package project

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	projecttemplate "github.com/IsraelAraujo70/whisky-game-engine/internal/template"
)

type ScaffoldOptions struct {
	Name              string
	TargetDir         string
	Module            string
	EngineModule      string
	ReplaceEnginePath string
}

type templateData struct {
	Name              string
	Title             string
	Module            string
	EngineModule      string
	ReplaceEnginePath string
	GoVersion         string
}

func Scaffold(opts ScaffoldOptions) error {
	name := slugify(opts.Name)
	if name == "" {
		return fmt.Errorf("project name is required")
	}

	if opts.TargetDir == "" {
		opts.TargetDir = name
	}
	if opts.Module == "" {
		opts.Module = "github.com/IsraelAraujo70/" + name
	}
	if opts.EngineModule == "" {
		opts.EngineModule = DefaultEngineModule
	}

	if err := ensureEmptyDir(opts.TargetDir); err != nil {
		return err
	}

	data := templateData{
		Name:              name,
		Title:             titleCase(name),
		Module:            opts.Module,
		EngineModule:      opts.EngineModule,
		ReplaceEnginePath: filepath.ToSlash(opts.ReplaceEnginePath),
		GoVersion:         DefaultGoVersion,
	}

	if err := os.MkdirAll(filepath.Join(opts.TargetDir, "assets"), 0o755); err != nil {
		return err
	}

	if err := writeTemplates(opts.TargetDir, data); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(opts.TargetDir, "assets", "README.md"), []byte("# Assets\n"), 0o644)
}

func FindEngineRoot(startDir, engineModule string) (string, bool) {
	current := startDir

	for {
		modulePath, ok := readModulePath(filepath.Join(current, "go.mod"))
		if ok && modulePath == engineModule {
			return current, true
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func FinalizeModule(targetDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeTemplates(targetDir string, data templateData) error {
	return fs.WalkDir(projecttemplate.Files, "files", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}

		raw, err := fs.ReadFile(projecttemplate.Files, path)
		if err != nil {
			return err
		}

		relative := strings.TrimPrefix(path, "files/")
		output := strings.TrimSuffix(relative, ".tmpl")
		if err := os.MkdirAll(filepath.Dir(filepath.Join(targetDir, output)), 0o755); err != nil {
			return err
		}

		rendered, err := renderTemplate(string(raw), data)
		if err != nil {
			return fmt.Errorf("render %s: %w", path, err)
		}

		return os.WriteFile(filepath.Join(targetDir, output), []byte(rendered), 0o644)
	})
}

func renderTemplate(raw string, data templateData) (string, error) {
	tpl, err := template.New("file").Parse(raw)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func ensureEmptyDir(targetDir string) error {
	info, err := os.Stat(targetDir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s exists and is not a directory", targetDir)
		}

		entries, err := os.ReadDir(targetDir)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf("%s is not empty", targetDir)
		}
		return nil
	}

	if !os.IsNotExist(err) {
		return err
	}

	return os.MkdirAll(targetDir, 0o755)
}

func readModulePath(goModPath string) (string, bool) {
	raw, err := os.ReadFile(goModPath)
	if err != nil {
		return "", false
	}

	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), true
		}
	}

	return "", false
}

func slugify(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "-")
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-':
			return r
		default:
			return -1
		}
	}, value)
}

func titleCase(value string) string {
	parts := strings.Split(value, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}

	return strings.Join(parts, " ")
}
