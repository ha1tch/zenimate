package model

import "testing"

func TestSetSelectionNormalizesReversedDrag(t *testing.T) {
	s := New(16, 16)
	// Drag from bottom-right to top-left — bounds should still come out
	// correctly ordered, not negative or empty.
	s.SetSelection(10, 10, 4, 4)
	x, y, w, h, ok := s.Selection()
	if !ok || x != 4 || y != 4 || w != 7 || h != 7 {
		t.Errorf("Selection() = (%d,%d,%d,%d,%v), want (4,4,7,7,true)", x, y, w, h, ok)
	}
}

func TestSetSelectionClampsToFrame(t *testing.T) {
	s := New(16, 16)
	s.SetSelection(-5, -5, 20, 20)
	x, y, w, h, ok := s.Selection()
	if !ok || x != 0 || y != 0 || w != 16 || h != 16 {
		t.Errorf("Selection() = (%d,%d,%d,%d,%v), want clamped to (0,0,16,16,true)", x, y, w, h, ok)
	}
}

func TestSetSelectionZeroSizeClearsSelection(t *testing.T) {
	s := New(16, 16)
	s.SetSelection(5, 5, 10, 10)
	s.SetSelection(-3, -3, -1, -1) // fully off-frame -> no valid selection
	if s.HasSelection() {
		t.Error("expected no selection after a fully out-of-bounds SetSelection")
	}
}

func TestLiftSelectionMoveClearsOriginal(t *testing.T) {
	s := New(16, 16)
	for y := 2; y < 6; y++ {
		for x := 2; x < 6; x++ {
			s.frames[0].Set(x, y, 16, true)
		}
	}
	s.SetSelection(2, 2, 5, 5) // inclusive endpoints -> bounds (2,2,4,4)
	s.LiftSelection(false)     // plain move: clear original

	if !s.IsFloating() {
		t.Fatal("expected IsFloating() after LiftSelection")
	}
	for y := 2; y < 6; y++ {
		for x := 2; x < 6; x++ {
			if s.frames[0].At(x, y, 16) {
				t.Errorf("pixel (%d,%d) should be cleared after a move-lift", x, y)
			}
		}
	}
}

func TestLiftSelectionDuplicateLeavesOriginal(t *testing.T) {
	s := New(16, 16)
	for y := 2; y < 6; y++ {
		for x := 2; x < 6; x++ {
			s.frames[0].Set(x, y, 16, true)
		}
	}
	s.SetSelection(2, 2, 5, 5)
	s.LiftSelection(true) // Alt/Option-drag: duplicate, leave original

	for y := 2; y < 6; y++ {
		for x := 2; x < 6; x++ {
			if !s.frames[0].At(x, y, 16) {
				t.Errorf("pixel (%d,%d) should remain set after a duplicate-lift", x, y)
			}
		}
	}
}

func TestCommitFloatingSelectionOrsOntoDestination(t *testing.T) {
	s := New(16, 16)
	// Source: a single "on" pixel at (2,2) within a 2x2 selection — so the
	// lifted buffer has (0,0)=true, (1,0)/(0,1)/(1,1)=false.
	s.frames[0].Set(2, 2, 16, true)
	s.SetSelection(2, 2, 3, 3) // bounds (2,2,2,2)
	s.LiftSelection(false)

	// Destination already has an "on" pixel at (11,11) — the buffer-relative
	// (1,1) position, which is FALSE in the lifted content. Committing
	// should OR the buffer onto the destination: this stray pixel should
	// survive untouched, matching Photoshop's actual behaviour where "off"
	// in our bitmap-only model is the equivalent of transparency in a layer
	// with an alpha channel — dropping content never lets its transparent
	// areas erase what's underneath. (An earlier version of this test
	// asserted the opposite — that the stray pixel should be cleared — on
	// the mistaken assumption that a move should replace the destination
	// exactly. That assumption was wrong; this replaces that test.)
	s.frames[0].Set(11, 11, 16, true)
	s.MoveFloatingTo(10, 10)
	s.CommitFloatingSelection()

	if !s.frames[0].At(10, 10, 16) {
		t.Error("the moved 'on' pixel should be set at the new position")
	}
	if !s.frames[0].At(11, 11, 16) {
		t.Error("stray pre-existing content at a FALSE position in the moved buffer should survive an OR-merge commit, not be cleared")
	}
	if s.IsFloating() {
		t.Error("IsFloating() should be false after commit")
	}
}

