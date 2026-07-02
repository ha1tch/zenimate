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

func TestBundleChooserNewFileOffersCreateOnly(t *testing.T) {
	bc := newBundleChooser(ui.New(16, 16), "/tmp/x.zbun", "knight", "", false)
	if len(bc.options) != 1 || bc.options[0].mode != bundleCreate {
		t.Errorf("new-file chooser should offer only Create, got %+v", bc.options)
	}
}

func TestBundleChooserExistingOffersAddAndReplace(t *testing.T) {
	bc := newBundleChooser(ui.New(16, 16), "/tmp/x.zbun", "knight", "", true)
	if len(bc.options) != 2 {
		t.Fatalf("existing-file chooser should offer 2 options, got %d", len(bc.options))
	}
	if bc.options[0].mode != bundleAdd || bc.options[1].mode != bundleCreate {
		t.Errorf("expected Add then Replace, got %+v", bc.options)
	}
}

func TestBundleChooserPickAndCancel(t *testing.T) {
	bc := newBundleChooser(ui.New(16, 16), "/tmp/x.zbun", "knight", "", true)
	bc.layout(stubRenderer{}, 900, 700)
	rc := bc.rects[0] // Add
	if r := bc.update(filepick.Input{MouseX: rc.X + 4, MouseY: rc.Y + 4, MousePressed: true}); r.state != chooserPicked || r.mode != bundleAdd {
		t.Errorf("expected pick Add, got %+v", r)
	}
	if bc.update(filepick.Input{Keys: []filepick.Key{filepick.KeyEscape}}).state != chooserCancelled {
		t.Error("escape should cancel")
	}
}

// End-to-end: create a bundle via writeBundle, then add a second sprite to it.
func TestWriteBundleCreateThenAdd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "game.zbun")

	c := ui.New(24, 16)
	c.Sprite.Set(2, 2, true)
	c.Sprite.SetName("knight")
	var f fileOps
	f.writeBundle(c, path, "knight", "hero", bundleCreate)
	if !fileExists(path) {
		t.Fatal("bundle file was not created")
	}
	// Load and check one animation is present.
	data, _ := os.ReadFile(path)
	b, err := model.OpenBundle(data)
	if err != nil {
		t.Fatal(err)
	}
	if b.Len() != 1 || b.Entries()[0].Name != "knight" || b.Entries()[0].Label != "hero" {
		t.Fatalf("after create: %+v", b.Entries())
	}

	// Now add a second sprite to the existing bundle.
	c2 := ui.New(16, 16)
	c2.Sprite.Set(1, 1, true)
	f.writeBundle(c2, path, "goblin", "enemy", bundleAdd)
	data2, _ := os.ReadFile(path)
	b2, err := model.OpenBundle(data2)
	if err != nil {
		t.Fatal(err)
	}
	if b2.Len() != 2 {
		t.Fatalf("after add: Len = %d, want 2", b2.Len())
	}
	names := map[string]bool{}
	for _, e := range b2.Entries() {
		names[e.Name] = true
	}
	if !names["knight"] || !names["goblin"] {
		t.Errorf("bundle missing expected animations: %+v", b2.Entries())
	}
}
