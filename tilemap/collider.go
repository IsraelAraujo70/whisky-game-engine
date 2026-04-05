package tilemap

import (
	"fmt"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
)

// ColliderConfig controls how tile colliders are generated and added to a
// physics.World.
type ColliderConfig struct {
	Prefix       string
	SolidLayer   physics.Layer
	SolidMask    physics.Layer
	TriggerLayer physics.Layer
	TriggerMask  physics.Layer
}

// DefaultColliderConfig returns sensible defaults for tile collider generation.
func DefaultColliderConfig() ColliderConfig {
	return ColliderConfig{
		Prefix:       "tile",
		SolidLayer:   physics.LayerWorld,
		SolidMask:    physics.LayerPlayer,
		TriggerLayer: physics.LayerTrigger,
		TriggerMask:  physics.LayerPlayer,
	}
}

// GenerateColliders produces physics.Collider entries from all layers of a
// TileMap. Solid tiles are merged into larger rectangles using a greedy
// algorithm. OneWay and Trigger tiles produce individual per-tile colliders.
//
// offset is added to every collider position so the tilemap can inherit
// placement from its parent scene node.
func GenerateColliders(m *TileMap, offset geom.Vec2, cfg ColliderConfig) []physics.Collider {
	var colliders []physics.Collider

	tw := m.TileSet.TileWidth
	th := m.TileSet.TileHeight

	for _, layer := range m.Layers {
		w := layer.Width
		h := layer.Height
		visited := make([]bool, w*h)

		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				idx := y*w + x
				if visited[idx] {
					continue
				}

				id := layer.Get(x, y)
				if id == 0 {
					continue
				}

				props := m.TileSet.GetProperties(id)

				// OneWay tiles: individual collider, not merged.
				if props.OneWay {
					visited[idx] = true
					colliders = append(colliders, physics.Collider{
						ID: fmt.Sprintf("%s:%s:%d,%d:1x1:oneway",
							cfg.Prefix, layer.Name, x, y),
						Bounds: geom.Rect{
							X: offset.X + float64(x*tw),
							Y: offset.Y + float64(y*th),
							W: float64(tw),
							H: float64(th),
						},
						Layer: cfg.SolidLayer,
						Mask:  cfg.SolidMask,
					})
					continue
				}

				// Trigger tiles: individual collider, not merged.
				if props.Trigger {
					visited[idx] = true
					colliders = append(colliders, physics.Collider{
						ID: fmt.Sprintf("%s:%s:%d,%d:1x1:trigger",
							cfg.Prefix, layer.Name, x, y),
						Bounds: geom.Rect{
							X: offset.X + float64(x*tw),
							Y: offset.Y + float64(y*th),
							W: float64(tw),
							H: float64(th),
						},
						Layer:   cfg.TriggerLayer,
						Mask:    cfg.TriggerMask,
						Trigger: true,
					})
					continue
				}

				// Solid tiles: greedy merge.
				if props.Solid {
					rw, rh := greedyMerge(m, layer, visited, x, y)
					colliders = append(colliders, physics.Collider{
						ID: fmt.Sprintf("%s:%s:%d,%d:%dx%d",
							cfg.Prefix, layer.Name, x, y, rw, rh),
						Bounds: geom.Rect{
							X: offset.X + float64(x*tw),
							Y: offset.Y + float64(y*th),
							W: float64(rw * tw),
							H: float64(rh * th),
						},
						Layer: cfg.SolidLayer,
						Mask:  cfg.SolidMask,
					})
					continue
				}

				// Non-physical tile — skip.
				visited[idx] = true
			}
		}
	}

	return colliders
}

// greedyMerge expands from (sx, sy) to the largest axis-aligned rectangle of
// solid (non-oneway, non-trigger) tiles and marks them visited. Returns the
// rectangle dimensions in tiles.
func greedyMerge(m *TileMap, layer *TileLayer, visited []bool, sx, sy int) (rw, rh int) {
	w := layer.Width
	h := layer.Height

	// Determine max width by extending right.
	rw = 0
	for x := sx; x < w; x++ {
		id := layer.Get(x, sy)
		if id == 0 || visited[sy*w+x] {
			break
		}
		props := m.TileSet.GetProperties(id)
		if !props.Solid || props.OneWay || props.Trigger {
			break
		}
		rw++
	}

	// Determine max height by extending down, requiring the full width span
	// to be solid.
	rh = 1
	for y := sy + 1; y < h; y++ {
		spanOK := true
		for x := sx; x < sx+rw; x++ {
			id := layer.Get(x, y)
			if id == 0 || visited[y*w+x] {
				spanOK = false
				break
			}
			props := m.TileSet.GetProperties(id)
			if !props.Solid || props.OneWay || props.Trigger {
				spanOK = false
				break
			}
		}
		if !spanOK {
			break
		}
		rh++
	}

	// Mark rectangle as visited.
	for y := sy; y < sy+rh; y++ {
		for x := sx; x < sx+rw; x++ {
			visited[y*w+x] = true
		}
	}

	return rw, rh
}

// AddToWorld generates colliders from the TileMap and adds them to the world.
func AddToWorld(m *TileMap, world *physics.World, offset geom.Vec2, cfg ColliderConfig) {
	for _, c := range GenerateColliders(m, offset, cfg) {
		world.Add(c)
	}
}
