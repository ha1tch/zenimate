package main

import (
	"fmt"
	"os"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/filepick"
)

// fileOps owns the modal file dialog. At most one dialog is open at a time; while
// open it captures all input (the editor underneath is frozen) and is drawn on
// top. When the user accepts, the stored action runs with the chosen path.
//
// This is deliberately a thin state machine: Step 3 (ZCUT save/load) and Step 5
// (SCR/tape/snapshot export) add new openers and actions without touching the
// editor's main loop, which only has to ask "is a dialog active?" and route
// input accordingly.
type fileOps struct {
	dlg      *filepick.Dialog
	action   func(path string) // run on Accepted, with the chosen path
	chooser  *exportChooser    // export format chooser, when active
	bundle   *bundleChooser    // bundle create/add chooser, when active
	fit      *fitChooser       // image-import fit chooser, when active
	saveProv *saveProvChooser  // save-provenance chooser (bundle vs separate)
}

// active reports whether a dialog or any chooser is currently open (and thus the
// editor should be modal).
func (f *fileOps) active() bool {
	return f.dlg != nil || f.chooser != nil || f.bundle != nil || f.fit != nil || f.saveProv != nil
}

// open starts a dialog with the given config and the action to run when the user
// confirms a choice. Any dialog already open is replaced.
func (f *fileOps) open(cfg filepick.Config, action func(path string)) {
	f.dlg = filepick.New(cfg)
	f.action = action
}

// close dismisses any open dialog without running its action.
func (f *fileOps) close() {
	f.dlg = nil
	f.action = nil
}

// update feeds one frame of input to the open dialog and resolves its lifecycle.
// On Accepted it runs the stored action with the result and closes; on Cancelled
// it just closes. It is a no-op when no dialog is open. Returns true if a dialog
// was active this frame (so the caller can suppress editor input).
func (f *fileOps) update(in filepick.Input) bool {
	// The save-provenance chooser takes precedence, then the image fit chooser,
	// then the bundle chooser, then the export chooser, then the file dialog.
	if f.saveProv != nil {
		switch r := f.saveProv.update(in); r.state {
		case chooserPicked:
			sp := f.saveProv
			f.saveProv = nil
			f.applySaveProv(sp.ctrl, sp.src, r.toBundle)
		case chooserCancelled:
			f.saveProv = nil
		}
		return true
	}
	if f.fit != nil {
		switch r := f.fit.update(in); r.state {
		case chooserPicked:
			fc := f.fit
			f.fit = nil
			f.applyImageImport(fc.ctrl, fc.data, fc.name, r.mode)
		case chooserCancelled:
			f.fit = nil
		}
		return true
	}
	if f.bundle != nil {
		switch r := f.bundle.update(in); r.state {
		case chooserPicked:
			bc := f.bundle
			f.bundle = nil
			f.writeBundle(bc.ctrl, bc.path, bc.name, bc.label, r.mode)
		case chooserCancelled:
			f.bundle = nil
		}
		return true
	}
	if f.chooser != nil {
		switch r := f.chooser.update(in); r.state {
		case chooserPicked:
			pick := r.format
			f.chooser = nil
			f.startSaveExport(r.ctrl, pick)
		case chooserCancelled:
			f.chooser = nil
		}
		return true
	}
	if f.dlg == nil {
		return false
	}
	switch f.dlg.Update(in) {
	case filepick.Accepted:
		path := f.dlg.Result()
		act := f.action
		f.close()
		if act != nil {
			act(path)
		}
	case filepick.Cancelled:
		f.close()
	}
	return true
}

// draw renders the export chooser or the open dialog over the editor.
func (f *fileOps) draw(r fpRenderer, screenW, screenH int) {
	if f.saveProv != nil {
		f.saveProv.draw(r, screenW, screenH)
		return
	}
	if f.fit != nil {
		f.fit.draw(r, screenW, screenH)
		return
	}
	if f.bundle != nil {
		f.bundle.draw(r, screenW, screenH)
		return
	}
	if f.chooser != nil {
		f.chooser.draw(r, screenW, screenH)
		return
	}
	if f.dlg == nil {
		return
	}
	f.dlg.Draw(r, screenW, screenH, fpTheme())
}

// applySaveProv records the user's bundle-vs-separate save decision on the
// source (so Save won't re-ask) and performs it.
func (f *fileOps) applySaveProv(c *ui.Controller, src ui.SpriteSource, toBundle bool) {
	src.SaveResolved = true
	src.SaveToBundle = toBundle
	c.SetSource(src)
	if toBundle {
		f.saveIntoSourceBundle(c, src)
	} else {
		f.startSaveAs(c)
	}
}

