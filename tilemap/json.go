package tilemap

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
)

// JSON format version.
const formatVersion = 1

// --- JSON wire types ---

type jsonTileProperties struct {
	Solid   bool              `json:"solid,omitempty"`
	OneWay  bool              `json:"one_way,omitempty"`
	Trigger bool              `json:"trigger,omitempty"`
	Tags    map[string]string `json:"tags,omitempty"`
}

type jsonTileSet struct {
	Name       string                        `json:"name"`
	TileWidth  int                           `json:"tile_width"`
	TileHeight int                           `json:"tile_height"`
	TileCount  int                           `json:"tile_count"`
	Properties map[string]jsonTileProperties `json:"properties,omitempty"`
}

type jsonLayer struct {
	Name  string   `json:"name"`
	Tiles []uint16 `json:"tiles"`
}

type jsonTileMap struct {
	Version int         `json:"version"`
	TileSet jsonTileSet `json:"tileset"`
	Width   int         `json:"width"`
	Height  int         `json:"height"`
	Layers  []jsonLayer `json:"layers"`
}

// --- Public API ---

// Marshal serializes a TileMap to JSON bytes.
func Marshal(m *TileMap) ([]byte, error) {
	jm := toJSON(m)
	return json.MarshalIndent(jm, "", "  ")
}

// SaveToFile writes a TileMap as JSON to the given path.
func SaveToFile(m *TileMap, path string) error {
	data, err := Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadFromBytes parses a TileMap from JSON bytes.
func LoadFromBytes(data []byte) (*TileMap, error) {
	var jm jsonTileMap
	if err := json.Unmarshal(data, &jm); err != nil {
		return nil, fmt.Errorf("tilemap: unmarshal: %w", err)
	}
	return fromJSON(&jm)
}

// LoadFromReader parses a TileMap from an io.Reader.
func LoadFromReader(r io.Reader) (*TileMap, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("tilemap: read: %w", err)
	}
	return LoadFromBytes(data)
}

// LoadFromFile reads and parses a TileMap from a JSON file.
func LoadFromFile(path string) (*TileMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tilemap: open %s: %w", path, err)
	}
	return LoadFromBytes(data)
}

// --- Conversion helpers ---

func toJSON(m *TileMap) *jsonTileMap {
	jts := jsonTileSet{
		Name:       m.TileSet.Name,
		TileWidth:  m.TileSet.TileWidth,
		TileHeight: m.TileSet.TileHeight,
		TileCount:  m.TileSet.TileCount,
	}

	if len(m.TileSet.props) > 0 {
		jts.Properties = make(map[string]jsonTileProperties, len(m.TileSet.props))
		for id, p := range m.TileSet.props {
			jts.Properties[strconv.Itoa(int(id))] = jsonTileProperties{
				Solid:   p.Solid,
				OneWay:  p.OneWay,
				Trigger: p.Trigger,
				Tags:    p.Tags,
			}
		}
	}

	layers := make([]jsonLayer, len(m.Layers))
	for i, l := range m.Layers {
		tiles := make([]uint16, len(l.tiles))
		for j, t := range l.tiles {
			tiles[j] = uint16(t)
		}
		layers[i] = jsonLayer{
			Name:  l.Name,
			Tiles: tiles,
		}
	}

	return &jsonTileMap{
		Version: formatVersion,
		TileSet: jts,
		Width:   m.Width,
		Height:  m.Height,
		Layers:  layers,
	}
}

func fromJSON(jm *jsonTileMap) (*TileMap, error) {
	if jm.Version != formatVersion {
		return nil, fmt.Errorf("tilemap: unsupported format version %d (expected %d)", jm.Version, formatVersion)
	}

	if jm.TileSet.TileWidth <= 0 || jm.TileSet.TileHeight <= 0 {
		return nil, fmt.Errorf("tilemap: tile dimensions must be positive, got %dx%d",
			jm.TileSet.TileWidth, jm.TileSet.TileHeight)
	}
	if jm.Width <= 0 || jm.Height <= 0 {
		return nil, fmt.Errorf("tilemap: map dimensions must be positive, got %dx%d",
			jm.Width, jm.Height)
	}

	ts := NewTileSet(jm.TileSet.Name, jm.TileSet.TileWidth, jm.TileSet.TileHeight, jm.TileSet.TileCount)
	for key, jp := range jm.TileSet.Properties {
		id, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("tilemap: invalid tile ID %q: %w", key, err)
		}
		ts.SetProperties(TileID(id), TileProperties{
			Solid:   jp.Solid,
			OneWay:  jp.OneWay,
			Trigger: jp.Trigger,
			Tags:    jp.Tags,
		})
	}

	m := New(ts, jm.Width, jm.Height)

	expectedLen := jm.Width * jm.Height
	for _, jl := range jm.Layers {
		if len(jl.Tiles) != expectedLen {
			return nil, fmt.Errorf("tilemap: layer %q has %d tiles, expected %d",
				jl.Name, len(jl.Tiles), expectedLen)
		}
		layer := m.AddLayer(jl.Name)
		for i, t := range jl.Tiles {
			layer.tiles[i] = TileID(t)
		}
	}

	return m, nil
}
