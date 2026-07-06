package model

// selectionState is the sprite's current rectangular selection, if any, and
// (while a move or paste is in progress) a floating buffer of lifted pixel
// content not yet written back into the frame.
//
// Only bitmap content is carried by selection/move/copy/paste — never
// per-cell attributes. ZX attributes apply to whole 8x8 cells, but a
// selection's pixel bounds generally don't align to cell boundaries; lifting
// or moving attributes at arbitrary pixel bounds would risk corrupting
// unselected pixels sharing an edge cell with the selection. This matches
// the existing convention that plain painting only ever touches the bitmap,
// never attributes (only Ctrl/Option-paint stamps those) — v1 selection
// operations follow the same rule, not a new one.
type selectionState struct {
	active     bool
	x, y, w, h int // bounds, pixel coordinates, clamped to the frame

	floating  bool
	floatBits Frame // lifted bitmap, valid only while floating, sized w x h

	clipboard   Frame // copied bitmap, nil if none, sized clipW x clipH
	clipW       int
	clipH       int
	clipOriginX int // where the clipboard content was copied from, for paste-in-place
	clipOriginY int
}

// SetSelection defines or replaces the active selection, clamped to the
// frame and normalised so w,h are always positive regardless of drag
// direction. Any pending floating content is committed first — starting a
// new selection elsewhere is what finalises a move or paste in most paint
// tools, matching that convention here.
func (s *Sprite) SetSelection(x0, y0, x1, y1 int) {
	s.CommitFloatingSelection()

	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	x1++ // treat the drag endpoints as inclusive pixel coordinates
	y1++

	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > s.width {
		x1 = s.width
	}
	if y1 > s.height {
		y1 = s.height
	}
	w, h := x1-x0, y1-y0
	if w <= 0 || h <= 0 {
		s.selection = selectionState{}
		return
	}
	s.selection = selectionState{active: true, x: x0, y: y0, w: w, h: h}
}

// HasSelection reports whether a selection is currently active.
func (s *Sprite) HasSelection() bool { return s.selection.active }

// Selection returns the active selection's bounds. ok is false if there is
// no active selection, in which case the bounds are zero.
func (s *Sprite) Selection() (x, y, w, h int, ok bool) {
	if !s.selection.active {
		return 0, 0, 0, 0, false
	}
	return s.selection.x, s.selection.y, s.selection.w, s.selection.h, true
}

// IsFloating reports whether the selection currently holds lifted content
// not yet written back into the frame (a move in progress, or a paste that
// hasn't been committed).
func (s *Sprite) IsFloating() bool { return s.selection.floating }

// ClearSelection commits any pending floating content, then deselects.
func (s *Sprite) ClearSelection() {
	s.CommitFloatingSelection()
	s.selection = selectionState{}
}

// LiftSelection begins a move: it captures the active selection's current
// bitmap into the floating buffer. If duplicate is false (a plain move), the
// original area is cleared to off — the classic "cut" half of drag-to-move.
// If duplicate is true (Alt/Option-drag), the original is left untouched, so
// the drag produces a copy rather than relocating the source. A no-op if
// there is no active selection or a lift is already in progress.
func (s *Sprite) LiftSelection(duplicate bool) {
	sel := s.selection
	if !sel.active || sel.floating {
		return
	}
	fr := s.frames[s.selected]
	lifted := newFrame(sel.w, sel.h)
	for ly := 0; ly < sel.h; ly++ {
		for lx := 0; lx < sel.w; lx++ {
			if fr.At(sel.x+lx, sel.y+ly, s.width) {
				lifted.Set(lx, ly, sel.w, true)
			}
		}
	}
	if !duplicate {
		for ly := 0; ly < sel.h; ly++ {
			for lx := 0; lx < sel.w; lx++ {
				fr.Set(sel.x+lx, sel.y+ly, s.width, false)
			}
		}
	}
	s.selection.floating = true
	s.selection.floatBits = lifted
}

