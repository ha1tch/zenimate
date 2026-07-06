// Command zenimate-gui is the raylib desktop frontend for the ZX Spectrum
// animated sprite editor. It presents the pixel grid, an eight-frame selector, a
// live preview, and action buttons, driving the shared ui.Controller.
//
// All on-screen text is rendered through the bundled Sinclair ZX Spectrum BDF
// font via the guidraw.BDFText renderer (pkg/bdf -> raylib textures). raylib's own font
// API is never used.
//
// The window needs an OpenGL-capable display, so this binary runs on a desktop
// (it will not run in a headless container). It still compiles everywhere the
// raylib cgo/purego toolchain is available.
package main

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guidraw"
	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guiutil"
	"github.com/ha1tch/zenimate/internal/fonts"
	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/version"
	"github.com/ha1tch/zenimate/pkg/zenui"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

// Layout constants. winW/winH are the INITIAL window size; the window is
// resizable and the live size is read each frame, so these are only the
// starting dimensions.
const (
	winW = 980
	winH = 680

	minWinW = 640
	minWinH = 520

	pad        = 16
	cellPx     = 20  // editor pixel size on screen
	previewBox = 162 // fixed detail-preview box size in window pixels — matches the
	// ToolPalette/attribute-palette column width (4 cols x 36px + 3x6px gaps) so
	// all three share the same left AND right edge, not just an incidental overlap

	frameBtnW = 56
	frameBtnH = 28
	frameGap  = 6
	modeBtnH  = 42 // taller row for word-wrapped two-line mode/onion labels

	btnW   = 120
	btnH   = 32
	btnGap = 10

	// doubleClickWindow is the maximum gap, in seconds, between two clicks
	// on the hand tool button for it to count as a double-click (re-fit
	// the viewport) rather than two independent single clicks.
	doubleClickWindow = 0.4
)

