package game

import (
	"math"
	"math/rand"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	whisky "github.com/IsraelAraujo70/whisky-game-engine/whisky"
)

// particle is a single visual particle.
type particle struct {
	pos      geom.Vec2
	vel      geom.Vec2
	color    geom.Color
	size     float64
	life     float64 // seconds remaining
	maxLife  float64
	gravityY float64
}

// particleSystem manages a pool of particles.
type particleSystem struct {
	particles []particle
}

func newParticleSystem() *particleSystem {
	return &particleSystem{particles: make([]particle, 0, 256)}
}

func (ps *particleSystem) emitBurst(count int, x, y float64, color geom.Color, speed, life float64, gravityY float64, rng *rand.Rand) {
	for i := 0; i < count; i++ {
		angle := rng.Float64() * 2 * math.Pi
		spd := speed * (0.3 + rng.Float64()*0.7)
		p := particle{
			pos:      geom.Vec2{X: x, Y: y},
			vel:      geom.Vec2{X: math.Cos(angle) * spd, Y: math.Sin(angle) * spd},
			color:    color,
			size:     1 + rng.Float64()*2,
			life:     life * (0.5 + rng.Float64()*0.5),
			maxLife:  life,
			gravityY: gravityY,
		}
		ps.particles = append(ps.particles, p)
	}
}

func (ps *particleSystem) emitJump(x, y float64, rng *rand.Rand) {
	ps.emitBurst(8, x, y+playerH, geom.RGBA(0.7, 0.7, 0.6, 1), 20, 0.3, 80, rng)
}

func (ps *particleSystem) emitLand(x, y float64, rng *rand.Rand) {
	ps.emitBurst(6, x+playerW/2, y+playerH, geom.RGBA(0.5, 0.5, 0.45, 1), 15, 0.25, 40, rng)
}

func (ps *particleSystem) emitAttackHit(x, y float64, rng *rand.Rand) {
	ps.emitBurst(10, x, y, geom.RGBA(1.0, 0.85, 0.2, 1), 40, 0.2, 0, rng)
}

func (ps *particleSystem) emitEnemyDeath(x, y float64, rng *rand.Rand) {
	ps.emitBurst(16, x+enemyW/2, y+enemyH/2, geom.RGBA(0.9, 0.3, 0.25, 1), 50, 0.4, 60, rng)
}

func (ps *particleSystem) emitPlayerHit(x, y float64, rng *rand.Rand) {
	ps.emitBurst(12, x+playerW/2, y+playerH/2, geom.RGBA(0.9, 0.1, 0.1, 1), 35, 0.35, 40, rng)
}

func (ps *particleSystem) emitCollect(x, y float64, color geom.Color, rng *rand.Rand) {
	ps.emitBurst(8, x, y, color, 25, 0.3, -30, rng)
}

func (ps *particleSystem) emitProjectileTrail(x, y float64, rng *rand.Rand) {
	ps.emitBurst(2, x, y, geom.RGBA(0.9, 0.5, 0.2, 1), 5, 0.15, 0, rng)
}

func (ps *particleSystem) Update(dt float64) {
	alive := ps.particles[:0]
	for i := range ps.particles {
		p := &ps.particles[i]
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.vel.Y += p.gravityY * dt
		p.pos.X += p.vel.X * dt
		p.pos.Y += p.vel.Y * dt
		alive = append(alive, *p)
	}
	ps.particles = alive
}

func (ps *particleSystem) Draw(ctx *whisky.Context) {
	for i := range ps.particles {
		p := &ps.particles[i]
		alpha := p.life / p.maxLife
		if alpha > 1 {
			alpha = 1
		}
		col := geom.Color{
			R: p.color.R,
			G: p.color.G,
			B: p.color.B,
			A: p.color.A * float32(alpha),
		}
		s := p.size * alpha
		ctx.DrawRect(geom.Rect{X: p.pos.X - s/2, Y: p.pos.Y - s/2, W: s, H: s}, col)
	}
}

func (ps *particleSystem) Count() int {
	return len(ps.particles)
}
