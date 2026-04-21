package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
)

const maxSaveSlots = 3

// saveSlot represents one player save.
type saveSlot struct {
	SlotIndex       int               `json:"slot_index"`
	LevelIndex      int               `json:"level_index"`
	PlayerHP        int               `json:"player_hp"`
	Score           scoreState        `json:"score"`
	UnlockedLevels  []bool            `json:"unlocked_levels"`
	KeyMap          map[string][]string `json:"key_map"`
	Volume          float64           `json:"volume"`
	Difficulty      int               `json:"difficulty"`
	LastSaveTime    string            `json:"last_save_time"`
}

// saveData holds the global save structure (slots + settings).
type saveData struct {
	Slots          []saveSlot        `json:"slots"`
	LastUsedSlot   int               `json:"last_used_slot"`
	UnlockedLevels []bool            `json:"unlocked_levels"`
}

func newSaveData() *saveData {
	return &saveData{
		Slots:          make([]saveSlot, 0, maxSaveSlots),
		LastUsedSlot:   -1,
		UnlockedLevels: []bool{true, false, false},
	}
}

func saveFilePath() string {
	return filepath.Join("assets", "saves.json")
}

func loadSaveData() (*saveData, error) {
	path := saveFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newSaveData(), nil
		}
		return nil, err
	}
	var sd saveData
	if err := json.Unmarshal(data, &sd); err != nil {
		return newSaveData(), nil
	}
	// Ensure unlocked levels array exists.
	if len(sd.UnlockedLevels) < len(allLevels) {
		sd.UnlockedLevels = append(sd.UnlockedLevels, make([]bool, len(allLevels)-len(sd.UnlockedLevels))...)
	}
	for i := range sd.Slots {
		if len(sd.Slots[i].UnlockedLevels) < len(allLevels) {
			sd.Slots[i].UnlockedLevels = append(sd.Slots[i].UnlockedLevels, make([]bool, len(allLevels)-len(sd.Slots[i].UnlockedLevels))...)
		}
	}
	return &sd, nil
}

func saveSaveData(sd *saveData) error {
	path := saveFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// createSnapshot builds a save slot from the current game state.
func (g *pixelQuest) createSnapshot(slotIndex int) saveSlot {
	unlocked := make([]bool, len(allLevels))
	copy(unlocked, g.saveData.UnlockedLevels)
	for i := range unlocked {
		if i <= g.currentLevel {
			unlocked[i] = true
		}
	}
	hp := playerMaxHP
	if g.playerHealth != nil {
		hp = g.playerHealth.Current
	}
	return saveSlot{
		SlotIndex:      slotIndex,
		LevelIndex:     g.currentLevel,
		PlayerHP:       hp,
		Score:          *g.score,
		UnlockedLevels: unlocked,
		KeyMap:         g.config.KeyMap,
		Volume:         g.config.Volume,
		Difficulty:     g.config.Difficulty,
		LastSaveTime:   time.Now().Format("2006-01-02 15:04"),
	}
}

// applySnapshot restores game state from a save slot.
func (g *pixelQuest) applySnapshot(slot saveSlot) {
	g.currentLevel = slot.LevelIndex
	g.config.KeyMap = slot.KeyMap
	g.config.Volume = slot.Volume
	g.config.Difficulty = slot.Difficulty
	g.score = &slot.Score
	g.saveData.UnlockedLevels = make([]bool, len(allLevels))
	copy(g.saveData.UnlockedLevels, slot.UnlockedLevels)
}

// screenSaveSlots handles slot selection (New / Load / Delete).
type screenSaveSlots struct {
	menu   *uiMenu
	mode   string // "new" or "load"
}

func newScreenSaveSlots(g *pixelQuest, mode string) *screenSaveSlots {
	s := &screenSaveSlots{menu: newUIMenu(), mode: mode}
	for i := 0; i < maxSaveSlots; i++ {
		idx := i
		var label string
		var enabled bool
		if idx < len(g.saveData.Slots) {
			slot := g.saveData.Slots[idx]
			label = fmt.Sprintf("Slot %d: Lv%d %s", idx+1, slot.LevelIndex+1, slot.LastSaveTime)
			enabled = true
		} else {
			label = fmt.Sprintf("Slot %d: Empty", idx+1)
			enabled = mode == "new"
		}
		btn := s.menu.AddButton(label, func() {
			if mode == "new" {
				// Create new game in this slot.
				g.score = newScoreState()
				g.saveData.LastUsedSlot = idx
				if idx >= len(g.saveData.Slots) {
					g.saveData.Slots = append(g.saveData.Slots, saveSlot{})
				}
				g.saveData.Slots[idx] = g.createSnapshot(idx)
				g.saveData.UnlockedLevels = make([]bool, len(allLevels))
				g.saveData.UnlockedLevels[0] = true
				_ = saveSaveData(g.saveData)
				g.loadLevel(g.ctx, 0)
				g.changeState(statePlaying)
			} else {
				// Load existing.
				if idx < len(g.saveData.Slots) {
					g.saveData.LastUsedSlot = idx
					g.applySnapshot(g.saveData.Slots[idx])
					_ = saveSaveData(g.saveData)
					g.loadLevel(g.ctx, g.currentLevel)
					g.changeState(statePlaying)
				}
			}
		})
		btn.Enabled = enabled
	}
	s.menu.AddButton("Back", func() { g.popState() })
	return s
}

func (s *screenSaveSlots) Update(g *pixelQuest, ctx *whisky.Context, dt float64) {
	s.menu.Update(ctx, dt)
}

func (s *screenSaveSlots) Draw(g *pixelQuest, ctx *whisky.Context) {
	vw, vh := ctx.VirtualSize()
	ctx.DrawRect(geom.Rect{X: 0, Y: 0, W: vw, H: vh}, geom.RGBA(0.02, 0.02, 0.04, 0.9))
	panelW, panelH := 200.0, float64(maxSaveSlots+2)*22+20
	px := (vw - panelW) / 2
	py := (vh - panelH) / 2
	uiPanel(ctx, px, py, panelW, panelH)
	title := "New Game"
	if s.mode == "load" {
		title = "Load Game"
	}
	uiTitle(ctx, px, py, panelW, 20, title)

	s.menu.LayoutCentered(py+26, 170, 16, 5, vw, vh)
	s.menu.Draw(ctx)
}
