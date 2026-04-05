package tilemap

import (
	"bytes"
	"testing"
)

func TestMarshalAndLoadRoundTrip(t *testing.T) {
	ts := NewTileSet("dungeon", 16, 16, 64)
	ts.SetProperties(1, TileProperties{Solid: true})
	ts.SetProperties(2, TileProperties{Solid: true, OneWay: true})
	ts.SetProperties(3, TileProperties{Trigger: true, Tags: map[string]string{"type": "spike"}})

	m := New(ts, 5, 3)
	terrain := m.AddLayer("terrain")
	m.AddLayer("background")

	m.FillRow("terrain", 0, 2, 5, 1)
	m.SetTile("terrain", 2, 1, 2)
	m.SetTile("terrain", 4, 0, 3)

	data, err := Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	loaded, err := LoadFromBytes(data)
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	// Compare structure.
	if loaded.Width != m.Width || loaded.Height != m.Height {
		t.Fatalf("dimensions mismatch: %dx%d vs %dx%d", loaded.Width, loaded.Height, m.Width, m.Height)
	}
	if len(loaded.Layers) != 2 {
		t.Fatalf("expected 2 layers, got %d", len(loaded.Layers))
	}
	if loaded.Layers[0].Name != "terrain" {
		t.Fatalf("expected first layer 'terrain', got %q", loaded.Layers[0].Name)
	}

	// Compare tileset.
	if loaded.TileSet.Name != "dungeon" {
		t.Fatalf("tileset name mismatch: %q", loaded.TileSet.Name)
	}
	if !loaded.TileSet.IsSolid(1) {
		t.Fatal("tile 1 should be solid")
	}
	if !loaded.TileSet.IsOneWay(2) {
		t.Fatal("tile 2 should be one-way")
	}
	props3 := loaded.TileSet.GetProperties(3)
	if !props3.Trigger || props3.Tags["type"] != "spike" {
		t.Fatal("tile 3 properties mismatch")
	}

	// Compare tile data.
	lTerrain := loaded.Layer("terrain")
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if lTerrain.Get(x, y) != terrain.Get(x, y) {
				t.Fatalf("tile mismatch at (%d,%d): %d vs %d", x, y, lTerrain.Get(x, y), terrain.Get(x, y))
			}
		}
	}
}

func TestLoadFromReader(t *testing.T) {
	ts := NewTileSet("test", 8, 8, 4)
	ts.SetProperties(1, TileProperties{Solid: true})

	m := New(ts, 3, 2)
	m.AddLayer("terrain")
	m.SetTile("terrain", 1, 1, 1)

	data, err := Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	loaded, err := LoadFromReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("LoadFromReader failed: %v", err)
	}

	if loaded.Layer("terrain").Get(1, 1) != 1 {
		t.Fatal("tile data mismatch after LoadFromReader")
	}
}

func TestLoadFromBytesInvalidVersion(t *testing.T) {
	data := []byte(`{"version": 99, "tileset": {"name":"x","tile_width":8,"tile_height":8,"tile_count":1}, "width":1, "height":1, "layers":[]}`)
	_, err := LoadFromBytes(data)
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
}

func TestLoadFromBytesInvalidJSON(t *testing.T) {
	_, err := LoadFromBytes([]byte(`{invalid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadFromBytesLayerSizeMismatch(t *testing.T) {
	data := []byte(`{"version":1,"tileset":{"name":"x","tile_width":8,"tile_height":8,"tile_count":1},"width":2,"height":2,"layers":[{"name":"a","tiles":[0,0,0]}]}`)
	_, err := LoadFromBytes(data)
	if err == nil {
		t.Fatal("expected error for layer size mismatch")
	}
}

func TestEmptyMapRoundTrip(t *testing.T) {
	ts := NewTileSet("empty", 16, 16, 0)
	m := New(ts, 2, 2)
	m.AddLayer("bg")

	data, err := Marshal(m)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	loaded, err := LoadFromBytes(data)
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}
	if loaded.Width != 2 || loaded.Height != 2 {
		t.Fatal("dimensions mismatch")
	}
	if len(loaded.Layers) != 1 {
		t.Fatal("layer count mismatch")
	}
}

// [A1] JSON with zero or negative tile dimensions must fail.
func TestLoadFromBytesZeroTileDimensions(t *testing.T) {
	data := []byte(`{"version":1,"tileset":{"name":"x","tile_width":0,"tile_height":8,"tile_count":1},"width":1,"height":1,"layers":[]}`)
	_, err := LoadFromBytes(data)
	if err == nil {
		t.Fatal("expected error for zero tile width in JSON")
	}
}

func TestLoadFromBytesNegativeTileDimensions(t *testing.T) {
	data := []byte(`{"version":1,"tileset":{"name":"x","tile_width":-1,"tile_height":8,"tile_count":1},"width":1,"height":1,"layers":[]}`)
	_, err := LoadFromBytes(data)
	if err == nil {
		t.Fatal("expected error for negative tile width in JSON")
	}
}

func TestLoadFromBytesZeroMapDimensions(t *testing.T) {
	data := []byte(`{"version":1,"tileset":{"name":"x","tile_width":8,"tile_height":8,"tile_count":1},"width":0,"height":1,"layers":[]}`)
	_, err := LoadFromBytes(data)
	if err == nil {
		t.Fatal("expected error for zero map width in JSON")
	}
}
