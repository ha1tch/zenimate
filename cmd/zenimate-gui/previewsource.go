package main

import (
	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guidraw"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

// spritePreviewSource adapts the current sprite and view state to
// zenui.ImageSource, so PreviewPane can sample it without knowing anything
// about zenimate's Sprite type, ZX attributes, or view modes. Constructed
// once and reused every frame — Width/Height/Region always read live state
// through the Controller and theme, so a resize or a chequer-toggle flip is
// reflected immediately. theme is a pointer to the live guidraw.Theme main
// holds (not a copy), so it always sees the current chequer on/off state.
type spritePreviewSource struct {
	c     *ui.Controller
	theme *guidraw.Theme
}

func (s spritePreviewSource) Width() int  { return s.c.Sprite.Width() }
func (s spritePreviewSource) Height() int { return s.c.Sprite.Height() }

// Region converts a rectangle of sprite pixels to colours in one pass,
// reusing Theme.PixelColour (the same mode-switch logic the main canvas
// uses) and falling back to the chequer pattern for pixels it reports as
// "clear" — Region has no "don't draw" signal, so every pixel needs a
// definite colour, exactly as Theme.CheckerColour already provides for the
// main canvas's own chequer rendering.
func (s spritePreviewSource) Region(x0, y0, w, h int) []zenui.Colour {
	region := make([]zenui.Colour, w*h)
	mode := s.c.Mode()
	chequerOn := s.theme.ChequerVisibleFor(mode)
	for j := 0; j < h; j++ {
		y := y0 + j
		for i := 0; i < w; i++ {
			x := x0 + i
			col, draw := s.theme.PixelColour(s.c, x, y)
			if !draw {
				col = s.theme.CheckerColour(x, y, mode, chequerOn, 1)
			}
			region[j*w+i] = zenui.Colour{R: col.R, G: col.G, B: col.B, A: col.A}
		}
	}
	return region
}
