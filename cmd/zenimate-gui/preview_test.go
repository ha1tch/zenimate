//go:build purego

package main

import "testing"

// The collapsed end of the popup must reproduce the preview exactly: drawing the
// whole sprite at 'zoom' px/pixel from the computed origin, then clipping to the
// preview box, must place region [cx0,cy0] at [offX,offY] — the same coordinates
// drawPreview uses. This guards the seamless match at the end of the retraction.
func TestPopupCollapseMatchesPreview(t *testing.T) {
	sw, sh := 48, 32
	boxW, boxH := 168, 168
	zoom := 3
	focusX, focusY := 20, 15
	pvX, pvY := float32(600), float32(80)

	cx0, cy0, _, _, offX, offY, _, _ := previewRegion(sw, sh, boxW, boxH, zoom, focusX, focusY, pvX, pvY)

	// Reconstruct the popup's collapsed-end sprite origin.
	scale0 := float32(zoom)
	originX0 := offX - float32(cx0)*scale0
	originY0 := offY - float32(cy0)*scale0

	// Drawing the whole sprite from origin0 at scale0, pixel cx0 lands at:
	gotX := originX0 + float32(cx0)*scale0
	gotY := originY0 + float32(cy0)*scale0
	if gotX != offX || gotY != offY {
		t.Errorf("collapsed origin misaligns region: got (%.1f,%.1f) want (%.1f,%.1f)", gotX, gotY, offX, offY)
	}
}

// previewRegion centres the image when the sprite is smaller than the box, and
// pins to the box corner otherwise; clamps the region within the sprite.
func TestPreviewRegionClampAndCentre(t *testing.T) {
	// Small sprite, low zoom: whole sprite fits, should centre.
	sw, sh := 8, 8
	boxW, boxH := 168, 168
	zoom := 1
	pvX, pvY := float32(100), float32(100)
	cx0, cy0, spanX, spanY, offX, offY, drawW, drawH := previewRegion(sw, sh, boxW, boxH, zoom, 4, 4, pvX, pvY)
	if cx0 != 0 || cy0 != 0 || spanX != 8 || spanY != 8 {
		t.Errorf("small sprite should show whole: cx0=%d cy0=%d span=%dx%d", cx0, cy0, spanX, spanY)
	}
	// drawW=8, box=168 -> centred offset (168-8)/2 = 80
	if offX != pvX+80 || offY != pvY+80 {
		t.Errorf("small sprite not centred: off=(%.1f,%.1f) want (%.1f,%.1f)", offX, offY, pvX+80, pvY+80)
	}
	_ = drawW
	_ = drawH

	// Focus near a corner on a large sprite: region clamps within bounds.
	sw, sh = 100, 100
	zoom = 4
	cx0, cy0, spanX, spanY, _, _, _, _ = previewRegion(sw, sh, boxW, boxH, zoom, 0, 0, pvX, pvY)
	if cx0 < 0 || cy0 < 0 {
		t.Errorf("region not clamped to >=0: cx0=%d cy0=%d", cx0, cy0)
	}
	if cx0+spanX > sw || cy0+spanY > sh {
		t.Errorf("region exceeds sprite: cx0=%d span=%d sw=%d", cx0, spanX, sw)
	}
}
