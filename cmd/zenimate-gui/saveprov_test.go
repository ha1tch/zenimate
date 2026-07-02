//go:build purego

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/filepick"
)

// Saving a file-sourced sprite overwrites that file silently (no dialog).
func TestSaveFileSourceOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "knight.zani")
	os.WriteFile(path, []byte("placeholder"), 0o644)

	c := ui.New(24, 16)
	c.Sprite.Set(3, 3, true)
	c.SetSource(ui.SpriteSource{Kind: ui.SourceFile, Path: path})

	var f fileOps
	f.save(c)
	if f.active() {
		t.Error("saving a file source should not open any dialog")
	}
	// The file should now be a real zcut with our pixel.
	data, _ := os.ReadFile(path)
	s, err := model.LoadZCUT(data)
	if err != nil {
		t.Fatalf("saved file is not a valid animation: %v", err)
	}
	if !s.Frame(0)[3*24+3] {
		t.Error("saved file lost pixel (3,3)")
	}
}

// Saving a new (sourceless) sprite falls back to Save As (opens a dialog).
func TestSaveNoSourceOpensDialog(t *testing.T) {
	c := ui.New(16, 16)
	var f fileOps
	f.save(c)
	if f.dlg == nil {
		t.Error("saving with no source should open the Save As dialog")
	}
}

// Saving a bundle-sourced sprite opens the provenance chooser.
func TestSaveBundleSourceOpensChooser(t *testing.T) {
	c := ui.New(16, 16)
	c.SetSource(ui.SpriteSource{Kind: ui.SourceBundle, Path: "/x/game.zbun", Entry: "knight"})
	var f fileOps
	f.save(c)
	if f.saveProv == nil {
		t.Fatal("bundle source should open the save-provenance chooser")
	}
}

// The full bundle-update round trip: open from a bundle, edit, save into bundle.
func TestSaveIntoSourceBundleRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "game.zbun")
	// Seed a bundle with two animations.
	b := model.NewBundle()
	k := model.New(24, 16)
	b.AddSprite("knight", "hero", k)
	g := model.New(16, 16)
	b.AddSprite("goblin", "enemy", g)
	data, _ := b.Encode()
	os.WriteFile(path, data, 0o644)

	// Open knight from the bundle.
	c := ui.New(16, 16)
	var f fileOps
	f.openBundleEntry(c, path, "knight")
	if c.Source().Kind != ui.SourceBundle || c.Source().Label != "hero" {
		t.Fatalf("bundle provenance not set: %+v", c.Source())
	}
	// Edit and save straight into the bundle.
	c.Sprite.Set(5, 5, true)
	f.saveIntoSourceBundle(c, c.Source())

	// Reopen the bundle: knight must have the edit; goblin untouched; label kept.
	data2, _ := os.ReadFile(path)
	b2, err := model.OpenBundle(data2)
	if err != nil {
		t.Fatal(err)
	}
	if b2.Len() != 2 {
		t.Fatalf("bundle now has %d entries, want 2", b2.Len())
	}
	kn, _ := b2.Sprite("knight")
	if !kn.Frame(0)[5*24+5] {
		t.Error("edit did not persist into the bundle")
	}
	var knightLabel string
	for _, e := range b2.Entries() {
		if e.Name == "knight" {
			knightLabel = e.Label
		}
	}
	if knightLabel != "hero" {
		t.Errorf("label not preserved on bundle update: %q", knightLabel)
	}
}

// Save As always writes a standalone file and sets file provenance.
func TestSaveAsSetsFileSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hero.zani")
	c := ui.New(20, 16)
	var f fileOps
	f.writeAnimationFile(c, path)
	if c.Source().Kind != ui.SourceFile || c.Source().Path != path {
		t.Errorf("writeAnimationFile did not set file source: %+v", c.Source())
	}
	if !fileExists(path) {
		t.Error("file not written")
	}
}

// The provenance chooser offers update-in-bundle and separate options.
func TestSaveProvChooserOptions(t *testing.T) {
	c := ui.New(16, 16)
	src := ui.SpriteSource{Kind: ui.SourceBundle, Path: "/x/game.zbun", Entry: "knight"}
	sp := newSaveProvChooser(c, src)
	sp.layout(stubRenderer{}, 900, 700)
	if len(sp.options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(sp.options))
	}
	// First option updates the bundle.
	rc := sp.rects[0]
	if r := sp.update(filepick.Input{MouseX: rc.X + 4, MouseY: rc.Y + 4, MousePressed: true}); r.state != chooserPicked || !r.toBundle {
		t.Errorf("first option should pick update-in-bundle, got %+v", r)
	}
}

func TestDropBundleOpensBrowser(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "game.zbun")
	b := model.NewBundle()
	b.AddSprite("knight", "hero", model.New(24, 16))
	data, _ := b.Encode()
	os.WriteFile(path, data, 0o644)

	c := ui.New(16, 16)
	var f fileOps
	f.handleDrop(c, []string{path})
	if f.dlg == nil {
		t.Fatal("dropping a .zbun should open the browse dialog")
	}
	if f.dlg.Status() != filepick.Active {
		t.Error("dialog should be active")
	}
}
