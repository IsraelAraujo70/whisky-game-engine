# Package: `scene`

The `scene` package provides the initial gameplay composition model: a scene owns a root node, nodes form a hierarchy, and components hang off nodes.

## Files

| File | Purpose |
|------|---------|
| `scene.go` | `Scene`, `Node`, `Component`, traversal and lifecycle |
| `drawable.go` | `Drawable` draw pass over the scene tree |
| `sprite_component.go` | `SpriteComponent` for texture-backed rendering |

## Current design

- `Scene` wraps a root node and lazily starts the tree on first update
- `Node` contains local position, parent, children, and components
- `WorldPosition()` is computed by walking the parent chain
- components receive `Start`, `Update`, and `Destroy`
- renderable components may additionally implement `Drawable`
- `Scene.Draw(ctx)` walks the tree after gameplay updates and emits draw commands

## Why this model now

The scene graph is simpler than a full ECS and fits the code-first 2D bootstrap well. It now also supports renderer-facing components without changing the base `Component` interface, which keeps existing gameplay code source-compatible while enabling sprite-backed nodes.
