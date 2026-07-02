//go:build purego

package main

import (
	"testing"

	"github.com/ha1tch/zenimate/internal/ui"
)

func TestTruncateLabel(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"short.zani", 30, "short.zani"},
		{"exactly-thirty-chars-long-abcd", 30, "exactly-thirty-chars-long-abcd"}, // 30 chars, unchanged
		{"this-is-a-very-long-filename-that-exceeds-limit.zani", 30, "this-is-a-very-long-filename-t..."},
	}
	for _, c := range cases {
		if got := truncateLabel(c.in, c.max); got != c.want {
			t.Errorf("truncateLabel(%q,%d) = %q, want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestTruncateLabelBoundary(t *testing.T) {
	// A 31-char string should become 30 chars + "...".
	s := "0123456789012345678901234567890" // 31 chars
	got := truncateLabel(s, 30)
	if len([]rune(got)) != 33 { // 30 + 3 dots
		t.Errorf("expected 33 runes, got %d (%q)", len([]rune(got)), got)
	}
	if got[len(got)-3:] != "..." {
		t.Errorf("expected trailing ellipsis, got %q", got)
	}
}

func TestSelectModeForExtScr(t *testing.T) {
	c := ui.New(16, 16)
	c.SetMode(ui.BitmapBlack)
	selectModeForExt(c, "scr")
	if c.Mode() != ui.SpectrumColour {
		t.Errorf("after .scr load, mode = %v, want SpectrumColour", c.Mode())
	}
}

func TestSelectModeForExtOthersUnchanged(t *testing.T) {
	for _, ext := range []string{"zani", "tap", "png", "z80", ""} {
		c := ui.New(16, 16)
		c.SetMode(ui.BitmapWhite)
		selectModeForExt(c, ext)
		if c.Mode() != ui.BitmapWhite {
			t.Errorf("ext %q changed mode to %v, want unchanged", ext, c.Mode())
		}
	}
}

func TestHelpAnchorFixedAcrossFrameCounts(t *testing.T) {
	// The HELP button must stay put as frames are added: its X should not depend
	// on the current frame count.
	mk := func(frames int) float32 {
		c := ui.New(16, 16)
		for c.Sprite.FrameCount() > 1 {
			c.Sprite.RemoveFrame()
		}
		for c.Sprite.FrameCount() < frames {
			c.Sprite.AddFrame()
		}
		var files fileOps
		l := computeLayout(1200, 800, c, &files, 0, 1)
		return l.helpRect.X
	}
	x8 := mk(8)
	for _, n := range []int{1, 2, 4, 8, 12, 16} {
		if got := mk(n); got != x8 {
			t.Errorf("helpRect.X at %d frames = %.1f, want %.1f (fixed)", n, got, x8)
		}
	}
}

func TestForEachLinePixelContiguous(t *testing.T) {
	// A line's pixels must be 8-connected (no gaps): each step moves at most 1 in
	// x and y from the previous.
	var pts [][2]int
	forEachLinePixel(0, 0, 10, 4, func(x, y int) { pts = append(pts, [2]int{x, y}) })
	if len(pts) == 0 {
		t.Fatal("no pixels produced")
	}
	if pts[0] != [2]int{0, 0} || pts[len(pts)-1] != [2]int{10, 4} {
		t.Errorf("endpoints wrong: %v .. %v", pts[0], pts[len(pts)-1])
	}
	for i := 1; i < len(pts); i++ {
		dx := pts[i][0] - pts[i-1][0]
		dy := pts[i][1] - pts[i-1][1]
		if dx < -1 || dx > 1 || dy < -1 || dy > 1 {
			t.Errorf("gap between %v and %v", pts[i-1], pts[i])
		}
	}
}

func TestForEachLinePixelSinglePoint(t *testing.T) {
	n := 0
	forEachLinePixel(3, 3, 3, 3, func(x, y int) {
		n++
		if x != 3 || y != 3 {
			t.Errorf("unexpected pixel (%d,%d)", x, y)
		}
	})
	if n != 1 {
		t.Errorf("single point should yield 1 pixel, got %d", n)
	}
}

func TestForEachLinePixelReversed(t *testing.T) {
	// Reversed endpoints cover the same set of pixels.
	fwd := map[[2]int]bool{}
	forEachLinePixel(2, 1, 9, 7, func(x, y int) { fwd[[2]int{x, y}] = true })
	rev := map[[2]int]bool{}
	forEachLinePixel(9, 7, 2, 1, func(x, y int) { rev[[2]int{x, y}] = true })
	if len(fwd) != len(rev) {
		t.Errorf("forward %d pixels, reverse %d", len(fwd), len(rev))
	}
	for p := range fwd {
		if !rev[p] {
			t.Errorf("pixel %v missing in reverse", p)
		}
	}
}

func TestOnionAnchorFixedAcrossFrameCounts(t *testing.T) {
	// Onion buttons must stay put as frames are added, like the HELP button.
	mk := func(frames int) int {
		c := ui.New(16, 16)
		for c.Sprite.FrameCount() > 1 {
			c.Sprite.RemoveFrame()
		}
		for c.Sprite.FrameCount() < frames {
			c.Sprite.AddFrame()
		}
		var files fileOps
		l := computeLayout(1200, 800, c, &files, 0, 1)
		return l.onionButtons[0].x
	}
	x8 := mk(8)
	for _, n := range []int{1, 2, 4, 8, 12, 16} {
		if got := mk(n); got != x8 {
			t.Errorf("onion X at %d frames = %d, want %d (fixed)", n, got, x8)
		}
	}
}

func TestCopyPasteButtonsPresent(t *testing.T) {
	c := ui.New(16, 16)
	var files fileOps
	l := computeLayout(1200, 800, c, &files, 0, 1)
	var hasCopy, hasPaste bool
	for _, b := range l.buttons {
		switch b.label {
		case "Copy":
			hasCopy = true
		case "Paste":
			hasPaste = true
		}
	}
	if !hasCopy || !hasPaste {
		t.Errorf("expected COPY and PASTE buttons; copy=%v paste=%v", hasCopy, hasPaste)
	}
}

func TestSizingButtonsRelabeled(t *testing.T) {
	c := ui.New(16, 16)
	var files fileOps
	l := computeLayout(1400, 800, c, &files, 0, 1)
	have := map[string]bool{}
	for _, b := range l.buttons {
		have[b.label] = true
	}
	// Cell-unit stepper labels and the new preset buttons.
	for _, want := range []string{"W -1", "W +1", "H -1", "H +1", "32x24", "2x2"} {
		if !have[want] {
			t.Errorf("missing button %q", want)
		}
	}
	// Old pixel labels and the old name must be gone.
	for _, gone := range []string{"W -8", "W +8", "256x192", "Full Screen"} {
		if have[gone] {
			t.Errorf("stale button %q still present", gone)
		}
	}
}

func TestExportBundleAlignUnderCopyPaste(t *testing.T) {
	c := ui.New(16, 16)
	var files fileOps
	l := computeLayout(1400, 800, c, &files, 0, 1)
	pos := map[string]button{}
	for _, b := range l.buttons {
		pos[b.label] = b
	}
	// Export directly below Copy (same x, same width); Bundle below Paste.
	if pos["Export"].x != pos["Copy"].x || pos["Export"].w != pos["Copy"].w {
		t.Errorf("Export not aligned under Copy: export(x=%d,w=%d) copy(x=%d,w=%d)",
			pos["Export"].x, pos["Export"].w, pos["Copy"].x, pos["Copy"].w)
	}
	if pos["Bundle"].x != pos["Paste"].x || pos["Bundle"].w != pos["Paste"].w {
		t.Errorf("Bundle not aligned under Paste: bundle(x=%d,w=%d) paste(x=%d,w=%d)",
			pos["Bundle"].x, pos["Bundle"].w, pos["Paste"].x, pos["Paste"].w)
	}
}

func TestResponsiveButtonsShrink(t *testing.T) {
	c := ui.New(16, 16)
	var files fileOps
	wide := computeLayout(1600, 800, c, &files, 0, 1)
	narrow := computeLayout(560, 800, c, &files, 0, 1)
	if narrow.stripBtnW >= wide.stripBtnW {
		t.Errorf("buttons should shrink in a narrow window: narrow=%d wide=%d",
			narrow.stripBtnW, wide.stripBtnW)
	}
	if narrow.stripBtnW < 40 {
		t.Errorf("shrunk button width %d below the floor", narrow.stripBtnW)
	}
}

func TestScrubberPresent(t *testing.T) {
	c := ui.New(16, 16)
	var files fileOps
	l := computeLayout(1400, 800, c, &files, 0, 1)
	if l.scrubRect.Width <= 0 || l.scrubRect.Height <= 0 {
		t.Errorf("scrubber rect not laid out: %+v", l.scrubRect)
	}
	// The scrubber sits above the frame strip.
	if l.scrubRect.Y >= float32(l.frameStripY) {
		t.Errorf("scrubber Y %.0f should be above frame strip Y %d", l.scrubRect.Y, l.frameStripY)
	}
}

func TestCellGuideFade(t *testing.T) {
	// Zoom percentage: full at >=100%, gone at <=37%.
	cases := []struct{ pct, want float32 }{
		{120, 1}, {100, 1}, {68.5, 0.5}, {37, 0}, {20, 0},
	}
	for _, c := range cases {
		if got := cellGuideFade(c.pct); got-c.want > 0.001 || got-c.want < -0.001 {
			t.Errorf("cellGuideFade(%.1f) = %.3f, want %.3f", c.pct, got, c.want)
		}
	}
}

func TestPixGridFade(t *testing.T) {
	// Full at >=168%, gone at <=42%.
	cases := []struct{ pct, want float32 }{
		{180, 1}, {168, 1}, {105, 0.5}, {42, 0}, {20, 0},
	}
	for _, c := range cases {
		if got := pixGridFade(c.pct); got-c.want > 0.001 || got-c.want < -0.001 {
			t.Errorf("pixGridFade(%.1f) = %.3f, want %.3f", c.pct, got, c.want)
		}
	}
}

func TestCellGuideFadesEarlierThanPixGrid(t *testing.T) {
	// Cell guides (37-100) reach full before the pixel grid (42-168), so at any
	// percentage the guides are at least as visible as the pixel grid.
	for _, p := range []float32{42, 60, 100, 150, 168} {
		if cellGuideFade(p) < pixGridFade(p) {
			t.Errorf("at %.0f%% cell guide (%.2f) less visible than pixgrid (%.2f)", p, cellGuideFade(p), pixGridFade(p))
		}
	}
}

func TestButtonVisible(t *testing.T) {
	// Viewport right edge at x=500. Button width 100.
	cases := []struct {
		bx, bw, vp int
		want       bool
	}{
		{100, 100, 500, true},  // right border 200, within edge: visible
		{400, 100, 500, true},  // right border exactly at edge: visible
		{401, 100, 500, false}, // right border 501, just past: hidden
		{500, 100, 500, false}, // wholly past: hidden
		{560, 100, 500, false}, // entirely beyond: hidden
	}
	for _, c := range cases {
		if got := buttonVisible(c.bx, c.bw, c.vp); got != c.want {
			t.Errorf("buttonVisible(%d,%d,%d) = %v, want %v", c.bx, c.bw, c.vp, got, c.want)
		}
	}
}

func TestChequerOffColour(t *testing.T) {
	// Bitmap White off -> darkest chequer for that mode = dark base shaded darker.
	wantWhite := shadeChequer(colChkDark, ui.BitmapWhite)
	if got := chequerOffColour(ui.BitmapWhite); got != wantWhite {
		t.Errorf("BitmapWhite off colour = %v, want %v (darkest)", got, wantWhite)
	}
	// Bitmap Black off -> lightest chequer for that mode = light base shaded lighter.
	wantBlack := shadeChequer(colChkLight, ui.BitmapBlack)
	if got := chequerOffColour(ui.BitmapBlack); got != wantBlack {
		t.Errorf("BitmapBlack off colour = %v, want %v (lightest)", got, wantBlack)
	}
	// Sanity: the White-off shade is darker than the Black-off shade.
	if chequerOffColour(ui.BitmapWhite).R >= chequerOffColour(ui.BitmapBlack).R {
		t.Errorf("White-off (%d) should be darker than Black-off (%d)",
			chequerOffColour(ui.BitmapWhite).R, chequerOffColour(ui.BitmapBlack).R)
	}
}

func TestChequerLedsLaidOut(t *testing.T) {
	c := ui.New(16, 16)
	var files fileOps
	l := computeLayout(1400, 800, c, &files, 0, 1)
	if l.chkLedWhite.Width <= 0 || l.chkLedBlack.Width <= 0 {
		t.Fatal("chequer LEDs not laid out")
	}
	// Each LED sits below its mode button and is horizontally centred on it.
	wb := l.modeButtons[0]
	bb := l.modeButtons[1]
	if l.chkLedWhite.Y <= float32(wb.y) {
		t.Error("white LED should be below the Bitmap White button")
	}
	wantCx := float32(wb.x) + float32(wb.w)/2
	gotCx := l.chkLedWhite.X + l.chkLedWhite.Width/2
	if diff := gotCx - wantCx; diff > 1 || diff < -1 {
		t.Errorf("white LED not centred: centre %.1f, button centre %.1f", gotCx, wantCx)
	}
	wantCx2 := float32(bb.x) + float32(bb.w)/2
	gotCx2 := l.chkLedBlack.X + l.chkLedBlack.Width/2
	if diff := gotCx2 - wantCx2; diff > 1 || diff < -1 {
		t.Errorf("black LED not centred: centre %.1f, button centre %.1f", gotCx2, wantCx2)
	}
}

func TestTransformButtonsPresent(t *testing.T) {
	c := ui.New(16, 16)
	var files fileOps
	l := computeLayout(1600, 800, c, &files, 0, 1)
	have := map[string]bool{}
	for _, b := range l.buttons {
		have[b.label] = true
	}
	for _, want := range []string{"H FLIP", "V FLIP", "ROT 90", "INVERT"} {
		if !have[want] {
			t.Errorf("missing transform button %q", want)
		}
	}
}

func TestEqualFoldYES(t *testing.T) {
	for _, s := range []string{"YES", "yes", "Yes", "yEs"} {
		if !equalFoldYES(s) {
			t.Errorf("equalFoldYES(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"", "Y", "YE", "YESS", "NO", "y e s"} {
		if equalFoldYES(s) {
			t.Errorf("equalFoldYES(%q) = true, want false", s)
		}
	}
}

func TestResetClsButtonsPresent(t *testing.T) {
	c := ui.New(16, 16)
	var files fileOps
	l := computeLayout(1600, 800, c, &files, 0, 1)
	have := map[string]bool{}
	for _, b := range l.buttons {
		have[b.label] = true
	}
	if !have["RESET"] || !have["CLS"] {
		t.Errorf("expected RESET and CLS buttons; reset=%v cls=%v", have["RESET"], have["CLS"])
	}
	if have["Reset"] {
		t.Error("old single 'Reset' button should be gone")
	}
}

func TestFlatCellFade(t *testing.T) {
	// Full at >=411%, gone at <=168%.
	cases := []struct{ pct, want float32 }{
		{450, 1}, {411, 1}, {289.5, 0.5}, {168, 0}, {100, 0},
	}
	for _, c := range cases {
		if got := flatCellFade(c.pct); got-c.want > 0.001 || got-c.want < -0.001 {
			t.Errorf("flatCellFade(%.1f) = %.3f, want %.3f", c.pct, got, c.want)
		}
	}
}

func TestPppToPercent(t *testing.T) {
	setZoomRangeForScreen(1920) // fit=10, pppMin=8 (0.8x), pppMax=80
	defer setZoomRangeForScreen(1920)
	cases := []struct{ ppp, want float32 }{
		{8, 0},    // pppMin -> 0%
		{80, 800}, // pppMax -> 800%
		{44, 400}, // midpoint
	}
	for _, c := range cases {
		if got := pppToPercent(c.ppp); got-c.want > 0.5 || got-c.want < -0.5 {
			t.Errorf("pppToPercent(%.1f) = %.2f, want ~%.1f", c.ppp, got, c.want)
		}
	}
}

func TestSetZoomRangeForScreen(t *testing.T) {
	setZoomRangeForScreen(1920)
	// 800% anchor: tallest sprite (192px) at pppMax spans 8x its screen-height fit.
	fit := float32(1920) / float32(MaxSpriteHeightPx)
	if pppMax-fit*8 > 0.01 || pppMax-fit*8 < -0.01 {
		t.Errorf("pppMax=%.3f, want 8x fit=%.3f", pppMax, fit*8)
	}
	// 0% floor is 0.8x the fit-to-screen size (20%% more zoom-out room).
	if pppMin-fit*0.8 > 0.01 || pppMin-fit*0.8 < -0.01 {
		t.Errorf("pppMin=%.3f, want 0.8x fit=%.3f", pppMin, fit*0.8)
	}
	// minZoom must let the wheel reach pppMin.
	if got := cellPx * minZoom; got-pppMin > 0.01 || got-pppMin < -0.01 {
		t.Errorf("cellPx*minZoom = %.3f, want pppMin %.3f", got, pppMin)
	}
}
