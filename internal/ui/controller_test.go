package ui

import (
	"testing"

	"github.com/ha1tch/zenimate/internal/model"
)

func TestPlayTickAdvances(t *testing.T) {
	c := New(8, 8)
	if c.Playing() {
		t.Fatal("should not be playing initially")
	}
	c.TogglePlay()
	if !c.Playing() {
		t.Fatal("should be playing after toggle")
	}
	start := c.Sprite.Selected()
	c.Tick()
	if c.Sprite.Selected() != (start+1)%c.Sprite.FrameCount() {
		t.Fatal("tick should advance one frame while playing")
	}
	c.TogglePlay()
	frozen := c.Sprite.Selected()
	c.Tick()
	if c.Sprite.Selected() != frozen {
		t.Fatal("tick should not advance while stopped")
	}
}

func TestNextPrevWrap(t *testing.T) {
	c := New(8, 8)
	c.SelectFrame(c.Sprite.FrameCount() - 1)
	c.NextFrame()
	if c.Sprite.Selected() != 0 {
		t.Fatal("next from last should wrap to 0")
	}
	c.PrevFrame()
	if c.Sprite.Selected() != c.Sprite.FrameCount()-1 {
		t.Fatal("prev from 0 should wrap to last")
	}
}

func TestPaintDoesNotStampAttribute(t *testing.T) {
	c := New(16, 16)
	c.SetMode(SpectrumColour)
	c.SetInk(2)
	c.Paint(3, 3, true) // plain paint must NOT touch attributes
	if c.Sprite.AttrCell(0, 0) != model.DefaultAttr {
		t.Errorf("plain Paint must not change attribute; got %#x", c.Sprite.AttrCell(0, 0))
	}
}

func TestPaintAttrStampsSelectedCell(t *testing.T) {
	c := New(16, 16)
	c.SetMode(SpectrumColour)
	c.SetInk(2)   // red
	c.SetPaper(5) // cyan
	c.SetBright(true)
	c.PaintAttr(9, 9) // cell (1,1)

	a := c.Sprite.AttrCell(1, 1)
	if int(a&0x07) != 2 {
		t.Errorf("ink = %d, want 2", a&0x07)
	}
	if int((a>>3)&0x07) != 5 {
		t.Errorf("paper = %d, want 5", (a>>3)&0x07)
	}
	if (a>>6)&1 != 1 {
		t.Error("bright not set")
	}
	// Cell (0,0) untouched.
	if c.Sprite.AttrCell(0, 0) != model.DefaultAttr {
		t.Error("only the painted cell should change")
	}
}

func TestPaintAttrIsPerFrame(t *testing.T) {
	c := New(16, 16)
	c.SetMode(SpectrumColour)
	c.SelectFrame(0)
	c.SetInk(1)
	c.PaintAttr(0, 0)
	c.SelectFrame(1)
	c.SetInk(4)
	c.PaintAttr(0, 0)
	if int(c.Sprite.AttrCellFrame(0, 0, 0)&0x07) != 1 {
		t.Error("frame 0 ink should be 1")
	}
	if int(c.Sprite.AttrCellFrame(1, 0, 0)&0x07) != 4 {
		t.Error("frame 1 ink should be 4")
	}
}

func TestOnionFrameIndicesWrap(t *testing.T) {
	c := New(8, 8) // DefaultFrames frames
	n := c.Sprite.FrameCount()
	c.SelectFrame(0)
	if c.PrevFrameIndex() != n-1 {
		t.Errorf("prev of 0 = %d, want %d (wrap)", c.PrevFrameIndex(), n-1)
	}
	if c.NextFrameIndex() != 1 {
		t.Errorf("next of 0 = %d, want 1", c.NextFrameIndex())
	}
	c.SelectFrame(n - 1)
	if c.NextFrameIndex() != 0 {
		t.Errorf("next of last = %d, want 0 (wrap)", c.NextFrameIndex())
	}
	if c.PrevFrameIndex() != n-2 {
		t.Errorf("prev of last = %d, want %d", c.PrevFrameIndex(), n-2)
	}
}

func TestOnionTogglesDefaultOff(t *testing.T) {
	c := New(8, 8)
	if c.OnionPrev() || c.OnionNext() {
		t.Fatal("onion skins should default off")
	}
	c.ToggleOnionPrev()
	c.ToggleOnionNext()
	if !c.OnionPrev() || !c.OnionNext() {
		t.Fatal("toggles should turn on")
	}
	c.ToggleOnionPrev()
	if c.OnionPrev() {
		t.Fatal("second toggle should turn off")
	}
}

func TestPaintClipsToActiveSelection(t *testing.T) {
	c := New(16, 16)
	c.SetSelection(2, 2, 5, 5) // bounds (2,2,3,3)

	c.Paint(10, 10, true) // well outside the selection
	if c.Sprite.At(10, 10) {
		t.Error("Paint outside the active selection should be clipped (a no-op)")
	}

	c.Paint(3, 3, true) // inside the selection
	if !c.Sprite.At(3, 3) {
		t.Error("Paint inside the active selection should work normally")
	}
}

func TestPaintUnaffectedWithNoSelection(t *testing.T) {
	c := New(16, 16)
	c.Paint(10, 10, true)
	if !c.Sprite.At(10, 10) {
		t.Error("Paint with no active selection should work anywhere, unclipped")
	}
}

func TestPaintAttrClipsToActiveSelection(t *testing.T) {
	c := New(16, 16)
	c.SetSelection(0, 0, 7, 7) // bounds (0,0,8,8): exactly the first attribute cell
	c.SetInk(3)

	beforeOutside := c.Sprite.AttrCell(1, 1) // the SECOND cell, outside the selection
	c.PaintAttr(9, 9)                        // clicked pixel is in cell (1,1), outside selection
	if c.Sprite.AttrCell(1, 1) != beforeOutside {
		t.Error("PaintAttr outside the active selection should be clipped (a no-op)")
	}

	beforeInside := c.Sprite.AttrCell(0, 0) // the first cell, inside the selection
	c.PaintAttr(3, 3)                       // inside the selection, cell (0,0)
	if c.Sprite.AttrCell(0, 0) == beforeInside {
		t.Error("PaintAttr inside the active selection should still work normally")
	}
}
