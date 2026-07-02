package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/filepick"
)

// saveProvChooser appears when saving a sprite that was opened from a bundle. It
// asks whether to update the animation inside its source bundle or to save it as
// a separate standalone .zani. The decision is remembered on the source so this
// is asked at most once per opened sprite.
type saveProvChooser struct {
	ctrl    *ui.Controller
	src     ui.SpriteSource
	options []saveProvOption
	rects   []filepick.Rect
	cancel  filepick.Rect
	panel   filepick.Rect
}

type saveProvOption struct {
	label    string
	toBundle bool
}

type saveProvResult struct {
	state    chooserState
	toBundle bool
}

func newSaveProvChooser(c *ui.Controller, src ui.SpriteSource) *saveProvChooser {
	return &saveProvChooser{
		ctrl: c,
		src:  src,
		options: []saveProvOption{
			{"Update in bundle (" + baseName(src.Path) + ")", true},
			{"Save as separate .zani", false},
		},
	}
}

func (e *saveProvChooser) layout(r filepick.Renderer, screenW, screenH int) {
	lh := r.LineHeight(2)
	pad := lh
	rowH := lh + 12
	gap := 6

	n := len(e.options)
	bw := 460
	innerH := lh + 8 + lh + 8 + n*(rowH+gap) + 8 + rowH
	pw := bw + 2*pad
	ph := innerH + 2*pad
	px := (screenW - pw) / 2
	py := (screenH - ph) / 2
	e.panel = filepick.Rect{X: px, Y: py, W: pw, H: ph}

	x := px + pad
	y := py + pad + lh + 8 + lh + 8
	e.rects = e.rects[:0]
	for range e.options {
		e.rects = append(e.rects, filepick.Rect{X: x, Y: y, W: bw, H: rowH})
		y += rowH + gap
	}
	e.cancel = filepick.Rect{X: x, Y: y + 8, W: bw, H: rowH}
}

func (e *saveProvChooser) update(in filepick.Input) saveProvResult {
	for _, k := range in.Keys {
		if k == filepick.KeyEscape {
			return saveProvResult{state: chooserCancelled}
		}
	}
	if in.MousePressed {
		for i, rc := range e.rects {
			if rc.Contains(in.MouseX, in.MouseY) {
				return saveProvResult{state: chooserPicked, toBundle: e.options[i].toBundle}
			}
		}
		if e.cancel.Contains(in.MouseX, in.MouseY) {
			return saveProvResult{state: chooserCancelled}
		}
		if !e.panel.Contains(in.MouseX, in.MouseY) {
			return saveProvResult{state: chooserCancelled}
		}
	}
	return saveProvResult{state: chooserOpen}
}

func (e *saveProvChooser) draw(r fpRenderer, screenW, screenH int) {
	e.layout(filepick.Renderer(r), screenW, screenH)
	th := fpTheme()
	lh := r.LineHeight(2)

	r.FillRect(filepick.Rect{X: 0, Y: 0, W: screenW, H: screenH}, th.Backdrop)
	r.FillRect(e.panel, th.Panel)
	r.StrokeRect(e.panel, th.Border, 1)

	mx := int(rl.GetMouseX())
	my := int(rl.GetMouseY())

	r.DrawText("SAVE ANIMATION", e.panel.X+lh, e.panel.Y+lh, 2, th.Text)
	r.DrawText(upper("\""+e.src.Entry+"\" came from a bundle"),
		e.panel.X+lh, e.panel.Y+lh+lh+8, 1, th.DimText)

	for i, rc := range e.rects {
		bg := th.Button
		if rc.Contains(mx, my) {
			bg = th.ButtonHot
		}
		r.FillRect(rc, bg)
		r.StrokeRect(rc, th.Border, 1)
		r.DrawText(upper(e.options[i].label), rc.X+8, rc.Y+(rc.H-r.LineHeight(1))/2, 1, th.ButtonText)
	}

	cbg := th.Button
	if e.cancel.Contains(mx, my) {
		cbg = th.ButtonHot
	}
	r.FillRect(e.cancel, cbg)
	r.StrokeRect(e.cancel, th.Border, 1)
	clab := "CANCEL"
	r.DrawText(clab, e.cancel.X+(e.cancel.W-r.TextWidth(clab, 1))/2,
		e.cancel.Y+(e.cancel.H-r.LineHeight(1))/2, 1, th.ButtonText)
}
