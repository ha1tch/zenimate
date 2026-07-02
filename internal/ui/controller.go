// Package ui holds the frontend-independent controller that sits between the
// sprite model and whatever presents it (the TUI or the raylib GUI). It owns the
// editing actions and the animation player state, so both frontends share one
// behaviour and only differ in presentation and input.
//
// The controller does not draw anything and does not block. A frontend:
//   - calls action methods (Toggle, NextFrame, Play, AddFrame, ...);
//   - reads state through the model and the controller's status fields;
//   - drives animation by calling Tick on its own timer when Playing is true.
package ui

import (
	"strconv"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

// PlayInterval is the animation step in milliseconds, matching the original
// editor's 50ms frame cadence (20 fps).
const PlayIntervalMS = 50

// ViewMode selects how the sprite is displayed/edited.
type ViewMode int

const (
	// BitmapBlack shows set pixels as black on the transparency chequer.
	BitmapBlack ViewMode = iota
	// BitmapWhite shows set pixels as white on the transparency chequer.
	BitmapWhite
	// SpectrumColour shows set pixels in each cell's ink colour and clear pixels
	// in its paper colour, using ZX attributes.
	SpectrumColour
)

func (m ViewMode) String() string {
	switch m {
	case BitmapWhite:
		return "Bitmap White"
	case SpectrumColour:
		return "Spectrum Colour"
	default:
		return "Bitmap Black"
	}
}

// Controller wraps a sprite model with editor actions and player state.
type Controller struct {
	Sprite *model.Sprite

	playing bool

	// status carries the last user-facing message (errors, confirmations). A
	// frontend shows it however suits its medium.
	status string

	// statusSeq increments on every status update so a frontend can detect a
	// fresh message (even identical text) and re-trigger an animation.
	statusSeq int

	onStatus func(string)

	// View mode and the current attribute selection used when painting colours
	// in Spectrum Colour mode.
	mode   ViewMode
	ink    int // 0-7
	paper  int // 0-7
	bright bool

	// Onion-skin toggles (bitmap views only): show the previous frame in
	// translucent red and the next frame in translucent green.
	onionPrev bool
	onionNext bool

	// lastX, lastY is the most recently modified pixel, used as the preview focus
	// when the cursor is not over the paint area.
	lastX, lastY int

	// saveForm selects which extension spelling files are saved with (long .zani
	// / .zbun, or 8.3 .zan / .zbu). Loading accepts every form regardless.
	saveForm model.SaveForm

	// source records where the edited sprite came from, so Save writes back to
	// the right place (a file, a bundle entry, or nowhere yet).
	source SpriteSource
}

// SaveForm returns the current save-extension form.
func (c *Controller) SaveForm() model.SaveForm { return c.saveForm }

// SetSaveForm selects the save-extension form (long or 8.3).
func (c *Controller) SetSaveForm(f model.SaveForm) { c.saveForm = f }

// ToggleSaveForm flips between long and 8.3 save forms and reports the new state.
func (c *Controller) ToggleSaveForm() {
	if c.saveForm == model.SaveFormLong {
		c.saveForm = model.SaveForm83
		c.setStatus("Save form: 8.3 (.zan / .zbu)")
	} else {
		c.saveForm = model.SaveFormLong
		c.setStatus("Save form: long (.zani / .zbun)")
	}
}

// New builds a controller over a fresh sprite of the given dimensions.
func New(w, h int) *Controller {
	s := model.New(w, h)
	return &Controller{
		Sprite: s,
		mode:   BitmapBlack, // default view mode
		ink:    7,           // white ink
		paper:  0,           // black paper
		bright: false,
		lastX:  s.Width() / 2,
		lastY:  s.Height() / 2,
	}
}

// Mode returns the current view mode.
func (c *Controller) Mode() ViewMode { return c.mode }

// LoadSprite replaces the edited sprite (e.g. after opening a file) and resets
// the sprite-derived editor state: the preview focus re-centres on the new
// sprite, and a status message is posted. The view mode and colour selection are
// intentionally preserved, since they are editor preferences, not document data.
// SourceKind describes where the edited sprite came from, which determines what
// Save writes to.
type SourceKind int

const (
	// SourceNone: a new or imported sprite with no save target yet.
	SourceNone SourceKind = iota
	// SourceFile: opened from a standalone .zani/.zan/.zcut file.
	SourceFile
	// SourceBundle: opened from an entry inside a .zbun bundle.
	SourceBundle
)

// SpriteSource records the provenance of the edited sprite so Save can write
// back to the right place. For SourceBundle, Path is the bundle file, Entry is
// the animation name within it, and Label preserves the manifest label across a
// round-trip. SaveResolved (bundle case only) remembers a prior "update in
// bundle vs save separate" choice so Save asks at most once.
type SpriteSource struct {
	Kind         SourceKind
	Path         string // file path (SourceFile) or bundle path (SourceBundle)
	Entry        string // animation name within the bundle (SourceBundle)
	Label        string // manifest label to preserve (SourceBundle)
	SaveResolved bool   // a save decision has been made for this bundle source
	SaveToBundle bool   // the remembered decision: true = update in bundle
}

func (c *Controller) LoadSprite(s *model.Sprite) {
	c.Sprite = s
	c.lastX = s.Width() / 2
	c.lastY = s.Height() / 2
	c.source = SpriteSource{Kind: SourceNone} // caller sets provenance if any
	c.setStatus("Loaded " + s.Name())
}

// Source returns the current sprite's provenance.
func (c *Controller) Source() SpriteSource { return c.source }

// SetSource records the current sprite's provenance (called by open/save flows).
func (c *Controller) SetSource(src SpriteSource) { c.source = src }

// SourceLabel returns a short human description of the current source for the
// header: "name.zani", "name - bundle.zbun", or "name (unsaved)".
func (c *Controller) SourceLabel() string {
	switch c.source.Kind {
	case SourceFile:
		return baseNameOf(c.source.Path)
	case SourceBundle:
		return c.source.Entry + " - " + baseNameOf(c.source.Path)
	default:
		name := c.Sprite.Name()
		if name == "" {
			name = "untitled"
		}
		return name + " (unsaved)"
	}
}

// baseNameOf returns the final path element of p.
func baseNameOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}