// save is the provenance-aware Save (Ctrl+S). It writes back to wherever the
// sprite came from: a standalone file is overwritten silently; a bundle entry
// prompts (once) whether to update the bundle or split off a separate .zani; a
// sprite with no source falls back to Save As.
func (f *fileOps) save(c *ui.Controller) {
	src := c.Source()
	switch src.Kind {
	case ui.SourceFile:
		f.writeAnimationFile(c, src.Path)
	case ui.SourceBundle:
		if src.SaveResolved {
			if src.SaveToBundle {
				f.saveIntoSourceBundle(c, src)
			} else {
				f.startSaveAs(c) // user previously chose "separate .zani"
			}
			return
		}
		// Ask once: update in the bundle, or save as a separate .zani.
		f.saveProv = newSaveProvChooser(c, src)
	default:
		f.startSaveAs(c)
	}
}

// startSaveAs always opens a Save dialog for a standalone .zani (Ctrl+Shift+S),
// regardless of provenance. On success the sprite's source becomes that file.
func (f *fileOps) startSaveAs(c *ui.Controller) {
	ext := model.AnimationExt(c.SaveForm())
	f.open(filepick.Config{
		Mode:       filepick.ModeSave,
		Title:      "Save animation as",
		Filters:    model.AnimationExtensions(),
		DefaultExt: ext,
	}, func(path string) {
		f.writeAnimationFile(c, path)
	})
}

// writeAnimationFile writes the sprite as a standalone .zani at path and records
// file provenance.
func (f *fileOps) writeAnimationFile(c *ui.Controller, path string) {
	data, err := c.Sprite.MarshalZCUT()
	if err != nil {
		c.SetStatus("Save failed: " + err.Error())
		return
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		c.SetStatus("Save failed: " + err.Error())
		return
	}
	c.SetSource(ui.SpriteSource{Kind: ui.SourceFile, Path: path})
	c.SetStatus(fmt.Sprintf("Saved %s (%d frames)", baseName(path), c.Sprite.FrameCount()))
}

// saveIntoSourceBundle updates the sprite's entry inside the bundle it came
// from, preserving its label, and remembers the decision on the source.
func (f *fileOps) saveIntoSourceBundle(c *ui.Controller, src ui.SpriteSource) {
	data, err := os.ReadFile(src.Path)
	if err != nil {
		c.SetStatus("Bundle save failed: " + err.Error())
		return
	}
	b, err := model.OpenBundle(data)
	if err != nil {
		c.SetStatus("Bundle save failed: " + err.Error())
		return
	}
	if err := b.AddSprite(src.Entry, src.Label, c.Sprite); err != nil {
		c.SetStatus("Bundle save failed: " + err.Error())
		return
	}
	out, err := b.Encode()
	if err != nil {
		c.SetStatus("Bundle save failed: " + err.Error())
		return
	}
	if err := os.WriteFile(src.Path, out, 0o644); err != nil {
		c.SetStatus("Bundle save failed: " + err.Error())
		return
	}
	src.SaveResolved = true
	src.SaveToBundle = true
	c.SetSource(src)
	c.SetStatus(fmt.Sprintf("Updated %s in %s", src.Entry, baseName(src.Path)))
}

// startOpen opens an Open dialog for every file type zenimate can load — native
// animations (.zani/.zan, and the .zcut alias), raw screens (.scr), screen-bearing
// tape/snapshot containers (.tap/.tzx/.sna/.z80), and raster images
// (.jpg/.jpeg/.png/.gif). Images route through the fit-strategy chooser; all
// other types load directly by extension.
func (f *fileOps) startOpen(c *ui.Controller) {
	filters := append(model.LoadableExtensions(), "jpg", "jpeg", "png", "gif")
	filters = append(filters, model.BundleExtensions()...)
	f.open(filepick.Config{
		Mode:    filepick.ModeOpen,
		Title:   "Open sprite, animation, screen or image",
		Filters: filters,
		FS:      bundleFS(),
		Preview: bundlePreview,
	}, func(path string) {
		// A result inside a bundle is "<bundle>#<entry>": load that animation.
		if bundlePath, entry, ok := splitBundleRef(path); ok {
			f.openBundleEntry(c, bundlePath, entry)
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			c.SetStatus("Open failed: " + err.Error())
			return
		}
		if model.IsImageExt(extOf(path)) {
			f.startImageImport(c, data, baseName(path))
			return
		}
		s, err := model.LoadByExtension(extOf(path), data)
		if err != nil {
			c.SetStatus("Open failed: " + err.Error())
			return
		}
		s.SetName(baseName(path))
		c.LoadSprite(s)
		selectModeForExt(c, extOf(path))
		// Only animation files are a save target; screens/tapes are imports.
		if model.IsAnimationExt(extOf(path)) {
			c.SetSource(ui.SpriteSource{Kind: ui.SourceFile, Path: path})
		}
	})
}

