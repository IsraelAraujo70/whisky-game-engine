package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldCreatesStarterProject(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "demo")

	err := Scaffold(ScaffoldOptions{
		Name:              "demo",
		TargetDir:         target,
		ReplaceEnginePath: "/tmp/whisky",
	})
	if err != nil {
		t.Fatalf("scaffold project: %v", err)
	}

	requiredFiles := []string{
		"go.mod",
		"README.md",
		"whisky.json",
		"cmd/game/main.go",
		"game/game.go",
		"assets/README.md",
	}

	for _, path := range requiredFiles {
		if _, err := os.Stat(filepath.Join(target, path)); err != nil {
			t.Fatalf("expected file %s to exist: %v", path, err)
		}
	}

	goMod, err := os.ReadFile(filepath.Join(target, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	if !strings.Contains(string(goMod), "replace github.com/IsraelAraujo70/whisky-game-engine => /tmp/whisky") {
		t.Fatalf("expected go.mod to contain local replace, got:\n%s", string(goMod))
	}
}

func TestFindEngineRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+DefaultEngineModule+"\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	found, ok := FindEngineRoot(nested, DefaultEngineModule)
	if !ok {
		t.Fatal("expected engine root to be found")
	}

	if found != root {
		t.Fatalf("unexpected root: %s", found)
	}
}
