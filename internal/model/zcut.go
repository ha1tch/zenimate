package model

import (
	"fmt"

	"github.com/ha1tch/zentools/pkg/scr"
)

// ZCUT (zen cut) is zenimate's native on-disk format. It is the format produced
// by zentools' scr package: an ordered collection of named "assets", each a
// bitmap plus a per-cell attribute grid. A zenimate sprite maps onto it cleanly,
// one asset per animation frame, so the round-trip is lossless: dimensions,
// every frame's pixels, and every frame's attributes are preserved.
//
// We use zentools as the single source of truth for the byte layout rather than
// inventing a parallel format, which keeps zenimate sprites interoperable with
// the rest of the toolchain (the zx CLI, KnightQuest, and so on).

// MarshalZCUT serialises the whole sprite — all frames and their attributes — to
// ZCUT bytes. Frame i becomes asset "frameNN"; the asset order is the frame
// order, so decoding restores the animation exactly.
func (s *Sprite) MarshalZCUT() ([]byte, error) {
	col := &scr.Collection{Assets: make([]scr.Asset, 0, len(s.frames))}
	for f := range s.frames {
		col.Assets = append(col.Assets, s.frameAsset(f, fmt.Sprintf("frame%02d", f)))
	}
	return scr.EncodeCollection(col)
}

// frameAsset converts frame f into an scr.Asset (bitmap + per-cell attributes),
// the shared building block for both ZCUT serialisation and screen export.
func (s *Sprite) frameAsset(f int, name string) scr.Asset {
	a := scr.Asset{
		Name:   name,
		Width:  s.width,
		Height: s.height,
		Ink:    make([][]bool, s.height),
		Attr:   make([][]scr.Attribute, s.attrRows),
	}
	fr := s.frames[f]
	for y := 0; y < s.height; y++ {
		row := make([]bool, s.width)
		for x := 0; x < s.width; x++ {
			row[x] = fr[y*s.width+x]
		}
		a.Ink[y] = row
	}
	am := s.frameAttrs[f]
	for cy := 0; cy < s.attrRows; cy++ {
		arow := make([]scr.Attribute, s.attrCols)
		for cx := 0; cx < s.attrCols; cx++ {
			arow[cx] = scr.AttributeFromByte(am[cy*s.attrCols+cx])
		}
		a.Attr[cy] = arow
	}
	return a
}

// LoadZCUT decodes ZCUT bytes into a new sprite. Every asset becomes one frame;
// all assets must share the first asset's dimensions (a sprite has a single
// size across its frames) and must be cell-aligned with attributes present, as
// MarshalZCUT always writes them. The returned sprite has frame 0 selected.
func LoadZCUT(data []byte) (*Sprite, error) {
	col, err := scr.DecodeCollection(data)
	if err != nil {
		return nil, fmt.Errorf("zcut: %w", err)
	}
	if len(col.Assets) == 0 {
		return nil, fmt.Errorf("zcut: no frames in collection")
	}

	first := col.Assets[0]
	w, h := first.Width, first.Height
	if w < MinWidth || h < MinHeight || w > MaxWidth || h > MaxHeight ||
		w%Cell != 0 || h%Cell != 0 {
		return nil, fmt.Errorf("zcut: frame size %dx%d is not a valid sprite size", w, h)
	}
	nframes := len(col.Assets)
	if nframes > MaxFrames {
		return nil, fmt.Errorf("zcut: %d frames exceeds the maximum of %d", nframes, MaxFrames)
	}

	s := &Sprite{
		width:    w,
		height:   h,
		name:     "frame",
		attrCols: w / Cell,
		attrRows: h / Cell,
	}
	s.frames = make([]Frame, nframes)
	s.frameAttrs = make([][]byte, nframes)

	for f, a := range col.Assets {
		if a.Width != w || a.Height != h {
			return nil, fmt.Errorf("zcut: frame %d is %dx%d, expected %dx%d (frames must share one size)",
				f, a.Width, a.Height, w, h)
		}
		if len(a.Ink) != h {
			return nil, fmt.Errorf("zcut: frame %d bitmap has %d rows, want %d", f, len(a.Ink), h)
		}
		fr := make(Frame, w*h)
		for y := 0; y < h; y++ {
			if len(a.Ink[y]) != w {
				return nil, fmt.Errorf("zcut: frame %d row %d has %d px, want %d", f, y, len(a.Ink[y]), w)
			}
			for x := 0; x < w; x++ {
				fr[y*w+x] = a.Ink[y][x]
			}
		}
		s.frames[f] = fr

		am := make([]byte, s.attrCols*s.attrRows)
		if a.HasAttrs() {
			if len(a.Attr) != s.attrRows {
				return nil, fmt.Errorf("zcut: frame %d attr has %d rows, want %d", f, len(a.Attr), s.attrRows)
			}
			for cy := 0; cy < s.attrRows; cy++ {
				if len(a.Attr[cy]) != s.attrCols {
					return nil, fmt.Errorf("zcut: frame %d attr row %d has %d cells, want %d",
						f, cy, len(a.Attr[cy]), s.attrCols)
				}
				for cx := 0; cx < s.attrCols; cx++ {
					am[cy*s.attrCols+cx] = a.Attr[cy][cx].Byte()
				}
			}
		} else {
			// Attribute-less asset (e.g. a mask or a bitmap-only cut): fall back to
			// the default attribute so the sprite is still well-formed.
			for i := range am {
				am[i] = DefaultAttr
			}
		}
		s.frameAttrs[f] = am
	}

	s.selected = 0
	return s, nil
}
