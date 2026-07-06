package fonts

import (
	"image/color"
	"testing"
)

func TestSinclairParses(t *testing.T) {
	f, err := Sinclair()
	if err != nil {
		t.Fatalf("Sinclair() error: %v", err)
	}
	if f.Glyphs() == 0 {
		t.Error("Sinclair() has zero glyphs")
	}
}

func TestCozetteParses(t *testing.T) {
	f, err := Cozette()
	if err != nil {
		t.Fatalf("Cozette() error: %v", err)
	}
	if f.Glyphs() == 0 {
		t.Error("Cozette() has zero glyphs")
	}
}

func TestTomThumbParses(t *testing.T) {
	f, err := TomThumb()
	if err != nil {
		t.Fatalf("TomThumb() error: %v", err)
	}
	if f.Glyphs() == 0 {
		t.Error("TomThumb() has zero glyphs")
	}
	if _, ok := f.GlyphImage('A', color.NRGBA{A: 255}); !ok {
		t.Error("TomThumb() is missing 'A'")
	}
}

func TestSpleen5x8Parses(t *testing.T) {
	f, err := Spleen5x8()
	if err != nil {
		t.Fatalf("Spleen5x8() error: %v", err)
	}
	if f.Glyphs() == 0 {
		t.Error("Spleen5x8() has zero glyphs")
	}
	if _, ok := f.GlyphImage('A', color.NRGBA{A: 255}); !ok {
		t.Error("Spleen5x8() is missing 'A'")
	}
}

func TestCreepParses(t *testing.T) {
	f, err := Creep()
	if err != nil {
		t.Fatalf("Creep() error: %v", err)
	}
	if f.Glyphs() == 0 {
		t.Error("Creep() has zero glyphs")
	}
	if _, ok := f.GlyphImage('A', color.NRGBA{A: 255}); !ok {
		t.Error("Creep() is missing 'A'")
	}
}

func TestHaxorMediumParses(t *testing.T) {
	f, err := HaxorMedium()
	if err != nil {
		t.Fatalf("HaxorMedium() error: %v", err)
	}
	if f.Glyphs() == 0 {
		t.Error("HaxorMedium() has zero glyphs")
	}
	if _, ok := f.GlyphImage('A', color.NRGBA{A: 255}); !ok {
		t.Error("HaxorMedium() is missing 'A'")
	}
}

// TestToolIconsHasAllExpectedGlyphs verifies every icon is present at both
// sizes, at the exact codepoints the eventual ToolPalette component will look
// them up by — the fixed order defined in cmd/zenimate-gui's tool list.
func TestToolIconsHasAllExpectedGlyphs(t *testing.T) {
	f, err := ToolIcons()
	if err != nil {
		t.Fatalf("ToolIcons() error: %v", err)
	}
	if got := f.Glyphs(); got != 26 {
		t.Errorf("ToolIcons() has %d glyphs, want 26 (13 icons x 2 sizes)", got)
	}

	// customshape stays in the font even though the toolbar no longer uses
	// it directly — it's reserved for the future pen brush-shape picker.
	names := []string{"select", "paintbrush", "fill", "eyedropper", "line", "rectangle",
		"ellipse", "triangle", "polygon", "customshape", "hand", "zoom", "text"}
	ink := color.NRGBA{R: 0, G: 0, B: 0, A: 255}

	for size, base := range map[int]rune{32: 0xE000, 24: 0xE100} {
		for i, name := range names {
			cp := base + rune(i)
			if !f.Has(cp) {
				t.Errorf("missing glyph: size=%d name=%s codepoint=U+%04X", size, name, cp)
				continue
			}
			img, ok := f.GlyphImage(cp, ink)
			if !ok {
				t.Errorf("GlyphImage reported not-present for size=%d name=%s codepoint=U+%04X", size, name, cp)
			}
			// The font's cell is fixed at 32x32 (the larger of the two sizes) —
			// both the 32px and 24px glyphs render into that same cell, the 24px
			// ones just occupying less of it. See BBX/DWIDTH in icons.bdf.
			if b := img.Bounds(); b.Dx() != 32 || b.Dy() != 32 {
				t.Errorf("size=%d name=%s: image bounds = %v, want 32x32 cell", size, name, b)
			}
		}
	}
}

// TestToolIconsRejectsUnknownCodepoint confirms a codepoint outside the
// defined range behaves as "absent" rather than panicking or silently
// returning a wrong glyph — the same not-present contract every other BDF
// lookup in this codebase relies on.
func TestToolIconsRejectsUnknownCodepoint(t *testing.T) {
	f, err := ToolIcons()
	if err != nil {
		t.Fatalf("ToolIcons() error: %v", err)
	}
	if f.Has(0xE200) {
		t.Error("Has(0xE200) = true, want false (no icon defined at that codepoint)")
	}
	_, ok := f.GlyphImage(0xE200, color.NRGBA{A: 255})
	if ok {
		t.Error("GlyphImage(0xE200) reported present, want not-present")
	}
}
