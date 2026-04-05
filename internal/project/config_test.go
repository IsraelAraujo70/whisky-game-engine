package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	content := `{"name":"demo","module":"example.com/demo"}`

	if err := os.WriteFile(filepath.Join(dir, "whisky.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.EntryPoint != "./cmd/game" {
		t.Fatalf("unexpected entrypoint: %s", cfg.EntryPoint)
	}

	if cfg.TargetFPS != 60 {
		t.Fatalf("unexpected target fps: %d", cfg.TargetFPS)
	}
}
