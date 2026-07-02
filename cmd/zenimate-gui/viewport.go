package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// viewport carries the pan/zoom state of the editor grid and the velocity state
// that gives zoom and pan their inertia (glide-after-release). It persists
// across frames; computeLayout supplies the base (fit-to-window) cell size and
// the grid's anchor, and the viewport scales and offsets from there.
//
// zoom multiplies the base cell size. pan is a screen-pixel offset added to the
// grid origin. Both ease toward rest via decaying velocities so a flick of the
// wheel or a quick drag keeps moving briefly after the input stops.
type viewport struct {
	zoom       float32
	panX, panY float32

	zoomVel float32
	panVelX float32
	panVelY float32

	dragging bool
	lastMX   int
	lastMY   int

	// Fit animation: when animating is true the viewport eases its zoom/pan
	// toward (tgtZoom, tgtPanX, tgtPanY), used to zoom-to-fit after a resize.
	animating        bool
	tgtZoom          float32
	tgtPanX, tgtPanY float32
}

func newViewport() *viewport {
	return &viewport{zoom: 1}
}

// minZoom and maxZoom bound the interactive wheel zoom. They are screen-relative:
// set at startup so that at minZoom the tallest possible sprite (MaxSpriteHeight
// px) exactly fits the screen height (= 0% on the readout), and maxZoom is 8x that
// (= 800%). setZoomRangeForScreen computes them from the monitor height.
var (
	minZoom float32 = 0.25
	maxZoom float32 = 8.0
)

// MaxSpriteHeightPx is the pixel height of the largest sprite (32x24 cells).
const MaxSpriteHeightPx = 24 * 8 // 192

// pppMin / pppMax are the on-screen pixel-per-virtual-pixel sizes at 0% and 800%.
// Screen-relative: pppMin fits the tallest sprite to the screen height, pppMax is
// 8x that. Set by setZoomRangeForScreen at startup.
var (
	pppMin float32 = 5.0
	pppMax float32 = 40.0
)

// zoomOutHeadroom widens the low end of the zoom range: the floor (0%) is this
// fraction of the fit-to-screen pixel size, so the user can zoom out somewhat
// further than "tallest sprite exactly fills the screen height". 0.8 = 20% more
// zoom-out room. The top (pppMax / 800%) is unaffected.
const zoomOutHeadroom = 0.8

// setZoomRangeForScreen anchors the zoom scale to the screen height. At 800% each
// virtual pixel is 8x the fit-to-screen size. The 0% floor is zoomOutHeadroom x
// the fit-to-screen size, giving a little room to zoom out past a screen-height
// fit. Called at startup and whenever the window moves to a different monitor.
func setZoomRangeForScreen(screenH int) {
	if screenH <= 0 {
		return
	}
	fit := float32(screenH) / float32(MaxSpriteHeightPx) // pixel size that fits the tallest sprite to screen height
	pppMin = fit * zoomOutHeadroom
	pppMax = fit * 8
	minZoom = pppMin / cellPx
	maxZoom = pppMax / cellPx
}

const (
	// Wheel sensitivity and how fast the zoom velocity decays (per second).
	zoomWheelGain = 6.0
	zoomDecay     = 8.0
	zoomMinVel    = 0.001

	// Pan inertia decay (per second) and the minimum velocity below which the
	// glide stops.
	panDecay  = 9.0
	panMinVel = 2.0

	// Fit-animation easing rate (per second) and the snap threshold.
	fitEase = 12.0
	fitSnap = 0.5
)

// animateFit begins easing the viewport toward a zoom/pan that fits a sprite of
// (spriteW x spriteH) px centred in a (boxW x boxH) box at base cell size
// baseCell. Called when the sprite is resized.
func (v *viewport) animateFit(spriteW, spriteH, boxW, boxH, baseCell int) {
	if spriteW <= 0 || spriteH <= 0 || baseCell <= 0 {
		return
	}
	// Fit zoom: largest zoom whose scaled sprite fits the box, capped to 1 so we
	// never enlarge beyond the base cell size, and clamped to the zoom range.
	zx := float32(boxW) / float32(spriteW*baseCell)
	zy := float32(boxH) / float32(spriteH*baseCell)
	z := zx
	if zy < z {
		z = zy
	}
	if z > 1 {
		z = 1
	}
	// The fit target may legitimately fall below the interactive minZoom for a
	// large sprite in a small box; clamp only to maxZoom and a small positive
	// floor so the whole sprite is always brought into view.
	if z > maxZoom {
		z = maxZoom
	}
	if z < 0.02 {
		z = 0.02
	}

	// Centre the sprite in the box: pan is relative to the grid origin, which is
	// the box top-left, so centring offset is half the slack.
	cell := float32(baseCell) * z
	v.tgtPanX = (float32(boxW) - float32(spriteW)*cell) / 2
	v.tgtPanY = (float32(boxH) - float32(spriteH)*cell) / 2
	v.tgtZoom = z
	v.animating = true
	// Cancel any residual inertia so the animation reads cleanly.
	v.zoomVel, v.panVelX, v.panVelY = 0, 0, 0
}

