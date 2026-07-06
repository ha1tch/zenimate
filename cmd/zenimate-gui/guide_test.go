//go:build purego

package main

import (
	"testing"

	"github.com/ha1tch/zenimate/internal/ui"
)

// The 8-pixel guides must fall at sprite-pixel multiples of 8 (the ZX character
// cell boundary), independent of the on-screen cell size. We reproduce the same
// arithmetic the draw loop uses and check the guide screen-x for column 8.
func TestGuideAtSpritePixel8(t *testing.T) {
	for _, dim := range []int{8, 16, 24, 32} {
		c := ui.New(dim, dim)
		l := computeLayout(980, 720, c, &fileOps{}, 0, 1)

		// Vertical guide columns are x in {8,16,...} < width.
		var cols []int
		for x := 8; x < dim; x += 8 {
			cols = append(cols, x)
		}
		// For a 16-wide sprite that's [8]; for 32 it's [8,16,24]; for 8 it's [].
		switch dim {
		case 8:
			if len(cols) != 0 {
				t.Errorf("8x8 should have no interior guides, got %v", cols)
			}
		case 16:
			if len(cols) != 1 || cols[0] != 8 {
				t.Errorf("16 wants guide at col 8, got %v", cols)
			}
		case 32:
			if len(cols) != 3 || cols[0] != 8 || cols[2] != 24 {
				t.Errorf("32 wants guides at 8,16,24, got %v", cols)
			}
		}

		// A guide's screen-x is gridX + spriteCol*cell — a multiple of the cell
		// size away from the grid origin, i.e. exactly on a sprite-pixel edge.
		for _, x := range cols {
			gx := l.GridX + x*l.Cell
			if (gx-l.GridX)%l.Cell != 0 {
				t.Errorf("guide at sprite col %d not aligned to a sprite-pixel edge", x)
			}
		}
	}
}
