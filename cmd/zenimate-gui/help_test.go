//go:build purego

package main

import (
	"strings"
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

func TestHelpModalHasContent(t *testing.T) {
	h := newHelpModal()
	if len(h.lines) < 20 {
		t.Fatalf("help text seems too short: %d lines", len(h.lines))
	}
	joined := strings.Join(h.lines, "\n")
	// Sanity: the help should mention the formats and key shortcuts.
	for _, want := range []string{".zani", ".zbun", "Ctrl+S", "Bitmap", "Spectrum"} {
		if !strings.Contains(joined, want) {
			t.Errorf("help text missing %q", want)
		}
	}
}

func TestHelpModalScrollClamps(t *testing.T) {
	h := newHelpModal()
	h.layout(stubRenderer{}, 800, 600)
	// Scroll far past the end; clamp must keep it within [0, total-visible].
	h.scroll = 100000
	h.clampScroll()
	maxScroll := h.total - h.visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if h.scroll != maxScroll {
		t.Errorf("scroll clamp: got %d want %d", h.scroll, maxScroll)
	}
	// Scroll before the start.
	h.scroll = -50
	h.clampScroll()
	if h.scroll != 0 {
		t.Errorf("scroll clamp low: got %d want 0", h.scroll)
	}
}

func TestHelpModalEscCloses(t *testing.T) {
	h := newHelpModal()
	h.layout(stubRenderer{}, 800, 600)
	open := h.update(zenui.Input{Keys: []zenui.Key{zenui.KeyEscape}})
	if open {
		t.Error("Esc should close the help modal")
	}
}

func TestHelpModalClickOutsideCloses(t *testing.T) {
	h := newHelpModal()
	h.layout(stubRenderer{}, 800, 600)
	// A click well outside the panel closes it.
	open := h.update(zenui.Input{MouseX: 0, MouseY: 0, MousePressed: true})
	if open {
		t.Error("click outside panel should close the help modal")
	}
}

func TestChequerShade(t *testing.T) {
	base := rl.NewColor(0xb0, 0xb0, 0xb0, 0xff)
	// White mode: two notches darker.
	w := shadeChequer(base, ui.BitmapWhite)
	if w.R >= base.R {
		t.Error("Bitmap White chequer should be darker")
	}
	if int(base.R)-int(w.R) != 2*chequerNotch {
		t.Errorf("white shade delta = %d, want %d", int(base.R)-int(w.R), 2*chequerNotch)
	}
	// Black mode: two notches lighter.
	b := shadeChequer(base, ui.BitmapBlack)
	if b.R <= base.R {
		t.Error("Bitmap Black chequer should be lighter")
	}
	if int(b.R)-int(base.R) != 2*chequerNotch {
		t.Errorf("black shade delta = %d, want %d", int(b.R)-int(base.R), 2*chequerNotch)
	}
}

func TestHelpBodyScaledLineHeight(t *testing.T) {
	h := newHelpModal()
	// Wide screen: effective scale is the base (2). stub LineHeight = 8*scale.
	h.layout(stubRenderer{}, 2000, 1200)
	if h.effScale != helpBodyScaleBase {
		t.Fatalf("wide screen effScale = %d, want %d", h.effScale, helpBodyScaleBase)
	}
	if h.bodyLH != 8*helpBodyScaleBase {
		t.Errorf("body line height = %d, want %d", h.bodyLH, 8*helpBodyScaleBase)
	}
}

func TestHelpScrollbarDragMapping(t *testing.T) {
	h := newHelpModal()
	h.total = 200
	h.visible = 20
	h.track = zenui.Rect{X: 500, Y: 100, W: 8, H: 400}
	h.thumb = zenui.Rect{X: 500, Y: 100, W: 8, H: 40}

	// Press on the thumb (grab), offset 0.
	h.update(zenui.Input{MouseX: 504, MouseY: 100, MousePressed: true, MouseDown: true})
	if !h.dragging {
		t.Fatal("pressing the thumb should start dragging")
	}
	// Drag to the very bottom of the track: scroll should reach max (total-visible).
	h.update(zenui.Input{MouseX: 504, MouseY: 100 + 400, MouseDown: true})
	if h.scroll != h.total-h.visible {
		t.Errorf("drag to bottom: scroll=%d want %d", h.scroll, h.total-h.visible)
	}
	// Release stops dragging.
	h.update(zenui.Input{MouseX: 504, MouseY: 500, MouseDown: false})
	if h.dragging {
		t.Error("releasing should stop dragging")
	}
}

func TestMacWheelSign(t *testing.T) {
	// wheelSign depends on macDetected; verify the mapping is one of the two
	// expected values and that inversion is exactly a negation.
	s := wheelSign()
	if macDetected && s != -1 {
		t.Errorf("on detected Mac, wheelSign()=%v want -1", s)
	}
	if !macDetected && s != 1 {
		t.Errorf("off Mac, wheelSign()=%v want 1", s)
	}
}

func TestHelpBodyScaleIsInteger(t *testing.T) {
	// The base scale must be a whole number so bitmap glyphs stay crisp.
	if helpBodyScaleBase != 2 && helpBodyScaleBase != 3 {
		t.Errorf("helpBodyScaleBase = %d; must be integer 2 or 3", helpBodyScaleBase)
	}
	// A screen wide enough for the x2 panel uses the base scale; a narrow one
	// drops to 1. Width is measured from the actual panel-width helper.
	wide := helpPanelWidth(stubRenderer{}, helpBodyScaleBase) + 2*stubRenderer{}.LineHeight(1)
	if got := helpBodyScaleFor(stubRenderer{}, wide); got != helpBodyScaleBase {
		t.Errorf("wide screen scale = %d, want %d", got, helpBodyScaleBase)
	}
	if got := helpBodyScaleFor(stubRenderer{}, wide-1); got != 1 {
		t.Errorf("just-too-narrow screen scale = %d, want 1", got)
	}
}

func TestHelpNarrowScreenUsesScale1(t *testing.T) {
	h := newHelpModal()
	// A screen too narrow for the x2 panel must fall back to scale 1.
	narrow := helpPanelWidth(stubRenderer{}, helpBodyScaleBase) // exactly the x2 width, no margin
	h.layout(stubRenderer{}, narrow, 900)
	if h.effScale != 1 {
		t.Errorf("narrow screen effScale = %d, want 1", h.effScale)
	}
}

func TestHelpPanelTargets70Cols(t *testing.T) {
	h := newHelpModal()
	// Roomy screen so the panel is not screen-capped.
	h.layout(stubRenderer{}, 3000, 1400)
	// stub TextWidth(s, scale) = len(s)*8*scale. 70 cols at the effective scale:
	want70 := 70 * 8 * h.effScale
	// Body width equals the 70-col text width (panel minus padding and gutter).
	if h.body.W < want70 {
		t.Errorf("body width %d should fit 70 cols (%d) at scale %d", h.body.W, want70, h.effScale)
	}
	if h.body.W > want70+64 {
		t.Errorf("body width %d far exceeds 70 cols (%d)", h.body.W, want70)
	}
}

func TestHelpTextFitsTargetColumns(t *testing.T) {
	h := newHelpModal()
	for i, ln := range h.lines {
		_, text := helpLineKind(ln) // measure the visible text (markers stripped)
		if len(text) > helpTargetCols {
			t.Errorf("help line %d is %d chars (>%d): %q", i+1, len(text), helpTargetCols, text)
		}
	}
}

func TestHelpMarkdownHeadings(t *testing.T) {
	cases := []struct {
		in   string
		kind helpKind
		text string
	}{
		{"# ZENIMATE", helpH1, "ZENIMATE"},
		{"## Painting", helpH2, "Painting"},
		{"  Ctrl+S    save", helpIndented, "  Ctrl+S    save"},
		{"A plain paragraph.", helpBody, "A plain paragraph."},
	}
	for _, c := range cases {
		k, txt := helpLineKind(c.in)
		if k != c.kind || txt != c.text {
			t.Errorf("helpLineKind(%q) = (%d, %q), want (%d, %q)", c.in, k, txt, c.kind, c.text)
		}
	}
	// The real help must contain at least one H1 and several H2 headings.
	h := newHelpModal()
	var h1, h2 int
	for _, ln := range h.lines {
		switch k, _ := helpLineKind(ln); k {
		case helpH1:
			h1++
		case helpH2:
			h2++
		}
	}
	if h1 < 1 {
		t.Error("help text should have at least one H1 heading")
	}
	if h2 < 3 {
		t.Errorf("help text should have several H2 headings, got %d", h2)
	}
}
