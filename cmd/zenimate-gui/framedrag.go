package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// dragMoveThreshold is the pointer movement, in pixels, past which a
// press-and-hold on a frame button is treated as a reorder drag rather than a
// plain click-to-select. Matches the ~6px figure used elsewhere in the GUI
// for press/drag disambiguation.
const dragMoveThreshold = 6

// frameDrag tracks a frame-reorder drag gesture. source is set on press over
// a frame button; active becomes true only once the pointer has moved past
// dragMoveThreshold, distinguishing a drag from a plain click (which the
// existing left-click handler already resolves independently — this struct
// only adds the reorder behaviour on top, it never suppresses the click).
type frameDrag struct {
	source           int // frame index the drag started on, or -1 when idle
	active           bool
	startMX, startMY int
	pulse            float32 // accumulated time, drives the ghost badge's pulsate
}

func newFrameDrag() frameDrag { return frameDrag{source: -1} }

// press begins tracking a potential drag from frame index i at the current
// pointer position. Does not itself select the frame — the existing
// left-click handler already does that independently.
func (d *frameDrag) press(i, mx, my int) {
	d.source = i
	d.active = false
	d.startMX, d.startMY = mx, my
	d.pulse = 0
}

// update advances the pulsate clock and promotes the gesture to active once
// the pointer has moved past the threshold. Call every frame while the left
// button is held and d.source >= 0.
func (d *frameDrag) update(dt float32, mx, my int) {
	d.pulse += dt
	if d.active {
		return
	}
	dx := float32(mx - d.startMX)
	dy := float32(my - d.startMY)
	if dx*dx+dy*dy > dragMoveThreshold*dragMoveThreshold {
		d.active = true
	}
}

// reset clears the drag, returning to idle.
func (d *frameDrag) reset() {
	d.source = -1
	d.active = false
}

// frameDropGap returns the insertion gap index nearest pointer x, given the
// current per-frame button rects. It walks the actual rects rather than
// assuming uniform spacing, so it stays correct even when button width has
// been clamped for a narrow window. Returns len(rects) (append at the end)
// if x is past every button's midpoint.
func frameDropGap(rects []rl.Rectangle, x float32) int {
	for i, r := range rects {
		if x < r.X+r.Width/2 {
			return i
		}
	}
	return len(rects)
}

// dragGapToMoveTarget converts a visual drop gap (computed against the
// static, pre-move layout that still includes the source frame's own slot)
// into the target index MoveFrame expects. A gap before or at the source's
// own original position needs no adjustment; a gap after it shifts down by
// one, because MoveFrame's target index is expressed after the source has
// already been removed. Hand-verified: source=1,gap=3 (drop after C, before
// D) -> to=2 -> [A,C,B,D,E]; source=3,gap=1 (drop after A, before B) -> to=1
// -> [A,D,B,C,E]; gap==source or gap==source+1 both map to a same-position
// no-op.
func dragGapToMoveTarget(source, gap int) int {
	if gap > source {
		return gap - 1
	}
	return gap
}

// dragPulseScale returns the ghost badge's scale factor at the drag's
// current elapsed time: a gentle continuous oscillation, driven by
// accumulated dt so it stays framerate-independent (the same principle as
// the button-fade animation, applied to a repeating pulse rather than a
// one-shot ease-to-target).
func dragPulseScale(pulse float32) float32 {
	return 1 + 0.15*float32(math.Sin(float64(pulse)*8))
}

// drawFrameDrag renders the insertion line at the current drop gap and a
// small pulsating badge showing the dragged frame's label, following the
// pointer. Call only while drag.active.
func drawFrameDrag(txt *bdfText, l layout, d frameDrag, mx, my int) {
	gap := frameDropGap(l.frameRects, float32(mx))

	// Insertion line: a bright vertical line spanning the frame strip height,
	// at the gap boundary (start of the strip if before frame 0, end of the
	// last button if past the last).
	lineX := float32(l.frameStripX)
	if n := len(l.frameRects); n > 0 {
		switch {
		case gap <= 0:
			lineX = l.frameRects[0].X
		case gap >= n:
			r := l.frameRects[n-1]
			lineX = r.X + r.Width
		default:
			lineX = l.frameRects[gap].X
		}
	}
	rl.DrawRectangle(int32(lineX)-1, int32(l.frameStripY), 2, int32(frameBtnH), rl.Yellow)

	// Ghost badge: the source frame's label, pulsating, offset from the cursor
	// so it doesn't sit directly under the pointer.
	label := "F" + itoa(d.source+1)
	scale := 2
	w := txt.Measure(label, scale)
	h := txt.CellH() * scale
	s := dragPulseScale(d.pulse)
	bw := int32(float32(w+12) * s)
	bh := int32(float32(h+8) * s)
	bx := int32(mx) + 14
	by := int32(my) + 14
	rl.DrawRectangle(bx, by, bw, bh, rl.NewColor(0x30, 0x50, 0x80, 0xe0))
	rl.DrawRectangleLines(bx, by, bw, bh, rl.Yellow)
	txt.Draw(label, int(bx)+6, int(by)+4, scale, rl.White)
}
