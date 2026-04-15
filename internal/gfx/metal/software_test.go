package metal

import (
	"testing"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

func TestComputePresentationLayoutPixelPerfect(t *testing.T) {
	layout := computePresentationLayout(1280, 720, 320, 180, true)
	if layout.viewportWidth != 1280 || layout.viewportHeight != 720 {
		t.Fatalf("expected full viewport, got %+v", layout)
	}
	if layout.scissorWidth != 1280 || layout.scissorHeight != 720 {
		t.Fatalf("expected full scissor, got %+v", layout)
	}
}

func TestComputePresentationLayoutLetterboxes(t *testing.T) {
	layout := computePresentationLayout(1000, 700, 320, 180, true)
	if layout.viewportWidth != 960 {
		t.Fatalf("expected viewport width 960, got %v", layout.viewportWidth)
	}
	if layout.viewportHeight != 540 {
		t.Fatalf("expected viewport height 540, got %v", layout.viewportHeight)
	}
	if layout.scissorX != 20 || layout.scissorY != 80 {
		t.Fatalf("expected centered scissor, got %+v", layout)
	}
}

func TestBuildDrawDataAppendsDebugOverlay(t *testing.T) {
	renderer, err := newSoftwareRenderer()
	if err != nil {
		t.Fatalf("newSoftwareRenderer() error = %v", err)
	}
	renderer.setLogicalSize(64, 48, true)
	vertices, batches, logicalWidth, logicalHeight, err := renderer.buildDrawData([]render.DrawCmd{
		render.FillRect{Rect: geom.Rect{X: 8, Y: 8, W: 16, H: 12}, Color: geom.RGBA(1, 0, 0, 1)},
	}, []string{"A"}, 64, 48)
	if err != nil {
		t.Fatalf("buildDrawData() error = %v", err)
	}
	if logicalWidth != 64 || logicalHeight != 48 {
		t.Fatalf("unexpected logical size %dx%d", logicalWidth, logicalHeight)
	}
	if len(vertices) != 18 {
		t.Fatalf("expected 18 vertices (rect + overlay bg + glyph), got %d", len(vertices))
	}
	if len(batches) != 2 {
		t.Fatalf("expected 2 batches, got %d", len(batches))
	}
	if batches[0].vertexCount != 12 {
		t.Fatalf("expected first batch to merge rect and overlay background, got %d", batches[0].vertexCount)
	}
	if batches[1].texture != renderer.debugFont.texture {
		t.Fatalf("expected second batch to use debug font texture")
	}
}

func TestAppendQuadUsesPixelCoordinatesAndFlipFlags(t *testing.T) {
	vertices := appendQuad(nil, geom.Rect{X: 4, Y: 6, W: 8, H: 10}, geom.Rect{X: 2, Y: 1, W: 3, H: 5}, whiteColor(), true, false, 16, 16)
	if len(vertices) != 6 {
		t.Fatalf("expected 6 vertices, got %d", len(vertices))
	}
	if vertices[0].Position != [2]float32{4, 6} {
		t.Fatalf("unexpected top-left vertex position %+v", vertices[0].Position)
	}
	if vertices[0].UV[0] <= vertices[2].UV[0] {
		t.Fatalf("expected horizontal flip to swap U coordinates, got %v and %v", vertices[0].UV[0], vertices[2].UV[0])
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
