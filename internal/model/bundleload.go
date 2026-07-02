package model

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// OpenBundle reads a .zbun (zip) into a Bundle. Every ".zani"/".zan"/".zcut"
// entry is loaded as an animation; a manifest.json, if present, supplies the
// per-entry labels and ordering. When the manifest is missing or partial, the
// bundle is still usable — entries are indexed from the zip directly and shapes
// are read from each animation, with empty labels.
func OpenBundle(data []byte) (*Bundle, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("bundle: %w", err)
	}

	b := NewBundle()

	// Collect the raw animation entries.
	raw := map[string][]byte{}
	for _, fz := range zr.File {
		if fz.Name == "manifest.json" {
			continue
		}
		if !isAnimationFile(fz.Name) {
			continue
		}
		rc, err := fz.Open()
		if err != nil {
			return nil, fmt.Errorf("bundle: %s: %w", fz.Name, err)
		}
		bs, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("bundle: %s: %w", fz.Name, err)
		}
		raw[fz.Name] = bs
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("bundle: no animation entries found")
	}

	// Prefer the manifest for order and labels.
	var man bundleManifest
	haveMan := false
	if mf, err := openManifest(zr); err == nil && mf != nil {
		if json.Unmarshal(mf, &man) == nil && man.Format == "zbun" {
			haveMan = true
		}
	}

	used := map[string]bool{}
	add := func(file, name, label string) error {
		bs, ok := raw[file]
		if !ok {
			return nil // manifest names a missing file; skip gracefully
		}
		s, err := LoadZCUT(bs)
		if err != nil {
			return fmt.Errorf("bundle: %s: %w", file, err)
		}
		if name == "" {
			name = baseNameNoExt(file)
		}
		e := BundleEntry{
			Name:   name,
			File:   file,
			Frames: s.FrameCount(),
			Width:  s.Width(),
			Height: s.Height(),
			Label:  label,
		}
		b.entries = append(b.entries, e)
		b.data[file] = bs
		used[file] = true
		return nil
	}

	if haveMan {
		for _, e := range man.Entries {
			if err := add(e.File, e.Name, e.Label); err != nil {
				return nil, err
			}
		}
	}
	// Add any animation entries not covered by the manifest, in name order.
	var leftover []string
	for file := range raw {
		if !used[file] {
			leftover = append(leftover, file)
		}
	}
	sort.Strings(leftover)
	for _, file := range leftover {
		if err := add(file, "", ""); err != nil {
			return nil, err
		}
	}

	if b.Len() == 0 {
		return nil, fmt.Errorf("bundle: no readable animations")
	}
	return b, nil
}

// openManifest returns the manifest.json bytes from the zip, or (nil,nil) if
// absent.
func openManifest(zr *zip.Reader) ([]byte, error) {
	for _, fz := range zr.File {
		if fz.Name == "manifest.json" {
			rc, err := fz.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, nil
}

// isAnimationFile reports whether a zip entry name is an animation by extension.
func isAnimationFile(name string) bool {
	dot := strings.LastIndexByte(name, '.')
	if dot < 0 {
		return false
	}
	return IsAnimationExt(name[dot+1:])
}

// baseNameNoExt returns a zip entry's base name without directory or extension.
func baseNameNoExt(file string) string {
	if i := strings.LastIndexAny(file, `/\`); i >= 0 {
		file = file[i+1:]
	}
	if dot := strings.LastIndexByte(file, '.'); dot > 0 {
		file = file[:dot]
	}
	return file
}
