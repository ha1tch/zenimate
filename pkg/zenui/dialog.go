package zenui

import "strings"

// Mode selects the dialog's behaviour.
type Mode int

const (
	// ModeOpen picks an existing file; the action button is disabled unless a
	// file (not a directory) is selected.
	ModeOpen Mode = iota
	// ModeSave names a file to write; the action button is enabled whenever the
	// filename field is non-empty. An existing name may be picked to overwrite.
	ModeSave
)

// EntryPreview is a small rendered thumbnail for an entry, supplied by the host
// via DialogConfig.Preview. Pixels is row-major, length W*H; a fully transparent
// pixel (A==0) is not drawn (the panel background shows through).
type EntryPreview struct {
	W, H   int
	Pixels []Colour
}

// DialogConfig sets up a dialog. Only FS is required; the rest have sensible
// defaults.
type DialogConfig struct {
	Mode       Mode
	Title      string   // window title; defaults to "Open"/"Save"
	StartDir   string   // initial directory; defaults to FS.UserHome()
	Filters    []string // extensions to show (e.g. {"zcut","scr"}); empty = all
	DefaultExt string   // appended on save if the typed name has no extension
	FS         FS       // filesystem; defaults to OSFS
	// Preview, if set, is called for the selected entry to obtain a thumbnail for
	// the preview pane. container is the path of the container the entry lives in
	// (empty for a plain directory entry). Returning nil means "no thumbnail"
	// (text metadata, if any, is still shown).
	Preview func(container string, entry Entry) *EntryPreview
	// StartContainer, if set, opens the dialog directly inside this container file
	// (e.g. a dropped .zbun), listing its entries; ".." then lands in the
	// container's directory.
	StartContainer string
}

// Status is the dialog's lifecycle state.
type Status int

const (
	// Active: the dialog is open and should keep being drawn.
	Active Status = iota
	// Accepted: the user confirmed a choice; read Result for the path.
	Accepted
	// Cancelled: the user dismissed the dialog.
	Cancelled
)

// Dialog is a file Open/Save dialog. Construct with New, then each frame call
// Update(input) and Draw(renderer). When Update returns a status other than
// Active the dialog is finished and Result holds the chosen path (for Accepted).
type Dialog struct {
	cfg DialogConfig
	fs  FS

	dir       string  // current directory (absolute)
	container string  // if non-empty, we are browsing inside this container file
	entries   []Entry // filtered, sorted listing of dir (or container)
	sel       int     // index into entries, or -1
	scroll    int     // first visible row
	name      string  // filename field contents (Save mode, and echo of selection)
	err       string  // last filesystem error, shown in the status line
	places    []Place // favourites shown in the sidebar

	status Status
	result string

	// layout caches from the last Draw, used by Update's hit-testing.
	bounds      Rect
	sideRect    Rect
	placeRects  []Rect
	listRect    Rect
	previewRect Rect
	rowH        int
	nameRect    Rect
	okRect      Rect
	cancRect    Rect
	upRect      Rect
}

// NewDialog creates a dialog from cfg, applying defaults and loading the
// start directory. It never returns nil; a bad start directory leaves the
// dialog at the filesystem root-ish home with an error message visible.
func NewDialog(cfg DialogConfig) *Dialog {
	if cfg.FS == nil {
		cfg.FS = OSFS{}
	}
	if cfg.Title == "" {
		if cfg.Mode == ModeSave {
			cfg.Title = "Save"
		} else {
			cfg.Title = "Open"
		}
	}
	d := &Dialog{cfg: cfg, fs: cfg.FS, sel: -1}
	d.places = d.fs.Places()
	start := cfg.StartDir
	if start == "" {
		start = d.fs.UserHome()
	}
	if abs, err := d.fs.Abs(start); err == nil {
		start = abs
	}
	d.setDir(start)
	// Optionally descend straight into a container (e.g. a dropped bundle).
	if cfg.StartContainer != "" {
		if abs, err := d.fs.Abs(cfg.StartContainer); err == nil {
			d.dir = d.fs.Dir(abs)
			d.setContainer(abs)
		}
	}
	return d
}

