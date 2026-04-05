package tilemap

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
)

func TestNewTileSet(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 10)
	if ts.Name != "test" {
		t.Fatalf("expected name 'test', got %q", ts.Name)
	}
	if ts.TileWidth != 16 || ts.TileHeight != 16 {
		t.Fatalf("expected 16x16 tiles, got %dx%d", ts.TileWidth, ts.TileHeight)
	}
}

func TestTileSetProperties(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})
	ts.SetProperties(2, TileProperties{Solid: true, OneWay: true})
	ts.SetProperties(3, TileProperties{Trigger: true})

	if !ts.IsSolid(1) {
		t.Fatal("tile 1 should be solid")
	}
	if ts.IsOneWay(1) {
		t.Fatal("tile 1 should not be one-way")
	}
	if !ts.IsOneWay(2) {
		t.Fatal("tile 2 should be one-way")
	}
	if ts.IsSolid(0) {
		t.Fatal("tile 0 (empty) should not be solid")
	}

	props := ts.GetProperties(3)
	if !props.Trigger {
		t.Fatal("tile 3 should be trigger")
	}
}

func TestNewTileLayer(t *testing.T) {
	l := NewTileLayer("terrain", 10, 5)
	if l.Width != 10 || l.Height != 5 {
		t.Fatalf("expected 10x5, got %dx%d", l.Width, l.Height)
	}

	// Default is all zeros.
	for y := 0; y < l.Height; y++ {
		for x := 0; x < l.Width; x++ {
			if got := l.Get(x, y); got != 0 {
				t.Fatalf("expected empty tile at (%d,%d), got %d", x, y, got)
			}
		}
	}
}

func TestTileLayerSetGet(t *testing.T) {
	l := NewTileLayer("terrain", 5, 5)
	l.Set(2, 3, 42)

	if got := l.Get(2, 3); got != 42 {
		t.Fatalf("expected tile 42 at (2,3), got %d", got)
	}

	// Out of bounds reads return 0.
	if got := l.Get(-1, 0); got != 0 {
		t.Fatalf("expected 0 for out-of-bounds read, got %d", got)
	}
	if got := l.Get(5, 0); got != 0 {
		t.Fatalf("expected 0 for out-of-bounds read, got %d", got)
	}
}

func TestTileLayerInBounds(t *testing.T) {
	l := NewTileLayer("test", 4, 3)

	if !l.InBounds(0, 0) {
		t.Fatal("(0,0) should be in bounds")
	}
	if !l.InBounds(3, 2) {
		t.Fatal("(3,2) should be in bounds")
	}
	if l.InBounds(4, 0) {
		t.Fatal("(4,0) should be out of bounds")
	}
	if l.InBounds(-1, 0) {
		t.Fatal("(-1,0) should be out of bounds")
	}
}

func TestTileLayerTilesRoundTrip(t *testing.T) {
	l := NewTileLayer("test", 3, 2)
	l.Set(0, 0, 1)
	l.Set(1, 0, 2)
	l.Set(2, 1, 3)

	data := l.Tiles()
	l2 := NewTileLayer("copy", 3, 2)
	l2.SetTiles(data)

	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			if l.Get(x, y) != l2.Get(x, y) {
				t.Fatalf("mismatch at (%d,%d): %d vs %d", x, y, l.Get(x, y), l2.Get(x, y))
			}
		}
	}
}

func TestTileMapNew(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	m := New(ts, 20, 12)

	if m.Width != 20 || m.Height != 12 {
		t.Fatalf("expected 20x12, got %dx%d", m.Width, m.Height)
	}
	if len(m.Layers) != 0 {
		t.Fatalf("expected 0 layers, got %d", len(m.Layers))
	}
}

func TestTileMapAddAndGetLayer(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	m := New(ts, 10, 10)

	bg := m.AddLayer("background")
	terrain := m.AddLayer("terrain")

	if len(m.Layers) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(m.Layers))
	}
	if bg.Name != "background" {
		t.Fatalf("expected layer name 'background', got %q", bg.Name)
	}

	got := m.Layer("terrain")
	if got != terrain {
		t.Fatal("Layer() did not return the correct layer")
	}
	if m.Layer("nonexistent") != nil {
		t.Fatal("Layer() should return nil for unknown name")
	}
}

func TestTileMapTileSize(t *testing.T) {
	ts := NewTileSet("test", 8, 8, 4)
	m := New(ts, 10, 10)

	w, h := m.TileSize()
	if w != 8 || h != 8 {
		t.Fatalf("expected tile size 8x8, got %dx%d", w, h)
	}
}

func TestTileMapWorldBounds(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	m := New(ts, 20, 12)

	bounds := m.WorldBounds()
	if bounds.W != 320 || bounds.H != 192 {
		t.Fatalf("expected world bounds 320x192, got %.0fx%.0f", bounds.W, bounds.H)
	}
}

func TestTileMapCoordinateConversion(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	m := New(ts, 20, 12)

	// Tile to world.
	pos := m.TileToWorld(3, 5)
	if pos.X != 48 || pos.Y != 80 {
		t.Fatalf("expected world pos (48, 80), got (%.0f, %.0f)", pos.X, pos.Y)
	}

	// World to tile.
	tx, ty := m.WorldToTile(geom.Vec2{X: 50, Y: 82})
	if tx != 3 || ty != 5 {
		t.Fatalf("expected tile (3, 5), got (%d, %d)", tx, ty)
	}

	// Edge case: exact tile boundary.
	tx, ty = m.WorldToTile(geom.Vec2{X: 48, Y: 80})
	if tx != 3 || ty != 5 {
		t.Fatalf("expected tile (3, 5), got (%d, %d)", tx, ty)
	}

	// [C2] Negative coordinates: floor division.
	tx, ty = m.WorldToTile(geom.Vec2{X: -1, Y: -1})
	if tx != -1 || ty != -1 {
		t.Fatalf("expected tile (-1, -1) for negative coords, got (%d, %d)", tx, ty)
	}
	tx, ty = m.WorldToTile(geom.Vec2{X: -16, Y: -16})
	if tx != -1 || ty != -1 {
		t.Fatalf("expected tile (-1, -1) for -16, got (%d, %d)", tx, ty)
	}
	tx, ty = m.WorldToTile(geom.Vec2{X: -17, Y: -17})
	if tx != -2 || ty != -2 {
		t.Fatalf("expected tile (-2, -2) for -17, got (%d, %d)", tx, ty)
	}
}

// [A1] Zero tile dimensions must panic.
func TestNewTileSetPanicsOnZeroDimensions(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for zero tile width")
		}
	}()
	NewTileSet("bad", 0, 16, 4)
}

// [A2] SetProperties for TileID(0) must be a no-op.
func TestSetPropertiesIgnoresZeroID(t *testing.T) {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(0, TileProperties{Solid: true})
	if ts.IsSolid(0) {
		t.Fatal("TileID(0) should never have properties")
	}
}
