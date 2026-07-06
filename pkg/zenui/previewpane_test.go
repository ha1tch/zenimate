package zenui

import (
	"sort"
	"testing"
)

// fakeImageSource is a minimal ImageSource for tests: fixed dimensions, a
// single flat colour for every pixel.
type fakeImageSource struct {
	w, h   int
	colour Colour
}

func (f fakeImageSource) Width() int  { return f.w }
func (f fakeImageSource) Height() int { return f.h }
func (f fakeImageSource) Region(x0, y0, w, h int) []Colour {
	region := make([]Colour, w*h)
	for i := range region {
		region[i] = f.colour
	}
	return region
}

func TestPreviewRegionSmallImageFitsEntirely(t *testing.T) {
	// 8x8 image, zoom=4 -> box holds 40/4=10 px per axis, clamped to the 8px
	// image; the drawn area (8*4=32) is smaller than the box (40), so it's
	// centred with a (40-32)/2=4px offset on each axis.
	cx, cy, spanX, spanY, ox, oy, dw, dh := previewRegion(8, 8, 40, 40, 4, 4, 4, 100, 50)
	if cx != 0 || cy != 0 || spanX != 8 || spanY != 8 {
		t.Errorf("region = (%d,%d)+(%d,%d), want (0,0)+(8,8)", cx, cy, spanX, spanY)
	}
	if ox != 104 || oy != 54 || dw != 32 || dh != 32 {
		t.Errorf("draw = (%v,%v) %vx%v, want (104,54) 32x32", ox, oy, dw, dh)
	}
}

func TestPreviewRegionClampsNearEdge(t *testing.T) {
	// 16x16 image, zoom=2 -> box holds 20/2=10 px per axis. Focus at (1,1),
	// near the top-left corner: centring would want cx0=1-5=-4, clamped to 0.
	cx, cy, spanX, spanY, _, _, _, _ := previewRegion(16, 16, 20, 20, 2, 1, 1, 0, 0)
	if cx != 0 || cy != 0 {
		t.Errorf("cx,cy = %d,%d, want 0,0 (clamped)", cx, cy)
	}
	if spanX != 10 || spanY != 10 {
		t.Errorf("span = %d,%d, want 10,10", spanX, spanY)
	}
}

func TestPreviewRegionClampsAtFarEdge(t *testing.T) {
	// Same 16x16 image, zoom=2 setup as the near-edge case: box holds
	// 20/2=10 px per axis. Focus at (15,15), near the bottom-right corner:
	// centring wants cx0=15-5=10, cy0=10; cx0+spanX=20 > iw(16), so clamped
	// to cx0=iw-spanX=6 (same for cy0).
	cx, cy, spanX, spanY, _, _, _, _ := previewRegion(16, 16, 20, 20, 2, 15, 15, 0, 0)
	if cx != 6 || cy != 6 {
		t.Errorf("cx,cy = %d,%d, want 6,6 (clamped)", cx, cy)
	}
	if spanX != 10 || spanY != 10 {
		t.Errorf("span = %d,%d, want 10,10", spanX, spanY)
	}
}

func TestPreviewPaneSetZoomClamps(t *testing.T) {
	p := NewPreviewPane(PreviewPaneConfig{
		Bounds: Rect{X: 0, Y: 0, W: 40, H: 40}, Source: fakeImageSource{w: 16, h: 16}, MinZoom: 1, MaxZoom: 4,
	})
	p.SetZoom(2)
	if p.Zoom() != 2 {
		t.Errorf("SetZoom(2): Zoom() = %d, want 2", p.Zoom())
	}
	p.SetZoom(99)
	if p.Zoom() != 4 {
		t.Errorf("SetZoom(99): Zoom() = %d, want 4 (clamped to MaxZoom)", p.Zoom())
	}
	p.SetZoom(-5)
	if p.Zoom() != 1 {
		t.Errorf("SetZoom(-5): Zoom() = %d, want 1 (clamped to MinZoom)", p.Zoom())
	}
}

func TestPreviewPaneZoomCycles(t *testing.T) {
	p := NewPreviewPane(PreviewPaneConfig{
		Bounds: Rect{X: 0, Y: 0, W: 40, H: 40}, Source: fakeImageSource{w: 16, h: 16}, MinZoom: 1, MaxZoom: 4,
	})
	if p.Zoom() != 1 {
		t.Fatalf("initial zoom = %d, want 1 (MinZoom)", p.Zoom())
	}
	for want := 2; want <= 4; want++ {
		p.Update(Input{MouseX: 10, MouseY: 10, MouseRightPressed: true}, 0)
		if p.Zoom() != want {
			t.Errorf("zoom = %d, want %d", p.Zoom(), want)
		}
	}
	// One more right-click wraps back to MinZoom.
	p.Update(Input{MouseX: 10, MouseY: 10, MouseRightPressed: true}, 0)
	if p.Zoom() != 1 {
		t.Errorf("zoom after wrap = %d, want 1", p.Zoom())
	}
}

