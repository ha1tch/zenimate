package model

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func makePNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if (x+y)%2 == 0 {
				img.Set(x, y, color.White)
			} else {
				img.Set(x, y, color.Black)
			}
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestLoadImageFitModes(t *testing.T) {
	data := makePNG(100, 40) // odd size, exercises every fit mode
	for _, m := range []FitMode{FitStretch, FitBestFit, FitCentre} {
		s, err := LoadImage(data, m)
		if err != nil {
			t.Fatalf("%s: %v", FitModeName(m), err)
		}
		if s.Width() != 256 || s.Height() != 192 {
			t.Errorf("%s: got %dx%d, want 256x192", FitModeName(m), s.Width(), s.Height())
		}
	}
}

func TestImageNotInLoadByExtension(t *testing.T) {
	// Images require a fit-strategy choice, so they are intentionally NOT handled
	// by the silent LoadByExtension dispatcher; the GUI routes them via LoadImage
	// after asking the user. LoadByExtension should report them as unsupported.
	data := makePNG(64, 64)
	if _, err := LoadByExtension(".png", data); err == nil {
		t.Error("png should not load silently via LoadByExtension (needs a fit choice)")
	}
}

func TestIsImageExt(t *testing.T) {
	for _, e := range []string{"jpg", ".JPEG", "png", ".gif"} {
		if !IsImageExt(e) {
			t.Errorf("%q should be an image ext", e)
		}
	}
	if IsImageExt("zcut") {
		t.Error("zcut is not an image ext")
	}
}
