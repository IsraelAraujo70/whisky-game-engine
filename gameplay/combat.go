package gameplay

import (
	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
)

type Team string

const (
	TeamNeutral Team = "neutral"
	TeamPlayer  Team = "player"
	TeamEnemy   Team = "enemy"
)

// Box is an AABB relative to a scene node. If Node is nil, Offset is treated as
// an absolute world position.
type Box struct {
	Node   *scene.Node
	Offset geom.Vec2
	W      float64
	H      float64
}

func (b Box) Rect() geom.Rect {
	pos := b.Offset
	if b.Node != nil {
		pos = b.Node.WorldPosition().Add(b.Offset)
	}
	return geom.Rect{X: pos.X, Y: pos.Y, W: b.W, H: b.H}
}

type DamageSource struct {
	ID       string
	Team     Team
	Box      Box
	Damage   int
	Disabled bool
}

type DamageTarget struct {
	ID       string
	Team     Team
	Box      Box
	Health   *Health
	Disabled bool
}

type DamageEvent struct {
	SourceID string
	TargetID string
	Amount   int
}

// ResolveDamage applies overlapping damage sources to eligible targets and
// returns the resulting events. Damage is skipped for the same team, the same
// ID, disabled entries, and invulnerable or dead targets.
func ResolveDamage(sources []DamageSource, targets []DamageTarget) []DamageEvent {
	var events []DamageEvent

	for _, source := range sources {
		if source.Disabled || source.Damage <= 0 || source.Box.W <= 0 || source.Box.H <= 0 {
			continue
		}

		sourceRect := source.Box.Rect()
		for _, target := range targets {
			if target.Disabled || target.Health == nil || target.Box.W <= 0 || target.Box.H <= 0 {
				continue
			}
			if source.ID != "" && source.ID == target.ID {
				continue
			}
			if source.Team != TeamNeutral && source.Team == target.Team {
				continue
			}
			if !sourceRect.Intersects(target.Box.Rect()) {
				continue
			}
			if target.Health.Damage(source.Damage) {
				events = append(events, DamageEvent{
					SourceID: source.ID,
					TargetID: target.ID,
					Amount:   source.Damage,
				})
			}
		}
	}

	return events
}
