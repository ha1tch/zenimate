//go:build purego

package main

import (
	"testing"

	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
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
		if l.cell < 2 {
			t.Errorf("%v: cell=%d too small", cse, l.cell)
		}
		gridBottom := l.gridY + l.gridH
		topBtnY := l.buttons[0].y
		if gridBottom > topBtnY {
			t.Errorf("%v: grid box bottom %d overlaps button row at %d", cse, gridBottom, topBtnY)
		}
		gridRight := l.gridX + l.gridW
		if gridRight > l.previewX {
			t.Errorf("%v: grid box right %d overlaps preview at %d", cse, gridRight, l.previewX)
		}
	}
}

// At the maximum sprite size the fixed preview box must not starve the viewport:
// gridW should still leave substantial room for editing.
func TestViewportNotStarvedAtMaxSize(t *testing.T) {
	c := ui.New(256, 192) // 32x24 cells
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	if l.gridW < 300 {
		t.Errorf("viewport width %d too small at max sprite size (preview starving it?)", l.gridW)
	}
	// The preview width stays fixed regardless of sprite size; its height grows
	// downward to meet the palette, so it is at least the base box height.
	if l.previewW != previewBox {
		t.Errorf("preview width = %d, want fixed %d", l.previewW, previewBox)
	}
	if l.previewH < previewBox {
		t.Errorf("preview height %d should be at least the base box %d", l.previewH, previewBox)
	}
	// Compare against a small sprite: preview width is identical (fixed).
	c2 := ui.New(16, 16)
	l2 := computeLayout(980, 680, c2, &fileOps{}, 0, 1)
	if l2.previewW != l.previewW {
		t.Errorf("preview width should be fixed across sizes: %d vs %d", l2.previewW, l.previewW)
	}
}

