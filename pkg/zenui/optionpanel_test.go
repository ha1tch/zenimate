package zenui

import (
	"strings"
	"testing"
)

func TestOptionPanelAutoSizesToWidestContent(t *testing.T) {
	// noopRenderer: TextWidth(s,scale) = len(s)*8*scale, LineHeight(scale) = 8*scale.
	// dlgScale=2 -> lh=16, pad=16, rowH=28, gap=6.
	// widen(Title="T",2)  -> 1*16+16 = 32   (below minBW=200, no effect)
	// widen(opt, 1)       -> 30*8+16 = 256  (the widest, sets bw)
	// widen("CANCEL",1)   -> 6*8+16 = 64    (below 256, no effect)
	label := strings.Repeat("A", 30)
	p := NewOptionPanel(OptionPanelConfig{
		Title:   "T",
		Options: []Item{{Label: label}},
	})
	p.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	wantPanel := Rect{X: 496, Y: 337, W: 288, H: 126}
	if p.panel != wantPanel {
		t.Fatalf("panel = %+v, want %+v", p.panel, wantPanel)
	}
	wantItem := Rect{X: 512, Y: 377, W: 256, H: 28}
	if p.itemRects[0] != wantItem {
		t.Fatalf("itemRects[0] = %+v, want %+v", p.itemRects[0], wantItem)
	}
	wantCancel := Rect{X: 512, Y: 419, W: 256, H: 28}
	if p.cancelRect != wantCancel {
		t.Fatalf("cancelRect = %+v, want %+v", p.cancelRect, wantCancel)
	}
}

func TestOptionPanelClickOptionAccepts(t *testing.T) {
	p := NewOptionPanel(OptionPanelConfig{
		Title:   "T",
		Options: []Item{{Label: "First"}, {Label: "Second"}},
	})
	p.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	rec := p.itemRects[1]
	status := p.Update(Input{MouseX: rec.X + 4, MouseY: rec.Y + 4, MousePressed: true})

	if status != Accepted || p.Result() != 1 {
		t.Fatalf("status = %v, Result() = %d, want Accepted/1", status, p.Result())
	}
}

func TestOptionPanelClickCancelCancels(t *testing.T) {
	p := NewOptionPanel(OptionPanelConfig{
		Title:   "T",
		Options: []Item{{Label: "First"}},
	})
	p.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	rec := p.cancelRect
	status := p.Update(Input{MouseX: rec.X + 4, MouseY: rec.Y + 4, MousePressed: true})

	if status != Cancelled {
		t.Fatalf("status = %v, want Cancelled", status)
	}
}

func TestOptionPanelClickOutsideCancels(t *testing.T) {
	p := NewOptionPanel(OptionPanelConfig{
		Title:   "T",
		Options: []Item{{Label: "First"}},
	})
	p.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	status := p.Update(Input{MouseX: 5, MouseY: 5, MousePressed: true})

	if status != Cancelled {
		t.Fatalf("status = %v, want Cancelled", status)
	}
}

func TestOptionPanelEscapeCancels(t *testing.T) {
	p := NewOptionPanel(OptionPanelConfig{
		Title:   "T",
		Options: []Item{{Label: "First"}},
	})
	p.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	status := p.Update(Input{Keys: []Key{KeyEscape}})

	if status != Cancelled {
		t.Fatalf("status = %v, want Cancelled", status)
	}
}

func TestOptionPanelClickDisabledOptionIsNoop(t *testing.T) {
	p := NewOptionPanel(OptionPanelConfig{
		Title:   "T",
		Options: []Item{{Label: "Disabled one", Disabled: true}},
	})
	p.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	rec := p.itemRects[0]
	status := p.Update(Input{MouseX: rec.X + 4, MouseY: rec.Y + 4, MousePressed: true})

	if status != Active {
		t.Fatalf("status = %v, want Active (click on disabled option is a no-op)", status)
	}
}
