//go:build purego

package main

import (
	"math"
	"testing"
)

// spriteUnderCursor returns the sprite-pixel coordinate currently under the
// cursor for a given viewport, origin and base cell.
func spriteUnderCursor(v *viewport, mx, my, ox, oy, baseCell int) (float64, float64) {
	cell := float64(baseCell) * float64(v.zoom)
	gx := float64(ox) + float64(v.panX)
	gy := float64(oy) + float64(v.panY)
	return (float64(mx) - gx) / cell, (float64(my) - gy) / cell
}

// Zooming must keep the sprite point under the cursor fixed (cursor-anchored).
func TestZoomAnchorsAtCursor(t *testing.T) {
	v := newViewport()
	ox, oy, base := 100, 100, 20
	mx, my := 250, 180

	bx, by := spriteUnderCursor(v, mx, my, ox, oy, base)

	// Apply several zoom-in frames (positive wheel once, then inertia frames).
	v.step(1.0/60, 1.0, mx, my, ox, oy, base, false)
	for i := 0; i < 30; i++ {
		v.step(1.0/60, 0, mx, my, ox, oy, base, false)
	}

	ax, ay := spriteUnderCursor(v, mx, my, ox, oy, base)
	if math.Abs(ax-bx) > 0.05 || math.Abs(ay-by) > 0.05 {
		t.Fatalf("zoom not anchored: before (%.3f,%.3f) after (%.3f,%.3f)", bx, by, ax, ay)
	}
	if v.zoom <= 1.0 {
		t.Fatalf("zoom should have increased, got %.3f", v.zoom)
	}
}

func TestZoomClamps(t *testing.T) {
	v := newViewport()
	// Hammer zoom-in well past the max.
	for i := 0; i < 200; i++ {
		v.step(1.0/60, 5.0, 200, 200, 0, 0, 20, false)
	}
	if v.zoom > maxZoom+1e-3 {
		t.Fatalf("zoom exceeded max: %.3f", v.zoom)
	}
	// Hammer zoom-out past the min.
	for i := 0; i < 400; i++ {
		v.step(1.0/60, -5.0, 200, 200, 0, 0, 20, false)
	}
	if v.zoom < minZoom-1e-3 {
		t.Fatalf("zoom below min: %.3f", v.zoom)
	}
}

func TestPanInertiaDecaysToStop(t *testing.T) {
	v := newViewport()
	ox, oy, base := 0, 0, 20
	// Simulate a quick drag: three frames moving right+down 30px each.
	v.step(1.0/60, 0, 0, 0, ox, oy, base, true)
	v.step(1.0/60, 0, 30, 30, ox, oy, base, true)
	v.step(1.0/60, 0, 60, 60, ox, oy, base, true)
	if v.panVelX <= 0 || v.panVelY <= 0 {
		t.Fatalf("expected positive pan velocity after drag, got (%.1f,%.1f)", v.panVelX, v.panVelY)
	}
	panAtRelease := v.panX
	// Release: inertia frames should keep moving briefly, then stop.
	moved := false
	for i := 0; i < 240; i++ {
		before := v.panX
		v.step(1.0/60, 0, 60, 60, ox, oy, base, false)
		if v.panX != before {
			moved = true
		}
	}
	if !moved {
		t.Fatal("expected pan to glide after release")
	}
	if v.panVelX != 0 || v.panVelY != 0 {
		t.Fatalf("pan velocity should decay to zero, got (%.3f,%.3f)", v.panVelX, v.panVelY)
	}
	if v.panX <= panAtRelease {
		t.Fatal("inertia should have carried the pan further right after release")
	}
}

func TestAnimateFitConvergesAndCentres(t *testing.T) {
	v := newViewport()
	// 32x24 cell sprite (256x192 px) into a 400x300 box at base cell 10.
	spriteW, spriteH := 256, 192
	boxW, boxH, base := 400, 300, 10
	v.animateFit(spriteW, spriteH, boxW, boxH, base)
	if !v.animating {
		t.Fatal("animateFit should set animating")
	}
	// Run frames until it settles (cap to avoid infinite loop).
	for i := 0; i < 600 && v.animating; i++ {
		v.step(1.0/60, 0, 0, 0, 0, 0, base, false)
	}
	if v.animating {
		t.Fatal("fit animation did not converge")
	}
	// Fit zoom: 400/2560=0.156 vs 300/1920=0.156 -> ~0.156, clamped to >=minZoom.
	cell := float32(base) * v.zoom
	gw := float32(spriteW) * cell
	gh := float32(spriteH) * cell
	if gw > float32(boxW)+0.5 || gh > float32(boxH)+0.5 {
		t.Fatalf("fitted sprite %vx%v exceeds box %dx%d", gw, gh, boxW, boxH)
	}
	// Centred: left and right slack approximately equal.
	if abs32(v.panX-(float32(boxW)-gw)/2) > 0.5 {
		t.Errorf("not horizontally centred: panX=%v", v.panX)
	}
	if abs32(v.panY-(float32(boxH)-gh)/2) > 0.5 {
		t.Errorf("not vertically centred: panY=%v", v.panY)
	}
}

func TestAnimateFitCancelledByInput(t *testing.T) {
	v := newViewport()
	v.animateFit(128, 128, 400, 400, 10)
	if !v.animating {
		t.Fatal("should be animating")
	}
	v.step(1.0/60, 1.0, 200, 200, 0, 0, 10, false) // wheel input
	if v.animating {
		t.Fatal("wheel input should cancel the fit animation")
	}
}
