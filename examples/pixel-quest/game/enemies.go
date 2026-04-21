package game

import (
	"math"
	"math/rand"

	"github.com/IsraelAraujo70/whisky-game-engine/gameplay"
	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/physics"
	"github.com/IsraelAraujo70/whisky-game-engine/scene"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
)

// enemyType defines enemy behavior variants.
type enemyType int

const (
	enemyPatrol enemyType = iota
	enemyJumper
	enemyShooter
	enemyBoss
)

// enemyState holds per-enemy runtime state beyond the shared components.
type enemyState struct {
	typ           enemyType
	jumpTimer     float64
	jumpCooldown  float64
	shootTimer    float64
	shootCooldown float64
	projectiles   []*projectile
}

// projectile is a hostile bullet/spell.
type projectile struct {
	pos  geom.Vec2
	vel  geom.Vec2
	size float64
	life float64
}

func (p *projectile) Update(dt float64) {
	p.pos.X += p.vel.X * dt
	p.pos.Y += p.vel.Y * dt
	p.life -= dt
}

func (p *projectile) Rect() geom.Rect {
	return geom.Rect{X: p.pos.X, Y: p.pos.Y, W: p.size, H: p.size}
}

// spawnEnemyByType creates an enemy with a specific behavior pattern.
func (g *pixelQuest) spawnEnemyByType(root *scene.Node, id string, x, y float64, typ enemyType, hp int, speed float64, dropEntries []gameplay.DropEntry) {
	node := scene.NewNode(id)
	node.Position = geom.Vec2{X: x, Y: y}

	patrol := &gameplay.PatrolComponent{
		MinX:      x - 40,
		MaxX:      x + 40,
		Speed:     speed,
		Direction: 1,
	}
	if typ == enemyBoss {
		patrol.MinX = x - 60
		patrol.MaxX = x + 60
		patrol.Speed = speed * 0.6
	}

	health := gameplay.NewHealth(hp)
	health.InvulnerableFor = 0.15
	if typ == enemyBoss {
		health.InvulnerableFor = 0.3
	}

	drops := &gameplay.DropComponent{Entries: dropEntries}
	target := &gameplay.TargetComponent{
		TargetBox: gameplay.Box{
			Node: g.player,
			W:    playerW,
			H:    playerH,
		},
		Sight: gameplay.Box{
			Node:   node,
			Offset: geom.Vec2{X: -enemySightRangeX, Y: -6},
			W:      enemySightRangeX * 2,
			H:      enemySightRangeY,
		},
		Speed:          speed + 10,
		StopDistance:   6,
		HorizontalOnly: true,
		Patrol:         patrol,
	}
	if typ == enemyShooter {
		target.StopDistance = 48 // keep distance
	}
	if typ == enemyBoss {
		target.StopDistance = 24
		target.Sight.W = enemySightRangeX * 3
	}

	node.AddComponent(patrol)
	node.AddComponent(target)
	node.AddComponent(health)
	node.AddComponent(drops)
	root.AddChild(node)

	en := &enemy{
		id:     id,
		node:   node,
		patrol: patrol,
		target: target,
		health: health,
		drops:  drops,
	}
	g.enemies = append(g.enemies, en)

	// Store extended state in a separate map keyed by id.
	if g.enemyStates == nil {
		g.enemyStates = make(map[string]*enemyState)
	}
	st := &enemyState{
		typ:           typ,
		jumpCooldown:  1.5 + rand.Float64(),
		shootCooldown: 1.2 + rand.Float64()*0.8,
	}
	if typ == enemyBoss {
		st.shootCooldown = 0.6 + rand.Float64()*0.4
	}
	g.enemyStates[id] = st
}

func rectOverlaps(a, b geom.Rect) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