// computeLayout builds the layout for the given window size and controller. The
// button actions are bound to the controller here. The editor cell size adapts
// to the space between the frame strip and the button block so the grid always
// fits, whatever the window size and sprite dimensions.
func computeLayout(w, h int, c *ui.Controller, files *fileOps, titleCollapse float32, drawerOpen float32) guidraw.Layout {
	var l guidraw.Layout
	l.WinW, l.WinH = w, h
	l.TitleCollapse = titleCollapse
	// Past the halfway point the block reads as collapsed (drives the Z vs full
	// text and the height that the header budget uses).
	l.TitleCollapsed = titleCollapse >= 0.5
	l.DrawerOpen = drawerOpen

	// Title block, top-left. Expanded it carries "ZENIMATE" + subtitle + the
	// size/frame header at titleW wide; collapsed it shrinks to a small button.
	// The width eases between the two so the toolbars to its right slide rather
	// than snap. The grid begins just below whichever is taller.
	const titleW = 250
	const titleH = 64
	const collapsedW = 30
	const collapsedH = 28
	curW := float32(titleW) + (float32(collapsedW)-float32(titleW))*titleCollapse
	curH := titleH
	if l.TitleCollapsed {
		curH = collapsedH
	}
	l.TitleRect = rl.NewRectangle(pad, pad, curW, float32(curH))

	// Toolbars begin to the right of the title block and run to the window edge.
	toolX := int(l.TitleRect.X+l.TitleRect.Width) + 20
	toolRight := w - pad
	toolW := toolRight - toolX
	if toolW < 200 {
		toolW = 200 // keep usable even in a very narrow window
	}

	// Row 1: the frame strip (one button per frame) plus -/+ buttons, packed to
	// use the available width. Row 2: view-mode + onion buttons. A thin scrubber
	// slider sits just above the frame buttons for dragging between frames.
	const scrubH = 7
	const scrubGap = 3
	scrubY := pad
	row1Y := pad + scrubH + scrubGap
	row2Y := row1Y + frameBtnH + 8

	// Frame strip: size each frame button so the whole strip (frames + the two
	// small +/- buttons) fits toolW, clamped to a sensible maximum width.
	nf := c.Sprite.FrameCount()
	small := frameBtnH // square-ish +/- buttons
	avail := toolW - 2*(small+6)
	fbw := frameBtnW
	if nf > 0 {
		if fit := (avail - (nf-1)*frameGap) / nf; fit < fbw {
			fbw = fit
		}
	}
	if fbw < 24 {
		fbw = 24
	}
	l.FrameStripX = toolX
	l.FrameStripY = row1Y
	l.FrameRects = make([]rl.Rectangle, nf)
	for i := 0; i < nf; i++ {
		fx := l.FrameStripX + i*(fbw+frameGap)
		l.FrameRects[i] = rl.NewRectangle(float32(fx), float32(row1Y),
			float32(fbw), float32(frameBtnH))
	}
	rx := float32(l.FrameStripX + nf*(fbw+frameGap))
	l.AddFrameRect = rl.NewRectangle(rx, float32(row1Y), float32(small), float32(frameBtnH))

	// Scrubber track: a thin slider spanning the frame buttons (not the +/-),
	// sitting just above them. Dragging it selects a frame proportionally.
	scrubW := 0
	if nf > 0 {
		scrubW = nf*fbw + (nf-1)*frameGap
	}
	if scrubW < fbw {
		scrubW = fbw
	}
	l.ScrubRect = rl.NewRectangle(float32(l.FrameStripX), float32(scrubY),
		float32(scrubW), float32(scrubH))

	// HELP button: fixed at its startup position, directly below where the '-' and
	// '+' buttons sit with the default 8 frames at full width. It deliberately does
	// NOT track the live frame strip: adding frames moves '+'/'-' rightward, but
	// HELP stays put where it was on startup. Width spans the '-'-to-'+' gap
	// (2*small + 6). Height matches the onion/mode buttons.
	helpAnchorX := float32(l.FrameStripX + model.DefaultFrames*(frameBtnW+frameGap))
	helpW := 2*float32(small) + 6
	l.HelpRect = rl.NewRectangle(helpAnchorX, float32(row1Y)+float32(frameBtnH)+6,
		helpW, float32(modeBtnH))

	// Bottom button strip, as a sliding drawer. Two rows: actions and file
	// operations. When open (drawerOpen=1) the rows sit across the bottom; when
	// closed (0) they slide down off-screen and the viewport reclaims the space.
	// The width/height sizing buttons are not part of the strip — they sit in a
	// 2x2 block immediately to the right of the last strip column.
	bx := pad
	const drawerRows = 2

	// Responsive button sizing. At full size the strip uses btnW/btnGap, but in a
	// narrow window that would overflow, so past the point where the strip no
	// longer fits the buttons (and the gaps between them) shrink proportionally
	// until they do. Below a threshold width the labels are drawn on two lines so
	// they stay legible when the buttons are small (handled at draw time).
	ebtnW := btnW
	ebtnGap := btnGap
	// The strip spans columns 0..2, then the sizing area (preset column + a
	// two-column stepper block) and a transform group (another two-column block).
	// Expressed in full-column widths from bx, the rightmost extent is ~6 columns.
	naturalW := 6*(btnW+btnGap) - btnGap
	availStrip := w - bx - pad
	if naturalW > availStrip && naturalW > 0 {
		scale := float32(availStrip) / float32(naturalW)
		ebtnW = int(float32(btnW) * scale)
		ebtnGap = int(float32(btnGap) * scale)
		if ebtnW < 40 {
			ebtnW = 40 // floor so buttons stay clickable
		}
		if ebtnGap < 2 {
			ebtnGap = 2
		}
	}
	l.StripBtnW = ebtnW // remembered so draw can pick 1- vs 2-line labels

	stripBandH := drawerRows*btnH + (drawerRows-1)*8 // rows + inter-row gaps
	slide := int(float32(stripBandH+pad) * (1 - drawerOpen))
	byBottom := h - pad - btnH + slide // file-ops row (lowest)
	byMid := byBottom - btnH - 8       // actions row (top of the 2-row strip)
	row := func(i int) int { return bx + i*(ebtnW+ebtnGap) }
	// COPY and PASTE share the column-2 slot, each half its width with a tight
	// inner gap, so together they span one full button column. EXPORT and BUNDLE
	// sit directly beneath them at the same widths and x positions: EXPORT below
	// COPY, BUNDLE below PASTE.
	const cpInnerGap = 4
	cpW := (ebtnW - cpInnerGap) / 2
	cpLeftX := row(2)
	cpRightX := row(2) + cpW + cpInnerGap
	// RESET and CLS share column 0's slot the way Copy/Paste share column 2's.
	// RESET is destructive (whole animation) and asks for typed confirmation; CLS
	// clears only the current frame and needs no confirmation.
	rcW := (ebtnW - cpInnerGap) / 2
	l.Buttons = []guidraw.Button{
		{X: row(0), Y: byMid, W: rcW, H: btnH, Label: "RESET", Action: func() { resetRequested = true }},
		{X: row(0) + rcW + cpInnerGap, Y: byMid, W: rcW, H: btnH, Label: "CLS", Action: func() { c.Checkpoint(); c.ClearFrameCLS() }},
		{X: row(1), Y: byMid, W: ebtnW, H: btnH, Label: "Play/Stop", Action: c.TogglePlay},
		{X: cpLeftX, Y: byMid, W: cpW, H: btnH, Label: "Copy", Action: c.CopyFrame},
		{X: cpRightX, Y: byMid, W: cpW, H: btnH, Label: "Paste", Action: func() { c.Checkpoint(); c.PasteFrame() }},
		{X: row(0), Y: byBottom, W: ebtnW, H: btnH, Label: "Open", Action: func() { files.startOpen(c) }},
		{X: row(1), Y: byBottom, W: ebtnW, H: btnH, Label: "Save", Action: func() { files.save(c) }},
		{X: cpLeftX, Y: byBottom, W: cpW, H: btnH, Label: "Export", Action: func() { files.startExport(c) }},
		{X: cpRightX, Y: byBottom, W: cpW, H: btnH, Label: "Bundle", Action: func() { files.startBundleExport(c) }},
	}

	// Sizing area, occupying the old column-3 slot onward. Left sub-column holds
	// two preset-size buttons stacked (32x24 = full screen, 2x2 = smallest);
	// immediately to its right is the 2x2 block of per-cell width/height steppers.
	// All these buttons share the compact width szW and the strip button height.
	const szInnerGap = 4
	szW := (ebtnW - szInnerGap) / 2
	presetX := row(3)
	szX := presetX + szW + szInnerGap // steppers sit right of the presets
	l.Buttons = append(l.Buttons,
		// Preset sizes (left sub-column).
		guidraw.Button{X: presetX, Y: byMid, W: szW, H: btnH, Label: "32x24", Action: func() { c.Checkpoint(); c.SetSize(model.MaxWidth, model.MaxHeight) }},
		guidraw.Button{X: presetX, Y: byBottom, W: szW, H: btnH, Label: "2x2", Action: func() { c.Checkpoint(); c.SetSize(2*model.Cell, 2*model.Cell) }},
		// Per-cell width/height steppers (right 2x2 block); labels in cell units.
		guidraw.Button{X: szX, Y: byMid, W: szW, H: btnH, Label: "W -1", Action: func() { resizeCells(c, -1, 0) }},
		guidraw.Button{X: szX + szW + szInnerGap, Y: byMid, W: szW, H: btnH, Label: "W +1", Action: func() { resizeCells(c, +1, 0) }},
		guidraw.Button{X: szX, Y: byBottom, W: szW, H: btnH, Label: "H -1", Action: func() { resizeCells(c, 0, -1) }},
		guidraw.Button{X: szX + szW + szInnerGap, Y: byBottom, W: szW, H: btnH, Label: "H +1", Action: func() { resizeCells(c, 0, +1) }},
	)

	// Transform group: a 2x2 block of small buttons to the right of the steppers.
	// H FLIP / V FLIP on the top row, ROT 90 / INVERT on the bottom. ROT 90
	// rotates in place; holding Ctrl also resizes a non-square frame so nothing is
	// clipped. All three act on the active selection instead of the whole
	// frame when one exists — the resize-on-rotate modifier only applies to
	// the whole-frame case, since RotateSelection90 always resizes the
	// selection's own bounds to fit, with no clip-vs-resize choice to make.
	txX := szX + 2*szW + szInnerGap + btnGap // one gap past the stepper block
	l.Buttons = append(l.Buttons,
		guidraw.Button{X: txX, Y: byMid, W: szW, H: btnH, Label: "H FLIP", Action: func() {
			c.Checkpoint()
			if c.HasSelection() {
				c.FlipSelectionH()
			} else {
				c.FlipH()
			}
		}},
		guidraw.Button{X: txX + szW + szInnerGap, Y: byMid, W: szW, H: btnH, Label: "V FLIP", Action: func() {
			c.Checkpoint()
			if c.HasSelection() {
				c.FlipSelectionV()
			} else {
				c.FlipV()
			}
		}},
		guidraw.Button{X: txX, Y: byBottom, W: szW, H: btnH, Label: "ROT 90", Action: func() {
			c.Checkpoint()
			if c.HasSelection() {
				c.RotateSelection90()
				return
			}
			// Ctrl or Option (Alt) both trigger the resize-on-rotate behaviour —
			// Option is the conventional Mac modifier for this kind of held
			// alternate-action gesture, Ctrl the cross-platform default.
			resize := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) ||
				rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)
			c.Rotate90(resize)
		}},
		guidraw.Button{X: txX + szW + szInnerGap, Y: byBottom, W: szW, H: btnH, Label: "INVERT", Action: func() { c.Checkpoint(); c.Invert() }},
	)

	// Row 2: view-mode buttons then onion toggles, also starting at toolX. These
	// buttons are narrow with word-wrapped two-line labels, so the row is taller
	// than a normal button row.
	modeY := row2Y
	modeW := 84
	mrow := func(i int) int { return toolX + i*(modeW+btnGap) }
	l.ModeButtons = []guidraw.Button{
		{X: mrow(0), Y: modeY, W: modeW, H: modeBtnH, Label: "Bitmap White", Action: func() { c.SetMode(ui.BitmapWhite) }},
		{X: mrow(1), Y: modeY, W: modeW, H: modeBtnH, Label: "Bitmap Black", Action: func() { c.SetMode(ui.BitmapBlack) }},
		{X: mrow(2), Y: modeY, W: modeW, H: modeBtnH, Label: "Spectrum Colour", Action: func() { c.SetMode(ui.SpectrumColour) }},
	}

	// Tiny chequer-toggle LEDs centred below the Bitmap White / Bitmap Black
	// buttons.
	const ledW, ledH, ledGap = 10, 6, 3
	ledY := float32(modeY + modeBtnH + ledGap)
	l.ChkLedWhite = rl.NewRectangle(float32(mrow(0)+(modeW-ledW)/2), ledY, ledW, ledH)
	l.ChkLedBlack = rl.NewRectangle(float32(mrow(1)+(modeW-ledW)/2), ledY, ledW, ledH)

	// Onion-skin toggle buttons, fixed at their startup position: aligned to where
	// the F6 frame button sits with the default frame count at full button width.
	// Like the HELP button, they deliberately do NOT track the live frame strip, so
	// adding frames (which shrinks the buttons and shifts F6) leaves them put.
	onionW := 72
	ox0 := l.FrameStripX + 5*(frameBtnW+frameGap)
	l.OnionButtons = []guidraw.Button{
		{X: ox0, Y: modeY, W: onionW, H: modeBtnH, Label: "Onion Prev", Action: c.ToggleOnionPrev},
		{X: ox0 + onionW + btnGap, Y: modeY, W: onionW, H: modeBtnH, Label: "Onion Next", Action: c.ToggleOnionNext},
	}

	// The header occupies the taller of the title block and the two toolbar rows;
	// the grid starts just beneath it. Collapsing the title lets the toolbars
	// move up, shrinking the header and giving the viewport more vertical room.
	toolBottom := row2Y + modeBtnH
	headerBottom := toolBottom
	if tb := int(l.TitleRect.Y + l.TitleRect.Height); tb > headerBottom {
		headerBottom = tb
	}

	// Editor grid sits below the header; its cell size adapts so the whole grid
	// fits the box between the header and the button block, and within the
	// horizontal space left of the fixed preview/palette column.
	l.GridX = pad
	l.GridY = headerBottom + 16

	// Horizontal budget: [pad | grid | pad | previewBox | pad]. The gap between
	// the viewport and the preview column equals pad — the same margin the bottom
	// buttons keep from the window border.
	availW := w - 3*pad - previewBox // room left of preview column

	// The viewport extends down to just above the drawer (when open) or close to
	// the window bottom (when closed). The triangle toggle now sits inside the
	// viewport's bottom-right corner, so no external room is reserved for it; the
	// gaps are kept small.
	const triH = 14
	const triW = 18
	vpBottomOpen := byMid - 8     // small gap above the open drawer (top row = byMid)
	vpBottomClosed := h - pad - 4 // small margin to the window border
	vpBottom := vpBottomOpen
	if vpBottomClosed < vpBottom || drawerOpen < 1 {
		// While closing, follow the larger (lower) of the two so the viewport
		// grows smoothly as the drawer slides away.
		vb := float32(vpBottomOpen) + (float32(vpBottomClosed)-float32(vpBottomOpen))*(1-drawerOpen)
		vpBottom = int(vb)
	}
	availH := vpBottom - l.GridY // room above the drawer

	// Base cell size is FIXED at cellPx and never derived from window or sprite
	// size. This is the Quag lesson: there is one persistent scale. The on-screen
	// size of a virtual pixel is cellPx * v.zoom, where v.zoom is changed only by
	// the user (wheel) or explicit fit — a window resize never alters it (it only
	// re-pans to keep the view centred). Fitting a large sprite into the box is
	// done by lowering v.zoom via animateFit, not by shrinking this base, so the
	// grid/overlay thresholds (stated in device px per virtual pixel) mean the
	// same thing in every window size and for every sprite.
	l.Cell = cellPx

	// The grid is clipped to the available box (not the sprite-exact size), so
	// zooming/panning stays within a stable rectangle and never overdraws the
	// buttons or preview.
	l.GridW = availW
	l.GridH = availH
	if l.GridW < 0 {
		l.GridW = 0
	}
	if l.GridH < 0 {
		l.GridH = 0
	}

	// Drawer-toggle triangle, inside the viewport's bottom-right corner (a few px
	// in from the border so it reads as part of the viewport). The clickable hit
	// area is larger than the triangle itself: it spans from a small margin above/
	// left of the triangle out to the viewport's bottom-right edge, so a click
	// anywhere on or around the toggle activates it (and never paints).
	const triInset = 4
	const triPad = 6 // extra clickable margin around the triangle
	triX := l.GridX + l.GridW - triW - triInset
	triY := l.GridY + l.GridH - triH - triInset
	l.DrawerToggle = rl.NewRectangle(float32(triX), float32(triY), float32(triW), float32(triH))
	// Hit area: from (triX-triPad, triY-triPad) to the viewport's bottom-right.
	hx := triX - triPad
	hy := triY - triPad
	l.DrawerToggleHit = rl.NewRectangle(float32(hx), float32(hy),
		float32(l.GridX+l.GridW-hx), float32(l.GridY+l.GridH-hy))

	// Attribute palette anchor: bottom-aligned to the window with the same
	// margin as the bottom button strip. The 4x4 swatch grid itself (size,
	// classic Spectrum colour-key arrangement, hit-testing) is owned by a
	// zenui.ZXClassicPaletteChooser; only the anchor position is computed
	// here, since it depends on the window size computeLayout already has.
	const (
		paletteSwatchW, paletteSwatchH = 36, 24
		paletteGapX, paletteGapY       = 6, 5
		paletteCols, paletteRows       = 4, 4
	)
	paletteW := paletteCols*paletteSwatchW + (paletteCols-1)*paletteGapX
	paletteH := paletteRows*paletteSwatchH + (paletteRows-1)*paletteGapY
	l.PaletteX = w - pad - paletteW
	l.PaletteY = (h - pad) - paletteH // bottom edge aligns with the button strip's

	// Tool palette anchor, tucked directly above the attribute palette (same
	// gap as everywhere else in this column) — same width as the palette
	// below it (4 cols x 36px + 3 gaps = 162px either way), so the two grids
	// line up visually. The grid itself is owned by a zenui.ToolPalette; only
	// the anchor is computed here.
	const (
		toolBtnW, toolBtnH = 36, 36
		toolGapX, toolGapY = 6, 5
		toolCols, toolRows = 4, 3 // 12 tools, exactly 3 rows, no partial row
	)
	toolPaletteW := toolCols*toolBtnW + (toolCols-1)*toolGapX
	toolPaletteH := toolRows*toolBtnH + (toolRows-1)*toolGapY
	l.ToolPaletteX = w - pad - toolPaletteW
	l.ToolPaletteY = l.PaletteY - 16 - toolPaletteH

	// Fixed-size detail preview box in the top-right corner. It grows downward
	// to meet just above the tool palette (with a small gap), reclaiming the
	// vertical space the old fixed-height preview left empty.
	l.PreviewX = w - pad - previewBox
	l.PreviewY = l.GridY
	l.PreviewW = previewBox
	l.PreviewH = l.ToolPaletteY - 16 - l.PreviewY
	if l.PreviewH < previewBox {
		l.PreviewH = previewBox // never smaller than the base box
	}

	return l
}

