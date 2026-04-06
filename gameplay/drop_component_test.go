package gameplay

import (
	"math/rand"
	"testing"
)

func TestRollDropsReturnsDeterministicResults(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	drops := RollDrops(rng, []DropEntry{
		{Kind: "xp", MinAmount: 1, MaxAmount: 3, Chance: 1},
		{Kind: "item", ID: "potion", MinAmount: 1, MaxAmount: 1, Chance: 1},
	})

	if len(drops) != 2 {
		t.Fatalf("expected 2 drops, got %d", len(drops))
	}
	if drops[0].Kind != "xp" || drops[0].Amount < 1 || drops[0].Amount > 3 {
		t.Fatalf("unexpected xp drop: %+v", drops[0])
	}
	if drops[1].Kind != "item" || drops[1].ID != "potion" || drops[1].Amount != 1 {
		t.Fatalf("unexpected item drop: %+v", drops[1])
	}
}

func TestDropComponentRollsOnce(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	drops := &DropComponent{
		Entries: []DropEntry{
			{Kind: "health", MinAmount: 1, MaxAmount: 1, Chance: 1},
		},
	}

	first := drops.Roll(rng)
	second := drops.Roll(rng)

	if len(first) != 1 {
		t.Fatalf("expected 1 drop on first roll, got %d", len(first))
	}
	if len(second) != 0 {
		t.Fatalf("expected second roll to be empty, got %d", len(second))
	}
}
