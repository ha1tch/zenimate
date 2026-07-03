// Package model holds the sprite editor's data model. It is deliberately free of
// any UI, rendering, or assembly-format concern: it knows only about a fixed
// number of animation frames, each a rectangular grid of on/off pixels, plus a
// sprite name and the currently selected frame. Frontends (TUI, GUI) observe it
// through a change callback and drive it through its methods.
//
// Pixel storage is row-major: index = y*Width + x, matching the original
// browser editor so exported data is byte-identical.
package model

// DefaultFrames is the initial number of animation frames. The frame count is
// variable (see AddFrame/RemoveFrame), bounded by MinFrames..MaxFrames.
const (
	DefaultFrames = 8
	MinFrames     = 1
	// MaxFrames bounds the variable frame count. The old 8-frame ceiling was a
	// constraint of the assembly export's per-frame bit-shift; with that codec
	// gone, frames are limited only by this editor cap.
	MaxFrames = 16
)

// Cell is the ZX Spectrum character-cell size in pixels.
const Cell = 8

// DefaultWidth and DefaultHeight are the dimensions of a fresh sprite (and the
// size a full reset restores). Two character cells square.
const (
	DefaultWidth  = 2 * Cell
	DefaultHeight = 2 * Cell
)

// Sprite size limits, in character cells and the pixels they imply. The full ZX
// screen is 32x24 cells (256x192 px).
const (
	MinCellsW = 1
	MinCellsH = 1
	MaxCellsW = 32
	MaxCellsH = 24

	MinWidth  = MinCellsW * Cell
	MinHeight = MinCellsH * Cell
	MaxWidth  = MaxCellsW * Cell
	MaxHeight = MaxCellsH * Cell
)

// Frame is one animation frame: a row-major grid of on/off pixels of the model's
// current Width x Height.
type Frame []bool

// clone returns an independent copy of the frame.
func (f Frame) clone() Frame {
	c := make(Frame, len(f))
	copy(c, f)
	return c
}

// Sprite is the full editable document: dimensions, a variable number of frames,
// the selected frame index, and the sprite name used as the export label prefix.
type Sprite struct {
	width, height int
	frames        []Frame
	selected      int
	name          string

	clipboard     Frame  // last copied frame, nil if none
	clipboardAttr []byte // attribute map copied alongside the frame

	// frameAttrs holds one attribute map per frame (parallel to frames): each is
	// one ZX attribute byte per 8x8 character cell, row-major over attrCols x
	// attrRows. Attributes are per-frame, so frames can carry different colours.
	frameAttrs         [][]byte
	attrCols, attrRows int

	onChange func()
}

// DefaultAttr is the attribute applied to fresh cells: white ink (7) on black
// paper (0), not bright — the classic Spectrum default.
const DefaultAttr = byte(0x07)

// New creates a sprite of the given dimensions with DefaultFrames empty frames.
// Dimensions are snapped to whole character cells and clamped to the valid
// range.
func New(w, h int) *Sprite {
	w, h = clampSize(w, h)
	s := &Sprite{width: w, height: h, name: "frame"}
	s.frames = make([]Frame, DefaultFrames)
	s.frameAttrs = make([][]byte, DefaultFrames)
	s.resetFrames()
	s.resetAttrs()
	return s
}

// OnChange registers a callback invoked after any mutation. Passing nil clears
// it. Only one observer is supported; frontends own the single UI refresh path.
func (s *Sprite) OnChange(fn func()) { s.onChange = fn }

func (s *Sprite) notify() {
	if s.onChange != nil {
		s.onChange()
	}
}

// --- accessors -------------------------------------------------------------

// Width and Height are the current grid dimensions in pixels.
func (s *Sprite) Width() int  { return s.width }
func (s *Sprite) Height() int { return s.height }

// Name is the sprite name used as the export label prefix.
func (s *Sprite) Name() string { return s.name }

// Selected is the index (0..FrameCount-1) of the active frame.
func (s *Sprite) Selected() int { return s.selected }

// HasClipboard reports whether a frame has been copied and can be pasted.
func (s *Sprite) HasClipboard() bool { return s.clipboard != nil }

// Frame returns the frame at index i. The returned slice is the live backing
// store; callers must not retain or mutate it directly — use Set/Toggle.
func (s *Sprite) Frame(i int) Frame {
	if i < 0 || i >= len(s.frames) {
		return nil
	}
	return s.frames[i]
}

// Current returns the live backing slice of the selected frame.
func (s *Sprite) Current() Frame { return s.frames[s.selected] }

