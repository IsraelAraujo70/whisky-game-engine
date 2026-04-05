package tilemap

import (
	"fmt"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
)

// TileID identifies a tile type within a TileSet. ID 0 always means empty.
type TileID uint16

// TileProperties holds behavioral metadata for a tile type.
type TileProperties struct {
	Solid   bool
	OneWay  bool
	Trigger bool
	Tags    map[string]string
}

// TileSet describes the visual and behavioral catalogue of tile types used by
// a TileMap. Every tile is a fixed-size rectangle (e.g. 16x16 pixels).
type TileSet struct {
	Name       string
	TileWidth  int
	TileHeight int
	TileCount  int
	props      map[TileID]TileProperties
}

// NewTileSet creates a TileSet with the given tile dimensions and capacity.
// Panics if tileW or tileH are not positive.
func NewTileSet(name string, tileW, tileH, count int) *TileSet {
	if tileW <= 0 || tileH <= 0 {
		panic(fmt.Sprintf("tilemap: tile dimensions must be positive, got %dx%d", tileW, tileH))
	}
	return &TileSet{
		Name:       name,
		TileWidth:  tileW,
		TileHeight: tileH,
		TileCount:  count,
		props:      make(map[TileID]TileProperties),
	}
}

// SetProperties assigns behavioral properties to a tile ID.
// ID 0 is reserved for empty tiles and is silently ignored.
func (ts *TileSet) SetProperties(id TileID, props TileProperties) {
	if id == 0 {
		return
	}
	ts.props[id] = props
}

// GetProperties returns the properties for a tile ID.
// Returns a zero-value TileProperties for unknown IDs.
func (ts *TileSet) GetProperties(id TileID) TileProperties {
	return ts.props[id]
}

// IsSolid reports whether a tile ID has the Solid flag.
func (ts *TileSet) IsSolid(id TileID) bool {
	return ts.props[id].Solid
}

// IsOneWay reports whether a tile ID has the OneWay flag.
func (ts *TileSet) IsOneWay(id TileID) bool {
	return ts.props[id].OneWay
}

// TileLayer is a named 2D grid of tile IDs stored in row-major order.
type TileLayer struct {
	Name   string
	Width  int
	Height int
	tiles  []TileID
}

// NewTileLayer creates a layer with all tiles set to 0 (empty).
func NewTileLayer(name string, w, h int) *TileLayer {
	return &TileLayer{
		Name:   name,
		Width:  w,
		Height: h,
		tiles:  make([]TileID, w*h),
	}
}

// InBounds reports whether (x, y) is inside the layer grid.
func (l *TileLayer) InBounds(x, y int) bool {
	return x >= 0 && x < l.Width && y >= 0 && y < l.Height
}

// Get returns the tile ID at (x, y). Returns 0 for out-of-bounds coordinates.
func (l *TileLayer) Get(x, y int) TileID {
	if !l.InBounds(x, y) {
		return 0
	}
	return l.tiles[y*l.Width+x]
}

// Set stores a tile ID at (x, y). Out-of-bounds writes are silently ignored.
func (l *TileLayer) Set(x, y int, id TileID) {
	if !l.InBounds(x, y) {
		return
	}
	l.tiles[y*l.Width+x] = id
}

// Tiles returns a copy of the underlying tile data in row-major order.
func (l *TileLayer) Tiles() []TileID {
	out := make([]TileID, len(l.tiles))
	copy(out, l.tiles)
	return out
}

// SetTiles replaces the underlying tile data. The slice length must match
// Width*Height; otherwise the call is silently ignored.
func (l *TileLayer) SetTiles(data []TileID) {
	if len(data) != l.Width*l.Height {
		return
	}
	copy(l.tiles, data)
}

// TileMap is the top-level container for a tile-based level. It holds a
// TileSet reference, map dimensions in tiles, and an ordered list of layers.
type TileMap struct {
	TileSet *TileSet
	Width   int
	Height  int
	Layers  []*TileLayer
}

// New creates a TileMap with the given TileSet and dimensions.
func New(ts *TileSet, width, height int) *TileMap {
	return &TileMap{
		TileSet: ts,
		Width:   width,
		Height:  height,
	}
}

// AddLayer creates a new named layer appended at the back (drawn last / on top).
func (m *TileMap) AddLayer(name string) *TileLayer {
	layer := NewTileLayer(name, m.Width, m.Height)
	m.Layers = append(m.Layers, layer)
	return layer
}

// Layer returns the first layer with the given name, or nil.
func (m *TileMap) Layer(name string) *TileLayer {
	for _, l := range m.Layers {
		if l.Name == name {
			return l
		}
	}
	return nil
}

// TileSize returns the tile dimensions from the TileSet.
func (m *TileMap) TileSize() (w, h int) {
	return m.TileSet.TileWidth, m.TileSet.TileHeight
}

// WorldBounds returns the bounding rectangle of the entire map in world units.
func (m *TileMap) WorldBounds() geom.Rect {
	return geom.Rect{
		X: 0,
		Y: 0,
		W: float64(m.Width * m.TileSet.TileWidth),
		H: float64(m.Height * m.TileSet.TileHeight),
	}
}

// TileToWorld converts tile grid coordinates to the top-left world position
// of that tile.
func (m *TileMap) TileToWorld(tx, ty int) geom.Vec2 {
	return geom.Vec2{
		X: float64(tx * m.TileSet.TileWidth),
		Y: float64(ty * m.TileSet.TileHeight),
	}
}

// WorldToTile converts a world position to the tile grid coordinates that
// contain it. Uses floor division so negative coordinates map correctly.
func (m *TileMap) WorldToTile(pos geom.Vec2) (tx, ty int) {
	tx = floorDiv(int(pos.X), m.TileSet.TileWidth)
	ty = floorDiv(int(pos.Y), m.TileSet.TileHeight)
	return tx, ty
}

// floorDiv performs integer division that rounds toward negative infinity,
// unlike Go's built-in / which truncates toward zero.
func floorDiv(a, b int) int {
	q := a / b
	if (a^b) < 0 && q*b != a {
		q--
	}
	return q
}
