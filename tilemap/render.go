package tilemap

import "github.com/IsraelAraujo70/whisky-game-engine/geom"

// TileRenderInfo describes a single tile that should be drawn by a renderer.
type TileRenderInfo struct {
	ID       TileID
	WorldPos geom.Vec2
	Layer    string
}

// Renderer is the interface a rendering backend must satisfy to draw tiles.
// This is a stub — no concrete implementation exists yet.
type Renderer interface {
	DrawTiles(tiles []TileRenderInfo) error
}

// VisibleTiles returns all non-empty tiles whose world positions overlap the
// camera rectangle. offset shifts the entire tilemap (e.g. from the scene
// node's world position). Layers are returned in back-to-front order.
func VisibleTiles(m *TileMap, camera geom.Rect, offset geom.Vec2) []TileRenderInfo {
	minX, minY, maxX, maxY := VisibleTileRange(m, camera, offset)

	var tiles []TileRenderInfo
	tw := m.TileSet.TileWidth
	th := m.TileSet.TileHeight

	for _, layer := range m.Layers {
		for y := minY; y <= maxY; y++ {
			for x := minX; x <= maxX; x++ {
				id := layer.Get(x, y)
				if id == 0 {
					continue
				}
				tiles = append(tiles, TileRenderInfo{
					ID: id,
					WorldPos: geom.Vec2{
						X: offset.X + float64(x*tw),
						Y: offset.Y + float64(y*th),
					},
					Layer: layer.Name,
				})
			}
		}
	}

	return tiles
}

// VisibleTileRange returns the inclusive tile coordinate range that overlaps
// the camera rectangle. The range is clamped to the map bounds. If the camera
// does not overlap the map at all, minX > maxX or minY > maxY.
func VisibleTileRange(m *TileMap, camera geom.Rect, offset geom.Vec2) (minX, minY, maxX, maxY int) {
	tw := float64(m.TileSet.TileWidth)
	th := float64(m.TileSet.TileHeight)

	// Convert camera to tilemap-local coordinates.
	localX := camera.X - offset.X
	localY := camera.Y - offset.Y

	minX = floorDiv(int(localX), m.TileSet.TileWidth)
	minY = floorDiv(int(localY), m.TileSet.TileHeight)
	maxX = int((localX + camera.W) / tw)
	maxY = int((localY + camera.H) / th)

	// Clamp to map bounds.
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= m.Width {
		maxX = m.Width - 1
	}
	if maxY >= m.Height {
		maxY = m.Height - 1
	}

	return minX, minY, maxX, maxY
}