// AttrCols and AttrRows are the character-cell dimensions of the attribute map.
func (s *Sprite) AttrCols() int { return s.attrCols }
func (s *Sprite) AttrRows() int { return s.attrRows }

// AttrAt returns the selected frame's attribute byte for the character cell
// containing pixel (x,y). Out-of-range reads return DefaultAttr.
func (s *Sprite) AttrAt(x, y int) byte { return s.AttrAtFrame(s.selected, x, y) }

// AttrAtFrame returns frame f's attribute byte for the cell containing (x,y).
func (s *Sprite) AttrAtFrame(f, x, y int) byte {
	return s.AttrCellFrame(f, x/8, y/8)
}

// AttrCell returns the selected frame's attribute byte for cell (cx,cy).
func (s *Sprite) AttrCell(cx, cy int) byte { return s.AttrCellFrame(s.selected, cx, cy) }

// AttrCellFrame returns frame f's attribute byte for character-cell (cx,cy).
func (s *Sprite) AttrCellFrame(f, cx, cy int) byte {
	if f < 0 || f >= len(s.frameAttrs) || cx < 0 || cy < 0 || cx >= s.attrCols || cy >= s.attrRows {
		return DefaultAttr
	}
	return s.frameAttrs[f][cy*s.attrCols+cx]
}

// SetAttrCell sets the selected frame's attribute byte for cell (cx,cy).
func (s *Sprite) SetAttrCell(cx, cy int, attr byte) {
	if cx < 0 || cy < 0 || cx >= s.attrCols || cy >= s.attrRows {
		return
	}
	s.frameAttrs[s.selected][cy*s.attrCols+cx] = attr
	s.notify()
}

// At reports whether pixel (x,y) of the selected frame is on. Out-of-range
// coordinates read as off.
func (s *Sprite) At(x, y int) bool {
	if x < 0 || y < 0 || x >= s.width || y >= s.height {
		return false
	}
	return s.frames[s.selected][y*s.width+x]
}

// --- mutations -------------------------------------------------------------

// SetName updates the sprite name (export label prefix).
func (s *Sprite) SetName(name string) {
	s.name = name
	s.notify()
}

// FrameCount returns the current number of animation frames.
func (s *Sprite) FrameCount() int { return len(s.frames) }

// Select changes the active frame, clamped to a valid index.
func (s *Sprite) Select(i int) {
	if i < 0 {
		i = 0
	}
	if i >= len(s.frames) {
		i = len(s.frames) - 1
	}
	if i == s.selected {
		return
	}
	s.selected = i
	s.notify()
}

// Advance moves to the next frame, wrapping after the last. This is the step the
// animation player uses.
func (s *Sprite) Advance() {
	s.selected = (s.selected + 1) % len(s.frames)
	s.notify()
}

// AddFrame appends one empty frame (with default attributes) and selects it, up
// to MaxFrames. Returns false if already at the maximum.
func (s *Sprite) AddFrame() bool {
	if len(s.frames) >= MaxFrames {
		return false
	}
	s.frames = append(s.frames, make(Frame, s.width*s.height))
	m := make([]byte, s.attrCols*s.attrRows)
	for i := range m {
		m[i] = DefaultAttr
	}
	s.frameAttrs = append(s.frameAttrs, m)
	s.selected = len(s.frames) - 1
	s.notify()
	return true
}

// RemoveFrame deletes the last frame, down to MinFrames. The selection is
// clamped. Returns false if already at the minimum.
func (s *Sprite) RemoveFrame() bool {
	if len(s.frames) <= MinFrames {
		return false
	}
	s.frames = s.frames[:len(s.frames)-1]
	s.frameAttrs = s.frameAttrs[:len(s.frameAttrs)-1]
	if s.selected >= len(s.frames) {
		s.selected = len(s.frames) - 1
	}
	s.notify()
	return true
}

// InsertFrameAt inserts one empty frame (with default attributes) at index i
// and selects it, up to MaxFrames. i is clamped to [0, FrameCount()]. Returns
// false if already at the maximum.
func (s *Sprite) InsertFrameAt(i int) bool {
	if len(s.frames) >= MaxFrames {
		return false
	}
	if i < 0 {
		i = 0
	}
	if i > len(s.frames) {
		i = len(s.frames)
	}
	blank := make(Frame, s.width*s.height)
	attr := make([]byte, s.attrCols*s.attrRows)
	for j := range attr {
		attr[j] = DefaultAttr
	}

	s.frames = append(s.frames, nil)
	copy(s.frames[i+1:], s.frames[i:])
	s.frames[i] = blank

	s.frameAttrs = append(s.frameAttrs, nil)
	copy(s.frameAttrs[i+1:], s.frameAttrs[i:])
	s.frameAttrs[i] = attr

	s.selected = i
	s.notify()
	return true
}

