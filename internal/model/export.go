package model

import (
	"fmt"

	"github.com/ha1tch/zentools/pkg/build"
	"github.com/ha1tch/zentools/pkg/scr"
)

// Export turns sprite art into formats a real Spectrum or an emulator can load.
// Everything flows from a single composition step: one frame is laid onto a full
// 256x192 screen image (the SCR), and the tape/snapshot formats wrap those same
// 6912 bytes. The sprite is placed at the top-left of the screen; the rest of
// the screen keeps the default attribute (black paper), which a real machine
// shows as a black border around the art.

// ScreenAddr is the Spectrum display file's load address. A 6912-byte screen
// written here lands exactly on the bitmap+attribute regions, so a snapshot
// built with this origin boots showing the picture.
const ScreenAddr uint16 = 0x4000

// ExportFormat selects the byte container produced by ExportScreen.
type ExportFormat int

const (
	FormatSCR       ExportFormat = iota // raw 6912-byte .scr
	FormatTAP                           // .tap CODE block at 0x4000
	FormatTAPLoader                     // auto-running .tap (BASIC loader + CODE)
	FormatTZX                           // .tzx wrapping the TAP
	FormatSNA                           // .sna snapshot (boots showing the screen)
	FormatZ80                           // .z80 snapshot (boots showing the screen)
)

// composeScreen lays frame f onto a fresh 256x192 screen and returns the encoded
// 6912-byte SCR image. The frame is placed at the top-left; a frame larger than
// the screen in either axis is an error (the editor caps sprites at 256x192, so
// this only guards against misuse).
func (s *Sprite) composeScreen(f int) ([]byte, error) {
	if f < 0 || f >= len(s.frames) {
		return nil, fmt.Errorf("export: frame %d out of range", f)
	}
	if s.width > scr.Width || s.height > scr.Height {
		return nil, fmt.Errorf("export: sprite %dx%d exceeds the %dx%d screen",
			s.width, s.height, scr.Width, scr.Height)
	}
	screen := &scr.Screen{}
	asset := s.frameAsset(f, "art")
	if err := scr.Paste(screen, &asset, 0, 0, scr.PasteCOPY); err != nil {
		return nil, fmt.Errorf("export: %w", err)
	}
	return scr.Encode(screen), nil
}

// ExportScreen renders frame f of the sprite to the requested format and returns
// the file bytes. name is embedded in tape/snapshot containers (clamped to the
// tape's 10-char limit by zentools); pass a neutral title, not personal data.
func (s *Sprite) ExportScreen(f int, format ExportFormat, name string) ([]byte, error) {
	img, err := s.composeScreen(f)
	if err != nil {
		return nil, err
	}
	if format == FormatSCR {
		return img, nil
	}

	req := build.Request{
		Name:   name,
		Code:   img,
		Origin: ScreenAddr,
		Start:  ScreenAddr, // snapshots: PC at the screen; with no code to run the
		SP:     build.DefaultSP,
		Model:  build.Model48K,
	}
	switch format {
	case FormatTAP:
		return build.EncodeTAP(req), nil
	case FormatTAPLoader:
		return build.EncodeTAPWithLoader(req)
	case FormatTZX:
		return build.EncodeTZX(req)
	case FormatSNA:
		return build.EncodeSNA(req)
	case FormatZ80:
		return build.EncodeZ80(req)
	default:
		return nil, fmt.Errorf("export: unknown format %d", format)
	}
}

// ExportExt returns the conventional file extension (without a dot) for a format.
func ExportExt(format ExportFormat) string {
	switch format {
	case FormatSCR:
		return "scr"
	case FormatTAP, FormatTAPLoader:
		return "tap"
	case FormatTZX:
		return "tzx"
	case FormatSNA:
		return "sna"
	case FormatZ80:
		return "z80"
	default:
		return "bin"
	}
}
