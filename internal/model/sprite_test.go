package model

import "testing"

func TestNewDimensionsAndEmpty(t *testing.T) {
	s := New(16, 16)
	if s.Width() != 16 || s.Height() != 16 {
		t.Fatalf("dims = %dx%d, want 16x16", s.Width(), s.Height())
	}
	for i := 0; i < DefaultFrames; i++ {
		f := s.Frame(i)
		if len(f) != 16*16 {
			t.Fatalf("frame %d len = %d, want 256", i, len(f))
		}
		for _, on := range f {
			if on {
				t.Fatalf("frame %d should start empty", i)
			}
		}
	}
}

func TestNewSnapsToCells(t *testing.T) {
	s := New(20, 7) // 20 -> 24 (snap up to whole cells), 7 -> 8
	if s.Width() != 24 {
		t.Errorf("width snap = %d, want 24", s.Width())
	}
	if s.Height() != 8 {
		t.Errorf("height snap = %d, want 8", s.Height())
	}
	// Clamp to the maximum (32x24 cells = 256x192 px).
	big := New(9999, 9999)
	if big.Width() != MaxWidth || big.Height() != MaxHeight {
		t.Errorf("max clamp = %dx%d, want %dx%d", big.Width(), big.Height(), MaxWidth, MaxHeight)
	}
}

func TestToggleAndAt(t *testing.T) {
	s := New(8, 8)
	if s.At(3, 4) {
		t.Fatal("pixel should start off")
	}
	s.Toggle(3, 4)
	if !s.At(3, 4) {
		t.Fatal("pixel should be on after toggle")
	}
	s.Toggle(3, 4)
	if s.At(3, 4) {
		t.Fatal("pixel should be off after second toggle")
	}
	// Out of range is ignored, not panicking.
	s.Toggle(-1, 0)
	s.Toggle(8, 0)
	s.Toggle(0, 8)
}

func TestNotifyFires(t *testing.T) {
	s := New(8, 8)
	n := 0
	s.OnChange(func() { n++ })
	s.Toggle(0, 0) // 1
	s.Select(2)    // 2
	s.SetName("x") // 3
	if n != 3 {
		t.Fatalf("onChange fired %d times, want 3", n)
	}
	// Selecting the same frame should not fire.
	before := n
	s.Select(2)
	if n != before {
		t.Fatal("selecting the current frame should not notify")
	}
}

func TestCopyPasteFrame(t *testing.T) {
	s := New(8, 8)
	s.Select(0)
	s.Toggle(1, 1)
	s.Toggle(2, 2)
	if !s.HasClipboard() {
		// nothing copied yet
	}
	s.CopyFrame()
	if !s.HasClipboard() {
		t.Fatal("clipboard should be set after copy")
	}
	s.Select(3)
	if s.At(1, 1) || s.At(2, 2) {
		t.Fatal("frame 3 should be empty before paste")
	}
	s.PasteFrame()
	if !s.At(1, 1) || !s.At(2, 2) {
		t.Fatal("frame 3 should match copied frame after paste")
	}
	// Paste must be a deep copy: mutating frame 3 must not affect frame 0.
	s.Toggle(1, 1)
	s.Select(0)
	if !s.At(1, 1) {
		t.Fatal("frame 0 changed after editing pasted frame 3 — not a deep copy")
	}
}

func TestAdvanceWraps(t *testing.T) {
	s := New(8, 8)
	s.Select(DefaultFrames - 1)
	s.Advance()
	if s.Selected() != 0 {
		t.Fatalf("advance from last should wrap to 0, got %d", s.Selected())
	}
}

func TestResizeIsNonDestructive(t *testing.T) {
	s := New(8, 8)
	s.Toggle(0, 0)
	s.Toggle(7, 7)
	s.CopyFrame()
	if !s.HasClipboard() {
		t.Fatal("expected clipboard set")
	}
	s.Resize(16, 16)
	if s.Width() != 16 || s.Height() != 16 {
		t.Fatalf("dims after resize = %dx%d, want 16x16", s.Width(), s.Height())
	}
	// Existing pixels within the old region must be preserved.
	if !s.At(0, 0) || !s.At(7, 7) {
		t.Fatal("resize must preserve existing pixels (non-destructive)")
	}
	// New area is empty.
	if s.At(15, 15) {
		t.Fatal("newly exposed area should be empty")
	}
	if s.HasClipboard() {
		t.Fatal("resize should drop the clipboard (geometry changed)")
	}
}

