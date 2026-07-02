// Command zenimate-tui is the terminal frontend for the ZX Spectrum animated
// sprite editor. It renders the pixel grid with Unicode half-block characters
// (two pixel rows packed into one text row). Like the GUI it has three view
// modes, cycled with Tab: Bitmap Black and Bitmap White (set pixels black/white
// over a transparency chequer) and Spectrum Colour (real ZX attributes). The
// cursor cell blinks so the pixel beneath stays visible. Editing, frames and
// animation are driven through the shared ui.Controller.
//
// The TUI edits the bitmap plane and is colour-aware: it displays attributes and
// can stamp ink/paper per cell in colour-paint mode, but no action silently
// destroys colour. Clearing pixels keeps colour; the colour-wiping clear and
// reset require a confirmation.
//
// Controls:
//
//	arrows / hjkl  move cursor
//	tab            cycle view mode (Bitmap Black / White / Spectrum Colour)
//	space          set pixel (draw); in colour-paint mode, stamp ink+paper
//	backspace/del  clear pixel (erase)
//	enter          toggle pixel
//	i / o          decrease / increase the selected ink colour (0-7)
//	I / O          decrease / increase the selected paper colour (0-7)
//	b              toggle bright for painting
//	m              toggle colour-paint mode (stamp ink+paper on the cursor's cell)
//	[ ]            previous / next frame
//	1..9           jump to frame
//	+ / -          add / remove a frame
//	p              play / stop animation
//	c / v          copy / paste frame
//	x              clear pixels (keeps colour)
//	X              clear pixels AND colour (asks to confirm)
//	R              reset all frames (asks to confirm)
//	w / W          shrink / grow width by one character cell
//	t / T          shrink / grow height by one character cell
//	S              save to a .zani file (prompts for a name)
//	L              load a .zani/.zcut file (prompts for a name)
//	q              quit
//
// This frontend deliberately uses no third-party TUI library: it puts the
// terminal in raw mode and writes ANSI escapes directly.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

const (
	esc       = "\x1b"
	clearScr  = esc + "[2J" + esc + "[H"
	hideCur   = esc + "[?25l"
	showCur   = esc + "[?25h"
	reset     = esc + "[0m"
	inverse   = esc + "[7m"
	dim       = esc + "[2m"
	boldSeq   = esc + "[1m"
	yellowSeq = esc + "[33m"
)

// The monochrome bitmap views use xterm-256 indexed colour, exactly as the
// original TUI did, so the chequer and cursor render identically on every
// terminal (including those without truecolor). Spectrum Colour mode uses
// truecolor because accurate ZX hues need exact RGB.
const (
	chkLight  = 252 // xterm-256 light grey (208,208,208): transparency chequer
	chkDark   = 245 // xterm-256 medium grey (138,138,138): transparency chequer
	inkIdx    = 231 // xterm-256 pure white: a set pixel in Bitmap White
	blackIdx  = 16  // xterm-256 pure black: a set pixel in Bitmap Black
	cursorRed = 196 // xterm-256 bright red (255,0,0): the blinking cursor
)

// fg256 / bg256 build xterm-256 indexed SGR sequences. The TUI renders entirely
// in xterm-256 (no 24-bit truecolour), so it displays correctly on terminals
// that support indexed but not 24-bit colour, such as macOS Terminal.app.
func fg256(i int) string { return fmt.Sprintf("%s[38;5;%dm", esc, i) }
func bg256(i int) string { return fmt.Sprintf("%s[48;5;%dm", esc, i) }

// chequerIdx returns the transparency-chequer grey index for an off pixel at
// (x,y), alternating light/dark by parity.
func chequerIdx(x, y int) int {
	if (x+y)&1 == 0 {
		return chkLight
	}
	return chkDark
}

// pixelFG returns the SGR foreground sequence for a pixel in the given view
// mode. Bitmap modes emit xterm-256 (matching the original exactly); Spectrum
// Colour mode emits truecolor from the cell's real ZX attributes.
func pixelFG(s *model.Sprite, mode ui.ViewMode, x, y int) string {
	return colorSeq(s, mode, x, y, false)
}

// pixelBG is like pixelFG but emits a background sequence.
func pixelBG(s *model.Sprite, mode ui.ViewMode, x, y int) string {
	return colorSeq(s, mode, x, y, true)
}