// attrOnLabel returns the current ATTR ON/OFF indicator's text.
func attrOnLabel(attrOnGlobal bool) string {
	if attrOnGlobal {
		return "ATTR ON"
	}
	return "ATTR OFF"
}

// attrOnIndicatorRect computes the ATTR ON/OFF indicator's clickable rect,
// right-aligned above the palette column. Shared by drawing and
// click-detection so the two can't drift apart — "ATTR OFF" is a
// character wider than "ATTR ON", so the rect's position genuinely shifts
// with state, not just its label.
func attrOnIndicatorRect(l guidraw.Layout, txt *guidraw.BDFText, attrOnGlobal bool) zenui.Rect {
	label := attrOnLabel(attrOnGlobal)
	lw := len(label) * txt.CellW()
	lx := l.PaletteX + previewBox - lw
	ly := l.PaletteY - 12
	return zenui.Rect{X: lx - 2, Y: ly - 1, W: lw + 4, H: txt.CellH() + 2}
}

func main() {
	font, err := fonts.Sinclair()
	if err != nil {
		panic(err)
	}
	iconFont, err := fonts.ToolIcons()
	if err != nil {
		panic(err)
	}
	// The text tool's own, separately switchable font choice — distinct
	// from `font` above, which backs all regular UI text via `txt` and must
	// stay Sinclair regardless of what the text tool is set to.
	textFontsList := loadTextFonts()
	textFontIdx := 0 // Sinclair, the default

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(winW, winH, "zenimate "+version.Version)
	defer rl.CloseWindow()
	rl.SetWindowMinSize(minWinW, minWinH)
	rl.SetExitKey(0) // disable Esc-to-close; the window closes only via its button
	rl.SetTargetFPS(60)

	// Anchor the zoom scale to the screen: at 0% the tallest sprite fits the screen
	// height, at 800% each virtual pixel is 8x that. Uses the monitor the window
	// opened on, and re-anchors if the window is moved to a different monitor.
	curMonitor := rl.GetCurrentMonitor()
	setZoomRangeForScreen(rl.GetMonitorHeight(curMonitor))

	txt := guidraw.NewBDFText(font, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	defer txt.Unload()
	// A second text renderer over the icon face, for the tool palette only —
	// everything else keeps drawing through the regular Sinclair-backed txt.
	iconTxt := guidraw.NewBDFText(iconFont, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	defer iconTxt.Unload()

	c := ui.New(16, 16)
	vp := newViewport()
	osdNote := newOSD()
	var files fileOps         // modal file dialog (save/open/export)
	var help *helpModal       // scrollable help reader, when open
	var reset *resetConfirm   // typed reset confirmation, when open
	var frameMenu *zenui.Menu // frame-strip right-click context menu, when open
	frameMenuFrame := 0       // which frame index frameMenu was opened for
	drag := newFrameDrag()    // frame-strip drag-to-reorder state
	palette := zenui.NewZXClassicPaletteChooser(zenui.ZXClassicPaletteChooserConfig{
		SwatchW: 36, SwatchH: 24, GapX: 6, GapY: 5,
	})
	theme := guidraw.DefaultTheme()
	preview := zenui.NewPreviewPane(zenui.PreviewPaneConfig{
		Source: spritePreviewSource{c: c, theme: &theme}, MinZoom: 1, MaxZoom: 4,
	})
	preview.SetZoom(2) // matches the original fixed detail-preview default

	// The tool palette, tucked between the preview box and the attribute
	// palette. Glyphs are the 24px icon set (U+E100+i) — matching what
	// GlyphSize below expects, and the size already chosen as looking best
	// when these icons were first extracted. IDs are placeholders: no actual
	// tool behaviour is wired up yet (selection, fill, shapes, etc. are all
	// still just chrome at this point), so picking one only changes which
	// button is highlighted.
	toolPalette := zenui.NewToolPalette(zenui.ToolPaletteConfig{
		Tools: []zenui.Tool{
			{ID: "paintbrush", Glyph: 0xE101}, {ID: "select", Glyph: 0xE100},
			{ID: "fill", Glyph: 0xE102}, {ID: "eyedropper", Glyph: 0xE103},
			{ID: "line", Glyph: 0xE104}, {ID: "rectangle", Glyph: 0xE105},
			{ID: "ellipse", Glyph: 0xE106}, {ID: "triangle", Glyph: 0xE107},
			{ID: "polygon", Glyph: 0xE108}, {ID: "text", Glyph: 0xE10C},
			{ID: "hand", Glyph: 0xE10A}, {ID: "zoom", Glyph: 0xE10B},
		},
		Cols: 4, ButtonW: 36, ButtonH: 36, GapX: 6, GapY: 5, GlyphSize: 24,
	})

	// Pen shape/size: applies to the paintbrush's freehand stroke only (see
	// paintPixel's own comment on why attribute stamping stays single-pixel).
	// Defaults to round, size 1 — identical to plain single-pixel painting
	// until the user actively opens the options panel and changes something,
	// so existing muscle memory isn't disturbed by this feature existing.
	penShapeID := penShapeRound
	penSize := 1
	var penPanel *penOptions // nil when closed
	var fontMenu *zenui.Menu // nil when closed
	lastHandClickTime := float32(-1)
	lastCtrlReleaseTime := float32(-1)
	attrOnGlobal := false

	// Track the sprite dimensions so a resize can trigger a zoom-to-fit animation.
	lastW, lastH := c.Sprite.Width(), c.Sprite.Height()

	// Animation timing via raylib's clock.
	var playAccum float32

	// Per-button on-screen opacity, eased toward each button's target visibility
	// (1 = shown, 0 = hidden past the viewport edge) so buttons animate in/out on
	// a window resize even when the resize is reported as a single completed step.
	var btnOpacity []float32

	// Attribute palette fade: eases to 1 in Spectrum Colour mode and to 0 in the
	// bitmap modes, so the palette fades in/out when the mode changes rather than
	// snapping. Initialised to match the starting mode so it does not fade in on
	// launch.
	paletteFade := float32(0)
	if c.Mode() == ui.SpectrumColour {
		paletteFade = 1
	}

	// Title collapse: eased progress (0 = expanded, 1 = collapsed) toward a target.
	titleCollapse := float32(0)
	titleCollapseTarget := float32(0)

	// Shift axis-lock for straight-line drawing: the stroke's anchor pixel (mouse-
	// down point) and the locked axis, decided once per stroke and held.
	strokeActive := false
	strokeAnchorX, strokeAnchorY := 0, 0
	strokeAxis := guiutil.AxisNone

	// Select tool drag state: either defining a brand new selection (anchor
	// corner held, live-updated as the drag continues) or dragging an
	// existing selection's lifted content (offset from the click point to
	// the selection's own origin, so the drag doesn't snap the origin to the
	// cursor). Both moves and pastes stay floating after the drag ends,
	// rather than auto-committing — matching Photoshop's actual behaviour of
	// keeping pasted/moved content movable until an explicit deselect.
	selDragging := false
	selDraggingMove := false // true = dragging existing content; false = defining a new selection
	selAnchorX, selAnchorY := 0, 0
	selDragOffX, selDragOffY := 0, 0
	// True when this drag started as a "define a new selection" attempt
	// (not a move/duplicate) while a selection already existed — used at
	// release to tell a genuine click (no movement) from a real drag: a
	// plain click in that situation should deselect, not leave behind a
	// nonsensical 1x1-pixel selection at the click point.
	selNewAttemptHadPriorSelection := false

	// Shape tools (line/rectangle/triangle) drag state: anchor pixel held
	// from mouse-down, current end point tracked live for the preview
	// overlay, committed to the sprite only on release.
	shapeDragging := false
	shapeStartX, shapeStartY := 0, 0
	shapeEndX, shapeEndY := 0, 0
	// Polygon tool's current side count, adjustable mid-drag via number keys
	// 3-9 (see the polygon drag block below). Persists across drags so
	// picking, say, an octagon once keeps it an octagon next time.
	polygonSides := 6

	// Text tool state: click starts an entry, typing appends to it, Enter
	// commits into the sprite, Escape discards. Defaults to the Sinclair
	// face (font, already loaded above) — font choice is a later addition.
	var textState textEntry
	// Last pixel painted this stroke, used to interpolate a continuous line when
	// the pointer moves faster than one pixel per frame (otherwise fast strokes
	// leave sparse dotted gaps).
	lastPaintX, lastPaintY := 0, 0

	// When a press lands on the drawer toggle, painting is suppressed for the
	// entire hold — otherwise the viewport growing as the drawer closes would
	// slide paintable area under a still-held cursor and draw a stray pixel.
	toggleHeld := false

	// Bottom drawer: open by default, slides closed/open with an eased progress.
	drawerOpen := float32(1)
	drawerTarget := float32(1)

	// When the '+' (add-frame) button is clicked the button shifts right as the
	// new frame widens the strip; glide the pointer to the '+' button's new centre
	// over a few frames rather than snapping it there instantly.
	warpToAdd := false
	pointerGliding := false
	var pointerTargetX, pointerTargetY float32
	var pointerX, pointerY float32
	// Frame scrubber drag state: true while the pointer is dragging the slider.
	scrubbing := false

	for !rl.WindowShouldClose() {
		mx := int(rl.GetMouseX())
		my := int(rl.GetMouseY())

		// One Input snapshot per frame, reused by every consumer below.
		// rl.GetCharPressed() (inside fpInput) destructively drains a
		// shared, stateful character queue — calling fpInput() more than
		// once per frame silently hands all typed characters to whichever
		// caller happens to run first and leaves every later caller with
		// nothing, regardless of what was actually typed. This was the
		// real cause of the text tool appearing to do nothing: the preview
		// pane's own unconditional per-frame fpInput() call was draining
		// the queue before the text tool ever got a turn.
		frameIn := fpInput()
		// Captured once, here, before any selection-handling code this frame
		// can commit a pending float — used by handleKeys' gate below instead
		// of a fresh c.IsFloating() call, which would otherwise reflect
		// state already mutated earlier in this same frame (e.g. Enter
		// commits a floating selection via the selection-specific handling
		// further down, then a fresh IsFloating() check later in the same
		// frame would incorrectly see "not floating" and let handleKeys
		// process that same Enter keypress too, also toggling Play).
		wasFloatingThisFrame := c.IsFloating()

		// Re-anchor the zoom scale if the window has moved to a different monitor
		// (e.g. dragged to a display of a different height).
		if m := rl.GetCurrentMonitor(); m != curMonitor {
			curMonitor = m
			setZoomRangeForScreen(rl.GetMonitorHeight(m))
		}

		// Recompute layout from the live (resizable) window each frame.
		l := computeLayout(rl.GetScreenWidth(), rl.GetScreenHeight(), c, &files, titleCollapse, drawerOpen)
		s := c.Sprite
		palette.SetBounds(zenui.Rect{X: l.PaletteX, Y: l.PaletteY})
		toolPalette.SetBounds(zenui.Rect{X: l.ToolPaletteX, Y: l.ToolPaletteY})
		preview.SetBounds(zenui.Rect{X: l.PreviewX, Y: l.PreviewY, W: l.PreviewW, H: l.PreviewH})

		// Frame time, clamped, used by every ease this frame.
		dt := rl.GetFrameTime()
		if dt <= 0 || dt > 0.1 {
			dt = 1.0 / 60.0
		}

		// Animate each strip button's opacity toward its target visibility (fully
		// shown or fully hidden past the viewport edge). This makes buttons fade
		// in/out over a fixed time on resize, independent of how many resize steps
		// the platform reports. On first sight a button snaps to its target so the
		// initial layout doesn't fade in.
		vpRight := l.GridX + l.GridW
		if len(btnOpacity) != len(l.Buttons) {
			// Button set changed (or first frame): (re)initialise at the target.
			btnOpacity = make([]float32, len(l.Buttons))
			for i, b := range l.Buttons {
				if guiutil.ButtonVisible(b.X, b.W, vpRight) {
					btnOpacity[i] = 1
				}
			}
		}
		const btnFadeRate = 10 // higher = quicker fade
		k := clamp32(btnFadeRate*dt, 0, 1)
		for i, b := range l.Buttons {
			target := float32(0)
			if guiutil.ButtonVisible(b.X, b.W, vpRight) {
				target = 1
			}
			btnOpacity[i] += (target - btnOpacity[i]) * k
			if btnOpacity[i] > 0.999 {
				btnOpacity[i] = 1
			}
			if btnOpacity[i] < 0.001 {
				btnOpacity[i] = 0
			}
		}

		// Ease the attribute palette in/out on mode change (same rate as buttons).
		paletteTarget := float32(0)
		if c.Mode() == ui.SpectrumColour {
			paletteTarget = 1
		}
		paletteFade += (paletteTarget - paletteFade) * k
		if paletteFade > 0.999 {
			paletteFade = 1
		}
		if paletteFade < 0.001 {
			paletteFade = 0
		}

		// Deferred pointer glide: the layout above now reflects the added frame, so
		// the '+' button rect is at its new position. Start (or retarget) a glide
		// from the pointer's current spot to the button's new centre.
		if warpToAdd {
			warpToAdd = false
			pointerTargetX = l.AddFrameRect.X + l.AddFrameRect.Width/2
			pointerTargetY = l.AddFrameRect.Y + l.AddFrameRect.Height/2
			pointerX, pointerY = float32(mx), float32(my)
			pointerGliding = true
		}
		if pointerGliding {
			// Ease the pointer toward the target; snap and stop when close enough.
			k := clamp32(18*dt, 0, 1)
			pointerX += (pointerTargetX - pointerX) * k
			pointerY += (pointerTargetY - pointerY) * k
			dxp := pointerTargetX - pointerX
			dyp := pointerTargetY - pointerY
			if dxp*dxp+dyp*dyp < 1 {
				pointerX, pointerY = pointerTargetX, pointerTargetY
				pointerGliding = false
			}
			rl.SetMousePosition(int(pointerX), int(pointerY))
			mx, my = int(pointerX), int(pointerY)
		}

		// On a sprite-size change, animate the viewport to fit the whole sprite.
		if s.Width() != lastW || s.Height() != lastH {
			vp.animateFit(s.Width(), s.Height(), l.GridW, l.GridH, l.Cell)
			lastW, lastH = s.Width(), s.Height()
		}

		// Is the cursor over the editor grid box? (Gates wheel zoom and paint.)
		overGrid := mx >= l.GridX && mx < l.GridX+l.GridW &&
			my >= l.GridY && my < l.GridY+l.GridH

		// The help reader, when open, captures all input ahead of everything else.
		// The editor still renders underneath (drawn below), with the help overlay
		// on top.
		helpOpen := help != nil
		if helpOpen {
			if !help.update(frameIn) {
				help = nil
				helpOpen = false
			}
		}

		// The RESET button requests a destructive full reset; open the typed
		// confirmation modal (unless another modal is already up).
		if resetRequested {
			resetRequested = false
			if !helpOpen && !files.active() && reset == nil {
				reset = newResetConfirm()
			}
		}
		// The reset-confirm modal, when open, captures all input.
		resetOpen := reset != nil
		if resetOpen {
			switch reset.update(frameIn) {
			case resetConfirmConfirmed:
				c.Checkpoint()
				c.ResetAll()
				reset = nil
				resetOpen = false
			case resetConfirmCancelled:
				reset = nil
				resetOpen = false
			}
		}

		// The frame context menu, when open, captures all input.
		frameMenuOpen := frameMenu != nil
		if frameMenuOpen {
			switch frameMenu.Update(frameIn) {
			case zenui.Accepted:
				applyFrameMenuPick(c, frameMenu.Result(), frameMenuFrame)
				frameMenu = nil
				frameMenuOpen = false
			case zenui.Cancelled:
				frameMenu = nil
				frameMenuOpen = false
			}
		}

		// A modal file dialog, when open, captures all input: it runs first and the
		// editor's own interaction is frozen for the frame (geometry below is still
		// computed so the editor renders normally underneath the dialog).
		modal := helpOpen || resetOpen || frameMenuOpen || files.active()

		// Global ATTR ON toggle: releasing Ctrl twice in quick succession
		// flips a persistent, tool-independent mode — separate from the
		// existing per-stroke Ctrl-held attribute-paint gesture, this stays
		// on until toggled again, regardless of which tool is active or
		// whether Ctrl is currently held.
		if !modal {
			if rl.IsKeyReleased(rl.KeyLeftControl) || rl.IsKeyReleased(rl.KeyRightControl) {
				now := float32(rl.GetTime())
				if now-lastCtrlReleaseTime < doubleClickWindow {
					attrOnGlobal = !attrOnGlobal
				}
				lastCtrlReleaseTime = now
			}
		}
		if helpOpen || resetOpen || frameMenuOpen {
			// help/reset/frameMenu already updated above; suppress other interaction
		} else if files.active() {
			files.update(frameIn)
		} else {
			// File shortcuts: Ctrl+S save, Ctrl+O open, Ctrl+F toggle save form.
			// ctrl also matches Cmd (Super) so every shortcut in this block works
			// with either keymap — raylib-go/GLFW does not remap Cmd to Ctrl on
			// macOS (KeyLeftSuper/KeyRightSuper are the actual Cmd keys, entirely
			// distinct key codes from KeyLeftControl/KeyRightControl), so both
			// must be checked explicitly to support Mac users idiomatically while
			// keeping Ctrl working everywhere, including on a Mac with an
			// external/cross-platform keyboard.
			ctrl := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) ||
				rl.IsKeyDown(rl.KeyLeftSuper) || rl.IsKeyDown(rl.KeyRightSuper)
			if ctrl && rl.IsKeyPressed(rl.KeyS) {
				if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
					files.startSaveAs(c)
				} else {
					files.save(c)
				}
			}
			if ctrl && rl.IsKeyPressed(rl.KeyO) {
				files.startOpen(c)
			}
			if rl.IsKeyPressed(rl.KeyF1) {
				help = newHelpModal()
			}
			if ctrl && rl.IsKeyPressed(rl.KeyF) {
				c.ToggleSaveForm()
			}
			if ctrl && rl.IsKeyPressed(rl.KeyZ) {
				if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
					c.Redo()
				} else {
					c.Undo()
				}
			}
			if ctrl && rl.IsKeyPressed(rl.KeyC) {
				if c.HasSelection() {
					c.CopySelectionToClipboard()
				} else {
					c.CopyFrame()
				}
			}
			if ctrl && rl.IsKeyPressed(rl.KeyV) {
				c.Checkpoint()
				if c.HasSelectionClipboard() {
					c.PasteSelectionClipboard()
				} else {
					c.PasteFrame()
				}
			}
			if ctrl && rl.IsKeyPressed(rl.KeyD) {
				c.ClearSelection()
			}
			// Enter/Escape on a pending floating selection (a move or paste
			// not yet finalised): Enter confirms it in place, keeping the
			// selection active so it can still be copied/moved again.
			// Escape reverts the whole gesture via Undo — safe and exact,
			// since LiftSelection/PasteSelectionClipboard already pushed a
			// Checkpoint specifically to make this possible, rather than
			// needing a separate "restore the lifted content" primitive.
			if c.IsFloating() {
				if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
					c.CommitFloatingSelection()
				} else if rl.IsKeyPressed(rl.KeyEscape) {
					c.Undo()
				}
			}
			if (rl.IsKeyPressed(rl.KeyDelete) || rl.IsKeyPressed(rl.KeyBackspace)) && c.HasSelection() {
				c.Checkpoint()
				c.ClearSelectionArea()
			}
			if ctrl && rl.IsKeyPressed(rl.KeyE) {
				shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
				if shift {
					files.startBundleExport(c)
				} else {
					files.startExport(c)
				}
			}

			// Drag-and-drop: load a sprite/screen from a file dropped on the window.
			if rl.IsFileDropped() {
				dropped := rl.LoadDroppedFiles()
				files.handleDrop(c, dropped)
				rl.UnloadDroppedFiles()
			}
		}

		// Which tool is active in the tool palette, looked up once and shared
		// by every tool-specific behaviour below (panning via the hand tool,
		// eyedropper, zoom, fill) rather than re-querying per feature.
		selectedTool, _ := toolPalette.Selected()
		handToolActive := selectedTool == "hand"
		zoomToolActive := selectedTool == "zoom"
		fillToolActive := selectedTool == "fill"
		eyedropperActive := selectedTool == "eyedropper"
		selectToolActive := selectedTool == "select"
		lineToolActive := selectedTool == "line"
		rectToolActive := selectedTool == "rectangle"
		triangleToolActive := selectedTool == "triangle"
		ellipseToolActive := selectedTool == "ellipse"
		polygonToolActive := selectedTool == "polygon"
		textToolActive := selectedTool == "text"
		shapeToolActive := lineToolActive || rectToolActive || triangleToolActive || ellipseToolActive || polygonToolActive

		// Pan gesture: space + left-drag (space acts as a modifier, like Shift),
		// a middle-mouse-button drag on its own, or plain left-drag while the
		// hand tool is selected (matching the conventional paint-app Hand tool,
		// which needs no modifier key). Releasing the relevant button (or
		// space) ends the pan.
		spaceHeld := rl.IsKeyDown(rl.KeySpace) && !modal
		middlePan := rl.IsMouseButtonDown(rl.MouseButtonMiddle) && !modal
		handPan := handToolActive && rl.IsMouseButtonDown(rl.MouseLeftButton) && !modal
		panning := (spaceHeld && rl.IsMouseButtonDown(rl.MouseLeftButton)) || middlePan || handPan

		// Zoom tool: a click is treated as one wheel notch, reusing the same
		// cursor-anchored velocity model real wheel-scrolling drives — so it
		// eases in and decays identically, just triggered by a click instead
		// of a scroll. Alt/Option-click zooms out, matching the conventional
		// paint-app Zoom tool.
		if zoomToolActive && !modal && overGrid && rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			zoomOut := rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)
			notch := float32(1)
			if zoomOut {
				notch = -1
			}
			vp.zoomVel += notch * zoomWheelGain
		}

		// Update pan/zoom (wheel zoom anchored at cursor, drag pan, inertia). While
		// a modal dialog is open the viewport ignores the wheel and pan entirely.
		vp.update(mx, my, l.GridX, l.GridY, l.Cell, panning, overGrid && !modal)

		cellF := vp.cellF(l.Cell)
		oxF, oyF := vp.originF(l.GridX, l.GridY)

		// Preview focus point: the cursor's pixel while hovering the paint area,
		// otherwise the last-modified pixel. Frozen while a modal is open so the
		// preview does not track the cursor behind the overlay.
		focusX, focusY := c.LastPixel()
		if overGrid && !modal {
			fx := int((float32(mx) - oxF) / cellF)
			fy := int((float32(my) - oyF) / cellF)
			if float32(mx) >= oxF && float32(my) >= oyF &&
				fx >= 0 && fy >= 0 && fx < s.Width() && fy < s.Height() {
				focusX, focusY = fx, fy
			}
		}

		// Still needed below to suppress painting when a click lands on the
		// preview box rather than the canvas.
		overPreview := l.PreviewX <= mx && mx < l.PreviewX+l.PreviewW &&
			l.PreviewY <= my && my < l.PreviewY+l.PreviewH
		overToolPalette := toolPalette.Bounds().Contains(mx, my)

		// The preview pane owns its own press-hold/zoom-cycle/popup-easing state.
		// Press-start and right-click zoom-cycling are suppressed while a modal
		// is open, or while the pen options panel is open (it now lives above
		// the toolpalette, overlapping the preview box's own screen area, so
		// without this it silently steals clicks meant for the panel) — by
		// clearing those two Input fields only. MouseDown still reflects
		// reality, so release-based clearing and the popup's decay toward 0
		// keep running regardless — matching the old code's asymmetry, where
		// a stuck-open popup could never survive a modal opening mid-hold.
		preview.SetFocus(focusX, focusY)
		previewIn := frameIn
		if modal || penPanel != nil || fontMenu != nil {
			previewIn.MousePressed = false
			previewIn.MouseRightPressed = false
		}
		preview.Update(previewIn, dt)

		// Ease the drawer open/closed.
		drawerOpen += (drawerTarget - drawerOpen) * clamp32(9*dt, 0, 1)
		if drawerOpen > 0.999 {
			drawerOpen = 1
		}
		if drawerOpen < 0.001 {
			drawerOpen = 0
		}

		// Ease the title collapse/expand.
		titleCollapse += (titleCollapseTarget - titleCollapse) * clamp32(9*dt, 0, 1)
		if titleCollapse > 0.999 {
			titleCollapse = 1
		}
		if titleCollapse < 0.001 {
			titleCollapse = 0
		}

		// Animation tick.
		if c.Playing() {
			playAccum += rl.GetFrameTime()
			if playAccum >= float32(ui.PlayIntervalMS)/1000.0 {
				playAccum = 0
				c.Tick()
			}
		}

		// Ctrl acts as an attribute-paint modifier (like Shift): painting while
		// Ctrl is held stamps the current ink/paper/bright onto the cell rather
		// than drawing/erasing the bitmap.
		// Ctrl or Option (Alt) both act as the attribute-paint modifier — Option
		// is the conventional Mac modifier for this kind of held alternate-paint
		// gesture, Ctrl the cross-platform default.
		attrPaint := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl) ||
			rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)

		// The drawer triangle sits inside the viewport's bottom-right corner, so
		// painting must be suppressed over its (enlarged) hit area; the click
		// toggles the drawer instead.
		overDrawerToggle := guidraw.RectHit(l.DrawerToggleHit, mx, my)

		shiftHeld := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

		// A mouse release ends any in-progress drawer-toggle hold.
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			toggleHeld = false
		}

		// Editor grid interaction. Suppressed while the pan gesture is active so a
		// pan-drag never paints, over the drawer toggle, and for the remainder of
		// a hold that began on the drawer toggle.
		painting := !modal && !panning && overGrid && !overDrawerToggle && !toggleHeld && !eyedropperActive && !fillToolActive && !zoomToolActive && !selectToolActive && !shapeToolActive && !textToolActive &&
			(rl.IsMouseButtonDown(rl.MouseLeftButton) || rl.IsMouseButtonDown(rl.MouseRightButton))
		if painting {
			px := int((float32(mx) - oxF) / cellF)
			py := int((float32(my) - oyF) / cellF)

			// Stroke bookkeeping for Shift axis-lock: the anchor is the pixel where
			// the stroke began; the locked axis is decided on the first move away
			// from it and kept for the rest of the stroke.
			if !strokeActive {
				strokeActive = true
				c.Checkpoint()
				strokeAnchorX, strokeAnchorY = px, py
				strokeAxis = guiutil.AxisNone
				lastPaintX, lastPaintY = px, py
			}
			if shiftHeld {
				px, py, strokeAxis = guiutil.LockAxis(strokeAnchorX, strokeAnchorY, px, py, strokeAxis)
			} else {
				// Shift released mid-stroke: stop locking, but a later Shift press in
				// the same stroke re-decides from the anchor.
				strokeAxis = guiutil.AxisNone
			}

			if float32(mx) >= oxF && float32(my) >= oyF {
				// Paint one pixel (attribute stamp) or a whole brush stamp
				// (bitmap paint), bounds-checked, using the current tool mode.
				// Attribute stamping stays single-pixel: ZX attributes apply
				// per 8x8 cell, so a multi-pixel "brush" has no well-defined
				// meaning there the way it does for the bitmap.
				paintPixel := func(x, y int) {
					if attrPaint && !spaceHeld {
						if x >= 0 && y >= 0 && x < s.Width() && y < s.Height() {
							c.PaintAttr(x, y)
						}
						return
					}
					on := rl.IsMouseButtonDown(rl.MouseLeftButton)
					guiutil.BrushStamp(brushShapeFor(penShapeID), penSize, func(bdx, bdy int) {
						bx, by := x+bdx, y+bdy
						if bx < 0 || by < 0 || bx >= s.Width() || by >= s.Height() {
							return
						}
						c.Paint(bx, by, on)
					})
				}
				// Interpolate a straight line from the last painted pixel to the
				// current one so a fast stroke stays continuous instead of dotted.
				guiutil.ForEachLinePixel(lastPaintX, lastPaintY, px, py, paintPixel)
				lastPaintX, lastPaintY = px, py
			}
		} else {
			strokeActive = false
			strokeAxis = guiutil.AxisNone
		}

		// Eyedropper tool: a single click (not a drag) picks the clicked cell's
		// ink (left button) or paper (right button) as the new current
		// selection, matching the palette swatches' left=ink/right=paper
		// convention. No Checkpoint: like the swatches, this changes tool
		// state, not document content, so it isn't part of the undo history.
		if eyedropperActive && !modal && overGrid {
			epx := int((float32(mx) - oxF) / cellF)
			epy := int((float32(my) - oyF) / cellF)
			if float32(mx) >= oxF && float32(my) >= oyF &&
				epx >= 0 && epy >= 0 && epx < s.Width() && epy < s.Height() {
				attr := s.AttrAt(epx, epy)
				switch {
				case rl.IsMouseButtonPressed(rl.MouseLeftButton):
					c.SetInk(zxpalette.Ink(attr))
					c.SetBright(zxpalette.Bright(attr))
				case rl.IsMouseButtonPressed(rl.MouseRightButton):
					c.SetPaper(zxpalette.Paper(attr))
					c.SetBright(zxpalette.Bright(attr))
				}
			}
		}

		// Fill tool: a single click floods the clicked pixel's connected region
		// to the opposite state — left-click fills to on, right-click to off,
		// matching the paint tool's own convention for the two buttons.
		if fillToolActive && !modal && overGrid &&
			(rl.IsMouseButtonPressed(rl.MouseLeftButton) || rl.IsMouseButtonPressed(rl.MouseRightButton)) {
			fpx := int((float32(mx) - oxF) / cellF)
			fpy := int((float32(my) - oyF) / cellF)
			if float32(mx) >= oxF && float32(my) >= oyF {
				ctrlHeld := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
				floodFill(c, fpx, fpy, rl.IsMouseButtonPressed(rl.MouseLeftButton), ctrlHeld)
			}
		}

		// Select tool: dragging outside the current selection defines a new
		// one; dragging inside it moves the lifted content, or duplicates it
		// if Alt/Option is held at the moment the drag starts. Checkpointed
		// once at lift time (covering the whole gesture, however it resolves
		// — one undo step whether it's eventually dropped in place or dragged
		// far away), not again at commit.
		if selectToolActive && !modal && overGrid {
			spx := int((float32(mx) - oxF) / cellF)
			spy := int((float32(my) - oyF) / cellF)
			validClick := float32(mx) >= oxF && float32(my) >= oyF
			ctrlHeld := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
			altHeld := rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)

			switch {
			case rl.IsMouseButtonPressed(rl.MouseLeftButton) && validClick:
				selDragging = true
				sx, sy, sw, sh, hasSel := c.Selection()
				insideSelection := hasSel && spx >= sx && spx < sx+sw && spy >= sy && spy < sy+sh
				// Ctrl or Alt forces a move/duplicate of the existing
				// selection even when the click starts outside its bounds —
				// Photoshop's temporary-Move-tool (Ctrl) and duplicate-
				// while-dragging (Alt) modifiers, extended to work
				// regardless of click position rather than only once
				// already clicking inside the marquee.
				if hasSel && (insideSelection || ctrlHeld || altHeld) {
					selDraggingMove = true
					c.Checkpoint()
					c.LiftSelection(altHeld)
					selDragOffX, selDragOffY = spx-sx, spy-sy
				} else {
					selDraggingMove = false
					selAnchorX, selAnchorY = spx, spy
					selNewAttemptHadPriorSelection = hasSel
					// Alt-drag when defining a brand new selection (nothing
					// existing to move/duplicate) grows it from the clicked
					// point outward instead of treating that point as a
					// corner — the same centre/corner convention the
					// ellipse tool uses, and the Photoshop modifier
					// Horatio specifically asked for here.
					x0, y0, x1, y1 := guiutil.CenterOrCornerBounds(selAnchorX, selAnchorY, spx, spy, altHeld)
					c.SetSelection(x0, y0, x1, y1)
				}
			case selDragging && rl.IsMouseButtonDown(rl.MouseLeftButton):
				if selDraggingMove {
					c.MoveFloatingTo(spx-selDragOffX, spy-selDragOffY)
				} else {
					x0, y0, x1, y1 := guiutil.CenterOrCornerBounds(selAnchorX, selAnchorY, spx, spy, altHeld)
					c.SetSelection(x0, y0, x1, y1)
				}
			}
			if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
				// A genuine click (no movement from the anchor) that started
				// as a new-selection attempt while something was already
				// selected is a deselect gesture, not "select this one
				// pixel" — which has no sensible meaning as a user action.
				if !selDraggingMove && selNewAttemptHadPriorSelection && spx == selAnchorX && spy == selAnchorY {
					c.ClearSelection()
				}
				selDragging = false
			}
		}

		// Shape tools: drag from an anchor point to the current cursor
		// position; the live end point is tracked for the preview overlay
		// while dragging, and the shape is only actually drawn into the
		// sprite on release — one Checkpoint per shape, however long the
		// drag. All three reduce to straight-line walks (guiutil.
		// ForEachLinePixel/RectOutline/TriangleOutline), just with different
		// point sequences.
		if shapeToolActive && !modal && overGrid {
			spx := int((float32(mx) - oxF) / cellF)
			spy := int((float32(my) - oyF) / cellF)
			validClick := float32(mx) >= oxF && float32(my) >= oyF

			switch {
			case rl.IsMouseButtonPressed(rl.MouseLeftButton) && validClick:
				shapeDragging = true
				shapeStartX, shapeStartY = spx, spy
				shapeEndX, shapeEndY = spx, spy
			case shapeDragging && rl.IsMouseButtonDown(rl.MouseLeftButton):
				shapeEndX, shapeEndY = spx, spy
			}
			// Polygon side count is adjustable mid-drag: number keys 3-9 set
			// it directly, live-updating both the preview and the eventual
			// commit shape.
			if polygonToolActive && shapeDragging {
				for n := 3; n <= 9; n++ {
					if rl.IsKeyPressed(int32(rl.KeyOne) + int32(n-1)) {
						polygonSides = n
					}
				}
			}
			if shapeDragging && rl.IsMouseButtonReleased(rl.MouseLeftButton) {
				shapeDragging = false
				c.Checkpoint()
				// Ctrl applies attributes instead of the bitmap, matching the
				// paintbrush's own Ctrl-attribute-paint gesture — Ctrl only
				// here, not Alt, since Alt already means something else for
				// these tools (the ellipse's centre/corner toggle, the
				// selection's duplicate/centre-mode).
				shapeAttrPaint := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
				plot := func(x, y int) {
					if x < 0 || y < 0 || x >= s.Width() || y >= s.Height() {
						return
					}
					if shapeAttrPaint {
						c.PaintAttr(x, y)
					} else {
						c.Paint(x, y, true)
					}
				}
				switch {
				case lineToolActive:
					guiutil.ForEachLinePixel(shapeStartX, shapeStartY, shapeEndX, shapeEndY, plot)
				case rectToolActive:
					guiutil.RectOutline(shapeStartX, shapeStartY, shapeEndX, shapeEndY, plot)
				case triangleToolActive:
					guiutil.TriangleOutline(shapeStartX, shapeStartY, shapeEndX, shapeEndY, plot)
				case ellipseToolActive:
					altHeld := rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)
					ex0, ey0, ex1, ey1 := guiutil.CenterOrCornerBounds(shapeStartX, shapeStartY, shapeEndX, shapeEndY, !altHeld)
					guiutil.EllipseOutline(ex0, ey0, ex1, ey1, plot)
				case polygonToolActive:
					guiutil.PolygonOutline(polygonSides, shapeStartX, shapeStartY, shapeEndX, shapeEndY, plot)
				}
			}
		}

		// Text tool: a click starts (or, if one is already active, commits
		// the current entry and starts a new one at the new position —
		// matching the "starting elsewhere finalises the old one" convention
		// already used for selections). Typing is captured every frame while
		// active; Enter commits, Escape discards.
		if textToolActive && !modal {
			if rl.IsMouseButtonPressed(rl.MouseLeftButton) && overGrid {
				tpx := int((float32(mx) - oxF) / cellF)
				tpy := int((float32(my) - oyF) / cellF)
				if float32(mx) >= oxF && float32(my) >= oyF {
					if textState.active {
						commitText(c, textFontsList[textFontIdx].font, &textState)
					}
					ctrlHeldAtStart := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
					textState = textEntry{active: true, x: tpx, y: tpy, desiredCol: -1, attrPaint: ctrlHeldAtStart || attrOnGlobal}
				}
			}
			if textState.active {
				if textState.Update(frameIn, textFontsList[textFontIdx].font) {
					textState = textEntry{}
				}
			}
		} else if textState.active {
			// Switched to a different tool with an entry still pending:
			// commit it, matching the same rule selections and pen moves
			// follow rather than silently discarding typed work.
			commitText(c, textFontsList[textFontIdx].font, &textState)
			textState = textEntry{}
		}

		spectrum := c.Mode() == ui.SpectrumColour

		// Clicking the title block toggles collapse (and consumes the click).
		overTitle := guidraw.RectHit(l.TitleRect, mx, my)
		if !modal && rl.IsMouseButtonPressed(rl.MouseLeftButton) && overTitle {
			if titleCollapseTarget >= 0.5 {
				titleCollapseTarget = 0
			} else {
				titleCollapseTarget = 1
			}
		}

		// Clicking the drawer triangle toggles the bottom button drawer.
		if !modal && rl.IsMouseButtonPressed(rl.MouseLeftButton) && overDrawerToggle {
			toggleHeld = true
			if drawerTarget >= 0.5 {
				drawerTarget = 0
			} else {
				drawerTarget = 1
			}
		}

		// Left-click UI: action buttons, mode buttons, frame strip, +/- frame,
		// palette ink. Skipped when the press lands on the preview (a press-and-
		// hold gesture) or the title block (a collapse toggle).
		// Frame scrubber: press within the track (or drag onto it) selects a frame
		// proportional to the pointer's X; the drag continues until release.
		frameFromScrubX := func(px int) int {
			nf := c.Sprite.FrameCount()
			if nf <= 1 || l.ScrubRect.Width <= 0 {
				return 0
			}
			t := (float32(px) - l.ScrubRect.X) / l.ScrubRect.Width
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			idx := int(t * float32(nf))
			if idx >= nf {
				idx = nf - 1
			}
			return idx
		}
		if !modal && rl.IsMouseButtonPressed(rl.MouseLeftButton) && guidraw.RectHit(l.ScrubRect, mx, my) {
			if textState.active {
				commitText(c, textFontsList[textFontIdx].font, &textState)
				textState = textEntry{}
			}
			scrubbing = true
		}
		if scrubbing {
			if rl.IsMouseButtonDown(rl.MouseLeftButton) {
				c.SelectFrame(frameFromScrubX(mx))
			} else {
				scrubbing = false
			}
		}

		if !modal && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !overPreview && !overToolPalette && !overTitle && !overDrawerToggle && !overGrid && !scrubbing && fontMenu == nil {
			// A pending text entry is committed before any button action —
			// play, save, flip, invert, frame navigation, or anything else —
			// not just when clicking the canvas or switching tools. The user
			// clicking a button is clearly done with the text.
			if textState.active {
				commitText(c, textFontsList[textFontIdx].font, &textState)
				textState = textEntry{}
			}
			vpRight := l.GridX + l.GridW
			for _, b := range l.Buttons {
				// A button that has faded out past the viewport's right edge is not
				// clickable (it is on its way out or already gone).
				if !guiutil.ButtonVisible(b.X, b.W, vpRight) {
					continue
				}
				if b.Hit(mx, my) {
					b.Action()
				}
			}
			for _, b := range l.ModeButtons {
				if b.Hit(mx, my) {
					b.Action()
				}
			}
			// Chequer-toggle LEDs below the two bitmap-mode buttons. Clicking a LED
			// also selects that bitmap mode (as if its button were pressed).
			if guidraw.RectHit(l.ChkLedWhite, mx, my) {
				theme.ChequerOnWhite = !theme.ChequerOnWhite
				c.SetMode(ui.BitmapWhite)
			}
			if guidraw.RectHit(l.ChkLedBlack, mx, my) {
				theme.ChequerOnBlack = !theme.ChequerOnBlack
				c.SetMode(ui.BitmapBlack)
			}
			for _, b := range l.OnionButtons {
				if b.Hit(mx, my) {
					b.Action()
				}
			}
			for i := range l.FrameRects {
				if guidraw.RectHit(l.FrameRects[i], mx, my) {
					c.SelectFrame(i)
					drag.press(i, mx, my)
				}
			}
			if guidraw.RectHit(l.AddFrameRect, mx, my) {
				before := c.Sprite.FrameCount()
				c.Checkpoint()
				c.AddFrame()
				if c.Sprite.FrameCount() > before {
					warpToAdd = true
				}
			}
			if guidraw.RectHit(l.HelpRect, mx, my) {
				help = newHelpModal()
			}
			if attrOnIndicatorRect(l, txt, attrOnGlobal).Contains(mx, my) {
				attrOnGlobal = !attrOnGlobal
			}
		}

		// Palette swatch picks (Spectrum mode only): left-click sets ink,
		// right-click sets paper, both from a single Update call.
		if !modal && spectrum {
			res := palette.Update(frameIn)
			if res.InkPicked {
				c.SetInk(res.Base)
				c.SetBright(res.Bright)
			}
			if res.PaperPicked {
				c.SetPaper(res.Base)
				c.SetBright(res.Bright)
			}
		}

		// Tool palette pick: available in every mode, since tool choice isn't
		// a Spectrum-Colour-specific concept. Switching away from the select
		// tool while something is floating (a pending move or paste) commits
		// it first — matching Photoshop's "choose a different tool" commit
		// trigger, alongside Ctrl/Cmd+D and starting a new selection.
		if !modal && penPanel == nil && fontMenu == nil {
			pick := toolPalette.Update(frameIn)
			if pick.Picked && pick.ID != "select" && c.IsFloating() {
				c.CommitFloatingSelection()
			}
			// Double-clicking the hand tool re-fits the whole sprite to the
			// viewport — the same manual "return to fit" gesture Photoshop
			// uses for its own hand tool, and the only way back to a fitted
			// view once the user has zoomed away from it (animateFit
			// otherwise only runs automatically, once, on a sprite resize).
			if pick.Picked && pick.ID == "hand" {
				now := float32(rl.GetTime())
				if now-lastHandClickTime < doubleClickWindow {
					vp.animateFit(s.Width(), s.Height(), l.GridW, l.GridH, l.Cell)
				}
				lastHandClickTime = now
			}
		}

		// Pen options panel: right-click the paintbrush button to open it,
		// click outside it or press Escape to close. While open, it owns
		// mouse input for shape/size picks; penShapeID/penSize are read back
		// every frame so paintPixel always sees the latest choice.
		if !modal {
			if penPanel != nil {
				penShapeID, penSize = penPanel.Update(frameIn)
				closing := (rl.IsMouseButtonPressed(rl.MouseLeftButton) || rl.IsMouseButtonPressed(rl.MouseRightButton)) &&
					!penPanel.Bounds().Contains(mx, my)
				if closing || rl.IsKeyPressed(rl.KeyEscape) {
					penPanel = nil
				}
			} else if rl.IsMouseButtonPressed(rl.MouseRightButton) {
				if id, ok := toolPalette.HitTest(mx, my); ok && id == "paintbrush" {
					if btnRect, ok := toolPalette.RectFor("paintbrush"); ok {
						// Anchored above the button, not below: the panel's total
						// height is the shape row (32) + gap (8) + size row (24) =
						// 64, matching the layout newPenOptions itself computes.
						anchor := zenui.Rect{X: btnRect.X, Y: btnRect.Y - 64 - 6}
						penPanel = newPenOptions(anchor, penShapeID, penSize)
					}
				}
			}
		}

		// Text font picker: right-click the text tool button to open it.
		// Menu already handles Escape and click-outside as Cancelled
		// internally, so no manual bounds/closing check is needed here,
		// unlike the pen panel above (which uses a plain Rect, not Menu).
		if !modal {
			if fontMenu != nil {
				switch fontMenu.Update(frameIn) {
				case zenui.Accepted:
					textFontIdx = fontMenu.Result()
					fontMenu = nil
				case zenui.Cancelled:
					fontMenu = nil
				}
			} else if rl.IsMouseButtonPressed(rl.MouseRightButton) {
				if id, ok := toolPalette.HitTest(mx, my); ok && id == "text" {
					if btnRect, ok := toolPalette.RectFor("text"); ok {
						fontMenu = newTextFontMenu(btnRect, textFontsList, textFontIdx)
					}
				}
			}
		}

		// Right-click on a frame button opens its context menu (Insert/Duplicate/
		// Copy/Paste/Insert-and-paste/Delete). The frame is selected here, before
		// the menu opens — see applyFrameMenuPick's comment for why that matters.
		if !modal && frameMenu == nil && rl.IsMouseButtonPressed(rl.MouseRightButton) {
			for i := range l.FrameRects {
				if guidraw.RectHit(l.FrameRects[i], mx, my) {
					c.SelectFrame(i)
					fr := l.FrameRects[i]
					anchor := zenui.Rect{X: int(fr.X), Y: int(fr.Y), W: int(fr.Width), H: int(fr.Height)}
					frameMenu = newFrameMenu(c, anchor)
					frameMenuFrame = i
					break
				}
			}
		}

		// Frame drag-to-reorder: advance the pulsate clock and promote to active
		// past the movement threshold while held, then dispatch (or cancel) on
		// release. The press itself was already recorded above, alongside the
		// ordinary click-to-select, so a plain click still works unchanged.
		if drag.source >= 0 {
			if rl.IsMouseButtonDown(rl.MouseLeftButton) {
				drag.update(rl.GetFrameTime(), mx, my)
			}
			if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
				if drag.active {
					inBand := my >= l.FrameStripY && my < l.FrameStripY+frameBtnH
					if inBand && drag.source < len(l.FrameRects) {
						gap := frameDropGap(l.FrameRects, float32(mx))
						c.Checkpoint()
						c.MoveFrame(drag.source, dragGapToMoveTarget(drag.source, gap))
					}
				}
				drag.reset()
			} else if rl.IsKeyPressed(rl.KeyEscape) {
				drag.reset()
			}
		}

		// Keyboard shortcuts. Suppressed while the text tool, a pending
		// selection, or an in-progress polygon drag has exclusive keyboard
		// focus — handleKeys' IsKeyPressed checks run independently of
		// GetCharPressed's character queue, so without this guard they fire
		// simultaneously with typing/confirming/adjusting (Enter both
		// commits text AND toggles play; typing the digit '1' both appends
		// to the string AND selects frame 1; setting a polygon to 3 sides
		// mid-drag would also jump to frame 3).
		if !modal && !textState.active && !wasFloatingThisFrame && !(polygonToolActive && shapeDragging) {
			handleKeys(c)
		}

		// --- draw ---
		rl.BeginDrawing()
		rl.ClearBackground(theme.BG)
		guidraw.DrawUI(txt, c, l, theme, cellF, oxF, oyF, pppMin, pppMax, mx, my, focusX, focusY, btnOpacity)
		drawSelectionOverlay(c, l, oxF, oyF, cellF)
		drawShapePreview(c, l, oxF, oyF, cellF, shapeDragging, shapeStartX, shapeStartY, shapeEndX, shapeEndY, lineToolActive, rectToolActive, triangleToolActive, ellipseToolActive, polygonToolActive, polygonSides)
		drawTextPreview(textFontsList[textFontIdx].font, &textState, int32(l.GridX), int32(l.GridY), int32(l.GridW), int32(l.GridH), oxF, oyF, cellF, previewInkColour(c))
		drawZenuiOverlays(txt, iconTxt, c, l, theme, cellF, paletteFade, palette, preview, toolPalette, penPanel, penSize, attrOnGlobal)

		// Animated OSD caption for the latest status message: rises from the
		// bottom-right corner, past the palette, fading as it goes, with magical
		// pixels scattered around it.
		tw := txt.Measure(guiutil.Upper(c.Status()), osdTextScale)
		th := txt.CellH() * osdTextScale
		osdNote.update(rl.GetFrameTime(), c.StatusSeq(), guiutil.Upper(c.Status()), tw, th)
		osdNote.draw(txt, l.WinW-pad, l.WinH, tw, th)

		// Modal file dialog overlays the whole editor when open.
		files.draw(fpRenderer{txt: txt}, l.WinW, l.WinH)
		if help != nil {
			help.draw(fpRenderer{txt: txt}, l.WinW, l.WinH)
		}
		if reset != nil {
			reset.draw(txt, l.WinW, l.WinH)
		}
		if frameMenu != nil {
			frameMenu.Draw(fpRenderer{txt: txt}, l.WinW, l.WinH, fpTheme())
		}
		if fontMenu != nil {
			fontMenu.Draw(fpRenderer{txt: txt}, l.WinW, l.WinH, fpTheme())
		}
		if drag.active {
			drawFrameDrag(txt, l, drag, mx, my)
		}

		rl.EndDrawing()
	}
}

