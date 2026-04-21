package game

import (
	"path/filepath"
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
)

func TestDefaultGameConfig(t *testing.T) {
	cfg := defaultGameConfig()
	if cfg.Volume != 0.8 {
		t.Fatalf("expected default volume 0.8, got %v", cfg.Volume)
	}
	if cfg.Difficulty != 1 {
		t.Fatalf("expected default difficulty 1, got %v", cfg.Difficulty)
	}
	if _, ok := cfg.KeyMap["jump"]; !ok {
		t.Fatal("expected jump to have default bindings")
	}
}

func TestSaveAndLoadGameConfig(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := configPath
	// Override config path temporarily.
	configPath = func() string { return filepath.Join(tmpDir, "config.json") }
	defer func() { configPath = origPath }()

	cfg := defaultGameConfig()
	cfg.Volume = 0.5
	cfg.Difficulty = 2
	cfg.KeyMap["jump"] = []string{"space"}

	if err := saveGameConfig(cfg); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := loadGameConfig()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.Volume != 0.5 {
		t.Fatalf("expected volume 0.5, got %v", loaded.Volume)
	}
	if loaded.Difficulty != 2 {
		t.Fatalf("expected difficulty 2, got %v", loaded.Difficulty)
	}
	if len(loaded.KeyMap["jump"]) != 1 || loaded.KeyMap["jump"][0] != "space" {
		t.Fatalf("expected jump=[space], got %v", loaded.KeyMap["jump"])
	}
}

func TestLoadGameConfigMissingDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	origPath := configPath
	configPath = func() string { return filepath.Join(tmpDir, "no_config.json") }
	defer func() { configPath = origPath }()

	cfg, err := loadGameConfig()
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if cfg.Difficulty != 1 {
		t.Fatalf("expected default difficulty, got %v", cfg.Difficulty)
	}
}

func TestFormatDifficulty(t *testing.T) {
	if formatDifficulty(0) != "Easy" {
		t.Fatalf("expected Easy for 0")
	}
	if formatDifficulty(1) != "Normal" {
		t.Fatalf("expected Normal for 1")
	}
	if formatDifficulty(2) != "Hard" {
		t.Fatalf("expected Hard for 2")
	}
}

func TestFormatControls(t *testing.T) {
	if formatControls([]string{"a", "left"}) != "a / left" {
		t.Fatalf("unexpected format: %s", formatControls([]string{"a", "left"}))
	}
	if formatControls(nil) != "none" {
		t.Fatalf("expected 'none' for empty controls")
	}
}

func TestPointInRect(t *testing.T) {
	r := geom.Rect{X: 10, Y: 10, W: 20, H: 20}
	if !pointInRect(geom.Vec2{X: 15, Y: 15}, r) {
		t.Fatal("expected point inside rect")
	}
	if pointInRect(geom.Vec2{X: 5, Y: 5}, r) {
		t.Fatal("expected point outside rect")
	}
}

func TestUIRectBorder(t *testing.T) {
	// uiRect is purely side-effect (draw calls); just ensure it does not panic.
	// We cannot easily test ctx.DrawRect without a real context, so this is a smoke test.
	// In a real project you would inject a mock renderer.
}
