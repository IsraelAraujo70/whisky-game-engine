# Package: `gameplay`

The `gameplay` package provides reusable higher-level building blocks on top of `geom` and `scene`: hit points, AABB hitbox/hurtbox resolution, and a simple patrol AI component.

## Files

| File | Purpose |
|------|---------|
| `health.go` | `Health` component with HP, healing, and invulnerability frames |
| `combat.go` | scene-aware hitbox/hurtbox boxes and overlap-based damage resolution |
| `patrol_component.go` | `PatrolComponent` for basic horizontal enemy movement |
| `target_component.go` | `TargetComponent` for sight-based chase behavior |
| `drop_component.go` | `DropComponent` and drop table helpers for enemy rewards |

## Current capabilities

- attach `Health` directly to a node as a `scene.Component`
- apply damage and healing with optional invulnerability windows
- resolve AABB hitbox/hurtbox combat using `scene.Node` world positions
- skip friendly fire through simple team tags
- move enemies or hazards between patrol bounds with reusable AI logic
- chase a target when it enters a configurable sight box
- define reusable enemy drops for health, xp, or item rewards
