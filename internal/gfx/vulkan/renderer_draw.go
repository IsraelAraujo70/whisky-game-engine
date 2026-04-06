package vulkan

import (
	"unsafe"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

type bitmapFont struct {
	texture     *gpuTexture
	glyphWidth  int
	glyphHeight int
	lineHeight  int
	glyphs      map[rune]geom.Rect
}

type overlayLayout struct {
	scale           float64
	glyphWidth      float64
	glyphHeight     float64
	lineHeight      float64
	padding         float64
	margin          float64
	maxColumns      int
	backgroundColor [4]float32
	textColor       [4]float32
}

func (r *Renderer2D) buildDrawData(cmds []render.DrawCmd, lines []string) ([]quadVertex, []drawBatch) {
	vertices := make([]quadVertex, 0, len(cmds)*6)
	batches := make([]drawBatch, 0, len(cmds))
	virtualWidth := float64(r.virtualWidth)
	virtualHeight := float64(r.virtualHeight)
	if virtualWidth <= 0 || virtualHeight <= 0 {
		virtualWidth = float64(r.swapchain.desc.Extent.Width)
		virtualHeight = float64(r.swapchain.desc.Extent.Height)
	}

	for _, cmd := range cmds {
		switch drawCmd := cmd.(type) {
		case render.FillRect:
			first := uint32(len(vertices))
			vertices = appendQuad(vertices, drawCmd.Rect, geom.Rect{W: 1, H: 1}, quadColor(drawCmd.Color), virtualWidth, virtualHeight, false, false, 1, 1)
			batches = appendOrMergeBatch(batches, r.whiteTexture, first, 6)
		case render.SpriteCmd:
			texture := r.texturesByID[drawCmd.Texture]
			if texture == nil {
				continue
			}
			first := uint32(len(vertices))
			vertices = appendQuad(vertices, drawCmd.Dst, drawCmd.Src, whiteColor(), virtualWidth, virtualHeight, drawCmd.FlipH, drawCmd.FlipV, texture.width, texture.height)
			batches = appendOrMergeBatch(batches, texture, first, 6)
		}
	}
	vertices, batches = r.appendDebugOverlay(vertices, batches, lines, virtualWidth, virtualHeight)
	return vertices, batches
}

func appendOrMergeBatch(batches []drawBatch, texture *gpuTexture, firstVertex uint32, vertexCount uint32) []drawBatch {
	if texture == nil || vertexCount == 0 {
		return batches
	}
	if len(batches) > 0 {
		last := &batches[len(batches)-1]
		if last.texture == texture && last.firstVertex+last.vertexCount == firstVertex {
			last.vertexCount += vertexCount
			return batches
		}
	}
	return append(batches, drawBatch{
		texture:     texture,
		firstVertex: firstVertex,
		vertexCount: vertexCount,
	})
}

func appendQuad(vertices []quadVertex, dst geom.Rect, src geom.Rect, color [4]float32, virtualWidth, virtualHeight float64, flipH, flipV bool, textureWidth, textureHeight int) []quadVertex {
	if src.W == 0 {
		src.W = float64(textureWidth)
	}
	if src.H == 0 {
		src.H = float64(textureHeight)
	}
	u0 := float32(src.X / float64(textureWidth))
	v0 := float32(src.Y / float64(textureHeight))
	u1 := float32((src.X + src.W) / float64(textureWidth))
	v1 := float32((src.Y + src.H) / float64(textureHeight))
	if flipH {
		u0, u1 = u1, u0
	}
	if flipV {
		v0, v1 = v1, v0
	}

	x0 := clipX(dst.X, virtualWidth)
	y0 := clipY(dst.Y, virtualHeight)
	x1 := clipX(dst.X+dst.W, virtualWidth)
	y1 := clipY(dst.Y+dst.H, virtualHeight)

	topLeft := quadVertex{Position: [2]float32{x0, y0}, UV: [2]float32{u0, v0}, Color: color}
	topRight := quadVertex{Position: [2]float32{x1, y0}, UV: [2]float32{u1, v0}, Color: color}
	bottomLeft := quadVertex{Position: [2]float32{x0, y1}, UV: [2]float32{u0, v1}, Color: color}
	bottomRight := quadVertex{Position: [2]float32{x1, y1}, UV: [2]float32{u1, v1}, Color: color}

	return append(vertices,
		topLeft,
		bottomLeft,
		topRight,
		topRight,
		bottomLeft,
		bottomRight,
	)
}

func (r *Renderer2D) appendDebugOverlay(vertices []quadVertex, batches []drawBatch, lines []string, virtualWidth, virtualHeight float64) ([]quadVertex, []drawBatch) {
	if r == nil || r.debugFont == nil || r.debugFont.texture == nil || len(lines) == 0 {
		return vertices, batches
	}

	layout := r.overlayLayout(virtualWidth)
	visibleLines := wrapOverlayLines(lines, layout.maxColumns)
	longest := 0
	for _, line := range visibleLines {
		if n := len([]rune(line)); n > longest {
			longest = n
		}
	}
	if len(visibleLines) == 0 || longest == 0 {
		return vertices, batches
	}

	backgroundRect := geom.Rect{
		X: layout.margin,
		Y: layout.margin,
		W: float64(longest)*layout.glyphWidth + layout.padding*2,
		H: float64(len(visibleLines))*layout.lineHeight + layout.padding*2,
	}
	first := uint32(len(vertices))
	vertices = appendQuad(vertices, backgroundRect, geom.Rect{W: 1, H: 1}, layout.backgroundColor, virtualWidth, virtualHeight, false, false, 1, 1)
	batches = appendOrMergeBatch(batches, r.whiteTexture, first, 6)

	baseX := layout.margin + layout.padding
	baseY := layout.margin + layout.padding
	for lineIndex, line := range visibleLines {
		cursorX := baseX
		cursorY := baseY + float64(lineIndex)*layout.lineHeight
		for _, rawRune := range line {
			if rawRune == ' ' {
				cursorX += layout.glyphWidth
				continue
			}
			glyphRune := normalizeOverlayRune(rawRune)
			src, ok := r.debugFont.glyphs[glyphRune]
			if !ok {
				cursorX += layout.glyphWidth
				continue
			}

			dst := geom.Rect{
				X: cursorX,
				Y: cursorY,
				W: layout.glyphWidth,
				H: layout.glyphHeight,
			}
			first = uint32(len(vertices))
			vertices = appendQuad(vertices, dst, src, layout.textColor, virtualWidth, virtualHeight, false, false, r.debugFont.texture.width, r.debugFont.texture.height)
			batches = appendOrMergeBatch(batches, r.debugFont.texture, first, 6)
			cursorX += layout.glyphWidth
		}
	}

	return vertices, batches
}

func normalizeOverlayRune(value rune) rune {
	switch {
	case value >= 32 && value <= 126:
		return value
	default:
		return '?'
	}
}

func (r *Renderer2D) overlayLayout(virtualWidth float64) overlayLayout {
	maxColumns := 40
	if virtualWidth > 0 && r.debugFont != nil && r.debugFont.glyphWidth > 0 {
		scaledGlyphWidth := float64(r.debugFont.glyphWidth) * textOverlayScale
		usableWidth := virtualWidth * textOverlayMaxWidthRatio
		if scaledGlyphWidth > 0 {
			columns := int((usableWidth - float64(textOverlayPadding*2)) / scaledGlyphWidth)
			if columns > 12 {
				maxColumns = columns
			}
		}
	}
	return overlayLayout{
		scale:           textOverlayScale,
		glyphWidth:      float64(r.debugFont.glyphWidth) * textOverlayScale,
		glyphHeight:     float64(r.debugFont.glyphHeight) * textOverlayScale,
		lineHeight:      float64(r.debugFont.lineHeight)*textOverlayScale + 1,
		padding:         float64(textOverlayPadding),
		margin:          float64(textOverlayMargin),
		maxColumns:      maxColumns,
		backgroundColor: [4]float32{0, 0, 0, 0.52},
		textColor:       [4]float32{0.96, 0.97, 0.99, 0.92},
	}
}

func wrapOverlayLines(lines []string, maxColumns int) []string {
	if maxColumns <= 0 {
		maxColumns = 40
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			out = append(out, " ")
			continue
		}
		out = append(out, wrapOverlayLine(line, maxColumns)...)
	}
	return out
}