func TestPreviewPaneZoomIgnoresClickOutsideBounds(t *testing.T) {
	p := NewPreviewPane(PreviewPaneConfig{
		Bounds: Rect{X: 0, Y: 0, W: 40, H: 40}, Source: fakeImageSource{w: 16, h: 16}, MinZoom: 1, MaxZoom: 4,
	})
	p.Update(Input{MouseX: 999, MouseY: 999, MouseRightPressed: true}, 0)
	if p.Zoom() != 1 {
		t.Errorf("right-click outside bounds should not cycle zoom, got %d", p.Zoom())
	}
}

func TestPreviewPanePopupEasesTowardHeld(t *testing.T) {
	p := NewPreviewPane(PreviewPaneConfig{
		Bounds: Rect{X: 0, Y: 0, W: 40, H: 40}, Source: fakeImageSource{w: 16, h: 16}, MinZoom: 1, MaxZoom: 4,
	})
	// Press and hold inside the bounds.
	p.Update(Input{MouseX: 10, MouseY: 10, MousePressed: true, MouseDown: true}, 0.1)
	if p.Popup() <= 0 {
		t.Fatal("popup should have started easing toward 1 after one held frame")
	}
	// Keep holding for enough frames to approach 1.
	for i := 0; i < 50; i++ {
		p.Update(Input{MouseX: 10, MouseY: 10, MouseDown: true}, 0.1)
	}
	if p.Popup() < 0.99 {
		t.Errorf("popup = %v after sustained hold, want close to 1", p.Popup())
	}
	// Release: popup should ease back down.
	for i := 0; i < 50; i++ {
		p.Update(Input{MouseX: 10, MouseY: 10, MouseDown: false}, 0.1)
	}
	if p.Popup() != 0 {
		t.Errorf("popup = %v after release and decay, want 0", p.Popup())
	}
}

func TestPreviewPaneDrawSkipsPopupAtZero(t *testing.T) {
	p := NewPreviewPane(PreviewPaneConfig{
		Bounds:  Rect{X: 0, Y: 0, W: 40, H: 40},
		Source:  fakeImageSource{w: 4, h: 4, colour: Colour{R: 1, G: 2, B: 3, A: 255}},
		MinZoom: 1, MaxZoom: 4,
	})
	calls := 0
	p.Draw(countingRenderer{fillCalls: &calls}, 800, 600, DefaultTheme())
	baseline := calls

	// Force the popup open, then draw again — call count must increase.
	p.held = true
	for i := 0; i < 50; i++ {
		p.Update(Input{MouseDown: true}, 0.1)
	}
	p.Draw(countingRenderer{fillCalls: &calls}, 800, 600, DefaultTheme())
	if calls <= baseline {
		t.Errorf("expected more FillRect calls with the popup open, got %d (baseline %d)", calls, baseline)
	}
}

// gridImageSource returns a distinct colour per cell, encoding (row,col) in
// R/G, so a test can verify a drawn colour landed at the position it should
// have — not just that some colour was drawn somewhere.
type gridImageSource struct{ w, h int }

func (g gridImageSource) Width() int  { return g.w }
func (g gridImageSource) Height() int { return g.h }
func (g gridImageSource) Region(x0, y0, w, h int) []Colour {
	region := make([]Colour, w*h)
	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			region[row*w+col] = Colour{R: uint8(y0 + row), G: uint8(x0 + col), B: 0, A: 255}
		}
	}
	return region
}

