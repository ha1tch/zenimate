package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
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

// update processes keyboard input for one frame and returns the modal state.
func (r *resetConfirm) update() resetConfirmState {
	if rl.IsKeyPressed(rl.KeyEscape) {
		return resetConfirmCancelled
	}
	// Accept typed letters (only letters are relevant to spelling YES).
	for {
		ch := rl.GetCharPressed()
		if ch == 0 {
			break
		}
		if ch >= 32 && ch < 128 && len(r.typed) < 8 {
			r.typed += string(rune(ch))
		}
	}
	if rl.IsKeyPressed(rl.KeyBackspace) && len(r.typed) > 0 {
		r.typed = r.typed[:len(r.typed)-1]
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		if equalFoldYES(r.typed) {
			return resetConfirmConfirmed
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
func (r *resetConfirm) draw(txt *bdfText, screenW, screenH int) {
	// Dim the backdrop.
	rl.DrawRectangle(0, 0, int32(screenW), int32(screenH), rl.NewColor(0x0a, 0x0a, 0x10, 200))

	pw, ph := 420, 150
	px := (screenW - pw) / 2
	py := (screenH - ph) / 2
	rl.DrawRectangle(int32(px), int32(py), int32(pw), int32(ph), colBtn)
	rl.DrawRectangleLines(int32(px), int32(py), int32(pw), int32(ph), colGrid)

	pad := 16
	txt.Draw("RESET EVERYTHING?", px+pad, py+pad, 2, colYellow)
	txt.Draw("This clears ALL frames and restores defaults.", px+pad, py+pad+26, 1, colText)
	txt.Draw("Type YES to confirm, or Esc to cancel.", px+pad, py+pad+42, 1, colDim)

	// Input box with the typed text and a caret.
	boxY := py + pad + 66
	boxH := 26
	rl.DrawRectangle(int32(px+pad), int32(boxY), int32(pw-2*pad), int32(boxH), colBG)
	rl.DrawRectangleLines(int32(px+pad), int32(boxY), int32(pw-2*pad), int32(boxH), colGrid)
	shown := upper(r.typed) + "_"
	txt.Draw(shown, px+pad+6, boxY+(boxH-txt.CellH()*2)/2, 2, colText)
}
