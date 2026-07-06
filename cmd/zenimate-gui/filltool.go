package main

import "github.com/ha1tch/zenimate/internal/ui"

// floodFill sets every pixel 4-connected to (startX,startY) that shares its
// current on/off state, to the opposite state — matching the paint tool's own
// left-click=on / right-click=off convention, just applied to a whole region
// instead of a single stroke. A no-op if the start pixel is out of bounds or
// already at the target state (clicking an already-filled region does
// nothing, rather than needlessly pushing a Checkpoint).
//
// attrPaint mirrors the paintbrush's own Ctrl-attribute-paint gesture: when
// true, the filled region's cells get the current ink/paper stamped instead
// of the bitmap being touched.
//
// Collects every affected coordinate first, then applies them as a single
// Checkpoint-guarded batch — one undo step fills the whole region, not one
// step per pixel.
func floodFill(c *ui.Controller, startX, startY int, fillOn, attrPaint bool) {
	s := c.Sprite
	w, h := s.Width(), s.Height()
	if startX < 0 || startY < 0 || startX >= w || startY >= h {
		return
	}
	target := s.At(startX, startY)
	if target == fillOn {
		return // already the fill state: nothing to do
	}

	visited := make([]bool, w*h)
	idx := func(x, y int) int { return y*w + x }
	visited[idx(startX, startY)] = true

	queue := []struct{ x, y int }{{startX, startY}}
	var toFill []struct{ x, y int }
	for len(queue) > 0 {
		p := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		toFill = append(toFill, p)

		neighbours := [4]struct{ x, y int }{
			{p.x - 1, p.y}, {p.x + 1, p.y}, {p.x, p.y - 1}, {p.x, p.y + 1},
		}
		for _, n := range neighbours {
			if n.x < 0 || n.y < 0 || n.x >= w || n.y >= h {
				continue
			}
			if visited[idx(n.x, n.y)] {
				continue
			}
			visited[idx(n.x, n.y)] = true
			if s.At(n.x, n.y) == target {
				queue = append(queue, n)
			}
		}
	}

	c.Checkpoint()
	for _, p := range toFill {
		if attrPaint {
			c.PaintAttr(p.x, p.y)
		} else {
			c.Paint(p.x, p.y, fillOn)
		}
	}
}
