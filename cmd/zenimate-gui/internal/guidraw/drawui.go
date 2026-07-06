package guidraw

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guiutil"
	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

// pad matches the window-edge margin used throughout this GUI (same value
// package main uses for its own layout math) — a small, deliberate
// duplication of one constant, the same trade-off already made in
// previewpane_draw.go's screenPad, rather than threading it through as a
// parameter for something that never varies.
const pad = 16

func DrawUI(txt *BDFText, c *ui.Controller, l Layout, theme Theme, cell, ox, oy, pppMin, pppMax float32, mx, my, focusX, focusY int, btnOpacity []float32) {
	s := c.Sprite

	// Transparency chequer's own zoom-based fade: fully visible at 40% and
	// above, gone entirely at 10% and below. Computed once here from the
	// same cell/pppMin/pppMax the zoom readout itself uses, so it tracks
	// the readout exactly rather than a separately-derived value.
	chequerFade := guiutil.ChequerFade(guiutil.PPPToPercent(cell, pppMin, pppMax))

	// Title block. Expanded: ZENIMATE large, dimmer subtitle, then the size/frame
	// header. Collapsed: a small button that restores the title on click. The
	// whole block is the click target that toggles collapse.
	if l.TitleCollapsed {
		r := l.TitleRect
		bc := theme.Btn
		if RectHit(r, mx, my) {
			bc = theme.BtnHot
		}
		rl.DrawRectangleRec(r, bc)
		rl.DrawRectangleLinesEx(r, 1, theme.Grid)
		const zscale = 2
		zw := txt.Measure("Z", zscale)
		zh := txt.CellH() * zscale
		txt.Draw("Z", int(r.X)+(int(r.Width)-zw)/2, int(r.Y)+(int(r.Height)-zh)/2, zscale, theme.Yellow)
	} else {
		txt.Draw("ZENIMATE", pad, pad, 3, theme.Yellow)
		txt.Draw("ZX SPECTRUM, PAINT AND ANIMATE", pad, pad+30, 1, theme.Dim)
		header := "SIZE " + guiutil.Itoa(s.Width()) + "X" + guiutil.Itoa(s.Height()) +
			"  FRAME " + guiutil.Itoa(s.Selected()+1) + "/" + guiutil.Itoa(s.FrameCount())
		txt.Draw(header, pad, pad+50, 1, theme.Text)
		// Source line: which file/bundle the sprite is from (or "unsaved").
		// Long labels are truncated to 30 characters followed by an ellipsis.
		txt.Draw(guiutil.Upper(guiutil.TruncateLabel(c.SourceLabel(), 30)), pad, pad+64, 1, theme.Dim)
	}

	// Horizontal frame strip near the top. When any label is three characters
	// (F10 and beyond) the font is reduced for the whole strip so the wider
	// labels fit; otherwise the larger size is used.
	// Either way each label is centred in its button.
	labelScale := 2
	if s.FrameCount() >= 10 {
		labelScale = 1
	}
	// Frame scrubber slider above the buttons: a thin track with a small square
	// indicator marking the current frame. The square sits on the slider's
	// baseline and rises a few pixels above its top; drag it to move between
	// frames.
	if l.ScrubRect.Width > 0 {
		rl.DrawRectangleRec(l.ScrubRect, theme.Btn)
		rl.DrawRectangleLinesEx(l.ScrubRect, 1, theme.Grid)
		nf := s.FrameCount()
		if nf > 0 {
			const overhang = 4 // pixels the square rises above the slider top
			side := l.ScrubRect.Height + overhang
			// Centre the square over the current frame's slot.
			slot := l.ScrubRect.Width / float32(nf)
			cx := l.ScrubRect.X + (float32(s.Selected())+0.5)*slot
			sqX := cx - side/2
			// Keep the square within the slider's horizontal span.
			if sqX < l.ScrubRect.X {
				sqX = l.ScrubRect.X
			}
			if sqX+side > l.ScrubRect.X+l.ScrubRect.Width {
				sqX = l.ScrubRect.X + l.ScrubRect.Width - side
			}
			// Bottom-aligned to the slider baseline, extending upward by overhang.
			sqY := l.ScrubRect.Y + l.ScrubRect.Height - side
			rl.DrawRectangleRec(rl.NewRectangle(sqX, sqY, side, side), theme.Sel)
		}
	}
	for i := range l.FrameRects {
		r := l.FrameRects[i]
		fillc := theme.Btn
		if i == s.Selected() {
			fillc = theme.Sel
		} else if RectHit(r, mx, my) {
			fillc = theme.BtnHot
		}
		rl.DrawRectangle(int32(r.X), int32(r.Y), int32(r.Width), int32(r.Height), fillc)
		rl.DrawRectangleLines(int32(r.X), int32(r.Y), int32(r.Width), int32(r.Height), theme.Grid)
		label := "F" + guiutil.Itoa(i+1)
		lw := txt.Measure(label, labelScale)
		lh := txt.CellH() * labelScale
		txt.Draw(label, int(r.X)+(int(r.Width)-lw)/2, int(r.Y)+(int(r.Height)-lh)/2, labelScale, theme.Text)
	}

	// Frame +/- buttons to the right of the strip.
	drawSmallBtn := func(r rl.Rectangle, label string, enabled bool) {
		bc := theme.Btn
		if !enabled {
			bc = theme.BG
		} else if RectHit(r, mx, my) {
			bc = theme.BtnHot
		}
		rl.DrawRectangleRec(r, bc)
		rl.DrawRectangleLinesEx(r, 1, theme.Grid)
		lc := theme.Text
		if !enabled {
			lc = theme.Dim
		}
		// Smaller symbol (scale 1) centred in the button box.
		const gscale = 1
		gw := txt.Measure(label, gscale)
		gh := txt.CellH() * gscale
		txt.Draw(label, int(r.X)+(int(r.Width)-gw)/2, int(r.Y)+(int(r.Height)-gh)/2, gscale, lc)
	}
	drawSmallBtn(l.AddFrameRect, "+", s.FrameCount() < model.MaxFrames)
	drawSmallBtn(l.HelpRect, "HELP", true)

	// View-mode buttons; the active mode is highlighted.
	modeNames := []ui.ViewMode{ui.BitmapWhite, ui.BitmapBlack, ui.SpectrumColour}
	for i, b := range l.ModeButtons {
		bc := theme.Btn
		if c.Mode() == modeNames[i] {
			bc = theme.Sel
		} else if b.Hit(mx, my) {
			bc = theme.BtnHot
		}
		rl.DrawRectangle(int32(b.X), int32(b.Y), int32(b.W), int32(b.H), bc)
		rl.DrawRectangleLines(int32(b.X), int32(b.Y), int32(b.W), int32(b.H), theme.Grid)
		DrawWrappedLabel(txt, b, theme.Text)
	}

	// Chequer-toggle LEDs below the two bitmap-mode buttons: lit (green) when the
	// chequer is on for that mode, dark when off.
	drawLed := func(r rl.Rectangle, on bool) {
		fill := theme.Btn // dark when off
		if on {
			fill = theme.Sel // green when on
		}
		rl.DrawRectangleRec(r, fill)
		rl.DrawRectangleLinesEx(r, 1, theme.Grid)
	}
	drawLed(l.ChkLedWhite, theme.ChequerOnWhite)
	drawLed(l.ChkLedBlack, theme.ChequerOnBlack)

	// Onion-skin toggle buttons: tinted to their silhouette colour when active.
	// Dimmed in Spectrum Colour mode, where onion skins are not shown.
	onionActive := c.Mode() != ui.SpectrumColour
	onionStates := []bool{c.OnionPrev(), c.OnionNext()}
	onionTints := []rl.Color{
		rl.NewColor(0x80, 0x20, 0x20, 0xff),
		rl.NewColor(0x20, 0x80, 0x20, 0xff),
	}
	for i, b := range l.OnionButtons {
		bc := theme.Btn
		if onionStates[i] {
			bc = onionTints[i]
		} else if b.Hit(mx, my) {
			bc = theme.BtnHot
		}
		rl.DrawRectangle(int32(b.X), int32(b.Y), int32(b.W), int32(b.H), bc)
		rl.DrawRectangleLines(int32(b.X), int32(b.Y), int32(b.W), int32(b.H), theme.Grid)
		lc := theme.Text
		if !onionActive {
			lc = theme.Dim
		}
		DrawWrappedLabel(txt, b, lc)
	}

	// Editor grid, pan/zoom transformed with FRACTIONAL cell/origin so zoom is
	// smooth (no integer snapping). Clipped to its layout box with a scissor so a
	// panned/zoomed sprite never overdraws the surrounding UI. Each cell spans
	// [ox+x*cell, ox+(x+1)*cell] exactly, so cells tile seamlessly at any scale.
	sw, sh := s.Width(), s.Height()
	gw := float32(sw) * cell
	gh := float32(sh) * cell
	boxX, boxY := float32(l.GridX), float32(l.GridY)
	boxR, boxB := boxX+float32(l.GridW), boxY+float32(l.GridH)

	rl.BeginScissorMode(int32(l.GridX), int32(l.GridY), int32(l.GridW), int32(l.GridH))

	// Lighter backing so the grid box reads clearly even when the sprite is
	// panned away.
	rl.DrawRectangle(int32(l.GridX), int32(l.GridY), int32(l.GridW), int32(l.GridH), theme.GridArea)

	mode := c.Mode()
	for y := 0; y < sh; y++ {
		ry0 := oy + float32(y)*cell
		ry1 := oy + float32(y+1)*cell
		if ry1 < boxY || ry0 > boxB {
			continue
		}
		for x := 0; x < sw; x++ {
			rx0 := ox + float32(x)*cell
			rx1 := ox + float32(x+1)*cell
			if rx1 < boxX || rx0 > boxR {
				continue
			}
			rect := rl.NewRectangle(rx0, ry0, rx1-rx0, ry1-ry0)
			on := s.At(x, y)

			switch mode {
			case ui.SpectrumColour:
				// 1 -> cell ink colour, 0 -> cell paper colour.
				attr := s.AttrAt(x, y)
				var idx int
				if on {
					idx = zxpalette.Index(zxpalette.Ink(attr), zxpalette.Bright(attr))
				} else {
					idx = zxpalette.Index(zxpalette.Paper(attr), zxpalette.Bright(attr))
				}
				rl.DrawRectangleRec(rect, ZxColor(zxpalette.RGBA[idx]))
			case ui.BitmapWhite:
				if on {
					rl.DrawRectangleRec(rect, theme.Ink) // white
				} else {
					theme.DrawCheckerPixel(rx0, ry0, rx1-rx0, ry1-ry0, x, y, ui.BitmapWhite, theme.ChequerOnWhite, chequerFade)
				}
			default: // BitmapBlack
				if on {
					rl.DrawRectangleRec(rect, rl.Black)
				} else {
					theme.DrawCheckerPixel(rx0, ry0, rx1-rx0, ry1-ry0, x, y, ui.BitmapBlack, theme.ChequerOnBlack, chequerFade)
				}
			}
		}
	}

	// Onion skins: in the bitmap views only, overlay the previous frame's set
	// pixels in translucent red and the next frame's in translucent green, each
	// independently toggleable. Drawn over the current frame so the ghosts read.
	if mode != ui.SpectrumColour {
		drawOnion := func(f int, col rl.Color) {
			fr := s.Frame(f)
			if fr == nil {
				return
			}
			for y := 0; y < sh; y++ {
				ry0 := oy + float32(y)*cell
				ry1 := oy + float32(y+1)*cell
				if ry1 < boxY || ry0 > boxB {
					continue
				}
				for x := 0; x < sw; x++ {
					if !fr.At(x, y, sw) {
						continue
					}
					rx0 := ox + float32(x)*cell
					rx1 := ox + float32(x+1)*cell
					if rx1 < boxX || rx0 > boxR {
						continue
					}
					rl.DrawRectangleRec(rl.NewRectangle(rx0, ry0, rx1-rx0, ry1-ry0), col)
				}
			}
		}
		if c.OnionPrev() {
			drawOnion(c.PrevFrameIndex(), theme.OnionPrev)
		}
		if c.OnionNext() {
			drawOnion(c.NextFrameIndex(), theme.OnionNext)
		}
	}

	// Single scale unit: zoom percentage, mapped linearly from the on-screen pixel
	// size (ppp = device px per virtual pixel = cellF) across the fixed zoom range.
	// The scale is window-independent (fixed base cell x persistent v.zoom, the Quag
	// model), so this percentage means the same thing for every sprite and window:
	//   pppMin (5px)  -> 0%      pppMax (160px) -> 800%.
	// The readout shows this same percentage and the grid/overlay thresholds are
	// stated in it, so what is read is exactly what drives the fades.
	ppp := cell
	zoomPct := guiutil.PPPToPercent(ppp, pppMin, pppMax)

	// In Spectrum Colour mode, overlay an almost-invisible 1px-resolution grid so
	// individual virtual pixels are discernible. Fades in between the pixGrid
	// thresholds (device px per virtual pixel).
	if mode == ui.SpectrumColour {
		if pf := guiutil.PixGridFade(zoomPct); pf > 0 {
			pc := theme.PixGrid
			pc.A = uint8(float32(theme.PixGrid.A) * pf)
			for x := 1; x < sw; x++ {
				gx := ox + float32(x)*cell
				rl.DrawRectangleRec(rl.NewRectangle(gx, oy, 1, gh), pc)
			}
			for y := 1; y < sh; y++ {
				gy := oy + float32(y)*cell
				rl.DrawRectangleRec(rl.NewRectangle(ox, gy, gw, 1), pc)
			}
		}

		// Flat-cell overlay: when very zoomed in, mark set pixels that are visually
		// invisible because their cell's ink and paper are the same colour (common
		// after image import). Each such set pixel gets a thin inner stroke so the
		// hidden pixels can be seen and edited. Full at >= 600% zoom, fading out by
		// 400%.
		if ff := guiutil.FlatCellFade(zoomPct); ff > 0 {
			for cy := 0; cy < s.AttrRows(); cy++ {
				for cx := 0; cx < s.AttrCols(); cx++ {
					attr := s.AttrCell(cx, cy)
					if zxpalette.Ink(attr) != zxpalette.Paper(attr) {
						continue // not a flat cell — pixels are already visible
					}
					// Stroke each set pixel in this cell. Contrast against the cell's
					// (single) colour using the same black/white chooser as the marks.
					strokeC := theme.MarkColour(zxpalette.Ink(attr))
					strokeC.A = uint8(255 * ff)
					x0 := cx * 8
					y0 := cy * 8
					for py := y0; py < y0+8 && py < sh; py++ {
						for px := x0; px < x0+8 && px < sw; px++ {
							if !s.At(px, py) {
								continue
							}
							rx := ox + float32(px)*cell
							ry := oy + float32(py)*cell
							if rx+cell < boxX || rx > boxR || ry+cell < boxY || ry > boxB {
								continue
							}
							// Thin inner stroke inset one pixel inside the pixel square.
							rl.DrawRectangleLinesEx(rl.NewRectangle(rx+1, ry+1, cell-2, cell-2), 1, strokeC)
						}
					}
				}
			}
		}
	}

	// Character-cell guides: a dark-grey line every 8 sprite pixels. Full strength
	// at >= 250% zoom, fading linearly to fully transparent at 80% (no guides at
	// 80% and below).
	if cf := guiutil.CellGuideFade(zoomPct); cf > 0 {
		gc := theme.Guide
		gc.A = uint8(float32(theme.Guide.A) * cf)
		for x := 8; x < sw; x += 8 {
			rl.DrawRectangleRec(rl.NewRectangle(ox+float32(x)*cell, oy, 1, gh), gc)
		}
		for y := 8; y < sh; y += 8 {
			rl.DrawRectangleRec(rl.NewRectangle(ox, oy+float32(y)*cell, gw, 1), gc)
		}
	}
	// Outer border around the sprite (always drawn, at full strength).
	rl.DrawRectangleLinesEx(rl.NewRectangle(ox, oy, gw, gh), 1, theme.Guide)

	rl.EndScissorMode()

	// Medium-grey border around the viewport box itself, drawn outside the
	// scissor so it is never clipped and stays fixed regardless of pan/zoom.
	rl.DrawRectangleLinesEx(rl.NewRectangle(float32(l.GridX), float32(l.GridY),
		float32(l.GridW), float32(l.GridH)), 1, theme.VPBorder)

	// Drawer-toggle triangle just below the viewport's bottom border: points up
	// when the drawer is closed (click to open), down when open (click to close).
	drawDrawerTriangle(l, theme, mx, my)

	// Buttons.
	// Strip buttons fade in/out as the window resize changes whether they fit
	// beside the viewport. The opacity is animated over time (btnOpacity), so the
	// transition plays fully even when a resize is reported as one completed step.
	for i, b := range l.Buttons {
		af := float32(1)
		if i < len(btnOpacity) {
			af = btnOpacity[i]
		}
		if af <= 0 {
			continue // fully faded: skip drawing entirely
		}
		a := uint8(255 * af)
		bc := theme.Btn
		if b.Hit(mx, my) {
			bc = theme.BtnHot
		}
		bc.A = a
		gc := theme.Grid
		gc.A = a
		tc := theme.Text
		tc.A = a
		rl.DrawRectangle(int32(b.X), int32(b.Y), int32(b.W), int32(b.H), bc)
		rl.DrawRectangleLines(int32(b.X), int32(b.Y), int32(b.W), int32(b.H), gc)
		theme.DrawButtonLabelColour(txt, guiutil.Upper(b.Label), b.X, b.Y, b.W, b.H, tc)
	}
}

func drawDrawerTriangle(l Layout, theme Theme, mx, my int) {
	r := l.DrawerToggle
	// Subtle backing so the triangle reads over any sprite content beneath it.
	pad2 := float32(3)
	bg := rl.NewRectangle(r.X-pad2, r.Y-pad2, r.Width+2*pad2, r.Height+2*pad2)
	rl.DrawRectangleRec(bg, rl.NewColor(0x10, 0x10, 0x18, 0xb0))
	col := theme.VPBorder
	if RectHit(r, mx, my) {
		col = theme.Text
	}
	cx := r.X + r.Width/2
	open := l.DrawerOpen >= 0.5
	if open {
		// Pointing down: apex at the bottom centre.
		rl.DrawTriangle(
			rl.NewVector2(r.X, r.Y),
			rl.NewVector2(cx, r.Y+r.Height),
			rl.NewVector2(r.X+r.Width, r.Y),
			col)
	} else {
		// Pointing up: apex at the top centre.
		rl.DrawTriangle(
			rl.NewVector2(r.X, r.Y+r.Height),
			rl.NewVector2(r.X+r.Width, r.Y+r.Height),
			rl.NewVector2(cx, r.Y),
			col)
	}
}
