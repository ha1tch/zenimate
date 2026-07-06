//go:build purego

package main

import (
	"testing"

	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guidraw"
	"github.com/ha1tch/zenimate/internal/ui"
)

// computeLayout must keep the editor grid BOX from overlapping the button block
// or the preview column at any window size. The sprite floats inside this box via
// pan/zoom and is clipped to it (Quag model), so the invariant is on the box
// rectangle, not on sprite-width x cell — the base cell is fixed and the sprite is
// fitted by zoom, so sprite x baseCell may legitimately exceed the box.
func TestLayoutGridFits(t *testing.T) {
	cases := []struct{ w, h, sw, sh int }{
		{980, 680, 16, 16},
		{980, 680, 32, 32},
		{minWinW, minWinH, 32, 32},
		{1400, 1000, 8, 8},
		{minWinW, minWinH, 8, 32},
	}
	for _, cse := range cases {
		c := ui.New(cse.sw, cse.sh)
		l := computeLayout(cse.w, cse.h, c, &fileOps{}, 0, 1)
		if l.Cell < 2 {
			t.Errorf("%v: cell=%d too small", cse, l.Cell)
		}
		gridBottom := l.GridY + l.GridH
		topBtnY := l.Buttons[0].Y
		if gridBottom > topBtnY {
			t.Errorf("%v: grid box bottom %d overlaps button row at %d", cse, gridBottom, topBtnY)
		}
		gridRight := l.GridX + l.GridW
		if gridRight > l.PreviewX {
			t.Errorf("%v: grid box right %d overlaps preview at %d", cse, gridRight, l.PreviewX)
		}
	}
}

// At the maximum sprite size the fixed preview box must not starve the viewport:
// gridW should still leave substantial room for editing.
func TestViewportNotStarvedAtMaxSize(t *testing.T) {
	c := ui.New(256, 192) // 32x24 cells
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	if l.GridW < 300 {
		t.Errorf("viewport width %d too small at max sprite size (preview starving it?)", l.GridW)
	}
	// The preview width stays fixed regardless of sprite size; its height grows
	// downward to meet the palette, so it is at least the base box height.
	if l.PreviewW != previewBox {
		t.Errorf("preview width = %d, want fixed %d", l.PreviewW, previewBox)
	}
	if l.PreviewH < previewBox {
		t.Errorf("preview height %d should be at least the base box %d", l.PreviewH, previewBox)
	}
	// Compare against a small sprite: preview width is identical (fixed).
	c2 := ui.New(16, 16)
	l2 := computeLayout(980, 680, c2, &fileOps{}, 0, 1)
	if l2.PreviewW != l.PreviewW {
		t.Errorf("preview width should be fixed across sizes: %d vs %d", l2.PreviewW, l.PreviewW)
	}
}

// Collapsing the title reclaims its horizontal space: the toolbars start further
// left. The header stays compact (grid does not move down).
func TestCollapseTitleReclaimsWidth(t *testing.T) {
	c := ui.New(64, 64)
	exp := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	col := computeLayout(980, 680, c, &fileOps{}, 1, 1)
	if col.FrameStripX >= exp.FrameStripX {
		t.Errorf("collapsed frame strip X %d should be left of expanded %d", col.FrameStripX, exp.FrameStripX)
	}
	if col.GridY > exp.GridY {
		t.Errorf("collapse should not push the grid down: %d vs %d", col.GridY, exp.GridY)
	}
}

// When the frame count is high the strip is width-constrained, and collapsing
// the title (freeing horizontal space) lets each frame button grow wider.
func TestCollapseWidensConstrainedStrip(t *testing.T) {
	c := ui.New(64, 64)
	for c.Sprite.FrameCount() < 16 {
		c.AddFrame()
	}
	narrow := 760 // a window where 16 frames must shrink to fit
	exp := computeLayout(narrow, 680, c, &fileOps{}, 0, 1)
	col := computeLayout(narrow, 680, c, &fileOps{}, 1, 1)
	if col.FrameRects[0].Width <= exp.FrameRects[0].Width {
		t.Errorf("collapsed frame buttons should be wider when constrained: %v vs %v",
			col.FrameRects[0].Width, exp.FrameRects[0].Width)
	}
}

// The toolbars-beside-title layout keeps the header compact: the grid starts
// well within the top of an 680px window (proof the viewport gained vertical
// room versus a stacked header).
func TestHeaderIsCompact(t *testing.T) {
	c := ui.New(64, 64)
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	// The header now includes a thin frame-scrubber row above the frame buttons,
	// so the guard allows a little more than the pre-scrubber 110px.
	if l.GridY > 122 {
		t.Errorf("grid starts at %d; header taller than expected", l.GridY)
	}
}

