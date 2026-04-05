package geom

import "testing"

func TestVec2Operations(t *testing.T) {
	start := Vec2{X: 4, Y: 2}
	got := start.Add(Vec2{X: 1, Y: 3}).Sub(Vec2{X: 2, Y: 1}).Scale(2)

	if got != (Vec2{X: 6, Y: 8}) {
		t.Fatalf("unexpected vector result: %#v", got)
	}
}

func TestRectIntersectionsAndContainment(t *testing.T) {
	a := Rect{X: 0, Y: 0, W: 10, H: 10}
	b := Rect{X: 5, Y: 5, W: 3, H: 3}
	c := Rect{X: 20, Y: 20, W: 2, H: 2}

	if !a.Contains(Vec2{X: 1, Y: 1}) {
		t.Fatal("expected point to be inside rect")
	}

	if !a.Intersects(b) {
		t.Fatal("expected rectangles to intersect")
	}

	if a.Intersects(c) {
		t.Fatal("did not expect rectangles to intersect")
	}
}
