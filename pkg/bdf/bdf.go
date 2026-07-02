// Package bdf is a standalone reader for BDF (Glyph Bitmap Distribution Format)
// bitmap fonts. It parses a BDF file into a set of decoded glyphs and rasterises
// each glyph into a full-cell RGBA pixmap, ink-coloured on a transparent ground.
//
// BDF is the classic X11 bitmap font format. Each glyph is a block:
//
//	STARTCHAR <name>
//	ENCODING <codepoint>
//	DWIDTH <dx> <dy>            ; device advance width
//	BBX <w> <h> <xoff> <yoff>  ; bitmap bounding box + offset from origin
//	BITMAP
//	<hexrow> ...               ; h rows, each ceil(w/8)*2 hex digits, byte-aligned
//	ENDCHAR
//
// A BDF font also carries a global FONTBOUNDINGBOX that, together with
// FONT_ASCENT / FONT_DESCENT, fixes the cell box every glyph is placed into.
//
// This package has no dependency beyond the standard library. It was lifted from
// the subterm terminal renderer's bdf.go, with the buffer/registry coupling
// removed so it can be reused on its own.
package bdf

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"strconv"
	"strings"
)

// Font holds a parsed BDF font: its cell metrics and decoded glyphs.
type Font struct {
	Name       string
	CellWidth  int // glyph advance (DWIDTH) — the true monospace cell width
	CellHeight int // FONT_ASCENT+FONT_DESCENT — the cell height
	bbWidth    int // FONTBOUNDINGBOX width (union extent; not the advance)
	ascent     int // FONT_ASCENT (rows above baseline)
	descent    int // FONT_DESCENT (rows below baseline)
	glyphs     map[rune]glyph
}

type glyph struct {
	width, height int
	xoff, yoff    int
	dwidth        int
	rows          []uint32 // each row left-justified in low `width` bits
}

// ParseFile parses a BDF file at path into a Font.
func ParseFile(path string) (*Font, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse parses a BDF font from r.
func Parse(r io.Reader) (*Font, error) {
	bf := &Font{glyphs: make(map[rune]glyph)}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)

	var (
		inChar   bool
		inBitmap bool
		cur      glyph
		curCP    rune
		bbW, bbH int
		bbX, bbY int
		rowAcc   []uint32
	)

	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), " \r\n")
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		key := fields[0]

		if inBitmap {
			if key == "ENDCHAR" {
				cur.rows = rowAcc
				cur.width, cur.height = bbW, bbH
				cur.xoff, cur.yoff = bbX, bbY
				bf.glyphs[curCP] = cur
				inBitmap, inChar = false, false
				continue
			}
			// A bitmap hex row: ceil(width/8) bytes = len/2 bytes. Left-justify
			// the `width` significant bits into a uint32.
			v, err := strconv.ParseUint(line, 16, 64)
			if err == nil {
				bytes := len(line) / 2
				totalBits := bytes * 8
				shift := totalBits - bbW
				if shift < 0 {
					shift = 0
				}
				rowAcc = append(rowAcc, uint32(v>>uint(shift)))
			}
			continue
		}

		switch key {
		case "FONT":
			if len(fields) > 1 {
				bf.Name = fields[1]
			}
		case "FONTBOUNDINGBOX":
			if len(fields) >= 3 {
				bf.bbWidth, _ = strconv.Atoi(fields[1])
				bf.CellHeight, _ = strconv.Atoi(fields[2])
			}
		case "FONT_ASCENT":
			if len(fields) >= 2 {
				bf.ascent, _ = strconv.Atoi(fields[1])
			}
		case "FONT_DESCENT":
			if len(fields) >= 2 {
				bf.descent, _ = strconv.Atoi(fields[1])
			}
		case "STARTCHAR":
			inChar = true
			cur = glyph{}
			rowAcc = nil
		case "ENCODING":
			if inChar && len(fields) >= 2 {
				cp, _ := strconv.Atoi(fields[1])
				curCP = rune(cp)
			}
		case "DWIDTH":
			if inChar && len(fields) >= 2 {
				cur.dwidth, _ = strconv.Atoi(fields[1])
				// Monospace: the advance is constant; capture it once as the
				// true cell width (FONTBOUNDINGBOX over-counts via outliers).
				if bf.CellWidth == 0 && cur.dwidth > 0 {
					bf.CellWidth = cur.dwidth
				}
			}
		case "BBX":
			if inChar && len(fields) >= 5 {
				bbW, _ = strconv.Atoi(fields[1])
				bbH, _ = strconv.Atoi(fields[2])
				bbX, _ = strconv.Atoi(fields[3])
				bbY, _ = strconv.Atoi(fields[4])
			}
		case "BITMAP":
			if inChar {
				inBitmap = true
				rowAcc = nil
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	// Cell height is best derived from ascent+descent (the line box); fall back
	// to FONTBOUNDINGBOX height, then a default.
	if bf.ascent > 0 && bf.descent >= 0 {
		bf.CellHeight = bf.ascent + bf.descent
	}
	if bf.CellHeight == 0 {
		bf.CellHeight = 16
	}
	if bf.CellWidth == 0 {
		bf.CellWidth = bf.bbWidth
	}
	if bf.CellWidth == 0 {
		bf.CellWidth = 8
	}
	if bf.ascent == 0 {
		bf.ascent = bf.CellHeight * 3 / 4
	}
	if len(bf.glyphs) == 0 {
		return nil, fmt.Errorf("bdf: no glyphs parsed")
	}
	return bf, nil
}

// Has reports whether the font has a glyph for r.
func (f *Font) Has(r rune) bool {
	_, ok := f.glyphs[r]
	return ok
}

// Glyphs returns the number of decoded glyphs.
func (f *Font) Glyphs() int { return len(f.glyphs) }

// GlyphImage renders one glyph into a full-cell RGBA pixmap, placing the glyph's
// bitmap at the correct offset within the cell box (using BBX offsets and the
// font ascent), ink-coloured on a transparent ground. Pixels outside the cell
// box are clipped. If the font has no glyph for r, a blank (fully transparent)
// cell-sized image is returned and ok is false.
func (f *Font) GlyphImage(r rune, ink color.NRGBA) (img *image.RGBA, ok bool) {
	cw, ch := f.CellWidth, f.CellHeight
	if cw < 1 {
		cw = 8
	}
	if ch < 1 {
		ch = 16
	}
	img = image.NewRGBA(image.Rect(0, 0, cw, ch))

	g, present := f.glyphs[r]
	if !present {
		return img, false
	}

	// Vertical placement: the BDF origin is the baseline; yoff is the distance
	// from baseline to the bottom of the bitmap. The cell baseline sits `ascent`
	// rows from the top. Top of glyph = ascent - (height + yoff).
	top := f.ascent - (g.height + g.yoff)
	left := g.xoff
	set := color.RGBA{ink.R, ink.G, ink.B, ink.A}
	for r := 0; r < g.height; r++ {
		if r >= len(g.rows) {
			break
		}
		row := g.rows[r]
		for c := 0; c < g.width; c++ {
			if row&(1<<uint(g.width-1-c)) != 0 {
				x := left + c
				y := top + r
				if x >= 0 && y >= 0 && x < cw && y < ch {
					img.SetRGBA(x, y, set)
				}
			}
		}
	}
	return img, true
}