// openBundleEntry loads one named animation out of a .zbun bundle.
func (f *fileOps) openBundleEntry(c *ui.Controller, bundlePath, entry string) {
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		c.SetStatus("Open failed: " + err.Error())
		return
	}
	b, err := model.OpenBundle(data)
	if err != nil {
		c.SetStatus("Open failed: " + err.Error())
		return
	}
	s, err := b.Sprite(entry)
	if err != nil {
		c.SetStatus("Open failed: " + err.Error())
		return
	}
	s.SetName(entry)
	c.LoadSprite(s)
	// Record bundle provenance, preserving the entry's label for round-tripping.
	label := ""
	for _, e := range b.Entries() {
		if e.Name == entry {
			label = e.Label
		}
	}
	c.SetSource(ui.SpriteSource{Kind: ui.SourceBundle, Path: bundlePath, Entry: entry, Label: label})
}

// splitBundleRef splits a "<bundle>#<entry>" reference produced by the file
// dialog when a bundle entry is chosen. ok is false for a plain path.
func splitBundleRef(path string) (bundlePath, entry string, ok bool) {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '#' {
			return path[:i], path[i+1:], true
		}
	}
	return "", "", false
}

// baseName returns the final path element (after the last '/'), so status
// messages show a filename rather than a full path.
func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// startExport opens the export-format chooser. Picking a format then opens a
// Save dialog (see startSaveExport).
func (f *fileOps) startExport(c *ui.Controller) {
	f.chooser = newExportChooser(c)
}

// startSaveExport opens a Save dialog for a chosen export format and, on accept,
// renders the current frame to that format and writes it.
func (f *fileOps) startSaveExport(c *ui.Controller, format model.ExportFormat) {
	ext := model.ExportExt(format)
	f.open(filepick.Config{
		Mode:       filepick.ModeSave,
		Title:      "Export " + upper(ext),
		Filters:    []string{ext},
		DefaultExt: ext,
	}, func(path string) {
		// Use a neutral embedded name (never personal data) for tape/snapshot
		// containers: the output file's base name without its extension.
		name := baseName(path)
		if dot := lastDot(name); dot > 0 {
			name = name[:dot]
		}
		data, err := c.Sprite.ExportScreen(c.Sprite.Selected(), format, name)
		if err != nil {
			c.SetStatus("Export failed: " + err.Error())
			return
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			c.SetStatus("Export failed: " + err.Error())
			return
		}
		c.SetStatus("Exported " + baseName(path))
	})
}

// lastDot returns the index of the last '.' in s, or -1.
func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

// startBundleExport begins adding the current sprite to a .zbun bundle. It opens
// a Save dialog to pick the bundle path; the accept handler then decides (via the
// bundle chooser) whether to create a new bundle or add to an existing one, and
// prompts for the entry's name/label.
func (f *fileOps) startBundleExport(c *ui.Controller) {
	ext := model.BundleExt(c.SaveForm())
	f.open(filepick.Config{
		Mode:       filepick.ModeSave,
		Title:      "Save to bundle",
		Filters:    model.BundleExtensions(),
		DefaultExt: ext,
	}, func(path string) {
		// The entry name defaults to the sprite's current name.
		name := c.Sprite.Name()
		exists := fileExists(path)
		f.bundle = newBundleChooser(c, path, name, "", exists)
	})
}

// writeBundle performs the chosen bundle operation: create a fresh bundle or add
// to (a copy of) the existing one, insert the current sprite under name/label,
// and write the result back to path.
func (f *fileOps) writeBundle(c *ui.Controller, path, name, label string, mode bundleMode) {
	var b *model.Bundle
	switch mode {
	case bundleAdd:
		data, err := os.ReadFile(path)
		if err != nil {
			c.SetStatus("Bundle add failed: " + err.Error())
			return
		}
		b, err = model.OpenBundle(data)
		if err != nil {
			c.SetStatus("Bundle add failed: " + err.Error())
			return
		}
	default: // bundleCreate
		b = model.NewBundle()
	}

	if err := b.AddSprite(name, label, c.Sprite); err != nil {
		c.SetStatus("Bundle failed: " + err.Error())
		return
	}
	out, err := b.Encode()
	if err != nil {
		c.SetStatus("Bundle failed: " + err.Error())
		return
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		c.SetStatus("Bundle failed: " + err.Error())
		return
	}
	verb := "Created"
	if mode == bundleAdd {
		verb = "Added to"
	}
	c.SetStatus(fmt.Sprintf("%s %s (%d animations)", verb, baseName(path), b.Len()))
}