// update advances the viewport for one frame by reading raylib input and the
// frame clock, then delegating to step (which is pure and testable).
//
//	mx, my       cursor position
//	originX/Y    the grid's base origin (top-left) from layout
//	baseCell     the fit-to-window cell size from layout
//	panning      whether hand-drag (space held) is active this frame
//	overGrid     whether the cursor is over the grid (gates wheel zoom)
func (v *viewport) update(mx, my, originX, originY, baseCell int, panning, overGrid bool) {
	dt := rl.GetFrameTime()
	if dt <= 0 || dt > 0.1 {
		dt = 1.0 / 60.0 // clamp on stalls so inertia stays sane
	}
	wheel := float32(0)
	if overGrid {
		wheel = rl.GetMouseWheelMove() * wheelSign()
	}
	v.step(dt, wheel, mx, my, originX, originY, baseCell, panning)
}

// step is the pure pan/zoom integrator: given the frame time, wheel delta, and
// current cursor/origin, it applies cursor-anchored zoom, hand-drag panning, and
// the inertia that keeps both easing after input stops. It takes no raylib
// state, so it can be unit-tested directly.
func (v *viewport) step(dt, wheel float32, mx, my, originX, originY, baseCell int, panning bool) {
	if dt <= 0 {
		dt = 1.0 / 60.0
	}

	// --- fit animation (zoom-to-fit after a resize) ---
	// Any user input (wheel or pan) cancels the animation and hands control back.
	if v.animating {
		if wheel != 0 || panning {
			v.animating = false
		} else {
			a := clamp32(fitEase*dt, 0, 1)
			v.zoom += (v.tgtZoom - v.zoom) * a
			v.panX += (v.tgtPanX - v.panX) * a
			v.panY += (v.tgtPanY - v.panY) * a
			if abs32(v.tgtZoom-v.zoom) < 0.001 &&
				abs32(v.tgtPanX-v.panX) < fitSnap && abs32(v.tgtPanY-v.panY) < fitSnap {
				v.zoom, v.panX, v.panY = v.tgtZoom, v.tgtPanX, v.tgtPanY
				v.animating = false
			}
			return // while animating, ignore the normal zoom/pan/inertia path
		}
	}

	// --- wheel zoom (anchored at the cursor) ---
	if wheel != 0 {
		v.zoomVel += wheel * zoomWheelGain
	}
	if abs32(v.zoomVel) > zoomMinVel {
		cellBefore := float32(baseCell) * v.zoom
		gx := float32(originX) + v.panX
		gy := float32(originY) + v.panY
		var wxu, wyu float32
		if cellBefore != 0 {
			wxu = (float32(mx) - gx) / cellBefore
			wyu = (float32(my) - gy) / cellBefore
		}

		factor := 1 + v.zoomVel*dt
		if factor < 0.1 {
			factor = 0.1
		}
		v.zoom = clamp32(v.zoom*factor, minZoom, maxZoom)

		cellAfter := float32(baseCell) * v.zoom
		v.panX = float32(mx) - float32(originX) - wxu*cellAfter
		v.panY = float32(my) - float32(originY) - wyu*cellAfter

		v.zoomVel -= v.zoomVel * zoomDecay * dt
		if abs32(v.zoomVel) <= zoomMinVel {
			v.zoomVel = 0
		}
	}

	// --- hand-drag pan (active while the pan gesture is held) ---
	if panning {
		if !v.dragging {
			v.dragging = true
			v.lastMX, v.lastMY = mx, my
			v.panVelX, v.panVelY = 0, 0
		}
		dx := float32(mx - v.lastMX)
		dy := float32(my - v.lastMY)
		v.panX += dx
		v.panY += dy
		if dt > 0 {
			v.panVelX = dx / dt
			v.panVelY = dy / dt
		}
		v.lastMX, v.lastMY = mx, my
	} else if v.dragging {
		v.dragging = false // released: keep last velocity as the flick
	}

	// --- pan inertia (glide after release) ---
	if !panning && (abs32(v.panVelX) > panMinVel || abs32(v.panVelY) > panMinVel) {
		v.panX += v.panVelX * dt
		v.panY += v.panVelY * dt
		v.panVelX -= v.panVelX * panDecay * dt
		v.panVelY -= v.panVelY * panDecay * dt
		if abs32(v.panVelX) <= panMinVel {
			v.panVelX = 0
		}
		if abs32(v.panVelY) <= panMinVel {
			v.panVelY = 0
		}
	}
}

// cellF returns the fractional effective cell size. Rendering uses this so the
// grid scales smoothly with zoom instead of snapping when the integer cell size
// changes.
func (v *viewport) cellF(baseCell int) float32 {
	c := float32(baseCell) * v.zoom
	if c < 0.5 {
		c = 0.5
	}
	return c
}

// originF returns the fractional effective grid top-left, for smooth panning.
func (v *viewport) originF(originX, originY int) (float32, float32) {
	return float32(originX) + v.panX, float32(originY) + v.panY
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func clamp32(x, lo, hi float32) float32 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
