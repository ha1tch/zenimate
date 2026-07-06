package main

import (
	"github.com/ha1tch/zenimate/internal/model"
	"github.com/ha1tch/zenimate/internal/ui"
	"github.com/ha1tch/zenimate/pkg/zenui"
)

// Frame context menu item indices, in display order.
const (
	frameMenuInsert = iota
	frameMenuDuplicate
	frameMenuCopy
	frameMenuPaste
	frameMenuInsertPaste
	frameMenuDelete
)

// newFrameMenu builds the right-click context menu for the frame at index i,
// anchored just below its button rect. Items are disabled to match the
// current frame-count bounds and clipboard state, so nothing offered here can
// fail when picked.
func newFrameMenu(c *ui.Controller, anchor zenui.Rect) *zenui.Menu {
	atMax := c.Sprite.FrameCount() >= model.MaxFrames
	atMin := c.Sprite.FrameCount() <= model.MinFrames
	noClip := !c.Sprite.HasClipboard()

	items := make([]zenui.Item, 6)
	items[frameMenuInsert] = zenui.Item{Label: "INSERT EMPTY FRAME", Disabled: atMax}
	items[frameMenuDuplicate] = zenui.Item{Label: "DUPLICATE FRAME", Disabled: atMax}
	items[frameMenuCopy] = zenui.Item{Label: "COPY FRAME"}
	items[frameMenuPaste] = zenui.Item{Label: "PASTE FRAME", Disabled: noClip}
	items[frameMenuInsertPaste] = zenui.Item{Label: "INSERT AND PASTE", Disabled: atMax || noClip}
	items[frameMenuDelete] = zenui.Item{Label: "DELETE FRAME", Disabled: atMin}

	return zenui.NewMenu(zenui.MenuConfig{Items: items, Anchor: anchor})
}

// applyFrameMenuPick runs the action for picked menu item idx against frame
// i. The frame was selected the moment the menu opened (see the right-click
// handler in main.go), and the menu is modal, so selection cannot have
// drifted by the time a pick is dispatched — Copy and Paste, which only ever
// act on the current selection, are safe to call directly without
// re-selecting first.
func applyFrameMenuPick(c *ui.Controller, idx, i int) {
	switch idx {
	case frameMenuInsert:
		c.Checkpoint()
		c.InsertFrameAfter(i)
	case frameMenuDuplicate:
		c.Checkpoint()
		c.DuplicateFrameAt(i)
	case frameMenuCopy:
		c.CopyFrame()
	case frameMenuPaste:
		c.Checkpoint()
		c.PasteFrame()
	case frameMenuInsertPaste:
		c.Checkpoint()
		c.InsertAndPasteAfter(i)
	case frameMenuDelete:
		c.Checkpoint()
		c.DeleteFrameAt(i)
	}
}
