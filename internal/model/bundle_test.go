package model

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

// distinctSprite makes a sprite whose frame 0 has a pixel unique to `mark`.
func distinctSprite(w, h, frames, mark int) *Sprite {
	s := New(w, h)
	for s.FrameCount() < frames {
		s.AddFrame()
	}
	for s.FrameCount() > frames {
		s.RemoveFrame()
	}
	s.Select(0)
	s.Set(mark%w, 0, true)
	return s
}

func TestBundleAddEncodeReopen(t *testing.T) {
	b := NewBundle()
	if err := b.AddSprite("knight", "player hero", distinctSprite(24, 16, 4, 1)); err != nil {
		t.Fatal(err)
	}
	if err := b.AddSprite("goblin", "enemy", distinctSprite(16, 16, 2, 2)); err != nil {
		t.Fatal(err)
	}
	if b.Len() != 2 {
		t.Fatalf("Len = %d, want 2", b.Len())
	}
	data, err := b.Encode()
	if err != nil {
		t.Fatal(err)
	}

	b2, err := OpenBundle(data)
	if err != nil {
		t.Fatal(err)
	}
	if b2.Len() != 2 {
		t.Fatalf("reopened Len = %d, want 2", b2.Len())
	}
	entries := b2.Entries()
	// Order preserved (manifest order): knight then goblin.
	if entries[0].Name != "knight" || entries[1].Name != "goblin" {
		t.Errorf("entry order/names wrong: %+v", entries)
	}
	if entries[0].Frames != 4 || entries[0].Width != 24 || entries[0].Height != 16 {
		t.Errorf("knight shape wrong: %+v", entries[0])
	}
	if entries[0].Label != "player hero" {
		t.Errorf("label lost: %q", entries[0].Label)
	}
	// Decode an animation back and check its distinct pixel.
	kn, err := b2.Sprite("knight")
	if err != nil {
		t.Fatal(err)
	}
	if kn.FrameCount() != 4 || !kn.Frame(0).At(1, 0, 24) {
		t.Error("knight animation did not round-trip through the bundle")
	}
}

func TestBundleAddReplaceInPlace(t *testing.T) {
	b := NewBundle()
	b.AddSprite("a", "first", distinctSprite(16, 16, 2, 1))
	b.AddSprite("b", "second", distinctSprite(16, 16, 3, 2))
	// Replace "a" with a different sprite; position and count must be preserved.
	b.AddSprite("a", "updated", distinctSprite(24, 24, 5, 3))
	if b.Len() != 2 {
		t.Fatalf("replace changed Len to %d, want 2", b.Len())
	}
	e := b.Entries()
	if e[0].Name != "a" || e[0].Frames != 5 || e[0].Width != 24 || e[0].Label != "updated" {
		t.Errorf("replace did not update entry in place: %+v", e[0])
	}
}

func TestBundleZipStructure(t *testing.T) {
	b := NewBundle()
	b.AddSprite("knight", "hero", distinctSprite(16, 16, 2, 1))
	data, _ := b.Encode()

	zr, _ := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	names := map[string]bool{}
	for _, f := range zr.File {
		names[f.Name] = true
	}
	if !names["manifest.json"] {
		t.Error("bundle missing manifest.json")
	}
	if !names["knight.zani"] {
		t.Error("bundle missing knight.zani entry")
	}
	// Manifest is valid JSON with the right format tag.
	for _, f := range zr.File {
		if f.Name == "manifest.json" {
			rc, _ := f.Open()
			mb, _ := io.ReadAll(rc)
			rc.Close()
			var m bundleManifest
			if err := json.Unmarshal(mb, &m); err != nil {
				t.Fatalf("manifest invalid: %v", err)
			}
			if m.Format != "zbun" || m.Version != bundleManifestVersion {
				t.Errorf("manifest header wrong: %+v", m)
			}
		}
	}
}

func TestOpenBundleWithoutManifest(t *testing.T) {
	// A zip of .zani entries but no manifest should still open (names from files).
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	s := distinctSprite(16, 16, 3, 4)
	zc, _ := s.MarshalZCUT()
	w, _ := zw.Create("orc.zani")
	w.Write(zc)
	zw.Close()

	b, err := OpenBundle(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if b.Len() != 1 || b.Entries()[0].Name != "orc" {
		t.Errorf("manifest-less bundle wrong: %+v", b.Entries())
	}
	if b.Entries()[0].Frames != 3 {
		t.Errorf("frames from file wrong: %d", b.Entries()[0].Frames)
	}
}

func TestOpenBundleRejectsNonBundle(t *testing.T) {
	if _, err := OpenBundle([]byte("not a zip")); err == nil {
		t.Error("expected error on non-zip data")
	}
	// A zip with no animation entries.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("readme.txt")
	w.Write([]byte("hello"))
	zw.Close()
	if _, err := OpenBundle(buf.Bytes()); err == nil {
		t.Error("expected error on a zip with no animations")
	}
}

func TestBundleNameSanitisation(t *testing.T) {
	b := NewBundle()
	// A name with a path and extension should reduce to a clean base name.
	if err := b.AddSprite("sprites/knight.zani", "x", distinctSprite(16, 16, 1, 1)); err != nil {
		t.Fatal(err)
	}
	if b.Entries()[0].Name != "knight" {
		t.Errorf("name not sanitised: %q", b.Entries()[0].Name)
	}
	if b.Entries()[0].File != "knight.zani" {
		t.Errorf("file name wrong: %q", b.Entries()[0].File)
	}
}
