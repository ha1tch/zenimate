package main

import (
	"github.com/ha1tch/zenimate/internal/fonts"
	"github.com/ha1tch/zenimate/pkg/bdf"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

// namedFont pairs a display name with its loaded font, for the text tool's
// font picker menu.
type namedFont struct {
	name string
	font *bdf.Font
}

// loadTextFonts loads every bundled font available to the text tool, in the
// fixed order they appear in the picker menu (Sinclair first, matching its
// role as the default). Panics on load failure, matching the existing
// startup convention for bundled, embedded assets — a failure here means
// the binary itself is broken, not a runtime condition to recover from.
func loadTextFonts() []namedFont {
	loaders := []struct {
		name string
		load func() (*bdf.Font, error)
	}{
		{"Sinclair", fonts.Sinclair},
		{"Cozette", fonts.Cozette},
		{"TomThumb", fonts.TomThumb},
		{"Spleen5x8", fonts.Spleen5x8},
		{"Creep", fonts.Creep},
		{"HaxorMedium", fonts.HaxorMedium},
	}
	result := make([]namedFont, 0, len(loaders))
	for _, l := range loaders {
		f, err := l.load()
		if err != nil {
			panic(err)
		}
		result = append(result, namedFont{name: l.name, font: f})
	}
	return result
}

// newTextFontMenu builds the font picker menu anchored at the given rect,
// marking the currently-selected font so reopening the menu shows what's
// actually active rather than always looking unselected.
func newTextFontMenu(anchor zenui.Rect, list []namedFont, currentIdx int) *zenui.Menu {
	items := make([]zenui.Item, len(list))
	for i, nf := range list {
		label := nf.name
		if i == currentIdx {
			label = "> " + label
		}
		items[i] = zenui.Item{Label: label}
	}
	return zenui.NewMenu(zenui.MenuConfig{Items: items, Anchor: anchor})
}
