// Command zenimate-gui is the raylib desktop frontend for the ZX Spectrum
// animated sprite editor. It presents the pixel grid, an eight-frame selector, a
// live preview, and action buttons, driving the shared ui.Controller.
//
// All on-screen text is rendered through the bundled Sinclair ZX Spectrum BDF
// font via the bdfText renderer (pkg/bdf -> raylib textures). raylib's own font
// API is never used.
//
// The window needs an OpenGL-capable display, so this binary runs on a desktop
// (it will not run in a headless container). It still compiles everywhere the
// raylib cgo/purego toolchain is available.
package main

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"

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
	previewBox = 168 // fixed detail-preview box size in window pixels

	frameBtnW = 56
	frameBtnH = 28
	frameGap  = 6
	modeBtnH  = 42 // taller row for word-wrapped two-line mode/onion labels

	btnW   = 120
	btnH   = 32
	btnGap = 10
)

// ZX-ish palette.
var (
	colBG       = rl.NewColor(0x10, 0x10, 0x18, 0xff)
	colGridArea = rl.NewColor(0x2c, 0x2c, 0x38, 0xff) // lighter backing behind the sprite grid
	colGrid     = rl.NewColor(0x30, 0x30, 0x40, 0xff)
	colInk      = rl.NewColor(0xff, 0xff, 0xff, 0xff)
	colSel      = rl.NewColor(0x00, 0x80, 0x00, 0xff)
	colBtn      = rl.NewColor(0x28, 0x28, 0x34, 0xff)
	colBtnHot   = rl.NewColor(0x3a, 0x3a, 0x4a, 0xff)
	colYellow   = rl.NewColor(0xff, 0xd0, 0x00, 0xff)
	colText     = rl.NewColor(0xe0, 0xe0, 0xe8, 0xff)
	colDim      = rl.NewColor(0x80, 0x80, 0x90, 0xff) // dimmer subtitle / secondary text
	colGuide    = rl.NewColor(0x40, 0x40, 0x40, 0xff) // dark grey: 8-pixel character-cell guides
	colVPBorder = rl.NewColor(0x6a, 0x6a, 0x78, 0xff) // medium grey: viewport box border
	colPixGrid  = rl.NewColor(0x00, 0x00, 0x00, 0x28) // almost-invisible 1px grid (Spectrum mode)

	// Translucent onion-skin silhouettes: previous frame red, next frame green.
	colOnionPrev = rl.NewColor(0xff, 0x30, 0x30, 0x70)
	colOnionNext = rl.NewColor(0x30, 0xff, 0x30, 0x70)

	// Photoshop-style transparency chequer for empty cells: one square per
	// virtual pixel (8x8 squares per character cell), alternating these greys.
	colChkLight = rl.NewColor(0xb0, 0xb0, 0xb0, 0xff)
	colChkDark  = rl.NewColor(0x88, 0x88, 0x88, 0xff)
)

// zxColor converts a zxpalette colour to a raylib colour.
func zxColor(n color.NRGBA) rl.Color {
	return rl.NewColor(n.R, n.G, n.B, n.A)
}

type button struct {
	x, y, w, h int
	label      string
	action     func()
}

func (b button) hit(mx, my int) bool {
	return mx >= b.x && mx < b.x+b.w && my >= b.y && my < b.y+b.h
}

// rectHit reports whether (mx,my) falls within an rl.Rectangle.
func rectHit(r rl.Rectangle, mx, my int) bool {
	return float32(mx) >= r.X && float32(mx) < r.X+r.Width &&
		float32(my) >= r.Y && float32(my) < r.Y+r.Height
}

// layout holds every interactive region for the current window size, computed
// once per frame so drawing and hit-testing always agree. The window is
// resizable, so all coordinates derive from the live screen dimensions rather
// than fixed constants.
type layout struct {
	winW, winH int

	// Title block (top-left). When collapsed it shrinks to a small button; the
	// toolbars expand to fill the reclaimed width and the viewport gains vertical
	// room. titleRect is the click target that toggles collapse.
	titleRect      rl.Rectangle
	titleCollapsed bool    // true past the halfway point of the collapse
	titleCollapse  float32 // eased collapse progress (0 = expanded, 1 = collapsed)

	// Horizontal frame strip near the top (mirrors the TUI's frame row), one
	// rect per current frame, with +/- buttons to its right.
	frameStripX, frameStripY int
	scrubRect                rl.Rectangle // frame scrubber slider above the strip
	frameRects               []rl.Rectangle
	addFrameRect             rl.Rectangle
	removeFrameRect          rl.Rectangle
	helpRect                 rl.Rectangle

	// Editor grid.
	gridX, gridY int
	gridW, gridH int // the box the grid is clipped to (base fit size)
	cell         int // adaptive on-screen pixel size of one Spectrum cell

	// View-mode buttons (Bitmap White / Bitmap Black / Spectrum Colour).
	modeButtons []button

	// Tiny LED toggles centred below the Bitmap White / Bitmap Black buttons that
	// switch the transparency chequer on/off for that mode.
	chkLedWhite rl.Rectangle
	chkLedBlack rl.Rectangle

	// Onion-skin toggle buttons (bitmap modes only).
	onionButtons []button

	// Attribute palette (shown in Spectrum Colour mode): 16 swatches in a 4x4
	// grid (8 base colours, each as a normal+bright pair). swatchBase/swatchBright
	// map each swatch rect to the colour it represents, so layout and hit-testing
	// stay in sync regardless of the on-screen ordering.
	paletteX, paletteY int
	swatchW, swatchH   int
	swatchRects        [16]rl.Rectangle
	swatchBase         [16]int
	swatchBright       [16]bool

	// Preview (top-right corner) — fixed size, independent of sprite dimensions.
	previewX, previewY int
	previewW, previewH int

	// Action buttons across the bottom, in a sliding drawer.
	buttons         []button
	stripBtnW       int          // effective strip button width (shrinks in narrow windows)
	drawerToggle    rl.Rectangle // small triangle that opens/closes the drawer
	drawerToggleHit rl.Rectangle // larger clickable area around the triangle
	drawerOpen      float32      // 0 = closed, 1 = open (current eased progress)
}

