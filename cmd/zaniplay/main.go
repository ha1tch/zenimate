// Command zaniplay is a standalone terminal player for .zani animation files.
// It loads an animation (either physical form — zip or tzx), then loops through
// the frames, drawing each with ANSI colour using the upper-half-block trick
// (one character carries two vertical pixels). It is intentionally dependency-
// light: it reuses zenimate's model loader and zxpalette, nothing from the GUI.
//
// Usage:
//
//	zaniplay [-fps N] [-once] file.zani
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/pkg/zxpalette"
)

const (
	esc      = "\x1b"
	clearScr = esc + "[2J" + esc + "[H"
	hideCur  = esc + "[?25l"
	showCur  = esc + "[?25h"
	reset    = esc + "[0m"
)

// loadTarget loads an animation from either a standalone .zani/.zan/.zcut file
// or a "bundle.zbun#entry" reference selecting one animation from a bundle.
func loadTarget(target string) (*model.Sprite, error) {
	if bundlePath, entry, ok := splitBundleRef(target); ok {
		data, err := os.ReadFile(bundlePath)
		if err != nil {
			return nil, err
		}
		b, err := model.OpenBundle(data)
		if err != nil {
			return nil, err
		}
		return b.Sprite(entry)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	return model.LoadZCUT(data)
}

// splitBundleRef splits "bundle#entry"; ok is false for a plain path.
func splitBundleRef(target string) (bundlePath, entry string, ok bool) {
	for i := len(target) - 1; i >= 0; i-- {
		if target[i] == '#' {
			return target[:i], target[i+1:], true
		}
	}
	return "", "", false
}

func main() {
	fps := flag.Int("fps", 8, "frames per second")
	once := flag.Bool("once", false, "play once and stop on the last frame instead of looping")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: zaniplay [-fps N] [-once] file.zani | bundle.zbun#entry")
		os.Exit(2)
	}
	if *fps < 1 {
		*fps = 1
	}

	sprite, err := loadTarget(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "zaniplay: %v\n", err)
		os.Exit(1)
	}

	out := bufio.NewWriter(os.Stdout)
	out.WriteString(hideCur)
	out.Flush()

	// Restore the cursor on Ctrl-C.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		out.WriteString(showCur + reset)
		out.Flush()
		os.Exit(0)
	}()
	defer func() {
		out.WriteString(showCur + reset)
		out.Flush()
	}()

	delay := time.Second / time.Duration(*fps)
	n := sprite.FrameCount()
	for frame := 0; ; frame = (frame + 1) % n {
		renderFrame(out, sprite, frame)
		out.Flush()
		time.Sleep(delay)
		if *once && frame == n-1 {
			out.WriteString("\r\n")
			out.Flush()
			return
		}
	}
}

// renderFrame draws one frame using ZX attribute colours. Two vertical pixels
// share a character cell via the upper-half block: foreground = top pixel,
// background = bottom pixel. Each pixel shows its cell's ink colour when set, or
// paper colour when clear, at the cell's brightness.
func renderFrame(out *bufio.Writer, s *model.Sprite, frame int) {
	w, h := s.Width(), s.Height()
	out.WriteString(clearScr)
	fmt.Fprintf(out, "%s   %dx%d   frame %d/%d   name=%q%s\r\n\r\n",
		esc+"[1m", w, h, frame+1, s.FrameCount(), s.Name(), reset)

	for y := 0; y < h; y += 2 {
		out.WriteString("  ")
		for x := 0; x < w; x++ {
			top := pixelColour(s, frame, x, y)
			bot := top
			if y+1 < h {
				bot = pixelColour(s, frame, x, y+1)
			} else {
				bot = rgb{0, 0, 0} // off-grid bottom half is black
			}
			fmt.Fprintf(out, "%s%s\u2580", fgTrue(top), bgTrue(bot))
		}
		out.WriteString(reset + "\r\n")
	}
}

type rgb struct{ r, g, b uint8 }

// pixelColour resolves a pixel to its rendered RGB using the frame's attribute
// for that cell: ink when the pixel is set, paper when clear.
func pixelColour(s *model.Sprite, frame, x, y int) rgb {
	attr := s.AttrCellFrame(frame, x/8, y/8)
	ink := int(attr & 0x07)
	paper := int((attr >> 3) & 0x07)
	bright := attr&0x40 != 0
	idx := paper
	fr := s.Frame(frame)
	if fr[y*s.Width()+x] {
		idx = ink
	}
	col := zxpalette.Colour(idx, bright)
	return rgb{col.R, col.G, col.B}
}

func fgTrue(c rgb) string { return fmt.Sprintf("%s[38;2;%d;%d;%dm", esc, c.r, c.g, c.b) }
func bgTrue(c rgb) string { return fmt.Sprintf("%s[48;2;%d;%d;%dm", esc, c.r, c.g, c.b) }
