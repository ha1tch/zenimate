package main

import (
	_ "embed"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

//go:embed help.txt
var helpText string

// helpModal is a scrollable text reader shown over the editor. It displays the
// embedded help.txt with wheel/arrow/PgUp/PgDn/Home/End scrolling, a scrollbar,
// and a close box; Esc or the close box dismisses it.
// help text sizing. Bitmap fonts only look crisp at whole-number scales, so the
// body scale is always an integer: helpBodyScaleBase when a 70-column panel fits
// the screen width at that scale, otherwise 1. Height is never considered.
const (
	helpBodyScaleBase = 2
	helpTargetCols    = 70 // preferred panel width, in characters
)

// helpPanelWidth returns the panel width for a 70-column body at the given text
// scale: the measured text width plus horizontal padding and the scrollbar
// gutter.
func helpPanelWidth(r zenui.Renderer, scale int) int {
	pad := r.LineHeight(1)
	cols := strings.Repeat("M", helpTargetCols)
	return r.TextWidth(cols, scale) + 2*pad + 12
}

// helpBodyScaleFor picks the body text scale: the base scale if a 70-column
// panel fits within the screen width at that scale, otherwise 1. Only width
// matters.
func helpBodyScaleFor(r zenui.Renderer, screenW int) int {
	pad := r.LineHeight(1)
	if helpPanelWidth(r, helpBodyScaleBase) <= screenW-2*pad {
		return helpBodyScaleBase
	}
	return 1
}

type helpModal struct {
	lines    []string
	scroll   int // first visible line index
	panel    zenui.Rect
	body     zenui.Rect // text area (inside the panel, above nothing)
	closeBx  zenui.Rect
	visible  int // lines that fit in the body (set during layout)
	total    int
	bodyLH   int // body line height at the effective scale (set during layout)
	effScale int // effective body text scale for the current screen width

	// scrollbar drag state
	track      zenui.Rect
	thumb      zenui.Rect
	dragging   bool
	dragOffset int // pointer offset within the thumb at grab time
}

func newHelpModal() *helpModal {
	return &helpModal{lines: strings.Split(strings.TrimRight(helpText, "\n"), "\n")}
}

// help line classification for the lightweight markdown the reader understands.
type helpKind int

const (
	helpBody     helpKind = iota // ordinary prose
	helpIndented                 // indented (shortcut tables etc.)
	helpH1                       // "# " heading
	helpH2                       // "## " heading
)

// helpLineKind classifies a help line and returns the text to draw (with any
// markdown heading marker stripped).
func helpLineKind(line string) (helpKind, string) {
	switch {
	case strings.HasPrefix(line, "## "):
		return helpH2, line[3:]
	case strings.HasPrefix(line, "# "):
		return helpH1, line[2:]
	case strings.HasPrefix(line, "  "):
		return helpIndented, line
	default:
		return helpBody, line
	}
}

func (h *helpModal) layout(r zenui.Renderer, screenW, screenH int) {
	lh := r.LineHeight(1)
	pad := lh
	// raylib 2D drawing is in logical screen coordinates, so font width at a given
	// scale and screenW are in the same space; the scale decision uses screenW.
	h.effScale = helpBodyScaleFor(r, screenW)

	// Panel width holds 70 columns at the effective scale, capped to the screen.
	pw := helpPanelWidth(r, h.effScale)
	if min := 320; pw < min {
		pw = min
	}
	if pw > screenW-2*pad {
		pw = screenW - 2*pad
	}
	ph := screenH - 4*pad
	if ph < lh*10 {
		ph = lh * 10
	}
	px := (screenW - pw) / 2
	py := (screenH - ph) / 2
	h.panel = zenui.Rect{X: px, Y: py, W: pw, H: ph}

	titleH := r.LineHeight(2) + 8
	// Close box: top-right corner of the panel.
	cb := r.LineHeight(2)
	h.closeBx = zenui.Rect{X: px + pw - pad - cb, Y: py + pad, W: cb, H: cb}

	h.body = zenui.Rect{
		X: px + pad,
		Y: py + pad + titleH,
		W: pw - 2*pad - 12, // leave room for the scrollbar on the right
		H: ph - 2*pad - titleH,
	}
	// Body text is rendered at an integer scale so the bitmap font stays crisp.
	// Line height follows the same scale.
	h.bodyLH = r.LineHeight(h.effScale)
	if h.bodyLH < 1 {
		h.bodyLH = 1
	}
	h.visible = h.body.H / h.bodyLH
	if h.visible < 1 {
		h.visible = 1
	}
	h.total = len(h.lines)
	h.clampScroll()
}

func (h *helpModal) clampScroll() {
	maxScroll := h.total - h.visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if h.scroll > maxScroll {
		h.scroll = maxScroll
	}
	if h.scroll < 0 {
		h.scroll = 0
	}
}

// update handles scrolling and dismissal. It returns false when the modal should
// close.
func (h *helpModal) update(in zenui.Input) bool {
	for _, k := range in.Keys {
		switch k {
		case zenui.KeyEscape:
			return false
		case zenui.KeyUp:
			h.scroll--
		case zenui.KeyDown:
			h.scroll++
		case zenui.KeyPageUp:
			h.scroll -= h.visible
		case zenui.KeyPageDown:
			h.scroll += h.visible
		}
	}
	// Home/End are not in the shared zenui key set; read them directly.
	if rl.IsKeyPressed(rl.KeyHome) {
		h.scroll = 0
	}
	if rl.IsKeyPressed(rl.KeyEnd) {
		h.scroll = h.total
	}
	if in.WheelY != 0 {
		h.scroll -= int(in.WheelY) * 3
	}

	// Scrollbar dragging. On press over the thumb, grab it; while held, map the
	// pointer to a scroll position; a press on the track (not the thumb) pages
	// toward the pointer.
	if in.MousePressed && h.thumb.W > 0 {
		if h.thumb.Contains(in.MouseX, in.MouseY) {
			h.dragging = true
			h.dragOffset = in.MouseY - h.thumb.Y
		} else if h.track.Contains(in.MouseX, in.MouseY) {
			if in.MouseY < h.thumb.Y {
				h.scroll -= h.visible
			} else {
				h.scroll += h.visible
			}
		}
	}
	if h.dragging {
		if !in.MouseDown {
			h.dragging = false
		} else if h.track.H > h.thumb.H {
			// Position the thumb top at pointer-minus-offset, map to scroll range.
			rel := float32(in.MouseY-h.dragOffset-h.track.Y) / float32(h.track.H-h.thumb.H)
			if rel < 0 {
				rel = 0
			}
			if rel > 1 {
				rel = 1
			}
			h.scroll = int(rel * float32(h.total-h.visible))
		}
	}
	h.clampScroll()

	if in.MousePressed {
		if h.closeBx.Contains(in.MouseX, in.MouseY) {
			return false
		}
		// A click outside the panel also closes, matching the other choosers.
		if !h.panel.Contains(in.MouseX, in.MouseY) {
			return false
		}
	}
	return true
}

func (h *helpModal) draw(r fpRenderer, screenW, screenH int) {
	h.layout(zenui.Renderer(r), screenW, screenH)
	th := fpTheme()
	lh := r.LineHeight(1)
	pad := lh

	r.FillRect(zenui.Rect{X: 0, Y: 0, W: screenW, H: screenH}, th.Backdrop)
	r.FillRect(h.panel, th.Panel)
	r.StrokeRect(h.panel, th.Border, 1)

	r.DrawText("HELP", h.panel.X+pad, h.panel.Y+pad, 2, th.Text)

	// Close box.
	mx := int(rl.GetMouseX())
	my := int(rl.GetMouseY())
	cbg := th.Button
	if h.closeBx.Contains(mx, my) {
		cbg = th.ButtonHot
	}
	r.FillRect(h.closeBx, cbg)
	r.StrokeRect(h.closeBx, th.Border, 1)
	r.DrawText("x", h.closeBx.X+(h.closeBx.W-r.TextWidth("x", 1))/2,
		h.closeBx.Y+(h.closeBx.H-lh)/2, 1, th.ButtonText)

	// Body text, clipped, at the effective integer scale. Markdown headings
	// (# H1, ## H2) render in the accent colour and are emboldened by a one-pixel
	// horizontal double-strike; the leading markers are stripped. Indented lines
	// are dimmer body text; other lines are normal body text.
	r.Clip(h.body)
	y := h.body.Y
	end := h.scroll + h.visible
	if end > h.total {
		end = h.total
	}
	for i := h.scroll; i < end; i++ {
		kind, text := helpLineKind(h.lines[i])
		switch kind {
		case helpH1, helpH2:
			// Accent colour, bold via a one-pixel horizontal double-strike.
			r.DrawText(text, h.body.X, y, h.effScale, th.DirText)
			r.DrawText(text, h.body.X+1, y, h.effScale, th.DirText)
		case helpIndented:
			r.DrawText(text, h.body.X, y, h.effScale, th.DimText)
		default:
			r.DrawText(text, h.body.X, y, h.effScale, th.Text)
		}
		y += h.bodyLH
	}
	r.ClipEnd()

	// Scrollbar on the right of the body, when content overflows. The track and
	// thumb rects are stored so update() can hit-test drags.
	if h.total > h.visible {
		h.track = zenui.Rect{X: h.body.X + h.body.W + 6, Y: h.body.Y, W: 8, H: h.body.H}
		r.FillRect(h.track, th.Button)
		frac := float32(h.visible) / float32(h.total)
		thumbH := int(float32(h.track.H) * frac)
		if thumbH < 16 {
			thumbH = 16
		}
		var pos float32
		if h.total-h.visible > 0 {
			pos = float32(h.scroll) / float32(h.total-h.visible)
		}
		thumbY := h.track.Y + int(float32(h.track.H-thumbH)*pos)
		h.thumb = zenui.Rect{X: h.track.X, Y: thumbY, W: h.track.W, H: thumbH}
		thumbCol := th.DirText
		if h.dragging {
			thumbCol = th.Text
		}
		r.FillRect(h.thumb, thumbCol)
	} else {
		h.track = zenui.Rect{}
		h.thumb = zenui.Rect{}
	}
}
