package model

import "testing"

// setPixels turns on a list of (x,y) pixels in the selected frame.
func setPixels(s *Sprite, pts ...[2]int) {
	for _, p := range pts {
		s.Set(p[0], p[1], true)
	}
}

func TestFlipH(t *testing.T) {
	s := New(16, 16) // 16x16
	s.Set(0, 0, true)
	s.Set(1, 3, true)
	s.FlipH()
	if !s.At(15, 0) {
		t.Error("(0,0) should map to (15,0) after H flip")
	}
	if !s.At(14, 3) {
		t.Error("(1,3) should map to (14,3) after H flip")
	}
	if s.At(0, 0) {
		t.Error("(0,0) should be empty after H flip")
	}
}

func TestFlipV(t *testing.T) {
	s := New(16, 16)
	s.Set(2, 0, true)
	s.FlipV()
	if !s.At(2, 15) {
		t.Error("(2,0) should map to (2,15) after V flip")
	}
	if s.At(2, 0) {
		t.Error("(2,0) should be empty after V flip")
	}
}

func TestInvert(t *testing.T) {
	s := New(16, 16)
	s.Set(0, 0, true)
	s.Invert()
	if s.At(0, 0) {
		t.Error("set pixel should be off after invert")
	}
	if !s.At(1, 1) {
		t.Error("unset pixel should be on after invert")
	}
	// Count: exactly one pixel was on; after invert all but one are on.
	on := 0
	for y := 0; y < s.Height(); y++ {
		for x := 0; x < s.Width(); x++ {
			if s.At(x, y) {
				on++
			}
		}
	}
	if want := s.Width()*s.Height() - 1; on != want {
		t.Errorf("invert count = %d, want %d", on, want)
	}
}

func TestRotate90SquareInPlace(t *testing.T) {
	s := New(16, 16)  // square: dimensions unchanged
	s.Set(0, 0, true) // top-left
	s.Rotate90(false)
	if s.Width() != 16 || s.Height() != 16 {
		t.Fatalf("square rotate changed size to %dx%d", s.Width(), s.Height())
	}
	// Clockwise: top-left (0,0) goes to top-right (w-1,0).
	if !s.At(15, 0) {
		t.Errorf("(0,0) should rotate CW to (15,0); got grid without it")
	}
}

func TestRotate90NonSquareInPlaceKeepsSize(t *testing.T) {
	s := New(24, 8) // wide, non-square
	s.Rotate90(false)
	if s.Width() != 24 || s.Height() != 8 {
		t.Errorf("in-place rotate should keep size 24x8, got %dx%d", s.Width(), s.Height())
	}
}

func TestRotate90NonSquareResizeSwaps(t *testing.T) {
	s := New(24, 8)
	s.Rotate90(true)
	if s.Width() != 8 || s.Height() != 24 {
		t.Errorf("resize rotate should swap to 8x24, got %dx%d", s.Width(), s.Height())
	}
}

func TestRotateAttrPixelTogether(t *testing.T) {
	s := New(16, 8)
	s.Set(1, 1, true)
	s.SetAttrCell(0, 0, 0x05)
	s.Set(9, 1, true)
	s.SetAttrCell(1, 0, 0x06)
	s.Rotate90(true)
	if s.Width() != 8 || s.Height() != 16 {
		t.Fatalf("size %dx%d", s.Width(), s.Height())
	}
	found := 0
	for y := 0; y < s.Height(); y++ {
		for x := 0; x < s.Width(); x++ {
			if s.At(x, y) {
				a := s.AttrAt(x, y)
				if a != 0x05 && a != 0x06 {
					t.Errorf("pixel (%d,%d) attr %#x, colour didn't travel with it", x, y, a)
				}
				found++
			}
		}
	}
	if found != 2 {
		t.Errorf("expected 2 pixels after rotate, got %d", found)
	}
}

func TestResetAllRestoresDefaults(t *testing.T) {
	s := New(64, 48) // non-default size
	// Grow frames past default and dirty things.
	for s.FrameCount() < MaxFrames {
		s.AddFrame()
	}
	s.Set(3, 3, true)
	s.ResetAll()
	if s.Width() != DefaultWidth || s.Height() != DefaultHeight {
		t.Errorf("size after ResetAll = %dx%d, want %dx%d", s.Width(), s.Height(), DefaultWidth, DefaultHeight)
	}
	if s.FrameCount() != DefaultFrames {
		t.Errorf("frame count after ResetAll = %d, want %d", s.FrameCount(), DefaultFrames)
	}
	if s.Selected() != 0 {
		t.Errorf("selected after ResetAll = %d, want 0", s.Selected())
	}
	// Every pixel must be clear.
	for f := 0; f < s.FrameCount(); f++ {
		s.Select(f)
		for y := 0; y < s.Height(); y++ {
			for x := 0; x < s.Width(); x++ {
				if s.At(x, y) {
					t.Fatalf("pixel (%d,%d) frame %d still set after ResetAll", x, y, f)
				}
			}
		}
	}
}
