package model

import (
	"bytes"
	"fmt"

	"github.com/ha1tch/zentools/pkg/scr"
	"github.com/ha1tch/zentools/pkg/snapshot"
	"github.com/ha1tch/zentools/pkg/tap"
	"github.com/ha1tch/zentools/pkg/tzx"
)

// LoadSCR builds a full 256x192 sprite from a raw 6912-byte ZX screen image.
// The bitmap and per-cell attributes are taken directly from the screen, so the
// result is the picture as a real Spectrum would display it, ready to edit.
func LoadSCR(data []byte) (*Sprite, error) {
	screen, err := scr.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("scr: %w", err)
	}
	return spriteFromScreen(screen), nil
}

// spriteFromScreen converts a decoded scr.Screen into a single-frame 256x192
// sprite, copying pixels and attributes cell-for-cell.
func spriteFromScreen(screen *scr.Screen) *Sprite {
	w, h := scr.Width, scr.Height
	s := &Sprite{
		width:    w,
		height:   h,
		name:     "screen",
		attrCols: w / Cell,
		attrRows: h / Cell,
		selected: 0,
	}
	fr := newFrame(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fr.Set(x, y, w, screen.Ink[y][x])
		}
	}
	s.frames = []Frame{fr}

	am := make([]byte, s.attrCols*s.attrRows)
	for cy := 0; cy < s.attrRows; cy++ {
		for cx := 0; cx < s.attrCols; cx++ {
			am[cy*s.attrCols+cx] = screen.Attr[cy][cx].Byte()
		}
	}
	s.frameAttrs = [][]byte{am}
	return s
}

// LoadScreenFromTAP extracts a screen from a TAP image: the first CODE data
// block whose payload is exactly a screen (6912 bytes). Many tapes load a
// loading screen this way. Returns an error if no screen-sized block is present.
func LoadScreenFromTAP(data []byte) (*Sprite, error) {
	blocks, err := tap.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("tap: %w", err)
	}
	for _, b := range blocks {
		if !b.IsHeader && b.Flag == 0xFF && len(b.Data) == scr.FileLen {
			return LoadSCR(b.Data)
		}
	}
	return nil, fmt.Errorf("tap: no screen-sized (6912-byte) block found")
}

// LoadScreenFromTZX extracts a screen from a TZX image by reassembling its
// standard-speed data blocks into a TAP image and reusing the TAP path.
func LoadScreenFromTZX(data []byte) (*Sprite, error) {
	tzxBlocks, err := tzx.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("tzx: %w", err)
	}
	var tapImage []byte
	for _, b := range tzxBlocks {
		if b.ID != 0x10 || len(b.Data) == 0 {
			continue
		}
		tapImage = appendU16LE(tapImage, uint16(len(b.Data)))
		tapImage = append(tapImage, b.Data...)
	}
	return LoadScreenFromTAP(tapImage)
}

// LoadScreenFromSnapshot extracts the display file (the screen) from a .sna or
// .z80 snapshot. The display file is the first 6912 bytes of bank 5 (the memory
// at 0x4000), which both snapshot decoders expose via the machine state.
func LoadScreenFromSnapshot(data []byte, kind string) (*Sprite, error) {
	var st *snapshot.MachineState
	var err error
	switch kind {
	case "sna":
		st, err = decodeSNAany(data)
	case "z80":
		st, err = snapshot.DecodeZ80(data)
	default:
		return nil, fmt.Errorf("snapshot: unknown kind %q", kind)
	}
	if err != nil {
		return nil, fmt.Errorf("snapshot: %w", err)
	}
	disp, err := displayFile(st)
	if err != nil {
		return nil, err
	}
	return LoadSCR(disp)
}

// decodeSNAany decodes either a 48K or 128K .sna by length.
func decodeSNAany(data []byte) (*snapshot.MachineState, error) {
	const sna48Len = 49179
	if len(data) == sna48Len {
		return snapshot.DecodeSNA(data)
	}
	return snapshot.DecodeSNA128(data)
}

// displayFile returns the 6912-byte screen from a machine state: bank 5 holds
// the standard display file (mapped at 0x4000 on a 48K layout).
func displayFile(st *snapshot.MachineState) ([]byte, error) {
	bank := st.Memory.RAM[5]
	if len(bank) < scr.FileLen {
		return nil, fmt.Errorf("snapshot: bank 5 too small for a display file")
	}
	out := make([]byte, scr.FileLen)
	copy(out, bank[:scr.FileLen])
	return out, nil
}

