package metal

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"github.com/IsraelAraujo70/whisky-game-engine/geom"
	"github.com/IsraelAraujo70/whisky-game-engine/render"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	textOverlayMargin        = 3
	textOverlayPadding       = 2
	textOverlayScale         = 0.75
	textOverlayMaxWidthRatio = 0.52
)

type softwareTexture struct {
	id     render.TextureID
	width  int
	height int
	rgba   *image.RGBA
}

type bitmapFont struct {
	texture     *softwareTexture
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

type presentationLayout struct {
	viewportX      float64
	viewportY      float64
	viewportWidth  float64
	viewportHeight float64
	scissorX       uint64
	scissorY       uint64
	scissorWidth   uint64
	scissorHeight  uint64
}

type quadVertex struct {
	Position [2]float32
	UV       [2]float32
	Color    [4]float32
}

type drawBatch struct {
	texture     *softwareTexture
	firstVertex uint32
	vertexCount uint32
}

type softwareRenderer struct {
	texturesByPath map[string]*softwareTexture
	texturesByID   map[render.TextureID]*softwareTexture
	nextTextureID  render.TextureID
	whiteTexture   *softwareTexture
	debugFont      *bitmapFont
	virtualWidth   int
	virtualHeight  int
	pixelPerfect   bool
}

func newSoftwareRenderer() (*softwareRenderer, error) {
	font, err := createDebugFont()
	if err != nil {
		return nil, err
	}
	return &softwareRenderer{
		texturesByPath: make(map[string]*softwareTexture),
		texturesByID:   make(map[render.TextureID]*softwareTexture),
		whiteTexture:   newSolidTexture(255, 255, 255, 255),
		debugFont:      font,
	}, nil
}

func (r *softwareRenderer) setLogicalSize(width, height int, pixelPerfect bool) {
	r.virtualWidth = width
	r.virtualHeight = height
	r.pixelPerfect = pixelPerfect
}

func (r *softwareRenderer) loadTexture(path string) (render.TextureID, int, int, error) {
	if r == nil {
		return 0, 0, 0, fmt.Errorf("metal: renderer is nil")
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return 0, 0, 0, err
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()

	src, err := png.Decode(file)
	if err != nil {
		return 0, 0, 0, err
	}
	rgba := imageToRGBA(src)

	if old, ok := r.texturesByPath[cleanPath]; ok {
		// Re-upload: replace pixel data in-place preserving TextureID.
		// The GPU texture will be re-created lazily on next use.
		old.rgba = rgba
		old.width = rgba.Bounds().Dx()
		old.height = rgba.Bounds().Dy()
		return old.id, old.width, old.height, nil
	}

	r.nextTextureID++
	texture := &softwareTexture{
		id:     r.nextTextureID,
		width:  rgba.Bounds().Dx(),
		height: rgba.Bounds().Dy(),
		rgba:   rgba,
	}
	r.texturesByPath[cleanPath] = texture
	r.texturesByID[texture.id] = texture
	return texture.id, texture.width, texture.height, nil
}

func (r *softwareRenderer) buildDrawData(cmds []render.DrawCmd, lines []string, fallbackWidth, fallbackHeight int) ([]quadVertex, []drawBatch, int, int, error) {
	if r == nil {
		return nil, nil, 0, 0, fmt.Errorf("metal: renderer is nil")
	}
	virtualWidth, virtualHeight, err := r.drawSize(fallbackWidth, fallbackHeight)
	if err != nil {
		return nil, nil, 0, 0, err
	}

	vertices := make([]quadVertex, 0, len(cmds)*6)
	batches := make([]drawBatch, 0, len(cmds)+1)
	for _, cmd := range cmds {
		switch drawCmd := cmd.(type) {
		case render.FillRect:
			first := uint32(len(vertices))
			vertices = appendQuad(vertices, drawCmd.Rect, geom.Rect{W: 1, H: 1}, quadColor(drawCmd.Color), false, false, 1, 1)
			batches = appendOrMergeBatch(batches, r.whiteTexture, first, 6)
		case render.SpriteCmd:
			texture := r.texturesByID[drawCmd.Texture]
			if texture == nil {
				continue
			}
			first := uint32(len(vertices))
			vertices = appendQuad(vertices, drawCmd.Dst, drawCmd.Src, whiteColor(), drawCmd.FlipH, drawCmd.FlipV, texture.width, texture.height)
			batches = appendOrMergeBatch(batches, texture, first, 6)
		case render.TextCmd:
			vertices, batches = r.appendTextCmd(vertices, batches, drawCmd)
		}
	}
	vertices, batches = r.appendDebugOverlay(vertices, batches, lines, float64(virtualWidth), float64(virtualHeight))
	return vertices, batches, virtualWidth, virtualHeight, nil
}

func (r *softwareRenderer) drawSize(fallbackWidth, fallbackHeight int) (int, int, error) {
	if r == nil {
		return 0, 0, fmt.Errorf("metal: renderer is nil")
	}
	width := r.virtualWidth
	height := r.virtualHeight
	if width <= 0 {
		width = fallbackWidth
	}
	if height <= 0 {
		height = fallbackHeight
	}
	if width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("metal: invalid framebuffer size %dx%d", width, height)
	}
	return width, height, nil
}

func appendOrMergeBatch(batches []drawBatch, texture *softwareTexture, firstVertex uint32, vertexCount uint32) []drawBatch {
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
	return append(batches, drawBatch{texture: texture, firstVertex: firstVertex, vertexCount: vertexCount})
}

func appendQuad(vertices []quadVertex, dst geom.Rect, src geom.Rect, color [4]float32, flipH, flipV bool, textureWidth, textureHeight int) []quadVertex {
	if textureWidth <= 0 || textureHeight <= 0 {
		return vertices
	}
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

	x0 := float32(dst.X)
	y0 := float32(dst.Y)
	x1 := float32(dst.X + dst.W)
	y1 := float32(dst.Y + dst.H)

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

func (r *softwareRenderer) appendTextCmd(vertices []quadVertex, batches []drawBatch, tc render.TextCmd) ([]quadVertex, []drawBatch) {
	if r == nil || r.debugFont == nil || r.debugFont.texture == nil {
		return vertices, batches
	}
	scale := tc.Scale
	if scale <= 0 {
		scale = 1
	}
	glyphW := float64(r.debugFont.glyphWidth) * scale
	glyphH := float64(r.debugFont.glyphHeight) * scale
	color := quadColor(tc.Color)
	cursorX := tc.Pos.X
	cursorY := tc.Pos.Y

	for _, ch := range tc.Text {
		if ch == ' ' {
			cursorX += glyphW
			continue
		}
		glyphRune := normalizeOverlayRune(ch)
		src, ok := r.debugFont.glyphs[glyphRune]
		if !ok {
			cursorX += glyphW
			continue
		}
		dst := geom.Rect{X: cursorX, Y: cursorY, W: glyphW, H: glyphH}
		first := uint32(len(vertices))
		vertices = appendQuad(vertices, dst, src, color, false, false, r.debugFont.texture.width, r.debugFont.texture.height)
		batches = appendOrMergeBatch(batches, r.debugFont.texture, first, 6)
		cursorX += glyphW
	}
	return vertices, batches
}

func (r *softwareRenderer) appendDebugOverlay(vertices []quadVertex, batches []drawBatch, lines []string, virtualWidth, virtualHeight float64) ([]quadVertex, []drawBatch) {
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
	vertices = appendQuad(vertices, backgroundRect, geom.Rect{W: 1, H: 1}, layout.backgroundColor, false, false, 1, 1)
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

			dst := geom.Rect{X: cursorX, Y: cursorY, W: layout.glyphWidth, H: layout.glyphHeight}
			first = uint32(len(vertices))
			vertices = appendQuad(vertices, dst, src, layout.textColor, false, false, r.debugFont.texture.width, r.debugFont.texture.height)
			batches = appendOrMergeBatch(batches, r.debugFont.texture, first, 6)
			cursorX += layout.glyphWidth
		}
	}

	_ = virtualHeight
	return vertices, batches
}

