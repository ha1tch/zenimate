package zenui

// bodyScale is the text scale used for subtitle, option rows, and Cancel —
// one step down from the title, matching every existing instance of this
// modal shape.
const bodyScale = 1

// layout sizes and centres the panel, auto-widening it to fit the title, the
// subtitle (if any), every option label, and "CANCEL" — rather than a fixed
// width chosen per instance.
func (p *OptionPanel) layout(r Renderer, screenW, screenH int) {
	lh := r.LineHeight(dlgScale)
	pad := lh
	rowH := lh + 12
	gap := 6

	const minBW = 200
	bw := minBW
	widen := func(s string, scale int) {
		if w := r.TextWidth(s, scale) + 16; w > bw {
			bw = w
		}
	}
	widen(p.cfg.Title, dlgScale)
	if p.cfg.Subtitle != "" {
		widen(p.cfg.Subtitle, bodyScale)
	}
	for _, it := range p.cfg.Options {
		widen(it.Label, bodyScale)
	}
	widen("CANCEL", bodyScale)

	headH := lh + 8
	if p.cfg.Subtitle != "" {
		headH += lh + 8
	}

	n := len(p.cfg.Options)
	innerH := headH + n*(rowH+gap) + 8 + rowH
	pw := bw + 2*pad
	ph := innerH + 2*pad
	px := (screenW - pw) / 2
	py := (screenH - ph) / 2
	p.panel = Rect{X: px, Y: py, W: pw, H: ph}

	x := px + pad
	y := py + pad + headH
	p.itemRects = p.itemRects[:0]
	for range p.cfg.Options {
		p.itemRects = append(p.itemRects, Rect{X: x, Y: y, W: bw, H: rowH})
		y += rowH + gap
	}
	p.cancelRect = Rect{X: x, Y: y + 8, W: bw, H: rowH}
}

// Draw lays out and renders the panel over a dimmed full-screen backdrop.
// Call it before Update each frame.
func (p *OptionPanel) Draw(r Renderer, screenW, screenH int, theme Theme) {
	p.layout(r, screenW, screenH)
	if p.status != Active {
		return
	}
	lh := r.LineHeight(dlgScale)

	r.FillRect(Rect{X: 0, Y: 0, W: screenW, H: screenH}, theme.Backdrop)
	r.FillRect(p.panel, theme.Panel)
	r.StrokeRect(p.panel, theme.Border, 1)

	r.DrawText(p.cfg.Title, p.panel.X+lh, p.panel.Y+lh, dlgScale, theme.Text)
	if p.cfg.Subtitle != "" {
		r.DrawText(p.cfg.Subtitle, p.panel.X+lh, p.panel.Y+lh+lh+8, bodyScale, theme.DimText)
	}

	for i, it := range p.cfg.Options {
		rec := p.itemRects[i]
		bg := theme.Button
		if i == p.hover && p.itemEnabled(i) {
			bg = theme.ButtonHot
		}
		r.FillRect(rec, bg)
		r.StrokeRect(rec, theme.Border, 1)
		col := theme.ButtonText
		if it.Disabled {
			col = theme.Disabled
		}
		r.DrawText(it.Label, rec.X+8, rec.Y+(rec.H-r.LineHeight(bodyScale))/2, bodyScale, col)
	}

	cbg := theme.Button
	if p.hover == hoverCancel {
		cbg = theme.ButtonHot
	}
	r.FillRect(p.cancelRect, cbg)
	r.StrokeRect(p.cancelRect, theme.Border, 1)
	clab := "CANCEL"
	r.DrawText(clab, p.cancelRect.X+(p.cancelRect.W-r.TextWidth(clab, bodyScale))/2,
		p.cancelRect.Y+(p.cancelRect.H-r.LineHeight(bodyScale))/2, bodyScale, theme.ButtonText)
}

// Update advances the panel's state from an input snapshot, hit-testing
// against the bounds cached by the last Draw. Escape, a click on Cancel, or a
// click outside the panel cancels; a click on an enabled option accepts
// (Result holds its index).
func (p *OptionPanel) Update(in Input) Status {
	if p.status != Active {
		return p.status
	}

	if in.pressed(KeyEscape) {
		p.status = Cancelled
		return p.status
	}

	p.hover = -1
	for i, rec := range p.itemRects {
		if rec.Contains(in.MouseX, in.MouseY) {
			p.hover = i
			break
		}
	}
	if p.hover < 0 && p.cancelRect.Contains(in.MouseX, in.MouseY) {
		p.hover = hoverCancel
	}

	if in.MousePressed {
		switch {
		case p.itemEnabled(p.hover):
			p.result = p.hover
			p.status = Accepted
		case p.hover == hoverCancel:
			p.status = Cancelled
		case !p.panel.Contains(in.MouseX, in.MouseY):
			p.status = Cancelled
		}
	}
	return p.status
}
