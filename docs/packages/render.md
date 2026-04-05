# Package: `render`

Lightweight rendering primitives and camera transform for 2D drawing.

## Files

| File | Purpose |
|------|---------|
| `render.go` | `FillRect` draw command and `Camera2D` with viewport / world-to-screen helpers |

## Key Types

### FillRect

A colored filled rectangle, used as the basic draw command:

```go
type FillRect struct {
    Rect  geom.Rect
    Color geom.Color
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

Games interact with rendering via `ctx.DrawRect(worldRect, color)` on the whisky Context, which uses the Camera2D internally. The render package itself has no SDL dependency — it only depends on `geom`.

## Dependencies

- `geom` — Vec2, Rect, Color
