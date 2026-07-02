package model

import "testing"

// paintDistinct fills a sprite with a per-frame, per-pixel and per-cell pattern
// so any frame mix-up, transposition, or attribute drift shows up on compare.
func paintDistinct(s *Sprite) {
	for f := 0; f < s.FrameCount(); f++ {
		s.Select(f)
		for y := 0; y < s.Height(); y++ {
			for x := 0; x < s.Width(); x++ {
				// A pattern that depends on frame, x and y.
				s.Set(x, y, (x+y*3+f*7)%5 == 0)
			}
		}
		for cy := 0; cy < s.AttrRows(); cy++ {
			for cx := 0; cx < s.AttrCols(); cx++ {
				ink := byte((cx + f) % 8)
				paper := byte((cy + f + 1) % 8)
				bright := (cx+cy+f)%2 == 0
				flash := (cx*cy+f)%3 == 0
				attr := ink | paper<<3
				if bright {
					attr |= 0x40
				}
				if flash {
					attr |= 0x80
				}
				s.SetAttrCell(cx, cy, attr)
			}
		}
	}
	s.Select(0)
}

// equalSprites compares two sprites pixel- and attribute-exactly across all
// frames, reporting the first divergence.
func equalSprites(t *testing.T, a, b *Sprite) {
	t.Helper()
	if a.Width() != b.Width() || a.Height() != b.Height() {
		t.Fatalf("size mismatch: %dx%d vs %dx%d", a.Width(), a.Height(), b.Width(), b.Height())
	}
	if a.FrameCount() != b.FrameCount() {
		t.Fatalf("frame count mismatch: %d vs %d", a.FrameCount(), b.FrameCount())
	}
	for f := 0; f < a.FrameCount(); f++ {
		fa, fb := a.Frame(f), b.Frame(f)
		for i := range fa {
			if fa[i] != fb[i] {
				t.Fatalf("frame %d pixel %d differs: %v vs %v", f, i, fa[i], fb[i])
			}
		}
		for cy := 0; cy < a.AttrRows(); cy++ {
			for cx := 0; cx < a.AttrCols(); cx++ {
				if a.AttrCellFrame(f, cx, cy) != b.AttrCellFrame(f, cx, cy) {
					t.Fatalf("frame %d attr (%d,%d) differs: 0x%02X vs 0x%02X",
						f, cx, cy, a.AttrCellFrame(f, cx, cy), b.AttrCellFrame(f, cx, cy))
				}
			}
		}
	}
}

func TestZCUTRoundTripDefault(t *testing.T) {
	orig := New(32, 24) // 4x3 cells, default 8 frames
	paintDistinct(orig)

	data, err := orig.MarshalZCUT()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	back, err := LoadZCUT(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	equalSprites(t, orig, back)
}

func TestZCUTRoundTripSizes(t *testing.T) {
	sizes := [][2]int{
		{MinWidth, MinHeight}, // 8x8, 1 cell
		{MaxWidth, MaxHeight}, // 256x192, full screen
		{64, 16},              // wide-ish
		{16, 96},              // tall-ish
	}
	for _, sz := range sizes {
		s := New(sz[0], sz[1])
		// Vary the frame count too: add a couple, remove handled by New default.
		s.AddFrame()
		s.AddFrame()
		paintDistinct(s)

		data, err := s.MarshalZCUT()
		if err != nil {
			t.Fatalf("%dx%d marshal: %v", sz[0], sz[1], err)
		}
		back, err := LoadZCUT(data)
		if err != nil {
			t.Fatalf("%dx%d load: %v", sz[0], sz[1], err)
		}
		equalSprites(t, s, back)
	}
}

func TestZCUTFrameCountPreserved(t *testing.T) {
	s := New(16, 16)
	// Trim to a single frame, then check the round-trip keeps exactly one.
	for s.FrameCount() > 1 {
		s.RemoveFrame()
	}
	paintDistinct(s)
	data, err := s.MarshalZCUT()
	if err != nil {
		t.Fatal(err)
	}
	back, err := LoadZCUT(data)
	if err != nil {
		t.Fatal(err)
	}
	if back.FrameCount() != 1 {
		t.Fatalf("frame count = %d, want 1", back.FrameCount())
	}
	equalSprites(t, s, back)
}

func TestZCUTSelectedResetsToZero(t *testing.T) {
	s := New(16, 16)
	s.Select(3)
	data, _ := s.MarshalZCUT()
	back, err := LoadZCUT(data)
	if err != nil {
		t.Fatal(err)
	}
	if back.Selected() != 0 {
		t.Errorf("loaded sprite selected = %d, want 0", back.Selected())
	}
}

func TestLoadZCUTRejectsGarbage(t *testing.T) {
	if _, err := LoadZCUT([]byte("not a zcut file")); err == nil {
		t.Error("expected error decoding garbage")
	}
	if _, err := LoadZCUT(nil); err == nil {
		t.Error("expected error decoding nil")
	}
}
