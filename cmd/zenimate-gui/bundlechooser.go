package main

import (
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

// bundleMode is the operation to perform on a .zbun target.
type bundleMode int

const (
	bundleCreate bundleMode = iota // start a fresh bundle (overwrites any existing file)
	bundleAdd                      // add to the existing bundle at the path
)

// bundleChooser asks whether saving into a .zbun should create a new bundle or
// add to an existing one. It only offers "add" when the target file already
// exists. It carries the target path and the entry name/label so the caller
// can perform the write once a mode is chosen. Wraps a zenui.OptionPanel for
// layout, hit-testing and drawing.
type bundleChooser struct {
	ctrl    *ui.Controller
	path    string
	name    string
	label   string
	exists  bool
	options []bundleModeOption
	panel   *zenui.OptionPanel
}

type bundleModeOption struct {
	label string
	mode  bundleMode
}

type bundleResult struct {
	state zenui.Status
	mode  bundleMode
}

func newBundleChooser(c *ui.Controller, path, name, label string, exists bool) *bundleChooser {
	var options []bundleModeOption
	if exists {
		options = []bundleModeOption{
			{"Add to existing bundle", bundleAdd},
			{"Replace with new bundle", bundleCreate},
		}
	} else {
		options = []bundleModeOption{
			{"Create new bundle", bundleCreate},
		}
	}
	items := make([]zenui.Item, len(options))
	for i, o := range options {
		items[i] = zenui.Item{Label: upper(o.label)}
	}
	title := "NEW BUNDLE"
	if exists {
		title = "BUNDLE EXISTS"
	}
	return &bundleChooser{
		ctrl:    c,
		path:    path,
		name:    name,
		label:   label,
		exists:  exists,
		options: options,
		panel: zenui.NewOptionPanel(zenui.OptionPanelConfig{
			Title:    title,
			Subtitle: upper("add \"" + name + "\" to this bundle"),
			Options:  items,
		}),
	}
}

func (e *bundleChooser) update(in zenui.Input) bundleResult {
	switch e.panel.Update(in) {
	case zenui.Accepted:
		return bundleResult{state: zenui.Accepted, mode: e.options[e.panel.Result()].mode}
	case zenui.Cancelled:
		return bundleResult{state: zenui.Cancelled}
	default:
		return bundleResult{state: zenui.Active}
	}
}

func (e *bundleChooser) draw(r fpRenderer, screenW, screenH int) {
	e.panel.Draw(zenui.Renderer(r), screenW, screenH, fpTheme())
}
