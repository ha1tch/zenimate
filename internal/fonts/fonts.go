// Package fonts embeds the bundled BDF bitmap fonts and decodes them through
// pkg/bdf. The fonts are compiled into the binary so the GUI has no runtime
// file dependency.
//
// Bundled faces:
//
//	Sinclair    8x8    — the ZX Spectrum 48K ROM character set (period-correct;
//	                     the Text tool's default)
//	Cozette     6x13   — a modern, legible programming bitmap face (fallback / UI)
//	ToolIcons   32x32  — zenimate's own tool-palette pictograms, two sizes in one
//	                     face (32px at U+E000, 24px at U+E100, Private Use Area).
//	                     Not a third-party face — no separate licence file, unlike
//	                     the others below.
//	TomThumb    4x6    — extremely small; the only genuinely practical choice
//	                     for stamping text on the smallest sprite sizes
//	Spleen5x8   5x8    — small, clean, modern bitmap face
//	Creep       7x11   — a distinctive, decorative face — not another plain
//	                     monospace terminal font
//	HaxorMedium 8x14   — a stylised "hacker" face, thematically fitting for a
//	                     retro pixel tool
//
// TomThumb, Spleen5x8, Creep, and HaxorMedium were selected from Horatio's own
// BDF font catalogue (github.com/ha1tch/bdf-fonts), chosen for being genuinely
// usable at ZX Spectrum sprite scale (16x16 up to 32x24) and for stylistic
// variety, rather than picking several near-identical sizes of the same
// monospace family.
//
// Licences: Sinclair carries an Amstrad copyright notice (see LICENSE-
// Sinclair.txt); Cozette and TomThumb are MIT (LICENSE-Cozette.txt; TomThumb's
// MIT notice is embedded in its own BDF COPYRIGHT property); Spleen5x8, Creep,
// and HaxorMedium are Public Domain per the bdf-fonts catalogue's README,
// which states all fonts in that repository are public domain except for a
// short, explicitly named list (tom-thumb, cozette, sinclair, tahoma), none
// of which these three are. ToolIcons is zenimate's own asset.
package fonts

import (
	"bytes"
	_ "embed"
	"io"

	"github.com/ha1tch/zenimate/pkg/bdf"
)

//go:embed sinclair.bdf
var sinclairBDF []byte

//go:embed cozette.bdf
var cozetteBDF []byte

//go:embed icons.bdf
var iconsBDF []byte

//go:embed tom-thumb.bdf
var tomThumbBDF []byte

//go:embed spleen-5x8.bdf
var spleen5x8BDF []byte

//go:embed creep.bdf
var creepBDF []byte

//go:embed haxor-medium-12.bdf
var haxorMediumBDF []byte

// Sinclair decodes the bundled ZX Spectrum 8x8 face.
func Sinclair() (*bdf.Font, error) {
	return bdf.Parse(bytesReader(sinclairBDF))
}

// Cozette decodes the bundled Cozette 6x13 face.
func Cozette() (*bdf.Font, error) {
	return bdf.Parse(bytesReader(cozetteBDF))
}

// ToolIcons decodes the bundled tool-palette icon face. Look up icons by
// codepoint: U+E000 + index for the 32px set, U+E100 + index for the 24px
// set, in the fixed order defined by cmd/zenimate-gui's tool list.
func ToolIcons() (*bdf.Font, error) {
	return bdf.Parse(bytesReader(iconsBDF))
}

// TomThumb decodes the bundled 4x6 face — the smallest built-in option.
func TomThumb() (*bdf.Font, error) {
	return bdf.Parse(bytesReader(tomThumbBDF))
}

// Spleen5x8 decodes the bundled Spleen 5x8 face.
func Spleen5x8() (*bdf.Font, error) {
	return bdf.Parse(bytesReader(spleen5x8BDF))
}

// Creep decodes the bundled Creep 7x11 decorative face.
func Creep() (*bdf.Font, error) {
	return bdf.Parse(bytesReader(creepBDF))
}

// HaxorMedium decodes the bundled Haxor Medium 8x14 face.
func HaxorMedium() (*bdf.Font, error) {
	return bdf.Parse(bytesReader(haxorMediumBDF))
}

// SinclairBytes / CozetteBytes / ToolIconsBytes / TomThumbBytes /
// Spleen5x8Bytes / CreepBytes / HaxorMediumBytes expose the raw embedded BDF
// data, for callers that want to parse or persist the source themselves.
func SinclairBytes() []byte    { return sinclairBDF }
func CozetteBytes() []byte     { return cozetteBDF }
func ToolIconsBytes() []byte   { return iconsBDF }
func TomThumbBytes() []byte    { return tomThumbBDF }
func Spleen5x8Bytes() []byte   { return spleen5x8BDF }
func CreepBytes() []byte       { return creepBDF }
func HaxorMediumBytes() []byte { return haxorMediumBDF }

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }
