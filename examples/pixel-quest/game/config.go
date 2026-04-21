package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
)

// controlNames are all the physical controls the platform layer can emit.
// These must match the keys accepted by each native backend.
var controlNames = []string{
	"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m",
	"n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
	"up", "down", "left", "right",
	"space", "enter", "escape", "backspace", "tab",
	"lshift", "rshift", "lctrl", "rctrl", "lalt", "ralt",
	"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10", "f11", "f12",
}

// defaultControls maps semantic actions to their default physical controls.
var defaultControls = map[string][]string{
	"move_left":  {"a", "left"},
	"move_right": {"d", "right"},
	"sprint":     {"lshift"},
	"jump":       {"space", "up", "w"},
	"attack":     {"j", "k"},
	"menu_up":    {"up", "w"},
	"menu_down":  {"down", "s"},
	"menu_confirm": {"enter", "space"},
	"menu_back":  {"escape"},
}

// GameConfig holds user-editable settings persisted to disk.
type GameConfig struct {
	KeyMap       map[string][]string `json:"key_map"`
	Volume       float64             `json:"volume"`
	Difficulty   int                 `json:"difficulty"` // 0=easy, 1=normal, 2=hard
	WindowWidth  int                 `json:"window_width,omitempty"`  // 0 = use engine default
	WindowHeight int                 `json:"window_height,omitempty"` // 0 = use engine default
	WindowMode   int                 `json:"window_mode,omitempty"`   // 0=windowed, 1=borderless
	MonitorIndex int                 `json:"monitor_index,omitempty"`
}

func defaultGameConfig() GameConfig {
	m := make(map[string][]string, len(defaultControls))
	for k, v := range defaultControls {
		cp := make([]string, len(v))
		copy(cp, v)
		m[k] = cp
	}
	return GameConfig{
		KeyMap:     m,
		Volume:     0.8,
		Difficulty: 1,
	}
}

var configPath = func() string {
	// Store next to the binary / working dir for simplicity.
	return filepath.Join("assets", "config.json")
}

func loadGameConfig() (GameConfig, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultGameConfig(), nil
		}
		return GameConfig{}, err
	}
	var cfg GameConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultGameConfig(), nil
	}
	// Ensure every action has a binding.
	for action, defaults := range defaultControls {
		if _, ok := cfg.KeyMap[action]; !ok {
			cp := make([]string, len(defaults))
			copy(cp, defaults)
			cfg.KeyMap[action] = cp
		}
	}
	return cfg, nil
}

func saveGameConfig(cfg GameConfig) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// applyKeyMap binds actions to controls inside the whisky input system.
func applyKeyMap(ctx *whisky.Context, keyMap map[string][]string) {
	for action, controls := range keyMap {
		ctx.Input.Bind(action, controls...)
	}
}

// color presets for UI.
var (
	uiPanelColor     = geom.RGBA(0.08, 0.08, 0.12, 0.92)
	uiButtonColor    = geom.RGBA(0.18, 0.20, 0.28, 1)
	uiButtonHover    = geom.RGBA(0.28, 0.32, 0.42, 1)
	uiButtonActive   = geom.RGBA(0.38, 0.44, 0.56, 1)
	uiTextColor      = geom.RGBA(0.92, 0.94, 0.96, 1)
	uiTitleColor     = geom.RGBA(0.96, 0.76, 0.20, 1)
	uiDangerColor    = geom.RGBA(0.90, 0.25, 0.20, 1)
	uiSuccessColor   = geom.RGBA(0.25, 0.85, 0.35, 1)
	uiDisabledColor  = geom.RGBA(0.35, 0.35, 0.38, 1)
	uiHighlightColor = geom.RGBA(0.96, 0.76, 0.20, 1)
)

// formatDifficulty returns a human-readable difficulty label.
func formatDifficulty(d int) string {
	switch d {
	case 0:
		return "Easy"
	case 2:
		return "Hard"
	default:
		return "Normal"
	}
}

// formatControls returns a concise string like "A / Left" for a list of controls.
func formatControls(controls []string) string {
	if len(controls) == 0 {
		return "none"
	}
	out := controls[0]
	for _, c := range controls[1:] {
		out += fmt.Sprintf(" / %s", c)
	}
	return out
}

// Common resolutions available for cycling.
var commonResolutions = [][2]int{
	{1280, 720},
	{1600, 900},
	{1920, 1080},
	{2560, 1440},
}

// formatResolution returns a human-readable resolution string.
func formatResolution(w, h int) string {
	if w == 0 || h == 0 {
		return "1280x720"
	}
	return fmt.Sprintf("%dx%d", w, h)
}

// formatWindowMode returns a human-readable window mode label.
func formatWindowMode(mode int) string {
	switch mode {
	case 1:
		return "Borderless"
	default:
		return "Windowed"
	}
}
