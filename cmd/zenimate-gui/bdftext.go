package main

import (
	"image"
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/ha1tch/zenimate/pkg/bdf"
)

// bdfText renders text using a BDF bitmap font through raylib textures. It never
// touches raylib's own font facilities: every glyph is rasterised by pkg/bdf
// into an RGBA cell, uploaded once as a raylib texture, cached by codepoint, and
// drawn as a textured rectangle. This keeps all on-screen text in the period
// ZX Spectrum face and honours the "no raylib native fonts" rule.
type bdfText struct {
	font  *bdf.Font
	cellW int
	cellH int
	ink   color.NRGBA
	cache map[rune]rl.Texture2D
}

// newBDFText builds a text renderer for the given font and ink colour. Glyph
// textures are created lazily on first use; the GL context must be current (i.e.
// call this after rl.InitWindow).
func newBDFText(font *bdf.Font, ink color.NRGBA) *bdfText {
	return &bdfText{
		font:  font,
		cellW: font.CellWidth,
		cellH: font.CellHeight,
		ink:   ink,
		cache: make(map[rune]rl.Texture2D),
	}
}

// CellW and CellH are the font's cell dimensions in source pixels.
func (t *bdfText) CellW() int { return t.cellW }
func (t *bdfText) CellH() int { return t.cellH }

// glyph returns (creating if needed) the texture for rune r. The zero Texture2D
// is returned for glyphs the font lacks, which draw as nothing.
func (t *bdfText) glyph(r rune) rl.Texture2D {
	if tex, ok := t.cache[r]; ok {
		return tex
	}
	img, ok := t.font.GlyphImage(r, t.ink)
	if !ok {
		// Cache a blank so we do not retry every frame.
		blank := image.NewRGBA(image.Rect(0, 0, t.cellW, t.cellH))
		tex := texFromRGBA(blank)
		t.cache[r] = tex
		return tex
	}
	tex := texFromRGBA(img)
	t.cache[r] = tex
	return tex
}

// Draw renders s with its top-left at (x,y), scaled by an integer factor. It
// returns the x position just past the last glyph.
func (t *bdfText) Draw(s string, x, y, scale int, tint rl.Color) int {
	if scale < 1 {
		scale = 1
	}
	cx := x
	for _, r := range s {
		tex := t.glyph(r)
		dst := rl.NewRectangle(float32(cx), float32(y), float32(t.cellW*scale), float32(t.cellH*scale))
		src := rl.NewRectangle(0, 0, float32(t.cellW), float32(t.cellH))
		rl.DrawTexturePro(tex, src, dst, rl.NewVector2(0, 0), 0, tint)
		cx += t.cellW * scale
	}
	return cx
}

// Measure returns the pixel width of s at the given scale.
func (t *bdfText) Measure(s string, scale int) int {
	if scale < 1 {
		scale = 1
	}
	n := 0
	for range s {
		n++
	}
	return n * t.cellW * scale
}

// Unload frees every cached glyph texture. Call before rl.CloseWindow.
func (t *bdfText) Unload() {
	for _, tex := range t.cache {
		rl.UnloadTexture(tex)
	}
	t.cache = map[rune]rl.Texture2D{}
}

// texFromRGBA uploads a Go RGBA image to a raylib texture.
func texFromRGBA(img *image.RGBA) rl.Texture2D {
	rlImg := rl.NewImageFromImage(img)
	tex := rl.LoadTextureFromImage(rlImg)
	rl.UnloadImage(rlImg)
	return tex
}
