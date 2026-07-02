package main

import (
	"bufio"
	"io"
	"testing"

	"github.com/ha1tch/zenimate/internal/ui"
)

// dispatch mirrors the exact read->handleKey pipeline in run(): read into an
// 8-byte buffer via bufio.Reader, copy the n bytes, and pass them to handleKey.
// This exercises the wiring the real loop uses, not handleKey in isolation.
func dispatch(ed *editor, r io.Reader) {
	in := bufio.NewReader(r)
	buf := make([]byte, 8)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			b := make([]byte, n)
			copy(b, buf[:n])
			quit := false
			ed.handleKey(b, &quit)
			if quit {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// A raw space byte, read through the same bufio path the terminal loop uses,
// must set the pixel under the cursor.
func TestReadPathSpaceDraws(t *testing.T) {
	ed := &editor{c: ui.New(8, 8)}
	ed.cx, ed.cy = 4, 5
	pr, pw := io.Pipe()
	go func() { pw.Write([]byte{' '}); pw.Close() }()
	dispatch(ed, pr)
	if !ed.c.Sprite.At(4, 5) {
		t.Fatal("space read through the terminal pipeline did not draw")
	}
}

// A realistic burst: move right (arrow), draw, move down (arrow), erase — all
// arriving as separate reads, exercising arrow multibyte + single-byte paint in
// sequence.
func TestReadPathMixedSequence(t *testing.T) {
	ed := &editor{c: ui.New(8, 8)}
	ed.cx, ed.cy = 0, 0
	pr, pw := io.Pipe()
	go func() {
		pw.Write([]byte{0x1b, '[', 'C'}) // right -> cx=1
		pw.Write([]byte{' '})            // draw at (1,0)
		pw.Write([]byte{0x1b, '[', 'B'}) // down -> cy=1
		pw.Write([]byte{'\r'})           // toggle at (1,1) on
		pw.Write([]byte{0x7f})           // erase at (1,1) -> off
		pw.Close()
	}()
	dispatch(ed, pr)
	if !ed.c.Sprite.At(1, 0) {
		t.Error("draw after arrow-right did not land at (1,0)")
	}
	if ed.c.Sprite.At(1, 1) {
		t.Error("toggle-on then erase should leave (1,1) clear")
	}
}

// If a space is ever coalesced with a following byte in one read, painting must
// still happen (guards the b[0]-only dispatch for the common paint key).
func TestReadPathCoalescedSpace(t *testing.T) {
	ed := &editor{c: ui.New(8, 8)}
	ed.cx, ed.cy = 2, 2
	pr, pw := io.Pipe()
	// Space immediately followed by 'l' (move right) in a single write; the
	// bufio reader may deliver them together.
	go func() { pw.Write([]byte{' ', 'l'}); pw.Close() }()
	dispatch(ed, pr)
	if !ed.c.Sprite.At(2, 2) {
		t.Error("coalesced space did not draw the pixel")
	}
}
