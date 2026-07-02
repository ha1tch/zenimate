package main

import (
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// OSD shows status messages as an animated on-screen caption: each new message
// rises from below the bottom-right window border, drifts upward past the
// palette, and fades out over a fixed travel distance. Small "magical pixel"
// sprites scatter around the caption's bounding box, refreshed periodically.

const (
	osdHold       = 100.0 // pixels risen fully-opaque before the fade begins
	osdFade       = 100.0 // pixels over which it then fades to nothing
	osdTravel     = osdHold + osdFade
	osdSpeed      = 60.0 // pixels per second the caption rises
	osdSparkMS    = 0.2  // seconds between magical-pixel refreshes
	osdSparkCount = 10   // magical pixels around the caption
	osdSparkScale = 3    // screen pixels per magical-pixel cell
	osdSparkBand  = 26   // how far (px) sparks scatter beyond the text box
	osdTextScale  = 2    // caption text render scale
)

// magicalShapes are the spark sprites, as rows of a pixel mask.
var magicalShapes = [][]string{
	{".#.", "#.#", ".#."},
	{"##", "##"},
	{"#.#", ".#.", "#.#"},
	{"#...", "....", "....", "...#"},
	{"...#", "....", "....", "#..."},
}

type spark struct {
	x, y  float32
	shape int
}

type osd struct {
	seq    int     // last status sequence we animated
	text   string  // caption text
	travel float32 // pixels risen so far (0..osdTravel)
	active bool
	sparkT float32 // time accumulator for spark refresh
	sparks []spark
	rng    *rand.Rand
}

func newOSD() *osd {
	return &osd{rng: rand.New(rand.NewSource(1))}
}

// trigger starts a fresh rise+fade for a new message.
func (o *osd) start(text string) {
	o.text = text
	o.travel = 0
	o.active = true
	o.sparkT = osdSparkMS // force an immediate spark layout on first update
}

// update advances the animation. seq/text come from the controller; when seq
// changes a fresh caption is launched. tw is the caption text width (px) so the
// sparks can scatter around its bounding box.
func (o *osd) update(dt float32, seq int, text string, tw, th int) {
	if seq != o.seq {
		o.seq = seq
		if text != "" {
			o.start(text)
		}
	}
	if !o.active {
		return
	}
	o.travel += float32(osdSpeed) * dt
	if o.travel >= float32(osdTravel) {
		o.active = false
		o.sparks = nil
		return
	}
	o.sparkT += dt
	if o.sparkT >= float32(osdSparkMS) {
		o.sparkT = 0
		o.layoutSparks(tw, th)
	}
}

// layoutSparks scatters the magical pixels in a band around a tw x th box whose
// top-left is the origin (0,0); positions are relative and offset at draw time.
func (o *osd) layoutSparks(tw, th int) {
	o.sparks = o.sparks[:0]
	b := float32(osdSparkBand)
	for i := 0; i < osdSparkCount; i++ {
		// Pick a point in the expanded box, rejecting the inner text area so the
		// sparks sit around (not on top of) the caption.
		var x, y float32
		for tries := 0; tries < 8; tries++ {
			x = -b + o.rng.Float32()*(float32(tw)+2*b)
			y = -b + o.rng.Float32()*(float32(th)+2*b)
			if x < 0 || x > float32(tw) || y < 0 || y > float32(th) {
				break // outside the text box: good
			}
		}
		o.sparks = append(o.sparks, spark{x: x, y: y, shape: o.rng.Intn(len(magicalShapes))})
	}
}

// alpha returns the caption opacity (0..1): fully opaque for the first osdHold
// pixels of travel, then fading linearly to zero over the next osdFade pixels.
func (o *osd) alpha() float32 {
	if o.travel <= float32(osdHold) {
		return 1
	}
	a := 1 - (o.travel-float32(osdHold))/float32(osdFade)
	if a < 0 {
		a = 0
	}
	return a
}

// draw renders the caption and its sparks, anchored to the bottom-right corner.
// (anchorX, anchorY) is the bottom-right point the caption rises from (the
// window's bottom-right inner corner). txt draws the text; tw/th measure it.
func (o *osd) draw(txt *bdfText, anchorX, anchorY int, tw, th int) {
	if !o.active {
		return
	}
	a := uint8(o.alpha() * 255)

	// Caption position: rises from below the bottom border (at travel 0 the text
	// sits just below the window edge), right-aligned, moving up as travel grows.
	x := anchorX - tw
	y := anchorY - int(o.travel)

	orange := rl.NewColor(0xff, 0x9a, 0x10, a)
	black := rl.NewColor(0, 0, 0, a)

	// Sparks first (behind the text), positioned around the text box.
	for _, sp := range o.sparks {
		drawMagical(sp.shape, x+int(sp.x), y+int(sp.y), orange, black)
	}

	// Outlined, bolded caption. Order: black outline in eight directions first
	// (behind), then the orange fill painted twice with a 1px offset to thicken
	// it, so the orange stays on top and reads bolder.
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
		txt.Draw(o.text, x+d[0]*osdTextScale, y+d[1]*osdTextScale, osdTextScale, black)
	}
	txt.Draw(o.text, x, y, osdTextScale, orange)
	txt.Draw(o.text, x+1, y, osdTextScale, orange)
}

// drawMagical renders one spark sprite at (x,y) with a 1px black outline.
func drawMagical(shape, x, y int, fill, outline rl.Color) {
	rows := magicalShapes[shape]
	s := int32(osdSparkScale)
	for r, row := range rows {
		for c := 0; c < len(row); c++ {
			if row[c] != '#' {
				continue
			}
			px := int32(x) + int32(c)*s
			py := int32(y) + int32(r)*s
			// Outline then fill for a crisp OSD look.
			rl.DrawRectangle(px-1, py-1, s+2, s+2, outline)
			rl.DrawRectangle(px, py, s, s, fill)
		}
	}
}
