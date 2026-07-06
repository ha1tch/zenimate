//go:build purego

package main

import (
	"testing"

	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guiutil"
)

func TestLockAxis(t *testing.T) {
	// Horizontal-dominant move locks to the anchor's row (py = anchorY).
	px, py, ax := guiutil.LockAxis(10, 10, 15, 12, guiutil.AxisNone)
	if ax != guiutil.AxisH || px != 15 || py != 10 {
		t.Errorf("horizontal-dominant: got px=%d py=%d ax=%d, want 15,10,guiutil.AxisH", px, py, ax)
	}
	// Vertical-dominant move locks to the anchor's column (px = anchorX).
	px, py, ax = guiutil.LockAxis(10, 10, 12, 16, guiutil.AxisNone)
	if ax != guiutil.AxisV || px != 10 || py != 16 {
		t.Errorf("vertical-dominant: got px=%d py=%d ax=%d, want 10,16,guiutil.AxisV", px, py, ax)
	}
	// Tie (|dx| == |dy|) breaks to horizontal (>=).
	_, _, ax = guiutil.LockAxis(10, 10, 13, 13, guiutil.AxisNone)
	if ax != guiutil.AxisH {
		t.Errorf("tie should pick guiutil.AxisH, got %d", ax)
	}
	// No movement: axis stays undecided, pixel passes through.
	px, py, ax = guiutil.LockAxis(10, 10, 10, 10, guiutil.AxisNone)
	if ax != guiutil.AxisNone || px != 10 || py != 10 {
		t.Errorf("no-move: got px=%d py=%d ax=%d, want 10,10,guiutil.AxisNone", px, py, ax)
	}
	// Already-locked axis is held even if the dominant direction changes: locked
	// horizontal, then a big vertical move still clamps py to the anchor row.
	px, py, ax = guiutil.LockAxis(10, 10, 11, 99, guiutil.AxisH)
	if ax != guiutil.AxisH || px != 11 || py != 10 {
		t.Errorf("held horizontal: got px=%d py=%d ax=%d, want 11,10,guiutil.AxisH", px, py, ax)
	}
}
