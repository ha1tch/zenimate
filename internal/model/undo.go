package model

import "sync"

// maxUndoLevels bounds the undo stack: a rolling window, oldest entry
// evicted once exceeded.
const maxUndoLevels = 100

// keyframeInterval tags every Nth checkpoint as a keyframe. At current
// sprite-size limits every checkpoint is already a full snapshot (worst
// case ~108KB packed; 100 levels is ~10.6MB), so this tag carries no
// different storage or restore logic — it's bookkeeping only, for a
// possible future history browser. As the rolling window evicts from the
// front, the remaining entries' keyframe spacing can drift from an exact
// multiple of ten; harmless, since nothing depends on the spacing being
// exact.
const keyframeInterval = 10

// spriteSnapshot is a full point-in-time copy of everything Undo/Redo
// restore: dimensions, every frame and its attributes, the selection, and
// the name. The clipboard is deliberately excluded — it's tool state, not
// document content, the same convention most editors use.
type spriteSnapshot struct {
	width, height      int
	attrCols, attrRows int
	frames             []Frame
	frameAttrs         [][]byte
	selected           int
	name               string
	isKeyframe         bool
}

// snapshot deep-copies the sprite's current document state. Every frame and
// attribute map is cloned, so the snapshot is fully independent of s: later
// edits to s cannot corrupt a snapshot already pushed onto a stack.
func (s *Sprite) snapshot() spriteSnapshot {
	frames := make([]Frame, len(s.frames))
	for i, f := range s.frames {
		frames[i] = f.clone()
	}
	attrs := make([][]byte, len(s.frameAttrs))
	for i, a := range s.frameAttrs {
		attrs[i] = append([]byte(nil), a...)
	}
	return spriteSnapshot{
		width: s.width, height: s.height,
		attrCols: s.attrCols, attrRows: s.attrRows,
		frames: frames, frameAttrs: attrs,
		selected: s.selected, name: s.name,
	}
}

// restore applies snap to the live sprite, deep-copying out of the snapshot
// so the stack's copy remains untouched by subsequent edits (the same
// independence guarantee as snapshot, in the other direction — a snapshot
// pulled off a stack may be restored again later, via a further Undo/Redo,
// so it must not be handed out as live mutable state).
func (s *Sprite) restore(snap spriteSnapshot) {
	frames := make([]Frame, len(snap.frames))
	for i, f := range snap.frames {
		frames[i] = f.clone()
	}
	attrs := make([][]byte, len(snap.frameAttrs))
	for i, a := range snap.frameAttrs {
		attrs[i] = append([]byte(nil), a...)
	}
	s.width, s.height = snap.width, snap.height
	s.attrCols, s.attrRows = snap.attrCols, snap.attrRows
	s.frames = frames
	s.frameAttrs = attrs
	s.selected = snap.selected
	s.name = snap.name
}

// Checkpoint pushes the sprite's current state onto the undo stack and
// clears the redo stack (a fresh action after an Undo invalidates the
// redone-forward timeline — standard editor semantics). The undo stack is a
// rolling window of maxUndoLevels: once full, the oldest entry is evicted.
//
// Checkpoint does not decide *when* to snapshot — that's the caller's job.
// Call it once per discrete user action (a button press, or the start of a
// paint-drag gesture), never once per primitive mutation (Set/Toggle), or a
// single brushstroke would exhaust the whole window one pixel at a time.
//
// Checkpoint's own bookkeeping (the stack slices) is protected by a mutex,
// so concurrent calls to Checkpoint/Undo/Redo/CanUndo/CanRedo are safe with
// respect to each other. This does not make the rest of Sprite's mutating
// methods (Set, Toggle, and so on) safe for concurrent use — they are not
// lock-protected, matching the GUI's current single-threaded design.
func (s *Sprite) Checkpoint() {
	s.undoMu.Lock()
	defer s.undoMu.Unlock()

	snap := s.snapshot()
	snap.isKeyframe = len(s.undoStack)%keyframeInterval == 0
	s.undoStack = append(s.undoStack, snap)
	if len(s.undoStack) > maxUndoLevels {
		s.undoStack = s.undoStack[1:]
	}
	s.redoStack = s.redoStack[:0]
}

// Undo restores the most recently checkpointed state, pushing the state it
// replaces onto the redo stack. Returns false if there is nothing to undo.
func (s *Sprite) Undo() bool {
	s.undoMu.Lock()
	defer s.undoMu.Unlock()

	if len(s.undoStack) == 0 {
		return false
	}
	prev := s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]

	s.redoStack = append(s.redoStack, s.snapshot())
	s.restore(prev)
	s.notify()
	return true
}

// Redo re-applies the most recently undone state, pushing the state it
// replaces back onto the undo stack. Returns false if there is nothing to
// redo.
func (s *Sprite) Redo() bool {
	s.undoMu.Lock()
	defer s.undoMu.Unlock()

	if len(s.redoStack) == 0 {
		return false
	}
	next := s.redoStack[len(s.redoStack)-1]
	s.redoStack = s.redoStack[:len(s.redoStack)-1]

	s.undoStack = append(s.undoStack, s.snapshot())
	s.restore(next)
	s.notify()
	return true
}

// CanUndo and CanRedo report whether Undo/Redo would currently do anything —
// for the GUI to grey out the corresponding controls.
func (s *Sprite) CanUndo() bool {
	s.undoMu.Lock()
	defer s.undoMu.Unlock()
	return len(s.undoStack) > 0
}

func (s *Sprite) CanRedo() bool {
	s.undoMu.Lock()
	defer s.undoMu.Unlock()
	return len(s.redoStack) > 0
}

// undoState holds the mutex-protected undo/redo stacks. Embedded into Sprite
// via its own type so the fields group together and the zero value (empty
// stacks, unlocked mutex) is immediately usable — Sprite needs no
// constructor changes.
type undoState struct {
	undoMu    sync.Mutex
	undoStack []spriteSnapshot
	redoStack []spriteSnapshot
}
