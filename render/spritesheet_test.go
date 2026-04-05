package render

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
)

func TestSpritesheetFrameRect(t *testing.T) {
	sheet := &Spritesheet{
		Texture:     7,
		FrameWidth:  16,
		FrameHeight: 8,
		Columns:     4,
		Rows:        2,
	}

	tests := []struct {
		name  string
		index int
		want  geom.Rect
	}{
		{name: "first", index: 0, want: geom.Rect{X: 0, Y: 0, W: 16, H: 8}},
		{name: "middle", index: 5, want: geom.Rect{X: 16, Y: 8, W: 16, H: 8}},
		{name: "last", index: 7, want: geom.Rect{X: 48, Y: 8, W: 16, H: 8}},
		{name: "negative clamps", index: -3, want: geom.Rect{X: 0, Y: 0, W: 16, H: 8}},
		{name: "overflow clamps", index: 99, want: geom.Rect{X: 48, Y: 8, W: 16, H: 8}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sheet.FrameRect(tt.index); got != tt.want {
				t.Fatalf("FrameRect(%d) = %#v, want %#v", tt.index, got, tt.want)
			}
		})
	}
}

func TestSpritesheetFrameCount(t *testing.T) {
	sheet := &Spritesheet{Columns: 3, Rows: 2}
	if got := sheet.FrameCount(); got != 6 {
		t.Fatalf("FrameCount() = %d, want 6", got)
	}
}
