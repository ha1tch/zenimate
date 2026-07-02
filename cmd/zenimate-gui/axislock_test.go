//go:build purego

package main

import "testing"

func TestLockAxis(t *testing.T) {
	// Horizontal-dominant move locks to the anchor's row (py = anchorY).
	px, py, ax := lockAxis(10, 10, 15, 12, axisNone)
	if ax != axisH || px != 15 || py != 10 {
		t.Errorf("horizontal-dominant: got px=%d py=%d ax=%d, want 15,10,axisH", px, py, ax)
	}
	// Vertical-dominant move locks to the anchor's column (px = anchorX).
	px, py, ax = lockAxis(10, 10, 12, 16, axisNone)
	if ax != axisV || px != 10 || py != 16 {
		t.Errorf("vertical-dominant: got px=%d py=%d ax=%d, want 10,16,axisV", px, py, ax)
	}
	// Tie (|dx| == |dy|) breaks to horizontal (>=).
	_, _, ax = lockAxis(10, 10, 13, 13, axisNone)
	if ax != axisH {
		t.Errorf("tie should pick axisH, got %d", ax)
	}
	// No movement: axis stays undecided, pixel passes through.
	px, py, ax = lockAxis(10, 10, 10, 10, axisNone)
	if ax != axisNone || px != 10 || py != 10 {
		t.Errorf("no-move: got px=%d py=%d ax=%d, want 10,10,axisNone", px, py, ax)
	}
	// Already-locked axis is held even if the dominant direction changes: locked
	// horizontal, then a big vertical move still clamps py to the anchor row.
	px, py, ax = lockAxis(10, 10, 11, 99, axisH)
	if ax != axisH || px != 11 || py != 10 {
		t.Errorf("held horizontal: got px=%d py=%d ax=%d, want 11,10,axisH", px, py, ax)
	}
}
