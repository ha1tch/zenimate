//go:build purego

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ha1tch/zenimate/internal/model"
)

// TestZCUTFileRoundTrip exercises the on-disk path used by Save/Open: marshal,
// write, read, load. (The dialog itself is covered in the filepick package.)
func TestZCUTFileRoundTrip(t *testing.T) {
	s := model.New(24, 16)
	s.Set(1, 1, true)
	s.SetAttrCell(0, 0, 0x45) // ink 5, bright
	data, err := s.MarshalZCUT()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "sprite.zcut")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	back, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := model.LoadZCUT(back)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Frame(0)[1*24+1] {
		t.Error("pixel (1,1) lost through file round-trip")
	}
	if loaded.AttrCellFrame(0, 0, 0) != 0x45 {
		t.Errorf("attr lost: got 0x%02X want 0x45", loaded.AttrCellFrame(0, 0, 0))
	}
}

func TestBaseName(t *testing.T) {
	cases := map[string]string{
		"/home/u/art/hero.zcut": "hero.zcut",
		"hero.zcut":             "hero.zcut",
		"/x":                    "x",
	}
	for in, want := range cases {
		if got := baseName(in); got != want {
			t.Errorf("baseName(%q) = %q, want %q", in, got, want)
		}
	}
}
