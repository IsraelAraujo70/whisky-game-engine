package physics

import "github.com/IsraelAraujo70/whisky-game-engine/geom"

type Layer uint32

const (
	LayerWorld Layer = 1 << iota
	LayerPlayer
	LayerTrigger
)

type Collider struct {
	ID      string
	Bounds  geom.Rect
	Layer   Layer
	Mask    Layer
	Trigger bool
}

func (c Collider) CanCollide(other Collider) bool {
	return c.Mask&other.Layer != 0 && other.Mask&c.Layer != 0
}

func Overlaps(a, b Collider) bool {
	return a.CanCollide(b) && a.Bounds.Intersects(b.Bounds)
}

type World struct {
	colliders []Collider
}

func NewWorld() *World {
	return &World{}
}

func (w *World) Add(c Collider) {
	w.colliders = append(w.colliders, c)
}

func (w *World) QueryPoint(point geom.Vec2, mask Layer) []Collider {
	var result []Collider
	for _, collider := range w.colliders {
		if collider.Layer&mask == 0 {
			continue
		}
		if collider.Bounds.Contains(point) {
			result = append(result, collider)
		}
	}

	return result
}

func (w *World) QueryRect(bounds geom.Rect, mask Layer) []Collider {
	var result []Collider
	for _, collider := range w.colliders {
		if collider.Layer&mask == 0 {
			continue
		}
		if collider.Bounds.Intersects(bounds) {
			result = append(result, collider)
		}
	}

	return result
}
