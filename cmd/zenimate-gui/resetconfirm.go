package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guidraw"
	"github.com/ha1tch/zenimate/cmd/zenimate-gui/internal/guiutil"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

// resetConfirm is a small typed-confirmation modal for the destructive Reset.
// The user must type the full word YES (case-insensitive) and press Enter to
// proceed; Esc cancels. This guards against accidentally destroying the whole
// animation with a single click.
type resetConfirm struct {
	typed string // characters entered so far
}

func newResetConfirm() *resetConfirm { return &resetConfirm{} }

// resetConfirmState is the outcome of one update tick.
type resetConfirmState int

const (
	resetConfirmOpen      resetConfirmState = iota // still waiting for input
	resetConfirmConfirmed                          // user typed YES and pressed Enter
	resetConfirmCancelled                          // user cancelled (Esc)
)

// update processes one frame's shared input and returns the modal state.
// Takes zenui.Input rather than reading raylib directly: rl.GetCharPressed()
// destructively drains a queue shared with the rest of the frame, and the
// main loop already calls it exactly once per frame (see fpInput() in
// main.go) — a second, direct call here would only ever see an
// already-emptied queue. This was the actual bug behind "can't type
// anything in the box": every character was being silently lost to the
// same queue-draining problem fixed earlier for the text tool, just in a
// spot that hadn't been audited for it at the time.
func (r *resetConfirm) update(in zenui.Input) resetConfirmState {
	for _, r2 := range in.Chars {
		if r2 >= 32 && r2 < 128 && len(r.typed) < 8 {
			r.typed += string(r2)
		}
	}
	for _, k := range in.Keys {
		switch k {
		case zenui.KeyEscape:
			return resetConfirmCancelled
		case zenui.KeyBackspace:
			if len(r.typed) > 0 {
				r.typed = r.typed[:len(r.typed)-1]
			}
		case zenui.KeyEnter:
			if equalFoldYES(r.typed) {
				return resetConfirmConfirmed
			}
		}
	}
	return resetConfirmOpen
}

// equalFoldYES reports whether s is "YES" ignoring case.
func equalFoldYES(s string) bool {
	if len(s) != 3 {
		return false
	}
	up := func(b byte) byte {
		if b >= 'a' && b <= 'z' {
			return b - 32
		}
		return b
	}
	return up(s[0]) == 'Y' && up(s[1]) == 'E' && up(s[2]) == 'S'
}

// draw renders the confirmation panel centred on screen.
func (r *resetConfirm) draw(txt *guidraw.BDFText, screenW, screenH int) {
	theme := guidraw.DefaultTheme()

	// Dim the backdrop.
	rl.DrawRectangle(0, 0, int32(screenW), int32(screenH), rl.NewColor(0x0a, 0x0a, 0x10, 200))

	pw, ph := 420, 150
	px := (screenW - pw) / 2
	py := (screenH - ph) / 2
	rl.DrawRectangle(int32(px), int32(py), int32(pw), int32(ph), theme.Btn)
	rl.DrawRectangleLines(int32(px), int32(py), int32(pw), int32(ph), theme.Grid)

	pad := 16
	txt.Draw("RESET EVERYTHING?", px+pad, py+pad, 2, theme.Yellow)
	txt.Draw("This clears ALL frames and restores defaults.", px+pad, py+pad+26, 1, theme.Text)
	txt.Draw("Type YES to confirm, or Esc to cancel.", px+pad, py+pad+42, 1, theme.Dim)

	// Input box with the typed text and a caret.
	boxY := py + pad + 66
	boxH := 26
	rl.DrawRectangle(int32(px+pad), int32(boxY), int32(pw-2*pad), int32(boxH), theme.BG)
	rl.DrawRectangleLines(int32(px+pad), int32(boxY), int32(pw-2*pad), int32(boxH), theme.Grid)
	shown := guiutil.Upper(r.typed) + "_"
	txt.Draw(shown, px+pad+6, boxY+(boxH-txt.CellH()*2)/2, 2, theme.Text)
}
