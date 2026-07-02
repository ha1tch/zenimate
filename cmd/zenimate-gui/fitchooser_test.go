//go:build purego

package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/filepick"
)

func pngBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if (x/4+y/4)%2 == 0 {
				img.Set(x, y, color.White)
			}
		}
	}
	var b bytes.Buffer
	png.Encode(&b, img)
	return b.Bytes()
}

func TestFitChooserPick(t *testing.T) {
	c := ui.New(16, 16)
	fc := newFitChooser(c, pngBytes(40, 40), "pic.png")
	fc.layout(stubRenderer{}, 900, 700)
	rc := fc.rects[1] // Stretch
	res := fc.update(filepick.Input{MouseX: rc.X + 4, MouseY: rc.Y + 4, MousePressed: true})
	if res.state != chooserPicked || res.mode != model.FitStretch {
		t.Fatalf("expected pick Stretch, got state=%d mode=%d", res.state, res.mode)
	}
}

func TestFitChooserCancel(t *testing.T) {
	fc := newFitChooser(ui.New(16, 16), pngBytes(8, 8), "p.png")
	fc.layout(stubRenderer{}, 900, 700)
	if fc.update(filepick.Input{Keys: []filepick.Key{filepick.KeyEscape}}).state != chooserCancelled {
		t.Error("escape should cancel")
	}
}

func TestDropImageOpensFitChooser(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "art.png")
	os.WriteFile(p, pngBytes(100, 60), 0o644)

	c := ui.New(16, 16)
	var f fileOps
	f.handleDrop(c, []string{p})
	// Dropping an image must NOT load immediately; it opens the fit chooser.
	if f.fit == nil {
		t.Fatal("dropping an image should open the fit chooser")
	}
	if c.Sprite.Width() != 16 {
		t.Error("sprite should be unchanged until a fit strategy is chosen")
	}
	// Now pick Best fit via update and confirm the sprite becomes a full screen.
	f.fit.layout(stubRenderer{}, 900, 700) // draw normally does this each frame
	rc := f.fit.rects[0]
	f.update(filepick.Input{MouseX: rc.X + 4, MouseY: rc.Y + 4, MousePressed: true})
	if f.fit != nil {
		t.Error("fit chooser should close after a pick")
	}
	if c.Sprite.Width() != 256 || c.Sprite.Height() != 192 {
		t.Errorf("after fit pick, sprite is %dx%d, want 256x192", c.Sprite.Width(), c.Sprite.Height())
	}
}