// DeleteFrameAt removes the frame at index i, down to MinFrames. The
// selection is clamped to stay valid. Returns false if already at the
// minimum or i is out of range.
func (s *Sprite) DeleteFrameAt(i int) bool {
	if len(s.frames) <= MinFrames || i < 0 || i >= len(s.frames) {
		return false
	}
	s.frames = append(s.frames[:i], s.frames[i+1:]...)
	s.frameAttrs = append(s.frameAttrs[:i], s.frameAttrs[i+1:]...)
	switch {
	case s.selected >= len(s.frames):
		s.selected = len(s.frames) - 1
	case s.selected > i:
		s.selected--
	}
	s.notify()
	return true
}

// MoveFrame relocates the frame at index from to index to, shifting the
// frames between them, and selects it at its new position. Returns false if
// either index is out of range.
func (s *Sprite) MoveFrame(from, to int) bool {
	n := len(s.frames)
	if from < 0 || from >= n || to < 0 || to >= n {
		return false
	}
	if from == to {
		s.selected = to
		s.notify()
		return true
	}
	f := s.frames[from]
	a := s.frameAttrs[from]

	s.frames = append(s.frames[:from], s.frames[from+1:]...)
	s.frameAttrs = append(s.frameAttrs[:from], s.frameAttrs[from+1:]...)

	s.frames = append(s.frames, nil)
	copy(s.frames[to+1:], s.frames[to:])
	s.frames[to] = f

	s.frameAttrs = append(s.frameAttrs, nil)
	copy(s.frameAttrs[to+1:], s.frameAttrs[to:])
	s.frameAttrs[to] = a

	s.selected = to
	s.notify()
	return true
}

// DuplicateFrameAt inserts a copy of the frame at index i immediately after
// it, and selects the new copy, up to MaxFrames. Unlike CopyFrame/PasteFrame,
// this does not touch the user-facing clipboard. Returns false if already at
// the maximum or i is out of range.
func (s *Sprite) DuplicateFrameAt(i int) bool {
	if len(s.frames) >= MaxFrames || i < 0 || i >= len(s.frames) {
		return false
	}
	dup := s.frames[i].clone()
	dupAttr := append([]byte(nil), s.frameAttrs[i]...)

	at := i + 1
	s.frames = append(s.frames, nil)
	copy(s.frames[at+1:], s.frames[at:])
	s.frames[at] = dup

	s.frameAttrs = append(s.frameAttrs, nil)
	copy(s.frameAttrs[at+1:], s.frameAttrs[at:])
	s.frameAttrs[at] = dupAttr

	s.selected = at
	s.notify()
	return true
}

// Set forces pixel (x,y) of the selected frame to the given value. Out-of-range
// coordinates are ignored.
func (s *Sprite) Set(x, y int, on bool) {
	if x < 0 || y < 0 || x >= s.width || y >= s.height {
		return
	}
	s.frames[s.selected][y*s.width+x] = on
	s.notify()
}

// Toggle flips pixel (x,y) of the selected frame. Out-of-range coordinates are
// ignored.
func (s *Sprite) Toggle(x, y int) {
	if x < 0 || y < 0 || x >= s.width || y >= s.height {
		return
	}
	idx := y*s.width + x
	s.frames[s.selected][idx] = !s.frames[s.selected][idx]
	s.notify()
}

// CopyFrame stores a copy of the selected frame (bitmap and attributes) in the
// clipboard.
func (s *Sprite) CopyFrame() {
	s.clipboard = s.frames[s.selected].clone()
	s.clipboardAttr = append([]byte(nil), s.frameAttrs[s.selected]...)
	s.notify()
}

// PasteFrame overwrites the selected frame with the clipboard contents (bitmap
// and attributes). It is a no-op when the clipboard is empty or its size no
// longer matches the grid.
func (s *Sprite) PasteFrame() {
	if s.clipboard == nil || len(s.clipboard) != s.width*s.height {
		return
	}
	s.frames[s.selected] = s.clipboard.clone()
	if len(s.clipboardAttr) == s.attrCols*s.attrRows {
		s.frameAttrs[s.selected] = append([]byte(nil), s.clipboardAttr...)
	}
	s.notify()
}

// ClearFrame empties the selected frame and resets its attributes to default.
func (s *Sprite) ClearFrame() {
	s.frames[s.selected] = make(Frame, s.width*s.height)
	m := s.frameAttrs[s.selected]
	for i := range m {
		m[i] = DefaultAttr
	}
	s.notify()
}

