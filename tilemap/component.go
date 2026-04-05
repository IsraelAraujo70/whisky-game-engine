package tilemap

import (
	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
)

// TileMapComponent integrates a TileMap with the scene graph and physics world.
// It implements scene.Component so it can be attached to a scene.Node.
//
// On Start it generates colliders from the TileMap using the node's world
// position as offset. When MarkDirty is called, colliders are rebuilt on the
// next Update.
//
// Each component builds a unique collider prefix from the node name so that
// multiple tilemaps in the same physics world do not interfere with each other.
type TileMapComponent struct {
	Map    *TileMap
	World  *physics.World
	Config ColliderConfig

	dirty   bool
	started bool
	prefix  string
	offset  geom.Vec2
}

// Compile-time interface check.
var _ scene.Component = (*TileMapComponent)(nil)

// Start generates tile colliders and adds them to the physics world.
// The collider ID prefix includes the node name to avoid collisions between
// multiple TileMapComponents in the same world.
func (c *TileMapComponent) Start(node *scene.Node) error {
	c.ensureConfig()
	c.prefix = c.Config.Prefix + ":" + node.Name + ":"
	c.offset = node.WorldPosition()

	// Override the config prefix so GenerateColliders uses the unique prefix.
	cfg := c.Config
	cfg.Prefix = c.Config.Prefix + ":" + node.Name
	AddToWorld(c.Map, c.World, c.offset, cfg)
	c.started = true
	return nil
}

// Update rebuilds colliders if the component was marked dirty.
func (c *TileMapComponent) Update(node *scene.Node, _ float64) error {
	if !c.dirty {
		return nil
	}
	c.dirty = false
	c.World.RemoveByPrefix(c.prefix)
	c.offset = node.WorldPosition()

	cfg := c.Config
	cfg.Prefix = c.Config.Prefix + ":" + node.Name
	AddToWorld(c.Map, c.World, c.offset, cfg)
	return nil
}

// Destroy removes all tile colliders from the physics world.
// Safe to call before Start — it is a no-op if the component was never started.
func (c *TileMapComponent) Destroy(node *scene.Node) error {
	if !c.started {
		return nil
	}
	c.World.RemoveByPrefix(c.prefix)
	return nil
}

// MarkDirty flags the component for collider rebuild on the next Update.
func (c *TileMapComponent) MarkDirty() {
	c.dirty = true
}

func (c *TileMapComponent) ensureConfig() {
	if c.Config.Prefix == "" {
		c.Config = DefaultColliderConfig()
	}
}
