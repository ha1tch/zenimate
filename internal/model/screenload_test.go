package model

import "testing"

func TestLoadSCRRoundTrip(t *testing.T) {
	// Make a sprite, export to SCR, load it back as a screen sprite.
	s := New(16, 16)
	s.Set(0, 0, true)
	s.Set(15, 8, true)
	s.SetAttrCell(0, 0, 0x45)
	img, err := s.ExportScreen(0, FormatSCR, "x")
	if err != nil {
		t.Fatal(err)
	}
	back, err := LoadSCR(img)
	if err != nil {
		t.Fatal(err)
	}
	if back.Width() != 256 || back.Height() != 192 {
		t.Fatalf("loaded screen is %dx%d, want 256x192", back.Width(), back.Height())
	}
	if !back.Frame(0)[0] {
		t.Error("pixel (0,0) lost")
	}
	if !back.Frame(0)[8*256+15] {
		t.Error("pixel (15,8) lost")
	}
	if back.AttrCellFrame(0, 0, 0) != 0x45 {
		t.Errorf("attr (0,0) = 0x%02X, want 0x45", back.AttrCellFrame(0, 0, 0))
	}
}

func TestLoadScreenFromTAPandSNA(t *testing.T) {
	s := New(24, 16)
	s.Set(2, 3, true)
	tapImg, _ := s.ExportScreen(0, FormatTAP, "scr")
	snaImg, _ := s.ExportScreen(0, FormatSNA, "scr")

	tapSprite, err := LoadScreenFromTAP(tapImg)
	if err != nil {
		t.Fatalf("tap: %v", err)
	}
	if !tapSprite.Frame(0)[3*256+2] {
		t.Error("tap screen lost pixel (2,3)")
	}
	snaSprite, err := LoadScreenFromSnapshot(snaImg, "sna")
	if err != nil {
		t.Fatalf("sna: %v", err)
	}
	if !snaSprite.Frame(0)[3*256+2] {
		t.Error("sna screen lost pixel (2,3)")
	}
}

func TestLoadScreenFromZ80(t *testing.T) {
	s := New(16, 16)
	s.Set(5, 5, true)
	z80Img, _ := s.ExportScreen(0, FormatZ80, "scr")
	sprite, err := LoadScreenFromSnapshot(z80Img, "z80")
	if err != nil {
		t.Fatalf("z80: %v", err)
	}
	if !sprite.Frame(0)[5*256+5] {
		t.Error("z80 screen lost pixel (5,5)")
	}
}

func TestLoadScreenFromTZX(t *testing.T) {
	s := New(16, 16)
	s.Set(7, 7, true)
	tzxImg, _ := s.ExportScreen(0, FormatTZX, "scr")
	sprite, err := LoadScreenFromTZX(tzxImg)
	if err != nil {
		t.Fatalf("tzx: %v", err)
	}
	if !sprite.Frame(0)[7*256+7] {
		t.Error("tzx screen lost pixel (7,7)")
	}
}

func TestLoadByExtension(t *testing.T) {
	s := New(16, 16)
	s.Set(1, 1, true)
	zcut, _ := s.MarshalZCUT()
	scrImg, _ := s.ExportScreen(0, FormatSCR, "x")

	if sp, err := LoadByExtension(".zcut", zcut); err != nil || sp.Width() != 16 {
		t.Errorf("zcut dispatch failed: %v", err)
	}
	if sp, err := LoadByExtension("SCR", scrImg); err != nil || sp.Width() != 256 {
		t.Errorf("scr dispatch (uppercase, no dot) failed: %v", err)
	}
	if _, err := LoadByExtension(".xyz", nil); err == nil {
		t.Error("unknown extension should error")
	}
}

func TestLoadScreenFromTAPNoScreen(t *testing.T) {
	// A TAP whose only block is small (not 6912) should report no screen.
	s := New(16, 16)
	// A zcut wrapped as a small TAP is not screen-sized.
	if _, err := LoadScreenFromTAP([]byte{0x02, 0x00, 0xFF, 0xFF}); err == nil {
		t.Error("expected 'no screen' error on a screenless tap")
	}
	_ = s
}
