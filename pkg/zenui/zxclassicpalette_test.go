package zenui

import (
	"testing"

	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

func testChooser() *ZXClassicPaletteChooser {
	return NewZXClassicPaletteChooser(ZXClassicPaletteChooserConfig{
		Anchor: Rect{X: 100, Y: 50}, SwatchW: 36, SwatchH: 24, GapX: 6, GapY: 5,
	})
}

// TestZXClassicPaletteLayout hand-verifies the classic colour-key grid:
// row 0 = Blue, Blue-bright, Black, Black-bright; row 1 = Red, Red-bright,
// Magenta, Magenta-bright — matching the arrangement printed on the
// Spectrum's own keyboard.
func TestZXClassicPaletteLayout(t *testing.T) {
	p := testChooser()

	cases := []struct {
		i          int
		wantRect   Rect
		wantBase   int
		wantBright bool
	}{
		{0, Rect{100, 50, 36, 24}, zxpalette.Blue, false},
		{1, Rect{142, 50, 36, 24}, zxpalette.Blue, true},
		{2, Rect{184, 50, 36, 24}, zxpalette.Black, false},
		{4, Rect{100, 79, 36, 24}, zxpalette.Red, false}, // row 1 starts at y=50+29=79
	}
	for _, c := range cases {
		sw := p.swatches[c.i]
		if sw.rect != c.wantRect {
			t.Errorf("swatch %d rect = %+v, want %+v", c.i, sw.rect, c.wantRect)
		}
		if sw.base != c.wantBase || sw.bright != c.wantBright {
			t.Errorf("swatch %d = base %d bright %v, want base %d bright %v",
				c.i, sw.base, sw.bright, c.wantBase, c.wantBright)
		}
	}
}

func TestZXClassicPaletteBounds(t *testing.T) {
	p := testChooser()
	want := Rect{X: 100, Y: 50, W: 162, H: 111} // 4*36+3*6=162, 4*24+3*5=111
	if got := p.Bounds(); got != want {
		t.Errorf("Bounds() = %+v, want %+v", got, want)
	}
}

func TestZXClassicPaletteInkPick(t *testing.T) {
	p := testChooser()
	rec := p.swatches[0] // Blue, normal
	res := p.Update(Input{MouseX: rec.rect.X + 4, MouseY: rec.rect.Y + 4, MousePressed: true})
	if !res.InkPicked || res.PaperPicked {
		t.Fatalf("expected ink pick only, got %+v", res)
	}
	if res.Base != zxpalette.Blue || res.Bright {
		t.Errorf("picked base=%d bright=%v, want Blue/false", res.Base, res.Bright)
	}
}

func TestZXClassicPalettePaperPick(t *testing.T) {
	p := testChooser()
	rec := p.swatches[1] // Blue, bright
	res := p.Update(Input{MouseX: rec.rect.X + 4, MouseY: rec.rect.Y + 4, MouseRightPressed: true})
	if !res.PaperPicked || res.InkPicked {
		t.Fatalf("expected paper pick only, got %+v", res)
	}
	if res.Base != zxpalette.Blue || !res.Bright {
		t.Errorf("picked base=%d bright=%v, want Blue/true", res.Base, res.Bright)
	}
}

func TestZXClassicPaletteClickOutsideIsNoop(t *testing.T) {
	p := testChooser()
	res := p.Update(Input{MouseX: 0, MouseY: 0, MousePressed: true})
	if res.InkPicked || res.PaperPicked {
		t.Errorf("click outside any swatch should be a no-op, got %+v", res)
	}
}

func TestZXClassicPaletteSetBounds(t *testing.T) {
	p := testChooser()
	p.SetBounds(Rect{X: 200, Y: 200})
	if p.swatches[0].rect.X != 200 || p.swatches[0].rect.Y != 200 {
		t.Errorf("SetBounds did not reposition swatch 0: %+v", p.swatches[0].rect)
	}
}

// countingRenderer wraps noopRenderer but counts FillRect calls, to prove
// Draw's alpha<=0 early return genuinely skips drawing rather than just
// drawing invisibly.
type countingRenderer struct {
	noopRenderer
	fillCalls *int
}

func (r countingRenderer) FillRect(Rect, Colour) { *r.fillCalls++ }

func TestZXClassicPaletteDrawSkipsAtZeroAlpha(t *testing.T) {
	p := testChooser()
	calls := 0
	p.Draw(countingRenderer{fillCalls: &calls}, DefaultTheme(), zxpalette.Blue, zxpalette.Black, false, 0)
	if calls != 0 {
		t.Errorf("Draw at alpha=0 called FillRect %d times, want 0", calls)
	}
	p.Draw(countingRenderer{fillCalls: &calls}, DefaultTheme(), zxpalette.Blue, zxpalette.Black, false, 1)
	if calls == 0 {
		t.Error("Draw at alpha=1 should have called FillRect")
	}
}
