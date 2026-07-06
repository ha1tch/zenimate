package zenui

import "testing"

func testTools(n int) []Tool {
	tools := make([]Tool, n)
	for i := range tools {
		tools[i] = Tool{ID: string(rune('a' + i)), Glyph: rune(0xE100 + i)}
	}
	return tools
}

func testPalette(n, cols int) *ToolPalette {
	return NewToolPalette(ToolPaletteConfig{
		Anchor: Rect{X: 100, Y: 50}, Tools: testTools(n), Cols: cols,
		ButtonW: 36, ButtonH: 36, GapX: 4, GapY: 4, GlyphSize: 24,
	})
}

// TestToolPaletteWrapsRows hand-verifies the grid wraps correctly when the
// tool count isn't an exact multiple of Cols — 5 tools over 4 columns must
// produce 2 rows, the second only partially filled, not padded or dropped.
func TestToolPaletteWrapsRows(t *testing.T) {
	p := testPalette(5, 4)
	cases := []struct {
		i        int
		wantRect Rect
	}{
		{0, Rect{100, 50, 36, 36}}, // row 0, col 0
		{3, Rect{220, 50, 36, 36}}, // row 0, col 3 (100+3*40)
		{4, Rect{100, 90, 36, 36}}, // row 1, col 0 (50+1*40)
	}
	for _, c := range cases {
		got := p.buttons[c.i].rect
		if got != c.wantRect {
			t.Errorf("button %d rect = %+v, want %+v", c.i, got, c.wantRect)
		}
	}
}

func TestToolPaletteBoundsAccountsForWrapping(t *testing.T) {
	p := testPalette(5, 4)
	// 4 cols -> w = 4*36+3*4 = 156; 5 tools/4cols -> 2 rows -> h = 2*36+1*4 = 76.
	want := Rect{X: 100, Y: 50, W: 156, H: 76}
	if got := p.Bounds(); got != want {
		t.Errorf("Bounds() = %+v, want %+v", got, want)
	}
}

func TestToolPaletteExactMultipleNoExtraRow(t *testing.T) {
	p := testPalette(12, 4) // exactly 3 rows, no partial 4th
	want := Rect{X: 100, Y: 50, W: 156, H: 3*36 + 2*4}
	if got := p.Bounds(); got != want {
		t.Errorf("Bounds() = %+v, want %+v (12 tools / 4 cols should be exactly 3 rows)", got, want)
	}
}

func TestToolPaletteDefaultSelectionIsFirstTool(t *testing.T) {
	p := testPalette(5, 4)
	id, ok := p.Selected()
	if !ok || id != "a" {
		t.Errorf("Selected() = %q, %v, want \"a\", true", id, ok)
	}
}

func TestToolPaletteEmptyToolsIsSafe(t *testing.T) {
	p := testPalette(0, 4)
	if _, ok := p.Selected(); ok {
		t.Error("Selected() on an empty palette should report ok=false")
	}
	b := p.Bounds()
	if b.W != 0 || b.H != 0 {
		t.Errorf("Bounds() on an empty palette = %+v, want zero size", b)
	}
	// Update must not panic on an empty palette.
	p.Update(Input{MouseX: 100, MouseY: 50, MousePressed: true})
}

func TestToolPalettePickUpdatesSelection(t *testing.T) {
	p := testPalette(5, 4)
	target := p.buttons[3].rect // row 0, col 3
	res := p.Update(Input{MouseX: target.X + 5, MouseY: target.Y + 5, MousePressed: true})
	if !res.Picked || res.ID != "d" {
		t.Fatalf("pick result = %+v, want Picked=true ID=\"d\"", res)
	}
	if id, ok := p.Selected(); !ok || id != "d" {
		t.Errorf("Selected() after pick = %q, %v, want \"d\", true", id, ok)
	}
}

func TestToolPaletteClickOutsideIsNoop(t *testing.T) {
	p := testPalette(5, 4)
	res := p.Update(Input{MouseX: 0, MouseY: 0, MousePressed: true})
	if res.Picked {
		t.Errorf("click outside any button should be a no-op, got %+v", res)
	}
	if id, _ := p.Selected(); id != "a" {
		t.Errorf("selection should be unchanged by a click outside, got %q", id)
	}
}