// SetMode changes the view mode.
func (c *Controller) SetMode(m ViewMode) {
	c.mode = m
	c.setStatus("Mode: " + m.String())
}

// Ink, Paper, Bright are the current attribute selection used when painting in
// Spectrum Colour mode.
func (c *Controller) Ink() int     { return c.ink }
func (c *Controller) Paper() int   { return c.paper }
func (c *Controller) Bright() bool { return c.bright }

// SetInk / SetPaper set the selected ink/paper colour (0-7).
func (c *Controller) SetInk(i int)   { c.ink = clampColour(i) }
func (c *Controller) SetPaper(i int) { c.paper = clampColour(i) }

// ToggleBright flips the bright bit of the current selection.
func (c *Controller) ToggleBright() { c.bright = !c.bright }

// SetBright sets the bright flag of the current selection.
func (c *Controller) SetBright(b bool) { c.bright = b }

// Onion-skin state. Onion skins are shown only in the bitmap view modes (never
// in Spectrum Colour); the frontend honours that.
func (c *Controller) OnionPrev() bool { return c.onionPrev }
func (c *Controller) OnionNext() bool { return c.onionNext }

// ToggleOnionPrev / ToggleOnionNext flip the previous/next onion-skin overlays.
func (c *Controller) ToggleOnionPrev() {
	c.onionPrev = !c.onionPrev
	c.setStatus("Onion (prev): " + onOff(c.onionPrev))
}
func (c *Controller) ToggleOnionNext() {
	c.onionNext = !c.onionNext
	c.setStatus("Onion (next): " + onOff(c.onionNext))
}