// ClearFrameBitmap clears only the pixels of the selected frame, leaving its
// per-cell attributes (colour) untouched. Used where colour must be preserved.
func (s *Sprite) ClearFrameBitmap() {
	s.frames[s.selected] = make(Frame, s.width*s.height)
	s.notify()
}

// --- transforms (selected frame) -------------------------------------------

// FlipH mirrors the selected frame left-to-right, both pixels and per-cell
// attributes, so the colours follow the image.
func (s *Sprite) FlipH() {
	w, h := s.width, s.height
	f := s.frames[s.selected]
	nf := make(Frame, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			nf[y*w+(w-1-x)] = f[y*w+x]
		}
	}
	s.frames[s.selected] = nf

	cols, rows := s.attrCols, s.attrRows
	a := s.frameAttrs[s.selected]
	na := make([]byte, cols*rows)
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			na[cy*cols+(cols-1-cx)] = a[cy*cols+cx]
		}
	}
	s.frameAttrs[s.selected] = na
	s.notify()
}

// FlipV mirrors the selected frame top-to-bottom, both pixels and per-cell
// attributes.
func (s *Sprite) FlipV() {
	w, h := s.width, s.height
	f := s.frames[s.selected]
	nf := make(Frame, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			nf[(h-1-y)*w+x] = f[y*w+x]
		}
	}
	s.frames[s.selected] = nf

	cols, rows := s.attrCols, s.attrRows
	a := s.frameAttrs[s.selected]
	na := make([]byte, cols*rows)
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			na[(rows-1-cy)*cols+cx] = a[cy*cols+cx]
		}
	}
	s.frameAttrs[s.selected] = na
	s.notify()
}

// Invert flips every pixel of the selected frame. Attributes (colour) are left
// unchanged — only the bitmap is inverted.
func (s *Sprite) Invert() {
	f := s.frames[s.selected]
	for i := range f {
		f[i] = !f[i]
	}
	s.notify()
}

// Rotate90 rotates the selected frame 90 degrees clockwise.
//
// When resize is false the sprite keeps its current dimensions: the rotation is
// computed about the frame centre and any content that falls outside the current
// bounds is clipped (a true in-place rotate, ideal for square sprites).
//
// When resize is true a non-square frame's dimensions are swapped so no content
// is lost. The rotation maps the source directly into the new geometry in one
// pass (it does not resize first, which would clip before rotating).
func (s *Sprite) Rotate90(resize bool) {
	srcW, srcH := s.width, s.height
	src := s.frames[s.selected]
	srcCols, srcRows := s.attrCols, s.attrRows
	srcAttr := s.frameAttrs[s.selected]

	dstW, dstH := srcW, srcH
	if resize {
		dstW, dstH = srcH, srcW // swap
	}

	// Clockwise mapping source (x,y) -> target position. In the resize case the
	// target is srcH x srcW and the map is the exact rotation: (x,y) -> (srcH-1-y, x).
	// In the in-place case we keep the current WxH and re-centre, so a square
	// rotates exactly and a non-square is centred (overflow clipped).
	offX := (dstW - srcH) / 2 // 0 when resizing (dstW==srcH)
	offY := (dstH - srcW) / 2 // 0 when resizing (dstH==srcW)

	nf := make(Frame, dstW*dstH)
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			if !src[y*srcW+x] {
				continue
			}
			nx := (srcH - 1 - y) + offX
			ny := x + offY
			if nx >= 0 && ny >= 0 && nx < dstW && ny < dstH {
				nf[ny*dstW+nx] = true
			}
		}
	}

	// Attribute cells rotate the same way on the cell grid.
	dstCols := (dstW + Cell - 1) / Cell
	dstRows := (dstH + Cell - 1) / Cell
	na := make([]byte, dstCols*dstRows)
	for i := range na {
		na[i] = DefaultAttr
	}
	cOffX := (dstCols - srcRows) / 2
	cOffY := (dstRows - srcCols) / 2
	for cy := 0; cy < srcRows; cy++ {
		for cx := 0; cx < srcCols; cx++ {
			ncx := (srcRows - 1 - cy) + cOffX
			ncy := cx + cOffY
			if ncx >= 0 && ncy >= 0 && ncx < dstCols && ncy < dstRows {
				na[ncy*dstCols+ncx] = srcAttr[cy*srcCols+cx]
			}
		}
	}

	if resize && (dstW != srcW || dstH != srcH) {
		// Apply the new geometry to every frame (dimensions are shared). SetSize
		// preserves other frames' content top-left; the selected frame is then
		// overwritten with the rotated buffers below.
		s.SetSize(dstW, dstH)
	}
	s.frames[s.selected] = nf
	s.frameAttrs[s.selected] = na
	s.notify()
}