// appendU16LE appends v as two little-endian bytes (a TAP block length prefix).
func appendU16LE(b []byte, v uint16) []byte {
	return append(b, byte(v), byte(v>>8))
}

// LoadByExtension picks the right loader for a file based on its extension
// (case-insensitive) and returns the resulting sprite. Supported: .zani/.zan and
// the .zcut alias (a single animated sprite), .scr (raw screen), and
// .tap/.tzx/.sna/.z80 (a screen extracted from the container, when one is
// present). Bundles (.zbun/.zbu) and images are handled elsewhere, since they
// need a picker or a fit choice. An unknown extension is an error.
func LoadByExtension(ext string, data []byte) (*Sprite, error) {
	e := normaliseExt(ext)
	if IsAnimationExt(e) {
		// .zani / .zan / .zcut are all a single ZCUT-encoded animation.
		return LoadZCUT(data)
	}
	switch e {
	case "scr":
		return LoadSCR(data)
	case "tap":
		return LoadScreenFromTAP(data)
	case "tzx":
		return LoadScreenFromTZX(data)
	case "sna":
		return LoadScreenFromSnapshot(data, "sna")
	case "z80":
		return LoadScreenFromSnapshot(data, "z80")
	default:
		return nil, fmt.Errorf("unsupported file type %q", ext)
	}
}

// LoadableExtensions lists the single-file extensions LoadByExtension accepts,
// for use in file-dialog filters and drag-and-drop messaging. Bundles and images
// are offered separately by the callers that know how to prompt for them.
func LoadableExtensions() []string {
	return []string{ExtAnimation, ExtAnimation83, ExtAnimationOld, "scr", "tap", "tzx", "sna", "z80"}
}

// normaliseExt lowercases an extension and strips a leading dot.
func normaliseExt(ext string) string {
	out := make([]byte, 0, len(ext))
	for i := 0; i < len(ext); i++ {
		c := ext[i]
		if i == 0 && c == '.' {
			continue
		}
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out = append(out, c)
	}
	return string(out)
}

// FitMode selects how an imported image is brought to the 256x192 screen before
// reduction to Spectrum colours. It mirrors scr.ResizeMode but is exposed here
// so the UI can offer the choice without importing zentools directly.
type FitMode int

const (
	// FitStretch scales to fill the screen, ignoring aspect ratio.
	FitStretch FitMode = iota
	// FitBestFit scales to fit within the screen preserving aspect ratio,
	// centring and padding the border (letterbox/pillarbox).
	FitBestFit
	// FitCentre does not scale: it centres the image, cropping overflow and
	// padding shortfall.
	FitCentre
)

// resizeMode maps a FitMode to the zentools ResizeMode.
func (m FitMode) resizeMode() scr.ResizeMode {
	switch m {
	case FitStretch:
		return scr.ResizeStretch
	case FitCentre:
		return scr.ResizeCentre
	default:
		return scr.ResizeBestFit
	}
}

// FitModeName returns a short human label for a fit mode.
func FitModeName(m FitMode) string {
	switch m {
	case FitStretch:
		return "Stretch"
	case FitCentre:
		return "Centre"
	default:
		return "Best fit"
	}
}

// LoadImage imports a JPEG, PNG or GIF as a 256x192 screen sprite. The source is
// first fitted to the screen with the chosen strategy (padding with black where
// a strategy leaves a border), then reduced to Spectrum ink/paper/bright per 8x8
// cell by zentools. The result is a single-frame, fully editable screen.
func LoadImage(data []byte, mode FitMode) (*Sprite, error) {
	img, err := scr.DecodeImage(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("image: %w", err)
	}
	fitted, err := scr.Fit(img, mode.resizeMode(), blackFill{})
	if err != nil {
		return nil, fmt.Errorf("image: %w", err)
	}
	screen, err := scr.FromImage(fitted)
	if err != nil {
		return nil, fmt.Errorf("image: %w", err)
	}
	return spriteFromScreen(screen), nil
}

// blackFill is the padding colour used for letterbox/pillarbox borders.
type blackFill struct{}

func (blackFill) RGBA() (r, g, b, a uint32) { return 0, 0, 0, 0xFFFF }

// IsImageExt reports whether ext (with or without a leading dot, any case) is an
// importable raster image format.
func IsImageExt(ext string) bool {
	switch normaliseExt(ext) {
	case "jpg", "jpeg", "png", "gif":
		return true
	}
	return false
}
