package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultEngineModule = "github.com/IsraelAraujo70/whisky-game-engine"
	DefaultGoVersion    = "1.26.0"
)

type Config struct {
	Name          string `json:"name"`
	Module        string `json:"module"`
	EntryPoint    string `json:"entry_point"`
	VirtualWidth  int    `json:"virtual_width"`
	VirtualHeight int    `json:"virtual_height"`
	TargetFPS     int    `json:"target_fps"`
	PixelPerfect  bool   `json:"pixel_perfect"`
}

func LoadConfig(dir string) (Config, error) {
	path := filepath.Join(dir, "whisky.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse whisky.json: %w", err)
	}

	if cfg.EntryPoint == "" {
		cfg.EntryPoint = "./cmd/game"
	}
	if cfg.TargetFPS == 0 {
		cfg.TargetFPS = 60
	}
	if cfg.VirtualWidth == 0 {
		cfg.VirtualWidth = 320
	}
	if cfg.VirtualHeight == 0 {
		cfg.VirtualHeight = 180
	}

	return cfg, nil
}
