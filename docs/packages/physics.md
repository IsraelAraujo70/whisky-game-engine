# Package: `physics`

The `physics` package provides the first collision-oriented gameplay helpers. It is not a full simulation engine. It is currently a queryable world of AABB colliders with layers and masks.

## Files

| File | Purpose |
|------|---------|
| `world.go` | collider data model, overlap checks, and queries |

## Current capabilities

- axis-aligned rectangle colliders
- layers and masks for broad interaction control
- trigger flag at the collider level
- point queries
- rectangle queries
- pairwise overlap testing

This gives the engine enough gameplay vocabulary for simple top-down and platformer prototypes before forces and advanced resolution exist.

