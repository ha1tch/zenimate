package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

// press feeds one key event to the editor and reports whether it asked to quit.
func press(ed *editor, b ...byte) bool {
	quit := false
	ed.handleKey(b, &quit)
	return quit
}

func TestDrawEraseToggleKeys(t *testing.T) {
	ed := &editor{c: ui.New(8, 8)}
	ed.cx, ed.cy = 3, 4
	s := ed.c.Sprite

	press(ed, ' ') // draw
	if !s.At(3, 4) {
		t.Fatal("space should set the pixel")
	}
	press(ed, 0x7f) // erase (DEL)
	if s.At(3, 4) {
		t.Fatal("backspace/DEL should clear the pixel")
	}
	press(ed, 0x08) // erase (BS) on an already-clear pixel stays clear
	if s.At(3, 4) {
		t.Fatal("erase should be idempotent")
	}
	press(ed, '\r') // toggle on
	if !s.At(3, 4) {
		t.Fatal("enter should toggle the pixel on")
	}
	press(ed, '\r') // toggle off
	if s.At(3, 4) {
		t.Fatal("enter should toggle the pixel off")
	}
}

func TestSpaceInColourModeStampsNotDraw(t *testing.T) {
	ed := &editor{c: ui.New(8, 8)}
	ed.cx, ed.cy = 2, 2
	ed.colourMode = true
	ed.c.SetInk(zxpalette.Red)
	ed.c.SetPaper(zxpalette.Yellow)

	press(ed, ' ') // in colour mode, space stamps colour, does NOT set the pixel
	if ed.c.Sprite.At(2, 2) {
		t.Error("colour-mode space must not set a pixel")
	}
	attr := ed.c.Sprite.AttrCell(0, 0)
	if zxpalette.Ink(attr) != zxpalette.Red || zxpalette.Paper(attr) != zxpalette.Yellow {
		t.Errorf("colour-mode space should stamp ink/paper, attr=0x%02X", attr)
	}
}

func TestTabCyclesModes(t *testing.T) {
	ed := &editor{c: ui.New(8, 8)}
	if ed.c.Mode() != ui.BitmapBlack {
		t.Fatal("default should be Bitmap Black")
	}
	press(ed, '\t')
	if ed.c.Mode() != ui.BitmapWhite {
		t.Error("tab should go to Bitmap White")
	}
	press(ed, '\t')
	if ed.c.Mode() != ui.SpectrumColour {
		t.Error("tab should go to Spectrum Colour")
	}
	press(ed, '\t')
	if ed.c.Mode() != ui.BitmapBlack {
		t.Error("tab should wrap back to Bitmap Black")
	}
}

// X (wipe colour) must not act until confirmed with y.
func TestWipeColourNeedsConfirm(t *testing.T) {
	ed := &editor{c: ui.New(8, 8)}
	ed.c.SetCellInk(0, 0, zxpalette.Cyan)
	ed.c.Set(1, 1, true)

	press(ed, 'X') // arms the confirm prompt; nothing wiped yet
	if ed.prompt != promptConfirm {
		t.Fatal("X should arm a confirmation prompt")
	}
	if zxpalette.Ink(ed.c.Sprite.AttrCell(0, 0)) != zxpalette.Cyan {
		t.Error("colour must survive until confirmed")
	}
	// Answer 'n' -> nothing destroyed.
	press(ed, 'n')
	if ed.prompt != promptNone {
		t.Fatal("prompt should clear after answer")
	}
	if zxpalette.Ink(ed.c.Sprite.AttrCell(0, 0)) != zxpalette.Cyan {
		t.Error("answering no must preserve colour")
	}

	// Now confirm with 'y' -> colour reset to default, pixels cleared.
	press(ed, 'X')
	press(ed, 'y')
	if ed.c.Sprite.At(1, 1) {
		t.Error("confirmed wipe should clear pixels")
	}
	if ed.c.Sprite.AttrCell(0, 0) != model.DefaultAttr {
		t.Errorf("confirmed wipe should reset colour to default, got 0x%02X", ed.c.Sprite.AttrCell(0, 0))
	}
}

// While a filename prompt is open, ordinary keys type into the field and never
// reach the editor (no accidental painting).
func TestPromptSwallowsKeys(t *testing.T) {
	ed := &editor{c: ui.New(8, 8)}
	ed.cx, ed.cy = 0, 0
	press(ed, 'S') // open save prompt (input seeded with default name)
	if ed.prompt != promptSave {
		t.Fatal("S should open the save prompt")
	}
	ed.input = "" // clear the seeded default for a clean test
	// Type "ab " — the space must go into the filename, not paint a pixel.
	press(ed, 'a')
	press(ed, 'b')
	press(ed, ' ')
	if ed.c.Sprite.At(0, 0) {
		t.Error("space during a prompt must not paint")
	}
	if ed.input != "ab " {
		t.Errorf("prompt input = %q, want %q", ed.input, "ab ")
	}
	// ESC cancels.
	press(ed, 0x1b)
	if ed.prompt != promptNone {
		t.Error("ESC should cancel the prompt")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hero.zani")

	ed := &editor{c: ui.New(24, 16)}
	// New starts with DefaultFrames; trim to exactly 3 for this test.
	for ed.c.Sprite.FrameCount() > 3 {
		ed.c.RemoveFrame()
	}
	for ed.c.Sprite.FrameCount() < 3 {
		ed.c.AddFrame()
	}
	for f := 0; f < 3; f++ {
		ed.c.SelectFrame(f)
		ed.c.Set(f, f, true)
	}
	ed.c.SelectFrame(0)
	ed.c.SetCellInk(0, 0, zxpalette.Magenta)
	ed.doSave(path)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("save did not write the file: %v", err)
	}

	// Load into a fresh editor and verify frames, pixels, and colour survived.
	ed2 := &editor{c: ui.New(8, 8)}
	ed2.doLoad(path)
	if ed2.c.Sprite.FrameCount() != 3 {
		t.Fatalf("loaded %d frames, want 3", ed2.c.Sprite.FrameCount())
	}
	for f := 0; f < 3; f++ {
		if !ed2.c.Sprite.Frame(f)[f*24+f] {
			t.Errorf("frame %d lost its pixel after round trip", f)
		}
	}
	if zxpalette.Ink(ed2.c.Sprite.AttrCell(0, 0)) != zxpalette.Magenta {
		t.Error("colour lost after save/load round trip")
	}
}

func TestLoadRejectsNonAnimation(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "pic.scr")
	os.WriteFile(p, make([]byte, 6912), 0o644)
	ed := &editor{c: ui.New(8, 8)}
	before := ed.c.Sprite.Width()
	ed.doLoad(p) // .scr is not an animation; TUI load should decline, not crash
	if ed.c.Sprite.Width() != before {
		t.Error("a non-animation load must not replace the sprite")
	}
}