// computeLayout builds the layout for the given window size and controller. The
// button actions are bound to the controller here. The editor cell size adapts
// to the space between the frame strip and the button block so the grid always
// fits, whatever the window size and sprite dimensions.
func computeLayout(w, h int, c *ui.Controller, files *fileOps, titleCollapse float32, drawerOpen float32) layout {
	var l layout
	l.winW, l.winH = w, h
	l.titleCollapse = titleCollapse
	// Past the halfway point the block reads as collapsed (drives the Z vs full
	// text and the height that the header budget uses).
	l.titleCollapsed = titleCollapse >= 0.5
	l.drawerOpen = drawerOpen

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
	if l.titleCollapsed {
		curH = collapsedH
	}
	l.titleRect = rl.NewRectangle(pad, pad, curW, float32(curH))

	// Toolbars begin to the right of the title block and run to the window edge.
	toolX := int(l.titleRect.X+l.titleRect.Width) + 20
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
	l.frameStripX = toolX
	l.frameStripY = row1Y
	l.frameRects = make([]rl.Rectangle, nf)
	for i := 0; i < nf; i++ {
		fx := l.frameStripX + i*(fbw+frameGap)
		l.frameRects[i] = rl.NewRectangle(float32(fx), float32(row1Y),
			float32(fbw), float32(frameBtnH))
	}
	rx := float32(l.frameStripX + nf*(fbw+frameGap))
	l.removeFrameRect = rl.NewRectangle(rx, float32(row1Y), float32(small), float32(frameBtnH))
	l.addFrameRect = rl.NewRectangle(rx+float32(small)+6, float32(row1Y), float32(small), float32(frameBtnH))

	// Scrubber track: a thin slider spanning the frame buttons (not the +/-),
	// sitting just above them. Dragging it selects a frame proportionally.
	scrubW := 0
	if nf > 0 {
		scrubW = nf*fbw + (nf-1)*frameGap
	}
	if scrubW < fbw {
		scrubW = fbw
	}
	l.scrubRect = rl.NewRectangle(float32(l.frameStripX), float32(scrubY),
		float32(scrubW), float32(scrubH))

	// HELP button: fixed at its startup position, directly below where the '-' and
	// '+' buttons sit with the default 8 frames at full width. It deliberately does
	// NOT track the live frame strip: adding frames moves '+'/'-' rightward, but
	// HELP stays put where it was on startup. Width spans the '-'-to-'+' gap
	// (2*small + 6). Height matches the onion/mode buttons.
	helpAnchorX := float32(l.frameStripX + model.DefaultFrames*(frameBtnW+frameGap))
	helpW := 2*float32(small) + 6
	l.helpRect = rl.NewRectangle(helpAnchorX, float32(row1Y)+float32(frameBtnH)+6,
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
	l.stripBtnW = ebtnW // remembered so draw can pick 1- vs 2-line labels

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
	l.buttons = []button{
		{row(0), byMid, rcW, btnH, "RESET", func() { resetRequested = true }},
		{row(0) + rcW + cpInnerGap, byMid, rcW, btnH, "CLS", c.ClearFrameCLS},
		{row(1), byMid, ebtnW, btnH, "Play/Stop", c.TogglePlay},
		{cpLeftX, byMid, cpW, btnH, "Copy", c.CopyFrame},
		{cpRightX, byMid, cpW, btnH, "Paste", c.PasteFrame},
		{row(0), byBottom, ebtnW, btnH, "Open", func() { files.startOpen(c) }},
		{row(1), byBottom, ebtnW, btnH, "Save", func() { files.save(c) }},
		{cpLeftX, byBottom, cpW, btnH, "Export", func() { files.startExport(c) }},
		{cpRightX, byBottom, cpW, btnH, "Bundle", func() { files.startBundleExport(c) }},
	}

	// Sizing area, occupying the old column-3 slot onward. Left sub-column holds
	// two preset-size buttons stacked (32x24 = full screen, 2x2 = smallest);
	// immediately to its right is the 2x2 block of per-cell width/height steppers.
	// All these buttons share the compact width szW and the strip button height.
	const szInnerGap = 4
	szW := (ebtnW - szInnerGap) / 2
	presetX := row(3)
	szX := presetX + szW + szInnerGap // steppers sit right of the presets
	l.buttons = append(l.buttons,
		// Preset sizes (left sub-column).
		button{presetX, byMid, szW, btnH, "32x24", func() { c.SetSize(model.MaxWidth, model.MaxHeight) }},
		button{presetX, byBottom, szW, btnH, "2x2", func() { c.SetSize(2*model.Cell, 2*model.Cell) }},
		// Per-cell width/height steppers (right 2x2 block); labels in cell units.
		button{szX, byMid, szW, btnH, "W -1", func() { resizeCells(c, -1, 0) }},
		button{szX + szW + szInnerGap, byMid, szW, btnH, "W +1", func() { resizeCells(c, +1, 0) }},
		button{szX, byBottom, szW, btnH, "H -1", func() { resizeCells(c, 0, -1) }},
		button{szX + szW + szInnerGap, byBottom, szW, btnH, "H +1", func() { resizeCells(c, 0, +1) }},
	)

	// Transform group: a 2x2 block of small buttons to the right of the steppers.
	// H FLIP / V FLIP on the top row, ROT 90 / INVERT on the bottom. ROT 90
	// rotates in place; holding Ctrl also resizes a non-square frame so nothing is
	// clipped.
	txX := szX + 2*szW + szInnerGap + btnGap // one gap past the stepper block
	l.buttons = append(l.buttons,
		button{txX, byMid, szW, btnH, "H FLIP", c.FlipH},
		button{txX + szW + szInnerGap, byMid, szW, btnH, "V FLIP", c.FlipV},
		button{txX, byBottom, szW, btnH, "ROT 90", func() {
			ctrl := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
			c.Rotate90(ctrl)
		}},
		button{txX + szW + szInnerGap, byBottom, szW, btnH, "INVERT", c.Invert},
	)

	// Row 2: view-mode buttons then onion toggles, also starting at toolX. These
	// buttons are narrow with word-wrapped two-line labels, so the row is taller
	// than a normal button row.
	modeY := row2Y
	modeW := 84
	mrow := func(i int) int { return toolX + i*(modeW+btnGap) }
	l.modeButtons = []button{
		{mrow(0), modeY, modeW, modeBtnH, "Bitmap White", func() { c.SetMode(ui.BitmapWhite) }},
		{mrow(1), modeY, modeW, modeBtnH, "Bitmap Black", func() { c.SetMode(ui.BitmapBlack) }},
		{mrow(2), modeY, modeW, modeBtnH, "Spectrum Colour", func() { c.SetMode(ui.SpectrumColour) }},
	}

	// Tiny chequer-toggle LEDs centred below the Bitmap White / Bitmap Black
	// buttons.
	const ledW, ledH, ledGap = 10, 6, 3
	ledY := float32(modeY + modeBtnH + ledGap)
	l.chkLedWhite = rl.NewRectangle(float32(mrow(0)+(modeW-ledW)/2), ledY, ledW, ledH)
	l.chkLedBlack = rl.NewRectangle(float32(mrow(1)+(modeW-ledW)/2), ledY, ledW, ledH)

	// Onion-skin toggle buttons, fixed at their startup position: aligned to where
	// the F6 frame button sits with the default frame count at full button width.
	// Like the HELP button, they deliberately do NOT track the live frame strip, so
	// adding frames (which shrinks the buttons and shifts F6) leaves them put.
	onionW := 72
	ox0 := l.frameStripX + 5*(frameBtnW+frameGap)
	l.onionButtons = []button{
		{ox0, modeY, onionW, modeBtnH, "Onion Prev", c.ToggleOnionPrev},
		{ox0 + onionW + btnGap, modeY, onionW, modeBtnH, "Onion Next", c.ToggleOnionNext},
	}

	// The header occupies the taller of the title block and the two toolbar rows;
	// the grid starts just beneath it. Collapsing the title lets the toolbars
	// move up, shrinking the header and giving the viewport more vertical room.
	toolBottom := row2Y + modeBtnH
	headerBottom := toolBottom
	if tb := int(l.titleRect.Y + l.titleRect.Height); tb > headerBottom {
		headerBottom = tb
	}

	// Editor grid sits below the header; its cell size adapts so the whole grid
	// fits the box between the header and the button block, and within the
	// horizontal space left of the fixed preview/palette column.
	l.gridX = pad
	l.gridY = headerBottom + 16

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
	availH := vpBottom - l.gridY // room above the drawer

	// Base cell size is FIXED at cellPx and never derived from window or sprite
	// size. This is the Quag lesson: there is one persistent scale. The on-screen
	// size of a virtual pixel is cellPx * v.zoom, where v.zoom is changed only by
	// the user (wheel) or explicit fit — a window resize never alters it (it only
	// re-pans to keep the view centred). Fitting a large sprite into the box is
	// done by lowering v.zoom via animateFit, not by shrinking this base, so the
	// grid/overlay thresholds (stated in device px per virtual pixel) mean the
	// same thing in every window size and for every sprite.
	l.cell = cellPx

	// The grid is clipped to the available box (not the sprite-exact size), so
	// zooming/panning stays within a stable rectangle and never overdraws the
	// buttons or preview.
	l.gridW = availW
	l.gridH = availH
	if l.gridW < 0 {
		l.gridW = 0
	}
	if l.gridH < 0 {
		l.gridH = 0
	}

	// Drawer-toggle triangle, inside the viewport's bottom-right corner (a few px
	// in from the border so it reads as part of the viewport). The clickable hit
	// area is larger than the triangle itself: it spans from a small margin above/
	// left of the triangle out to the viewport's bottom-right edge, so a click
	// anywhere on or around the toggle activates it (and never paints).
	const triInset = 4
	const triPad = 6 // extra clickable margin around the triangle
	triX := l.gridX + l.gridW - triW - triInset
	triY := l.gridY + l.gridH - triH - triInset
	l.drawerToggle = rl.NewRectangle(float32(triX), float32(triY), float32(triW), float32(triH))
	// Hit area: from (triX-triPad, triY-triPad) to the viewport's bottom-right.
	hx := triX - triPad
	hy := triY - triPad
	l.drawerToggleHit = rl.NewRectangle(float32(hx), float32(hy),
		float32(l.gridX+l.gridW-hx), float32(l.gridY+l.gridH-hy))

	// Attribute palette: 16 swatches in a 4x4 grid, bottom-aligned to the window
	// with the same margin as the bottom button strip. Each base colour is a
	// normal+bright pair; the 8 base colours are laid out two-per-row in the order
	// blue,black / red,magenta / green,cyan / yellow,white (matching the classic
	// Spectrum colour-key arrangement).
	l.swatchW = 36
	l.swatchH = 24
	const swGapX = 6
	const swGapY = 5
	paletteRows := 4
	paletteCols := 4
	paletteH := paletteRows*l.swatchH + (paletteRows-1)*swGapY
	paletteW := paletteCols*l.swatchW + (paletteCols-1)*swGapX
	l.paletteX = w - pad - paletteW
	l.paletteY = (h - pad) - paletteH // bottom edge aligns with the button strip's

	baseOrder := [8]int{
		zxpalette.Blue, zxpalette.Black,
		zxpalette.Red, zxpalette.Magenta,
		zxpalette.Green, zxpalette.Cyan,
		zxpalette.Yellow, zxpalette.White,
	}
	for i := 0; i < 16; i++ {
		pair := i / 2       // 0..7 base-colour slot
		within := i % 2     // 0 = normal, 1 = bright
		pairRow := pair / 2 // 0..3
		pairCol := pair % 2 // 0..1
		gridCol := pairCol*2 + within
		x := l.paletteX + gridCol*(l.swatchW+swGapX)
		y := l.paletteY + pairRow*(l.swatchH+swGapY)
		l.swatchRects[i] = rl.NewRectangle(float32(x), float32(y),
			float32(l.swatchW), float32(l.swatchH))
		l.swatchBase[i] = baseOrder[pair]
		l.swatchBright[i] = within == 1
	}

	// Fixed-size detail preview box in the top-right corner. It grows downward to
	// meet just above the palette (with a small gap), reclaiming the vertical
	// space the old fixed-height preview left empty.
	l.previewX = w - pad - previewBox
	l.previewY = l.gridY
	l.previewW = previewBox
	l.previewH = l.paletteY - 16 - l.previewY
	if l.previewH < previewBox {
		l.previewH = previewBox // never smaller than the base box
	}

	return l
}

func main() {
	font, err := fonts.Sinclair()
	if err != nil {
		panic(err)
	}

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

	txt := newBDFText(font, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	defer txt.Unload()

	c := ui.New(16, 16)
	vp := newViewport()
	osdNote := newOSD()
	var files fileOps       // modal file dialog (save/open/export)
	var help *helpModal     // scrollable help reader, when open
	var reset *resetConfirm // typed reset confirmation, when open
	var frameMenu *zenui.Menu // frame-strip right-click context menu, when open
	frameMenuFrame := 0        // which frame index frameMenu was opened for
	drag := newFrameDrag()     // frame-strip drag-to-reorder state

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

	// Press-and-hold full-preview popup progress (0 = closed, 1 = full overlay),
	// eased toward its target each frame.
	var popup float32
	var previewHeld bool
	// Title collapse: eased progress (0 = expanded, 1 = collapsed) toward a target.
	titleCollapse := float32(0)
	titleCollapseTarget := float32(0)

	// Shift axis-lock for straight-line drawing: the stroke's anchor pixel (mouse-
	// down point) and the locked axis, decided once per stroke and held.
	strokeActive := false
	strokeAnchorX, strokeAnchorY := 0, 0
	strokeAxis := axisNone
	// Last pixel painted this stroke, used to interpolate a continuous line when
	// the pointer moves faster than one pixel per frame (otherwise fast strokes
	// leave sparse dotted gaps).
	lastPaintX, lastPaintY := 0, 0

	// When a press lands on the drawer toggle, painting is suppressed for the
	// entire hold — otherwise the viewport growing as the drawer closes would
	// slide paintable area under a still-held cursor and draw a stray pixel.
	toggleHeld := false
	previewZoom := 2 // fixed detail-preview zoom factor (1..4), right-click cycles

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

		// Re-anchor the zoom scale if the window has moved to a different monitor
		// (e.g. dragged to a display of a different height).
		if m := rl.GetCurrentMonitor(); m != curMonitor {
			curMonitor = m
			setZoomRangeForScreen(rl.GetMonitorHeight(m))
		}

		// Recompute layout from the live (resizable) window each frame.
		l := computeLayout(rl.GetScreenWidth(), rl.GetScreenHeight(), c, &files, titleCollapse, drawerOpen)
		s := c.Sprite

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
		vpRight := l.gridX + l.gridW
		if len(btnOpacity) != len(l.buttons) {
			// Button set changed (or first frame): (re)initialise at the target.
			btnOpacity = make([]float32, len(l.buttons))
			for i, b := range l.buttons {
				if buttonVisible(b.x, b.w, vpRight) {
					btnOpacity[i] = 1
				}
			}
		}
		const btnFadeRate = 10 // higher = quicker fade
		k := clamp32(btnFadeRate*dt, 0, 1)
		for i, b := range l.buttons {
			target := float32(0)
			if buttonVisible(b.x, b.w, vpRight) {
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
			pointerTargetX = l.addFrameRect.X + l.addFrameRect.Width/2
			pointerTargetY = l.addFrameRect.Y + l.addFrameRect.Height/2
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
			vp.animateFit(s.Width(), s.Height(), l.gridW, l.gridH, l.cell)
			lastW, lastH = s.Width(), s.Height()
		}

		// Is the cursor over the editor grid box? (Gates wheel zoom and paint.)
		overGrid := mx >= l.gridX && mx < l.gridX+l.gridW &&
			my >= l.gridY && my < l.gridY+l.gridH

		// The help reader, when open, captures all input ahead of everything else.
		// The editor still renders underneath (drawn below), with the help overlay
		// on top.
		helpOpen := help != nil
		if helpOpen {
			if !help.update(fpInput()) {
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
			switch reset.update() {
			case resetConfirmConfirmed:
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
			switch frameMenu.Update(fpInput()) {
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
		if helpOpen || resetOpen || frameMenuOpen {
			// help/reset/frameMenu already updated above; suppress other interaction
		} else if files.active() {
			files.update(fpInput())
		} else {
			// File shortcuts: Ctrl+S save, Ctrl+O open, Ctrl+F toggle save form.
			ctrl := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
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

		// Pan gesture: either space + left-drag (space acts as a modifier, like
		// Shift), or a middle-mouse-button drag on its own. Releasing the relevant
		// button (or space) ends the pan. The middle button needs no modifier.
		spaceHeld := rl.IsKeyDown(rl.KeySpace) && !modal
		middlePan := rl.IsMouseButtonDown(rl.MouseButtonMiddle) && !modal
		panning := (spaceHeld && rl.IsMouseButtonDown(rl.MouseLeftButton)) || middlePan

		// Update pan/zoom (wheel zoom anchored at cursor, drag pan, inertia). While
		// a modal dialog is open the viewport ignores the wheel and pan entirely.
		vp.update(mx, my, l.gridX, l.gridY, l.cell, panning, overGrid && !modal)

		cellF := vp.cellF(l.cell)
		oxF, oyF := vp.originF(l.gridX, l.gridY)

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

		// Press-and-hold the preview box to grow the full-preview popup. The
		// press must begin over the box; holding keeps it open, release closes it.
		overPreview := mx >= l.previewX && mx < l.previewX+l.previewW &&
			my >= l.previewY && my < l.previewY+l.previewH
		if !modal && rl.IsMouseButtonPressed(rl.MouseLeftButton) && overPreview {
			previewHeld = true
		}
		if !rl.IsMouseButtonDown(rl.MouseLeftButton) {
			previewHeld = false
		}
		// Right-click on the preview cycles the fixed detail zoom (1 -> 2 -> 3 ->
		// 4 -> 1).
		if !modal && rl.IsMouseButtonPressed(rl.MouseRightButton) && overPreview {
			previewZoom = previewZoom%4 + 1
		}
		popupTarget := float32(0)
		if previewHeld {
			popupTarget = 1
		}
		// Ease in/out toward the target.
		popup += (popupTarget - popup) * clamp32(10*dt, 0, 1)
		if popup < 0.001 {
			popup = 0
		}

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
		attrPaint := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)

		// The drawer triangle sits inside the viewport's bottom-right corner, so
		// painting must be suppressed over its (enlarged) hit area; the click
		// toggles the drawer instead.
		overDrawerToggle := rectHit(l.drawerToggleHit, mx, my)

		shiftHeld := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

		// A mouse release ends any in-progress drawer-toggle hold.
		if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			toggleHeld = false
		}

		// Editor grid interaction. Suppressed while the pan gesture is active so a
		// pan-drag never paints, over the drawer toggle, and for the remainder of
		// a hold that began on the drawer toggle.
		painting := !modal && !panning && overGrid && !overDrawerToggle && !toggleHeld &&
			(rl.IsMouseButtonDown(rl.MouseLeftButton) || rl.IsMouseButtonDown(rl.MouseRightButton))
		if painting {
			px := int((float32(mx) - oxF) / cellF)
			py := int((float32(my) - oyF) / cellF)

			// Stroke bookkeeping for Shift axis-lock: the anchor is the pixel where
			// the stroke began; the locked axis is decided on the first move away
			// from it and kept for the rest of the stroke.
			if !strokeActive {
				strokeActive = true
				strokeAnchorX, strokeAnchorY = px, py
				strokeAxis = axisNone
				lastPaintX, lastPaintY = px, py
			}
			if shiftHeld {
				px, py, strokeAxis = lockAxis(strokeAnchorX, strokeAnchorY, px, py, strokeAxis)
			} else {
				// Shift released mid-stroke: stop locking, but a later Shift press in
				// the same stroke re-decides from the anchor.
				strokeAxis = axisNone
			}

			if float32(mx) >= oxF && float32(my) >= oyF {
				// Paint one pixel, bounds-checked, using the current tool mode.
				paintPixel := func(x, y int) {
					if x < 0 || y < 0 || x >= s.Width() || y >= s.Height() {
						return
					}
					switch {
					case attrPaint && !spaceHeld:
						c.PaintAttr(x, y)
					case rl.IsMouseButtonDown(rl.MouseLeftButton):
						c.Paint(x, y, true)
					default:
						c.Paint(x, y, false)
					}
				}
				// Interpolate a straight line from the last painted pixel to the
				// current one so a fast stroke stays continuous instead of dotted.
				forEachLinePixel(lastPaintX, lastPaintY, px, py, paintPixel)
				lastPaintX, lastPaintY = px, py
			}
		} else {
			strokeActive = false
			strokeAxis = axisNone
		}

		spectrum := c.Mode() == ui.SpectrumColour

		// Clicking the title block toggles collapse (and consumes the click).
		overTitle := rectHit(l.titleRect, mx, my)
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
			if nf <= 1 || l.scrubRect.Width <= 0 {
				return 0
			}
			t := (float32(px) - l.scrubRect.X) / l.scrubRect.Width
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
		if !modal && rl.IsMouseButtonPressed(rl.MouseLeftButton) && rectHit(l.scrubRect, mx, my) {
			scrubbing = true
		}
		if scrubbing {
			if rl.IsMouseButtonDown(rl.MouseLeftButton) {
				c.SelectFrame(frameFromScrubX(mx))
			} else {
				scrubbing = false
			}
		}

		if !modal && rl.IsMouseButtonPressed(rl.MouseLeftButton) && !overPreview && !overTitle && !overDrawerToggle && !scrubbing {
			vpRight := l.gridX + l.gridW
			for _, b := range l.buttons {
				// A button that has faded out past the viewport's right edge is not
				// clickable (it is on its way out or already gone).
				if !buttonVisible(b.x, b.w, vpRight) {
					continue
				}
				if b.hit(mx, my) {
					b.action()
				}
			}
			for _, b := range l.modeButtons {
				if b.hit(mx, my) {
					b.action()
				}
			}
			// Chequer-toggle LEDs below the two bitmap-mode buttons. Clicking a LED
			// also selects that bitmap mode (as if its button were pressed).
			if rectHit(l.chkLedWhite, mx, my) {
				chequerOnWhite = !chequerOnWhite
				c.SetMode(ui.BitmapWhite)
			}
			if rectHit(l.chkLedBlack, mx, my) {
				chequerOnBlack = !chequerOnBlack
				c.SetMode(ui.BitmapBlack)
			}
			for _, b := range l.onionButtons {
				if b.hit(mx, my) {
					b.action()
				}
			}
			for i := range l.frameRects {
				if rectHit(l.frameRects[i], mx, my) {
					c.SelectFrame(i)
					drag.press(i, mx, my)
				}
			}
			if rectHit(l.addFrameRect, mx, my) {
				before := c.Sprite.FrameCount()
				c.AddFrame()
				if c.Sprite.FrameCount() > before {
					warpToAdd = true
				}
			}
			if rectHit(l.removeFrameRect, mx, my) {
				c.RemoveFrame()
			}
			if rectHit(l.helpRect, mx, my) {
				help = newHelpModal()
			}
			if spectrum {
				for i := 0; i < 16; i++ {
					if rectHit(l.swatchRects[i], mx, my) {
						c.SetInk(l.swatchBase[i])
						c.SetBright(l.swatchBright[i])
					}
				}
			}
		}

		// Right-click on a palette swatch selects paper (Spectrum mode only).
		if !modal && spectrum && rl.IsMouseButtonPressed(rl.MouseRightButton) {
			for i := 0; i < 16; i++ {
				if rectHit(l.swatchRects[i], mx, my) {
					c.SetPaper(l.swatchBase[i])
					c.SetBright(l.swatchBright[i])
				}
			}
		}

		// Right-click on a frame button opens its context menu (Insert/Duplicate/
		// Copy/Paste/Insert-and-paste/Delete). The frame is selected here, before
		// the menu opens — see applyFrameMenuPick's comment for why that matters.
		if !modal && frameMenu == nil && rl.IsMouseButtonPressed(rl.MouseRightButton) {
			for i := range l.frameRects {
				if rectHit(l.frameRects[i], mx, my) {
					c.SelectFrame(i)
					fr := l.frameRects[i]
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
					inBand := my >= l.frameStripY && my < l.frameStripY+frameBtnH
					if inBand && drag.source < len(l.frameRects) {
						gap := frameDropGap(l.frameRects, float32(mx))
						c.MoveFrame(drag.source, dragGapToMoveTarget(drag.source, gap))
					}
				}
				drag.reset()
			} else if rl.IsKeyPressed(rl.KeyEscape) {
				drag.reset()
			}
		}

		// Keyboard shortcuts mirror the TUI where sensible.
		if !modal {
			handleKeys(c)
		}

		// --- draw ---
		rl.BeginDrawing()
		rl.ClearBackground(colBG)
		drawUI(txt, c, l, cellF, oxF, oyF, mx, my, focusX, focusY, popup, previewZoom, btnOpacity, paletteFade)

		// Animated OSD caption for the latest status message: rises from the
		// bottom-right corner, past the palette, fading as it goes, with magical
		// pixels scattered around it.
		tw := txt.Measure(upper(c.Status()), osdTextScale)
		th := txt.CellH() * osdTextScale
		osdNote.update(rl.GetFrameTime(), c.StatusSeq(), upper(c.Status()), tw, th)
		osdNote.draw(txt, l.winW-pad, l.winH, tw, th)

		// Modal file dialog overlays the whole editor when open.
		files.draw(fpRenderer{txt: txt}, l.winW, l.winH)
		if help != nil {
			help.draw(fpRenderer{txt: txt}, l.winW, l.winH)
		}
		if reset != nil {
			reset.draw(txt, l.winW, l.winH)
		}
		if frameMenu != nil {
			frameMenu.Draw(fpRenderer{txt: txt}, l.winW, l.winH, fpTheme())
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
	w := c.Sprite.Width() + dCols*model.Cell
	h := c.Sprite.Height() + dRows*model.Cell
	c.SetSize(w, h)
}

func drawUI(txt *bdfText, c *ui.Controller, l layout, cell, ox, oy float32, mx, my, focusX, focusY int, popup float32, previewZoom int, btnOpacity []float32, paletteFade float32) {
	s := c.Sprite

	// Title block. Expanded: ZENIMATE large, dimmer subtitle, then the size/frame
	// header. Collapsed: a small button that restores the title on click. The
	// whole block is the click target that toggles collapse.
	if l.titleCollapsed {
		r := l.titleRect
		bc := colBtn
		if rectHit(r, mx, my) {
			bc = colBtnHot
		}
		rl.DrawRectangleRec(r, bc)
		rl.DrawRectangleLinesEx(r, 1, colGrid)
		const zscale = 2
		zw := txt.Measure("Z", zscale)
		zh := txt.CellH() * zscale
		txt.Draw("Z", int(r.X)+(int(r.Width)-zw)/2, int(r.Y)+(int(r.Height)-zh)/2, zscale, colYellow)
	} else {
		txt.Draw("ZENIMATE", pad, pad, 3, colYellow)
		txt.Draw("ZX SPECTRUM, PAINT AND ANIMATE", pad, pad+30, 1, colDim)
		header := "SIZE " + itoa(s.Width()) + "X" + itoa(s.Height()) +
			"  FRAME " + itoa(s.Selected()+1) + "/" + itoa(s.FrameCount())
		txt.Draw(header, pad, pad+50, 1, colText)
		// Source line: which file/bundle the sprite is from (or "unsaved").
		// Long labels are truncated to 30 characters followed by an ellipsis.
		txt.Draw(upper(truncateLabel(c.SourceLabel(), 30)), pad, pad+64, 1, colDim)
	}

	// Horizontal frame strip near the top (mirrors the TUI's frame row). When any
	// label is three characters (F10 and beyond) the font is reduced for the
	// whole strip so the wider labels fit; otherwise the larger size is used.
	// Either way each label is centred in its button.
	labelScale := 2
	if s.FrameCount() >= 10 {
		labelScale = 1
	}
	// Frame scrubber slider above the buttons: a thin track with a small square
	// indicator marking the current frame. The square sits on the slider's
	// baseline and rises a few pixels above its top; drag it to move between
	// frames.
	if l.scrubRect.Width > 0 {
		rl.DrawRectangleRec(l.scrubRect, colBtn)
		rl.DrawRectangleLinesEx(l.scrubRect, 1, colGrid)
		nf := s.FrameCount()
		if nf > 0 {
			const overhang = 4 // pixels the square rises above the slider top
			side := l.scrubRect.Height + overhang
			// Centre the square over the current frame's slot.
			slot := l.scrubRect.Width / float32(nf)
			cx := l.scrubRect.X + (float32(s.Selected())+0.5)*slot
			sqX := cx - side/2
			// Keep the square within the slider's horizontal span.
			if sqX < l.scrubRect.X {
				sqX = l.scrubRect.X
			}
			if sqX+side > l.scrubRect.X+l.scrubRect.Width {
				sqX = l.scrubRect.X + l.scrubRect.Width - side
			}
			// Bottom-aligned to the slider baseline, extending upward by overhang.
			sqY := l.scrubRect.Y + l.scrubRect.Height - side
			rl.DrawRectangleRec(rl.NewRectangle(sqX, sqY, side, side), colSel)
		}
	}
	for i := range l.frameRects {
		r := l.frameRects[i]
		fillc := colBtn
		if i == s.Selected() {
			fillc = colSel
		} else if rectHit(r, mx, my) {
			fillc = colBtnHot
		}
		rl.DrawRectangle(int32(r.X), int32(r.Y), int32(r.Width), int32(r.Height), fillc)
		rl.DrawRectangleLines(int32(r.X), int32(r.Y), int32(r.Width), int32(r.Height), colGrid)
		label := "F" + itoa(i+1)
		lw := txt.Measure(label, labelScale)
		lh := txt.CellH() * labelScale
		txt.Draw(label, int(r.X)+(int(r.Width)-lw)/2, int(r.Y)+(int(r.Height)-lh)/2, labelScale, colText)
	}

	// Frame +/- buttons to the right of the strip.
	drawSmallBtn := func(r rl.Rectangle, label string, enabled bool) {
		bc := colBtn
		if !enabled {
			bc = colBG
		} else if rectHit(r, mx, my) {
			bc = colBtnHot
		}
		rl.DrawRectangleRec(r, bc)
		rl.DrawRectangleLinesEx(r, 1, colGrid)
		lc := colText
		if !enabled {
			lc = colDim
		}
		// Smaller symbol (scale 1) centred in the button box.
		const gscale = 1
		gw := txt.Measure(label, gscale)
		gh := txt.CellH() * gscale
		txt.Draw(label, int(r.X)+(int(r.Width)-gw)/2, int(r.Y)+(int(r.Height)-gh)/2, gscale, lc)
	}
	drawSmallBtn(l.removeFrameRect, "-", s.FrameCount() > model.MinFrames)
	drawSmallBtn(l.addFrameRect, "+", s.FrameCount() < model.MaxFrames)
	drawSmallBtn(l.helpRect, "HELP", true)

	// View-mode buttons; the active mode is highlighted.
	modeNames := []ui.ViewMode{ui.BitmapWhite, ui.BitmapBlack, ui.SpectrumColour}
	for i, b := range l.modeButtons {
		bc := colBtn
		if c.Mode() == modeNames[i] {
			bc = colSel
		} else if b.hit(mx, my) {
			bc = colBtnHot
		}
		rl.DrawRectangle(int32(b.x), int32(b.y), int32(b.w), int32(b.h), bc)
		rl.DrawRectangleLines(int32(b.x), int32(b.y), int32(b.w), int32(b.h), colGrid)
		drawWrappedLabel(txt, b, colText)
	}

	// Chequer-toggle LEDs below the two bitmap-mode buttons: lit (green) when the
	// chequer is on for that mode, dark when off.
	drawLed := func(r rl.Rectangle, on bool) {
		fill := colBtn // dark when off
		if on {
			fill = colSel // green when on
		}
		rl.DrawRectangleRec(r, fill)
		rl.DrawRectangleLinesEx(r, 1, colGrid)
	}
	drawLed(l.chkLedWhite, chequerOnWhite)
	drawLed(l.chkLedBlack, chequerOnBlack)

	// Onion-skin toggle buttons: tinted to their silhouette colour when active.
	// Dimmed in Spectrum Colour mode, where onion skins are not shown.
	onionActive := c.Mode() != ui.SpectrumColour
	onionStates := []bool{c.OnionPrev(), c.OnionNext()}
	onionTints := []rl.Color{
		rl.NewColor(0x80, 0x20, 0x20, 0xff),
		rl.NewColor(0x20, 0x80, 0x20, 0xff),
	}
	for i, b := range l.onionButtons {
		bc := colBtn
		if onionStates[i] {
			bc = onionTints[i]
		} else if b.hit(mx, my) {
			bc = colBtnHot
		}
		rl.DrawRectangle(int32(b.x), int32(b.y), int32(b.w), int32(b.h), bc)
		rl.DrawRectangleLines(int32(b.x), int32(b.y), int32(b.w), int32(b.h), colGrid)
		lc := colText
		if !onionActive {
			lc = colDim
		}
		drawWrappedLabel(txt, b, lc)
	}

	// Editor grid, pan/zoom transformed with FRACTIONAL cell/origin so zoom is
	// smooth (no integer snapping). Clipped to its layout box with a scissor so a
	// panned/zoomed sprite never overdraws the surrounding UI. Each cell spans
	// [ox+x*cell, ox+(x+1)*cell] exactly, so cells tile seamlessly at any scale.
	sw, sh := s.Width(), s.Height()
	gw := float32(sw) * cell
	gh := float32(sh) * cell
	boxX, boxY := float32(l.gridX), float32(l.gridY)
	boxR, boxB := boxX+float32(l.gridW), boxY+float32(l.gridH)

	rl.BeginScissorMode(int32(l.gridX), int32(l.gridY), int32(l.gridW), int32(l.gridH))

	// Lighter backing so the grid box reads clearly even when the sprite is
	// panned away.
	rl.DrawRectangle(int32(l.gridX), int32(l.gridY), int32(l.gridW), int32(l.gridH), colGridArea)

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
				rl.DrawRectangleRec(rect, zxColor(zxpalette.RGBA[idx]))
			case ui.BitmapWhite:
				if on {
					rl.DrawRectangleRec(rect, colInk) // white
				} else {
					drawCheckerPixel(rx0, ry0, rx1-rx0, ry1-ry0, x, y, ui.BitmapWhite, chequerOnWhite)
				}
			default: // BitmapBlack
				if on {
					rl.DrawRectangleRec(rect, rl.Black)
				} else {
					drawCheckerPixel(rx0, ry0, rx1-rx0, ry1-ry0, x, y, ui.BitmapBlack, chequerOnBlack)
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
					if !fr[y*sw+x] {
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
			drawOnion(c.PrevFrameIndex(), colOnionPrev)
		}
		if c.OnionNext() {
			drawOnion(c.NextFrameIndex(), colOnionNext)
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
	zoomPct := pppToPercent(ppp)

	// In Spectrum Colour mode, overlay an almost-invisible 1px-resolution grid so
	// individual virtual pixels are discernible. Fades in between the pixGrid
	// thresholds (device px per virtual pixel).
	if mode == ui.SpectrumColour {
		if pf := pixGridFade(zoomPct); pf > 0 {
			pc := colPixGrid
			pc.A = uint8(float32(colPixGrid.A) * pf)
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
		if ff := flatCellFade(zoomPct); ff > 0 {
			for cy := 0; cy < s.AttrRows(); cy++ {
				for cx := 0; cx < s.AttrCols(); cx++ {
					attr := s.AttrCell(cx, cy)
					if zxpalette.Ink(attr) != zxpalette.Paper(attr) {
						continue // not a flat cell — pixels are already visible
					}
					// Stroke each set pixel in this cell. Contrast against the cell's
					// (single) colour using the same black/white chooser as the marks.
					strokeC := markColour(zxpalette.Ink(attr))
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
	if cf := cellGuideFade(zoomPct); cf > 0 {
		gc := colGuide
		gc.A = uint8(float32(colGuide.A) * cf)
		for x := 8; x < sw; x += 8 {
			rl.DrawRectangleRec(rl.NewRectangle(ox+float32(x)*cell, oy, 1, gh), gc)
		}
		for y := 8; y < sh; y += 8 {
			rl.DrawRectangleRec(rl.NewRectangle(ox, oy+float32(y)*cell, gw, 1), gc)
		}
	}
	// Outer border around the sprite (always drawn, at full strength).
	rl.DrawRectangleLinesEx(rl.NewRectangle(ox, oy, gw, gh), 1, colGuide)

	rl.EndScissorMode()

	// Medium-grey border around the viewport box itself, drawn outside the
	// scissor so it is never clipped and stays fixed regardless of pan/zoom.
	rl.DrawRectangleLinesEx(rl.NewRectangle(float32(l.gridX), float32(l.gridY),
		float32(l.gridW), float32(l.gridH)), 1, colVPBorder)

	// Drawer-toggle triangle just below the viewport's bottom border: points up
	// when the drawer is closed (click to open), down when open (click to close).
	drawDrawerTriangle(l, mx, my)

	// Detail preview (fixed-size box, top-right) plus the press-and-hold full
	// preview popup. Rendered by drawPreview using the current mode's colours.
	drawPreview(c, l, focusX, focusY, previewZoom)
	// Just above the preview pane (2px gap): the preview pane's own scale factor
	// (right-aligned) and the editor viewport's zoom as a percentage (left-aligned)
	// on the fixed 5px=0% .. 160px=800% scale — the same value that drives the
	// grid/overlay fades, so the number read matches their behaviour regardless of
	// sprite size or window size.
	labelY := l.previewY - 2 - txt.CellH()
	txt.Draw("X"+itoa(previewZoom), l.previewX+l.previewW-20, labelY, 1, colDim)
	txt.Draw(itoa(int(zoomPct+0.5))+"%", l.previewX, labelY, 1, colDim)

	// Attribute palette (Spectrum Colour mode): 8 swatches, left-click sets ink,
	// right-click sets paper; a bright toggle beneath. The current ink and paper
	// selections are marked. It fades in/out on mode change (paletteFade), so it
	// keeps drawing during the fade-out even after the mode has switched away.
	if paletteFade > 0 {
		pa := uint8(255 * paletteFade)
		fadeC := func(c rl.Color) rl.Color { c.A = uint8(float32(c.A) * paletteFade); return c }
		tc := colText
		tc.A = pa
		txt.Draw("PALETTE", l.paletteX, l.paletteY-12, 1, tc)
		gc := colGrid
		gc.A = pa
		for i := 0; i < 16; i++ {
			r := l.swatchRects[i]
			base := l.swatchBase[i]
			bright := l.swatchBright[i]
			rl.DrawRectangleRec(r, fadeC(zxColor(zxpalette.Colour(base, bright))))
			rl.DrawRectangleLinesEx(r, 1, gc)
			// Mark the swatch matching the current ink / paper selection.
			if base == c.Ink() && bright == c.Bright() {
				mc := markColour(base)
				mc.A = pa
				txt.Draw("I", int(r.X)+3, int(r.Y)+3, 1, mc)
			}
			if base == c.Paper() && bright == c.Bright() {
				mc := markColour(base)
				mc.A = pa
				txt.Draw("P", int(r.X)+int(r.Width)-10, int(r.Y)+3, 1, mc)
			}
		}
	}

	// Buttons.
	// Strip buttons fade in/out as the window resize changes whether they fit
	// beside the viewport. The opacity is animated over time (btnOpacity), so the
	// transition plays fully even when a resize is reported as one completed step.
	for i, b := range l.buttons {
		af := float32(1)
		if i < len(btnOpacity) {
			af = btnOpacity[i]
		}
		if af <= 0 {
			continue // fully faded: skip drawing entirely
		}
		a := uint8(255 * af)
		bc := colBtn
		if b.hit(mx, my) {
			bc = colBtnHot
		}
		bc.A = a
		gc := colGrid
		gc.A = a
		tc := colText
		tc.A = a
		rl.DrawRectangle(int32(b.x), int32(b.y), int32(b.w), int32(b.h), bc)
		rl.DrawRectangleLines(int32(b.x), int32(b.y), int32(b.w), int32(b.h), gc)
		drawButtonLabelColour(txt, upper(b.label), b.x, b.y, b.w, b.h, tc)
	}

	// Press-and-hold full-preview popup, drawn last so it overlays everything.
	drawPreviewPopup(c, l, popup, focusX, focusY, previewZoom)
}

// markColour returns a high-contrast text colour for marking a palette swatch:
// black on bright/light colours (yellow, cyan, white, green), white otherwise.
func markColour(colour int) rl.Color {
	switch colour {
	case zxpalette.Yellow, zxpalette.Cyan, zxpalette.White, zxpalette.Green:
		return rl.Black
	default:
		return colInk
	}
}

// pixelColour returns the display colour of sprite pixel (x,y) in the current
// view mode. The second result is false for "transparent" pixels (clear pixels
// in the bitmap modes), which the caller renders as the chequer.
func pixelColour(c *ui.Controller, x, y int) (rl.Color, bool) {
	s := c.Sprite
	on := s.At(x, y)
	switch c.Mode() {
	case ui.SpectrumColour:
		attr := s.AttrAt(x, y)
		idx := zxpalette.Paper(attr)
		if on {
			idx = zxpalette.Ink(attr)
		}
		return zxColor(zxpalette.RGBA[zxpalette.Index(idx, zxpalette.Bright(attr))]), true
	case ui.BitmapWhite:
		if on {
			return colInk, true
		}
	default: // BitmapBlack
		if on {
			return rl.Black, true
		}
	}
	return rl.Color{}, false
}

// drawSpriteRegion renders a rectangular region of the sprite [cx0,cx0+span) x
// [cy0,cy0+span) into the screen box (bx,by,bw,bh), scaling to fill. Clear
// pixels in the bitmap modes show the transparency chequer; out-of-range cells
// are left as the box backing.
func drawSpriteRegion(c *ui.Controller, cx0, cy0, spanX, spanY int, bx, by, bw, bh float32) {
	s := c.Sprite
	pw := bw / float32(spanX)
	ph := bh / float32(spanY)
	for j := 0; j < spanY; j++ {
		sy := cy0 + j
		if sy < 0 || sy >= s.Height() {
			continue
		}
		for i := 0; i < spanX; i++ {
			sx := cx0 + i
			if sx < 0 || sx >= s.Width() {
				continue
			}
			rx := bx + float32(i)*pw
			ry := by + float32(j)*ph
			if col, draw := pixelColour(c, sx, sy); draw {
				rl.DrawRectangleRec(rl.NewRectangle(rx, ry, pw+0.5, ph+0.5), col)
			} else {
				drawCheckerPixel(rx, ry, pw+0.5, ph+0.5, sx, sy, c.Mode(), chequerVisibleFor(c.Mode()))
			}
		}
	}
}

// drawPreview renders the fixed-size detail preview at a fixed integer zoom
// (each sprite pixel is a zoom*zoom screen square; the visible span is
// previewBox/zoom pixels, centred on the focus and clamped within the sprite).
// previewRegion computes what the detail preview shows: the sprite region
// [rcx,rcy] sized [rspanX,rspanY], drawn at 'zoom' px per sprite pixel and placed
// at screen [rox,roy] with size [rdrawW,rdrawH] (centred in the box when smaller
// than it). It is shared by drawPreview and the collapsed end of the popup
// animation so the two match exactly.
func previewRegion(sw, sh, boxW, boxH, zoom, focusX, focusY int, pvX, pvY float32) (rcx, rcy, rspanX, rspanY int, rox, roy, rdrawW, rdrawH float32) {
	if zoom < 1 {
		zoom = 1
	}
	spanX := boxW / zoom
	spanY := boxH / zoom
	if spanX > sw {
		spanX = sw
	}
	if spanY > sh {
		spanY = sh
	}
	if spanX < 1 {
		spanX = 1
	}
	if spanY < 1 {
		spanY = 1
	}
	cx0 := focusX - spanX/2
	cy0 := focusY - spanY/2
	if cx0 < 0 {
		cx0 = 0
	}
	if cy0 < 0 {
		cy0 = 0
	}
	if cx0+spanX > sw {
		cx0 = sw - spanX
	}
	if cy0+spanY > sh {
		cy0 = sh - spanY
	}
	drawW := float32(spanX * zoom)
	drawH := float32(spanY * zoom)
	offX := pvX
	offY := pvY
	if drawW < float32(boxW) {
		offX += (float32(boxW) - drawW) / 2
	}
	if drawH < float32(boxH) {
		offY += (float32(boxH) - drawH) / 2
	}
	return cx0, cy0, spanX, spanY, offX, offY, drawW, drawH
}

func drawPreview(c *ui.Controller, l layout, focusX, focusY, zoom int) {
	s := c.Sprite
	pvX, pvY := float32(l.previewX), float32(l.previewY)
	pvW, pvH := float32(l.previewW), float32(l.previewH)

	// Box backing.
	rl.DrawRectangleRec(rl.NewRectangle(pvX, pvY, pvW, pvH), colGridArea)

	cx0, cy0, spanX, spanY, offX, offY, drawW, drawH :=
		previewRegion(s.Width(), s.Height(), l.previewW, l.previewH, zoom, focusX, focusY, pvX, pvY)

	rl.BeginScissorMode(int32(pvX), int32(pvY), int32(pvW), int32(pvH))
	drawSpriteRegion(c, cx0, cy0, spanX, spanY, offX, offY, drawW, drawH)
	rl.EndScissorMode()

	rl.DrawRectangleLinesEx(rl.NewRectangle(pvX, pvY, pvW, pvH), 1, colGrid)
}

// drawPreviewPopup renders the press-and-hold full-sprite overlay and its
// grow/shrink animation. The expanded state (t=1) is a large view of the whole
// frame anchored to the preview's top-right corner. The retraction does NOT
// scale the sprite down (which would distort/re-fit it); instead the sprite is
// drawn at an interpolated scale while the visible window is clipped, so the
// image is progressively clipped and, at t=0, lands on exactly the region and
// scale the preview pane shows — a seamless match. focusX/focusY and previewZoom
// are the same values drawPreview uses, so the collapsed end aligns pixel-for-
// pixel with the preview.
func drawPreviewPopup(c *ui.Controller, l layout, popup float32, focusX, focusY, previewZoom int) {
	if popup <= 0 {
		return
	}
	s := c.Sprite
	sw, sh := s.Width(), s.Height()
	pvX, pvY := float32(l.previewX), float32(l.previewY)
	pvW, pvH := float32(l.previewW), float32(l.previewH)
	anchorR := pvX + pvW // right edge stays fixed (top-right anchor)
	anchorT := pvY       // top edge stays fixed

	t := easeInOut(popup)
	scrW := float32(rl.GetScreenWidth())
	scrH := float32(rl.GetScreenHeight())

	// --- Expanded end (t=1): whole sprite fills the popup box. ---
	availW := anchorR - float32(pad)
	availH := scrH - float32(pad) - anchorT
	maxW := availW
	if maxW > scrW*0.75 {
		maxW = scrW * 0.75
	}
	maxH := availH
	if maxH > scrH*0.85 {
		maxH = scrH * 0.85
	}
	aspect := float32(sw) / float32(sh)
	tgtW := maxW
	tgtH := tgtW / aspect
	if tgtH > maxH {
		tgtH = maxH
		tgtW = tgtH * aspect
	}
	// Expanded box, anchored top-right; whole sprite drawn to fill it.
	boxW1, boxH1 := tgtW, tgtH
	boxX1 := anchorR - boxW1
	boxY1 := anchorT
	scale1 := tgtW / float32(sw) // screen px per sprite px, expanded
	originX1 := boxX1            // sprite pixel (0,0) screen position
	originY1 := boxY1

	// --- Collapsed end (t=0): match the preview pane exactly. ---
	zoom := previewZoom
	if zoom < 1 {
		zoom = 1
	}
	cx0, cy0, _, _, offX, offY, _, _ :=
		previewRegion(sw, sh, l.previewW, l.previewH, zoom, focusX, focusY, pvX, pvY)
	// Collapsed box is the preview rect; sprite drawn at 'zoom' px/pixel, with
	// pixel (0,0) at this screen origin so region [cx0,cy0] lands at [offX,offY].
	boxW0, boxH0 := pvW, pvH
	boxX0, boxY0 := pvX, pvY
	scale0 := float32(zoom)
	originX0 := offX - float32(cx0)*scale0
	originY0 := offY - float32(cy0)*scale0

	// --- Interpolate box, scale and sprite origin between the two ends. ---
	lerp := func(a, b float32) float32 { return a + (b-a)*t }
	boxX := lerp(boxX0, boxX1)
	boxY := lerp(boxY0, boxY1)
	boxW := lerp(boxW0, boxW1)
	boxH := lerp(boxH0, boxH1)
	scale := lerp(scale0, scale1)
	originX := lerp(originX0, originX1)
	originY := lerp(originY0, originY1)

	// Backdrop dim scales with the animation.
	rl.DrawRectangle(0, 0, int32(scrW), int32(scrH), rl.NewColor(0, 0, 0, uint8(150*t)))

	// Box backing, then the sprite drawn at the interpolated scale/origin, CLIPPED
	// to the box. Because scale tracks the box only through the shared endpoints,
	// the sprite is clipped (not squashed) as the box shrinks, and at t=0 the
	// visible window equals the preview's content exactly.
	rl.DrawRectangleRec(rl.NewRectangle(boxX, boxY, boxW, boxH), colGridArea)
	rl.BeginScissorMode(int32(boxX), int32(boxY), int32(boxW), int32(boxH))
	drawSpriteRegion(c, 0, 0, sw, sh, originX, originY, float32(sw)*scale, float32(sh)*scale)
	rl.EndScissorMode()
	rl.DrawRectangleLinesEx(rl.NewRectangle(boxX, boxY, boxW, boxH), 2, colGuide)
}

// easeInOut is a smoothstep ease for the popup animation.
func easeInOut(t float32) float32 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}

// drawCheckerPixel fills one virtual-pixel cell with a single transparency
// chequer square, alternating light/dark by pixel parity. One square per virtual
// pixel means an 8x8 character cell shows an 8x8 chequer. The chequer shade is
// nudged per view mode for contrast against the set-pixel colour: two notches
// darker in Bitmap White (so white pixels stand out against a dimmer chequer)
// and two notches lighter in Bitmap Black (so black pixels stand out against a
// brighter chequer). When chequerOn is false the whole area is a solid shade
// instead of a pattern (see chequerOffColour).
func drawCheckerPixel(x, y, w, h float32, px, py int, mode ui.ViewMode, chequerOn bool) {
	if !chequerOn {
		rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), chequerOffColour(mode))
		return
	}
	c := colChkLight
	if (px+py)%2 == 1 {
		c = colChkDark
	}
	c = shadeChequer(c, mode)
	rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), c)
}

// chequerOffColour is the solid fill used for the empty area when the chequer is
// toggled off. In Bitmap Black mode it takes the lightest chequer shade for that
// mode; in Bitmap White mode the darkest. This keeps the empty area distinct from
// the set-pixel colour (black pixels on a light field / white pixels on a dark
// field) without the busy chequer pattern.
func chequerOffColour(mode ui.ViewMode) rl.Color {
	switch mode {
	case ui.BitmapBlack:
		return shadeChequer(colChkLight, ui.BitmapBlack) // lightest for this bitmap
	default: // BitmapWhite
		return shadeChequer(colChkDark, ui.BitmapWhite) // darkest for this bitmap
	}
}

// chequerVisibleFor reports whether the transparency chequer is currently on for
// the given bitmap mode, per the LED toggles. The main loop keeps these in sync
// each frame; they are read from the preview drawing path, which does not thread
// the per-frame UI state through its signature.
var (
	chequerOnWhite = true
	chequerOnBlack = true
)

// resetRequested is set by the RESET button and consumed by the main loop, which
// then opens the typed confirmation modal. Package-level for the same reason as
// the chequer state: the button-action closures are built in computeLayout and
// cannot reach per-loop state.
var resetRequested bool

func chequerVisibleFor(mode ui.ViewMode) bool {
	if mode == ui.BitmapBlack {
		return chequerOnBlack
	}
	return chequerOnWhite
}

// chequerNotch is one shade step; two notches = 2*chequerNotch.
const chequerNotch = 0x14

// shadeChequer nudges a chequer grey per view mode.
func shadeChequer(c rl.Color, mode ui.ViewMode) rl.Color {
	var d int
	switch mode {
	case ui.BitmapWhite:
		d = -2 * chequerNotch // darker
	case ui.BitmapBlack:
		d = +2 * chequerNotch // lighter
	default:
		return c
	}
	adj := func(v uint8) uint8 {
		n := int(v) + d
		if n < 0 {
			n = 0
		}
		if n > 255 {
			n = 255
		}
		return uint8(n)
	}
	return rl.NewColor(adj(c.R), adj(c.G), adj(c.B), c.A)
}

// --- tiny string helpers (the Sinclair face is uppercase-friendly) ----------

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func upper(s string) string {
	b := []byte(s)
	for i, ch := range b {
		if ch >= 'a' && ch <= 'z' {
			b[i] = ch - 32
		}
	}
	return string(b)
}

// drawButtonLabel centres a label inside a button. If the label is too wide for
// the button at scale 1, it is split at a space into two centred lines so it
// stays legible when the button has shrunk in a narrow window. Labels with no
// space are drawn on one line (clipped by the button if truly too small).
// buttonVisible reports whether a strip button should be shown: true while its
// right border is within the viewport's right edge, false once it extends past.
// The actual on-screen fade is animated over time (see the button-opacity easing
// in the main loop), so this is a clean target rather than a gradual value — that
// way the fade animates fully even on platforms that report a resize only once it
// has finished, rather than as a continuous stream.
func buttonVisible(bx, bw, vpRight int) bool {
	return bx+bw <= vpRight
}

func drawButtonLabel(txt *bdfText, label string, bx, by, bw, bh int) {
	drawButtonLabelColour(txt, label, bx, by, bw, bh, colText)
}

// drawButtonLabelColour is drawButtonLabel with an explicit text colour, so the
// caller can fade the label (e.g. a button sliding past the viewport edge).
func drawButtonLabelColour(txt *bdfText, label string, bx, by, bw, bh int, col rl.Color) {
	lineH := txt.CellH()
	pad := 6
	if txt.Measure(label, 1) <= bw-2*pad {
		// Fits on one line: centre it.
		lw := txt.Measure(label, 1)
		txt.Draw(label, bx+(bw-lw)/2, by+(bh-lineH)/2, 1, col)
		return
	}
	// Split at the last space that keeps the first line within the button, else
	// the first space.
	split := -1
	for i := 0; i < len(label); i++ {
		if label[i] == ' ' {
			if txt.Measure(label[:i], 1) <= bw-2*pad {
				split = i
			} else if split == -1 {
				split = i
			}
		}
	}
	if split < 0 {
		// No space to break on: one line, centred (may clip).
		lw := txt.Measure(label, 1)
		txt.Draw(label, bx+(bw-lw)/2, by+(bh-lineH)/2, 1, col)
		return
	}
	l1 := label[:split]
	l2 := label[split+1:]
	gap := 1
	totalH := 2*lineH + gap
	y0 := by + (bh-totalH)/2
	w1 := txt.Measure(l1, 1)
	w2 := txt.Measure(l2, 1)
	txt.Draw(l1, bx+(bw-w1)/2, y0, 1, col)
	txt.Draw(l2, bx+(bw-w2)/2, y0+lineH+gap, 1, col)
}

// fadeRamp is a linear 0..1 ramp: 0 at or below lo, 1 at or above hi, linear
// between. Used by the grid/overlay fades, which are all keyed off the on-screen
// size of one virtual pixel (in device pixels) rather than a zoom ratio — that
// size is the true, invariant determinant of legibility, independent of window
// size, sprite dimensions, or the fitted base cell.
func fadeRamp(v, lo, hi float32) float32 {
	if hi <= lo {
		if v >= hi {
			return 1
		}
		return 0
	}
	f := (v - lo) / (hi - lo)
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// pppToPercent maps the on-screen pixel size to a zoom percentage on the
// screen-relative scale (pppMin -> 0%, pppMax -> 800%, linear). pppMin/pppMax are
// set from the monitor height at startup, so 0% means the tallest sprite fits the
// screen height. This is the one unit shown in the readout and used for the fades.
func pppToPercent(ppp float32) float32 {
	if pppMax <= pppMin {
		return 0
	}
	return (ppp - pppMin) / (pppMax - pppMin) * 800
}

// The grid/overlay visibility thresholds, in zoom percentage on the fixed
// 5px=0% .. 160px=800% scale — the same value shown in the readout. Because the
// scale is window-independent, these behave identically for every sprite size and
// window size and can be read/tuned directly against the readout.
// Recalculated for the widened zoom range (0% floor lowered to 0.8x fit-to-screen):
// these percentages keep each grid/overlay at the same physical pixel size it had
// under the previous range (guides 15-80, pixgrid 20-150, same-attr 150-400).
const (
	cellGuideFadeLo, cellGuideFadeHi = 37, 100  // character-cell guide lines
	pixGridFadeLo, pixGridFadeHi     = 42, 168  // Spectrum 1px grid
	flatCellFadeLo, flatCellFadeHi   = 168, 411 // flat-cell (same ink/paper) overlay
)

// cellGuideFade returns the opacity (0..1) for the character-cell guide lines at
// the given zoom percentage.
func cellGuideFade(pct float32) float32 {
	return fadeRamp(pct, cellGuideFadeLo, cellGuideFadeHi)
}

// pixGridFade returns the opacity (0..1) for the Spectrum-mode 1px grid at the
// given zoom percentage.
func pixGridFade(pct float32) float32 {
	return fadeRamp(pct, pixGridFadeLo, pixGridFadeHi)
}

// flatCellFade returns the opacity (0..1) for the flat-cell set-pixel overlay at
// the given zoom percentage. Only shows when very zoomed in.
func flatCellFade(pct float32) float32 {
	return fadeRamp(pct, flatCellFadeLo, flatCellFadeHi)
}

// truncateLabel shortens s to at most max characters, appending "..." when it is
// longer. Counts runes so multibyte names are not cut mid-character.
func truncateLabel(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

// forEachLinePixel walks the integer pixels of the line from (x0,y0) to (x1,y1)
// inclusive using Bresenham's algorithm, calling fn for each. It fills the gaps
// left when the pointer moves faster than one pixel per frame, so a fast stroke
// draws a continuous line rather than sparse dots.
func forEachLinePixel(x0, y0, x1, y1 int, fn func(x, y int)) {
	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y0
	if dy < 0 {
		dy = -dy
	}
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy
	for {
		fn(x0, y0)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// drawDrawerTriangle draws the small toggle triangle below the viewport: it
// points down while the drawer is open (click to close) and up while closed
// (click to open), following the eased drawerOpen progress.
func drawDrawerTriangle(l layout, mx, my int) {
	r := l.drawerToggle
	// Subtle backing so the triangle reads over any sprite content beneath it.
	pad2 := float32(3)
	bg := rl.NewRectangle(r.X-pad2, r.Y-pad2, r.Width+2*pad2, r.Height+2*pad2)
	rl.DrawRectangleRec(bg, rl.NewColor(0x10, 0x10, 0x18, 0xb0))
	col := colVPBorder
	if rectHit(r, mx, my) {
		col = colText
	}
	cx := r.X + r.Width/2
	open := l.drawerOpen >= 0.5
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

// drawWrappedLabel renders a button label on up to two lines, splitting at the
// first space, with each line horizontally centred and the block vertically
// centred in the button. Used only for the narrow mode/onion strip.
func drawWrappedLabel(txt *bdfText, b button, tint rl.Color) {
	label := upper(b.label)
	line1, line2 := label, ""
	if i := indexByte(label, ' '); i >= 0 {
		line1, line2 = label[:i], label[i+1:]
	}
	lineH := txt.CellH()
	totalH := lineH
	if line2 != "" {
		totalH = lineH*2 + 2
	}
	y0 := b.y + (b.h-totalH)/2
	cx := func(s string) int { return b.x + (b.w-txt.Measure(s, 1))/2 }
	txt.Draw(line1, cx(line1), y0, 1, tint)
	if line2 != "" {
		txt.Draw(line2, cx(line2), y0+lineH+2, 1, tint)
	}
}

// indexByte returns the index of the first occurrence of c in s, or -1.
const (
	axisNone = 0
	axisH    = 1
	axisV    = 2
)

// lockAxis applies Shift axis-lock: given the stroke anchor, the raw cursor
// pixel, and the current locked axis (axisNone until decided), it returns the
// constrained pixel and the (possibly newly decided) axis. The axis is chosen on
// the first move away from the anchor by the dominant direction and then held.
func lockAxis(anchorX, anchorY, px, py, axis int) (int, int, int) {
	if axis == axisNone {
		dx := px - anchorX
		dy := py - anchorY
		if dx != 0 || dy != 0 {
			if abs(dx) >= abs(dy) {
				axis = axisH
			} else {
				axis = axisV
			}
		}
	}
	switch axis {
	case axisH:
		py = anchorY
	case axisV:
		px = anchorX
	}
	return px, py, axis
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
