package main

import (
	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guiutil"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

// penOptions is the small popup panel for choosing the paintbrush's shape
// and size, opened by right-clicking the paintbrush button in the tool
// palette. Two nested ToolPalette instances provide the grid/selection/
// hit-test machinery for both rows, but only the size row uses ToolPalette's
// own glyph rendering — the shape row is drawn manually (see Draw) as a
// filled preview of the actual brush pattern, not a pictogram borrowed from
// an unrelated tool.
type penOptions struct {
	shape *zenui.ToolPalette
	size  *zenui.ToolPalette
}

// Shape IDs match guiutil.BrushShape's three values by name, translated at
// the point of use (see brushShapeFor) — kept as plain strings here rather
// than importing guiutil's type directly, since zenui widgets don't
// otherwise depend on cmd/zenimate-gui's own internal packages.
const (
	penShapeRound  = "round"
	penShapeSquare = "square"
	penShapeCustom = "custom"
)

// newPenOptions builds the panel anchored at the given point (typically just
// above the paintbrush button), restoring the current shape/size selection
// rather than always defaulting to the first option.
func newPenOptions(anchor zenui.Rect, currentShape string, currentSize int) *penOptions {
	// Glyph values are unused now that the shape row is drawn manually (see
	// Draw) rather than through ToolPalette's own glyph rendering — kept at
	// zero rather than removed, since zenui.Tool still requires the field.
	shapeTools := []zenui.Tool{
		{ID: penShapeRound},
		{ID: penShapeSquare},
		{ID: penShapeCustom},
	}
	shape := zenui.NewToolPalette(zenui.ToolPaletteConfig{
		Anchor: anchor, Tools: shapeTools, Cols: 3,
		ButtonW: 32, ButtonH: 32, GapX: 4, GapY: 4, GlyphSize: 24,
	})
	shape.SetSelected(currentShape)

	sizeTools := make([]zenui.Tool, 4)
	for i := range sizeTools {
		n := i + 1
		sizeTools[i] = zenui.Tool{ID: string(rune('0' + n)), Glyph: rune('0' + n)}
	}
	sizeAnchor := zenui.Rect{X: anchor.X, Y: anchor.Y + 32 + 8}
	size := zenui.NewToolPalette(zenui.ToolPaletteConfig{
		Anchor: sizeAnchor, Tools: sizeTools, Cols: 4,
		ButtonW: 24, ButtonH: 24, GapX: 3, GapY: 3, GlyphSize: 8,
	})
	size.SetSelected(string(rune('0' + currentSize)))

	return &penOptions{shape: shape, size: size}
}

// panelPadding is the margin between the containing panel's edge and the
// buttons it holds, on all sides.
const panelPadding = 6

// Bounds returns the panel's overall bounding rect (shape row plus size row
// plus the gap between them, plus the panel's own padding), for
// click-outside-to-close detection — expanded by panelPadding so clicking
// the padding margin itself doesn't count as "outside" and close the panel.
func (p *penOptions) Bounds() zenui.Rect {
	sb, zb := p.shape.Bounds(), p.size.Bounds()
	tight := zenui.Rect{
		X: sb.X, Y: sb.Y,
		W: maxInt(sb.W, zb.W),
		H: (zb.Y + zb.H) - sb.Y,
	}
	return zenui.Rect{
		X: tight.X - panelPadding, Y: tight.Y - panelPadding,
		W: tight.W + 2*panelPadding, H: tight.H + 2*panelPadding,
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Update dispatches the input to both sub-palettes. Returns the current
// shape and size selections (unchanged if nothing was picked this frame).
func (p *penOptions) Update(in zenui.Input) (shape string, size int) {
	p.shape.Update(in)
	p.size.Update(in)
	shapeID, _ := p.shape.Selected()
	sizeID, _ := p.size.Selected()
	sizeN := 1
	if len(sizeID) == 1 && sizeID[0] >= '1' && sizeID[0] <= '9' {
		sizeN = int(sizeID[0] - '0')
	}
	return shapeID, sizeN
}

// Draw renders the panel: first a solid containing rectangle (background
// plus border) sized to Bounds(), so the buttons sit inside a contained
// panel rather than floating with the gaps between them showing whatever's
// underneath — then the shape row, drawn manually as a filled preview of
// the actual BrushStamp pattern for each shape (see drawBrushShapeButton)
// rather than through ToolPalette's generic glyph rendering, so the buttons
// show the real shape the brush produces instead of a pictogram borrowed
// from the separate ellipse/rectangle/customshape drawing tools. The size
// row still uses ToolPalette's own glyph rendering, since plain digits are
// exactly what a glyph-based renderer is for.
//
// Uses theme.Sidebar rather than theme.Panel for the containing fill:
// fpTheme() maps both Panel and Button from the same source colour
// (dt.Btn), so a Panel-filled background would be visually identical to
// the buttons themselves and the gaps would still look uncontained.
// Sidebar maps from dt.BG, the app's actual darker background tone, giving
// real contrast against the buttons sitting on top of it.
func (p *penOptions) Draw(r zenui.Renderer, theme zenui.Theme) {
	bounds := p.Bounds()
	r.FillRect(bounds, theme.Sidebar)
	r.StrokeRect(bounds, theme.Border, 1)

	selected, _ := p.shape.Selected()
	for _, id := range []string{penShapeRound, penShapeSquare, penShapeCustom} {
		rect, ok := p.shape.RectFor(id)
		if !ok {
			continue
		}
		drawBrushShapeButton(r, rect, brushShapeFor(id), id == selected, theme)
	}
	p.size.Draw(r, theme)
}

// drawBrushShapeButton draws one shape-picker button: background/border
// chrome matching ToolPalette's own button styling, then a filled preview
// of BrushStamp's actual pattern at a representative size (3 — the most
// commonly used size, and the one where Round and Custom are guaranteed
// distinct by construction, not coincidence — see guiutil.BrushStamp's own
// doc comment), scaled up so each "brush pixel" is clearly visible.
func drawBrushShapeButton(r zenui.Renderer, rect zenui.Rect, shape guiutil.BrushShape, selected bool, theme zenui.Theme) {
	bg := theme.Button
	if selected {
		bg = theme.SelFill
	}
	r.FillRect(rect, bg)
	r.StrokeRect(rect, theme.Border, 1)

	const previewSize = 3
	const blockPx = 6
	cx := rect.X + rect.W/2
	cy := rect.Y + rect.H/2
	guiutil.BrushStamp(shape, previewSize, func(dx, dy int) {
		x := cx + dx*blockPx - blockPx/2
		y := cy + dy*blockPx - blockPx/2
		r.FillRect(zenui.Rect{X: x, Y: y, W: blockPx, H: blockPx}, theme.Text)
	})
}

// brushShapeFor maps a penOptions shape ID to the guiutil.BrushShape it
// selects — the one place these two representations meet, since zenui
// widgets don't depend on cmd/zenimate-gui's own internal packages.
func brushShapeFor(id string) guiutil.BrushShape {
	switch id {
	case penShapeSquare:
		return guiutil.BrushSquare
	case penShapeCustom:
		return guiutil.BrushCustom
	default:
		return guiutil.BrushRound
	}
}
