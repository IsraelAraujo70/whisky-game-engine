# Package: `render`

Lightweight rendering primitives, spritesheets, and camera transform for 2D drawing.

## Files

| File | Purpose |
|------|---------|
| `draw.go` | `TextureID`, closed `DrawCmd` interface, `SpriteCmd`, and `DrawContext` |
| `render.go` | `FillRect` draw command and `Camera2D` with viewport / world-to-screen helpers |
| `spritesheet.go` | `Spritesheet` frame indexing helpers |

## Key Types

### DrawCmd

The rendering pipeline now uses a closed draw-command interface so colored rectangles and sprites share one ordered command stream:

```go
type DrawCmd interface {
    drawCmd()
}
```

Built-in commands:

- `FillRect`
- `SpriteCmd`

### FillRect

A colored filled rectangle, used as the basic draw command:

```go
type FillRect struct {
    Rect  geom.Rect
    Color geom.Color
}
```

### SpriteCmd

Draws a region of a loaded texture into a destination rectangle:

```go
type SpriteCmd struct {
    Texture TextureID
    Src     geom.Rect
    Dst     geom.Rect
    FlipH   bool
    FlipV   bool
}
```

### Spritesheet

Maps a texture into a fixed grid of frames:

```go
type Spritesheet struct {
    Texture     TextureID
    FrameWidth  int
    FrameHeight int
    Columns     int
    Rows        int
}
```

### Camera2D

Provides world-to-screen coordinate transformation. `Position` is the center of the camera in world coordinates.

```go
type Camera2D struct {
    Position geom.Vec2
}
```

Methods:

- `ViewportRect(virtualW, virtualH float64) geom.Rect` — returns the axis-aligned rectangle in world space that the camera sees
- `WorldToScreen(worldPos geom.Vec2, virtualW, virtualH float64) geom.Vec2` — converts a world position to virtual screen coordinates

## Usage

Games interact with rendering through the whisky context, which implements `render.DrawContext`:

- `ctx.DrawRect(worldRect, color)`
- `ctx.DrawSprite(texture, src, dst, flipH, flipV)`
- `ctx.LoadTexture(path)`

The render package itself has no SDL dependency; it only depends on `geom`.

## Dependencies

- `geom` — Vec2, Rect, Color
