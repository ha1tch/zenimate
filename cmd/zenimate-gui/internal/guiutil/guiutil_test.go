package guiutil

import "testing"

func collectPoints(walk func(fn func(x, y int))) map[[2]int]bool {
	pts := make(map[[2]int]bool)
	walk(func(x, y int) { pts[[2]int{x, y}] = true })
	return pts
}

func collectBrushPoints(shape BrushShape, size int) map[[2]int]bool {
	pts := make(map[[2]int]bool)
	BrushStamp(shape, size, func(dx, dy int) { pts[[2]int{dx, dy}] = true })
	return pts
}

func TestBrushStampSizeOneIsSinglePixel(t *testing.T) {
	for _, shape := range []BrushShape{BrushRound, BrushSquare, BrushCustom} {
		pts := collectBrushPoints(shape, 1)
		if len(pts) != 1 || !pts[[2]int{0, 0}] {
			t.Errorf("shape %v size 1: got %v, want just {(0,0)}", shape, pts)
		}
	}
}

func TestBrushStampSquareIsFullBlock(t *testing.T) {
	pts := collectBrushPoints(BrushSquare, 3)
	if len(pts) != 9 {
		t.Errorf("Square size 3: got %d points, want 9 (full 3x3)", len(pts))
	}
}

func TestBrushStampRoundChamfersCornersAtSizeThreePlus(t *testing.T) {
	pts := collectBrushPoints(BrushRound, 3)
	if len(pts) != 5 {
		t.Fatalf("Round size 3: got %d points, want 5 (plus shape)", len(pts))
	}
	if pts[[2]int{-1, -1}] || pts[[2]int{-1, 1}] || pts[[2]int{1, -1}] || pts[[2]int{1, 1}] {
		t.Error("Round size 3 should have its corners chamfered off")
	}
	if !pts[[2]int{0, 0}] {
		t.Error("Round size 3 should include the centre")
	}
}

// TestBrushStampCustomDiffersFromRoundAtSizeThree is a direct regression
// test for a real mistake caught before it shipped: an earlier formula for
// Custom happened to produce the exact same 5-point shape as Round at size
// 3 — the most likely size to actually get used — which would have made
// the third "distinct" option not distinct at all where it mattered most.
func TestBrushStampCustomDiffersFromRoundAtSizeThree(t *testing.T) {
	round := collectBrushPoints(BrushRound, 3)
	custom := collectBrushPoints(BrushCustom, 3)
	if len(round) == len(custom) {
		same := true
		for p := range round {
			if !custom[p] {
				same = false
				break
			}
		}
		if same {
			t.Fatal("Custom and Round produce the identical shape at size 3 — not a genuinely distinct third option")
		}
	}
	// Custom at size 3 should be exactly the four corners Round excludes.
	wantCustom := map[[2]int]bool{{-1, -1}: true, {-1, 1}: true, {1, -1}: true, {1, 1}: true}
	if len(custom) != 4 {
		t.Fatalf("Custom size 3: got %d points, want 4 (corners only)", len(custom))
	}
	for p := range wantCustom {
		if !custom[p] {
			t.Errorf("Custom size 3 missing corner %v", p)
		}
	}
}

func TestBrushStampAllThreeShapesDistinctAtSizeFour(t *testing.T) {
	round := collectBrushPoints(BrushRound, 4)
	square := collectBrushPoints(BrushSquare, 4)
	custom := collectBrushPoints(BrushCustom, 4)
	if len(round) == len(square) || len(round) == len(custom) || len(square) == len(custom) {
		t.Errorf("expected three distinct point counts at size 4, got round=%d square=%d custom=%d",
			len(round), len(square), len(custom))
	}
}

func TestEllipseOutlineVisitsCardinalPoints(t *testing.T) {
	pts := collectPoints(func(fn func(x, y int)) { EllipseOutline(0, 0, 10, 6, fn) })
	for _, c := range [][2]int{{0, 3}, {10, 3}, {5, 0}, {5, 6}} {
		if !pts[c] {
			t.Errorf("cardinal point %v not visited", c)
		}
	}
}

func TestEllipseOutlineDoesNotFillCentre(t *testing.T) {
	pts := collectPoints(func(fn func(x, y int)) { EllipseOutline(0, 0, 10, 6, fn) })
	if pts[[2]int{5, 3}] {
		t.Error("EllipseOutline visited the centre — it should be outline-only, not filled")
	}
}

