package gameplay

import (
	"fmt"
	"math/rand"

	"github.com/IsraelAraujo70/whisky-game-engine/scene"
)

type DropEntry struct {
	Kind      string
	ID        string
	MinAmount int
	MaxAmount int
	Chance    float64
}

type DropResult struct {
	Kind   string
	ID     string
	Amount int
}

func (d DropResult) String() string {
	switch {
	case d.ID != "":
		return fmt.Sprintf("%s:%s x%d", d.Kind, d.ID, d.Amount)
	case d.Kind != "":
		return fmt.Sprintf("%s x%d", d.Kind, d.Amount)
	default:
		return fmt.Sprintf("drop x%d", d.Amount)
	}
}

// DropComponent stores drop metadata on a node and rolls it once.
type DropComponent struct {
	Entries []DropEntry
	Dropped bool
}

func (d *DropComponent) Start(node *scene.Node) error {
	return nil
}

func (d *DropComponent) Update(node *scene.Node, dt float64) error {
	return nil
}

func (d *DropComponent) Destroy(node *scene.Node) error {
	return nil
}

func (d *DropComponent) Roll(rng *rand.Rand) []DropResult {
	if d == nil || d.Dropped {
		return nil
	}
	d.Dropped = true
	return RollDrops(rng, d.Entries)
}

func RollDrops(rng *rand.Rand, entries []DropEntry) []DropResult {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}

	var drops []DropResult
	for _, entry := range entries {
		chance := entry.Chance
		if chance <= 0 {
			continue
		}
		if chance > 1 {
			chance = 1
		}
		if rng.Float64() > chance {
			continue
		}

		minAmount := entry.MinAmount
		maxAmount := entry.MaxAmount
		if minAmount <= 0 {
			minAmount = 1
		}
		if maxAmount < minAmount {
			maxAmount = minAmount
		}

		amount := minAmount
		if maxAmount > minAmount {
			amount += rng.Intn(maxAmount - minAmount + 1)
		}

		drops = append(drops, DropResult{
			Kind:   entry.Kind,
			ID:     entry.ID,
			Amount: amount,
		})
	}

	return drops
}

var _ scene.Component = (*DropComponent)(nil)
