package render

import (
	"math"
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
)

const epsilon = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestCamera2D_ViewportRect(t *testing.T) {
	cam := Camera2D{Position: geom.Vec2{X: 160, Y: 90}}
	vp := cam.ViewportRect(320, 180)

	if !almostEqual(vp.X, 0) || !almostEqual(vp.Y, 0) {
		t.Errorf("expected viewport origin (0,0), got (%.2f, %.2f)", vp.X, vp.Y)
	}
	if !almostEqual(vp.W, 320) || !almostEqual(vp.H, 180) {
		t.Errorf("expected viewport size (320,180), got (%.2f, %.2f)", vp.W, vp.H)
	}
}

func TestCamera2D_ViewportRect_Offset(t *testing.T) {
	cam := Camera2D{Position: geom.Vec2{X: 200, Y: 100}}
	vp := cam.ViewportRect(320, 180)

	if !almostEqual(vp.X, 40) || !almostEqual(vp.Y, 10) {
		t.Errorf("expected viewport origin (40,10), got (%.2f, %.2f)", vp.X, vp.Y)
	}
}

func TestCamera2D_WorldToScreen(t *testing.T) {
	cam := Camera2D{Position: geom.Vec2{X: 160, Y: 90}}

	// World origin should map to screen origin when camera is centered.
	s := cam.WorldToScreen(geom.Vec2{X: 0, Y: 0}, 320, 180)
	if !almostEqual(s.X, 0) || !almostEqual(s.Y, 0) {
		t.Errorf("expected screen (0,0), got (%.2f, %.2f)", s.X, s.Y)
	}

	// Camera center should map to screen center.
	s = cam.WorldToScreen(geom.Vec2{X: 160, Y: 90}, 320, 180)
	if !almostEqual(s.X, 160) || !almostEqual(s.Y, 90) {
		t.Errorf("expected screen (160,90), got (%.2f, %.2f)", s.X, s.Y)
	}

	// Bottom-right of virtual area.
	s = cam.WorldToScreen(geom.Vec2{X: 320, Y: 180}, 320, 180)
	if !almostEqual(s.X, 320) || !almostEqual(s.Y, 180) {
		t.Errorf("expected screen (320,180), got (%.2f, %.2f)", s.X, s.Y)
	}
}

func TestCamera2D_WorldToScreen_Offset(t *testing.T) {
	cam := Camera2D{Position: geom.Vec2{X: 200, Y: 100}}

	// World (200,100) is camera center → screen center (160,90).
	s := cam.WorldToScreen(geom.Vec2{X: 200, Y: 100}, 320, 180)
	if !almostEqual(s.X, 160) || !almostEqual(s.Y, 90) {
		t.Errorf("expected screen (160,90), got (%.2f, %.2f)", s.X, s.Y)
	}

	// World origin is 200 px left and 100 px above camera center.
	s = cam.WorldToScreen(geom.Vec2{X: 0, Y: 0}, 320, 180)
	if !almostEqual(s.X, -40) || !almostEqual(s.Y, -10) {
		t.Errorf("expected screen (-40,-10), got (%.2f, %.2f)", s.X, s.Y)
	}
}
