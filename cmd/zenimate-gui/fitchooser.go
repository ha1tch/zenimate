package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/filepick"
)

// fitChooser asks how an imported image (JPEG/PNG/GIF) should be brought to the
// 256x192 screen before reduction to Spectrum colours. It carries the pending
// image bytes and the display name so the chosen strategy can be applied
// immediately on pick.
type fitChooser struct {
	ctrl    *ui.Controller
	data    []byte
	name    string
	options []fitOption
	rects   []filepick.Rect
	cancel  filepick.Rect
	panel   filepick.Rect
}

type fitOption struct {
	label string
	mode  model.FitMode
}

type fitResult struct {
	state chooserState // reuses chooserOpen/Picked/Cancelled
	mode  model.FitMode
}

func newFitChooser(c *ui.Controller, data []byte, name string) *fitChooser {
	return &fitChooser{
		ctrl: c,
		data: data,
		name: name,
		options: []fitOption{
			{"Best fit (keep aspect, letterbox)", model.FitBestFit},
			{"Stretch (fill, ignore aspect)", model.FitStretch},
			{"Centre (no scale, crop/pad)", model.FitCentre},
		},
	}
}

func (e *fitChooser) layout(r filepick.Renderer, screenW, screenH int) {
	lh := r.LineHeight(2)
	pad := lh
	rowH := lh + 12
	gap := 6

	n := len(e.options)
	bw := 440
	innerH := lh + 8 + n*(rowH+gap) + 8 + rowH
	pw := bw + 2*pad
	ph := innerH + 2*pad
	px := (screenW - pw) / 2
	py := (screenH - ph) / 2
	e.panel = filepick.Rect{X: px, Y: py, W: pw, H: ph}

	x := px + pad
	y := py + pad + lh + 8
	e.rects = e.rects[:0]
	for range e.options {
		e.rects = append(e.rects, filepick.Rect{X: x, Y: y, W: bw, H: rowH})
		y += rowH + gap
	}
	e.cancel = filepick.Rect{X: x, Y: y + 8, W: bw, H: rowH}
}

func (e *fitChooser) update(in filepick.Input) fitResult {
	for _, k := range in.Keys {
		if k == filepick.KeyEscape {
			return fitResult{state: chooserCancelled}
		}
	}
	if in.MousePressed {
		for i, rc := range e.rects {
			if rc.Contains(in.MouseX, in.MouseY) {
				return fitResult{state: chooserPicked, mode: e.options[i].mode}
			}
		}
		if e.cancel.Contains(in.MouseX, in.MouseY) {
			return fitResult{state: chooserCancelled}
		}
		if !e.panel.Contains(in.MouseX, in.MouseY) {
			return fitResult{state: chooserCancelled}
		}
	}
	return fitResult{state: chooserOpen}
}

func (e *fitChooser) draw(r fpRenderer, screenW, screenH int) {
	e.layout(filepick.Renderer(r), screenW, screenH)
	th := fpTheme()
	lh := r.LineHeight(2)

	r.FillRect(filepick.Rect{X: 0, Y: 0, W: screenW, H: screenH}, th.Backdrop)
	r.FillRect(e.panel, th.Panel)
	r.StrokeRect(e.panel, th.Border, 1)

	mx := int(rl.GetMouseX())
	my := int(rl.GetMouseY())

	r.DrawText("IMPORT IMAGE - FIT", e.panel.X+lh, e.panel.Y+lh, 2, th.Text)

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
