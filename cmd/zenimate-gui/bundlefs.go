package main

import (
	"fmt"
	"os"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/pkg/zenui"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

// bundleContainers teaches zenui's OSFS to browse into .zbun/.zbu bundles: it
// recognises them by extension and lists their animations (with per-entry
// metadata) by reading the manifest via the model.
type bundleContainers struct{}

func (bundleContainers) IsContainer(path string) bool {
	return model.IsBundleExt(extOf(path))
}

func (bundleContainers) ReadContainer(path string) ([]zenui.Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	b, err := model.OpenBundle(data)
	if err != nil {
		return nil, err
	}
	var out []zenui.Entry
	for _, e := range b.Entries() {
		meta := []zenui.MetaLine{
			{Key: "frames", Value: fmt.Sprintf("%d", e.Frames)},
			{Key: "size", Value: fmt.Sprintf("%dx%d", e.Width, e.Height)},
		}
		if e.Label != "" {
			meta = append(meta, zenui.MetaLine{Key: "label", Value: e.Label})
		}
		out = append(out, zenui.Entry{Name: e.Name, Meta: meta})
	}
	return out, nil
}

// bundleFS returns an OSFS wired to browse .zbun bundles.
func bundleFS() zenui.FS {
	return zenui.OSFS{Containers: bundleContainers{}}
}

// bundlePreview is a zenui preview hook that renders the first frame of a
// selected in-bundle animation as a small thumbnail. The container path is the
// bundle the entry lives in (empty for plain directory entries, which have no
// thumbnail).
func bundlePreview(container string, e zenui.Entry) *zenui.EntryPreview {
	if container == "" {
		return nil
	}
	data, err := os.ReadFile(container)
	if err != nil {
		return nil
	}
	b, err := model.OpenBundle(data)
	if err != nil {
		return nil
	}
	raw, ok := b.EntryData(e.Name)
	if !ok {
		return nil
	}
	s, err := model.LoadZCUT(raw)
	if err != nil {
		return nil
	}
	return spriteThumbnail(s, 0)
}

// spriteThumbnail renders frame f of a sprite into a zenui preview: one
// preview pixel per sprite pixel, coloured by the frame's attributes (ink where
// set, paper where clear). Transparent is not used, so the whole grid is opaque.
func spriteThumbnail(s *model.Sprite, f int) *zenui.EntryPreview {
	w, h := s.Width(), s.Height()
	if f < 0 || f >= s.FrameCount() {
		return nil
	}
	fr := s.Frame(f)
	px := make([]zenui.Colour, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			attr := s.AttrCellFrame(f, x/8, y/8)
			ink := int(attr & 0x07)
			paper := int((attr >> 3) & 0x07)
			bright := attr&0x40 != 0
			idx := paper
			if fr.At(x, y, w) {
				idx = ink
			}
			col := zxpalette.Colour(idx, bright)
			px[y*w+x] = zenui.Colour{R: col.R, G: col.G, B: col.B, A: 0xFF}
		}
	}
	return &zenui.EntryPreview{W: w, H: h, Pixels: px}
}
