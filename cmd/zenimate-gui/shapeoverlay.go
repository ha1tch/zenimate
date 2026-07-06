package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guidraw"
	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guiutil"
	"github.com/ha1tch/zenimate/internal/ui"
)

// drawShapePreview renders a live preview of the shape tool currently being
// dragged — the exact same outline walk that will eventually commit into
// the sprite on release, drawn to screen instead. A no-op when nothing is
// being dragged. Clipped to the grid viewport, matching every other canvas
// overlay in this file.
func drawShapePreview(c *ui.Controller, l guidraw.Layout, ox, oy, cell float32, dragging bool, startX, startY, endX, endY int, line, rect, triangle, ellipse, polygon bool, polygonSides int) {
	if !dragging {
		return
	}

	rl.BeginScissorMode(int32(l.GridX), int32(l.GridY), int32(l.GridW), int32(l.GridH))
	defer rl.EndScissorMode()

	ink := previewInkColour(c)
	plot := func(x, y int) {
		rl.DrawRectangleRec(rl.NewRectangle(
			ox+float32(x)*cell, oy+float32(y)*cell, cell+0.5, cell+0.5,
		), ink)
	}
	switch {
	case line:
		guiutil.ForEachLinePixel(startX, startY, endX, endY, plot)
	case rect:
		guiutil.RectOutline(startX, startY, endX, endY, plot)
	case triangle:
		guiutil.TriangleOutline(startX, startY, endX, endY, plot)
	case ellipse:
		altHeld := rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)
		ex0, ey0, ex1, ey1 := guiutil.CenterOrCornerBounds(startX, startY, endX, endY, !altHeld)
		guiutil.EllipseOutline(ex0, ey0, ex1, ey1, plot)
	case polygon:
		guiutil.PolygonOutline(polygonSides, startX, startY, endX, endY, plot)
	}
}
