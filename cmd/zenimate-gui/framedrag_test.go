//go:build purego

package main

import (
	"testing"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func fiveRects() []rl.Rectangle {
	// Five buttons, width 56, gap 6, starting at x=100 — matches frameBtnW/frameGap.
	rects := make([]rl.Rectangle, 5)
	for i := range rects {
		rects[i] = rl.NewRectangle(float32(100+i*62), 50, 56, 24)
	}
	return rects
}

func TestFrameDropGap(t *testing.T) {
	rects := fiveRects()
	cases := []struct {
		x    float32
		want int
	}{
		{50, 0},   // before everything
		{120, 0},  // inside button 0 (100..156, mid=128), clearly left of midpoint
		{135, 1},  // inside button 0, clearly right of midpoint
		{180, 1},  // inside button 1 (162..218, mid=190), clearly left of midpoint
		{200, 2},  // inside button 1, clearly right of midpoint
		{1000, 5}, // past everything
	}
	for _, c := range cases {
		if got := frameDropGap(rects, c.x); got != c.want {
			t.Errorf("frameDropGap(x=%v) = %d, want %d", c.x, got, c.want)
		}
	}
}

// TestDragGapToMoveTarget hand-verifies against the same four directional
// cases already confirmed for MoveFrame itself (internal/model/sprite_test.go).
func TestDragGapToMoveTarget(t *testing.T) {
	cases := []struct {
		source, gap, want int
	}{
		{1, 3, 2}, // drop after C/before D -> [A,C,B,D,E]
		{3, 1, 1}, // drop after A/before B -> [A,D,B,C,E]
		{1, 1, 1}, // drop right before own original slot -> no-op
		{1, 2, 1}, // drop right after own original slot -> no-op
		{0, 0, 0}, // drop before everything, already frame 0 -> no-op
	}
	for _, c := range cases {
		if got := dragGapToMoveTarget(c.source, c.gap); got != c.want {
			t.Errorf("dragGapToMoveTarget(source=%d, gap=%d) = %d, want %d",
				c.source, c.gap, got, c.want)
		}
	}
}
