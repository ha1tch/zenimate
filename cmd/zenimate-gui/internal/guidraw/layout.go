// Package guidraw holds the zenimate-gui frontend's screen-layout data types
// and (eventually) its drawing logic. Layout and Button live here rather than
// in package main because they need to be shared between computeLayout
// (which stays in main — its button actions are closures bound to the
// Controller and other main-package state, which a sub-package cannot reach
// back into) and the drawing code, which only needs the resulting data, not
// the actions.
package guidraw

import rl "github.com/gen2brain/raylib-go/raylib"

// Layout is every computed screen position and size the zenimate-gui frontend
// needs for one frame: computeLayout (in package main) builds one fresh each
// frame from the window size and current state.
type Layout struct {
	WinW, WinH int

	// Title block (top-left). When collapsed it shrinks to a small button; the
	// toolbars expand to fill the reclaimed width and the viewport gains vertical
	// room. TitleRect is the click target that toggles collapse.
	TitleRect      rl.Rectangle
	TitleCollapsed bool    // true past the halfway point of the collapse
	TitleCollapse  float32 // eased collapse progress (0 = expanded, 1 = collapsed)

	// Horizontal frame strip near the top, one rect per current frame, with
	// +/- buttons to its right.
	FrameStripX, FrameStripY int
	ScrubRect                rl.Rectangle // frame scrubber slider above the strip
	FrameRects               []rl.Rectangle
	AddFrameRect             rl.Rectangle
	HelpRect                 rl.Rectangle

	// Editor grid.
	GridX, GridY int
	GridW, GridH int // the box the grid is clipped to (base fit size)
	Cell         int // adaptive on-screen pixel size of one Spectrum cell

	// View-mode buttons (Bitmap White / Bitmap Black / Spectrum Colour).
	ModeButtons []Button

	// Tiny LED toggles centred below the Bitmap White / Bitmap Black buttons that
	// switch the transparency chequer on/off for that mode.
	ChkLedWhite rl.Rectangle
	ChkLedBlack rl.Rectangle

	// Onion-skin toggle buttons (bitmap modes only).
	OnionButtons []Button

	// Attribute palette (shown in Spectrum Colour mode): owned by a
	// zenui.ZXClassicPaletteChooser; only the anchor position is needed here,
	// for the preview box's height calculation and the "PALETTE" label.
	PaletteX, PaletteY int

	// Tool palette, tucked between the preview box and the attribute palette:
	// owned by a zenui.ToolPalette; only the anchor is needed here, for the
	// same reason as PaletteX/Y above.
	ToolPaletteX, ToolPaletteY int

	// Preview (top-right corner) — fixed size, independent of sprite dimensions.
	PreviewX, PreviewY int
	PreviewW, PreviewH int

	// Action buttons across the bottom, in a sliding drawer.
	Buttons         []Button
	StripBtnW       int          // effective strip button width (shrinks in narrow windows)
	DrawerToggle    rl.Rectangle // small triangle that opens/closes the drawer
	DrawerToggleHit rl.Rectangle // larger clickable area around the triangle
	DrawerOpen      float32      // 0 = closed, 1 = open (current eased progress)
}

// Button is one clickable, labelled rectangle. Action is a closure supplied
// by package main at construction time — Button itself has no idea what it
// does, only where it is and what it's called, matching the same separation
// of "where" from "what happens" used throughout this codebase's zenui
// widgets.
type Button struct {
	X, Y, W, H int
	Label      string
	Action     func()
}

// Hit reports whether (mx,my) falls inside the button.
func (b Button) Hit(mx, my int) bool {
	return mx >= b.X && mx < b.X+b.W && my >= b.Y && my < b.Y+b.H
}
