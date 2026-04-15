package sdl3

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/Zyko0/go-sdl3/sdl"

	"github.com/IsraelAraujo70/whisky-game-engine/render"
)

type textureEntry struct {
	tex *sdl.Texture
	w   int
	h   int
}

type textureCache struct {
	renderer *sdl.Renderer
	textures map[render.TextureID]textureEntry
	paths    map[string]render.TextureID
	nextID   render.TextureID
}

func newTextureCache(renderer *sdl.Renderer) *textureCache {
	return &textureCache{
		renderer: renderer,
		textures: make(map[render.TextureID]textureEntry),
		paths:    make(map[string]render.TextureID),
	}
}

func (tc *textureCache) Load(path string) (render.TextureID, int, int, error) {
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return 0, 0, 0, err
	}
	if id, ok := tc.paths[cleanPath]; ok {
		entry := tc.textures[id]
		return id, entry.w, entry.h, nil
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return 0, 0, 0, err
	}

	nrgba := toNRGBA(img)
	surface, err := sdl.CreateSurfaceFrom(
		nrgba.Bounds().Dx(),
		nrgba.Bounds().Dy(),
		sdl.PIXELFORMAT_RGBA32,
		nrgba.Pix,
		nrgba.Stride,
	)
	if err != nil {
		return 0, 0, 0, err
	}
	defer surface.Destroy()

	texture, err := tc.renderer.CreateTextureFromSurface(surface)
	if err != nil {
		return 0, 0, 0, err
	}

	tc.nextID++
	id := tc.nextID
	tc.textures[id] = textureEntry{
		tex: texture,
		w:   nrgba.Bounds().Dx(),
		h:   nrgba.Bounds().Dy(),
	}
	tc.paths[cleanPath] = id

	return id, nrgba.Bounds().Dx(), nrgba.Bounds().Dy(), nil
}

func (tc *textureCache) Get(id render.TextureID) *sdl.Texture {
	entry, ok := tc.textures[id]
	if !ok {
		return nil
	}
	return entry.tex
}

// ReuploadTexture replaces the GPU texture for an existing ID with new pixel
// data decoded from img. The old SDL texture is destroyed. Returns an error if
// the ID is unknown or the upload fails.
func (tc *textureCache) ReuploadTexture(id render.TextureID, img image.Image) error {
	existing, ok := tc.textures[id]
	if !ok {
		return nil // unknown ID — silently skip
	}

	nrgba := toNRGBA(img)
	surface, err := sdl.CreateSurfaceFrom(
		nrgba.Bounds().Dx(),
		nrgba.Bounds().Dy(),
		sdl.PIXELFORMAT_RGBA32,
		nrgba.Pix,
		nrgba.Stride,
	)
	if err != nil {
		return err
	}
	defer surface.Destroy()

	newTex, err := tc.renderer.CreateTextureFromSurface(surface)
	if err != nil {
		return err
	}

	// Destroy old texture and swap in the new one.
	if existing.tex != nil {
		existing.tex.Destroy()
	}
	tc.textures[id] = textureEntry{
		tex: newTex,
		w:   nrgba.Bounds().Dx(),
		h:   nrgba.Bounds().Dy(),
	}
	return nil
}

// IDForPath returns the texture ID currently associated with the given absolute
// path, or 0 if none.
func (tc *textureCache) IDForPath(absPath string) render.TextureID {
	return tc.paths[absPath]
}

func (tc *textureCache) DestroyAll() {
	for id, entry := range tc.textures {
		if entry.tex != nil {
			entry.tex.Destroy()
		}
		delete(tc.textures, id)
	}
	for path := range tc.paths {
		delete(tc.paths, path)
	}
	tc.nextID = 0
}

func toNRGBA(src image.Image) *image.NRGBA {
	if img, ok := src.(*image.NRGBA); ok {
		return img
	}

	bounds := src.Bounds()
	dst := image.NewNRGBA(bounds)
	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)
	return dst
}