// Reset empties every frame, resets all attributes, and selects frame 0. The
// clipboard is preserved. Dimensions and frame count are kept.
func (s *Sprite) Reset() {
	s.resetFrames()
	s.resetAttrs()
	s.selected = 0
	s.notify()
}

// ResetAll restores the sprite to a fresh default state: default dimensions,
// DefaultFrames empty frames, all attributes default, frame 0 selected. The
// clipboard is dropped. This is the full "reset everything" a fresh document
// would have.
func (s *Sprite) ResetAll() {
	s.width, s.height = DefaultWidth, DefaultHeight
	s.attrCols = (s.width + Cell - 1) / Cell
	s.attrRows = (s.height + Cell - 1) / Cell
	s.frames = make([]Frame, DefaultFrames)
	s.frameAttrs = make([][]byte, DefaultFrames)
	s.resetFrames()
	s.resetAttrs()
	s.selected = 0
	s.name = "frame"
	s.clipboard = nil
	s.clipboardAttr = nil
	s.notify()
}

// SetSize changes the grid dimensions non-destructively: existing pixels and
// attributes are preserved within the overlapping region, new area is empty /
// default. Dimensions are snapped to whole cells and clamped to the valid range.
// The clipboard is dropped (its geometry no longer applies) and the selection is
// kept. SetSize and Resize are synonyms.
func (s *Sprite) SetSize(w, h int) {
	w, h = clampSize(w, h)
	if w == s.width && h == s.height {
		return
	}
	oldW, oldH := s.width, s.height
	oldFrames := s.frames
	oldAttrs := s.frameAttrs
	oldCols := s.attrCols

	newCols := (w + Cell - 1) / Cell
	newRows := (h + Cell - 1) / Cell
	copyW, copyH := min(oldW, w), min(oldH, h)
	copyCols, copyRows := min(oldCols, newCols), min(s.attrRows, newRows)

	s.width, s.height = w, h
	s.attrCols, s.attrRows = newCols, newRows

	for i := range s.frames {
		nf := make(Frame, w*h)
		for y := 0; y < copyH; y++ {
			for x := 0; x < copyW; x++ {
				nf[y*w+x] = oldFrames[i][y*oldW+x]
			}
		}
		s.frames[i] = nf

		na := make([]byte, newCols*newRows)
		for i := range na {
			na[i] = DefaultAttr
		}
		for cy := 0; cy < copyRows; cy++ {
			for cx := 0; cx < copyCols; cx++ {
				na[cy*newCols+cx] = oldAttrs[i][cy*oldCols+cx]
			}
		}
		s.frameAttrs[i] = na
	}

	s.clipboard = nil
	s.clipboardAttr = nil
	s.notify()
}

// Resize is a synonym for SetSize (kept for call-site compatibility).
func (s *Sprite) Resize(w, h int) { s.SetSize(w, h) }

// SetWidth and SetHeight resize a single dimension, non-destructively.
func (s *Sprite) SetWidth(w int)  { s.SetSize(w, s.height) }
func (s *Sprite) SetHeight(h int) { s.SetSize(s.width, h) }

// --- helpers ---------------------------------------------------------------

func (s *Sprite) resetFrames() {
	n := s.width * s.height
	for i := range s.frames {
		s.frames[i] = make(Frame, n)
	}
}

// resetAttrs (re)builds the attribute maps for all frames at the current size,
// every cell set to DefaultAttr. Character-cell grid is ceil(w/8) x ceil(h/8).
func (s *Sprite) resetAttrs() {
	s.attrCols = (s.width + Cell - 1) / Cell
	s.attrRows = (s.height + Cell - 1) / Cell
	n := s.attrCols * s.attrRows
	for f := range s.frameAttrs {
		m := make([]byte, n)
		for i := range m {
			m[i] = DefaultAttr
		}
		s.frameAttrs[f] = m
	}
}

// clampSize snaps w,h up to whole character cells and clamps to the valid pixel
// range [MinWidth..MaxWidth] x [MinHeight..MaxHeight].
func clampSize(w, h int) (int, int) {
	w = ((w + Cell - 1) / Cell) * Cell
	h = ((h + Cell - 1) / Cell) * Cell
	if w < MinWidth {
		w = MinWidth
	}
	if w > MaxWidth {
		w = MaxWidth
	}
	if h < MinHeight {
		h = MinHeight
	}
	if h > MaxHeight {
		h = MaxHeight
	}
	return w, h
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
