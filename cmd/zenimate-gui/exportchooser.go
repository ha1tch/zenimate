package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/filepick"
)

// exportChooser is a small modal that lets the user pick an export format before
// the Save dialog opens. It is deliberately minimal: a centred panel with one
// button per format and a Cancel. Input/drawing reuse the same filepick.Input
// snapshot and fpRenderer as the file dialog, so behaviour and look stay
// consistent across both modals.
type exportChooser struct {
	ctrl    *ui.Controller
	options []exportOption
	rects   []filepick.Rect
	cancel  filepick.Rect
	panel   filepick.Rect
}

type exportOption struct {
	label  string
	format model.ExportFormat
}

type chooserState int

const (
	chooserOpen chooserState = iota
	chooserPicked
	chooserCancelled
)

type chooserResult struct {
	state  chooserState
	format model.ExportFormat
	ctrl   *ui.Controller
}

func newExportChooser(c *ui.Controller) *exportChooser {
	return &exportChooser{
		ctrl: c,
		options: []exportOption{
			{"Screen (.scr)", model.FormatSCR},
			{"Tape (.tap)", model.FormatTAP},
			{"Auto-run tape (.tap)", model.FormatTAPLoader},
			{"Tape (.tzx)", model.FormatTZX},
			{"Snapshot (.sna)", model.FormatSNA},
			{"Snapshot (.z80)", model.FormatZ80},
		},
	}
}

// layout positions the panel and its buttons centred on the screen, caching the
// rects for hit-testing. It needs only text metrics, so it takes the
// filepick.Renderer interface (satisfied by fpRenderer, and by a stub in tests).
func (e *exportChooser) layout(r filepick.Renderer, screenW, screenH int) {
	lh := r.LineHeight(2)
	pad := lh
	rowH := lh + 12
	gap := 6

	n := len(e.options)
	bw := 320
	innerH := lh + 8 + n*(rowH+gap) + 8 + rowH // title + options + gap + cancel
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

// update handles input for the chooser, returning its resolved state.
func (e *exportChooser) update(in filepick.Input) chooserResult {
	// Escape cancels.
	for _, k := range in.Keys {
		if k == filepick.KeyEscape {
			return chooserResult{state: chooserCancelled}
		}
	}
	if in.MousePressed {
		for i, rc := range e.rects {
			if rc.Contains(in.MouseX, in.MouseY) {
				return chooserResult{state: chooserPicked, format: e.options[i].format, ctrl: e.ctrl}
			}
		}
		if e.cancel.Contains(in.MouseX, in.MouseY) {
			return chooserResult{state: chooserCancelled}
		}
		if !e.panel.Contains(in.MouseX, in.MouseY) {
			return chooserResult{state: chooserCancelled} // click outside dismisses
		}
	}
	return chooserResult{state: chooserOpen, ctrl: e.ctrl}
}

// draw renders the chooser over a dimmed backdrop.
func (e *exportChooser) draw(r fpRenderer, screenW, screenH int) {
	e.layout(filepick.Renderer(r), screenW, screenH)
	th := fpTheme()
	lh := r.LineHeight(2)

	// Backdrop + panel.
	r.FillRect(filepick.Rect{X: 0, Y: 0, W: screenW, H: screenH}, th.Backdrop)
	r.FillRect(e.panel, th.Panel)
	r.StrokeRect(e.panel, th.Border, 1)

	mx := int(rl.GetMouseX())
	my := int(rl.GetMouseY())

	r.DrawText("EXPORT AS", e.panel.X+lh, e.panel.Y+lh, 2, th.Text)

	for i, rc := range e.rects {
		bg := th.Button
		if rc.Contains(mx, my) {
			bg = th.ButtonHot
		}
		r.FillRect(rc, bg)
		r.StrokeRect(rc, th.Border, 1)
		label := upper(e.options[i].label)
		r.DrawText(label, rc.X+8, rc.Y+(rc.H-r.LineHeight(1))/2, 1, th.ButtonText)
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