// Result returns the chosen path (valid once Status() == Accepted).
func (d *Dialog) Result() string { return d.result }

// Status returns the dialog's current lifecycle state.
func (d *Dialog) Status() Status { return d.status }

// Dir returns the directory currently being browsed.
func (d *Dialog) Dir() string { return d.dir }

// setDir loads dir's listing (filtered and sorted) and resets selection/scroll.
func (d *Dialog) setDir(dir string) {
	es, err := d.fs.ReadDir(dir)
	if err != nil {
		d.err = err.Error()
		return
	}
	d.err = ""
	d.dir = dir
	d.container = "" // leaving any container
	filtered := es[:0]
	for _, e := range es {
		// Containers are shown regardless of the filename filter so the user can
		// always descend into them; directories likewise.
		if e.IsDir || e.IsContainer || matchFilter(e.Name, d.cfg.Filters) {
			filtered = append(filtered, e)
		}
	}
	d.entries = sortEntries(filtered)
	d.sel = -1
	d.scroll = 0
}

// setContainer enters a container file (e.g. a .zbun), listing its entries. The
// directory context (d.dir) is retained so goUp can climb back out.
func (d *Dialog) setContainer(path string) {
	es, err := d.fs.ReadContainer(path)
	if err != nil {
		d.err = err.Error()
		return
	}
	d.err = ""
	d.container = path
	d.entries = sortEntries(es)
	d.sel = -1
	d.scroll = 0
}

// inContainer reports whether the dialog is currently browsing inside a
// container file rather than a directory.
func (d *Dialog) inContainer() bool { return d.container != "" }

// canAccept reports whether the action button is currently enabled.
func (d *Dialog) canAccept() bool {
	if d.cfg.Mode == ModeSave {
		return strings.TrimSpace(d.name) != ""
	}
	// Open: need a selected entry that is not a directory or a container (those
	// are for navigating into, not opening directly).
	if d.sel < 0 || d.sel >= len(d.entries) {
		return false
	}
	e := d.entries[d.sel]
	return !e.IsDir && !e.IsContainer
}

// accept finalises the dialog with the chosen path. Inside a container the
// result is "<containerpath>#<entryname>" so the caller can locate the entry
// within the archive; otherwise it is a normal filesystem path.
func (d *Dialog) accept() {
	name := strings.TrimSpace(d.name)
	if d.cfg.Mode == ModeOpen {
		if d.sel < 0 || d.sel >= len(d.entries) {
			return
		}
		e := d.entries[d.sel]
		if e.IsDir || e.IsContainer {
			return
		}
		name = e.Name
	}
	if name == "" {
		return
	}
	if d.inContainer() {
		d.result = d.container + "#" + name
		d.status = Accepted
		return
	}
	if d.cfg.Mode == ModeSave && d.cfg.DefaultExt != "" && !hasExt(name) {
		name = name + "." + strings.TrimPrefix(d.cfg.DefaultExt, ".")
	}
	d.result = d.fs.Join(d.dir, name)
	d.status = Accepted
}

// openSelection enters a directory or container, or accepts a file.
func (d *Dialog) openSelection() {
	if d.sel < 0 || d.sel >= len(d.entries) {
		return
	}
	e := d.entries[d.sel]
	switch {
	case e.IsContainer && !d.inContainer():
		d.setContainer(d.fs.Join(d.dir, e.Name))
		return
	case e.IsDir:
		d.setDir(d.fs.Join(d.dir, e.Name))
		return
	}
	if d.cfg.Mode == ModeOpen {
		d.accept()
	} else {
		d.name = e.Name
	}
}

// goUp exits a container back to its directory, or navigates to the parent
// directory when already at directory level.
func (d *Dialog) goUp() {
	if d.inContainer() {
		d.setDir(d.dir)
		return
	}
	d.setDir(d.fs.Dir(d.dir))
}

func hasExt(name string) bool {
	dot := strings.LastIndexByte(name, '.')
	slash := strings.LastIndexByte(name, '/')
	return dot > slash && dot < len(name)-1
}
