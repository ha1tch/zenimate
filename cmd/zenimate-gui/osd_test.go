//go:build purego

package main

import "testing"

func TestOSDTriggersOnSeqChange(t *testing.T) {
	o := newOSD()
	// No animation until a new sequence with non-empty text.
	o.update(0.016, 0, "", 50, 8)
	if o.active {
		t.Fatal("OSD should not be active before any message")
	}
	o.update(0.016, 1, "FRAME ADDED", 50, 8)
	if !o.active || o.text != "FRAME ADDED" {
		t.Fatalf("OSD should activate with the message; active=%v text=%q", o.active, o.text)
	}
}

func TestOSDRisesAndFadesOut(t *testing.T) {
	o := newOSD()
	o.update(0.016, 1, "HELLO", 40, 8)
	start := o.travel
	// Advance enough frames to cover the full hold+fade travel, then it must
	// deactivate.
	for i := 0; i < 400 && o.active; i++ {
		o.update(0.016, 1, "HELLO", 40, 8)
	}
	if o.active {
		t.Fatal("OSD should deactivate after travelling its full distance")
	}
	if o.travel <= start {
		t.Fatal("OSD should have risen")
	}
}

func TestOSDReTriggersOnSameText(t *testing.T) {
	o := newOSD()
	o.update(0.016, 1, "COPY", 30, 8)
	// Let it rise partway.
	for i := 0; i < 10; i++ {
		o.update(0.016, 1, "COPY", 30, 8)
	}
	mid := o.travel
	// Same text, new sequence: must restart from 0.
	o.update(0.016, 2, "COPY", 30, 8)
	if o.travel >= mid {
		t.Fatalf("a fresh sequence should restart travel; got %v (was %v)", o.travel, mid)
	}
}

func TestMagicalShapesWellFormed(t *testing.T) {
	for i, sh := range magicalShapes {
		w := len(sh[0])
		any := false
		for _, row := range sh {
			if len(row) != w {
				t.Errorf("shape %d has ragged rows", i)
			}
			for _, ch := range row {
				if ch == '#' {
					any = true
				} else if ch != '.' {
					t.Errorf("shape %d has invalid char %q", i, ch)
				}
			}
		}
		if !any {
			t.Errorf("shape %d has no set pixels", i)
		}
	}
}

func TestSparksScatterAroundBox(t *testing.T) {
	o := newOSD()
	o.update(0.016, 1, "TEST", 40, 8)
	// Force a spark layout.
	o.layoutSparks(40, 8)
	if len(o.sparks) != osdSparkCount {
		t.Fatalf("expected %d sparks, got %d", osdSparkCount, len(o.sparks))
	}
}

func TestOSDHoldsThenFades(t *testing.T) {
	o := newOSD()
	o.active = true
	o.travel = 0
	if o.alpha() != 1 {
		t.Errorf("alpha at travel 0 = %v, want 1", o.alpha())
	}
	o.travel = float32(osdHold) // end of the opaque hold zone
	if o.alpha() != 1 {
		t.Errorf("alpha at end of hold = %v, want 1 (still fully opaque)", o.alpha())
	}
	o.travel = float32(osdHold) + float32(osdFade)/2 // halfway through the fade
	if a := o.alpha(); a < 0.4 || a > 0.6 {
		t.Errorf("alpha at mid-fade = %v, want ~0.5", a)
	}
	o.travel = float32(osdHold) + float32(osdFade) // fully faded
	if o.alpha() != 0 {
		t.Errorf("alpha at end of fade = %v, want 0", o.alpha())
	}
}
