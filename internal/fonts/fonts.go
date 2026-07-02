// Package fonts embeds the bundled BDF bitmap fonts and decodes them through
// pkg/bdf. The fonts are compiled into the binary so the GUI has no runtime
// file dependency.
//
// Bundled faces:
//
//	Sinclair  8x8   — the ZX Spectrum 48K ROM character set (period-correct)
//	Cozette   6x13  — a modern, legible programming bitmap face (fallback / UI)
//
// Licences for both faces live alongside the .bdf files and are reproduced in
// the project's THIRD-PARTY notices. Neither face is authored by this project.
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

// Sinclair decodes the bundled ZX Spectrum 8x8 face.
func Sinclair() (*bdf.Font, error) {
	return bdf.Parse(bytesReader(sinclairBDF))
}

// Cozette decodes the bundled Cozette 6x13 face.
func Cozette() (*bdf.Font, error) {
	return bdf.Parse(bytesReader(cozetteBDF))
}

// SinclairBytes / CozetteBytes expose the raw embedded BDF data, for callers
// that want to parse or persist the source themselves.
func SinclairBytes() []byte { return sinclairBDF }
func CozetteBytes() []byte  { return cozetteBDF }

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }
