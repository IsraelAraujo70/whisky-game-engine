package gameplay

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
)

func TestResolveDamageAppliesToOpposingTeam(t *testing.T) {
	targetHealth := NewHealth(2)
	target := scene.NewNode("enemy")
	target.Position = geom.Vec2{X: 16, Y: 4}

	events := ResolveDamage(
		[]DamageSource{
			{
				ID:     "player:attack",
				Team:   TeamPlayer,
				Damage: 1,
				Box: Box{
					Offset: geom.Vec2{X: 12, Y: 0},
					W:      16,
					H:      16,
				},
			},
		},
		[]DamageTarget{
			{
				ID:     "enemy",
				Team:   TeamEnemy,
				Health: targetHealth,
				Box: Box{
					Node: target,
					W:    12,
					H:    12,
				},
			},
		},
	)

	if len(events) != 1 {
		t.Fatalf("expected 1 damage event, got %d", len(events))
	}
	if targetHealth.Current != 1 {
		t.Fatalf("expected enemy hp=1, got %d", targetHealth.Current)
	}
}

func TestResolveDamageSkipsSameTeam(t *testing.T) {
	health := NewHealth(3)

	events := ResolveDamage(
		[]DamageSource{
			{
				ID:     "enemy:aura",
				Team:   TeamEnemy,
				Damage: 1,
				Box: Box{
					Offset: geom.Vec2{X: 0, Y: 0},
					W:      16,
					H:      16,
				},
			},
		},
		[]DamageTarget{
			{
				ID:     "enemy",
				Team:   TeamEnemy,
				Health: health,
				Box: Box{
					Offset: geom.Vec2{X: 4, Y: 4},
					W:      8,
					H:      8,
				},
			},
		},
	)

	if len(events) != 0 {
		t.Fatalf("expected 0 damage events, got %d", len(events))
	}
	if health.Current != 3 {
		t.Fatalf("expected hp=3, got %d", health.Current)
	}
}
