package zenui

import "testing"

// With noopRenderer, dlgScale=2: TextWidth(s,2) = len(s)*16, LineHeight(2) = 16.
// padX = lh/2 = 8, padY = lh/4 = 4, itemH = lh+2*padY = 24.
// "Insert" -> TextWidth = 6*16 = 96, +2*padX(16) = 112, which exceeds
// minMenuW(80), so menuW = 112 for all three positioning cases below.

func testItems() []Item {
	return []Item{
		{Label: "Insert"},
		{Label: "Delete", Disabled: true},
		{Label: "Copy"},
	}
}

func TestMenuLayoutRoomToRight(t *testing.T) {
	m := NewMenu(MenuConfig{Items: testItems(), Anchor: Rect{X: 10, Y: 50, W: 56, H: 24}})
	m.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	want := Rect{X: 10, Y: 74, W: 112, H: 24 * 3}
	if m.bounds != want {
		t.Fatalf("bounds = %+v, want %+v", m.bounds, want)
	}
}

func TestMenuLayoutRoomToLeftOnly(t *testing.T) {
	// screenW=1280; anchor near the right edge so the menu can't fit to the
	// right (1200+112=1312 > 1280) but fits aligned to the anchor's right
	// edge (1200+56-112=1144 >= 0).
	m := NewMenu(MenuConfig{Items: testItems(), Anchor: Rect{X: 1200, Y: 50, W: 56, H: 24}})
	m.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	want := Rect{X: 1144, Y: 74, W: 112, H: 24 * 3}
	if m.bounds != want {
		t.Fatalf("bounds = %+v, want %+v", m.bounds, want)
	}
}

func TestMenuLayoutClampedWhenNeitherFits(t *testing.T) {
	// screenW=100 is narrower than menuW=112, so neither the right-aligned
	// nor the left-aligned placement fits; clamp to x=0.
	m := NewMenu(MenuConfig{Items: testItems(), Anchor: Rect{X: 0, Y: 50, W: 56, H: 24}})
	m.Draw(noopRenderer{}, 100, 800, DefaultTheme())

	if m.bounds.X != 0 {
		t.Fatalf("bounds.X = %d, want 0 (clamped)", m.bounds.X)
	}
}

func TestMenuClickEnabledItemAccepts(t *testing.T) {
	m := NewMenu(MenuConfig{Items: testItems(), Anchor: Rect{X: 10, Y: 50, W: 56, H: 24}})
	m.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	rec := m.itemRects[0] // "Insert"
	status := m.Update(Input{MouseX: rec.X + 4, MouseY: rec.Y + 4, MousePressed: true})

	if status != Accepted || m.Status() != Accepted {
		t.Fatalf("status = %v, want Accepted", status)
	}
	if m.Result() != 0 {
		t.Fatalf("Result() = %d, want 0", m.Result())
	}
}

func TestMenuClickDisabledItemIsNoop(t *testing.T) {
	m := NewMenu(MenuConfig{Items: testItems(), Anchor: Rect{X: 10, Y: 50, W: 56, H: 24}})
	m.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	rec := m.itemRects[1] // "Delete", disabled
	status := m.Update(Input{MouseX: rec.X + 4, MouseY: rec.Y + 4, MousePressed: true})

	if status != Active {
		t.Fatalf("status = %v, want Active (click on disabled item is a no-op)", status)
	}
}

func TestMenuClickOutsideCancels(t *testing.T) {
	m := NewMenu(MenuConfig{Items: testItems(), Anchor: Rect{X: 10, Y: 50, W: 56, H: 24}})
	m.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	status := m.Update(Input{MouseX: 900, MouseY: 700, MousePressed: true})

	if status != Cancelled {
		t.Fatalf("status = %v, want Cancelled", status)
	}
}

func TestMenuEscapeCancels(t *testing.T) {
	m := NewMenu(MenuConfig{Items: testItems(), Anchor: Rect{X: 10, Y: 50, W: 56, H: 24}})
	m.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	status := m.Update(Input{Keys: []Key{KeyEscape}})

	if status != Cancelled {
		t.Fatalf("status = %v, want Cancelled", status)
	}
}

func TestMenuKeyboardNavSkipsDisabledThenAccepts(t *testing.T) {
	m := NewMenu(MenuConfig{Items: testItems(), Anchor: Rect{X: 10, Y: 50, W: 56, H: 24}})
	m.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	m.Update(Input{Keys: []Key{KeyDown}}) // selected -> 0 ("Insert")
	if m.selected != 0 {
		t.Fatalf("after first Down, selected = %d, want 0", m.selected)
	}
	m.Update(Input{Keys: []Key{KeyDown}}) // selected -> 2 ("Copy"), skipping disabled 1
	if m.selected != 2 {
		t.Fatalf("after second Down, selected = %d, want 2 (skip disabled index 1)", m.selected)
	}

	status := m.Update(Input{Keys: []Key{KeyEnter}})
	if status != Accepted || m.Result() != 2 {
		t.Fatalf("status = %v, Result() = %d, want Accepted/2", status, m.Result())
	}
}

func TestMenuUpdateAfterConclusionIsIdempotent(t *testing.T) {
	m := NewMenu(MenuConfig{Items: testItems(), Anchor: Rect{X: 10, Y: 50, W: 56, H: 24}})
	m.Draw(noopRenderer{}, 1280, 800, DefaultTheme())

	m.Update(Input{Keys: []Key{KeyEscape}})
	if m.Status() != Cancelled {
		t.Fatalf("Status() = %v, want Cancelled", m.Status())
	}
	// A second Update after conclusion must not change anything further.
	status := m.Update(Input{MouseX: m.itemRects[0].X + 4, MouseY: m.itemRects[0].Y + 4, MousePressed: true})
	if status != Cancelled {
		t.Fatalf("status after post-conclusion Update = %v, want Cancelled (unchanged)", status)
	}
}