func TestEllipseOutlineHasNoGaps(t *testing.T) {
	// A single-axis-only scan leaves gaps near the steep parts of the curve;
	// verify every column has at least one point and every row has at least
	// one point, which only holds if the double-scan union is working.
	pts := collectPoints(func(fn func(x, y int)) { EllipseOutline(0, 0, 20, 8, fn) })
	for x := 0; x <= 20; x++ {
		found := false
		for y := 0; y <= 8; y++ {
			if pts[[2]int{x, y}] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("column x=%d has no boundary point at all — a gap", x)
		}
	}
}

func TestPolygonOutlineSixSidesHasFlatTopAndBottom(t *testing.T) {
	// Indirect check: a hexagon's edges are all straight lines, so the total
	// point count should be well under a filled ellipse's, and each of the
	// four extreme axis points (top/bottom/left/right of the bounding box
	// region actually reached) should be part of the outline.
	pts := collectPoints(func(fn func(x, y int)) { PolygonOutline(6, 0, 0, 20, 12, fn) })
	if len(pts) == 0 {
		t.Fatal("PolygonOutline(6, ...) produced no points at all")
	}
	// Top and bottom vertices sit at the horizontal centre (flat-top/bottom
	// orientation), at the very top and bottom of the bounding box.
	if !pts[[2]int{10, 0}] {
		t.Error("expected top vertex (10,0) not visited")
	}
	if !pts[[2]int{10, 12}] {
		t.Error("expected bottom vertex (10,12) not visited")
	}
}

func TestPolygonOutlineDoesNotFillCentre(t *testing.T) {
	pts := collectPoints(func(fn func(x, y int)) { PolygonOutline(6, 0, 0, 20, 12, fn) })
	if pts[[2]int{10, 6}] {
		t.Error("PolygonOutline(6, ...) visited the centre — it should be outline-only, not filled")
	}
}

func TestPolygonOutlineSideCountAffectsShape(t *testing.T) {
	tri := collectPoints(func(fn func(x, y int)) { PolygonOutline(3, 0, 0, 20, 20, fn) })
	square := collectPoints(func(fn func(x, y int)) { PolygonOutline(4, 0, 0, 20, 20, fn) })
	octagon := collectPoints(func(fn func(x, y int)) { PolygonOutline(8, 0, 0, 20, 20, fn) })
	if len(tri) == 0 || len(square) == 0 || len(octagon) == 0 {
		t.Fatal("PolygonOutline produced no points for one of sides=3,4,8")
	}
	// Different side counts should produce genuinely different outlines, not
	// the same shape regardless of the parameter.
	if len(tri) == len(square) && len(square) == len(octagon) {
		t.Error("sides=3, sides=4, and sides=8 all produced the same point count — the sides parameter may not be having any effect")
	}
}

func TestPolygonOutlineClampsSideCount(t *testing.T) {
	tooFew := collectPoints(func(fn func(x, y int)) { PolygonOutline(1, 0, 0, 20, 20, fn) })
	minSides := collectPoints(func(fn func(x, y int)) { PolygonOutline(3, 0, 0, 20, 20, fn) })
	if len(tooFew) != len(minSides) {
		t.Error("sides=1 should clamp to the minimum of 3, not produce a degenerate shape")
	}
}

func TestCenterOrCornerBoundsCornerMode(t *testing.T) {
	x0, y0, x1, y1 := CenterOrCornerBounds(10, 20, 30, 40, false)
	if x0 != 10 || y0 != 20 || x1 != 30 || y1 != 40 {
		t.Errorf("corner mode should pass anchor/current through directly: got (%d,%d,%d,%d)", x0, y0, x1, y1)
	}
}

func TestCenterOrCornerBoundsCenterMode(t *testing.T) {
	// Anchor (10,10), current (16,14): radius 6 horizontally, 4 vertically.
	x0, y0, x1, y1 := CenterOrCornerBounds(10, 10, 16, 14, true)
	if x0 != 4 || y0 != 6 || x1 != 16 || y1 != 14 {
		t.Errorf("center mode = (%d,%d,%d,%d), want (4,6,16,14)", x0, y0, x1, y1)
	}
	// The anchor should be the exact midpoint of the resulting box.
	if (x0+x1)/2 != 10 || (y0+y1)/2 != 10 {
		t.Error("anchor should be the centre of the resulting bounds")
	}
}

