package main

import (
	"testing"

	"github.com/ha1tch/zenimate/internal/ui"
)

func TestFloodFillWholeEmptySprite(t *testing.T) {
	c := ui.New(16, 16)
	floodFill(c, 8, 8, true, false)
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if !c.Sprite.At(x, y) {
				t.Fatalf("pixel (%d,%d) not filled; expected the whole empty sprite to fill from any start point", x, y)
			}
		}
	}
}

// TestFloodFillRespectsWall verifies the fill doesn't leak across a solid
// dividing wall — the actual point of a 4-connected flood fill, not just
// "does it change some pixels".
func TestFloodFillRespectsWall(t *testing.T) {
	c := ui.New(16, 16)
	// Build a solid vertical wall at x=8, splitting the sprite in two.
	for y := 0; y < 16; y++ {
		c.Paint(8, y, true)
	}
	// Fill from the left side.
	floodFill(c, 2, 2, true, false)

	for y := 0; y < 16; y++ {
		for x := 0; x < 8; x++ {
			if !c.Sprite.At(x, y) {
				t.Errorf("left side (%d,%d) should be filled", x, y)
			}
		}
		for x := 9; x < 16; x++ {
			if c.Sprite.At(x, y) {
				t.Errorf("right side (%d,%d) should NOT be filled — the wall should have blocked it", x, y)
			}
		}
	}
}

func TestFloodFillNoOpWhenAlreadyAtTargetState(t *testing.T) {
	c := ui.New(16, 16) // starts empty (all off)
	before := c.CanUndo()
	floodFill(c, 5, 5, false, false) // already off; filling to off is a no-op
	if c.CanUndo() != before {
		t.Error("floodFill pushed a Checkpoint for a no-op fill (start pixel already at target state)")
	}
}

func TestFloodFillOutOfBoundsIsSafe(t *testing.T) {
	c := ui.New(16, 16)
	before := c.CanUndo()
	floodFill(c, -1, -1, true, false)
	floodFill(c, 100, 100, true, false)
	if c.CanUndo() != before {
		t.Error("floodFill pushed a Checkpoint for an out-of-bounds start point")
	}
}

func TestFloodFillPushesExactlyOneCheckpoint(t *testing.T) {
	c := ui.New(16, 16)
	floodFill(c, 8, 8, true, false) // fills everything in one call
	if !c.CanUndo() {
		t.Fatal("expected a Checkpoint to have been pushed for a real fill")
	}
	c.Undo()
	// One undo should fully revert the whole fill, not just part of it —
	// confirming it was one Checkpoint covering the whole region, not one
	// per pixel.
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if c.Sprite.At(x, y) {
				t.Fatalf("pixel (%d,%d) still set after a single Undo — fill should be one undo step", x, y)
			}
		}
	}
}

func TestFloodFillIsolatedRegionOnly(t *testing.T) {
	c := ui.New(16, 16)
	// A small enclosed box of "on" pixels forming a ring, with an "off"
	// interior — filling the interior should not spill outside the ring.
	for x := 4; x <= 8; x++ {
		c.Paint(x, 4, true)
		c.Paint(x, 8, true)
	}
	for y := 4; y <= 8; y++ {
		c.Paint(4, y, true)
		c.Paint(8, y, true)
	}
	floodFill(c, 6, 6, true, false) // interior point

	// Interior should now be filled.
	if !c.Sprite.At(6, 6) {
		t.Error("interior point (6,6) should be filled")
	}
	// Outside the ring should be untouched.
	if c.Sprite.At(0, 0) || c.Sprite.At(15, 15) {
		t.Error("fill leaked outside the enclosing ring")
	}
}