// colorSeq builds the fg or bg colour sequence for the pixel at (x,y) under the
// current mode.
func colorSeq(s *model.Sprite, mode ui.ViewMode, x, y int, bg bool) string {
	on := s.At(x, y)
	switch mode {
	case ui.SpectrumColour:
		attr := s.AttrCellFrame(s.Selected(), x/model.Cell, y/model.Cell)
		idx := zxpalette.Paper(attr)
		if on {
			idx = zxpalette.Ink(attr)
		}
		x256 := zxpalette.Xterm256(idx, zxpalette.Bright(attr))
		if bg {
			return bg256(x256)
		}
		return fg256(x256)
	default: // BitmapBlack / BitmapWhite: xterm-256
		var idx int
		switch {
		case on && mode == ui.BitmapWhite:
			idx = inkIdx
		case on: // BitmapBlack
			idx = blackIdx
		default:
			idx = chequerIdx(x, y)
		}
		if bg {
			return bg256(idx)
		}
		return fg256(idx)
	}
}

// cursorFG / cursorBG return the blinking-cursor colour. It is the original
// xterm-256 bright red in every mode, so the cursor reads the same regardless of
// the view.
func cursorFG() string { return fg256(cursorRed) }
func cursorBG() string { return bg256(cursorRed) }

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "zenimate-tui:", err)
		os.Exit(1)
	}
}

func run() error {
	restore, err := makeRaw()
	if err != nil {
		return fmt.Errorf("entering raw mode: %w", err)
	}
	defer restore()

	out := bufio.NewWriter(os.Stdout)
	out.WriteString(hideCur)
	out.Flush()
	defer func() {
		out.WriteString(showCur)
		out.WriteString(reset)
		out.Flush()
	}()

	// Restore the terminal on Ctrl-C as well as normal exit.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		restore()
		fmt.Print(showCur, reset)
		os.Exit(0)
	}()

	c := ui.New(16, 16)
	ed := &editor{c: c}

	// Key input is read on a goroutine so the animation timer can run
	// concurrently with blocking reads.
	keys := make(chan []byte, 8)
	go func() {
		buf := make([]byte, 8)
		in := bufio.NewReader(os.Stdin)
		for {
			n, err := in.Read(buf)
			if err != nil {
				close(keys)
				return
			}
			b := make([]byte, n)
			copy(b, buf[:n])
			keys <- b
		}
	}()

	ticker := time.NewTicker(ui.PlayIntervalMS * time.Millisecond)
	defer ticker.Stop()

	blink := time.NewTicker(250 * time.Millisecond)
	defer blink.Stop()
	cursorOn := true

	render(out, ed, cursorOn)

	for {
		select {
		case <-ticker.C:
			if c.Playing() {
				c.Tick()
				render(out, ed, cursorOn)
			}
		case <-blink.C:
			cursorOn = !cursorOn
			render(out, ed, cursorOn)
		case b, ok := <-keys:
			if !ok {
				return nil
			}
			quit := false
			ed.handleKey(b, &quit)
			if quit {
				return nil
			}
			cursorOn = true
			blink.Reset(250 * time.Millisecond)
			render(out, ed, cursorOn)
		}
	}
}

// promptKind is the current modal text/confirmation prompt, if any.
type promptKind int

const (
	promptNone    promptKind = iota
	promptSave               // typing a filename to save
	promptLoad               // typing a filename to load
	promptConfirm            // y/n confirmation (for colour-destroying actions)
)

// editor holds the TUI's mutable state: the shared controller, cursor, the
// colour-paint toggle, and any active modal prompt.
type editor struct {
	c          *ui.Controller
	cx, cy     int
	colourMode bool // when true, painting stamps ink/paper on the cursor's cell

	prompt  promptKind
	promptQ string // prompt label shown to the user
	input   string // accumulated text for save/load prompts
	confirm func() // action to run if a confirm prompt is answered yes
}

