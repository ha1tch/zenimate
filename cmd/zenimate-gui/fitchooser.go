package main

import (
	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

// fitChooser asks how an imported image (JPEG/PNG/GIF) should be brought to
// the 256x192 screen before reduction to Spectrum colours. It carries the
// pending image bytes and the display name so the chosen strategy can be
// applied immediately on pick. Wraps a zenui.OptionPanel for layout,
// hit-testing and drawing.
type fitChooser struct {
	ctrl    *ui.Controller
	data    []byte
	name    string
	options []fitOption
	panel   *zenui.OptionPanel
}

type fitOption struct {
	label string
	mode  model.FitMode
}

type fitResult struct {
	state zenui.Status
	mode  model.FitMode
}

func newFitChooser(c *ui.Controller, data []byte, name string) *fitChooser {
	options := []fitOption{
		{"Best fit (keep aspect, letterbox)", model.FitBestFit},
		{"Stretch (fill, ignore aspect)", model.FitStretch},
		{"Centre (no scale, crop/pad)", model.FitCentre},
	}
	items := make([]zenui.Item, len(options))
	for i, o := range options {
		items[i] = zenui.Item{Label: upper(o.label)}
	}
	return &fitChooser{
		ctrl:    c,
		data:    data,
		name:    name,
		options: options,
		panel:   zenui.NewOptionPanel(zenui.OptionPanelConfig{Title: "IMPORT IMAGE - FIT", Options: items}),
	}
}

func (e *fitChooser) update(in zenui.Input) fitResult {
	switch e.panel.Update(in) {
	case zenui.Accepted:
		return fitResult{state: zenui.Accepted, mode: e.options[e.panel.Result()].mode}
	case zenui.Cancelled:
		return fitResult{state: zenui.Cancelled}
	default:
		return fitResult{state: zenui.Active}
	}
}

func (e *fitChooser) draw(r fpRenderer, screenW, screenH int) {
	e.panel.Draw(zenui.Renderer(r), screenW, screenH, fpTheme())
}