func TestPaletteAnchorBottomAligned(t *testing.T) {
	c := ui.New(64, 64)
	c.SetMode(ui.SpectrumColour)
	w, h := 980, 680
	l := computeLayout(w, h, c, &fileOps{}, 0, 1)

	// computeLayout only computes the palette's anchor position now — the
	// 4x4 swatch grid itself (size, classic colour-key order, bright flags)
	// is owned and tested by zenui.ZXClassicPaletteChooser directly (see
	// pkg/zenui/zxclassicpalette_test.go's TestZXClassicPaletteLayout).
	// This test verifies only what computeLayout is still responsible for:
	// the anchor bottom-aligns with the same margin as the button strip.
	const paletteSwatchH, paletteGapY, paletteRows = 24, 5, 4
	paletteH := paletteRows*paletteSwatchH + (paletteRows-1)*paletteGapY
	if got := l.PaletteY + paletteH; got != h-pad {
		t.Errorf("palette bottom = %d, want %d (window margin)", got, h-pad)
	}

	// The preview grows down to just above the palette.
	if l.PreviewY+l.PreviewH >= l.PaletteY {
		t.Errorf("preview (bottom %d) should sit above palette top %d", l.PreviewY+l.PreviewH, l.PaletteY)
	}
}

func TestGridPreviewGapEqualsMargin(t *testing.T) {
	c := ui.New(64, 64)
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	gap := l.PreviewX - (l.GridX + l.GridW)
	if gap != pad {
		t.Errorf("grid-to-preview gap = %d, want pad=%d", gap, pad)
	}
	// And the preview's right edge keeps a pad margin from the window border.
	if rightMargin := l.WinW - (l.PreviewX + l.PreviewW); rightMargin != pad {
		t.Errorf("preview right margin = %d, want pad=%d", rightMargin, pad)
	}
}

func TestDrawerGivesViewportMoreRoomWhenClosed(t *testing.T) {
	c := ui.New(64, 64)
	open := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	closed := computeLayout(980, 680, c, &fileOps{}, 0, 0)
	if closed.GridH <= open.GridH {
		t.Errorf("closed drawer should grow the viewport: closed gridH %d vs open %d",
			closed.GridH, open.GridH)
	}
	// The triangle toggle sits inside the viewport's bottom-right corner.
	tr := closed.DrawerToggle
	if int(tr.Y+tr.Height) > closed.GridY+closed.GridH {
		t.Errorf("triangle should sit inside the viewport bottom, not below it")
	}
	if int(tr.X+tr.Width) > closed.GridX+closed.GridW {
		t.Errorf("triangle should sit inside the viewport right edge")
	}
	if open.DrawerToggle.Width <= 0 || open.DrawerToggle.Height <= 0 {
		t.Errorf("drawer toggle rect should be non-empty")
	}
}

func TestOnionAlignsToF6(t *testing.T) {
	c := ui.New(64, 64) // 8 frames, so F6 exists
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	f6x := l.FrameRects[5].X // F6 left edge
	if l.OnionButtons[0].X != int(f6x) {
		t.Errorf("Onion Prev x = %d, want F6 left edge %v", l.OnionButtons[0].X, f6x)
	}
}

func TestTitleCollapseAnimatesToolbars(t *testing.T) {
	c := ui.New(64, 64)
	exp := computeLayout(980, 680, c, &fileOps{}, 0, 1)   // fully expanded
	mid := computeLayout(980, 680, c, &fileOps{}, 0.5, 1) // halfway
	col := computeLayout(980, 680, c, &fileOps{}, 1, 1)   // fully collapsed

	// Title width eases between expanded and collapsed.
	if !(col.TitleRect.Width < mid.TitleRect.Width && mid.TitleRect.Width < exp.TitleRect.Width) {
		t.Errorf("title width should interpolate: exp=%v mid=%v col=%v",
			exp.TitleRect.Width, mid.TitleRect.Width, col.TitleRect.Width)
	}
	// The toolbars (frame strip x) slide left as the title shrinks.
	if !(col.FrameStripX < mid.FrameStripX && mid.FrameStripX < exp.FrameStripX) {
		t.Errorf("frame strip should slide: exp=%d mid=%d col=%d",
			exp.FrameStripX, mid.FrameStripX, col.FrameStripX)
	}
}

func TestDrawerToggleHitCoversTriangle(t *testing.T) {
	c := ui.New(64, 64)
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	tri := l.DrawerToggle
	hit := l.DrawerToggleHit
	// The hit area fully contains the visible triangle.
	if hit.X > tri.X || hit.Y > tri.Y ||
		hit.X+hit.Width < tri.X+tri.Width || hit.Y+hit.Height < tri.Y+tri.Height {
		t.Errorf("hit area %v does not contain triangle %v", hit, tri)
	}
	// The hit area reaches the viewport's bottom-right corner.
	if int(hit.X+hit.Width) != l.GridX+l.GridW || int(hit.Y+hit.Height) != l.GridY+l.GridH {
		t.Errorf("hit area should extend to the viewport corner (%d,%d), got (%v,%v)",
			l.GridX+l.GridW, l.GridY+l.GridH, hit.X+hit.Width, hit.Y+hit.Height)
	}
	// A click at the very corner pixel lands in the hit area.
	cornerX := l.GridX + l.GridW - 1
	cornerY := l.GridY + l.GridH - 1
	if !guidraw.RectHit(hit, cornerX, cornerY) {
		t.Errorf("corner pixel (%d,%d) should hit the drawer toggle", cornerX, cornerY)
	}
}

func TestDrawerHasFileButtons(t *testing.T) {
	c := ui.New(16, 16)
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	have := map[string]bool{}
	for _, b := range l.Buttons {
		have[b.Label] = true
	}
	for _, want := range []string{"Open", "Save", "RESET", "CLS", "32x24", "Copy", "Paste"} {
		if !have[want] {
			t.Errorf("drawer missing %q button", want)
		}
	}
}
