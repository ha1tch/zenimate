package zenui

import "github.com/ha1tch/zenimate/pkg/zxpalette"

// ZXClassicPaletteChooserConfig sets up the classic ZX Spectrum attribute
// palette: 16 swatches (8 base colours x normal/bright), laid out as a 4x4
// grid in the classic Spectrum colour-key pairing (blue,black / red,magenta /
// green,cyan / yellow,white), two base colours per row. Left-click a swatch
// picks it as ink; right-click picks it as paper.
//
// "Classic" distinguishes this from any future chooser for a different ZX
// palette (the ZX Spectrum Next's palette works on entirely different
// principles — 9-bit RGB, not eight named colours plus a bright flag — so a
// hypothetical ZXNextPaletteChooser would be its own type, not a variant of
// this one). It is not a claim that this is the only or definitive ZX
// palette chooser.
type ZXClassicPaletteChooserConfig struct {
	Anchor           Rect // top-left corner the grid is laid out from
	SwatchW, SwatchH int
	GapX, GapY       int
}

// zxSwatch is one cell of the 4x4 grid: its screen rect, and the ZX base
// colour + bright flag it represents.
type zxSwatch struct {
	rect   Rect
	base   int
	bright bool
}

// zxBaseOrder is the classic Spectrum colour-key pairing: two base colours
// per grid row, matching the arrangement printed on the Spectrum's own
// keyboard.
var zxBaseOrder = [8]int{
	zxpalette.Blue, zxpalette.Black,
	zxpalette.Red, zxpalette.Magenta,
	zxpalette.Green, zxpalette.Cyan,
	zxpalette.Yellow, zxpalette.White,
}

// ZXClassicPaletteChooser is a persistent 16-swatch tool palette, not a modal
// — unlike Dialog/Menu/OptionPanel it has no Status lifecycle; it simply
// reports picks from Update every frame, indefinitely. It does not own the
// current ink/paper/bright selection itself (the host does, the same
// separation as the code this widget was extracted from) — Draw takes the
// current selection each frame to mark it, and Update returns picks for the
// host to apply.
type ZXClassicPaletteChooser struct {
	cfg      ZXClassicPaletteChooserConfig
	swatches [16]zxSwatch
	hover    int // swatch index under the pointer this frame, or -1
}

// NewZXClassicPaletteChooser creates a chooser from cfg. It never returns
// nil.
func NewZXClassicPaletteChooser(cfg ZXClassicPaletteChooserConfig) *ZXClassicPaletteChooser {
	p := &ZXClassicPaletteChooser{cfg: cfg, hover: -1}
	p.layout()
	return p
}

// layout computes swatch rects from the anchor and swatch/gap dimensions.
// Pure geometry — unlike Menu/OptionPanel, it needs no Renderer, since swatch
// size is configured directly rather than measured from text.
func (p *ZXClassicPaletteChooser) layout() {
	for i := 0; i < 16; i++ {
		pair := i / 2       // 0..7 base-colour slot
		within := i % 2     // 0 = normal, 1 = bright
		pairRow := pair / 2 // 0..3
		pairCol := pair % 2 // 0..1
		gridCol := pairCol*2 + within
		x := p.cfg.Anchor.X + gridCol*(p.cfg.SwatchW+p.cfg.GapX)
		y := p.cfg.Anchor.Y + pairRow*(p.cfg.SwatchH+p.cfg.GapY)
		p.swatches[i] = zxSwatch{
			rect:   Rect{X: x, Y: y, W: p.cfg.SwatchW, H: p.cfg.SwatchH},
			base:   zxBaseOrder[pair],
			bright: within == 1,
		}
	}
}

// SetBounds repositions the grid (e.g. on window resize) without needing a
// full reconstruction.
func (p *ZXClassicPaletteChooser) SetBounds(anchor Rect) {
	p.cfg.Anchor = anchor
	p.layout()
}

// Bounds returns the grid's overall bounding rect (all 16 swatches plus the
// gaps between them), for the host to position other elements relative to it.
func (p *ZXClassicPaletteChooser) Bounds() Rect {
	w := 4*p.cfg.SwatchW + 3*p.cfg.GapX
	h := 4*p.cfg.SwatchH + 3*p.cfg.GapY
	return Rect{X: p.cfg.Anchor.X, Y: p.cfg.Anchor.Y, W: w, H: h}
}

// PaletteResult reports a pick from Update. At most one of InkPicked/
// PaperPicked is ever true per call — the widget makes no decision about
// what a pick means, it only reports which swatch was clicked and how; the
// host owns ink/paper/bright state and applies the pick itself.
type PaletteResult struct {
	InkPicked   bool
	PaperPicked bool
	Base        int  // valid when either Picked flag is true
	Bright      bool // valid when either Picked flag is true
}

// Update hit-tests the input snapshot against the swatch grid: a left click
// picks ink, a right click picks paper. Also updates the hover index Draw
// uses to highlight the swatch under the pointer.
func (p *ZXClassicPaletteChooser) Update(in Input) PaletteResult {
	p.hover = -1
	for i, sw := range p.swatches {
		if sw.rect.Contains(in.MouseX, in.MouseY) {
			p.hover = i
			break
		}
	}
	if p.hover < 0 {
		return PaletteResult{}
	}
	sw := p.swatches[p.hover]
	switch {
	case in.MousePressed:
		return PaletteResult{InkPicked: true, Base: sw.base, Bright: sw.bright}
	case in.MouseRightPressed:
		return PaletteResult{PaperPicked: true, Base: sw.base, Bright: sw.bright}
	default:
		return PaletteResult{}
	}
}

// zxToColour converts a zxpalette colour to zenui.Colour.
func zxToColour(base int, bright bool) Colour {
	c := zxpalette.Colour(base, bright)
	return Colour{R: c.R, G: c.G, B: c.B, A: c.A}
}

// zxMarkColour returns a legible ink/paper-marker colour against swatch base:
// black over the light colours, white otherwise. Specific to the eight named
// Spectrum colours (not a general luminance calculation), matching the exact
// contrast rule this widget was extracted from.
func zxMarkColour(base int) Colour {
	switch base {
	case zxpalette.Yellow, zxpalette.Cyan, zxpalette.White, zxpalette.Green:
		return Colour{R: 0, G: 0, B: 0, A: 0xff}
	default:
		return Colour{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	}
}

// Draw renders the 16 swatches. ink/paper/bright is the host's current
// selection, marked with "I"/"P" on the matching swatch(es); alpha scales
// every element uniformly (0..1), so the host can fade the whole palette in
// or out on a mode change without Draw needing its own animation state.
func (p *ZXClassicPaletteChooser) Draw(r Renderer, theme Theme, ink, paper int, bright bool, alpha float32) {
	if alpha <= 0 {
		return
	}
	fade := func(c Colour) Colour {
		return Colour{R: c.R, G: c.G, B: c.B, A: uint8(float32(c.A) * alpha)}
	}
	border := fade(theme.Border)
	for _, sw := range p.swatches {
		r.FillRect(sw.rect, fade(zxToColour(sw.base, sw.bright)))
		r.StrokeRect(sw.rect, border, 1)
		mark := fade(zxMarkColour(sw.base))
		if sw.base == ink && sw.bright == bright {
			r.DrawText("I", sw.rect.X+3, sw.rect.Y+3, 1, mark)
		}
		if sw.base == paper && sw.bright == bright {
			r.DrawText("P", sw.rect.X+sw.rect.W-10, sw.rect.Y+3, 1, mark)
		}
	}
}