func (r *softwareRenderer) overlayLayout(virtualWidth float64) overlayLayout {
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

func computePresentationLayout(targetWidth, targetHeight, logicalWidth, logicalHeight int, pixelPerfect bool) presentationLayout {
	if targetWidth <= 0 || targetHeight <= 0 {
		return presentationLayout{}
	}
	if logicalWidth <= 0 {
		logicalWidth = targetWidth
	}
	if logicalHeight <= 0 {
		logicalHeight = targetHeight
	}

	targetW := float64(targetWidth)
	targetH := float64(targetHeight)
	logicalW := float64(logicalWidth)
	logicalH := float64(logicalHeight)
	scale := math.Min(targetW/logicalW, targetH/logicalH)
	if pixelPerfect && scale >= 1 {
		scale = math.Floor(scale)
		if scale < 1 {
			scale = 1
		}
	}
	viewportWidth := logicalW * scale
	viewportHeight := logicalH * scale
	offsetX := (targetW - viewportWidth) * 0.5
	offsetY := (targetH - viewportHeight) * 0.5

	return presentationLayout{
		viewportX:      offsetX,
		viewportY:      offsetY,
		viewportWidth:  viewportWidth,
		viewportHeight: viewportHeight,
		scissorX:       uint64(math.Round(offsetX)),
		scissorY:       uint64(math.Round(offsetY)),
		scissorWidth:   uint64(math.Round(viewportWidth)),
		scissorHeight:  uint64(math.Round(viewportHeight)),
	}
}

func whiteColor() [4]float32 {
	return [4]float32{1, 1, 1, 1}
}

func quadColor(color geom.Color) [4]float32 {
	return [4]float32{color.R, color.G, color.B, color.A}
}

func imageToRGBA(src image.Image) *image.RGBA {
	if rgba, ok := src.(*image.RGBA); ok {
		return rgba
	}
	bounds := src.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, src, bounds.Min, draw.Src)
	return rgba
}