// Collapsing the title reclaims its horizontal space: the toolbars start further
// left. The header stays compact (grid does not move down).
func TestCollapseTitleReclaimsWidth(t *testing.T) {
	c := ui.New(64, 64)
	exp := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	col := computeLayout(980, 680, c, &fileOps{}, 1, 1)
	if col.frameStripX >= exp.frameStripX {
		t.Errorf("collapsed frame strip X %d should be left of expanded %d", col.frameStripX, exp.frameStripX)
	}
	if col.gridY > exp.gridY {
		t.Errorf("collapse should not push the grid down: %d vs %d", col.gridY, exp.gridY)
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
	if col.frameRects[0].Width <= exp.frameRects[0].Width {
		t.Errorf("collapsed frame buttons should be wider when constrained: %v vs %v",
			col.frameRects[0].Width, exp.frameRects[0].Width)
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
	if l.gridY > 122 {
		t.Errorf("grid starts at %d; header taller than expected", l.gridY)
	}
}

func TestPaletteBottomAlignedFourColumns(t *testing.T) {
	c := ui.New(64, 64)
	c.SetMode(ui.SpectrumColour)
	w, h := 980, 680
	l := computeLayout(w, h, c, &fileOps{}, 0, 1)

	// Bottom edge of the palette aligns with the bottom button strip margin:
	// the lowest swatch bottom should sit at h-pad.
	var maxBottom float32
	for _, r := range l.swatchRects {
		if b := r.Y + r.Height; b > maxBottom {
			maxBottom = b
		}
	}
	if int(maxBottom) != h-pad {
		t.Errorf("palette bottom = %v, want %d (window margin)", maxBottom, h-pad)
	}

	// Four distinct columns (distinct X values).
	xs := map[int]bool{}
	for _, r := range l.swatchRects {
		xs[int(r.X)] = true
	}
	if len(xs) != 4 {
		t.Errorf("palette has %d columns, want 4", len(xs))
	}

	// Base-colour order: swatch 0 (top-left pair, normal) is BLUE, swatch 2 is
	// BLACK, then RED, MAGENTA, GREEN, CYAN, YELLOW, WHITE.
	wantBase := []int{
		zxpalette.Blue, zxpalette.Black, zxpalette.Red, zxpalette.Magenta,
		zxpalette.Green, zxpalette.Cyan, zxpalette.Yellow, zxpalette.White,
	}
	for pair := 0; pair < 8; pair++ {
		if l.swatchBase[pair*2] != wantBase[pair] {
			t.Errorf("pair %d base = %d, want %d", pair, l.swatchBase[pair*2], wantBase[pair])
		}
		if l.swatchBright[pair*2] || !l.swatchBright[pair*2+1] {
			t.Errorf("pair %d bright flags wrong: %v,%v", pair, l.swatchBright[pair*2], l.swatchBright[pair*2+1])
		}
	}

	// The preview grows down to just above the palette.
	if l.previewY+l.previewH >= l.paletteY {
		t.Errorf("preview (bottom %d) should sit above palette top %d", l.previewY+l.previewH, l.paletteY)
	}
}

func TestGridPreviewGapEqualsMargin(t *testing.T) {
	c := ui.New(64, 64)
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	gap := l.previewX - (l.gridX + l.gridW)
	if gap != pad {
		t.Errorf("grid-to-preview gap = %d, want pad=%d", gap, pad)
	}
	// And the preview's right edge keeps a pad margin from the window border.
	if rightMargin := l.winW - (l.previewX + l.previewW); rightMargin != pad {
		t.Errorf("preview right margin = %d, want pad=%d", rightMargin, pad)
	}
}

func TestDrawerGivesViewportMoreRoomWhenClosed(t *testing.T) {
	c := ui.New(64, 64)
	open := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	closed := computeLayout(980, 680, c, &fileOps{}, 0, 0)
	if closed.gridH <= open.gridH {
		t.Errorf("closed drawer should grow the viewport: closed gridH %d vs open %d",
			closed.gridH, open.gridH)
	}
	// The triangle toggle sits inside the viewport's bottom-right corner.
	tr := closed.drawerToggle
	if int(tr.Y+tr.Height) > closed.gridY+closed.gridH {
		t.Errorf("triangle should sit inside the viewport bottom, not below it")
	}
	if int(tr.X+tr.Width) > closed.gridX+closed.gridW {
		t.Errorf("triangle should sit inside the viewport right edge")
	}
	if open.drawerToggle.Width <= 0 || open.drawerToggle.Height <= 0 {
		t.Errorf("drawer toggle rect should be non-empty")
	}
}

func TestOnionAlignsToF6(t *testing.T) {
	c := ui.New(64, 64) // 8 frames, so F6 exists
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	f6x := l.frameRects[5].X // F6 left edge
	if l.onionButtons[0].x != int(f6x) {
		t.Errorf("Onion Prev x = %d, want F6 left edge %v", l.onionButtons[0].x, f6x)
	}
}

func TestTitleCollapseAnimatesToolbars(t *testing.T) {
	c := ui.New(64, 64)
	exp := computeLayout(980, 680, c, &fileOps{}, 0, 1)   // fully expanded
	mid := computeLayout(980, 680, c, &fileOps{}, 0.5, 1) // halfway
	col := computeLayout(980, 680, c, &fileOps{}, 1, 1)   // fully collapsed

	// Title width eases between expanded and collapsed.
	if !(col.titleRect.Width < mid.titleRect.Width && mid.titleRect.Width < exp.titleRect.Width) {
		t.Errorf("title width should interpolate: exp=%v mid=%v col=%v",
			exp.titleRect.Width, mid.titleRect.Width, col.titleRect.Width)
	}
	// The toolbars (frame strip x) slide left as the title shrinks.
	if !(col.frameStripX < mid.frameStripX && mid.frameStripX < exp.frameStripX) {
		t.Errorf("frame strip should slide: exp=%d mid=%d col=%d",
			exp.frameStripX, mid.frameStripX, col.frameStripX)
	}
}

func TestDrawerToggleHitCoversTriangle(t *testing.T) {
	c := ui.New(64, 64)
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	tri := l.drawerToggle
	hit := l.drawerToggleHit
	// The hit area fully contains the visible triangle.
	if hit.X > tri.X || hit.Y > tri.Y ||
		hit.X+hit.Width < tri.X+tri.Width || hit.Y+hit.Height < tri.Y+tri.Height {
		t.Errorf("hit area %v does not contain triangle %v", hit, tri)
	}
	// The hit area reaches the viewport's bottom-right corner.
	if int(hit.X+hit.Width) != l.gridX+l.gridW || int(hit.Y+hit.Height) != l.gridY+l.gridH {
		t.Errorf("hit area should extend to the viewport corner (%d,%d), got (%v,%v)",
			l.gridX+l.gridW, l.gridY+l.gridH, hit.X+hit.Width, hit.Y+hit.Height)
	}
	// A click at the very corner pixel lands in the hit area.
	cornerX := l.gridX + l.gridW - 1
	cornerY := l.gridY + l.gridH - 1
	if !rectHit(hit, cornerX, cornerY) {
		t.Errorf("corner pixel (%d,%d) should hit the drawer toggle", cornerX, cornerY)
	}
}

func TestDrawerHasFileButtons(t *testing.T) {
	c := ui.New(16, 16)
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	have := map[string]bool{}
	for _, b := range l.buttons {
		have[b.label] = true
	}
	for _, want := range []string{"Open", "Save", "RESET", "CLS", "32x24", "Copy", "Paste"} {
		if !have[want] {
			t.Errorf("drawer missing %q button", want)
		}
	}
}
