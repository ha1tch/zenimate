package filepick

import "strings"

// Theme controls the dialog's colours. The zero Theme is usable but drab; hosts
// usually set their own. DefaultTheme returns a reasonable dark scheme.
type Theme struct {
	Backdrop   Colour // dim layer over the rest of the screen
	Panel      Colour // dialog background
	Sidebar    Colour // favourites sidebar background
	SideText   Colour // favourites label
	Border     Colour
	Text       Colour
	DimText    Colour
	DirText    Colour // directory entries
	SelFill    Colour // selected row background
	Field      Colour // filename field background
	Button     Colour
	ButtonHot  Colour
	ButtonText Colour
	Disabled   Colour
}

// DefaultTheme is a dark scheme that fits most editors.
func DefaultTheme() Theme {
	return Theme{
		Backdrop:   Colour{0, 0, 0, 0xa0},
		Panel:      Colour{0x20, 0x20, 0x2a, 0xff},
		Sidebar:    Colour{0x18, 0x18, 0x20, 0xff},
		SideText:   Colour{0xc0, 0xc0, 0xd0, 0xff},
		Border:     Colour{0x6a, 0x6a, 0x78, 0xff},
		Text:       Colour{0xe0, 0xe0, 0xe8, 0xff},
		DimText:    Colour{0x90, 0x90, 0xa0, 0xff},
		DirText:    Colour{0xff, 0xd0, 0x60, 0xff},
		SelFill:    Colour{0x30, 0x50, 0x80, 0xff},
		Field:      Colour{0x14, 0x14, 0x1c, 0xff},
		Button:     Colour{0x30, 0x30, 0x40, 0xff},
		ButtonHot:  Colour{0x44, 0x44, 0x58, 0xff},
		ButtonText: Colour{0xf0, 0xf0, 0xf8, 0xff},
		Disabled:   Colour{0x50, 0x50, 0x5a, 0xff},
	}
}

// scale is the integer text scale the dialog renders at.
const dlgScale = 2

// layout computes the dialog's sub-rectangles for a given outer screen size,
// centring a fixed-ish panel. A favourites sidebar runs down the left; the path
// bar, file list, filename field and buttons occupy the area to its right.
func (d *Dialog) layout(r Renderer, screenW, screenH int) {
	lh := r.LineHeight(dlgScale)
	pad := lh

	pw := screenW * 3 / 4
	if pw > 760 {
		pw = 760
	}
	if pw < 420 {
		pw = 420
	}
	ph := screenH * 3 / 4
	if ph > 560 {
		ph = 560
	}
	if ph < 300 {
		ph = 300
	}
	px := (screenW - pw) / 2
	py := (screenH - ph) / 2
	d.bounds = Rect{px, py, pw, ph}

	titleY := py + pad
	btnH := lh + 10
	rowY := py + ph - pad - btnH // button row top
	fieldH := lh + 8
	fieldY := rowY - 8 - fieldH // filename field

	midTop := titleY + lh + 8
	midBottom := fieldY - 8

	// Favourites sidebar on the left.
	sideW := pw / 4
	if sideW < 110 {
		sideW = 110
	}
	if sideW > 200 {
		sideW = 200
	}
	d.sideRect = Rect{px + pad, midTop, sideW, midBottom - midTop}
	d.placeRects = d.placeRects[:0]
	prowH := lh + 6
	for i := range d.places {
		py2 := d.sideRect.Y + 4 + i*prowH
		if py2+prowH > d.sideRect.Y+d.sideRect.H {
			break
		}
		d.placeRects = append(d.placeRects, Rect{d.sideRect.X + 2, py2, d.sideRect.W - 4, prowH})
	}

	// Content area to the right of the sidebar.
	contentX := d.sideRect.X + sideW + pad
	contentW := px + pw - pad - contentX

	pathY := midTop
	d.upRect = Rect{contentX, pathY - 2, lh + 6, lh + 4}

	listY := pathY + lh + 8
	listH := midBottom - listY
	// While browsing inside a container, reserve a preview pane on the right of
	// the content area for the selected entry's thumbnail and metadata.
	if d.inContainer() {
		previewW := contentW * 2 / 5
		if previewW < 120 {
			previewW = 120
		}
		listW := contentW - previewW - pad
		d.listRect = Rect{contentX, listY, listW, listH}
		d.previewRect = Rect{contentX + listW + pad, listY, previewW, listH}
	} else {
		d.listRect = Rect{contentX, listY, contentW, listH}
		d.previewRect = Rect{}
	}
	d.rowH = lh + 4

	d.nameRect = Rect{contentX, fieldY, contentW, fieldH}

	bw := r.TextWidth("CANCEL", dlgScale) + 2*pad
	d.okRect = Rect{px + pw - pad - bw, rowY, bw, btnH}
	d.cancRect = Rect{d.okRect.X - 12 - bw, rowY, bw, btnH}
}

