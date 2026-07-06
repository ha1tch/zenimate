package zenui

// previewRegion computes what the pane shows: the image region [rcx,rcy]
// sized [rspanX,rspanY], drawn at zoom px per image pixel and placed at
// screen [rox,roy] sized [rdrawW,rdrawH] (centred in the box when smaller
// than it). Shared by the normal box draw and the popup's collapsed end, so
// the two match exactly — a direct, faithful port of the geometry this
// widget was extracted from.
func previewRegion(iw, ih, boxW, boxH, zoom, focusX, focusY int, boxX, boxY float32) (rcx, rcy, rspanX, rspanY int, rox, roy, rdrawW, rdrawH float32) {
	if zoom < 1 {
		zoom = 1
	}
	spanX := boxW / zoom
	spanY := boxH / zoom
	if spanX > iw {
		spanX = iw
	}
	if spanY > ih {
		spanY = ih
	}
	if spanX < 1 {
		spanX = 1
	}
	if spanY < 1 {
		spanY = 1
	}
	cx0 := focusX - spanX/2
	cy0 := focusY - spanY/2
	if cx0 < 0 {
		cx0 = 0
	}
	if cy0 < 0 {
		cy0 = 0
	}
	if cx0+spanX > iw {
		cx0 = iw - spanX
	}
	if cy0+spanY > ih {
		cy0 = ih - spanY
	}
	drawW := float32(spanX * zoom)
	drawH := float32(spanY * zoom)
	offX := boxX
	offY := boxY
	if drawW < float32(boxW) {
		offX += (float32(boxW) - drawW) / 2
	}
	if drawH < float32(boxH) {
		offY += (float32(boxH) - drawH) / 2
	}
	return cx0, cy0, spanX, spanY, offX, offY, drawW, drawH
}

// drawSampledRegion draws image region [cx0,cy0] sized [spanX,spanY] at
// screen position [bx,by] sized [bw,bh]: one Region call for the whole
// rectangle, then one FillRect per visible image pixel from the result.
// Each pixel's rect is derived from the difference between consecutive
// rounded boundaries (this pixel's right edge is computed identically to
// the next pixel's left edge), not a fixed width applied uniformly — the
// latter leaves sub-pixel rounding gaps whenever bw/spanX isn't an exact
// integer, which shows through as gridlines. Worst during the popup's
// animated scale transition, where bw/bh change continuously every frame,
// which is why this appeared intermittently rather than consistently.
func (p *PreviewPane) drawSampledRegion(r Renderer, cx0, cy0, spanX, spanY int, bx, by, bw, bh float32) {
	colours := p.cfg.Source.Region(cx0, cy0, spanX, spanY)
	pw := bw / float32(spanX)
	ph := bh / float32(spanY)
	for j := 0; j < spanY; j++ {
		y0 := int(by + float32(j)*ph)
		y1 := int(by + float32(j+1)*ph)
		for i := 0; i < spanX; i++ {
			x0 := int(bx + float32(i)*pw)
			x1 := int(bx + float32(i+1)*pw)
			r.FillRect(Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}, colours[j*spanX+i])
		}
	}
}

// Draw renders the box at its normal size, then — while the popup animation
// is above zero — the press-and-hold full-image overlay on top, so a single
// Draw call produces the correct final layering (the host does not need to
// sequence two separate draw calls). Call Update before Draw each frame.
func (p *PreviewPane) Draw(r Renderer, screenW, screenH int, theme Theme) {
	b := p.cfg.Bounds
	pvX, pvY := float32(b.X), float32(b.Y)
	pvW, pvH := float32(b.W), float32(b.H)
	iw, ih := p.cfg.Source.Width(), p.cfg.Source.Height()

	r.FillRect(b, theme.Panel)
	cx0, cy0, spanX, spanY, offX, offY, drawW, drawH :=
		previewRegion(iw, ih, b.W, b.H, p.zoom, p.focusX, p.focusY, pvX, pvY)
	r.Clip(b)
	p.drawSampledRegion(r, cx0, cy0, spanX, spanY, offX, offY, drawW, drawH)
	r.ClipEnd()
	r.StrokeRect(b, theme.Border, 1)

	if p.popup <= 0 {
		return
	}
	t := easeInOut(p.popup)
	scrW, scrH := float32(screenW), float32(screenH)
	anchorR := pvX + pvW // right edge stays fixed (top-right anchor)
	anchorT := pvY

	// --- Expanded end (t=1): whole image fills the popup box. ---
	const screenPad = 16 // matches the window-edge margin used throughout this GUI
	availW := anchorR - screenPad
	availH := scrH - screenPad - anchorT
	maxW := availW
	if maxW > scrW*0.75 {
		maxW = scrW * 0.75
	}
	maxH := availH
	if maxH > scrH*0.85 {
		maxH = scrH * 0.85
	}
	aspect := float32(iw) / float32(ih)
	tgtW := maxW
	tgtH := tgtW / aspect
	if tgtH > maxH {
		tgtH = maxH
		tgtW = tgtH * aspect
	}
	boxW1, boxH1 := tgtW, tgtH
	boxX1 := anchorR - boxW1
	boxY1 := anchorT
	scale1 := tgtW / float32(iw)
	originX1, originY1 := boxX1, boxY1

	// --- Collapsed end (t=0): match the normal box exactly. ---
	zoom := p.zoom
	if zoom < 1 {
		zoom = 1
	}
	ccx0, ccy0, _, _, coffX, coffY, _, _ :=
		previewRegion(iw, ih, b.W, b.H, zoom, p.focusX, p.focusY, pvX, pvY)
	boxW0, boxH0 := pvW, pvH
	boxX0, boxY0 := pvX, pvY
	scale0 := float32(zoom)
	originX0 := coffX - float32(ccx0)*scale0
	originY0 := coffY - float32(ccy0)*scale0

	// --- Interpolate box, scale and image origin between the two ends. ---
	lerp := func(a, b float32) float32 { return a + (b-a)*t }
	boxX := lerp(boxX0, boxX1)
	boxY := lerp(boxY0, boxY1)
	boxW := lerp(boxW0, boxW1)
	boxH := lerp(boxH0, boxH1)
	scale := lerp(scale0, scale1)
	originX := lerp(originX0, originX1)
	originY := lerp(originY0, originY1)

	// Backdrop dim scales with the animation.
	r.FillRect(Rect{X: 0, Y: 0, W: screenW, H: screenH},
		Colour{R: 0, G: 0, B: 0, A: uint8(150 * t)})

	popupRect := Rect{X: int(boxX), Y: int(boxY), W: int(boxW), H: int(boxH)}
	r.FillRect(popupRect, theme.Panel)
	r.Clip(popupRect)
	p.drawSampledRegion(r, 0, 0, iw, ih, originX, originY,
		float32(iw)*scale, float32(ih)*scale)
	r.ClipEnd()
	r.StrokeRect(popupRect, theme.Border, 2)
}

// easeInOut is a smoothstep ease for the popup animation.
func easeInOut(t float32) float32 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}
