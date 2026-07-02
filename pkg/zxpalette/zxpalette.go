// Package zxpalette holds the ZX Spectrum colour palette and attribute-byte
// encoding, cloned from the ha1tch/zenzx emulator so the editor's colours match
// the emulator exactly. It is framework-neutral (image/color), so any frontend
// can consume it.
//
// The Spectrum has 8 base colours, each with a normal and a "bright" variant —
// 16 entries. An attribute byte packs: bits 0-2 ink, bits 3-5 paper, bit 6
// bright, bit 7 flash. A colour's palette index is colour | (bright << 3).
package zxpalette

import "image/color"

// Colour indices (0-7), in Spectrum order.
const (
	Black = iota
	Blue
	Red
	Magenta
	Green
	Cyan
	Yellow
	White
)

// Names of the eight base colours, by index.
var Names = [8]string{"Black", "Blue", "Red", "Magenta", "Green", "Cyan", "Yellow", "White"}

// RGBA is the 16-entry palette (8 normal + 8 bright), matching zenzx's
// ZXPaletteRGBA. The dim level is 0xC8 and bright is 0xFF.
var RGBA = [16]color.NRGBA{
	{0x00, 0x00, 0x00, 0xff}, // Black
	{0x00, 0x00, 0xc8, 0xff}, // Blue
	{0xc8, 0x00, 0x00, 0xff}, // Red
	{0xc8, 0x00, 0xc8, 0xff}, // Magenta
	{0x00, 0xc8, 0x00, 0xff}, // Green
	{0x00, 0xc8, 0xc8, 0xff}, // Cyan
	{0xc8, 0xc8, 0x00, 0xff}, // Yellow
	{0xc8, 0xc8, 0xc8, 0xff}, // White
	{0x00, 0x00, 0x00, 0xff}, // Bright Black
	{0x00, 0x00, 0xff, 0xff}, // Bright Blue
	{0xff, 0x00, 0x00, 0xff}, // Bright Red
	{0xff, 0x00, 0xff, 0xff}, // Bright Magenta
	{0x00, 0xff, 0x00, 0xff}, // Bright Green
	{0x00, 0xff, 0xff, 0xff}, // Bright Cyan
	{0xff, 0xff, 0x00, 0xff}, // Bright Yellow
	{0xff, 0xff, 0xff, 0xff}, // Bright White
}

// Index returns the palette index for a base colour (0-7) and bright flag.
func Index(colour int, bright bool) int {
	idx := colour & 0x07
	if bright {
		idx |= 0x08
	}
	return idx
}

// Colour returns the RGBA for a base colour (0-7) and bright flag.
func Colour(colour int, bright bool) color.NRGBA {
	return RGBA[Index(colour, bright)]
}

// xterm256 maps each of the 16 palette entries to its nearest xterm-256 colour
// index (from the 6x6x6 cube and greyscale ramp), computed by least-squares RGB
// distance. Terminals that support xterm-256 but not 24-bit truecolour (e.g.
// macOS Terminal.app) render these faithfully, whereas 38;2;R;G;B sequences
// degrade. The bright hues land on exact cube matches; the normal (0xC8) hues
// map to the 215-level cube, the closest available.
var xterm256 = [16]int{
	16,  // Black
	20,  // Blue
	160, // Red
	164, // Magenta
	40,  // Green
	44,  // Cyan
	184, // Yellow
	251, // White
	16,  // Bright Black
	21,  // Bright Blue
	196, // Bright Red
	201, // Bright Magenta
	46,  // Bright Green
	51,  // Bright Cyan
	226, // Bright Yellow
	231, // Bright White
}

// Xterm256 returns the nearest xterm-256 colour index for a base colour (0-7)
// and bright flag, for rendering the ZX palette in terminals that support
// indexed but not 24-bit colour.
func Xterm256(colour int, bright bool) int {
	return xterm256[Index(colour, bright)]
}

// Xterm256Attr returns the xterm-256 indices for an attribute byte's ink and
// paper at its brightness, a convenience for terminal renderers.
func Xterm256Attr(a byte) (ink, paper int) {
	b := Bright(a)
	return Xterm256(Ink(a), b), Xterm256(Paper(a), b)
}

// Attribute byte helpers (bits: 0-2 ink, 3-5 paper, 6 bright, 7 flash).

// Attr packs ink, paper (0-7), bright and flash into an attribute byte.
func Attr(ink, paper int, bright, flash bool) byte {
	a := byte(ink&0x07) | byte((paper&0x07)<<3)
	if bright {
		a |= 1 << 6
	}
	if flash {
		a |= 1 << 7
	}
	return a
}

// Ink, Paper, Bright, Flash decode an attribute byte.
func Ink(a byte) int     { return int(a & 0x07) }
func Paper(a byte) int   { return int((a >> 3) & 0x07) }
func Bright(a byte) bool { return (a>>6)&1 == 1 }
func Flash(a byte) bool  { return (a>>7)&1 == 1 }