// TestPreviewPaneDrawHasNoGapsBetweenAdjacentPixels is the regression test
// for the intermittent gridline bug: drawSampledRegion used to give every
// pixel a fixed width/height (int(pw+0.5)), which leaves a sub-pixel
// rounding gap between adjacent cells whenever the box size doesn't divide
// evenly by the image size — the background shows through the gap as a
// grid. Uses a 10px box over a 3px-wide image (pw = 10/3 = 3.33...,
// deliberately not an integer) and asserts every cell's right/bottom edge
// exactly matches its neighbour's left/top edge.
func TestPreviewPaneDrawHasNoGapsBetweenAdjacentPixels(t *testing.T) {
	p := NewPreviewPane(PreviewPaneConfig{
		Bounds:  Rect{X: 0, Y: 0, W: 12, H: 12},
		Source:  gridImageSource{w: 5, h: 5},
		MinZoom: 1, MaxZoom: 1,
	})

	var calls []struct {
		rect Rect
		col  Colour
	}
	// Called directly with exact, hand-verified parameters (bw=12, span=5
	// gives pw=2.4, which produces a real 1px gap under the old fixed-width
	// approach) rather than through Draw's focus/zoom computation, which
	// would introduce uncertainty about what bw/bh actually reach this
	// function.
	p.drawSampledRegion(recordingRenderer{calls: &calls}, 0, 0, 5, 5, 0, 0, 12, 12)

	// Index the per-pixel cells (skip the background fill) by their
	// top-left corner, then verify every cell's right edge equals the
	// start of the cell to its right, and every cell's bottom edge equals
	// the start of the cell below it — i.e. no gap, no overlap.
	// Group into rows and columns, then check that each cell's right edge
	// exactly equals the next cell's left edge (and same for bottom/top) —
	// comparing consecutive cells by position order, not by looking up an
	// exact expected coordinate, since a real gap means no cell exists at
	// that coordinate at all (a coordinate-lookup check would silently miss
	// exactly the case it's meant to catch).
	rows := make(map[int][]Rect)
	for _, c := range calls {
		rows[c.rect.Y] = append(rows[c.rect.Y], c.rect)
	}
	for y, row := range rows {
		sort.Slice(row, func(i, j int) bool { return row[i].X < row[j].X })
		for i := 0; i+1 < len(row); i++ {
			gotRight := row[i].X + row[i].W
			wantLeft := row[i+1].X
			if gotRight != wantLeft {
				t.Errorf("row y=%d: cell at x=%d has right edge %d, but next cell starts at x=%d (gap=%d)",
					y, row[i].X, gotRight, wantLeft, wantLeft-gotRight)
			}
		}
	}
	cols := make(map[int][]Rect)
	for _, c := range calls {
		cols[c.rect.X] = append(cols[c.rect.X], c.rect)
	}
	for x, col := range cols {
		sort.Slice(col, func(i, j int) bool { return col[i].Y < col[j].Y })
		for i := 0; i+1 < len(col); i++ {
			gotBottom := col[i].Y + col[i].H
			wantTop := col[i+1].Y
			if gotBottom != wantTop {
				t.Errorf("col x=%d: cell at y=%d has bottom edge %d, but next cell starts at y=%d (gap=%d)",
					x, col[i].Y, gotBottom, wantTop, wantTop-gotBottom)
			}
		}
	}
}

type recordingRenderer struct {
	noopRenderer
	calls *[]struct {
		rect Rect
		col  Colour
	}
}

func (r recordingRenderer) FillRect(rect Rect, col Colour) {
	*r.calls = append(*r.calls, struct {
		rect Rect
		col  Colour
	}{rect, col})
}

// TestPreviewPaneDrawMapsRegionRowMajorToCorrectScreenPosition verifies the
// colours[j*spanX+i] indexing in drawSampledRegion: the colour landing at
// each screen cell must match the (row,col) that cell's position implies,
// not merely be *a* colour from the region. This is exactly the kind of bug
// a row/column transposition would produce without being caught by a
// call-count-only test.
func TestPreviewPaneDrawMapsRegionRowMajorToCorrectScreenPosition(t *testing.T) {
	p := NewPreviewPane(PreviewPaneConfig{
		Bounds:  Rect{X: 0, Y: 0, W: 4, H: 4}, // 4x4 box, zoom=1 -> span=4x4, one screen px per image px
		Source:  gridImageSource{w: 4, h: 4},
		MinZoom: 1, MaxZoom: 1,
	})
	// Focus (2,2) with a 4x4 span in a 4x4 image clamps to the whole image,
	// so cx0=cy0=0 — the drawn region is exactly the source's (0,0)-(4,4).
	p.SetFocus(2, 2)

	var calls []struct {
		rect Rect
		col  Colour
	}
	p.Draw(recordingRenderer{calls: &calls}, 800, 600, DefaultTheme())

	if len(calls) != 17 { // 1 background fill + 16 per-pixel cells
		t.Fatalf("got %d FillRect calls, want 17 (1 background + 4x4)", len(calls))
	}
	for _, c := range calls[1:] { // skip the background fill (covers the whole box, not a cell)
		wantR := uint8(c.rect.Y) // screen y == image row at zoom 1, box origin (0,0)
		wantG := uint8(c.rect.X) // screen x == image col
		if c.col.R != wantR || c.col.G != wantG {
			t.Errorf("rect %+v got colour R=%d G=%d, want R=%d G=%d (row/col transposed?)",
				c.rect, c.col.R, c.col.G, wantR, wantG)
		}
	}
}
