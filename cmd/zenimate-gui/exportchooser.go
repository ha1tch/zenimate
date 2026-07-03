package main

import (
	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

// exportChooser is a small modal that lets the user pick an export format
// before the Save dialog opens. It wraps a zenui.OptionPanel, which owns
// layout, hit-testing and drawing; this type only maps the chosen index back
// to a model.ExportFormat and carries the controller through to the result.
type exportChooser struct {
	ctrl    *ui.Controller
	options []exportOption
	panel   *zenui.OptionPanel
}

type exportOption struct {
	label  string
	format model.ExportFormat
}

// chooserResult is returned by every chooser wrapper in this file family
// (export, bundle, fit, save-provenance). state reuses zenui's shared Status:
// Active while still open, Accepted once a choice is made (payload set),
// Cancelled otherwise.
type chooserResult struct {
	state  zenui.Status
	format model.ExportFormat
	ctrl   *ui.Controller
}

func newExportChooser(c *ui.Controller) *exportChooser {
	options := []exportOption{
		{"Screen (.scr)", model.FormatSCR},
		{"Tape (.tap)", model.FormatTAP},
		{"Auto-run tape (.tap)", model.FormatTAPLoader},
		{"Tape (.tzx)", model.FormatTZX},
		{"Snapshot (.sna)", model.FormatSNA},
		{"Snapshot (.z80)", model.FormatZ80},
	}
	items := make([]zenui.Item, len(options))
	for i, o := range options {
		items[i] = zenui.Item{Label: upper(o.label)}
	}
	return &exportChooser{
		ctrl:    c,
		options: options,
		panel:   zenui.NewOptionPanel(zenui.OptionPanelConfig{Title: "EXPORT AS", Options: items}),
	}
}

// update advances the chooser and returns its resolved state.
func (e *exportChooser) update(in zenui.Input) chooserResult {
	switch e.panel.Update(in) {
	case zenui.Accepted:
		return chooserResult{state: zenui.Accepted, format: e.options[e.panel.Result()].format, ctrl: e.ctrl}
	case zenui.Cancelled:
		return chooserResult{state: zenui.Cancelled}
	default:
		return chooserResult{state: zenui.Active, ctrl: e.ctrl}
	}
}

// draw renders the chooser over a dimmed backdrop.
func (e *exportChooser) draw(r fpRenderer, screenW, screenH int) {
	e.panel.Draw(zenui.Renderer(r), screenW, screenH, fpTheme())
}
