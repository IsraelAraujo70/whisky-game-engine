package geom

type Vec2 struct {
	X float64
	Y float64
}

func (v Vec2) Add(other Vec2) Vec2 {
	return Vec2{X: v.X + other.X, Y: v.Y + other.Y}
}

func (v Vec2) Sub(other Vec2) Vec2 {
	return Vec2{X: v.X - other.X, Y: v.Y - other.Y}
}

func (v Vec2) Scale(factor float64) Vec2 {
	return Vec2{X: v.X * factor, Y: v.Y * factor}
}

type Rect struct {
	X float64
	Y float64
	W float64
	H float64
}

func (r Rect) Contains(point Vec2) bool {
	return point.X >= r.X &&
		point.X <= r.X+r.W &&
		point.Y >= r.Y &&
		point.Y <= r.Y+r.H
}

func (r Rect) Intersects(other Rect) bool {
	return r.X < other.X+other.W &&
		r.X+r.W > other.X &&
		r.Y < other.Y+other.H &&
		r.Y+r.H > other.Y
}

type Color struct {
	R float32
	G float32
	B float32
	A float32
}

func RGBA(r, g, b, a float32) Color {
	return Color{R: r, G: g, B: b, A: a}
}