// MoveFloatingTo repositions the floating selection's target bounds, without
// writing anything into the frame yet — cheap enough to call every frame
// during a drag. Clamped so the floating content stays fully on the sprite.
// A no-op if nothing is floating.
func (s *Sprite) MoveFloatingTo(x, y int) {
	if !s.selection.floating {
		return
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+s.selection.w > s.width {
		x = s.width - s.selection.w
	}
	if y+s.selection.h > s.height {
		y = s.height - s.selection.h
	}
	s.selection.x, s.selection.y = x, y
}

// CommitFloatingSelection writes the floating buffer into the frame at its
// current position, replacing whatever was there — a move or paste
// overwrites the destination with the selection's exact content (on and
// off), it doesn't merge with what was underneath. Matches Paint's
// bitmap-only semantics, so it never touches attributes. A no-op if nothing
// is floating.
// CommitFloatingSelection writes the floating buffer into the frame at its
// current position, ORed onto whatever is already there — only pixels the
// buffer itself has set are written; positions where the buffer is unset
// are left untouched, not cleared. This matches Photoshop's actual
// behaviour: "off" in our bitmap-only selection model is the equivalent of
// transparency in a layer with an alpha channel, and dropping content onto
// a layer never lets its transparent areas erase what's underneath.
//
// An earlier version of this method did a full overwrite (explicitly
// clearing destination bits wherever the buffer was unset), on the
// reasoning that a move should replace the destination exactly rather than
// merge with it. That reasoning was wrong — Photoshop's real convention is
// the OR described above, confirmed directly rather than assumed — so this
// reverts that change. See TestCommitFloatingSelectionOrsOntoDestination
// for the specific regression this now guards against.
func (s *Sprite) CommitFloatingSelection() {
	sel := s.selection
	if !sel.floating {
		return
	}
	fr := s.frames[s.selected]
	for ly := 0; ly < sel.h; ly++ {
		for lx := 0; lx < sel.w; lx++ {
			if sel.floatBits.At(lx, ly, sel.w) {
				fr.Set(sel.x+lx, sel.y+ly, s.width, true)
			}
		}
	}
	s.selection.floating = false
	s.selection.floatBits = nil
}

// HasSelectionClipboard reports whether a selection has been copied.
func (s *Sprite) HasSelectionClipboard() bool { return s.selection.clipboard != nil }

// CopySelectionToClipboard copies the active selection's current bitmap
// content into the selection clipboard — separate from the whole-frame
// clipboard used by CopyFrame/PasteFrame, since a selection is generally a
// different size than the frame. Reads directly from the frame, not from any
// floating buffer, so copying never needs a lift in progress. A no-op if
// there is no active selection.
func (s *Sprite) CopySelectionToClipboard() {
	x, y, w, h, ok := s.Selection()
	if !ok {
		return
	}
	fr := s.frames[s.selected]
	buf := newFrame(w, h)
	for ly := 0; ly < h; ly++ {
		for lx := 0; lx < w; lx++ {
			if fr.At(x+lx, y+ly, s.width) {
				buf.Set(lx, ly, w, true)
			}
		}
	}
	s.selection.clipboard = buf
	s.selection.clipW, s.selection.clipH = w, h
	s.selection.clipOriginX, s.selection.clipOriginY = x, y
}

// PasteSelectionClipboard creates a new selection at the clipboard content's
// original copy position, holding it as floating (uncommitted) content — the
// pasted pixels stay selected and movable until a commit (ClearSelection,
// starting a new selection, or CommitFloatingSelection directly). A no-op if
// the clipboard is empty.
func (s *Sprite) PasteSelectionClipboard() {
	if s.selection.clipboard == nil {
		return
	}
	s.CommitFloatingSelection()
	x, y := s.selection.clipOriginX, s.selection.clipOriginY
	w, h := s.selection.clipW, s.selection.clipH
	if x+w > s.width {
		x = s.width - w
	}
	if y+h > s.height {
		y = s.height - h
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	s.selection = selectionState{
		active: true, x: x, y: y, w: w, h: h,
		floating: true, floatBits: s.selection.clipboard.clone(),
		clipboard: s.selection.clipboard, clipW: w, clipH: h,
		clipOriginX: s.selection.clipOriginX, clipOriginY: s.selection.clipOriginY,
	}
}

// ClearSelectionArea clears the active selection's bitmap to off, directly
// in the frame — the Delete/Backspace action, and independent of any
// float/commit machinery. A no-op if there is no active selection.
func (s *Sprite) ClearSelectionArea() {
	x, y, w, h, ok := s.Selection()
	if !ok {
		return
	}
	fr := s.frames[s.selected]
	for ly := 0; ly < h; ly++ {
		for lx := 0; lx < w; lx++ {
			fr.Set(x+lx, y+ly, s.width, false)
		}
	}
}

// FlipSelectionH mirrors the active selection's content horizontally, in
// place (dimensions unchanged). Lifts first if not already floating, then
// commits immediately — a button click is an instant transform, not a drag
// gesture, so there's no reason to leave it floating afterwards. A no-op if
// there is no active selection.
func (s *Sprite) FlipSelectionH() {
	if !s.selection.active {
		return
	}
	if !s.selection.floating {
		s.LiftSelection(false)
	}
	w, h := s.selection.w, s.selection.h
	nf := newFrame(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if s.selection.floatBits.At(x, y, w) {
				nf.Set(w-1-x, y, w, true)
			}
		}
	}
	s.selection.floatBits = nf
	s.CommitFloatingSelection()
}

// FlipSelectionV mirrors the active selection's content vertically, in
// place. Same lift/commit-immediately behaviour as FlipSelectionH.
func (s *Sprite) FlipSelectionV() {
	if !s.selection.active {
		return
	}
	if !s.selection.floating {
		s.LiftSelection(false)
	}
	w, h := s.selection.w, s.selection.h
	nf := newFrame(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if s.selection.floatBits.At(x, y, w) {
				nf.Set(x, h-1-y, w, true)
			}
		}
	}
	s.selection.floatBits = nf
	s.CommitFloatingSelection()
}

