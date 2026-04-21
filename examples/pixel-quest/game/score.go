package game

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/IsraelAraujo70/whisky-game-engine/gameplay"
	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
)

// xpThresholds define XP needed for each level (index 0 = level 1).
var xpThresholds = []int{0, 50, 120, 220, 350, 500}

// scoreState tracks player progression.
type scoreState struct {
	XP        int
	Level     int
	Kills     int
	Coins     int // generic currency from drops
	HighScore int
}

func newScoreState() *scoreState {
	return &scoreState{Level: 1}
}

func (s *scoreState) AddXP(amount int) {
	s.XP += amount
	for s.Level < len(xpThresholds) && s.XP >= xpThresholds[s.Level] {
		s.Level++
	}
}

func (s *scoreState) AddKill() {
	s.Kills++
}

func (s *scoreState) AddCoins(amount int) {
	s.Coins += amount
}

func (s *scoreState) XPToNext() int {
	if s.Level >= len(xpThresholds) {
		return 0
	}
	return xpThresholds[s.Level] - s.XP
}

func (s *scoreState) XPFraction() float64 {
	if s.Level <= 1 || s.Level >= len(xpThresholds) {
		return 1.0
	}
	prev := xpThresholds[s.Level-1]
	next := xpThresholds[s.Level]
	return float64(s.XP-prev) / float64(next-prev)
}

func (s *scoreState) Reset() {
	s.XP = 0
	s.Level = 1
	s.Kills = 0
	s.Coins = 0
}

// worldDrop is a physical pickup item left by a defeated enemy.
type worldDrop struct {
	pos    geom.Vec2
	size   float64
	kind   string // "xp", "health", "coin", "item"
	amount int
	itemID string
	life   float64 // despawn timer
	bob    float64 // animation offset
}

// dropManager tracks pickups floating in the world.
type dropManager struct {
	drops []worldDrop
}

func newDropManager() *dropManager {
	return &dropManager{drops: make([]worldDrop, 0, 32)}
}

func (dm *dropManager) Spawn(x, y float64, results []gameplay.DropResult, rng *rand.Rand) {
	for _, r := range results {
		d := worldDrop{
			pos:    geom.Vec2{X: x + (rng.Float64()-0.5)*8, Y: y + (rng.Float64()-0.5)*4},
			size:   4,
			kind:   r.Kind,
			amount: r.Amount,
			itemID: r.ID,
			life:   10.0, // 10 seconds before vanishing
			bob:    rng.Float64() * 6.28,
		}
		dm.drops = append(dm.drops, d)
	}
}

func (dm *dropManager) Update(dt float64, playerPos geom.Vec2, playerW, playerH float64, score *scoreState, health *gameplay.Health, ps *particleSystem, rng *rand.Rand) {
	alive := dm.drops[:0]
	for i := range dm.drops {
		d := &dm.drops[i]
		d.life -= dt
		d.bob += dt * 4
		if d.life <= 0 {
			continue
		}

		// Collect if player overlaps.
		if playerPos.X < d.pos.X+d.size && playerPos.X+playerW > d.pos.X &&
			playerPos.Y < d.pos.Y+d.size && playerPos.Y+playerH > d.pos.Y {
			dm.collect(d, score, health, ps, rng)
			continue
		}
		alive = append(alive, *d)
	}
	dm.drops = alive
}

func (dm *dropManager) collect(d *worldDrop, score *scoreState, health *gameplay.Health, ps *particleSystem, rng *rand.Rand) {
	switch d.kind {
	case "xp":
		score.AddXP(d.amount * 10)
		ps.emitCollect(d.pos.X, d.pos.Y, geom.RGBA(0.2, 0.6, 1.0, 1), rng)
	case "health":
		if health != nil {
			health.Heal(d.amount)
		}
		ps.emitCollect(d.pos.X, d.pos.Y, geom.RGBA(0.2, 0.9, 0.3, 1), rng)
	case "coin":
		score.AddCoins(d.amount)
		ps.emitCollect(d.pos.X, d.pos.Y, geom.RGBA(0.96, 0.76, 0.20, 1), rng)
	case "item":
		score.AddCoins(5) // items convert to coins for now
		ps.emitCollect(d.pos.X, d.pos.Y, geom.RGBA(0.8, 0.4, 0.9, 1), rng)
	}
}

func (dm *dropManager) Draw(ctx *whisky.Context) {
	for i := range dm.drops {
		d := &dm.drops[i]
		alpha := 1.0
		if d.life < 2.0 {
			alpha = d.life / 2.0
		}
		offsetY := math.Sin(d.bob) * 2
		var col geom.Color
		switch d.kind {
		case "xp":
			col = geom.RGBA(0.2, 0.6, 1.0, float32(alpha))
		case "health":
			col = geom.RGBA(0.2, 0.9, 0.3, float32(alpha))
		case "coin":
			col = geom.RGBA(0.96, 0.76, 0.20, float32(alpha))
		case "item":
			col = geom.RGBA(0.8, 0.4, 0.9, float32(alpha))
		default:
			col = geom.RGBA(0.8, 0.8, 0.8, float32(alpha))
		}
		ctx.DrawRect(geom.Rect{X: d.pos.X, Y: d.pos.Y + offsetY, W: d.size, H: d.size}, col)
	}
}

func (dm *dropManager) Count() int {
	return len(dm.drops)
}

// formatScoreLine returns a compact score display string.
func formatScoreLine(s *scoreState) string {
	return fmt.Sprintf("LV%d  XP:%d  K:%d  $:%d", s.Level, s.XP, s.Kills, s.Coins)
}
