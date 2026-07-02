package main

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

func renderString(ed *editor, cursorOn bool) string {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	render(w, ed, cursorOn)
	w.Flush()
	return buf.String()
}

// The blinking cursor is the original xterm-256 bright red (index 196), in every
// mode, and only while blink is on.
func TestCursorIsXterm256Red(t *testing.T) {
	c := ui.New(8, 8)
	ed := &editor{c: c}
	redSeq := fmt.Sprintf("[38;5;%dm", cursorRed) // 196
	if !strings.Contains(renderString(ed, true), redSeq) {
		t.Fatal("cursor should be xterm-256 red (fg 196) while blink on")
	}
	if strings.Contains(renderString(ed, false), redSeq) {
		t.Fatal("cursor should not be red while blink off")
	}
}

// The transparency chequer uses the original xterm-256 greys 252/245.
func TestChequerIsOriginalGreys(t *testing.T) {
	c := ui.New(8, 8) // empty sprite -> all chequer
	ed := &editor{c: c}
	out := renderString(ed, false)
	if !strings.Contains(out, fmt.Sprintf("[38;5;%dm", chkLight)) { // 252
		t.Error("chequer light grey 252 missing")
	}
	if !strings.Contains(out, fmt.Sprintf("[48;5;%dm", chkDark)) { // 245
		t.Error("chequer dark grey 245 missing")
	}
}

// The three view modes render a set pixel differently, mirroring the GUI:
// bitmap modes in xterm-256, Spectrum Colour in truecolor.
func TestThreeViewModes(t *testing.T) {
	c := ui.New(8, 8)
	c.Set(2, 2, true)
	ed := &editor{c: c}

	c.SetMode(ui.BitmapBlack)
	if !strings.Contains(renderString(ed, false), fmt.Sprintf("[38;5;%dm", blackIdx)) {
		t.Error("Bitmap Black should draw the set pixel in xterm-256 black (16)")
	}
	c.SetMode(ui.BitmapWhite)
	if !strings.Contains(renderString(ed, false), fmt.Sprintf("[38;5;%dm", inkIdx)) {
		t.Error("Bitmap White should draw the set pixel in xterm-256 white (231)")
	}
	c.SetMode(ui.SpectrumColour)
	c.SetCellInk(0, 0, zxpalette.Red)
	redX := zxpalette.Xterm256(zxpalette.Red, false)
	if !strings.Contains(renderString(ed, false), fmt.Sprintf("[38;5;%dm", redX)) {
		t.Error("Spectrum Colour should draw the set pixel in the ink's xterm-256 colour")
	}
	// And it must NOT emit any 24-bit truecolour sequences.
	if strings.Contains(renderString(ed, false), "[38;2;") {
		t.Error("TUI must not emit 24-bit truecolour sequences")
	}
}

// The grid must contain the upper half block glyph.
func TestGridUsesHalfBlock(t *testing.T) {
	ed := &editor{c: ui.New(8, 8)}
	if !strings.Contains(renderString(ed, false), "\u2580") {
		t.Fatal("expected upper half block in grid")
	}
}

func TestColourPaintPreservesPixels(t *testing.T) {
	c := ui.New(8, 8)
	c.Set(1, 1, true)
	ed := &editor{c: c, cx: 1, cy: 1}
	c.SetInk(zxpalette.Red)
	c.SetPaper(zxpalette.Yellow)
	ed.stampColour()
	if !c.Sprite.At(1, 1) {
		t.Fatal("colour stamp must not clear the pixel")
	}
	attr := c.Sprite.AttrCell(0, 0)
	if zxpalette.Ink(attr) != zxpalette.Red || zxpalette.Paper(attr) != zxpalette.Yellow {
		t.Errorf("stamp did not set ink/paper: attr=0x%02X", attr)
	}
}

func TestClearBitmapKeepsColour(t *testing.T) {
	c := ui.New(8, 8)
	c.SetCellInk(0, 0, zxpalette.Cyan)
	c.Set(2, 2, true)
	c.ClearFrameBitmap()
	if c.Sprite.At(2, 2) {
		t.Error("clear-bitmap should have cleared the pixel")
	}
	if zxpalette.Ink(c.Sprite.AttrCell(0, 0)) != zxpalette.Cyan {
		t.Error("clear-bitmap must preserve the cell ink colour")
	}
}
