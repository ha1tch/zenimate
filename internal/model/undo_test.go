package model

import (
	"sync"
	"testing"
)

func TestCheckpointUndoRestoresPriorState(t *testing.T) {
	s := New(16, 16)
	s.Set(0, 0, true)
	s.Checkpoint()
	s.Set(1, 1, true) // this edit should be undone

	if !s.Undo() {
		t.Fatal("Undo should have succeeded")
	}
	if s.At(1, 1) {
		t.Error("(1,1) should be gone after undo")
	}
	if !s.At(0, 0) {
		t.Error("(0,0) should still be set — it predates the checkpoint")
	}
}

func TestUndoWithNothingToUndo(t *testing.T) {
	s := New(16, 16)
	if s.Undo() {
		t.Error("Undo should fail with nothing on the stack")
	}
	if s.CanUndo() {
		t.Error("CanUndo should be false with nothing on the stack")
	}
}

func TestRedoRestoresUndoneState(t *testing.T) {
	s := New(16, 16)
	s.Checkpoint()
	s.Set(2, 2, true)
	s.Undo()
	if s.At(2, 2) {
		t.Fatal("setup: (2,2) should be gone after undo")
	}
	if !s.Redo() {
		t.Fatal("Redo should have succeeded")
	}
	if !s.At(2, 2) {
		t.Error("(2,2) should be back after redo")
	}
}

func TestRedoWithNothingToRedo(t *testing.T) {
	s := New(16, 16)
	if s.Redo() {
		t.Error("Redo should fail with nothing on the redo stack")
	}
	if s.CanRedo() {
		t.Error("CanRedo should be false with nothing on the redo stack")
	}
}

func TestCheckpointAfterUndoClearsRedo(t *testing.T) {
	s := New(16, 16)
	s.Checkpoint()
	s.Set(3, 3, true)
	s.Undo()
	if !s.CanRedo() {
		t.Fatal("setup: redo should be available after undo")
	}
	s.Set(4, 4, true)
	s.Checkpoint() // a fresh action after undo invalidates the redo timeline
	if s.CanRedo() {
		t.Error("a new checkpoint after undo should clear the redo stack")
	}
}

// TestUndoRedoRoundTripThroughSeveralStates exercises snapshot/restore
// independence specifically: A -> checkpoint -> B -> checkpoint -> C, then
// undo, undo, redo. If restore() aliased a stack entry's slices instead of
// cloning them, this sequence would corrupt an earlier or later state.
func TestUndoRedoRoundTripThroughSeveralStates(t *testing.T) {
	s := New(16, 16)
	s.Set(0, 0, true) // state A: (0,0)
	s.Checkpoint()

	s.Set(1, 1, true) // state B: (0,0),(1,1)
	s.Checkpoint()

	s.Set(2, 2, true) // state C: (0,0),(1,1),(2,2) — not yet checkpointed

	s.Undo() // -> B
	assertPixels(t, s, "after 1st undo (want B)", map[[2]int]bool{
		{0, 0}: true, {1, 1}: true, {2, 2}: false,
	})

	s.Undo() // -> A
	assertPixels(t, s, "after 2nd undo (want A)", map[[2]int]bool{
		{0, 0}: true, {1, 1}: false, {2, 2}: false,
	})

	s.Redo() // -> B
	assertPixels(t, s, "after redo (want B)", map[[2]int]bool{
		{0, 0}: true, {1, 1}: true, {2, 2}: false,
	})

	if !s.CanRedo() {
		t.Error("C should still be redo-able")
	}
	s.Redo() // -> C
	assertPixels(t, s, "after 2nd redo (want C)", map[[2]int]bool{
		{0, 0}: true, {1, 1}: true, {2, 2}: true,
	})
}

func assertPixels(t *testing.T, s *Sprite, label string, want map[[2]int]bool) {
	t.Helper()
	for xy, wantOn := range want {
		if got := s.At(xy[0], xy[1]); got != wantOn {
			t.Errorf("%s: (%d,%d) = %v, want %v", label, xy[0], xy[1], got, wantOn)
		}
	}
}

func TestUndoRollingWindowCapsAt100Levels(t *testing.T) {
	s := New(16, 16)
	for i := 0; i < 105; i++ {
		s.Checkpoint()
	}
	if len(s.undoStack) != maxUndoLevels {
		t.Fatalf("undoStack length = %d, want %d", len(s.undoStack), maxUndoLevels)
	}
	count := 0
	for s.Undo() {
		count++
	}
	if count != maxUndoLevels {
		t.Errorf("could undo %d times, want exactly %d", count, maxUndoLevels)
	}
}

func TestUndoRedoDoNotTouchClipboard(t *testing.T) {
	s := New(16, 16)
	s.Set(0, 0, true)
	s.CopyFrame()
	if !s.HasClipboard() {
		t.Fatal("setup: clipboard should be populated")
	}
	s.Checkpoint()
	s.Set(1, 1, true)

	s.Undo()
	if !s.HasClipboard() {
		t.Error("undo should not clear the clipboard")
	}
	s.Redo()
	if !s.HasClipboard() {
		t.Error("redo should not clear the clipboard")
	}
}

func TestCheckpointTagsEveryTenthAsKeyframe(t *testing.T) {
	s := New(16, 16)
	for i := 0; i < 25; i++ {
		s.Checkpoint()
	}
	// All 25 are still within the 100-level window (none evicted), so the
	// tagging should be exact: indices 0, 10, 20.
	for i, snap := range s.undoStack {
		want := i%keyframeInterval == 0
		if snap.isKeyframe != want {
			t.Errorf("undoStack[%d].isKeyframe = %v, want %v", i, snap.isKeyframe, want)
		}
	}
}

// TestUndoRedoConcurrentAccessIsRaceFree exists to be run under `go test
// -race`: it does not assert on a final state (concurrent Checkpoint/Undo/
// Redo from multiple goroutines has no single deterministic outcome — real
// usage is single-threaded), only that the stack bookkeeping itself never
// races.
func TestUndoRedoConcurrentAccessIsRaceFree(t *testing.T) {
	s := New(16, 16)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				s.Checkpoint()
				s.CanUndo()
				s.Undo()
				s.CanRedo()
				s.Redo()
			}
		}()
	}
	wg.Wait()
}
