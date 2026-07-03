package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

// fpRenderer adapts the bdfText/raylib drawing surface to zenui.Renderer.
type fpRenderer struct {
	txt *bdfText
}

func fpColor(c zenui.Colour) rl.Color { return rl.NewColor(c.R, c.G, c.B, c.A) }

func (r fpRenderer) FillRect(rc zenui.Rect, c zenui.Colour) {
	rl.DrawRectangle(int32(rc.X), int32(rc.Y), int32(rc.W), int32(rc.H), fpColor(c))
}

func (r fpRenderer) StrokeRect(rc zenui.Rect, c zenui.Colour, thickness int) {
	rl.DrawRectangleLinesEx(
		rl.NewRectangle(float32(rc.X), float32(rc.Y), float32(rc.W), float32(rc.H)),
		float32(thickness), fpColor(c))
}

func (r fpRenderer) DrawText(s string, x, y, scale int, c zenui.Colour) {
	r.txt.Draw(s, x, y, scale, fpColor(c))
}

func (r fpRenderer) TextWidth(s string, scale int) int { return r.txt.Measure(s, scale) }
func (r fpRenderer) LineHeight(scale int) int          { return r.txt.CellH() * scale }

func (r fpRenderer) Clip(rc zenui.Rect) {
	rl.BeginScissorMode(int32(rc.X), int32(rc.Y), int32(rc.W), int32(rc.H))
}
func (r fpRenderer) ClipEnd() { rl.EndScissorMode() }

// fpInput builds a zenui.Input from raylib's current input state. Printable
// runes are drained from raylib's character queue; the logical keys the dialog
// cares about are edge-triggered via IsKeyPressed.
func fpInput() zenui.Input {
	in := zenui.Input{
		MouseX:       int(rl.GetMouseX()),
		MouseY:       int(rl.GetMouseY()),
		MouseDown:    rl.IsMouseButtonDown(rl.MouseLeftButton),
		MousePressed: rl.IsMouseButtonPressed(rl.MouseLeftButton),
		WheelY:       rl.GetMouseWheelMove(),
	}
	for {
		ch := rl.GetCharPressed()
		if ch == 0 {
			break
		}
		in.Chars = append(in.Chars, ch)
	}
	keymap := []struct {
		rlKey int32
		k     zenui.Key
	}{
		{rl.KeyEnter, zenui.KeyEnter},
		{rl.KeyKpEnter, zenui.KeyEnter},
		{rl.KeyEscape, zenui.KeyEscape},
		{rl.KeyBackspace, zenui.KeyBackspace},
		{rl.KeyUp, zenui.KeyUp},
		{rl.KeyDown, zenui.KeyDown},
		{rl.KeyPageUp, zenui.KeyPageUp},
		{rl.KeyPageDown, zenui.KeyPageDown},
		{rl.KeyTab, zenui.KeyTab},
	}
	for _, m := range keymap {
		if rl.IsKeyPressed(m.rlKey) {
			in.Keys = append(in.Keys, m.k)
		}
	}
	return in
}

// fpTheme maps the editor's palette onto the dialog theme so the dialog matches
// zenimate's look.
func fpTheme() zenui.Theme {
	cv := func(c rl.Color) zenui.Colour { return zenui.Colour{R: c.R, G: c.G, B: c.B, A: c.A} }
	t := zenui.DefaultTheme()
	t.Panel = cv(colBtn)
	t.Sidebar = cv(colBG)
	t.SideText = cv(colText)
	t.Border = cv(colVPBorder)
	t.Text = cv(colText)
	t.DimText = cv(colDim)
	t.DirText = cv(colYellow)
	t.Button = cv(colBtn)
	t.ButtonHot = cv(colBtnHot)
	t.ButtonText = cv(colText)
	return t
}