// visibleRows is how many entries fit in the list area.
func (d *Dialog) visibleRows() int {
	if d.rowH <= 0 {
		return 0
	}
	n := d.listRect.H / d.rowH
	if n < 0 {
		return 0
	}
	return n
}

// Update advances the dialog by one frame of input. It must be called after at
// least one Draw so the layout rects exist; New performs no layout, so on the
// very first frame call Draw first (the typical loop draws every frame anyway).
// Returns the resulting Status.
func (d *Dialog) Update(in Input) Status {
	if d.status != Active {
		return d.status
	}

	// Keyboard: navigation and the filename field.
	if in.pressed(KeyEscape) {
		d.status = Cancelled
		return d.status
	}
	if in.pressed(KeyEnter) {
		if d.cfg.Mode == ModeOpen && d.sel >= 0 && d.sel < len(d.entries) && d.entries[d.sel].IsDir {
			d.openSelection()
		} else if d.canAccept() {
			d.accept()
		}
		return d.status
	}
	if in.pressed(KeyUp) {
		d.moveSel(-1)
	}
	if in.pressed(KeyDown) {
		d.moveSel(+1)
	}
	if in.pressed(KeyPageUp) {
		d.moveSel(-d.visibleRows())
	}
	if in.pressed(KeyPageDown) {
		d.moveSel(+d.visibleRows())
	}

	// Filename text entry (Save mode only).
	if d.cfg.Mode == ModeSave {
		if in.pressed(KeyBackspace) && d.name != "" {
			d.name = d.name[:len(d.name)-1]
		}
		for _, r := range in.Chars {
			if r >= 0x20 && r != 0x7f {
				d.name += string(r)
			}
		}
	}

	// Wheel scrolling over the list.
	if in.WheelY != 0 && d.listRect.Contains(in.MouseX, in.MouseY) {
		d.scroll -= int(in.WheelY)
		d.clampScroll()
	}

	// Mouse.
	if in.MousePressed {
		switch {
		case d.clickPlace(in.MouseX, in.MouseY):
			// handled: navigated to a favourite
		case d.upRect.Contains(in.MouseX, in.MouseY):
			d.goUp()
		case d.okRect.Contains(in.MouseX, in.MouseY):
			if d.canAccept() {
				d.accept()
			}
		case d.cancRect.Contains(in.MouseX, in.MouseY):
			d.status = Cancelled
		case d.listRect.Contains(in.MouseX, in.MouseY):
			row := (in.MouseY - d.listRect.Y) / d.rowH
			idx := d.scroll + row
			if idx >= 0 && idx < len(d.entries) {
				if idx == d.sel {
					d.openSelection() // second click on the same row = open/enter
				} else {
					d.sel = idx
					if d.cfg.Mode == ModeSave && !d.entries[idx].IsDir {
						d.name = d.entries[idx].Name
					}
				}
			}
		case !d.bounds.Contains(in.MouseX, in.MouseY):
			d.status = Cancelled // click outside dismisses
		}
	}

	return d.status
}

// clickPlace navigates to a favourite if (x,y) hits one of the sidebar rows.
// Returns true if a place was clicked.
func (d *Dialog) clickPlace(x, y int) bool {
	for i, pr := range d.placeRects {
		if pr.Contains(x, y) && i < len(d.places) {
			d.setDir(d.places[i].Path)
			return true
		}
	}
	return false
}