// RotateSelection90 rotates the active selection's content 90 degrees
// clockwise. Unlike Flip, this necessarily changes the selection's own
// dimensions (a WxH region becomes HxW) — the new bounds are centred on the
// same point the old ones were, clamped to stay on the sprite, matching how
// a transform box behaves in most paint tools rather than anchoring on a
// corner. Lifts first if not already floating, then commits immediately.
// A no-op if there is no active selection.
func (s *Sprite) RotateSelection90() {
	if !s.selection.active {
		return
	}
	if !s.selection.floating {
		s.LiftSelection(false)
	}
	w, h := s.selection.w, s.selection.h
	nf := newFrame(h, w) // dimensions swap
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if s.selection.floatBits.At(x, y, w) {
				nf.Set(h-1-y, x, h, true)
			}
		}
	}

	cx := s.selection.x + w/2
	cy := s.selection.y + h/2
	newX := cx - h/2
	newY := cy - w/2
	if newX < 0 {
		newX = 0
	}
	if newY < 0 {
		newY = 0
	}
	if newX+h > s.width {
		newX = s.width - h
	}
	if newY+w > s.height {
		newY = s.height - w
	}

	s.selection.floatBits = nf
	s.selection.w, s.selection.h = h, w
	s.selection.x, s.selection.y = newX, newY
	s.CommitFloatingSelection()
}

// FloatingAt reports the floating buffer's bit at local coordinate (lx,ly)
// — 0,0 is the selection's own top-left corner, not a frame coordinate. For
// rendering a live preview of lifted content that hasn't been committed to
// the frame yet. Returns false if nothing is floating or the coordinate is
// outside the selection's own bounds.
func (s *Sprite) FloatingAt(lx, ly int) bool {
	sel := s.selection
	if !sel.floating || lx < 0 || ly < 0 || lx >= sel.w || ly >= sel.h {
		return false
	}
	return sel.floatBits.At(lx, ly, sel.w)
}
