package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guidraw"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

// antDash is the length, in screen pixels, of one black or white segment of
// the marching-ants border — a fixed screen size regardless of zoom, so the
// animation reads at the same speed and density whatever the sprite's
// on-screen scale.
const antDash = 4

// antSpeed is how many screen pixels the dash pattern advances per second.
const antSpeed = 24

// drawSelectionOverlay renders the active selection, if any: a live preview
// of any floating (not yet committed) content on top of the canvas, and a
// marching-ants dashed border around the selection bounds. Drawn after the
// main canvas render, since floating content is deliberately not written
// into the frame data until commit — the normal grid render has no way to
// show it. Clipped to the same grid viewport as the main canvas render, so
// a selection near the edge at high zoom can't draw over the surrounding UI.
func drawSelectionOverlay(c *ui.Controller, l guidraw.Layout, ox, oy, cell float32) {
	x, y, w, h, ok := c.Selection()
	if !ok {
		return
	}

	rl.BeginScissorMode(int32(l.GridX), int32(l.GridY), int32(l.GridW), int32(l.GridH))
	defer rl.EndScissorMode()

	// Floating content preview: only pixels the lifted buffer actually has
	// set are drawn, in the current ink colour — matching how a plain paint
	// stroke looks, since floating content carries only bitmap state, never
	// attributes (see selection.go's doc comment for why).
	if c.IsFloating() {
		ink := previewInkColour(c)
		for ly := 0; ly < h; ly++ {
			for lx := 0; lx < w; lx++ {
				if !c.FloatingAt(lx, ly) {
					continue
				}
				px, py := x+lx, y+ly
				rl.DrawRectangleRec(rl.NewRectangle(
					ox+float32(px)*cell, oy+float32(py)*cell, cell+0.5, cell+0.5,
				), ink)
			}
		}
	}

	rectX := ox + float32(x)*cell
	rectY := oy + float32(y)*cell
	rectW := float32(w) * cell
	rectH := float32(h) * cell
	phase := float32(rl.GetTime()) * antSpeed
	drawMarchingAnts(rectX, rectY, rectW, rectH, phase)
}

// previewInkColour picks the colour the floating-content preview should
// show "on" pixels in, matching what committing will actually look like:
// the sprite's real ink colour in Spectrum Colour mode, white in Bitmap
// White, black in Bitmap Black — the same per-mode rule Theme.PixelColour
// already applies to every other pixel on the canvas.
func previewInkColour(c *ui.Controller) rl.Color {
	switch c.Mode() {
	case ui.SpectrumColour:
		idx := zxpalette.Index(c.Ink(), c.Bright())
		return guidraw.ZxColor(zxpalette.RGBA[idx])
	case ui.BitmapWhite:
		return rl.White
	default: // BitmapBlack
		return rl.Black
	}
}

// drawMarchingAnts draws a rectangle's outline as alternating black/white
// dashes, offset by phase — the classic high-contrast "marching ants"
// selection border, animated by advancing phase every frame rather than
// drawn with real gaps (which would just show whatever is underneath).
func drawMarchingAnts(x, y, w, h, phase float32) {
	// Walk the perimeter as one continuous strip so the dash pattern is
	// continuous around corners, not restarted on each edge.
	type seg struct{ x0, y0, x1, y1 float32 }
	segs := []seg{
		{x, y, x + w, y},         // top
		{x + w, y, x + w, y + h}, // right
		{x + w, y + h, x, y + h}, // bottom
		{x, y + h, x, y},         // left
	}
	dist := float32(0)
	for _, s := range segs {
		length := abs32(s.x1-s.x0) + abs32(s.y1-s.y0) // manhattan: each seg is axis-aligned
		if length <= 0 {
			continue
		}
		dx := (s.x1 - s.x0) / length
		dy := (s.y1 - s.y0) / length
		for t := float32(0); t < length; t += 1 {
			on := int(dist+phase)/antDash%2 == 0
			if on {
				px := s.x0 + dx*t
				py := s.y0 + dy*t
				rl.DrawRectangleRec(rl.NewRectangle(px, py, 1.5, 1.5), rl.White)
			}
			dist++
		}
	}
}