// handleKey applies one key event, honouring any active modal prompt first.
func (ed *editor) handleKey(b []byte, quit *bool) {
	if ed.prompt != promptNone {
		ed.handlePrompt(b)
		return
	}

	c := ed.c
	s := c.Sprite
	w, h := s.Width(), s.Height()

	// Arrow keys arrive as ESC [ A/B/C/D.
	if len(b) == 3 && b[0] == 0x1b && b[1] == '[' {
		switch b[2] {
		case 'A':
			move(&ed.cy, -1, h)
		case 'B':
			move(&ed.cy, 1, h)
		case 'C':
			move(&ed.cx, 1, w)
		case 'D':
			move(&ed.cx, -1, w)
		}
		return
	}
	if len(b) == 0 {
		return
	}

	switch b[0] {
	case 'q', 3: // q or Ctrl-C
		*quit = true
	case 'h':
		move(&ed.cx, -1, w)
	case 'l':
		move(&ed.cx, 1, w)
	case 'k':
		move(&ed.cy, -1, h)
	case 'j':
		move(&ed.cy, 1, h)
	case ' ': // set pixel (draw); in colour mode, stamp the cell's colour
		if ed.colourMode {
			ed.stampColour()
		} else {
			c.Set(ed.cx, ed.cy, true)
		}
	case 0x7f, 0x08: // backspace / delete: clear pixel (erase)
		c.Set(ed.cx, ed.cy, false)
	case '\r', '\n': // toggle
		c.Toggle(ed.cx, ed.cy)
	case '\t': // cycle view mode: Bitmap Black -> White -> Spectrum Colour
		c.SetMode(nextMode(c.Mode()))
	case 'm':
		ed.colourMode = !ed.colourMode
		if ed.colourMode {
			c.SetStatus("Colour-paint mode ON (space stamps ink/paper)")
		} else {
			c.SetStatus("Colour-paint mode OFF (space draws pixels)")
		}
	case 'i':
		c.SetInk(c.Ink() - 1)
		c.SetStatus(inkStatus(c))
	case 'o':
		c.SetInk(c.Ink() + 1)
		c.SetStatus(inkStatus(c))
	case 'I':
		c.SetPaper(c.Paper() - 1)
		c.SetStatus(inkStatus(c))
	case 'O':
		c.SetPaper(c.Paper() + 1)
		c.SetStatus(inkStatus(c))
	case 'b':
		c.ToggleBright()
		c.SetStatus(inkStatus(c))
	case '[':
		c.PrevFrame()
	case ']':
		c.NextFrame()
	case 'p':
		c.TogglePlay()
	case 'c':
		c.CopyFrame()
	case 'v':
		c.PasteFrame()
	case 'x': // clear pixels, keep colour
		c.ClearFrameBitmap()
	case 'X': // clear pixels AND colour — confirm first
		ed.ask("Clear pixels AND colour? (y/n)", func() { ed.c.ClearFrame() })
	case 'R': // reset all frames (destroys colour) — confirm first
		ed.ask("Reset ALL frames and colour? (y/n)", func() { ed.c.Reset() })
	case 'w':
		c.SetWidth(cycleSize(w, -1))
		ed.clampCursor()
	case 'W':
		c.SetWidth(cycleSize(w, 1))
		ed.clampCursor()
	case 't':
		c.SetHeight(cycleSize(h, -1))
		ed.clampCursor()
	case 'T':
		c.SetHeight(cycleSize(h, 1))
		ed.clampCursor()
	case '+', '=':
		c.AddFrame()
	case '-', '_':
		c.RemoveFrame()
		ed.clampCursor()
	case 'S':
		ed.prompt = promptSave
		ed.promptQ = "Save as (.zani): "
		ed.input = defaultSaveName(c)
	case 'L':
		ed.prompt = promptLoad
		ed.promptQ = "Load file: "
		ed.input = ""
	default:
		n := c.Sprite.FrameCount()
		if n > 9 {
			n = 9
		}
		if b[0] >= '1' && b[0] <= byte('0'+n) {
			c.SelectFrame(int(b[0] - '1'))
		}
	}
}

// stampColour applies the selected ink+paper+bright to the cursor's character
// cell, never clearing pixels — colour is added, not destroyed.
func (ed *editor) stampColour() {
	cx := ed.cx / model.Cell
	cy := ed.cy / model.Cell
	ed.c.SetCellInk(cx, cy, ed.c.Ink())
	ed.c.SetCellPaper(cx, cy, ed.c.Paper())
	ed.c.SetStatus(inkStatus(ed.c))
}

// ask arms a yes/no confirmation prompt guarding a colour-destroying action.
func (ed *editor) ask(q string, action func()) {
	ed.prompt = promptConfirm
	ed.promptQ = q
	ed.confirm = action
}

