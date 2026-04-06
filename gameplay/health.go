package gameplay

import "github.com/IsraelAraujo70/whisky-game-engine/scene"

// Health tracks hit points and a short invulnerability window after taking
// damage. It can be attached directly to a scene.Node as a component.
type Health struct {
	Max                 int
	Current             int
	InvulnerableFor     float64
	invulnerableElapsed float64
}

func NewHealth(max int) *Health {
	if max < 1 {
		max = 1
	}
	return &Health{
		Max:     max,
		Current: max,
	}
}

func (h *Health) Start(node *scene.Node) error {
	return nil
}

func (h *Health) Update(node *scene.Node, dt float64) error {
	if h == nil || h.invulnerableElapsed <= 0 {
		return nil
	}

	h.invulnerableElapsed -= dt
	if h.invulnerableElapsed < 0 {
		h.invulnerableElapsed = 0
	}
	return nil
}

func (h *Health) Destroy(node *scene.Node) error {
	return nil
}

func (h *Health) Alive() bool {
	return h != nil && h.Current > 0
}

func (h *Health) Invulnerable() bool {
	return h != nil && h.invulnerableElapsed > 0
}

func (h *Health) CanTakeDamage() bool {
	return h != nil && h.Alive() && !h.Invulnerable()
}

func (h *Health) Damage(amount int) bool {
	if h == nil || amount <= 0 || !h.CanTakeDamage() {
		return false
	}

	h.Current -= amount
	if h.Current < 0 {
		h.Current = 0
	}
	if h.Current > 0 && h.InvulnerableFor > 0 {
		h.invulnerableElapsed = h.InvulnerableFor
	}

	return true
}

func (h *Health) Heal(amount int) bool {
	if h == nil || amount <= 0 || !h.Alive() || h.Current >= h.Max {
		return false
	}

	h.Current += amount
	if h.Current > h.Max {
		h.Current = h.Max
	}
	return true
}

func (h *Health) Fraction() float64 {
	if h == nil || h.Max <= 0 {
		return 0
	}
	return float64(h.Current) / float64(h.Max)
}

var _ scene.Component = (*Health)(nil)
