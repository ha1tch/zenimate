package main

import (
	"image/color"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/bdf"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

// textEntry is the text tool's in-progress state: a click starts one,
// typing inserts at the cursor, Enter inserts a newline (multi-line text),
// arrow keys move the cursor, and clicking elsewhere or switching tools
// commits it into the sprite's bitmap. Escape discards it. Unlike a
// selection, its position is fixed at the original click point until
// committed — it doesn't float/drag.
type textEntry struct {
	active bool
	x, y   int
	text   string // may contain '\n' for multiple lines
	cursor int    // rune index into text
	// desiredCol is the pixel column remembered across consecutive Up/Down
	// presses, so moving up then down returns to the original horizontal
	// position rather than snapping to a fixed character index — the same
	// "sticky column" behaviour ordinary text editors use. Reset to -1
	// (unset) by any edit or horizontal move, so the next vertical move
	// picks it up fresh from the cursor's actual position.
	desiredCol int
	// attrPaint is captured once, when the entry starts (Ctrl held at the
	// click, or the global ATTR ON toggle active) — not re-checked at
	// commit time, since Ctrl may no longer be held by then. When true,
	// commitText also stamps the current ink/paper onto every touched
	// cell; when false (the default), only the bitmap is touched and
	// whatever attribute was already there is left alone.
	attrPaint bool
}

// lineBounds returns the rune-index [start,end) of the line containing rune
// index at within runes — end excludes the newline itself, if any.
func lineBounds(runes []rune, at int) (start, end int) {
	start = at
	for start > 0 && runes[start-1] != '\n' {
		start--
	}
	end = at
	for end < len(runes) && runes[end] != '\n' {
		end++
	}
	return start, end
}

// pixelColAt returns the pixel X offset of rune index at within its own
// line, summing each character's own advance width rather than assuming
// index*CellWidth — correct for today's monospace font, and for a
// proportional font if one is ever used.
func pixelColAt(font *bdf.Font, runes []rune, at int) int {
	start, _ := lineBounds(runes, at)
	col := 0
	for i := start; i < at; i++ {
		col += font.CellWidth
	}
	return col
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// Update captures typing and cursor movement for an active entry. Returns
// whether the entry should be discarded (Escape) this frame — committing is
// no longer Update's concern: clicking elsewhere or switching tools already
// commits (see main.go), and Enter now inserts a newline instead.
func (t *textEntry) Update(in zenui.Input, font *bdf.Font) (cancel bool) {
	if !t.active {
		return false
	}
	for _, r := range in.Chars {
		if r >= 0x20 && r != 0x7f {
			t.insertRune(r)
		}
	}
	for _, k := range in.Keys {
		switch k {
		case zenui.KeyBackspace:
			t.backspace()
		case zenui.KeyEnter:
			t.insertRune('\n')
		case zenui.KeyEscape:
			cancel = true
		case zenui.KeyLeft:
			if t.cursor > 0 {
				t.cursor--
			}
			t.desiredCol = -1
		case zenui.KeyRight:
			if t.cursor < len([]rune(t.text)) {
				t.cursor++
			}
			t.desiredCol = -1
		case zenui.KeyUp:
			t.moveVertical(font, -1)
		case zenui.KeyDown:
			t.moveVertical(font, 1)
		}
	}
	return cancel
}

func (t *textEntry) insertRune(r rune) {
	rs := []rune(t.text)
	next := make([]rune, 0, len(rs)+1)
	next = append(next, rs[:t.cursor]...)
	next = append(next, r)
	next = append(next, rs[t.cursor:]...)
	t.text = string(next)
	t.cursor++
	t.desiredCol = -1
}

func (t *textEntry) backspace() {
	if t.cursor == 0 {
		return
	}
	rs := []rune(t.text)
	next := make([]rune, 0, len(rs)-1)
	next = append(next, rs[:t.cursor-1]...)
	next = append(next, rs[t.cursor:]...)
	t.text = string(next)
	t.cursor--
	t.desiredCol = -1
}

// moveVertical moves the cursor to the closest pixel column on the previous
// (dir<0) or next (dir>0) line. A no-op if there is no such line.
func (t *textEntry) moveVertical(font *bdf.Font, dir int) {
	runes := []rune(t.text)
	curStart, curEnd := lineBounds(runes, t.cursor)

	col := t.desiredCol
	if col < 0 {
		col = pixelColAt(font, runes, t.cursor)
	}

	var targetStart int
	if dir < 0 {
		if curStart == 0 {
			return
		}
		targetStart, _ = lineBounds(runes, curStart-1)
	} else {
		if curEnd >= len(runes) {
			return
		}
		targetStart = curEnd + 1
	}
	_, targetEnd := lineBounds(runes, targetStart)

	best := targetStart
	bestDist := absInt(pixelColAt(font, runes, targetStart) - col)
	for i := targetStart + 1; i <= targetEnd; i++ {
		d := absInt(pixelColAt(font, runes, i) - col)
		if d < bestDist {
			bestDist = d
			best = i
		}
	}
	t.cursor = best
	t.desiredCol = col // preserved across consecutive up/down moves
}

// forEachGlyph calls fn(gx, gy, on) for every "on" pixel of every character
// in t.text, laid out multi-line (each '\n' advances to a new line, same
// left edge, font.CellHeight further down) — the single shared walk used by
// commitText, drawTextPreview, and the cursor-position renderer, so all
// three agree on layout by construction rather than by convention.
func forEachGlyph(font *bdf.Font, text string, ink color.NRGBA, fn func(gx, gy int)) {
	cx, cy := 0, 0
	for _, r := range text {
		if r == '\n' {
			cx = 0
			cy += font.CellHeight
			continue
		}
		img, ok := font.GlyphImage(r, ink)
		if ok {
			b := img.Bounds()
			for py := 0; py < b.Dy(); py++ {
				for px := 0; px < b.Dx(); px++ {
					_, _, _, a := img.At(b.Min.X+px, b.Min.Y+py).RGBA()
					if a == 0 {
						continue
					}
					fn(cx+px, cy+py)
				}
			}
		}
		cx += font.CellWidth
	}
}

// cursorPixelPos returns the cursor's current position in local (relative to
// t.x,t.y) pixel coordinates — used for both the preview's cursor dash and
// (after committing) is simply discarded, since a committed entry has no
// cursor to draw.
func (t *textEntry) cursorPixelPos(font *bdf.Font) (px, py int) {
	runes := []rune(t.text)
	lineIdx := 0
	for i := 0; i < t.cursor; i++ {
		if runes[i] == '\n' {
			lineIdx++
		}
	}
	col := pixelColAt(font, runes, t.cursor)
	return col, lineIdx * font.CellHeight
}

// commitText rasterises t's string into the sprite's bitmap using font,
// starting at t's position, bounds-clipped to the sprite — characters (or
// parts of characters) that fall outside it are silently dropped, matching
// this app's existing bounds-checked painting conventions. One Checkpoint
// covers the whole string (all lines) as a single undo step.
func commitText(c *ui.Controller, font *bdf.Font, t *textEntry) {
	if t.text == "" {
		return
	}
	c.Checkpoint()
	ink := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	forEachGlyph(font, t.text, ink, func(gx, gy int) {
		px, py := t.x+gx, t.y+gy
		if px < 0 || py < 0 || px >= c.Sprite.Width() || py >= c.Sprite.Height() {
			return
		}
		c.PaintUnclipped(px, py, true)
		if t.attrPaint {
			c.PaintAttrUnclipped(px, py)
		}
	})
}

// drawTextPreview renders t's current text as a live, uncommitted preview at
// its canvas position, plus a blinking-free vertical-dash cursor at the
// current edit point — the exact same glyph-walking as commitText for the
// text itself, so the preview is guaranteed pixel-identical to what
// committing will actually produce. Clipped to the grid viewport, matching
// every other canvas overlay.
func drawTextPreview(font *bdf.Font, t *textEntry, gridX, gridY, gridW, gridH int32, ox, oy, cell float32, ink rl.Color) {
	if !t.active {
		return
	}
	rl.BeginScissorMode(gridX, gridY, gridW, gridH)
	defer rl.EndScissorMode()

	if t.text != "" {
		glyphInk := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		forEachGlyph(font, t.text, glyphInk, func(gx, gy int) {
			px, py := t.x+gx, t.y+gy
			rl.DrawRectangleRec(rl.NewRectangle(
				ox+float32(px)*cell, oy+float32(py)*cell, cell+0.5, cell+0.5,
			), ink)
		})
	}

	// Cursor: a vertical dash one pixel wide, one character cell tall, at
	// the current edit point — visible even on an empty line, unlike the
	// glyph walk above which draws nothing for an empty string.
	cpx, cpy := t.cursorPixelPos(font)
	cx, cy := t.x+cpx, t.y+cpy
	rl.DrawRectangleRec(rl.NewRectangle(
		ox+float32(cx)*cell, oy+float32(cy)*cell, cell*0.5+0.5, cell*float32(font.CellHeight)+0.5,
	), ink)
}