// handlePrompt consumes a key while a modal prompt is active.
func (ed *editor) handlePrompt(b []byte) {
	switch ed.prompt {
	case promptConfirm:
		if len(b) > 0 && (b[0] == 'y' || b[0] == 'Y') {
			if ed.confirm != nil {
				ed.confirm()
			}
		} else {
			ed.c.SetStatus("Cancelled")
		}
		ed.prompt = promptNone
		ed.confirm = nil
	case promptSave, promptLoad:
		if len(b) == 0 {
			return
		}
		switch b[0] {
		case '\r', '\n': // submit
			name := strings.TrimSpace(ed.input)
			kind := ed.prompt
			ed.prompt = promptNone
			if name == "" {
				ed.c.SetStatus("Cancelled")
				return
			}
			if kind == promptSave {
				ed.doSave(name)
			} else {
				ed.doLoad(name)
			}
		case 0x1b: // ESC cancels
			ed.prompt = promptNone
			ed.c.SetStatus("Cancelled")
		case 0x7f, 0x08: // backspace
			if ed.input != "" {
				ed.input = ed.input[:len(ed.input)-1]
			}
		default:
			// Accept printable ASCII into the filename.
			for _, ch := range b {
				if ch >= 0x20 && ch < 0x7f {
					ed.input += string(ch)
				}
			}
		}
	}
}

func move(v *int, d, n int) {
	*v += d
	if *v < 0 {
		*v = 0
	}
	if *v >= n {
		*v = n - 1
	}
}

func (ed *editor) clampCursor() {
	if ed.cx >= ed.c.Sprite.Width() {
		ed.cx = ed.c.Sprite.Width() - 1
	}
	if ed.cy >= ed.c.Sprite.Height() {
		ed.cy = ed.c.Sprite.Height() - 1
	}
}

// nextMode cycles Bitmap Black -> Bitmap White -> Spectrum Colour -> Black.
func nextMode(m ui.ViewMode) ui.ViewMode {
	switch m {
	case ui.BitmapBlack:
		return ui.BitmapWhite
	case ui.BitmapWhite:
		return ui.SpectrumColour
	default:
		return ui.BitmapBlack
	}
}

// inkStatus summarises the current painting colour selection.
func inkStatus(c *ui.Controller) string {
	br := ""
	if c.Bright() {
		br = " bright"
	}
	return fmt.Sprintf("ink %d (%s) / paper %d (%s)%s",
		c.Ink(), zxpalette.Names[c.Ink()], c.Paper(), zxpalette.Names[c.Paper()], br)
}

// cycleSize steps a dimension by one character cell (8 px) in the given
// direction; the model clamps to its valid range.
func cycleSize(cur, dir int) int {
	return cur + dir*model.Cell
}

// defaultSaveName suggests a filename from the sprite's name, with the save-form
// extension.
func defaultSaveName(c *ui.Controller) string {
	name := c.Sprite.Name()
	if name == "" {
		name = "sprite"
	}
	return name + "." + model.AnimationExt(c.SaveForm())
}

// doSave writes the whole sprite (all frames and attributes) to a .zani file.
func (ed *editor) doSave(name string) {
	data, err := ed.c.Sprite.MarshalZCUT()
	if err != nil {
		ed.c.SetStatus("Save failed: " + err.Error())
		return
	}
	if err := os.WriteFile(name, data, 0o644); err != nil {
		ed.c.SetStatus("Save failed: " + err.Error())
		return
	}
	ed.c.SetStatus(fmt.Sprintf("Saved %s (%d frames)", name, ed.c.Sprite.FrameCount()))
}

// doLoad reads a .zani/.zan/.zcut animation and replaces the edited sprite.
func (ed *editor) doLoad(name string) {
	data, err := os.ReadFile(name)
	if err != nil {
		ed.c.SetStatus("Load failed: " + err.Error())
		return
	}
	ext := extOf(name)
	if !model.IsAnimationExt(ext) {
		ed.c.SetStatus("Load: only .zani/.zan/.zcut animations are supported here")
		return
	}
	s, err := model.LoadByExtension(ext, data)
	if err != nil {
		ed.c.SetStatus("Load failed: " + err.Error())
		return
	}
	s.SetName(baseNoExt(name))
	ed.c.LoadSprite(s)
	ed.clampCursor()
}

// extOf returns the lowercase extension (no dot) of a path.
func extOf(path string) string {
	dot := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			dot = i
			break
		}
		if path[i] == '/' {
			break
		}
	}
	if dot < 0 {
		return ""
	}
	return strings.ToLower(path[dot+1:])
}