func newSolidTexture(r, g, b, a uint8) *softwareTexture {
	rgba := image.NewRGBA(image.Rect(0, 0, 1, 1))
	rgba.Pix[0] = r
	rgba.Pix[1] = g
	rgba.Pix[2] = b
	rgba.Pix[3] = a
	return &softwareTexture{width: 1, height: 1, rgba: rgba}
}

func createDebugFont() (*bitmapFont, error) {
	face := basicfont.Face7x13
	printables := make([]rune, 0, 95)
	for value := rune(32); value <= 126; value++ {
		printables = append(printables, value)
	}

	cols := 16
	rows := (len(printables) + cols - 1) / cols
	atlasWidth := cols * face.Advance
	atlasHeight := rows * face.Height
	atlas := image.NewRGBA(image.Rect(0, 0, atlasWidth, atlasHeight))
	drawer := &font.Drawer{Dst: atlas, Src: image.White, Face: face}

	glyphs := make(map[rune]geom.Rect, len(printables))
	for index, value := range printables {
		col := index % cols
		row := index / cols
		x := col * face.Advance
		y := row * face.Height
		drawer.Dot = fixed.P(x, y+face.Ascent)
		drawer.DrawString(string(value))
		glyphs[value] = geom.Rect{X: float64(x), Y: float64(y), W: float64(face.Width), H: float64(face.Height)}
	}

	return &bitmapFont{
		texture:     &softwareTexture{width: atlas.Bounds().Dx(), height: atlas.Bounds().Dy(), rgba: atlas},
		glyphWidth:  face.Advance,
		glyphHeight: face.Height,
		lineHeight:  face.Height + 2,
		glyphs:      glyphs,
	}, nil
}

func normalizeOverlayRune(value rune) rune {
	switch {
	case value >= 32 && value <= 126:
		return value
	default:
		return '?'
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
