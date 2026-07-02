package bdf

import (
	"image/color"
	"strings"
	"testing"
)

// A minimal 8x8 BDF with a single glyph 'A' (encoding 65). The bitmap is a
// solid top row and a solid bottom row; everything else clear. This exercises
// the parser and the rasteriser without depending on a file.
const miniBDF = `STARTFONT 2.1
FONT -test-Mono-Medium-R-Normal--8-80-75-75-C-80-ISO8859-1
SIZE 8 75 75
FONTBOUNDINGBOX 8 8 0 0
STARTPROPERTIES 2
FONT_ASCENT 8
FONT_DESCENT 0
ENDPROPERTIES
CHARS 1
STARTCHAR A
ENCODING 65
SWIDTH 640 0
DWIDTH 8 0
BBX 8 8 0 0
BITMAP
FF
00
00
00
00
00
00
FF
ENDCHAR
ENDFONT
`

func TestParseMini(t *testing.T) {
	f, err := Parse(strings.NewReader(miniBDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if f.CellWidth != 8 || f.CellHeight != 8 {
		t.Fatalf("cell = %dx%d, want 8x8", f.CellWidth, f.CellHeight)
	}
	if !f.Has('A') {
		t.Fatalf("expected glyph for 'A'")
	}
	if f.Has('B') {
		t.Fatalf("did not expect glyph for 'B'")
	}

	ink := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	img, ok := f.GlyphImage('A', ink)
	if !ok {
		t.Fatalf("GlyphImage('A') ok=false")
	}
	b := img.Bounds()
	if b.Dx() != 8 || b.Dy() != 8 {
		t.Fatalf("image = %dx%d, want 8x8", b.Dx(), b.Dy())
	}

	// Top row (y=0) should be fully inked; the row below (y=1) fully clear.
	for x := 0; x < 8; x++ {
		if _, _, _, a := img.At(x, 0).RGBA(); a == 0 {
			t.Fatalf("top row pixel (%d,0) should be inked", x)
		}
		if _, _, _, a := img.At(x, 1).RGBA(); a != 0 {
			t.Fatalf("row pixel (%d,1) should be clear", x)
		}
	}
}

func TestGlyphImageMissingReturnsBlank(t *testing.T) {
	f, err := Parse(strings.NewReader(miniBDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ink := color.NRGBA{R: 255, A: 255}
	img, ok := f.GlyphImage('Z', ink)
	if ok {
		t.Fatalf("expected ok=false for missing glyph")
	}
	if img == nil {
		t.Fatalf("expected a blank image, got nil")
	}
	for x := 0; x < img.Bounds().Dx(); x++ {
		for y := 0; y < img.Bounds().Dy(); y++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0 {
				t.Fatalf("blank cell pixel (%d,%d) should be transparent", x, y)
			}
		}
	}
}

func TestParseSinclair(t *testing.T) {
	f, err := ParseFile("testdata/sinclair.bdf")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if f.CellWidth != 8 || f.CellHeight != 8 {
		t.Fatalf("sinclair cell = %dx%d, want 8x8", f.CellWidth, f.CellHeight)
	}
	if got := f.Glyphs(); got != 96 {
		t.Fatalf("sinclair glyphs = %d, want 96", got)
	}
	for _, r := range "ABZ09 ?" {
		if !f.Has(r) {
			t.Errorf("sinclair missing expected glyph %q", r)
		}
	}
	ink := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	if _, ok := f.GlyphImage('A', ink); !ok {
		t.Fatalf("GlyphImage('A') ok=false on sinclair")
	}
}