// startOpenInBundle opens the file dialog directly inside a bundle so the user
// can pick which animation to open (used when a .zbun is dropped on the window).
func (f *fileOps) startOpenInBundle(c *ui.Controller, bundlePath string) {
	f.open(filepick.Config{
		Mode:           filepick.ModeOpen,
		Title:          "Open animation from bundle",
		Filters:        model.AnimationExtensions(),
		FS:             bundleFS(),
		Preview:        bundlePreview,
		StartContainer: bundlePath,
	}, func(path string) {
		if bp, entry, ok := splitBundleRef(path); ok {
			f.openBundleEntry(c, bp, entry)
			return
		}
		// Fell out of the bundle and picked a normal file.
		data, err := os.ReadFile(path)
		if err != nil {
			c.SetStatus("Open failed: " + err.Error())
			return
		}
		s, err := model.LoadByExtension(extOf(path), data)
		if err != nil {
			c.SetStatus("Open failed: " + err.Error())
			return
		}
		s.SetName(baseName(path))
		c.LoadSprite(s)
		selectModeForExt(c, extOf(path))
		if model.IsAnimationExt(extOf(path)) {
			c.SetSource(ui.SpriteSource{Kind: ui.SourceFile, Path: path})
		}
	})
}

// fileExists reports whether a regular file exists at path.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// selectModeForExt switches the view to Spectrum Colour after loading a raw
// screen (.scr), which is inherently a colour image, so the user sees its real
// attributes immediately rather than a monochrome bitmap view.
func selectModeForExt(c *ui.Controller, ext string) {
	if ext == "scr" {
		c.SetMode(ui.SpectrumColour)
	}
}

// handleDrop loads the first dropped file whose extension zenimate recognises,
// replacing the edited sprite. Unsupported files are skipped; if none load, a
// status message explains why. Multiple files are not merged — the first usable
// one wins, which matches the common "drag a sprite onto the window" intent.
func (f *fileOps) handleDrop(c *ui.Controller, paths []string) {
	if len(paths) == 0 {
		return
	}
	var lastErr string
	for _, p := range paths {
		ext := extOf(p)
		if ext == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			lastErr = err.Error()
			continue
		}
		if model.IsImageExt(ext) {
			// Images need a fit strategy: open the chooser instead of loading now.
			f.startImageImport(c, data, baseName(p))
			return
		}
		if model.IsBundleExt(ext) {
			// A bundle isn't a single animation: open the browser to pick one.
			f.startOpenInBundle(c, p)
			return
		}
		s, err := model.LoadByExtension(ext, data)
		if err != nil {
			lastErr = err.Error()
			continue
		}
		s.SetName(baseName(p))
		c.LoadSprite(s)
		selectModeForExt(c, ext)
		return
	}
	if lastErr != "" {
		c.SetStatus("Drop failed: " + lastErr)
	} else {
		c.SetStatus("Drop: no supported file (try .zani .zan .scr .tap .tzx .sna .z80 or an image)")
	}
}

// extOf returns the lowercase extension (without the dot) of a path, or "".
func extOf(path string) string {
	dot := lastDot(path)
	slash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			slash = i
			break
		}
	}
	if dot <= slash || dot == len(path)-1 {
		return ""
	}
	ext := path[dot+1:]
	low := make([]byte, len(ext))
	for i := 0; i < len(ext); i++ {
		c := ext[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		low[i] = c
	}
	return string(low)
}

// startImageImport opens the fit-strategy chooser for an image already read into
// memory. Picking a strategy applies it and loads the result.
func (f *fileOps) startImageImport(c *ui.Controller, data []byte, name string) {
	f.fit = newFitChooser(c, data, name)
}

// applyImageImport reduces the pending image with the chosen fit strategy and
// replaces the edited sprite.
func (f *fileOps) applyImageImport(c *ui.Controller, data []byte, name string, mode model.FitMode) {
	s, err := model.LoadImage(data, mode)
	if err != nil {
		c.SetStatus("Import failed: " + err.Error())
		return
	}
	s.SetName(name)
	c.LoadSprite(s)
	c.SetStatus("Imported " + name + " (" + model.FitModeName(mode) + ")")
}
