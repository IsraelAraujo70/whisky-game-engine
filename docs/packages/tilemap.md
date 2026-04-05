# Package: tilemap

Tile-based level data model with physics integration and programmatic construction helpers.

## Purpose

Provides the data layer for 2D tile-based levels (platformers, top-down, etc.). The package handles tile storage, coordinate conversion, collider generation with greedy merging, scene graph integration, viewport culling, and JSON serialization. It does not depend on a renderer — rendering stubs are provided for when the 2D renderer lands.

## Files

| File | Responsibility |
|------|----------------|
| `tilemap.go` | Core types: TileID, TileProperties, TileSet, TileLayer, TileMap |
| `builder.go` | Programmatic construction: SetTile, Fill, FillRect, BuildPlatform, BuildBox |
| `collider.go` | Physics integration: greedy merge algorithm, collider generation |
| `component.go` | Scene integration: TileMapComponent (implements scene.Component) |
| `render.go` | Rendering interface stub and viewport culling helpers |
| `json.go` | JSON load/save for map files |

## Key Types

### TileID (uint16)

Identifies a tile type. ID 0 is always empty. Supports up to 65,535 tile types.

### TileProperties

Behavioral metadata per tile type:
- `Solid` — blocks movement
- `OneWay` — collision only from above
- `Trigger` — sensor, does not block movement
- `Tags` — custom string key-value pairs

### TileSet

Catalogue of tile types with fixed dimensions (e.g. 16x16) and per-ID properties.

### TileLayer

Named 2D grid of TileIDs in row-major order. Supports Get, Set, InBounds, and bulk data access via Tiles/SetTiles.

### TileMap

Top-level container: TileSet reference, map dimensions in tiles, and ordered list of layers (back-to-front). Provides coordinate conversion (TileToWorld, WorldToTile) and world bounds calculation.

## Physics Integration

### Greedy Merge Algorithm

Solid tiles are merged into the fewest possible axis-aligned rectangles:
1. Scan row-major through each layer
2. For each unvisited solid tile, extend right then down to find the largest solid rectangle
3. Emit one merged collider per rectangle

OneWay and Trigger tiles are never merged — each gets its own collider for per-tile identification.

### Collider IDs

Format: `{prefix}:{layer}:{x},{y}:{w}x{h}` with optional suffixes `:oneway` or `:trigger`.

Example: `tile:terrain:3,5:4x2` means a merged 4x2-tile collider starting at grid position (3,5).

### ColliderConfig

Controls layer/mask assignment. Defaults: solid tiles use LayerWorld/LayerPlayer, trigger tiles use LayerTrigger/LayerPlayer.

## Scene Integration

`TileMapComponent` implements `scene.Component`:
- **Start**: generates colliders using node.WorldPosition() as offset
- **Update**: rebuilds colliders when marked dirty via MarkDirty()
- **Destroy**: removes tile colliders from physics world via RemoveByPrefix

## JSON Format

Version 1 format with tileset definition, map dimensions, and ordered layer array. Tiles stored as flat uint16 arrays in row-major order. Properties keyed by string tile IDs.

Functions: LoadFromFile, LoadFromReader, LoadFromBytes, Marshal, SaveToFile.

## Dependencies

- `geom` — Vec2 and Rect for coordinates and bounds
- `physics` — Collider, Layer, World for collision integration
- `scene` — Component interface and Node for scene graph integration
