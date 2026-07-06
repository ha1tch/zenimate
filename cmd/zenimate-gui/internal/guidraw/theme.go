package guidraw

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guiutil"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

// Theme is this frontend's whole visual identity: the fixed ZX-ish colour
// palette plus the two chequer on/off toggles, grouped into one queryable,
// updatable value rather than scattered package-level state. Construct with
// DefaultTheme, hold the result (typically as a pointer) for the life of the
// program, mutate it directly when a toggle flips (e.g. the chequer LED
// buttons), and pass it into every drawing function that needs it — the
// same pattern already proven by pkg/zenui.Theme.
type Theme struct {
	BG       rl.Color
	GridArea rl.Color // lighter backing behind the sprite grid
	Grid     rl.Color
	Ink      rl.Color
	Sel      rl.Color
	Btn      rl.Color
	BtnHot   rl.Color
	Yellow   rl.Color
	Text     rl.Color
	Dim      rl.Color // dimmer subtitle / secondary text
	Guide    rl.Color // dark grey: 8-pixel character-cell guides
	VPBorder rl.Color // medium grey: viewport box border
	PixGrid  rl.Color // almost-invisible 1px grid (Spectrum mode)

	// Translucent onion-skin silhouettes: previous frame red, next frame green.
	OnionPrev rl.Color
	OnionNext rl.Color

	// Photoshop-style transparency chequer for empty cells: one square per
	// virtual pixel (8x8 squares per character cell), alternating these greys.
	ChkLight rl.Color
	ChkDark  rl.Color

	// ChequerOnWhite and ChequerOnBlack are the per-mode transparency-chequer
	// LED toggles. Part of Theme (rather than separate state) because they're
	// updated and queried through exactly the same lifecycle as the colours:
	// main flips them on a button click, the drawing functions read them
	// every frame.
	ChequerOnWhite bool
	ChequerOnBlack bool
}

// DefaultTheme returns zenimate-gui's fixed visual identity.
func DefaultTheme() Theme {
	return Theme{
		BG:       rl.NewColor(0x10, 0x10, 0x18, 0xff),
		GridArea: rl.NewColor(0x2c, 0x2c, 0x38, 0xff),
		Grid:     rl.NewColor(0x30, 0x30, 0x40, 0xff),
		Ink:      rl.NewColor(0xff, 0xff, 0xff, 0xff),
		Sel:      rl.NewColor(0x00, 0x80, 0x00, 0xff),
		Btn:      rl.NewColor(0x28, 0x28, 0x34, 0xff),
		BtnHot:   rl.NewColor(0x3a, 0x3a, 0x4a, 0xff),
		Yellow:   rl.NewColor(0xff, 0xd0, 0x00, 0xff),
		Text:     rl.NewColor(0xe0, 0xe0, 0xe8, 0xff),
		Dim:      rl.NewColor(0x80, 0x80, 0x90, 0xff),
		Guide:    rl.NewColor(0x40, 0x40, 0x40, 0xff),
		VPBorder: rl.NewColor(0x6a, 0x6a, 0x78, 0xff),
		PixGrid:  rl.NewColor(0x00, 0x00, 0x00, 0x28),

		OnionPrev: rl.NewColor(0xff, 0x30, 0x30, 0x70),
		OnionNext: rl.NewColor(0x30, 0xff, 0x30, 0x70),

		ChkLight: rl.NewColor(0xb0, 0xb0, 0xb0, 0xff),
		ChkDark:  rl.NewColor(0x88, 0x88, 0x88, 0xff),

		ChequerOnWhite: true,
		ChequerOnBlack: true,
	}
}

// ChequerVisibleFor reports whether the transparency chequer is currently on
// for the given bitmap mode, per the LED toggles.
func (th Theme) ChequerVisibleFor(mode ui.ViewMode) bool {
	if mode == ui.BitmapBlack {
		return th.ChequerOnBlack
	}
	return th.ChequerOnWhite
}

// ChequerNotch is one shade step; two notches = 2*ChequerNotch.
const ChequerNotch = 0x14