func handleKeys(c *ui.Controller) {
	// Space is reserved for hand-drag panning; Enter toggles play/stop.
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		c.TogglePlay()
	}
	if rl.IsKeyPressed(rl.KeyLeftBracket) {
		c.PrevFrame()
	}
	if rl.IsKeyPressed(rl.KeyRightBracket) {
		c.NextFrame()
	}
	n := c.Sprite.FrameCount()
	if n > 9 {
		n = 9 // keys 1..9 select the first nine frames
	}
	for k := 0; k < n; k++ {
		if rl.IsKeyPressed(int32(rl.KeyOne) + int32(k)) {
			c.SelectFrame(k)
		}
	}
}

// resizeCells grows or shrinks the sprite by whole character cells (8 px) in
// each axis, non-destructively, clamped to the model's size limits.
func resizeCells(c *ui.Controller, dCols, dRows int) {
	c.Checkpoint()
	w := c.Sprite.Width() + dCols*model.Cell
	h := c.Sprite.Height() + dRows*model.Cell
	c.SetSize(w, h)
}

func drawZenuiOverlays(txt, iconTxt *guidraw.BDFText, c *ui.Controller, l guidraw.Layout, theme guidraw.Theme, cell float32, paletteFade float32, palette *zenui.ZXClassicPaletteChooser, preview *zenui.PreviewPane, toolPalette *zenui.ToolPalette, penPanel *penOptions, penSize int, attrOnGlobal bool) {
	ppp := cell
	zoomPct := guiutil.PPPToPercent(ppp, pppMin, pppMax)

	// Just above the preview pane (2px gap): the preview pane's own scale factor
	// (right-aligned) and the editor viewport's zoom as a percentage (left-aligned)
	// on the fixed 5px=0% .. 160px=800% scale — the same value that drives the
	// grid/overlay fades, so the number read matches their behaviour regardless of
	// sprite size or window size.
	labelY := l.PreviewY - 2 - txt.CellH()
	txt.Draw("X"+guiutil.Itoa(preview.Zoom()), l.PreviewX+l.PreviewW-20, labelY, 1, theme.Dim)
	txt.Draw(guiutil.Itoa(int(zoomPct+0.5))+"%", l.PreviewX, labelY, 1, theme.Dim)

	// Tool palette, tucked between the preview box and the attribute palette.
	// Always visible (unlike the attribute palette, tool choice isn't mode-
	// specific), so no fade-alpha handling is needed here. Drawn through a
	// Renderer over iconTxt (the icon face), not the regular txt — the same
	// fpRenderer adapter, just wrapping a different BDFText instance.
	txt.Draw("TOOLS", l.ToolPaletteX, l.ToolPaletteY-12, 1, theme.Text)
	toolPalette.Draw(fpRenderer{txt: iconTxt}, fpTheme())

	// Brush size badge: a small digit in the paintbrush button's top-left
	// corner, shown only when the selected size is bigger than the default
	// (1) — nothing to indicate otherwise, since size 1 needs no reminder.
	// Drawn directly on the button's own background, with no backing patch
	// behind it — the same approach the attribute palette uses for its I/P
	// ink/paper indicators, rather than a solid rectangle that reads as a
	// censorship bar over the icon underneath.
	if penSize > 1 {
		if btnRect, ok := toolPalette.RectFor("paintbrush"); ok {
			txt.Draw(guiutil.Itoa(penSize), btnRect.X+2, btnRect.Y+1, 1, theme.Text)
		}
	}

	// Pen options panel moved below, after preview.Draw — see that call's
	// own comment for why: two things both claimed to be "drawn last", and
	// since preview.Draw ran second, it was silently painting over this
	// panel every frame once the panel started living above the toolpalette
	// (overlapping the preview box's own screen area). Found via direct
	// pixel dumps showing a uniform, undisturbed background colour where
	// the panel's buttons should have been — not a position or logic bug,
	// a draw-order bug.

	// Attribute palette (Spectrum Colour mode): owned by a
	// zenui.ZXClassicPaletteChooser now. It fades in/out on mode change
	// (paletteFade, passed through as Draw's alpha), so it keeps drawing
	// during the fade-out even after the mode has switched away.
	if paletteFade > 0 {
		pa := uint8(255 * paletteFade)
		tc := theme.Text
		tc.A = pa
		txt.Draw("PALETTE", l.PaletteX, l.PaletteY-12, 1, tc)
		palette.Draw(fpRenderer{txt: txt}, fpTheme(), c.Ink(), c.Paper(), c.Bright(), paletteFade)
	}

	// Global ATTR ON/OFF indicator: always shown, right-aligned above the
	// palette column — reverse video (light fill, dark text) when the
	// Ctrl-double-tap toggle is active, plain normal-video text when it
	// isn't. Drawn independent of paletteFade — the underlying attribute
	// data (and so the toggle's effect) exists regardless of which view
	// mode is currently displayed, not just Spectrum Colour. Also acts as
	// a click target (see the main loop's click handling), so its rect is
	// computed by the same attrOnIndicatorRect helper used there.
	{
		label := attrOnLabel(attrOnGlobal)
		r := attrOnIndicatorRect(l, txt, attrOnGlobal)
		if attrOnGlobal {
			rl.DrawRectangle(int32(r.X), int32(r.Y), int32(r.W), int32(r.H), theme.Text)
			txt.Draw(label, r.X+2, r.Y+1, 1, theme.BG)
		} else {
			txt.Draw(label, r.X+2, r.Y+1, 1, theme.Text)
		}
	}

	// Detail preview box (fixed-size, top-right) and its press-and-hold popup.
	preview.Draw(fpRenderer{txt: txt}, l.WinW, l.WinH, fpTheme())

	// Pen options panel, drawn genuinely last so it can't be painted over by
	// anything else in this function, including the preview box above.
	if penPanel != nil {
		penPanel.Draw(fpRenderer{txt: txt}, fpTheme())
	}
}

// markColour returns a high-contrast text colour for marking a palette swatch:
// black on bright/light colours (yellow, cyan, white, green), white otherwise.
// resetRequested is set by the RESET button and consumed by the main loop, which
// then opens the typed confirmation modal. Package-level for the same reason as
// the chequer state used to be: the button-action closures are built in
// computeLayout and cannot reach per-loop state.
var resetRequested bool

// --- tiny string helpers (the Sinclair face is uppercase-friendly) ----------