func (d *Dialog) moveSel(delta int) {
	if len(d.entries) == 0 {
		return
	}
	if d.sel < 0 {
		d.sel = 0
	} else {
		d.sel += delta
	}
	if d.sel < 0 {
		d.sel = 0
	}
	if d.sel >= len(d.entries) {
		d.sel = len(d.entries) - 1
	}
	// Keep the selection visible.
	if d.sel < d.scroll {
		d.scroll = d.sel
	}
	if vr := d.visibleRows(); vr > 0 && d.sel >= d.scroll+vr {
		d.scroll = d.sel - vr + 1
	}
	if d.cfg.Mode == ModeSave && !d.entries[d.sel].IsDir {
		d.name = d.entries[d.sel].Name
	}
}

func (d *Dialog) clampScroll() {
	max := len(d.entries) - d.visibleRows()
	if max < 0 {
		max = 0
	}
	if d.scroll > max {
		d.scroll = max
	}
	if d.scroll < 0 {
		d.scroll = 0
	}
}

// Draw renders the dialog. screenW/screenH are the host surface dimensions so
// the dialog can centre itself and dim the backdrop. theme controls colours.
func (d *Dialog) Draw(r Renderer, screenW, screenH int, theme Theme) {
	d.layout(r, screenW, screenH)

	// Backdrop over the whole screen.
	r.FillRect(Rect{0, 0, screenW, screenH}, theme.Backdrop)

	b := d.bounds
	r.FillRect(b, theme.Panel)
	r.StrokeRect(b, theme.Border, 1)

	lh := r.LineHeight(dlgScale)
	pad := lh
	innerX := b.X + pad

	// Title.
	r.DrawText(strings.ToUpper(d.cfg.Title), innerX, b.Y+pad, dlgScale, theme.Text)

	// Up button and current path.
	r.FillRect(d.upRect, theme.Button)
	r.StrokeRect(d.upRect, theme.Border, 1)
	upGlyph := ".."
	r.DrawText(upGlyph, d.upRect.X+(d.upRect.W-r.TextWidth(upGlyph, dlgScale))/2,
		d.upRect.Y+(d.upRect.H-lh)/2, dlgScale, theme.Text)
	pathX := d.upRect.X + d.upRect.W + 8
	d.drawClipped(r, d.dir, pathX, d.upRect.Y+(d.upRect.H-lh)/2, b.X+b.W-pad-pathX, theme.DimText)

	// Favourites sidebar.
	r.FillRect(d.sideRect, theme.Sidebar)
	r.Clip(d.sideRect)
	for i, pr := range d.placeRects {
		if i >= len(d.places) {
			break
		}
		if d.places[i].Path == d.dir {
			r.FillRect(pr, theme.SelFill)
		}
		r.DrawText(d.places[i].Label, pr.X+6, pr.Y+(pr.H-lh)/2, dlgScale, theme.SideText)
	}
	r.ClipEnd()
	r.StrokeRect(d.sideRect, theme.Border, 1)

	// File list.
	r.FillRect(d.listRect, theme.Field)
	r.Clip(d.listRect)
	vr := d.visibleRows()
	for i := 0; i < vr; i++ {
		idx := d.scroll + i
		if idx >= len(d.entries) {
			break
		}
		e := d.entries[idx]
		rowRect := Rect{d.listRect.X, d.listRect.Y + i*d.rowH, d.listRect.W, d.rowH}
		if idx == d.sel {
			r.FillRect(rowRect, theme.SelFill)
		}
		col := theme.Text
		label := e.Name
		if e.IsDir {
			col = theme.DirText
			label = label + "/"
		} else if e.IsContainer {
			col = theme.DirText
			label = label + "  [bundle]"
		}
		r.DrawText(label, rowRect.X+6, rowRect.Y+(d.rowH-lh)/2, dlgScale, col)
	}
	r.ClipEnd()
	r.StrokeRect(d.listRect, theme.Border, 1)

	// Preview pane (only while browsing inside a container).
	if d.inContainer() && d.previewRect.W > 0 {
		d.drawPreview(r, theme)
	}

	// Filename field (Save) or selected-name echo (Open).
	r.FillRect(d.nameRect, theme.Field)
	r.StrokeRect(d.nameRect, theme.Border, 1)
	shown := d.name
	if d.cfg.Mode == ModeOpen {
		if d.sel >= 0 && d.sel < len(d.entries) && !d.entries[d.sel].IsDir {
			shown = d.entries[d.sel].Name
		} else {
			shown = ""
		}
	}
	caret := ""
	if d.cfg.Mode == ModeSave {
		caret = "_"
	}
	d.drawClipped(r, shown+caret, d.nameRect.X+6, d.nameRect.Y+(d.nameRect.H-lh)/2,
		d.nameRect.W-12, theme.Text)

	// Buttons.
	d.drawButton(r, d.cancRect, "CANCEL", true, theme)
	okLabel := "OPEN"
	if d.cfg.Mode == ModeSave {
		okLabel = "SAVE"
	}
	d.drawButton(r, d.okRect, okLabel, d.canAccept(), theme)

	// Error line, if any, just under the title.
	if d.err != "" {
		r.DrawText("! "+d.err, innerX, b.Y+pad+lh+2, 1, theme.DimText)
	}
}