func TestMoveFloatingToClampsToFrame(t *testing.T) {
	s := New(16, 16)
	s.SetSelection(0, 0, 3, 3) // bounds (0,0,4,4)
	s.LiftSelection(false)
	s.MoveFloatingTo(100, 100) // way off-frame
	x, y, _, _, _ := s.Selection()
	if x != 12 || y != 12 { // 16-4=12, clamped so the 4x4 buffer stays fully on-frame
		t.Errorf("MoveFloatingTo did not clamp: got (%d,%d), want (12,12)", x, y)
	}
}

func TestCopyAndPasteInPlace(t *testing.T) {
	s := New(16, 16)
	s.frames[0].Set(3, 3, 16, true)
	s.SetSelection(2, 2, 4, 4) // bounds (2,2,3,3), covers (3,3)
	s.CopySelectionToClipboard()

	if !s.HasSelectionClipboard() {
		t.Fatal("expected a selection clipboard after CopySelectionToClipboard")
	}

	// Destroy the original, then paste — should recreate it at the same
	// position, as a floating (not yet committed) selection.
	s.ClearSelectionArea()
	if s.frames[0].At(3, 3, 16) {
		t.Fatal("test setup: ClearSelectionArea should have cleared (3,3)")
	}

	s.PasteSelectionClipboard()
	if !s.IsFloating() {
		t.Error("paste should leave the pasted content floating, not committed")
	}
	x, y, w, h, ok := s.Selection()
	if !ok || x != 2 || y != 2 || w != 3 || h != 3 {
		t.Errorf("pasted selection bounds = (%d,%d,%d,%d,%v), want (2,2,3,3,true) — paste-in-place", x, y, w, h, ok)
	}
	// Not committed yet, so the frame itself shouldn't show it.
	if s.frames[0].At(3, 3, 16) {
		t.Error("pasted content should not be written into the frame before commit")
	}

	s.CommitFloatingSelection()
	if !s.frames[0].At(3, 3, 16) {
		t.Error("pasted content should appear in the frame after commit")
	}
}

func TestClearSelectionCommitsFloatingFirst(t *testing.T) {
	s := New(16, 16)
	s.frames[0].Set(2, 2, 16, true)
	s.SetSelection(2, 2, 2, 2)
	s.LiftSelection(false)
	s.MoveFloatingTo(8, 8)

	s.ClearSelection() // should commit the pending move before deselecting

	if s.HasSelection() {
		t.Error("expected no selection after ClearSelection")
	}
	if !s.frames[0].At(8, 8, 16) {
		t.Error("ClearSelection should have committed the pending move to (8,8) first")
	}
}

func TestSetSelectionCommitsPreviousFloatingFirst(t *testing.T) {
	s := New(16, 16)
	s.frames[0].Set(2, 2, 16, true)
	s.SetSelection(2, 2, 2, 2)
	s.LiftSelection(false)
	s.MoveFloatingTo(8, 8)

	// Starting a brand new selection elsewhere should commit the pending
	// move first, matching "clicking to make a new selection finalises the
	// old one" in most paint tools.
	s.SetSelection(0, 0, 1, 1)

	if !s.frames[0].At(8, 8, 16) {
		t.Error("starting a new selection should have committed the pending move to (8,8) first")
	}
}

func TestClearSelectionAreaNoOpWithoutSelection(t *testing.T) {
	s := New(16, 16)
	s.frames[0].Set(5, 5, 16, true)
	s.ClearSelectionArea() // no active selection: must not panic or affect anything
	if !s.frames[0].At(5, 5, 16) {
		t.Error("ClearSelectionArea with no active selection should be a no-op")
	}
}