func TestResizePreservesAttributes(t *testing.T) {
	s := New(16, 16) // 2x2 cells
	s.SetAttrCell(0, 0, 0x21)
	s.SetAttrCell(1, 1, 0x35)
	s.Resize(24, 24) // 3x3 cells
	if s.AttrCell(0, 0) != 0x21 || s.AttrCell(1, 1) != 0x35 {
		t.Fatal("resize must preserve attributes in the overlapping region")
	}
	if s.AttrCell(2, 2) != DefaultAttr {
		t.Fatal("new cells should be default")
	}
}

func TestAddRemoveFrame(t *testing.T) {
	s := New(8, 8)
	start := s.FrameCount()
	if start != DefaultFrames {
		t.Fatalf("initial frame count = %d, want %d", start, DefaultFrames)
	}
	// Remove down to the minimum.
	for s.FrameCount() > MinFrames {
		if !s.RemoveFrame() {
			t.Fatal("RemoveFrame returned false before reaching min")
		}
	}
	if s.RemoveFrame() {
		t.Fatal("RemoveFrame should fail at MinFrames")
	}
	// Add up to the maximum.
	for s.FrameCount() < MaxFrames {
		if !s.AddFrame() {
			t.Fatal("AddFrame returned false before reaching max")
		}
	}
	if s.AddFrame() {
		t.Fatal("AddFrame should fail at MaxFrames")
	}
}

func TestAttributesDefaultAndSet(t *testing.T) {
	s := New(16, 16)
	if s.AttrCols() != 2 || s.AttrRows() != 2 {
		t.Fatalf("16x16 attr grid = %dx%d, want 2x2", s.AttrCols(), s.AttrRows())
	}
	// Default attribute on every cell.
	for cy := 0; cy < s.AttrRows(); cy++ {
		for cx := 0; cx < s.AttrCols(); cx++ {
			if s.AttrCell(cx, cy) != DefaultAttr {
				t.Fatalf("cell (%d,%d) attr = %#x, want default %#x", cx, cy, s.AttrCell(cx, cy), DefaultAttr)
			}
		}
	}
	// Set an attribute and read it back via pixel lookup.
	s.SetAttrCell(1, 1, 0x42)
	if s.AttrCell(1, 1) != 0x42 {
		t.Fatalf("set/get attr mismatch: %#x", s.AttrCell(1, 1))
	}
	// Pixel (8,8) is in character cell (1,1).
	if s.AttrAt(8, 8) != 0x42 {
		t.Fatalf("AttrAt(8,8) = %#x, want 0x42", s.AttrAt(8, 8))
	}
	// Pixel (0,0) is in cell (0,0) — still default.
	if s.AttrAt(0, 0) != DefaultAttr {
		t.Fatalf("AttrAt(0,0) = %#x, want default", s.AttrAt(0, 0))
	}
}

func TestAttributesGridGrowsOnResize(t *testing.T) {
	s := New(16, 16)
	s.SetAttrCell(0, 0, 0x15)
	s.Resize(32, 32)
	if s.AttrCols() != 4 || s.AttrRows() != 4 {
		t.Fatalf("32x32 attr grid = %dx%d, want 4x4", s.AttrCols(), s.AttrRows())
	}
	if s.AttrCell(0, 0) != 0x15 {
		t.Fatal("resize must preserve the existing cell attribute")
	}
}

func TestAttributesArePerFrame(t *testing.T) {
	s := New(16, 16)
	// Frame 0: set cell (0,0) to 0x12.
	s.Select(0)
	s.SetAttrCell(0, 0, 0x12)
	// Frame 1: set cell (0,0) to 0x34.
	s.Select(1)
	s.SetAttrCell(0, 0, 0x34)

	if got := s.AttrCellFrame(0, 0, 0); got != 0x12 {
		t.Errorf("frame 0 cell (0,0) = %#x, want 0x12", got)
	}
	if got := s.AttrCellFrame(1, 0, 0); got != 0x34 {
		t.Errorf("frame 1 cell (0,0) = %#x, want 0x34", got)
	}
	// Selected-frame accessor follows the selection.
	s.Select(0)
	if s.AttrCell(0, 0) != 0x12 {
		t.Error("selected accessor should read frame 0 after selecting it")
	}
	// Untouched frame 2 keeps the default.
	if s.AttrCellFrame(2, 0, 0) != DefaultAttr {
		t.Error("untouched frame should keep default attribute")
	}
}

func TestCopyPasteFrameCarriesAttributes(t *testing.T) {
	s := New(16, 16)
	s.Select(0)
	s.Set(0, 0, true)
	s.SetAttrCell(0, 0, 0x29)
	s.CopyFrame()
	s.Select(3)
	s.PasteFrame()
	if s.AttrCellFrame(3, 0, 0) != 0x29 {
		t.Errorf("pasted frame attr = %#x, want 0x29", s.AttrCellFrame(3, 0, 0))
	}
}
