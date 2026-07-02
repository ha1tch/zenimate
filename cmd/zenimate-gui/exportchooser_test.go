//go:build purego

package main

import (
	"testing"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/filepick"
)

// stubRenderer satisfies filepick.Renderer with fixed metrics, so chooser layout
// and hit-testing can be exercised without a GL context or real font textures.
type stubRenderer struct{}

func (stubRenderer) FillRect(filepick.Rect, filepick.Colour)         {}
func (stubRenderer) StrokeRect(filepick.Rect, filepick.Colour, int)  {}
func (stubRenderer) DrawText(string, int, int, int, filepick.Colour) {}
func (stubRenderer) TextWidth(s string, scale int) int               { return len(s) * 8 * scale }
func (stubRenderer) LineHeight(scale int) int                        { return 8 * scale }
func (stubRenderer) Clip(filepick.Rect)                              {}
func (stubRenderer) ClipEnd()                                        {}

func TestExportChooserPick(t *testing.T) {
	c := ui.New(16, 16)
	ec := newExportChooser(c)
	ec.layout(stubRenderer{}, 800, 600)
	// Click the first option (SCR).
	rc := ec.rects[0]
	res := ec.update(filepick.Input{MouseX: rc.X + 4, MouseY: rc.Y + 4, MousePressed: true})
	if res.state != chooserPicked || res.format != model.FormatSCR {
		t.Fatalf("expected pick SCR, got state=%d format=%d", res.state, res.format)
	}
}

func TestExportChooserCancel(t *testing.T) {
	c := ui.New(16, 16)
	ec := newExportChooser(c)
	ec.layout(stubRenderer{}, 800, 600)
	res := ec.update(filepick.Input{Keys: []filepick.Key{filepick.KeyEscape}})
	if res.state != chooserCancelled {
		t.Errorf("escape should cancel, got %d", res.state)
	}
	// Click outside the panel also cancels.
	res = ec.update(filepick.Input{MouseX: 0, MouseY: 0, MousePressed: true})
	if res.state != chooserCancelled {
		t.Errorf("outside click should cancel, got %d", res.state)
	}
}

func TestExportChooserHasAllFormats(t *testing.T) {
	ec := newExportChooser(ui.New(16, 16))
	if len(ec.options) != 6 {
		t.Errorf("expected 6 export formats, got %d", len(ec.options))
	}
}

func TestDrawerHasExportButton(t *testing.T) {
	c := ui.New(16, 16)
	l := computeLayout(980, 680, c, &fileOps{}, 0, 1)
	found := false
	for _, b := range l.buttons {
		if b.label == "Export" {
			found = true
		}
	}
	if !found {
		t.Error("drawer missing Export button")
	}
}
