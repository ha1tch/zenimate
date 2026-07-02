package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ha1tch/zenimate/internal/model"
)

func TestLoadTargetFileAndBundle(t *testing.T) {
	dir := t.TempDir()
	// Standalone .zani.
	s := model.New(24, 16)
	s.Set(2, 2, true)
	zc, _ := s.MarshalZCUT()
	zpath := filepath.Join(dir, "hero.zani")
	os.WriteFile(zpath, zc, 0o644)

	got, err := loadTarget(zpath)
	if err != nil || got.Width() != 24 {
		t.Fatalf("file load failed: %v", err)
	}

	// Bundle ref.
	b := model.NewBundle()
	b.AddSprite("knight", "hero", s)
	bdata, _ := b.Encode()
	bpath := filepath.Join(dir, "game.zbun")
	os.WriteFile(bpath, bdata, 0o644)

	got2, err := loadTarget(bpath + "#knight")
	if err != nil {
		t.Fatalf("bundle ref load failed: %v", err)
	}
	if !got2.Frame(0)[2*24+2] {
		t.Error("bundle-ref animation lost its pixel")
	}

	if _, err := loadTarget(bpath + "#nope"); err == nil {
		t.Error("missing entry should error")
	}
}