// PrevFrameIndex / NextFrameIndex return the wrap-around neighbour frame
// indices of the current selection.
func (c *Controller) PrevFrameIndex() int {
	n := c.Sprite.FrameCount()
	return (c.Sprite.Selected() - 1 + n) % n
}
func (c *Controller) NextFrameIndex() int {
	n := c.Sprite.FrameCount()
	return (c.Sprite.Selected() + 1) % n
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// SetCellInk sets the ink colour of character-cell (cx,cy), preserving its paper
// and bright. Used when left-clicking a cell with the selected ink.
func (c *Controller) SetCellInk(cx, cy, ink int) {
	a := c.Sprite.AttrCell(cx, cy)
	na := zxpalette.Attr(ink, zxpalette.Paper(a), c.bright, zxpalette.Flash(a))
	c.Sprite.SetAttrCell(cx, cy, na)
}

// SetCellPaper sets the paper colour of character-cell (cx,cy), preserving its
// ink and bright. Used when right-clicking a cell with the selected paper.
func (c *Controller) SetCellPaper(cx, cy, paper int) {
	a := c.Sprite.AttrCell(cx, cy)
	na := zxpalette.Attr(zxpalette.Ink(a), paper, c.bright, zxpalette.Flash(a))
	c.Sprite.SetAttrCell(cx, cy, na)
}

func clampColour(i int) int {
	if i < 0 {
		return 0
	}
	if i > 7 {
		return 7
	}
	return i
}

// OnStatus registers a callback for status messages (optional).
func (c *Controller) OnStatus(fn func(string)) { c.onStatus = fn }

func (c *Controller) setStatus(s string) {
	c.status = s
	c.statusSeq++
	if c.onStatus != nil {
		c.onStatus(s)
	}
}

// Status returns the last status message.
func (c *Controller) Status() string { return c.status }

// StatusSeq returns a counter that increments every time the status is set,
// even to the same text. A frontend can watch it to detect a fresh message and
// (re)trigger a notification animation.
func (c *Controller) StatusSeq() int { return c.statusSeq }

// SetStatus sets a user-facing status message (frontends use this for actions
// the controller itself does not perform, e.g. writing a file).
func (c *Controller) SetStatus(s string) { c.setStatus(s) }

// Playing reports whether the animation player is running.
func (c *Controller) Playing() bool { return c.playing }

// --- editing actions -------------------------------------------------------

// Toggle flips a pixel in the selected frame.
func (c *Controller) Toggle(x, y int) {
	c.Sprite.Toggle(x, y)
	c.markPixel(x, y)
}

// Set forces a pixel value (used for click-drag painting).
func (c *Controller) Set(x, y int, on bool) {
	c.Sprite.Set(x, y, on)
	c.markPixel(x, y)
}

// Paint sets pixel (x,y) on or off. It no longer stamps attributes — attribute
// painting is a separate, explicitly-modified action (see PaintAttr), so normal
// drawing never disturbs a cell's colours.
func (c *Controller) Paint(x, y int, on bool) {
	c.Sprite.Set(x, y, on)
	c.markPixel(x, y)
}

// PaintAttr stamps the current ink/paper/bright selection onto the character
// cell containing pixel (x,y), without changing the bitmap. The GUI invokes
// this when the user paints with Ctrl held (Ctrl acts as an attribute-paint
// modifier). Meaningful in Spectrum Colour mode; harmless in any mode.
func (c *Controller) PaintAttr(x, y int) {
	cx, cy := x/8, y/8
	a := c.Sprite.AttrCell(cx, cy)
	na := zxpalette.Attr(c.ink, c.paper, c.bright, zxpalette.Flash(a))
	c.Sprite.SetAttrCell(cx, cy, na)
	c.markPixel(x, y)
}

// markPixel records the most recently modified pixel, used as the preview's
// focus point when the cursor is not over the paint area.
func (c *Controller) markPixel(x, y int) {
	if x >= 0 && y >= 0 && x < c.Sprite.Width() && y < c.Sprite.Height() {
		c.lastX, c.lastY = x, y
	}
}

// LastPixel returns the most recently modified pixel (defaults to the sprite
// centre before any edit).
func (c *Controller) LastPixel() (int, int) { return c.lastX, c.lastY }

// SelectFrame switches to a frame by index.
func (c *Controller) SelectFrame(i int) { c.Sprite.Select(i) }

// NextFrame / PrevFrame move the selection by one, wrapping.
func (c *Controller) NextFrame() {
	n := c.Sprite.FrameCount()
	c.Sprite.Select((c.Sprite.Selected() + 1) % n)
}
func (c *Controller) PrevFrame() {
	n := c.Sprite.FrameCount()
	c.Sprite.Select((c.Sprite.Selected() - 1 + n) % n)
}

// CopyFrame / PasteFrame / ClearFrame operate on the selected frame.
func (c *Controller) CopyFrame()  { c.Sprite.CopyFrame(); c.setStatus("Frame copied") }
func (c *Controller) PasteFrame() { c.Sprite.PasteFrame(); c.setStatus("Frame pasted") }
func (c *Controller) ClearFrame() { c.Sprite.ClearFrame() }

// ClearFrameBitmap clears only the selected frame's pixels, preserving colour.
func (c *Controller) ClearFrameBitmap() {
	c.Sprite.ClearFrameBitmap()
	c.setStatus("Cleared pixels (colour kept)")
}

// FlipH / FlipV / Invert / Rotate90 transform the selected frame.
func (c *Controller) FlipH()  { c.Sprite.FlipH(); c.setStatus("Flipped horizontally") }
func (c *Controller) FlipV()  { c.Sprite.FlipV(); c.setStatus("Flipped vertically") }
func (c *Controller) Invert() { c.Sprite.Invert(); c.setStatus("Inverted pixels") }

// Rotate90 rotates the selected frame 90 degrees clockwise. When resize is true
// a non-square frame is resized (swapping width/height) so no content is lost.
func (c *Controller) Rotate90(resize bool) {
	c.Sprite.Rotate90(resize)
	if resize {
		c.setStatus("Rotated 90 (resized)")
	} else {
		c.setStatus("Rotated 90")
	}
}

// Reset empties all frames.
func (c *Controller) Reset() {
	c.Sprite.Reset()
	c.setStatus("Reset")
}

// ResetAll restores the sprite to a fresh default state (default size, default
// frame count, everything cleared). Destructive — the GUI confirms first.
func (c *Controller) ResetAll() {
	c.Sprite.ResetAll()
	c.setStatus("Reset to defaults")
}

// ClearFrameCLS clears the current frame (pixels and colour), the CLS action.
func (c *Controller) ClearFrameCLS() {
	c.Sprite.ClearFrame()
	c.setStatus("Cleared frame")
}

// SetName changes the export label prefix.
func (c *Controller) SetName(name string) { c.Sprite.SetName(name) }

// SetSize changes the sprite dimensions non-destructively (pixels and
// attributes are preserved within the overlapping region).
func (c *Controller) SetSize(w, h int) {
	c.Sprite.SetSize(w, h)
	c.setStatus("Size " + strconv.Itoa(c.Sprite.Width()) + "x" + strconv.Itoa(c.Sprite.Height()))
}

// SetWidth / SetHeight change one dimension, non-destructively.
func (c *Controller) SetWidth(w int)  { c.SetSize(w, c.Sprite.Height()) }
func (c *Controller) SetHeight(h int) { c.SetSize(c.Sprite.Width(), h) }

// AddFrame appends a frame (up to the maximum) and selects it.
func (c *Controller) AddFrame() {
	if c.Sprite.AddFrame() {
		c.setStatus("Frame added (" + strconv.Itoa(c.Sprite.FrameCount()) + ")")
	} else {
		c.setStatus("Maximum frames reached")
	}
}

// RemoveFrame deletes the last frame (down to the minimum).
func (c *Controller) RemoveFrame() {
	if c.Sprite.RemoveFrame() {
		c.setStatus("Frame removed (" + strconv.Itoa(c.Sprite.FrameCount()) + ")")
	} else {
		c.setStatus("Minimum frames reached")
	}
}

// --- animation player ------------------------------------------------------

// TogglePlay starts or stops the animation player.
func (c *Controller) TogglePlay() {
	c.playing = !c.playing
	if c.playing {
		c.setStatus("Playing")
	} else {
		c.setStatus("Stopped")
	}
}

// Tick advances one animation frame if playing. A frontend calls this from its
// own timer at PlayIntervalMS cadence.
func (c *Controller) Tick() {
	if c.playing {
		c.Sprite.Advance()
	}
}
