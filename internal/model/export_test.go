package model

import (
	"testing"

	"github.com/ha1tch/zentools/pkg/scr"
)

func TestExportSCRSizeAndContent(t *testing.T) {
	s := New(16, 16)
	s.Set(0, 0, true)   // top-left pixel
	s.Set(15, 15, true) // a pixel inside the second cell
	img, err := s.ExportScreen(0, FormatSCR, "art")
	if err != nil {
		t.Fatal(err)
	}
	if len(img) != scr.FileLen {
		t.Fatalf("SCR length = %d, want %d", len(img), scr.FileLen)
	}
	// Decode the SCR back via zentools and confirm the pixels survived the
	// screen-address scramble at their true screen positions (top-left placement).
	back, err := scr.Decode(img)
	if err != nil {
		t.Fatal(err)
	}
	if !back.Ink[0][0] {
		t.Error("pixel (0,0) missing in exported screen")
	}
	if !back.Ink[15][15] {
		t.Error("pixel (15,15) missing in exported screen")
	}
	// A pixel the sprite never set must be clear.
	if back.Ink[100][100] {
		t.Error("unexpected pixel set far outside the sprite")
	}
}

func TestExportAttributesPlaced(t *testing.T) {
	s := New(16, 16)
	s.SetAttrCell(1, 0, 0x46) // ink 6, bright, at cell (1,0)
	img, _ := s.ExportScreen(0, FormatSCR, "art")
	back, _ := scr.Decode(img)
	got := back.Attr[0][1].Byte()
	if got != 0x46 {
		t.Errorf("screen attr at cell (1,0) = 0x%02X, want 0x46", got)
	}
}

func TestExportFormatsProduceOutput(t *testing.T) {
	s := New(32, 24)
	s.Set(5, 5, true)
	formats := []struct {
		f       ExportFormat
		minSize int
		ext     string
	}{
		{FormatSCR, scr.FileLen, "scr"},
		{FormatTAP, scr.FileLen, "tap"},
		{FormatTAPLoader, scr.FileLen, "tap"},
		{FormatTZX, 100, "tzx"},
		{FormatSNA, 49152, "sna"},
		{FormatZ80, 100, "z80"},
	}
	for _, c := range formats {
		data, err := s.ExportScreen(0, c.f, "art")
		if err != nil {
			t.Errorf("format %d: %v", c.f, err)
			continue
		}
		if len(data) < c.minSize {
			t.Errorf("format %d: output %d bytes, want >= %d", c.f, len(data), c.minSize)
		}
		if ExportExt(c.f) != c.ext {
			t.Errorf("format %d ext = %q, want %q", c.f, ExportExt(c.f), c.ext)
		}
	}
}

func TestExportSnapshotContainsScreen(t *testing.T) {
	// A .sna boots showing the art: the 6912 screen bytes must appear at the
	// display-file offset in the 48K snapshot (27-byte SNA header + RAM from
	// 0x4000). We just confirm the snapshot embeds our exact SCR bytes.
	s := New(16, 16)
	s.Set(3, 4, true)
	img, _ := s.ExportScreen(0, FormatSCR, "art")
	sna, err := s.ExportScreen(0, FormatSNA, "art")
	if err != nil {
		t.Fatal(err)
	}
	const snaHeader = 27
	if len(sna) < snaHeader+scr.FileLen {
		t.Fatalf("sna too small: %d", len(sna))
	}
	disp := sna[snaHeader : snaHeader+scr.FileLen]
	for i := range img {
		if disp[i] != img[i] {
			t.Fatalf("snapshot display file diverges from SCR at byte %d", i)
		}
	}
}

func TestExportFrameOutOfRange(t *testing.T) {
	s := New(16, 16)
	if _, err := s.ExportScreen(99, FormatSCR, "art"); err == nil {
		t.Error("expected error for out-of-range frame")
	}
}