// ShadeChequer nudges a chequer grey per view mode: two notches darker in
// Bitmap White (so white pixels stand out against a dimmer chequer), two
// notches lighter in Bitmap Black (so black pixels stand out against a
// brighter chequer).
func ShadeChequer(c rl.Color, mode ui.ViewMode) rl.Color {
	var d int
	switch mode {
	case ui.BitmapWhite:
		d = -2 * ChequerNotch
	case ui.BitmapBlack:
		d = +2 * ChequerNotch
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

// ChequerOffColour is the solid fill used for the empty area when the
// chequer is toggled off: lightest chequer shade in Bitmap Black, darkest in
// Bitmap White, keeping the empty area distinct from the set-pixel colour.
func (th Theme) ChequerOffColour(mode ui.ViewMode) rl.Color {
	switch mode {
	case ui.BitmapBlack:
		return ShadeChequer(th.ChkLight, ui.BitmapBlack)
	default: // BitmapWhite
		return ShadeChequer(th.ChkDark, ui.BitmapWhite)
	}
}

// CheckerColour returns the colour one virtual-pixel chequer cell should be
// filled, without drawing it — shared by DrawCheckerPixel (the main canvas)
// fade is the chequer's own zoom-based opacity (1 = fully visible, 0 =
// fully faded to a solid ChequerOffColour), from guiutil.ChequerFade at the
// call site — kept as a plain float parameter here rather than importing
// guiutil, since zoom-percentage computation is the caller's concern.
func (th Theme) CheckerColour(px, py int, mode ui.ViewMode, chequerOn bool, fade float32) rl.Color {
	if !chequerOn {
		return th.ChequerOffColour(mode)
	}
	c := th.ChkLight
	if (px+py)%2 == 1 {
		c = th.ChkDark
	}
	shaded := ShadeChequer(c, mode)
	if fade >= 1 {
		return shaded
	}
	return lerpColor(th.ChequerOffColour(mode), shaded, fade)
}

// lerpColor blends linearly from a to b as t goes from 0 to 1.
func lerpColor(a, b rl.Color, t float32) rl.Color {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	lerp8 := func(a, b uint8, t float32) uint8 {
		return uint8(float32(a) + (float32(b)-float32(a))*t)
	}
	return rl.Color{
		R: lerp8(a.R, b.R, t),
		G: lerp8(a.G, b.G, t),
		B: lerp8(a.B, b.B, t),
		A: lerp8(a.A, b.A, t),
	}
}

// DrawCheckerPixel fills one virtual-pixel cell with a single transparency
// chequer square, alternating light/dark by pixel parity, faded toward a
// solid background as zoom decreases (see CheckerColour's fade parameter).
func (th Theme) DrawCheckerPixel(x, y, w, h float32, px, py int, mode ui.ViewMode, chequerOn bool, fade float32) {
	rl.DrawRectangleRec(rl.NewRectangle(x, y, w, h), th.CheckerColour(px, py, mode, chequerOn, fade))
}

// ZxColor converts a zxpalette colour to a raylib colour.
func ZxColor(n color.NRGBA) rl.Color {
	return rl.NewColor(n.R, n.G, n.B, n.A)
}

// MarkColour returns a legible ink/paper-marker colour against a ZX base
// colour: black over the light colours, white otherwise.
func (th Theme) MarkColour(colour int) rl.Color {
	switch colour {
	case zxpalette.Yellow, zxpalette.Cyan, zxpalette.White, zxpalette.Green:
		return rl.Black
	default:
		return th.Ink
	}
}

// PixelColour returns the display colour of sprite pixel (x,y) in the
// current view mode. The second result is false for "transparent" pixels
// (clear pixels in the bitmap modes), which the caller renders as the
// chequer.
func (th Theme) PixelColour(c *ui.Controller, x, y int) (rl.Color, bool) {
	s := c.Sprite
	on := s.At(x, y)
	switch c.Mode() {
	case ui.SpectrumColour:
		attr := s.AttrAt(x, y)
		idx := zxpalette.Paper(attr)
		if on {
			idx = zxpalette.Ink(attr)
		}
		return ZxColor(zxpalette.RGBA[zxpalette.Index(idx, zxpalette.Bright(attr))]), true
	case ui.BitmapWhite:
		if on {
			return th.Ink, true
		}
	default: // BitmapBlack
		if on {
			return rl.Black, true
		}
	}
	return rl.Color{}, false
}

// RectHit reports whether (mx,my) falls within an rl.Rectangle.
func RectHit(r rl.Rectangle, mx, my int) bool {
	return float32(mx) >= r.X && float32(mx) < r.X+r.Width &&
		float32(my) >= r.Y && float32(my) < r.Y+r.Height
}

// DrawButtonLabel centres a label inside a button, at the theme's default
// text colour. If the label is too wide for the button at scale 1, it
// splits at a space into two centred lines.
func (th Theme) DrawButtonLabel(txt *BDFText, label string, bx, by, bw, bh int) {
	th.DrawButtonLabelColour(txt, label, bx, by, bw, bh, th.Text)
}

// DrawButtonLabelColour is DrawButtonLabel with an explicit text colour, so
// the caller can fade the label (e.g. a button sliding past the viewport
// edge).
func (th Theme) DrawButtonLabelColour(txt *BDFText, label string, bx, by, bw, bh int, col rl.Color) {
	lineH := txt.CellH()
	pad := 6
	if txt.Measure(label, 1) <= bw-2*pad {
		lw := txt.Measure(label, 1)
		txt.Draw(label, bx+(bw-lw)/2, by+(bh-lineH)/2, 1, col)
		return
	}
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

// DrawWrappedLabel renders a button label on up to two lines, splitting at
// the first space, with each line horizontally centred and the block
// vertically centred in the button. Used only for the narrow mode/onion
// strip.
func DrawWrappedLabel(txt *BDFText, b Button, tint rl.Color) {
	label := guiutil.Upper(b.Label)
	line1, line2 := label, ""
	if i := guiutil.IndexByte(label, ' '); i >= 0 {
		line1, line2 = label[:i], label[i+1:]
	}
	lineH := txt.CellH()
	totalH := lineH
	if line2 != "" {
		totalH = lineH*2 + 2
	}
	y0 := b.Y + (b.H-totalH)/2
	cx := func(s string) int { return b.X + (b.W-txt.Measure(s, 1))/2 }
	txt.Draw(line1, cx(line1), y0, 1, tint)
	if line2 != "" {
		txt.Draw(line2, cx(line2), y0+lineH+2, 1, tint)
	}
}
