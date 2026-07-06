package zenui

// ImageSource is a read-only view onto raster image data PreviewPane can
// preview: dimensions plus a rectangular colour sample. PreviewPane never
// interprets pixel meaning itself — it just asks for a region and draws it
// — so it works for previewing any raster content, not just ZX sprites. The
// host implements this interface over its own live document (e.g. a
// zenimate Sprite); because Width/Height are queried fresh each frame
// rather than copied into a config at construction time, a resize is
// reflected immediately with no separate notification needed.
//
// Region rather than a per-pixel query: PreviewPane only ever needs a
// contiguous rectangle at a time (the visible span, or the whole image for
// the popup), and asking for it in one call lets the host do a single bulk
// conversion pass — decoding packed sprite bits and ZX attributes into
// colours together — instead of paying per-call dispatch overhead for
// every individual pixel (up to 49152 calls/frame in the worst case, for a
// full-screen sprite's popup). It also matches what a preview actually is:
// a sub-image to blit, not a set of independent scattered pixel queries.
type ImageSource interface {
	// Width and Height are the full image dimensions.
	Width() int
	Height() int
	// Region returns the colours of the w x h rectangle starting at (x0,y0),
	// row-major (index = row*w+col), length w*h. PreviewPane only ever calls
	// this with rectangles fully inside [0,Width()) x [0,Height()) — the
	// host does not need to handle out-of-range coordinates. A host wanting
	// a transparency chequer pattern (or any other mode-dependent shading)
	// computes it itself here.
	Region(x0, y0, w, h int) []Colour
}

// PreviewPaneConfig sets up a fixed-size detail preview with press-and-hold
// full-image popup and right-click zoom cycling.
type PreviewPaneConfig struct {
	Bounds Rect        // the box's screen position and size
	Source ImageSource // what the pane samples pixel colours from
	// MinZoom and MaxZoom bound the integer zoom right-click cycles through
	// (e.g. 1..4). Zoom starts at MinZoom.
	MinZoom, MaxZoom int
}

// PreviewPane is a fixed-size detail preview: a box showing part of an image
// at an integer zoom, centred on a host-supplied focus point, with
// right-click zoom cycling and a press-and-hold full-image popup that grows
// out of the box's top-right corner. Construct with NewPreviewPane, then
// each frame call SetFocus, Update, and Draw, in that order.
type PreviewPane struct {
	cfg            PreviewPaneConfig
	zoom           int
	focusX, focusY int
	held           bool
	popup          float32 // 0 = collapsed, 1 = fully expanded; eased over time
}

// NewPreviewPane creates a pane from cfg. It never returns nil.
func NewPreviewPane(cfg PreviewPaneConfig) *PreviewPane {
	zoom := cfg.MinZoom
	if zoom < 1 {
		zoom = 1
	}
	return &PreviewPane{cfg: cfg, zoom: zoom}
}

// SetBounds repositions/resizes the box (e.g. on window resize).
func (p *PreviewPane) SetBounds(bounds Rect) { p.cfg.Bounds = bounds }

// Bounds returns the box's current screen rect.
func (p *PreviewPane) Bounds() Rect { return p.cfg.Bounds }

// SetFocus sets which image pixel the preview is centred on. The host
// computes this from its own canvas/cursor logic each frame — PreviewPane
// has no notion of a canvas or viewport, only the image it samples from.
func (p *PreviewPane) SetFocus(x, y int) { p.focusX, p.focusY = x, y }

// Zoom returns the current integer zoom (px per image pixel).
func (p *PreviewPane) Zoom() int { return p.zoom }

// SetZoom forces the current zoom, clamped to [MinZoom,MaxZoom]. Mainly for
// setting a starting zoom other than MinZoom at construction time; right-click
// cycling (see Update) is the normal way zoom changes during use.
func (p *PreviewPane) SetZoom(z int) {
	if z < p.cfg.MinZoom {
		z = p.cfg.MinZoom
	}
	if z > p.cfg.MaxZoom {
		z = p.cfg.MaxZoom
	}
	p.zoom = z
}

// Popup returns the current popup animation progress, 0 (collapsed) to 1
// (fully expanded). Exposed mainly for tests; hosts do not normally need it.
func (p *PreviewPane) Popup() float32 { return p.popup }

// popupEaseRate controls how quickly the popup grows/shrinks: higher is
// snappier. Matches the same dt-eased-toward-target convention used
// elsewhere in this codebase (e.g. the button-fade animation), so the same
// visual pacing feels consistent across every animated element.
const popupEaseRate = 10

// Update advances the popup animation and handles right-click zoom-cycling
// and the press-and-hold gesture, hit-tested against the pane's own bounds.
// dt is the frame's elapsed time in seconds.
func (p *PreviewPane) Update(in Input, dt float32) {
	over := p.cfg.Bounds.Contains(in.MouseX, in.MouseY)
	if in.MousePressed && over {
		p.held = true
	}
	if !in.MouseDown {
		p.held = false
	}
	if in.MouseRightPressed && over {
		p.zoom++
		if p.zoom > p.cfg.MaxZoom {
			p.zoom = p.cfg.MinZoom
		}
	}

	target := float32(0)
	if p.held {
		target = 1
	}
	k := dt * popupEaseRate
	if k > 1 {
		k = 1
	}
	if k < 0 {
		k = 0
	}
	p.popup += (target - p.popup) * k
	if p.popup < 0.001 {
		p.popup = 0
	}
}