func TestToolPaletteRectForMatchesLayout(t *testing.T) {
	p := testPalette(5, 4)
	rect, ok := p.RectFor("d") // index 3, row 0 col 3
	if !ok || rect != p.buttons[3].rect {
		t.Errorf("RectFor(\"d\") = %+v, %v, want %+v, true", rect, ok, p.buttons[3].rect)
	}
	if _, ok := p.RectFor("nonexistent"); ok {
		t.Error("RectFor with an unknown ID should report ok=false")
	}
}

func TestToolPaletteSetSelectedRestoresChoice(t *testing.T) {
	p := testPalette(5, 4)
	p.SetSelected("d")
	if id, _ := p.Selected(); id != "d" {
		t.Errorf("Selected() = %q after SetSelected(\"d\"), want \"d\"", id)
	}
	p.SetSelected("nonexistent")
	if id, _ := p.Selected(); id != "d" {
		t.Errorf("SetSelected with an unknown ID should be a no-op, but selection changed to %q", id)
	}
}

func TestToolPaletteHitTestFindsCorrectButtonWithoutSideEffects(t *testing.T) {
	p := testPalette(5, 4)
	target := p.buttons[3].rect
	id, ok := p.HitTest(target.X+5, target.Y+5)
	if !ok || id != "d" {
		t.Fatalf("HitTest = %q, %v, want \"d\", true", id, ok)
	}
	// Must not have changed hover or selection — it's a pure query.
	if selID, _ := p.Selected(); selID != "a" {
		t.Errorf("HitTest changed selection to %q, want unchanged \"a\"", selID)
	}
	if _, ok := p.HitTest(0, 0); ok {
		t.Error("HitTest outside any button should report ok=false")
	}
}

func TestToolPaletteSetBoundsRepositions(t *testing.T) {
	p := testPalette(5, 4)
	p.SetBounds(Rect{X: 200, Y: 200})
	if p.buttons[0].rect.X != 200 || p.buttons[0].rect.Y != 200 {
		t.Errorf("SetBounds did not reposition button 0: %+v", p.buttons[0].rect)
	}
}

// glyphRecorder captures every DrawText call's text and position, to verify
// the centring math precisely rather than just that something was drawn.
type glyphRecorder struct {
	noopRenderer
	calls *[]struct {
		text string
		x, y int
	}
}

func (r glyphRecorder) DrawText(s string, x, y, scale int, c Colour) {
	*r.calls = append(*r.calls, struct {
		text string
		x, y int
	}{s, x, y})
}

// TestToolPaletteCentresGlyphByGlyphSizeNotCellSize is the important one: it
// confirms centring uses GlyphSize (the icon's real visual size) rather than
// something derived from the renderer's own TextWidth/LineHeight, which
// would reflect the font's full (and, for this icon face, much larger and
// top-left-anchored) cell box — see the doc comment on GlyphSize.
func TestToolPaletteCentresGlyphByGlyphSizeNotCellSize(t *testing.T) {
	p := NewToolPalette(ToolPaletteConfig{
		Anchor: Rect{X: 0, Y: 0}, Tools: testTools(1), Cols: 1,
		ButtonW: 36, ButtonH: 36, GapX: 0, GapY: 0, GlyphSize: 24,
	})
	var calls []struct {
		text string
		x, y int
	}
	p.Draw(glyphRecorder{calls: &calls}, DefaultTheme())
	if len(calls) != 1 {
		t.Fatalf("got %d DrawText calls, want 1", len(calls))
	}
	// Button is 36x36 at (0,0); a 24px glyph centred within it sits at
	// (36-24)/2 = 6 on each axis.
	if calls[0].x != 6 || calls[0].y != 6 {
		t.Errorf("glyph drawn at (%d,%d), want (6,6)", calls[0].x, calls[0].y)
	}
	if calls[0].text != string(rune(0xE100)) {
		t.Errorf("glyph text = %q, want the tool's own glyph rune", calls[0].text)
	}
}
