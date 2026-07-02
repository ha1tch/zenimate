package main

import (
	"fmt"
	"os"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/pkg/filepick"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

// bundleContainers teaches filepick's OSFS to browse into .zbun/.zbu bundles: it
// recognises them by extension and lists their animations (with per-entry
// metadata) by reading the manifest via the model.
type bundleContainers struct{}

func (bundleContainers) IsContainer(path string) bool {
	return model.IsBundleExt(extOf(path))
}

func (bundleContainers) ReadContainer(path string) ([]filepick.Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	b, err := model.OpenBundle(data)
	if err != nil {
		return nil, err
	}
	var out []filepick.Entry
	for _, e := range b.Entries() {
		meta := []filepick.MetaLine{
			{Key: "frames", Value: fmt.Sprintf("%d", e.Frames)},
			{Key: "size", Value: fmt.Sprintf("%dx%d", e.Width, e.Height)},
		}
		if e.Label != "" {
			meta = append(meta, filepick.MetaLine{Key: "label", Value: e.Label})
		}
		out = append(out, filepick.Entry{Name: e.Name, Meta: meta})
	}
	return out, nil
}

// bundleFS returns an OSFS wired to browse .zbun bundles.
func bundleFS() filepick.FS {
	return filepick.OSFS{Containers: bundleContainers{}}
}

// bundlePreview is a filepick preview hook that renders the first frame of a
// selected in-bundle animation as a small thumbnail. The container path is the
// bundle the entry lives in (empty for plain directory entries, which have no
// thumbnail).
func bundlePreview(container string, e filepick.Entry) *filepick.EntryPreview {
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

// spriteThumbnail renders frame f of a sprite into a filepick preview: one
// preview pixel per sprite pixel, coloured by the frame's attributes (ink where
// set, paper where clear). Transparent is not used, so the whole grid is opaque.
func spriteThumbnail(s *model.Sprite, f int) *filepick.EntryPreview {
	w, h := s.Width(), s.Height()
	if f < 0 || f >= s.FrameCount() {
		return nil
	}
	fr := s.Frame(f)
	px := make([]filepick.Colour, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			attr := s.AttrCellFrame(f, x/8, y/8)
			ink := int(attr & 0x07)
			paper := int((attr >> 3) & 0x07)
			bright := attr&0x40 != 0
			idx := paper
			if fr[y*w+x] {
				idx = ink
			}
			col := zxpalette.Colour(idx, bright)
			px[y*w+x] = filepick.Colour{R: col.R, G: col.G, B: col.B, A: 0xFF}
		}
	}
	return &filepick.EntryPreview{W: w, H: h, Pixels: px}
}
