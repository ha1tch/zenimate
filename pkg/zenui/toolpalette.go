package zenui

// Tool is one entry in a ToolPalette: an identifying ID the host uses to
// know which tool was picked, and the glyph rune to draw as its icon —
// looked up in whatever font the host's Renderer draws through. ToolPalette
// never interprets the glyph itself, matching the same separation used by
// PreviewPane's ImageSource and ZXClassicPaletteChooser's colour lookups:
// the widget draws, the host supplies meaning.
type Tool struct {
	ID    string
	Glyph rune
}

// ToolPaletteConfig sets up a wrapping grid of icon buttons, one per Tool.
// Unlike ZXClassicPaletteChooser's fixed 4x4 layout (which exists because ZX
// Spectrum attributes are always exactly 16 fixed colours), the grid here is
// derived from len(Tools) and Cols — adding or removing a tool changes the
// grid automatically, with no new layout arithmetic anywhere else.
type ToolPaletteConfig struct {
	Anchor           Rect // top-left corner the grid is laid out from
	Tools            []Tool
	Cols             int
	ButtonW, ButtonH int
	GapX, GapY       int
	// GlyphSize is the icon's actual visual pixel size (e.g. 24), used to
	// centre it within its button. Deliberately not derived from the
	// renderer's TextWidth/LineHeight: those reflect the font's full cell
	// box, which for a compact icon face is typically larger than the
	// icon's own ink (see icons.bdf, where every glyph — 24px or 32px — sits
	// at BBX offset 0,0 inside a fixed 32x32 cell). Centring against the
	// cell would visually shift every icon toward its button's top-left
	// corner; centring against the true glyph size does not.
	GlyphSize int
}

// toolButton is one laid-out tool: its screen rect and the Tool it shows.
type toolButton struct {
	rect Rect
	tool Tool
}

// ToolPalette is a persistent tool strip, not a modal — like
// ZXClassicPaletteChooser, it has no Status lifecycle, just per-frame
// Update/Draw. Unlike ZXClassicPaletteChooser, it owns "currently selected
// tool" as its own state: there is no existing host-side concept of an
// active tool for it to defer to (contrast the palette, where ink/paper
// selection already lives on the Controller).
type ToolPalette struct {
	cfg      ToolPaletteConfig
	buttons  []toolButton
	selected int // index into buttons; -1 if Tools is empty
	hover    int
}

// NewToolPalette creates a palette from cfg, selecting the first tool by
// default (or none, if Tools is empty). It never returns nil.
func NewToolPalette(cfg ToolPaletteConfig) *ToolPalette {
	p := &ToolPalette{cfg: cfg, hover: -1, selected: -1}
	if len(cfg.Tools) > 0 {
		p.selected = 0
	}
	p.layout()
	return p
}

// layout computes each button's rect from the anchor, grid dimensions, and
// column count, wrapping to a new row every Cols buttons.
func (p *ToolPalette) layout() {
	cols := p.cfg.Cols
	if cols < 1 {
		cols = 1
	}
	p.buttons = make([]toolButton, len(p.cfg.Tools))
	for i, tool := range p.cfg.Tools {
		row, col := i/cols, i%cols
		x := p.cfg.Anchor.X + col*(p.cfg.ButtonW+p.cfg.GapX)
		y := p.cfg.Anchor.Y + row*(p.cfg.ButtonH+p.cfg.GapY)
		p.buttons[i] = toolButton{
			rect: Rect{X: x, Y: y, W: p.cfg.ButtonW, H: p.cfg.ButtonH},
			tool: tool,
		}
	}
}

// SetBounds repositions the grid (e.g. on window resize) without needing a
// full reconstruction.
func (p *ToolPalette) SetBounds(anchor Rect) {
	p.cfg.Anchor = anchor
	p.layout()
}

