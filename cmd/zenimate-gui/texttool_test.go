package main

import (
	"testing"

	"github.com/ha1tch/zenimate/internal/fonts"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/bdf"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

func TestTextEntryInsertAndBackspace(t *testing.T) {
	var e textEntry
	e.active = true
	e.desiredCol = -1
	for _, r := range "HI" {
		e.insertRune(r)
	}
	if e.text != "HI" || e.cursor != 2 {
		t.Fatalf("after inserting 'HI': text=%q cursor=%d, want \"HI\", 2", e.text, e.cursor)
	}
	e.backspace()
	if e.text != "H" || e.cursor != 1 {
		t.Errorf("after backspace: text=%q cursor=%d, want \"H\", 1", e.text, e.cursor)
	}
}

func TestTextEntryBackspaceAtStartIsNoOp(t *testing.T) {
	var e textEntry
	e.active = true
	e.text = "AB"
	e.cursor = 0
	e.backspace()
	if e.text != "AB" || e.cursor != 0 {
		t.Errorf("backspace at cursor=0 should be a no-op, got text=%q cursor=%d", e.text, e.cursor)
	}
}

func TestTextEntryEnterInsertsNewlineNotCommit(t *testing.T) {
	e := textEntry{active: true, desiredCol: -1}
	cancel := e.Update(zenui.Input{Keys: []zenui.Key{zenui.KeyEnter}}, mustSinclair(t))
	if cancel {
		t.Error("Enter should not signal cancel")
	}
	if e.text != "\n" {
		t.Errorf("Enter should insert a newline, got text=%q", e.text)
	}
	if !e.active {
		t.Error("Enter should not deactivate the entry (it no longer commits)")
	}
}

func TestTextEntryEscapeSignalsCancel(t *testing.T) {
	e := textEntry{active: true, text: "x", desiredCol: -1}
	cancel := e.Update(zenui.Input{Keys: []zenui.Key{zenui.KeyEscape}}, mustSinclair(t))
	if !cancel {
		t.Error("Escape should signal cancel")
	}
}

func TestTextEntryLeftRightMovesCursor(t *testing.T) {
	e := textEntry{active: true, text: "AB", cursor: 2, desiredCol: -1}
	e.Update(zenui.Input{Keys: []zenui.Key{zenui.KeyLeft}}, mustSinclair(t))
	if e.cursor != 1 {
		t.Errorf("cursor after Left = %d, want 1", e.cursor)
	}
	e.Update(zenui.Input{Keys: []zenui.Key{zenui.KeyRight}}, mustSinclair(t))
	if e.cursor != 2 {
		t.Errorf("cursor after Right = %d, want 2", e.cursor)
	}
	// Clamped at both ends.
	e.cursor = 0
	e.Update(zenui.Input{Keys: []zenui.Key{zenui.KeyLeft}}, mustSinclair(t))
	if e.cursor != 0 {
		t.Errorf("Left at cursor=0 should clamp, got %d", e.cursor)
	}
}

// TestTextEntryUpDownFindsClosestColumn hand-verifies the "sticky column"
// navigation Horatio specifically asked for: moving up from the end of a
// short line to a longer one should land at the same pixel column, not at
// a fixed character index or the target line's own end.
func TestTextEntryUpDownFindsClosestColumn(t *testing.T) {
	font := mustSinclair(t)
	// "AB\nX": line 0 = "AB" (2 chars), line 1 = "X" (1 char). Cursor at the
	// very end (index 4, after 'X') is 1 character into line 1.
	e := textEntry{active: true, text: "AB\nX", cursor: 4, desiredCol: -1}
	e.moveVertical(font, -1)
	// 1 character into "AB" is index 1 (right after 'A').
	if e.cursor != 1 {
		t.Errorf("cursor after moving up = %d, want 1 (same pixel column, 1 char in)", e.cursor)
	}

	// Moving back down should return to the ORIGINAL column (end of "X"),
	// not reset to the start of the line — this is what desiredCol is for.
	e.moveVertical(font, 1)
	if e.cursor != 4 {
		t.Errorf("cursor after moving back down = %d, want 4 (sticky column should restore it)", e.cursor)
	}
}

func TestTextEntryUpAtFirstLineIsNoOp(t *testing.T) {
	font := mustSinclair(t)
	e := textEntry{active: true, text: "AB", cursor: 1, desiredCol: -1}
	e.moveVertical(font, -1)
	if e.cursor != 1 {
		t.Errorf("Up on the first line should be a no-op, cursor changed to %d", e.cursor)
	}
}

func mustSinclair(t *testing.T) *bdf.Font {
	t.Helper()
	f, err := fonts.Sinclair()
	if err != nil {
		t.Fatalf("fonts.Sinclair() error: %v", err)
	}
	return f
}

func TestCommitTextEmptyStringIsNoOp(t *testing.T) {
	c := ui.New(32, 24)
	font, err := fonts.Sinclair()
	if err != nil {
		t.Fatalf("fonts.Sinclair() error: %v", err)
	}
	before := c.CanUndo()
	commitText(c, font, &textEntry{x: 0, y: 0, text: ""})
	if c.CanUndo() != before {
		t.Error("commitText with an empty string should not push a Checkpoint")
	}
}

func TestCommitTextSetsSomePixels(t *testing.T) {
	c := ui.New(32, 24)
	font, err := fonts.Sinclair()
	if err != nil {
		t.Fatalf("fonts.Sinclair() error: %v", err)
	}
	commitText(c, font, &textEntry{x: 2, y: 2, text: "A"})

	found := false
	for y := 0; y < 24 && !found; y++ {
		for x := 0; x < 32; x++ {
			if c.Sprite.At(x, y) {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("commitText(\"A\") should have set at least one pixel")
	}
	if !c.CanUndo() {
		t.Error("commitText with real content should push a Checkpoint")
	}
}

func TestCommitTextClipsAtSpriteBounds(t *testing.T) {
	c := ui.New(16, 16)
	font, err := fonts.Sinclair()
	if err != nil {
		t.Fatalf("fonts.Sinclair() error: %v", err)
	}
	// Position and string deliberately chosen to run well past the sprite's
	// right and bottom edges — must not panic, and must not silently grow
	// the sprite or write anywhere out of bounds.
	commitText(c, font, &textEntry{x: 10, y: 10, text: "HELLO WORLD"})
	if c.Sprite.Width() != 16 || c.Sprite.Height() != 16 {
		t.Errorf("sprite dimensions changed: got %dx%d, want 16x16", c.Sprite.Width(), c.Sprite.Height())
	}
}
