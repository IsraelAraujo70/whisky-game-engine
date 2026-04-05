package tilemap

// SetTile sets a single tile on the named layer.
// Returns false if the layer is not found or coordinates are out of bounds.
func (m *TileMap) SetTile(layer string, x, y int, id TileID) bool {
	l := m.Layer(layer)
	if l == nil || !l.InBounds(x, y) {
		return false
	}
	l.Set(x, y, id)
	return true
}

// Fill sets every tile in the named layer to id.
func (m *TileMap) Fill(layer string, id TileID) {
	l := m.Layer(layer)
	if l == nil {
		return
	}
	for y := 0; y < l.Height; y++ {
		for x := 0; x < l.Width; x++ {
			l.Set(x, y, id)
		}
	}
}

// FillRect fills a rectangular region on the named layer with id.
func (m *TileMap) FillRect(layer string, x, y, w, h int, id TileID) {
	l := m.Layer(layer)
	if l == nil {
		return
	}
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			l.Set(x+dx, y+dy, id)
		}
	}
}

// FillRow fills a horizontal run of tiles starting at (x, y).
func (m *TileMap) FillRow(layer string, x, y, length int, id TileID) {
	m.FillRect(layer, x, y, length, 1, id)
}

// FillCol fills a vertical run of tiles starting at (x, y).
func (m *TileMap) FillCol(layer string, x, y, length int, id TileID) {
	m.FillRect(layer, x, y, 1, length, id)
}

// BuildPlatform places a horizontal one-tile-high platform.
// Alias for FillRow; exists for readability in level-building code.
func (m *TileMap) BuildPlatform(layer string, x, y, length int, id TileID) {
	m.FillRow(layer, x, y, length, id)
}

// BuildBox draws a hollow rectangle border of tiles on the named layer.
func (m *TileMap) BuildBox(layer string, x, y, w, h int, id TileID) {
	if w <= 0 || h <= 0 {
		return
	}
	// top and bottom rows
	m.FillRow(layer, x, y, w, id)
	if h > 1 {
		m.FillRow(layer, x, y+h-1, w, id)
	}
	// left and right columns (excluding corners already set)
	for dy := 1; dy < h-1; dy++ {
		m.SetTile(layer, x, y+dy, id)
		if w > 1 {
			m.SetTile(layer, x+w-1, y+dy, id)
		}
	}
}

// BuildFilledBox fills a solid rectangle of tiles on the named layer.
// Alias for FillRect; exists for readability in level-building code.
func (m *TileMap) BuildFilledBox(layer string, x, y, w, h int, id TileID) {
	m.FillRect(layer, x, y, w, h, id)
}