// Bounds returns the grid's overall bounding rect, accounting for however
// many rows len(Tools) and Cols produce.
func (p *ToolPalette) Bounds() Rect {
	if len(p.cfg.Tools) == 0 {
		return Rect{X: p.cfg.Anchor.X, Y: p.cfg.Anchor.Y}
	}
	cols := p.cfg.Cols
	if cols < 1 {
		cols = 1
	}
	rows := (len(p.cfg.Tools) + cols - 1) / cols
	return Rect{
		X: p.cfg.Anchor.X, Y: p.cfg.Anchor.Y,
		W: cols*p.cfg.ButtonW + (cols-1)*p.cfg.GapX,
		H: rows*p.cfg.ButtonH + (rows-1)*p.cfg.GapY,
	}
}

// RectFor returns the screen rect of the button for the given tool ID, for
// anchoring a popup panel to a specific button rather than the click point.
// ok is false if no tool with that ID is configured.
func (p *ToolPalette) RectFor(id string) (rect Rect, ok bool) {
	for _, b := range p.buttons {
		if b.tool.ID == id {
			return b.rect, true
		}
	}
	return Rect{}, false
}

// HitTest reports which tool's button (if any) contains (mx,my), without
// side effects — unlike Update, it doesn't change hover or selection state.
// For interactions Update doesn't cover, like a right-click options panel
// anchored to a specific button.
func (p *ToolPalette) HitTest(mx, my int) (id string, ok bool) {
	for _, b := range p.buttons {
		if b.rect.Contains(mx, my) {
			return b.tool.ID, true
		}
	}
	return "", false
}

// SetSelected sets the currently selected tool by ID, for restoring a prior
// choice (e.g. when a panel holding this palette is reopened) rather than
// always resetting to the first tool. A no-op if id doesn't match any tool.
func (p *ToolPalette) SetSelected(id string) {
	for i, b := range p.buttons {
		if b.tool.ID == id {
			p.selected = i
			return
		}
	}
}

// Selected returns the currently selected tool's ID. ok is false only when
// no tools are configured.
func (p *ToolPalette) Selected() (id string, ok bool) {
	if p.selected < 0 || p.selected >= len(p.buttons) {
		return "", false
	}
	return p.buttons[p.selected].tool.ID, true
}

// ToolPickResult reports a pick from Update.
type ToolPickResult struct {
	Picked bool
	ID     string
}

// Update hit-tests the input snapshot: a left click selects that tool
// (updating the persistent selection) and reports the pick to the host, so
// the host can wire up whatever the tool actually does. Also updates the
// hover index Draw uses to highlight the button under the pointer.
func (p *ToolPalette) Update(in Input) ToolPickResult {
	p.hover = -1
	for i, b := range p.buttons {
		if b.rect.Contains(in.MouseX, in.MouseY) {
			p.hover = i
			break
		}
	}
	if p.hover < 0 || !in.MousePressed {
		return ToolPickResult{}
	}
	p.selected = p.hover
	return ToolPickResult{Picked: true, ID: p.buttons[p.hover].tool.ID}
}

// Draw renders every tool button: background (selected fill takes priority
// over hover), border, and the tool's glyph centred inside using GlyphSize
// (see ToolPaletteConfig's doc comment for why not the renderer's own
// TextWidth/LineHeight).
func (p *ToolPalette) Draw(r Renderer, theme Theme) {
	for i, b := range p.buttons {
		bg := theme.Button
		switch {
		case i == p.selected:
			bg = theme.SelFill
		case i == p.hover:
			bg = theme.ButtonHot
		}
		r.FillRect(b.rect, bg)
		r.StrokeRect(b.rect, theme.Border, 1)

		glyph := string(b.tool.Glyph)
		gx := b.rect.X + (b.rect.W-p.cfg.GlyphSize)/2
		gy := b.rect.Y + (b.rect.H-p.cfg.GlyphSize)/2
		r.DrawText(glyph, gx, gy, 1, theme.ButtonText)
	}
}
