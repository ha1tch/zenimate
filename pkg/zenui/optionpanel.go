package zenui

// OptionPanelConfig sets up a centred option-list modal: a title, an optional
// subtitle, a list of options, and an implicit Cancel row. This is the shape
// shared by every "pick one of a few things before proceeding" dialog — an
// export format, a fit strategy, a bundle create/add decision, and so on.
type OptionPanelConfig struct {
	Title    string // drawn at the panel's title scale
	Subtitle string // optional; if empty, no subtitle line is reserved
	Options  []Item // reuses Item from menu.go (Label, Disabled)
}

// OptionPanel is a centred modal offering a small list of choices plus
// Cancel. Construct with NewOptionPanel, then each frame call
// Draw(renderer, ...) followed by Update(input) — the same convention as
// Dialog and Menu. Reuses the package's shared Status: Active while open,
// Accepted once an option is picked (Result holds its index), Cancelled on
// Escape, a click on Cancel, or a click outside the panel.
type OptionPanel struct {
	cfg    OptionPanelConfig
	hover  int // index of the option under the pointer this frame, or -1 (-2 = Cancel)
	status Status
	result int

	// layout cache from the last Draw, used by Update's hit-testing.
	panel      Rect
	itemRects  []Rect
	cancelRect Rect
}

// hoverCancel is the sentinel hover value meaning "the Cancel row", distinct
// from -1 ("nothing").
const hoverCancel = -2

// NewOptionPanel creates a panel from cfg. It never returns nil.
func NewOptionPanel(cfg OptionPanelConfig) *OptionPanel {
	return &OptionPanel{cfg: cfg, hover: -1, result: -1}
}

// Result returns the chosen option's index (valid once Status() == Accepted).
func (p *OptionPanel) Result() int { return p.result }

// Status returns the panel's current lifecycle state.
func (p *OptionPanel) Status() Status { return p.status }

func (p *OptionPanel) itemEnabled(i int) bool {
	return i >= 0 && i < len(p.cfg.Options) && !p.cfg.Options[i].Disabled
}

// ItemRect returns the screen rect of option i, valid after the most recent
// Draw. Hosts can use it to hit-test in their own tests, or to anchor
// additional UI (a tooltip, say) to a specific item. Returns the zero Rect
// for an out-of-range index.
func (p *OptionPanel) ItemRect(i int) Rect {
	if i < 0 || i >= len(p.itemRects) {
		return Rect{}
	}
	return p.itemRects[i]
}
