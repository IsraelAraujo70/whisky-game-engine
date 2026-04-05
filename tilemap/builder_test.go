package tilemap

import "testing"

func newTestMap() *TileMap {
	ts := NewTileSet("test", 16, 16, 4)
	ts.SetProperties(1, TileProperties{Solid: true})
	m := New(ts, 10, 8)
	m.AddLayer("terrain")
	return m
}

func TestSetTile(t *testing.T) {
	m := newTestMap()

	if !m.SetTile("terrain", 3, 4, 1) {
		t.Fatal("SetTile should return true for valid coordinates")
	}
	if m.Layer("terrain").Get(3, 4) != 1 {
		t.Fatal("SetTile did not write the tile")
	}

	// Out of bounds.
	if m.SetTile("terrain", -1, 0, 1) {
		t.Fatal("SetTile should return false for out-of-bounds")
	}

	// Invalid layer.
	if m.SetTile("nonexistent", 0, 0, 1) {
		t.Fatal("SetTile should return false for unknown layer")
	}
}

func TestFill(t *testing.T) {
	m := newTestMap()
	m.Fill("terrain", 1)

	layer := m.Layer("terrain")
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if layer.Get(x, y) != 1 {
				t.Fatalf("expected tile 1 at (%d,%d), got %d", x, y, layer.Get(x, y))
			}
		}
	}
}

func TestFillRect(t *testing.T) {
	m := newTestMap()
	m.FillRect("terrain", 2, 3, 4, 2, 1)

	layer := m.Layer("terrain")
	for y := 3; y < 5; y++ {
		for x := 2; x < 6; x++ {
			if layer.Get(x, y) != 1 {
				t.Fatalf("expected tile 1 at (%d,%d), got %d", x, y, layer.Get(x, y))
			}
		}
	}

	// Outside the rect should be empty.
	if layer.Get(1, 3) != 0 {
		t.Fatal("tile outside rect should be empty")
	}
	if layer.Get(6, 3) != 0 {
		t.Fatal("tile outside rect should be empty")
	}
}

func TestFillRow(t *testing.T) {
	m := newTestMap()
	m.FillRow("terrain", 0, 7, 10, 1)

	layer := m.Layer("terrain")
	for x := 0; x < 10; x++ {
		if layer.Get(x, 7) != 1 {
			t.Fatalf("expected tile 1 at (%d,7), got %d", x, layer.Get(x, 7))
		}
	}
	// Row above should be empty.
	if layer.Get(0, 6) != 0 {
		t.Fatal("row above should be empty")
	}
}

func TestFillCol(t *testing.T) {
	m := newTestMap()
	m.FillCol("terrain", 5, 0, 8, 1)

	layer := m.Layer("terrain")
	for y := 0; y < 8; y++ {
		if layer.Get(5, y) != 1 {
			t.Fatalf("expected tile 1 at (5,%d), got %d", y, layer.Get(5, y))
		}
	}
	// Column to the left should be empty.
	if layer.Get(4, 0) != 0 {
		t.Fatal("column to the left should be empty")
	}
}

func TestBuildPlatform(t *testing.T) {
	m := newTestMap()
	m.BuildPlatform("terrain", 2, 5, 6, 1)

	layer := m.Layer("terrain")
	for x := 2; x < 8; x++ {
		if layer.Get(x, 5) != 1 {
			t.Fatalf("expected tile 1 at (%d,5), got %d", x, layer.Get(x, 5))
		}
	}
	if layer.Get(2, 4) != 0 {
		t.Fatal("row above platform should be empty")
	}
}

func TestBuildBox(t *testing.T) {
	m := newTestMap()
	m.BuildBox("terrain", 1, 1, 4, 3, 1)

	layer := m.Layer("terrain")

	// Top row.
	for x := 1; x < 5; x++ {
		if layer.Get(x, 1) != 1 {
			t.Fatalf("top row: expected tile 1 at (%d,1)", x)
		}
	}
	// Bottom row.
	for x := 1; x < 5; x++ {
		if layer.Get(x, 3) != 1 {
			t.Fatalf("bottom row: expected tile 1 at (%d,3)", x)
		}
	}
	// Left and right walls in middle.
	if layer.Get(1, 2) != 1 {
		t.Fatal("left wall missing at (1,2)")
	}
	if layer.Get(4, 2) != 1 {
		t.Fatal("right wall missing at (4,2)")
	}
	// Interior should be empty.
	if layer.Get(2, 2) != 0 {
		t.Fatal("interior should be empty")
	}
	if layer.Get(3, 2) != 0 {
		t.Fatal("interior should be empty")
	}
}

func TestBuildFilledBox(t *testing.T) {
	m := newTestMap()
	m.BuildFilledBox("terrain", 2, 2, 3, 3, 1)

	layer := m.Layer("terrain")
	for y := 2; y < 5; y++ {
		for x := 2; x < 5; x++ {
			if layer.Get(x, y) != 1 {
				t.Fatalf("expected tile 1 at (%d,%d)", x, y)
			}
		}
	}
}
