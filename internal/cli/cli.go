package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/IsraelAraujo70/whisky-game-engine/internal/project"
)

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	switch args[0] {
	case "new":
		return runNew(args[1:], stdout, stderr)
	case "run":
		return runProject(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(stdout)
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func runNew(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: whisky new <name-or-path>")
		return 1
	}

	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "resolve working directory: %v\n", err)
		return 1
	}

	replacePath, ok := project.FindEngineRoot(workingDir, project.DefaultEngineModule)
	if !ok {
		replacePath = ""
	}

	targetDir := args[0]
	if !filepath.IsAbs(targetDir) {
		targetDir = filepath.Join(workingDir, targetDir)
	}
	targetDir = filepath.Clean(targetDir)

	if err := project.Scaffold(project.ScaffoldOptions{
		Name:              filepath.Base(targetDir),
		TargetDir:         targetDir,
		EngineModule:      project.DefaultEngineModule,
		ReplaceEnginePath: replacePath,
	}); err != nil {
		fmt.Fprintf(stderr, "scaffold project: %v\n", err)
		return 1
	}

	if err := project.FinalizeModule(targetDir); err != nil {
		fmt.Fprintf(stderr, "finalize project module: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "created project at %s\n", targetDir)
	return 0
}

func runProject(args []string, stdout, stderr io.Writer) int {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	cfg, err := project.LoadConfig(dir)
	if err != nil {
		fmt.Fprintf(stderr, "load project config: %v\n", err)
		return 1
	}

	cmd := exec.Command("go", "run", cfg.EntryPoint)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(stderr, "run project: %v\n", err)
		return 1
	}

	return 0
}

func runDoctor(stdout io.Writer) int {
	checks := []struct {
		name string
		ok   bool
		info string
	}{
		{name: "go", ok: hasCommand("go"), info: versionOf("go", "version")},
		{name: "git", ok: hasCommand("git"), info: versionOf("git", "--version")},
		{name: "pkg-config", ok: hasCommand("pkg-config"), info: versionOf("pkg-config", "--version")},
		{name: "sdl3", ok: pkgConfigExists("sdl3"), info: "pkg-config package"},
		{name: "gl", ok: pkgConfigExists("gl"), info: "pkg-config package"},
	}

	for _, check := range checks {
		status := "missing"
		if check.ok {
			status = "ok"
		}
		fmt.Fprintf(stdout, "%-10s %s", check.name+":", status)
		if strings.TrimSpace(check.info) != "" {
			fmt.Fprintf(stdout, " (%s)", strings.TrimSpace(check.info))
		}
		fmt.Fprintln(stdout)
	}

	return 0
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func versionOf(name string, args ...string) string {
	if !hasCommand(name) {
		return ""
	}

	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func pkgConfigExists(name string) bool {
	if !hasCommand("pkg-config") {
		return false
	}

	return exec.Command("pkg-config", "--exists", name).Run() == nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "whisky commands:")
	fmt.Fprintln(w, "  whisky new <name>   create a new Whisky game project")
	fmt.Fprintln(w, "  whisky run [dir]    run a Whisky project from its root")
	fmt.Fprintln(w, "  whisky doctor       inspect local prerequisites")
}
