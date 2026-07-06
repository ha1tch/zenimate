// Package guiutil holds small, pure utility functions used by the
// zenimate-gui frontend: string formatting, easing/fade curves, and simple
// pixel geometry (line walking, axis locking). None of it depends on
// raylib, the sprite model, or any other part of the GUI — every function
// here is a pure function of its inputs, which is what makes it safe to
// extract as the first step of breaking up a very large main.go.
//
// Nested under cmd/zenimate-gui/internal (rather than the module's top-level
// internal/) because none of this is meant for reuse outside this one
// binary, unlike internal/model and internal/ui.
package guiutil

import "math"

// Itoa converts an int to its decimal string representation, without
// depending on the standard library's strconv (this GUI avoids strconv in
// its hot rendering path to keep allocations predictable).
func Itoa(n int) string {
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

// Upper uppercases the ASCII letters in s, leaving everything else
// untouched. The bitmap font this GUI draws with has no lowercase glyphs, so
// every label is uppercased before drawing.
func Upper(s string) string {
	b := []byte(s)
	for i, ch := range b {
		if ch >= 'a' && ch <= 'z' {
			b[i] = ch - 32
		}
	}
	return string(b)
}

// TruncateLabel shortens s to at most max characters, appending "..." when
// it is longer. Counts runes so multibyte names are not cut mid-character.
func TruncateLabel(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

// IndexByte returns the index of the first occurrence of c in s, or -1.
func IndexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// Abs returns the absolute value of x.
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// FadeRamp is a linear 0..1 ramp: 0 at or below lo, 1 at or above hi, linear
// between. Used by the grid/overlay fades, which are all keyed off the
// on-screen size of one virtual pixel (in device pixels) rather than a zoom
// ratio — that size is the true, invariant determinant of legibility,
// independent of window size, sprite dimensions, or the fitted base cell.
func FadeRamp(v, lo, hi float32) float32 {
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

// PPPToPercent maps an on-screen pixel-per-sprite-pixel size to a zoom
// percentage on the screen-relative scale (pppMin -> 0%, pppMax -> 800%,
// linear). pppMin/pppMax are set from the monitor height at startup by the
// caller — this function takes them as parameters rather than depending on
// package-level state, keeping it a pure function of its inputs.
func PPPToPercent(ppp, pppMin, pppMax float32) float32 {
	if pppMax <= pppMin {
		return 0
	}
	return (ppp - pppMin) / (pppMax - pppMin) * 800
}

// The grid/overlay visibility thresholds, in zoom percentage on the fixed
// 5px=0% .. 160px=800% scale — the same value shown in the readout. Because
// the scale is window-independent, these behave identically for every
// sprite size and window size and can be read/tuned directly against the
// readout. Recalculated for the widened zoom range (0% floor lowered to
// 0.8x fit-to-screen): these percentages keep each grid/overlay at the same
// physical pixel size it had under the previous range (guides 15-80,
// pixgrid 20-150, same-attr 150-400).
const (
	CellGuideFadeLo, CellGuideFadeHi = 37, 100  // character-cell guide lines
	PixGridFadeLo, PixGridFadeHi     = 42, 168  // Spectrum 1px grid
	FlatCellFadeLo, FlatCellFadeHi   = 168, 411 // flat-cell (same ink/paper) overlay
	// ChequerFadeLo, ChequerFadeHi bound the transparency chequer's own
	// fade-out at low zoom: unlike the other overlays above, which fade IN
	// as you zoom in, the chequer fades OUT as you zoom out, so its factor
	// is used directly as an opacity rather than inverted.
	ChequerFadeLo, ChequerFadeHi = 10, 40
)

// ChequerFade returns the transparency chequer's opacity (0..1) at the given
// zoom percentage: fully visible at 40% and above, fully gone at 10% and
// below, ramping linearly in between.
func ChequerFade(pct float32) float32 {
	return FadeRamp(pct, ChequerFadeLo, ChequerFadeHi)
}

// CellGuideFade returns the opacity (0..1) for the character-cell guide
// lines at the given zoom percentage.
func CellGuideFade(pct float32) float32 {
	return FadeRamp(pct, CellGuideFadeLo, CellGuideFadeHi)
}

// PixGridFade returns the opacity (0..1) for the Spectrum-mode 1px grid at
// the given zoom percentage.
func PixGridFade(pct float32) float32 {
	return FadeRamp(pct, PixGridFadeLo, PixGridFadeHi)
}

// FlatCellFade returns the opacity (0..1) for the flat-cell set-pixel
// overlay at the given zoom percentage. Only shows when very zoomed in.
func FlatCellFade(pct float32) float32 {
	return FadeRamp(pct, FlatCellFadeLo, FlatCellFadeHi)
}

// EaseInOut is a smoothstep ease, used by the preview popup animation.
func EaseInOut(t float32) float32 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}

// ButtonVisible reports whether a strip button should be shown: true while
// its right border is within the viewport's right edge, false once it
// extends past. The actual on-screen fade is animated over time separately,
// so this is a clean target rather than a gradual value — that way the fade
// animates fully even on platforms that report a resize only once it has
// finished, rather than as a continuous stream.
func ButtonVisible(bx, bw, vpRight int) bool {
	return bx+bw <= vpRight
}

// ForEachLinePixel walks the integer pixels of the line from (x0,y0) to
// (x1,y1) inclusive using Bresenham's algorithm, calling fn for each. It
// fills the gaps left when the pointer moves faster than one pixel per
// frame, so a fast stroke draws a continuous line rather than sparse dots.
func ForEachLinePixel(x0, y0, x1, y1 int, fn func(x, y int)) {
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

// RectOutline walks the four edges of the rectangle with corners (x0,y0) and
// (x1,y1), calling fn for each pixel — the shape-tool equivalent of
// ForEachLinePixel, built from four calls to it. Outline only, not filled:
// the classic default for a pixel-art shape tool.
func RectOutline(x0, y0, x1, y1 int, fn func(x, y int)) {
	ForEachLinePixel(x0, y0, x1, y0, fn) // top
	ForEachLinePixel(x1, y0, x1, y1, fn) // right
	ForEachLinePixel(x1, y1, x0, y1, fn) // bottom
	ForEachLinePixel(x0, y1, x0, y0, fn) // left
}

// TriangleOutline walks the three edges of an isoceles triangle inscribed in
// the bounding box (x0,y0)-(x1,y1): apex centred on the top edge, base along
// the bottom edge — matching the classic triangle shape-tool icon.
func TriangleOutline(x0, y0, x1, y1 int, fn func(x, y int)) {
	apexX := (x0 + x1) / 2
	ForEachLinePixel(apexX, y0, x0, y1, fn) // apex to bottom-left
	ForEachLinePixel(apexX, y0, x1, y1, fn) // apex to bottom-right
	ForEachLinePixel(x0, y1, x1, y1, fn)    // base
}

// BrushShape identifies a pen/brush stamp shape.
type BrushShape int

const (
	BrushRound BrushShape = iota
	BrushSquare
	BrushCustom
)

// BrushStamp calls fn for every pixel offset (dx,dy), relative to a brush
// centred at the origin, that the given shape and size cover. size is the
// stamp's width in pixels (minimum 1, clamped).
//
// At pen-brush scale (1-4px, appropriate for ZX Spectrum sprites) a
// genuinely circular rasterisation is barely distinguishable from a plain
// square and fiddly to get looking right without visual iteration, so
// Round instead uses a well-defined, easily-verified convention: the full
// N x N square with its four corner pixels chamfered off once N >= 3. This
// is a real design choice, not a neutral default — see BrushCustom below
// for the third, genuinely distinct option it's paired with.
func BrushStamp(shape BrushShape, size int, fn func(dx, dy int)) {
	if size < 1 {
		size = 1
	}
	lo := -(size / 2)
	hi := size - 1 + lo
	switch shape {
	case BrushSquare:
		for dy := lo; dy <= hi; dy++ {
			for dx := lo; dx <= hi; dx++ {
				fn(dx, dy)
			}
		}
	case BrushCustom:
		// The X/diagonal-cross complement of Round's + shape: just the four
		// corners of the bounding box. Chosen specifically because it's
		// maximally distinct from Round (which removes exactly those same
		// corners) at every size where the distinction is meaningful — an
		// earlier diamond-distance formula looked plausible but happened to
		// produce the identical 5-point shape as Round at size 3, the most
		// likely size to actually get used; caught by hand-tracing before
		// this was ever tested or shipped.
		for dy := lo; dy <= hi; dy++ {
			for dx := lo; dx <= hi; dx++ {
				isCorner := (dx == lo || dx == hi) && (dy == lo || dy == hi)
				if size < 3 || isCorner {
					fn(dx, dy)
				}
			}
		}
	default: // BrushRound
		for dy := lo; dy <= hi; dy++ {
			for dx := lo; dx <= hi; dx++ {
				isCorner := size >= 3 && (dx == lo || dx == hi) && (dy == lo || dy == hi)
				if !isCorner {
					fn(dx, dy)
				}
			}
		}
	}
}

// EllipseOutline walks the boundary of the ellipse inscribed in the bounding
// box (x0,y0)-(x1,y1), calling fn for each pixel. Scans both axes (solving
// the ellipse equation for y given x, and separately for x given y) and
// takes the union — a single-axis scan leaves gaps near the steep parts of
// the curve (close to the left/right extremes when scanning by x, or the
// top/bottom extremes when scanning by y); scanning both and unioning the
// results has no such gap regardless of the bounding box's aspect ratio.
func EllipseOutline(x0, y0, x1, y1 int, fn func(x, y int)) {
	cx := float64(x0+x1) / 2
	cy := float64(y0+y1) / 2
	rx := float64(x1-x0) / 2
	ry := float64(y1-y0) / 2
	if rx <= 0 || ry <= 0 {
		fn(x0, y0)
		return
	}
	for x := x0; x <= x1; x++ {
		t := (float64(x) - cx) / rx
		if t*t > 1 {
			continue
		}
		dy := ry * math.Sqrt(1-t*t)
		fn(x, int(math.Round(cy-dy)))
		fn(x, int(math.Round(cy+dy)))
	}
	for y := y0; y <= y1; y++ {
		t := (float64(y) - cy) / ry
		if t*t > 1 {
			continue
		}
		dx := rx * math.Sqrt(1-t*t)
		fn(int(math.Round(cx-dx)), y)
		fn(int(math.Round(cx+dx)), y)
	}
}

// PolygonOutline walks the edges of a regular polygon with the given number
// of sides (clamped to 3-12), vertices inscribed in the ellipse of the
// bounding box (x0,y0)-(x1,y1). Vertices start at 90° so an even-sided
// polygon (6, matching the original hexagon icon) has a flat top and bottom
// edge; scaled independently on each axis to fit whatever aspect ratio the
// drag defines, the same way the ellipse and rectangle tools fit an
// arbitrary (not necessarily square) bounding box rather than staying
// strictly regular.
func PolygonOutline(sides int, x0, y0, x1, y1 int, fn func(x, y int)) {
	if sides < 3 {
		sides = 3
	}
	if sides > 12 {
		sides = 12
	}
	cx := float64(x0+x1) / 2
	cy := float64(y0+y1) / 2
	rx := float64(x1-x0) / 2
	ry := float64(y1-y0) / 2

	vx := make([]int, sides)
	vy := make([]int, sides)
	for i := 0; i < sides; i++ {
		angle := math.Pi/2 + float64(i)*2*math.Pi/float64(sides)
		vx[i] = int(math.Round(cx + rx*math.Cos(angle)))
		vy[i] = int(math.Round(cy - ry*math.Sin(angle)))
	}
	for i := 0; i < sides; i++ {
		j := (i + 1) % sides
		ForEachLinePixel(vx[i], vy[i], vx[j], vy[j], fn)
	}
}

// CenterOrCornerBounds computes an effective bounding box from a drag's
// anchor and current point, in one of two modes: corner mode treats anchor
// and current as opposite corners directly (the conventional drag-a-
// rectangle behaviour); center mode treats anchor as a fixed center point
// and current as defining the radius in each axis (the distance from
// anchor), so the shape grows symmetrically outward as the drag continues.
// Shared by any tool that offers both conventions rather than duplicating
// the same handful of lines per tool.
func CenterOrCornerBounds(anchorX, anchorY, curX, curY int, fromCenter bool) (x0, y0, x1, y1 int) {
	if !fromCenter {
		return anchorX, anchorY, curX, curY
	}
	rx := Abs(curX - anchorX)
	ry := Abs(curY - anchorY)
	return anchorX - rx, anchorY - ry, anchorX + rx, anchorY + ry
}

// Axis is a locked paint-stroke axis: none, horizontal, or vertical.
type Axis int

const (
	AxisNone Axis = iota
	AxisH
	AxisV
)

// LockAxis applies Shift axis-lock: given the stroke anchor, the raw cursor
// pixel, and the current locked axis (AxisNone until decided), it returns
// the constrained pixel and the (possibly newly decided) axis. The axis is
// chosen on the first move away from the anchor by the dominant direction
// and then held.
func LockAxis(anchorX, anchorY, px, py int, axis Axis) (int, int, Axis) {
	if axis == AxisNone {
		dx := px - anchorX
		dy := py - anchorY
		if dx != 0 || dy != 0 {
			if Abs(dx) >= Abs(dy) {
				axis = AxisH
			} else {
				axis = AxisV
			}
		}
	}
	switch axis {
	case AxisH:
		py = anchorY
	case AxisV:
		px = anchorX
	}
	return px, py, axis
}
