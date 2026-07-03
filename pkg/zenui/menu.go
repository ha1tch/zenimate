package zenui

// Item is one row in a Menu.
type Item struct {
	Label    string
	Disabled bool
}

// MenuConfig sets up a dropdown menu.
type MenuConfig struct {
	Items []Item
	// Anchor is the screen rect of the control that triggered the menu (for
	// example, the frame button that was right-clicked). The menu is
	// positioned just below it: its left edge aligns to Anchor's left edge if
	// there is room for the menu to the right; otherwise its right edge
	// aligns to Anchor's right edge, provided there is room to the left.
	Anchor Rect
}

// Menu is a dropdown context menu anchored to a triggering control. Construct
// with NewMenu, then each frame call Draw(renderer, ...) followed by
// Update(input) — Draw computes and caches the layout that Update hit-tests
// against, the same calling convention as Dialog.
//
// Menu reuses the package's Status type: Active while open, Accepted once an
// item is picked (Result holds its index), Cancelled on Escape or a click
// outside the menu.
type Menu struct {
	cfg      MenuConfig
	hover    int // index of the item under the pointer this frame, or -1
	selected int // keyboard-selected item, persists until moved, or -1
	status   Status
	result   int

	// layout cache from the last Draw, used by Update's hit-testing.
	bounds    Rect
	itemRects []Rect
}

// NewMenu creates a menu from cfg. It never returns nil.
func NewMenu(cfg MenuConfig) *Menu {
	return &Menu{cfg: cfg, hover: -1, selected: -1, result: -1}
}

// Result returns the chosen item's index (valid once Status() == Accepted).
func (m *Menu) Result() int { return m.result }

// Status returns the menu's current lifecycle state.
func (m *Menu) Status() Status { return m.status }

func (m *Menu) itemEnabled(i int) bool {
	return i >= 0 && i < len(m.cfg.Items) && !m.cfg.Items[i].Disabled
}

// ItemRect returns the screen rect of item i, valid after the most recent
// Draw. Returns the zero Rect for an out-of-range index.
func (m *Menu) ItemRect(i int) Rect {
	if i < 0 || i >= len(m.itemRects) {
		return Rect{}
	}
	return m.itemRects[i]
}
