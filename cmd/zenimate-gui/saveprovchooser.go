package main

import (
	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guiutil"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

// saveProvChooser appears when saving a sprite that was opened from a bundle.
// It asks whether to update the animation inside its source bundle or to save
// it as a separate standalone .zani. The decision is remembered on the source
// so this is asked at most once per opened sprite. Wraps a zenui.OptionPanel
// for layout, hit-testing and drawing.
type saveProvChooser struct {
	ctrl    *ui.Controller
	src     ui.SpriteSource
	options []saveProvOption
	panel   *zenui.OptionPanel
}

type saveProvOption struct {
	label    string
	toBundle bool
}

type saveProvResult struct {
	state    zenui.Status
	toBundle bool
}

func newSaveProvChooser(c *ui.Controller, src ui.SpriteSource) *saveProvChooser {
	options := []saveProvOption{
		{"Update in bundle (" + baseName(src.Path) + ")", true},
		{"Save as separate .zani", false},
	}
	items := make([]zenui.Item, len(options))
	for i, o := range options {
		items[i] = zenui.Item{Label: guiutil.Upper(o.label)}
	}
	return &saveProvChooser{
		ctrl:    c,
		src:     src,
		options: options,
		panel: zenui.NewOptionPanel(zenui.OptionPanelConfig{
			Title:    "SAVE ANIMATION",
			Subtitle: guiutil.Upper("\"" + src.Entry + "\" came from a bundle"),
			Options:  items,
		}),
	}
}

func (e *saveProvChooser) update(in zenui.Input) saveProvResult {
	switch e.panel.Update(in) {
	case zenui.Accepted:
		return saveProvResult{state: zenui.Accepted, toBundle: e.options[e.panel.Result()].toBundle}
	case zenui.Cancelled:
		return saveProvResult{state: zenui.Cancelled}
	default:
		return saveProvResult{state: zenui.Active}
	}
}

func (e *saveProvChooser) draw(r fpRenderer, screenW, screenH int) {
	e.panel.Draw(zenui.Renderer(r), screenW, screenH, fpTheme())
}
