package model

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// A .zbun bundle is a zip archive collecting several whole animations. Each
// entry is a complete .zani (ZCUT-encoded) animation — all of one sprite's
// frames — stored under "<name>.zani", accompanied by a single manifest.json
// describing the collection. This is one level up from a .zani: a .zani is one
// animated sprite; a .zbun gathers many, e.g. all the sprites of a game.
//
// The manifest is a convenience index (name, frame count, dimensions, and a
// free-text label per entry); the entries themselves remain plain .zani files,
// so a bundle can be unzipped and its animations used individually.

// bundleManifestVersion is the manifest schema version.
const bundleManifestVersion = 1

// BundleEntry describes one animation held in a bundle.
type BundleEntry struct {
	Name   string `json:"name"`   // entry base name (without extension)
	File   string `json:"file"`   // zip entry filename, e.g. "knight.zani"
	Frames int    `json:"frames"` // number of animation frames
	Width  int    `json:"width"`  // sprite width in pixels
	Height int    `json:"height"` // sprite height in pixels
	Label  string `json:"label"`  // free-text label/tag (may be empty)
}

// bundleManifest is the JSON index stored as manifest.json in the zip.
type bundleManifest struct {
	Format  string        `json:"format"`  // always "zbun"
	Version int           `json:"version"` // manifest schema version
	Entries []BundleEntry `json:"entries"` // in stored order
}

// Bundle is an in-memory collection of animations, read from or written to a
// .zbun. Entry order is preserved; names are unique within a bundle.
type Bundle struct {
	entries []BundleEntry
	// data maps a zip entry filename to its raw .zani (ZCUT) bytes.
	data map[string][]byte
}

// NewBundle returns an empty bundle ready to receive animations.
func NewBundle() *Bundle {
	return &Bundle{data: map[string][]byte{}}
}

// Entries returns the bundle's animation index, in order.
func (b *Bundle) Entries() []BundleEntry {
	out := make([]BundleEntry, len(b.entries))
	copy(out, b.entries)
	return out
}

// Len reports how many animations the bundle holds.
func (b *Bundle) Len() int { return len(b.entries) }

// Has reports whether an animation with the given name is already present.
func (b *Bundle) Has(name string) bool {
	for _, e := range b.entries {
		if e.Name == name {
			return true
		}
	}
	return false
}

// AddSprite adds (or replaces) an animation in the bundle. The sprite is encoded
// as a .zani (ZCUT); the manifest entry records its shape and the given label.
// name must be non-empty; if an entry of that name exists it is overwritten,
// keeping its position.
func (b *Bundle) AddSprite(name, label string, s *Sprite) error {
	name = sanitiseName(name)
	if name == "" {
		return fmt.Errorf("bundle: entry name is empty")
	}
	data, err := s.MarshalZCUT()
	if err != nil {
		return fmt.Errorf("bundle: encode %q: %w", name, err)
	}
	file := name + "." + ExtAnimation
	entry := BundleEntry{
		Name:   name,
		File:   file,
		Frames: s.FrameCount(),
		Width:  s.Width(),
		Height: s.Height(),
		Label:  label,
	}
	if b.data == nil {
		b.data = map[string][]byte{}
	}
	for i := range b.entries {
		if b.entries[i].Name == name {
			b.entries[i] = entry // replace in place
			b.data[file] = data
			return nil
		}
	}
	b.entries = append(b.entries, entry)
	b.data[file] = data
	return nil
}

// Sprite decodes and returns the named animation as a sprite.
func (b *Bundle) Sprite(name string) (*Sprite, error) {
	for _, e := range b.entries {
		if e.Name == name {
			raw, ok := b.data[e.File]
			if !ok {
				return nil, fmt.Errorf("bundle: %q has no data", name)
			}
			return LoadZCUT(raw)
		}
	}
	return nil, fmt.Errorf("bundle: no animation named %q", name)
}

// EntryData returns the raw .zani bytes for a named entry (for previews or
// extraction) without decoding to a sprite.
func (b *Bundle) EntryData(name string) ([]byte, bool) {
	for _, e := range b.entries {
		if e.Name == name {
			raw, ok := b.data[e.File]
			return raw, ok
		}
	}
	return nil, false
}

// Encode writes the bundle to .zbun (zip) bytes: every animation as its .zani
// entry plus manifest.json.
func (b *Bundle) Encode() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, e := range b.entries {
		raw := b.data[e.File]
		w, err := zw.Create(e.File)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(raw); err != nil {
			return nil, err
		}
	}

	man := bundleManifest{Format: "zbun", Version: bundleManifestVersion, Entries: b.entries}
	mj, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return nil, err
	}
	mw, err := zw.Create("manifest.json")
	if err != nil {
		return nil, err
	}
	if _, err := mw.Write(mj); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// sanitiseName reduces a proposed entry name to a safe base name: it strips any
// directory part and a trailing recognised extension, and rejects path
// separators and quotes by replacing them with underscores.
func sanitiseName(name string) string {
	// Drop any directory portion.
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	// Drop a trailing animation/bundle extension if present.
	if dot := strings.LastIndexByte(name, '.'); dot > 0 {
		ext := normaliseExt(name[dot+1:])
		if IsAnimationExt(ext) || IsBundleExt(ext) {
			name = name[:dot]
		}
	}
	// Replace characters that are unsafe in zip entry names.
	var sb strings.Builder
	for _, r := range name {
		switch r {
		case '"', '\'', '/', '\\':
			sb.WriteByte('_')
		default:
			sb.WriteRune(r)
		}
	}
	return strings.TrimSpace(sb.String())
}
