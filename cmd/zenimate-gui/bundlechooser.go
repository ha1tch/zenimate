package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/filepick"
)

// bundleMode is the operation to perform on a .zbun target.
type bundleMode int

const (
	bundleCreate bundleMode = iota // start a fresh bundle (overwrites any existing file)
	bundleAdd                      // add to the existing bundle at the path
)

// bundleChooser asks whether saving into a .zbun should create a new bundle or
// add to an existing one. It only offers "add" when the target file already
// exists. It carries the target path and the entry name/label so the caller can
// perform the write once a mode is chosen.
type bundleChooser struct {
	ctrl    *ui.Controller
	path    string
	name    string
	label   string
	exists  bool
	options []bundleModeOption
	rects   []filepick.Rect
	cancel  filepick.Rect
	panel   filepick.Rect
}

type bundleModeOption struct {
	label string
	mode  bundleMode
}

type bundleResult struct {
	state chooserState // reuses chooserOpen/Picked/Cancelled
	mode  bundleMode
}

func newBundleChooser(c *ui.Controller, path, name, label string, exists bool) *bundleChooser {
	bc := &bundleChooser{ctrl: c, path: path, name: name, label: label, exists: exists}
	if exists {
		bc.options = []bundleModeOption{
			{"Add to existing bundle", bundleAdd},
			{"Replace with new bundle", bundleCreate},
		}
	} else {
		bc.options = []bundleModeOption{
			{"Create new bundle", bundleCreate},
		}
	}
	return bc
}

func (e *bundleChooser) layout(r filepick.Renderer, screenW, screenH int) {
	lh := r.LineHeight(2)
	pad := lh
	rowH := lh + 12
	gap := 6

	n := len(e.options)
	bw := 420
	// Room for the title, a subtitle line (entry name), the options, and Cancel.
	innerH := lh + 8 + lh + 8 + n*(rowH+gap) + 8 + rowH
	pw := bw + 2*pad
	ph := innerH + 2*pad
	px := (screenW - pw) / 2
	py := (screenH - ph) / 2
	e.panel = filepick.Rect{X: px, Y: py, W: pw, H: ph}

	x := px + pad
	y := py + pad + lh + 8 + lh + 8 // below title + subtitle
	e.rects = e.rects[:0]
	for range e.options {
		e.rects = append(e.rects, filepick.Rect{X: x, Y: y, W: bw, H: rowH})
		y += rowH + gap
	}
	e.cancel = filepick.Rect{X: x, Y: y + 8, W: bw, H: rowH}
}

func (e *bundleChooser) update(in filepick.Input) bundleResult {
	for _, k := range in.Keys {
		if k == filepick.KeyEscape {
			return bundleResult{state: chooserCancelled}
		}
	}
	if in.MousePressed {
		for i, rc := range e.rects {
			if rc.Contains(in.MouseX, in.MouseY) {
				return bundleResult{state: chooserPicked, mode: e.options[i].mode}
			}
		}
		if e.cancel.Contains(in.MouseX, in.MouseY) {
			return bundleResult{state: chooserCancelled}
		}
		if !e.panel.Contains(in.MouseX, in.MouseY) {
			return bundleResult{state: chooserCancelled}
		}
	}
	return bundleResult{state: chooserOpen}
}

func (e *bundleChooser) draw(r fpRenderer, screenW, screenH int) {
	e.layout(filepick.Renderer(r), screenW, screenH)
	th := fpTheme()
	lh := r.LineHeight(2)

	r.FillRect(filepick.Rect{X: 0, Y: 0, W: screenW, H: screenH}, th.Backdrop)
	r.FillRect(e.panel, th.Panel)
	r.StrokeRect(e.panel, th.Border, 1)

	mx := int(rl.GetMouseX())
	my := int(rl.GetMouseY())

	title := "NEW BUNDLE"
	if e.exists {
		title = "BUNDLE EXISTS"
	}
	r.DrawText(title, e.panel.X+lh, e.panel.Y+lh, 2, th.Text)
	// Subtitle: which animation is being added, at the small font.
	r.DrawText(upper("add \""+e.name+"\" to this bundle"),
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