func (d *Dialog) drawButton(r Renderer, rec Rect, label string, enabled bool, theme Theme) {
	r.FillRect(rec, theme.Button)
	r.StrokeRect(rec, theme.Border, 1)
	col := theme.ButtonText
	if !enabled {
		col = theme.Disabled
	}
	lh := r.LineHeight(dlgScale)
	r.DrawText(label, rec.X+(rec.W-r.TextWidth(label, dlgScale))/2,
		rec.Y+(rec.H-lh)/2, dlgScale, col)
}

// drawClipped draws s at (x,y) clipped to maxW pixels (no ellipsis; the host's
// Clip handles overflow). It clips to a 1-line-high band.
func (d *Dialog) drawClipped(r Renderer, s string, x, y, maxW int, c Colour) {
	if maxW <= 0 {
		return
	}
	r.Clip(Rect{x, y - 2, maxW, r.LineHeight(dlgScale) + 4})
	r.DrawText(s, x, y, dlgScale, c)
	r.ClipEnd()
}

// drawPreview renders the preview pane for the selected in-container entry: an
// optional thumbnail (from Config.Preview) above its metadata lines.
func (d *Dialog) drawPreview(r Renderer, theme Theme) {
	pr := d.previewRect
	r.FillRect(pr, theme.Field)
	r.StrokeRect(pr, theme.Border, 1)

	if d.sel < 0 || d.sel >= len(d.entries) {
		lh := r.LineHeight(1)
		r.DrawText("select an entry", pr.X+6, pr.Y+6, 1, theme.DimText)
		_ = lh
		return
	}
	e := d.entries[d.sel]
	pad := 8
	x := pr.X + pad
	y := pr.Y + pad

	// Thumbnail, if the host supplies one.
	if d.cfg.Preview != nil {
		if pv := d.cfg.Preview(d.container, e); pv != nil && pv.W > 0 && pv.H > 0 && len(pv.Pixels) >= pv.W*pv.H {
			// Scale to fit the pane width (and a capped height), integer zoom >= 1.
			maxW := pr.W - 2*pad
			maxH := pr.H/2 - pad
			zoom := maxW / pv.W
			if zh := maxH / pv.H; zh < zoom {
				zoom = zh
			}
			if zoom < 1 {
				zoom = 1
			}
			r.Clip(pr)
			for py := 0; py < pv.H; py++ {
				for px := 0; px < pv.W; px++ {
					c := pv.Pixels[py*pv.W+px]
					if c.A == 0 {
						continue
					}
					r.FillRect(Rect{x + px*zoom, y + py*zoom, zoom, zoom}, c)
				}
			}
			r.ClipEnd()
			y += pv.H*zoom + pad
		}
	}

	// Name + metadata lines.
	lh := r.LineHeight(1)
	r.Clip(pr)
	r.DrawText(e.Name, x, y, 1, theme.Text)
	y += lh + 4
	for _, m := range e.Meta {
		r.DrawText(m.Key+": "+m.Value, x, y, 1, theme.DimText)
		y += lh + 2
	}
	r.ClipEnd()
}
