package zenui

// layout computes the menu's bounds and each item's row rect, positioned just
// below cfg.Anchor. The menu's left edge aligns to the anchor's left edge if
// there is room for the menu to the right of that point; otherwise its right
// edge aligns to the anchor's right edge, provided there is room to the left.
// If neither fits (a very narrow screen), the menu is clamped inside bounds.
func (m *Menu) layout(r Renderer, screenW, screenH int) {
	lh := r.LineHeight(dlgScale)
	padY := lh / 4
	padX := lh / 2
	itemH := lh + 2*padY

	const minMenuW = 80
	menuW := minMenuW
	for _, it := range m.cfg.Items {
		w := r.TextWidth(it.Label, dlgScale) + 2*padX
		if w > menuW {
			menuW = w
		}
	}
	menuH := itemH * len(m.cfg.Items)

	a := m.cfg.Anchor
	y := a.Y + a.H

	var x int
	switch {
	case a.X+menuW <= screenW:
		x = a.X
	case a.X+a.W-menuW >= 0:
		x = a.X + a.W - menuW
	default:
		x = screenW - menuW
		if x < 0 {
			x = 0
		}
	}

	m.bounds = Rect{x, y, menuW, menuH}
	m.itemRects = m.itemRects[:0]
	for i := range m.cfg.Items {
		m.itemRects = append(m.itemRects, Rect{x, y + i*itemH, menuW, itemH})
	}
}

// Draw lays out and renders the menu. Call it before Update each frame — the
// same convention as Dialog.
func (m *Menu) Draw(r Renderer, screenW, screenH int, theme Theme) {
	m.layout(r, screenW, screenH)
	if m.status != Active {
		return
	}
	lh := r.LineHeight(dlgScale)
	padX := lh / 2

	r.FillRect(m.bounds, theme.Panel)
	r.StrokeRect(m.bounds, theme.Border, 1)

	highlighted := m.hover
	if highlighted < 0 {
		highlighted = m.selected
	}

	for i, it := range m.cfg.Items {
		rec := m.itemRects[i]
		if i == highlighted && m.itemEnabled(i) {
			r.FillRect(rec, theme.SelFill)
		}
		col := theme.Text
		if it.Disabled {
			col = theme.Disabled
		}
		r.DrawText(it.Label, rec.X+padX, rec.Y+(rec.H-lh)/2, dlgScale, col)
	}
}

// Update advances the menu's state from an input snapshot, hit-testing
// against the bounds cached by the last Draw. Escape or a click outside the
// menu cancels; Enter or a click on an enabled item accepts (Result holds its
// index); Up/Down move the keyboard-hover, skipping disabled items.
func (m *Menu) Update(in Input) Status {
	if m.status != Active {
		return m.status
	}

	if in.pressed(KeyEscape) {
		m.status = Cancelled
		return m.status
	}
	if in.pressed(KeyUp) {
		m.moveSelected(-1)
	}
	if in.pressed(KeyDown) {
		m.moveSelected(+1)
	}
	if in.pressed(KeyEnter) {
		if m.itemEnabled(m.selected) {
			m.result = m.selected
			m.status = Accepted
		}
		return m.status
	}

	m.hover = -1
	for i, rec := range m.itemRects {
		if rec.Contains(in.MouseX, in.MouseY) {
			m.hover = i
			break
		}
	}

	if in.MousePressed {
		if m.bounds.Contains(in.MouseX, in.MouseY) {
			if m.itemEnabled(m.hover) {
				m.result = m.hover
				m.status = Accepted
			}
			// Clicked inside the menu but on a disabled item or the border:
			// stays Active.
		} else {
			m.status = Cancelled
		}
	}
	return m.status
}

// moveSelected shifts the keyboard-selected item by delta, wrapping, and
// skips disabled items. It is a no-op if every item is disabled.
func (m *Menu) moveSelected(delta int) {
	n := len(m.cfg.Items)
	if n == 0 {
		return
	}
	if m.selected < 0 {
		if delta > 0 {
			m.selected = 0
		} else {
			m.selected = n - 1
		}
		if m.itemEnabled(m.selected) {
			return
		}
	}
	next := m.selected
	for i := 0; i < n; i++ {
		next = (next + delta + n) % n
		if m.itemEnabled(next) {
			m.selected = next
			return
		}
	}
}
