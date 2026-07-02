package ui

import "testing"

func TestSourceLabel(t *testing.T) {
	c := New(16, 16)
	// New/unsaved.
	if got := c.SourceLabel(); got == "" || got[len(got)-9:] != "(unsaved)" {
		t.Errorf("unsaved label wrong: %q", got)
	}
	// File source.
	c.SetSource(SpriteSource{Kind: SourceFile, Path: "/art/knight.zani"})
	if c.SourceLabel() != "knight.zani" {
		t.Errorf("file label = %q", c.SourceLabel())
	}
	// Bundle source.
	c.SetSource(SpriteSource{Kind: SourceBundle, Path: "/art/game.zbun", Entry: "goblin"})
	if c.SourceLabel() != "goblin - game.zbun" {
		t.Errorf("bundle label = %q", c.SourceLabel())
	}
}

func TestLoadSpriteResetsSource(t *testing.T) {
	c := New(16, 16)
	c.SetSource(SpriteSource{Kind: SourceFile, Path: "/x.zani"})
	c.LoadSprite(c.Sprite) // reloading clears provenance until caller re-sets it
	if c.Source().Kind != SourceNone {
		t.Error("LoadSprite should reset source to None")
	}
}
