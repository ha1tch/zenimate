//go:build purego

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
)

func TestHandleDropLoadsZCUT(t *testing.T) {
	dir := t.TempDir()
	// Make a 24x24 sprite, save as zcut, then "drop" it.
	src := model.New(24, 24)
	src.Set(3, 3, true)
	data, _ := src.MarshalZCUT()
	p := filepath.Join(dir, "knight.zcut")
	os.WriteFile(p, data, 0o644)

	c := ui.New(16, 16) // starts 16x16
	var f fileOps
	f.handleDrop(c, []string{p})
	if c.Sprite.Width() != 24 || c.Sprite.Height() != 24 {
		t.Fatalf("drop did not load zcut: sprite is %dx%d", c.Sprite.Width(), c.Sprite.Height())
	}
	if !c.Sprite.Frame(0)[3*24+3] {
		t.Error("dropped sprite lost pixel (3,3)")
	}
}

func TestHandleDropLoadsSCR(t *testing.T) {
	dir := t.TempDir()
	src := model.New(16, 16)
	src.Set(1, 1, true)
	scrImg, _ := src.ExportScreen(0, model.FormatSCR, "x")
	p := filepath.Join(dir, "pic.scr")
	os.WriteFile(p, scrImg, 0o644)

	c := ui.New(16, 16)
	var f fileOps
	f.handleDrop(c, []string{p})
	if c.Sprite.Width() != 256 || c.Sprite.Height() != 192 {
		t.Fatalf("drop did not load scr as full screen: %dx%d", c.Sprite.Width(), c.Sprite.Height())
	}
}

func TestHandleDropSkipsUnsupported(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.txt")
	os.WriteFile(p, []byte("hello"), 0o644)
	c := ui.New(16, 16)
	var f fileOps
	f.handleDrop(c, []string{p})
	// Sprite unchanged (still 16x16) and a status set.
	if c.Sprite.Width() != 16 {
		t.Error("unsupported drop should not change the sprite")
	}
}

func TestHandleDropFirstUsableWins(t *testing.T) {
	dir := t.TempDir()
	txt := filepath.Join(dir, "a.txt")
	os.WriteFile(txt, []byte("x"), 0o644)
	src := model.New(32, 16)
	data, _ := src.MarshalZCUT()
	zc := filepath.Join(dir, "b.zcut")
	os.WriteFile(zc, data, 0o644)

	c := ui.New(16, 16)
	var f fileOps
	// txt first (skipped), zcut second (loads).
	f.handleDrop(c, []string{txt, zc})
	if c.Sprite.Width() != 32 {
		t.Errorf("expected the zcut to load, sprite is %dx%d", c.Sprite.Width(), c.Sprite.Height())
	}
}