func wrapOverlayLine(line string, maxColumns int) []string {
	runes := []rune(line)
	if len(runes) <= maxColumns {
		return []string{line}
	}

	out := make([]string, 0, len(runes)/maxColumns+1)
	for len(runes) > maxColumns {
		split := maxColumns
		for index := maxColumns; index > maxColumns/2; index-- {
			if runes[index] == ' ' {
				split = index
				break
			}
		}
		out = append(out, string(trimTrailingSpaces(runes[:split])))
		runes = trimLeadingSpaces(runes[split:])
	}
	if len(runes) > 0 {
		out = append(out, string(runes))
	}
	return out
}

func trimLeadingSpaces(runes []rune) []rune {
	for len(runes) > 0 && runes[0] == ' ' {
		runes = runes[1:]
	}
	return runes
}

func trimTrailingSpaces(runes []rune) []rune {
	for len(runes) > 0 && runes[len(runes)-1] == ' ' {
		runes = runes[:len(runes)-1]
	}
	return runes
}

func clipX(x float64, width float64) float32 {
	if width == 0 {
		return -1
	}
	return float32((x/width)*2 - 1)
}

func clipY(y float64, height float64) float32 {
	if height == 0 {
		return -1
	}
	return float32((y/height)*2 - 1)
}

func whiteColor() [4]float32 {
	return [4]float32{1, 1, 1, 1}
}

func quadColor(color geom.Color) [4]float32 {
	return [4]float32{color.R, color.G, color.B, color.A}
}

func vertexBufferSize(vertexCount int) vkDeviceSize {
	return vkDeviceSize(vertexCount) * vkDeviceSize(unsafe.Sizeof(quadVertex{}))
}