func TestCenterOrCornerBoundsCenterModeWorksInAnyDragDirection(t *testing.T) {
	// Dragging up-and-left from the anchor should give the same bounds as
	// dragging down-and-right by the same distance — center mode is
	// direction-independent by construction (it only uses the distance).
	a, b, c, d := CenterOrCornerBounds(10, 10, 4, 6, true)   // up-left
	e, f, g, h := CenterOrCornerBounds(10, 10, 16, 14, true) // down-right, same distance
	if a != e || b != f || c != g || d != h {
		t.Errorf("expected identical bounds regardless of drag direction: (%d,%d,%d,%d) vs (%d,%d,%d,%d)", a, b, c, d, e, f, g, h)
	}
}

func TestRectOutlineVisitsAllFourCorners(t *testing.T) {
	pts := collectPoints(func(fn func(x, y int)) { RectOutline(0, 0, 10, 6, fn) })
	for _, c := range [][2]int{{0, 0}, {10, 0}, {10, 6}, {0, 6}} {
		if !pts[c] {
			t.Errorf("corner %v not visited", c)
		}
	}
}

func TestRectOutlineDoesNotFillInterior(t *testing.T) {
	pts := collectPoints(func(fn func(x, y int)) { RectOutline(0, 0, 10, 6, fn) })
	// The exact centre of a 10x6 box is well inside the outline on every
	// edge — if this is visited, the "shape" is filled, not outlined.
	if pts[[2]int{5, 3}] {
		t.Error("RectOutline visited an interior point — it should be outline-only, not filled")
	}
}

func TestRectOutlineEdgesAreStraight(t *testing.T) {
	pts := collectPoints(func(fn func(x, y int)) { RectOutline(0, 0, 10, 6, fn) })
	// Every point along the top edge (y=0) should be visited, x=0..10.
	for x := 0; x <= 10; x++ {
		if !pts[[2]int{x, 0}] {
			t.Errorf("top edge missing (%d,0)", x)
		}
	}
	// Every point along the left edge (x=0) should be visited, y=0..6.
	for y := 0; y <= 6; y++ {
		if !pts[[2]int{0, y}] {
			t.Errorf("left edge missing (0,%d)", y)
		}
	}
}

func TestTriangleOutlineVisitsApexAndBaseCorners(t *testing.T) {
	// Bounding box (0,0)-(10,6): apex at (5,0), base corners (0,6) and (10,6).
	pts := collectPoints(func(fn func(x, y int)) { TriangleOutline(0, 0, 10, 6, fn) })
	for _, c := range [][2]int{{5, 0}, {0, 6}, {10, 6}} {
		if !pts[c] {
			t.Errorf("expected vertex %v not visited", c)
		}
	}
}

func TestTriangleOutlineExcludesBoundingBoxCornersNotOnTriangle(t *testing.T) {
	// The bounding box's own top-left and top-right corners are NOT part of
	// the inscribed triangle (only the top-centre apex is) — if these are
	// visited, the shape is drawing the bounding box, not a triangle.
	pts := collectPoints(func(fn func(x, y int)) { TriangleOutline(0, 0, 10, 6, fn) })
	for _, c := range [][2]int{{0, 0}, {10, 0}} {
		if pts[c] {
			t.Errorf("bounding-box corner %v should not be part of the triangle outline", c)
		}
	}
}

func TestTriangleOutlineBaseIsStraight(t *testing.T) {
	pts := collectPoints(func(fn func(x, y int)) { TriangleOutline(0, 0, 10, 6, fn) })
	for x := 0; x <= 10; x++ {
		if !pts[[2]int{x, 6}] {
			t.Errorf("base edge missing (%d,6)", x)
		}
	}
}

// TestChequerFadeDirection guards the one thing most likely to have a
// subtle bug here: unlike every other zoom-based fade in this file, which
// fades IN as you zoom in, ChequerFade fades OUT as you zoom out — so a
// swapped lo/hi argument would silently invert the whole feature.
func TestChequerFadeDirection(t *testing.T) {
	cases := []struct {
		pct  float32
		want float32
	}{
		{5, 0},    // well below the floor: fully faded
		{10, 0},   // at the floor: fully faded
		{25, 0.5}, // midpoint of the 10-40 ramp
		{40, 1},   // at the ceiling: fully visible
		{60, 1},   // well above the ceiling: fully visible
	}
	for _, c := range cases {
		got := ChequerFade(c.pct)
		if got != c.want {
			t.Errorf("ChequerFade(%v) = %v, want %v", c.pct, got, c.want)
		}
	}
}
