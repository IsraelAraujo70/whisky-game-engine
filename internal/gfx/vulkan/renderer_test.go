package vulkan

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

func TestAppendOrMergeBatchMergesContiguousTextureRuns(t *testing.T) {
	texture := &gpuTexture{}
	batches := []drawBatch{}

	batches = appendOrMergeBatch(batches, texture, 0, 6)
	batches = appendOrMergeBatch(batches, texture, 6, 6)
	batches = appendOrMergeBatch(batches, texture, 12, 6)

	if len(batches) != 1 {
		t.Fatalf("expected 1 merged batch, got %d", len(batches))
	}
	if batches[0].vertexCount != 18 {
		t.Fatalf("expected merged vertex count 18, got %d", batches[0].vertexCount)
	}
}

func TestBuildDrawDataAppendsDebugOverlay(t *testing.T) {
	fontTexture := &gpuTexture{width: 16, height: 8}
	renderer := &Renderer2D{
		virtualWidth:  320,
		virtualHeight: 180,
		whiteTexture:  &gpuTexture{width: 1, height: 1},
		debugFont: &bitmapFont{
			texture:     fontTexture,
			glyphWidth:  8,
			glyphHeight: 8,
			lineHeight:  10,
			glyphs: map[rune]geom.Rect{
				'A': {X: 0, Y: 0, W: 8, H: 8},
				'?': {X: 8, Y: 0, W: 8, H: 8},
			},
		},
	}

	vertices, batches := renderer.buildDrawData([]render.DrawCmd{
		render.FillRect{Rect: geom.Rect{X: 0, Y: 0, W: 10, H: 10}, Color: geom.RGBA(1, 0, 0, 1)},
		render.FillRect{Rect: geom.Rect{X: 12, Y: 0, W: 10, H: 10}, Color: geom.RGBA(0, 1, 0, 1)},
	}, []string{"A"})

	if len(batches) != 2 {
		t.Fatalf("expected 2 batches (rects + font), got %d", len(batches))
	}
	if batches[0].vertexCount != 18 {
		t.Fatalf("expected first batch to merge 3 quads (2 rects + overlay bg), got %d vertices", batches[0].vertexCount)
	}
	if batches[1].texture != fontTexture {
		t.Fatalf("expected second batch to use font texture")
	}
	if batches[1].vertexCount != 6 {
		t.Fatalf("expected one glyph quad in font batch, got %d vertices", batches[1].vertexCount)
	}
	if len(vertices) != 24 {
		t.Fatalf("expected 24 vertices total, got %d", len(vertices))
	}
}

func TestWrapOverlayLineBreaksLongText(t *testing.T) {
	lines := wrapOverlayLine("A/D move   Space/W/Up jump   J/K attack   LShift sprint", 20)
	if len(lines) < 2 {
		t.Fatalf("expected wrapped output, got %v", lines)
	}
	for _, line := range lines {
		if len([]rune(line)) > 20 {
			t.Fatalf("line exceeds wrap width: %q", line)
		}
	}
}