func (g *pixelQuest) updateEnemies(ctx *whisky.Context, dt float64, ps *particleSystem, dm *dropManager, score *scoreState) {
	for _, en := range g.enemies {
		if en.health == nil || !en.health.Alive() {
			continue
		}
		st, ok := g.enemyStates[en.id]
		if !ok {
			continue
		}

		pos := en.node.WorldPosition()

		switch st.typ {
		case enemyJumper:
			st.jumpTimer += dt
			if st.jumpTimer >= st.jumpCooldown && g.groundedCheck(pos.X, pos.Y+enemyH+1) {
				st.jumpTimer = 0
				st.jumpCooldown = 1.0 + rand.Float64()*1.5
				// Apply a small hop by temporarily moving Y (simplified).
				en.node.Position.Y -= 16
			}

		case enemyShooter:
			st.shootTimer += dt
			if st.shootTimer >= st.shootCooldown {
				st.shootTimer = 0
				// Fire toward player.
				pp := g.player.WorldPosition()
				dx := pp.X + playerW/2 - (pos.X + enemyW/2)
				dy := pp.Y + playerH/2 - (pos.Y + enemyH/2)
				mag := math.Hypot(dx, dy)
				if mag > 1 {
					vx := dx / mag * 60
					vy := dy / mag * 60
					st.projectiles = append(st.projectiles, &projectile{
						pos:  geom.Vec2{X: pos.X + enemyW/2, Y: pos.Y + enemyH/2},
						vel:  geom.Vec2{X: vx, Y: vy},
						size: 3,
						life: 3.0,
					})
				}
			}

		case enemyBoss:
			st.shootTimer += dt
			if st.shootTimer >= st.shootCooldown {
				st.shootTimer = 0
				pp := g.player.WorldPosition()
				for angle := -0.3; angle <= 0.3; angle += 0.3 {
					dx := pp.X + playerW/2 - (pos.X + 10)
					dy := pp.Y + playerH/2 - (pos.Y + 10)
					mag := math.Hypot(dx, dy)
					if mag > 1 {
						baseAngle := math.Atan2(dy, dx)
						vx := math.Cos(baseAngle+angle) * 70
						vy := math.Sin(baseAngle+angle) * 70
						st.projectiles = append(st.projectiles, &projectile{
							pos:  geom.Vec2{X: pos.X + 10, Y: pos.Y + 10},
							vel:  geom.Vec2{X: vx, Y: vy},
							size: 4,
							life: 4.0,
						})
					}
				}
			}
		}

		// Update projectiles for shooter/boss.
		aliveProj := st.projectiles[:0]
		for _, pr := range st.projectiles {
			pr.Update(dt)
			if pr.life > 0 {
				aliveProj = append(aliveProj, pr)
			}
		}
		st.projectiles = aliveProj
	}
}

func (g *pixelQuest) groundedCheck(x, y float64) bool {
	// Simple check: is there a solid tile below this point?
	r := geom.Rect{X: x, Y: y, W: 1, H: 1}
	hits := g.world.QueryRect(r, physics.LayerWorld)
	for _, h := range hits {
		if !h.OneWay {
			return true
		}
	}
	return false
}

func (g *pixelQuest) checkProjectileHits(score *scoreState, ps *particleSystem, dm *dropManager) {
	pp := g.player.WorldPosition()
	playerR := geom.Rect{X: pp.X, Y: pp.Y, W: playerW, H: playerH}

	for _, en := range g.enemies {
		st, ok := g.enemyStates[en.id]
		if !ok {
			continue
		}
		if en.health == nil || !en.health.Alive() {
			continue
		}
		alive := st.projectiles[:0]
		for _, pr := range st.projectiles {
			if rectOverlaps(pr.Rect(), playerR) {
				// Hit player.
				if g.playerHealth != nil && g.playerHealth.Alive() {
					g.playerHealth.Damage(1)
					ps.emitPlayerHit(pp.X+playerW/2, pp.Y+playerH/2, g.rng)
					if !g.playerHealth.Alive() && !g.playerDefeated {
						g.playerDefeated = true
						g.changeState(stateGameOver)
					}
				}
				continue // projectile consumed
			}
			alive = append(alive, pr)
		}
		st.projectiles = alive
	}
}

func (g *pixelQuest) drawProjectiles(ctx *whisky.Context) {
	for _, en := range g.enemies {
		st, ok := g.enemyStates[en.id]
		if !ok {
			continue
		}
		for _, pr := range st.projectiles {
			ctx.DrawRect(pr.Rect(), geom.RGBA(0.9, 0.5, 0.2, 1))
		}
	}
}

func (g *pixelQuest) drawEnemyByType(ctx *whisky.Context, en *enemy) {
	if en.health == nil || !en.health.Alive() {
		return
	}
	pos := en.node.WorldPosition()
	st := g.enemyStates[en.id]

	w, h := enemyW, enemyH
	color := geom.RGBA(0.9, 0.3, 0.25, 1)
	if st != nil {
		switch st.typ {
		case enemyJumper:
			color = geom.RGBA(0.3, 0.7, 0.4, 1)
		case enemyShooter:
			color = geom.RGBA(0.5, 0.3, 0.8, 1)
		case enemyBoss:
			color = geom.RGBA(0.8, 0.1, 0.1, 1)
			w, h = 20, 20
		}
	}
	if en.target != nil && en.target.Chasing {
		color = geom.RGBA(1, 0.1, 0.1, 1)
	}
	if en.health.Invulnerable() {
		color = geom.RGBA(1, 0.7, 0.55, 1)
	}
	ctx.DrawRect(geom.Rect{X: pos.X, Y: pos.Y, W: w, H: h}, color)
	g.drawHealthBar(ctx, pos.X, pos.Y-4, w, en.health)
}
