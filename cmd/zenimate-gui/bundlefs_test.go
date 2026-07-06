//go:build purego

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

func writeTestBundle(t *testing.T) string {
	t.Helper()
	b := model.NewBundle()
	s1 := model.New(24, 16)
	s1.Set(2, 2, true)
	b.AddSprite("knight", "hero", s1)
	s2 := model.New(16, 16)
	s2.Set(1, 1, true)
	b.AddSprite("goblin", "", s2)
	data, err := b.Encode()
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "game.zbun")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestBundleContainerIsContainer(t *testing.T) {
	var bc bundleContainers
	if !bc.IsContainer("/x/game.zbun") || !bc.IsContainer("/x/GAME.ZBU") {
		t.Error("should recognise .zbun/.zbu")
	}
	if bc.IsContainer("/x/hero.zani") {
		t.Error("a .zani is not a container")
	}
}

func TestBundleContainerReadEntries(t *testing.T) {
	p := writeTestBundle(t)
	var bc bundleContainers
	entries, err := bc.ReadContainer(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	// knight should carry frames/size/label metadata.
	var knight *zenui.Entry
	for i := range entries {
		if entries[i].Name == "knight" {
			knight = &entries[i]
		}
	}
	if knight == nil {
		t.Fatal("knight entry missing")
	}
	hasLabel := false
	for _, m := range knight.Meta {
		if m.Key == "label" && m.Value == "hero" {
			hasLabel = true
		}
	}
	if !hasLabel {
		t.Errorf("knight missing label metadata: %+v", knight.Meta)
	}
}

func TestBundlePreviewThumbnail(t *testing.T) {
	p := writeTestBundle(t)
	pv := bundlePreview(p, zenui.Entry{Name: "knight"})
	if pv == nil {
		t.Fatal("expected a thumbnail")
	}
	if pv.W != 24 || pv.H != 16 || len(pv.Pixels) != 24*16 {
		t.Errorf("thumbnail dims wrong: %dx%d, %d px", pv.W, pv.H, len(pv.Pixels))
	}
	// Preview for a plain (non-container) entry is nil.
	if bundlePreview("", zenui.Entry{Name: "x"}) != nil {
		t.Error("no container => no preview")
	}
}

func TestSplitBundleRef(t *testing.T) {
	bp, e, ok := splitBundleRef("/home/u/game.zbun#knight")
	if !ok || bp != "/home/u/game.zbun" || e != "knight" {
		t.Errorf("split wrong: %q %q %v", bp, e, ok)
	}
	if _, _, ok := splitBundleRef("/home/u/plain.zani"); ok {
		t.Error("plain path should not split")
	}
}

func TestOpenBundleEntryLoads(t *testing.T) {
	p := writeTestBundle(t)
	c := ui.New(16, 16)
	var f fileOps
	f.openBundleEntry(c, p, "knight")
	if c.Sprite.Width() != 24 || c.Sprite.Height() != 16 {
		t.Fatalf("opened wrong animation: %dx%d", c.Sprite.Width(), c.Sprite.Height())
	}
	if !c.Sprite.Frame(0).At(2, 2, 24) {
		t.Error("knight pixel (2,2) missing after bundle open")
	}
}