// baseNoExt returns the final path element without its extension.
func baseNoExt(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		path = path[i+1:]
	}
	if dot := strings.LastIndexByte(path, '.'); dot > 0 {
		path = path[:dot]
	}
	return path
}

// render paints the whole screen: header, frame selector, the colour half-block
// grid with the cursor highlighted, the status/prompt line, and the help panel.
func render(out *bufio.Writer, ed *editor, cursorOn bool) {
	c := ed.c
	s := c.Sprite
	w, h := s.Width(), s.Height()

	out.WriteString(clearScr)

	mode := ""
	if ed.colourMode {
		mode = "  [COLOUR-PAINT]"
	}
	fmt.Fprintf(out, "%s%sZX Spectrum sprite editor%s   %dx%d   frame %d/%d   %s%s\r\n",
		boldSeq, yellowSeq, reset, w, h, s.Selected()+1, s.FrameCount(), c.Mode().String(), mode)

	// Frame selector row.
	out.WriteString("frames: ")
	for i := 0; i < s.FrameCount(); i++ {
		label := fmt.Sprintf(" %d ", i+1)
		if i == s.Selected() {
			fmt.Fprintf(out, "%s%s%s", inverse, label, reset)
		} else {
			out.WriteString(label)
		}
	}
	out.WriteString("\r\n\r\n")

	// The grid packs two vertical pixels into one character using the upper half
	// block '▀': foreground paints the TOP pixel, background the BOTTOM pixel.
	// Each half is coloured per the current view mode (xterm-256 chequer in the
	// bitmap modes, truecolor ZX attributes in Spectrum Colour). The cursor half
	// is drawn red while blinking "on".
	for y := 0; y < h; y += 2 {
		out.WriteString("  ")
		for x := 0; x < w; x++ {
			bottomY := y + 1
			hasBottom := bottomY < h

			topSeq := pixelFG(s, c.Mode(), x, y)
			var botSeq string
			if hasBottom {
				botSeq = pixelBG(s, c.Mode(), x, bottomY)
			} else {
				botSeq = bg256(blackIdx) // off-grid bottom half is black
			}
			if cursorOn && ed.cx == x {
				if ed.cy == y {
					topSeq = cursorFG()
				}
				if ed.cy == bottomY {
					botSeq = cursorBG()
				}
			}
			fmt.Fprintf(out, "%s%s\u2580%s", topSeq, botSeq, reset)
		}
		out.WriteString("\r\n")
	}

	// Colour selection swatch line.
	out.WriteString("\r\n")
	inkX := zxpalette.Xterm256(c.Ink(), c.Bright())
	paperX := zxpalette.Xterm256(c.Paper(), c.Bright())
	fmt.Fprintf(out, "colour: ink %s  %s%s %s paper %s  %s%s %s %s\r\n",
		fg256(inkX)+"\u2588\u2588"+reset, dim, inkName(c.Ink()), reset,
		fg256(paperX)+"\u2588\u2588"+reset, dim, inkName(c.Paper()), reset,
		brightLabel(c.Bright()))

	// Status / prompt line.
	if ed.prompt == promptSave || ed.prompt == promptLoad {
		fmt.Fprintf(out, "%s%s%s%s\r\n", boldSeq, ed.promptQ, ed.input, reset)
	} else if ed.prompt == promptConfirm {
		fmt.Fprintf(out, "%s%s%s\r\n", boldSeq, ed.promptQ, reset)
	} else if st := c.Status(); st != "" {
		fmt.Fprintf(out, "%s%s%s\r\n", dim, st, reset)
	} else {
		out.WriteString("\r\n")
	}

	// Help.
	out.WriteString(dim)
	out.WriteString("move arrows/hjkl  draw space  erase bksp  toggle enter  tab view mode\r\n")
	out.WriteString("frame [ ] 1-9  +/- frame  colour i/o ink  I/O paper  b bright  m colour-paint\r\n")
	out.WriteString("copy c  paste v  clear x (keep colour)  X wipe colour  R reset  size w/W t/T\r\n")
	out.WriteString("save S  load L  play p  quit q")
	out.WriteString(reset)
	out.WriteString("\r\n")

	out.Flush()
}

func inkName(i int) string {
	if i >= 0 && i < len(zxpalette.Names) {
		return zxpalette.Names[i]
	}
	return "?"
}

func brightLabel(b bool) string {
	if b {
		return "bright"
	}
	return "normal"
}