func TestLiftSelectionNoOpWithoutSelection(t *testing.T) {
	s := New(16, 16)
	s.LiftSelection(false) // must not panic
	if s.IsFloating() {
		t.Error("LiftSelection with no active selection should not start floating")
	}
}

func TestFlipSelectionHMirrorsWithinBounds(t *testing.T) {
	s := New(16, 16)
	s.frames[0].Set(2, 2, 16, true) // local (0,0) within a (2,2,4,4) selection
	s.SetSelection(2, 2, 5, 5)      // bounds (2,2,4,4)
	s.FlipSelectionH()

	if s.frames[0].At(2, 2, 16) {
		t.Error("original position should be cleared after flip")
	}
	if !s.frames[0].At(5, 2, 16) { // local (3,0) = w-1-0
		t.Error("expected the pixel mirrored to local (3,0) = absolute (5,2)")
	}
	x, y, w, h, ok := s.Selection()
	if !ok || x != 2 || y != 2 || w != 4 || h != 4 {
		t.Errorf("selection bounds should be unchanged by a flip: got (%d,%d,%d,%d,%v)", x, y, w, h, ok)
	}
}

func TestFlipSelectionVMirrorsWithinBounds(t *testing.T) {
	s := New(16, 16)
	s.frames[0].Set(2, 2, 16, true)
	s.SetSelection(2, 2, 5, 5)
	s.FlipSelectionV()

	if s.frames[0].At(2, 2, 16) {
		t.Error("original position should be cleared after flip")
	}
	if !s.frames[0].At(2, 5, 16) { // local (0,3) = h-1-0
		t.Error("expected the pixel mirrored to local (0,3) = absolute (2,5)")
	}
}

func TestFlipSelectionNoOpWithoutSelection(t *testing.T) {
	s := New(16, 16)
	s.frames[0].Set(5, 5, 16, true)
	s.FlipSelectionH()
	if !s.frames[0].At(5, 5, 16) {
		t.Error("FlipSelectionH with no active selection should be a no-op")
	}
}

func TestRotateSelection90SquareStaysInPlace(t *testing.T) {
	s := New(16, 16)
	s.frames[0].Set(2, 2, 16, true) // local (0,0) of a square 4x4 selection
	s.SetSelection(2, 2, 5, 5)      // bounds (2,2,4,4)
	s.RotateSelection90()

	x, y, w, h, ok := s.Selection()
	if !ok || x != 2 || y != 2 || w != 4 || h != 4 {
		t.Errorf("a square selection's bounds should stay in place after rotate: got (%d,%d,%d,%d,%v)", x, y, w, h, ok)
	}
	// Clockwise: local (0,0) -> (h-1-0, 0) = (3,0) -> absolute (5,2).
	if !s.frames[0].At(5, 2, 16) {
		t.Error("expected the rotated pixel at absolute (5,2)")
	}
}

func TestRotateSelection90SwapsDimensionsAndRecentres(t *testing.T) {
	s := New(16, 16)
	s.SetSelection(0, 0, 5, 1) // bounds (0,0,6,2): a wide, short rectangle
	s.RotateSelection90()

	x, y, w, h, ok := s.Selection()
	if !ok {
		t.Fatal("expected a selection to remain active after rotate")
	}
	if w != 2 || h != 6 {
		t.Errorf("dimensions should swap: got w=%d h=%d, want w=2 h=6", w, h)
	}
	// Centre of the original (0,0,6,2) is (3,1); centred 2x6 bounds would be
	// x=3-1=2, y=1-3=-2, clamped to y=0.
	if x != 2 || y != 0 {
		t.Errorf("recentred position = (%d,%d), want (2,0) (clamped)", x, y)
	}
}

func TestRotateSelectionNoOpWithoutSelection(t *testing.T) {
	s := New(16, 16)
	s.RotateSelection90() // must not panic
	if s.HasSelection() {
		t.Error("RotateSelection90 with no active selection should not create one")
	}
}
