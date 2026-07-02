package filepick

import (
	"strings"
	"testing"
)

// memFS is an in-memory filesystem for tests. Paths use "/" separators.
type memFS struct {
	dirs       map[string][]Entry // dir -> entries
	containers map[string][]Entry // container path -> inner entries
	home       string
}

func (m *memFS) ReadDir(dir string) ([]Entry, error) {
	es, ok := m.dirs[dir]
	if !ok {
		return nil, &notExist{dir}
	}
	out := make([]Entry, len(es))
	copy(out, es)
	return out, nil
}
func (m *memFS) Abs(p string) (string, error) { return p, nil }
func (m *memFS) Join(elem ...string) string {
	return strings.TrimRight(strings.Join(elem, "/"), "/")
}
func (m *memFS) Dir(p string) string {
	i := strings.LastIndexByte(p, '/')
	if i <= 0 {
		return "/"
	}
	return p[:i]
}
func (m *memFS) UserHome() string { return m.home }
func (m *memFS) Places() []Place {
	return []Place{
		{Label: "Home", Path: "/home/u"},
		{Label: "Art", Path: "/home/u/art"},
	}
}

// containers maps a path to its inner entries, for browse-into-container tests.
func (m *memFS) IsContainer(path string) bool {
	_, ok := m.containers[path]
	return ok
}
func (m *memFS) ReadContainer(path string) ([]Entry, error) {
	es, ok := m.containers[path]
	if !ok {
		return nil, &notExist{path}
	}
	out := make([]Entry, len(es))
	copy(out, es)
	return out, nil
}

type notExist struct{ p string }

func (e *notExist) Error() string { return "no such dir: " + e.p }

func sampleFS() *memFS {
	return &memFS{
		home: "/home/u",
		dirs: map[string][]Entry{
			"/home/u": {
				{Name: "art", IsDir: true},
				{Name: "hero.zcut", IsDir: false},
				{Name: "notes.txt", IsDir: false},
				{Name: "title.scr", IsDir: false},
				{Name: "game.zbun", IsDir: false, IsContainer: true},
			},
			"/home/u/art": {
				{Name: "enemy.zcut", IsDir: false},
			},
		},
		containers: map[string][]Entry{
			"/home/u/game.zbun": {
				{Name: "knight", Meta: []MetaLine{{"frames", "4"}, {"size", "24x16"}, {"label", "hero"}}},
				{Name: "goblin", Meta: []MetaLine{{"frames", "2"}, {"size", "16x16"}}},
			},
		},
	}
}

// noopRenderer satisfies Renderer with fixed metrics so layout/hit-testing work
// in tests without any real drawing.
type noopRenderer struct{}

func (noopRenderer) FillRect(Rect, Colour)                  {}
func (noopRenderer) StrokeRect(Rect, Colour, int)           {}
func (noopRenderer) DrawText(string, int, int, int, Colour) {}
func (noopRenderer) TextWidth(s string, scale int) int      { return len(s) * 8 * scale }
func (noopRenderer) LineHeight(scale int) int               { return 8 * scale }
func (noopRenderer) Clip(Rect)                              {}
func (noopRenderer) ClipEnd()                               {}

func TestOpenFiltersAndSorts(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", Filters: []string{"zcut"}, FS: sampleFS()})
	// Directories and containers are always shown; among plain files only .zcut
	// matches the filter. Sort order: dirs first, then the rest alphabetically —
	// so "art" (dir), then "game.zbun" (container), then "hero.zcut".
	var names []string
	for _, e := range d.entries {
		names = append(names, e.Name)
	}
	want := []string{"art", "game.zbun", "hero.zcut"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("entries = %v, want %v", names, want)
	}
}

func TestOpenSelectFileEnablesAccept(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	if d.canAccept() {
		t.Fatal("nothing selected: accept should be disabled")
	}
	// Select the .scr file (index depends on sort: art/, hero.zcut, notes.txt, title.scr).
	for i, e := range d.entries {
		if e.Name == "title.scr" {
			d.sel = i
		}
	}
	if !d.canAccept() {
		t.Fatal("file selected: accept should be enabled")
	}
	d.accept()
	if d.Status() != Accepted || !strings.HasSuffix(d.Result(), "/home/u/title.scr") {
		t.Errorf("accept result = %q status=%v", d.Result(), d.Status())
	}
}

func TestOpenSelectDirDisablesAccept(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	d.sel = 0 // "art/" directory
	if d.canAccept() {
		t.Error("a directory selected should not enable Open")
	}
	d.openSelection() // entering a dir
	if d.Dir() != "/home/u/art" {
		t.Errorf("did not enter dir, now at %q", d.Dir())
	}
}

func TestGoUp(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u/art", FS: sampleFS()})
	d.goUp()
	if d.Dir() != "/home/u" {
		t.Errorf("goUp -> %q, want /home/u", d.Dir())
	}
}

func TestSaveTypingAndDefaultExt(t *testing.T) {
	d := New(Config{Mode: ModeSave, StartDir: "/home/u", DefaultExt: "zcut", FS: sampleFS()})
	if d.canAccept() {
		t.Fatal("empty filename: save disabled")
	}
	// Type "boss" via the input path.
	d.Update(Input{Chars: []rune("boss")})
	if d.name != "boss" {
		t.Fatalf("name = %q, want boss", d.name)
	}
	if !d.canAccept() {
		t.Fatal("non-empty filename: save enabled")
	}
	d.accept()
	if !strings.HasSuffix(d.Result(), "/home/u/boss.zcut") {
		t.Errorf("save result = %q, want .../boss.zcut", d.Result())
	}
}

func TestSaveKeepsTypedExtension(t *testing.T) {
	d := New(Config{Mode: ModeSave, StartDir: "/home/u", DefaultExt: "zcut", FS: sampleFS()})
	d.name = "pic.scr"
	d.accept()
	if !strings.HasSuffix(d.Result(), "/home/u/pic.scr") {
		t.Errorf("should keep typed extension, got %q", d.Result())
	}
}

func TestEscapeCancels(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	if st := d.Update(Input{Keys: []Key{KeyEscape}}); st != Cancelled {
		t.Errorf("escape -> %v, want Cancelled", st)
	}
}

func TestArrowNavigationWraDirsFirst(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	d.Update(Input{Keys: []Key{KeyDown}}) // sel 0
	d.Update(Input{Keys: []Key{KeyDown}}) // sel 1
	if d.sel != 1 {
		t.Errorf("after two downs sel = %d, want 1", d.sel)
	}
	d.Update(Input{Keys: []Key{KeyUp}})
	if d.sel != 0 {
		t.Errorf("after up sel = %d, want 0", d.sel)
	}
}

func TestMouseClickSelectsAndOpensViaDraw(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	r := noopRenderer{}
	// Draw once to populate layout rects.
	d.Draw(r, 800, 600, DefaultTheme())
	// Click the first row.
	rowY := d.listRect.Y + d.rowH/2
	rowX := d.listRect.X + 10
	d.Update(Input{MouseX: rowX, MouseY: rowY, MousePressed: true})
	if d.sel != 0 {
		t.Fatalf("click did not select row 0, sel=%d", d.sel)
	}
	// Second click on the same row enters the directory (row 0 is "art/").
	d.Update(Input{MouseX: rowX, MouseY: rowY, MousePressed: true})
	if d.Dir() != "/home/u/art" {
		t.Errorf("double-click did not enter dir, at %q", d.Dir())
	}
}

func TestClickOutsideCancels(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	d.Draw(noopRenderer{}, 800, 600, DefaultTheme())
	// Click at (0,0), which is on the backdrop, outside the panel.
	if st := d.Update(Input{MouseX: 0, MouseY: 0, MousePressed: true}); st != Cancelled {
		t.Errorf("outside click -> %v, want Cancelled", st)
	}
}

func TestSidebarPlaceNavigates(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	d.Draw(noopRenderer{}, 800, 600, DefaultTheme())
	// Click the second place ("Art" -> /home/u/art).
	if len(d.placeRects) < 2 {
		t.Fatalf("expected >=2 place rects, got %d", len(d.placeRects))
	}
	pr := d.placeRects[1]
	d.Update(Input{MouseX: pr.X + 4, MouseY: pr.Y + 2, MousePressed: true})
	if d.Dir() != "/home/u/art" {
		t.Errorf("clicking 'Art' place -> %q, want /home/u/art", d.Dir())
	}
}

func TestSidebarClickDoesNotPaintList(t *testing.T) {
	// A click on a sidebar place must not also register as a file-list selection.
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	d.Draw(noopRenderer{}, 800, 600, DefaultTheme())
	pr := d.placeRects[0]
	d.Update(Input{MouseX: pr.X + 4, MouseY: pr.Y + 2, MousePressed: true})
	if d.sel != -1 {
		t.Errorf("sidebar click should not select a list row, sel=%d", d.sel)
	}
}

func TestEnterAndExitContainer(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	// Find and select the bundle, then open it.
	bi := -1
	for i, e := range d.entries {
		if e.Name == "game.zbun" {
			bi = i
		}
	}
	if bi < 0 {
		t.Fatal("bundle not listed")
	}
	d.sel = bi
	d.openSelection()
	if !d.inContainer() {
		t.Fatal("did not enter the container")
	}
	// Inside: should list the two animations.
	if len(d.entries) != 2 {
		t.Fatalf("container listing = %d entries, want 2", len(d.entries))
	}
	// Accepting an entry yields "container#name".
	d.sel = 0
	name := d.entries[0].Name
	d.accept()
	if d.Status() != Accepted {
		t.Fatal("accept inside container did not finalise")
	}
	want := "/home/u/game.zbun#" + name
	if d.Result() != want {
		t.Errorf("result = %q, want %q", d.Result(), want)
	}
}

func TestContainerGoUpExits(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	for i, e := range d.entries {
		if e.Name == "game.zbun" {
			d.sel = i
		}
	}
	d.openSelection()
	if !d.inContainer() {
		t.Fatal("not in container")
	}
	d.goUp()
	if d.inContainer() {
		t.Error("goUp should have exited the container")
	}
	if d.Dir() != "/home/u" {
		t.Errorf("after exit, dir = %q, want /home/u", d.Dir())
	}
}

func TestContainerNotAcceptedDirectly(t *testing.T) {
	// Selecting the container itself must not enable Open (it is for descending).
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	for i, e := range d.entries {
		if e.Name == "game.zbun" {
			d.sel = i
		}
	}
	if d.canAccept() {
		t.Error("a container should not be directly acceptable")
	}
}

func TestPreviewPaneLayoutInContainer(t *testing.T) {
	d := New(Config{Mode: ModeOpen, StartDir: "/home/u", FS: sampleFS()})
	// Enter the container and lay out; the preview rect should be non-empty.
	for i, e := range d.entries {
		if e.Name == "game.zbun" {
			d.sel = i
		}
	}
	d.openSelection()
	d.Draw(noopRenderer{}, 900, 640, DefaultTheme())
	if d.previewRect.W <= 0 {
		t.Error("preview pane should have width while inside a container")
	}
	// Outside a container there is no preview pane.
	d.goUp()
	d.Draw(noopRenderer{}, 900, 640, DefaultTheme())
	if d.previewRect.W != 0 {
		t.Error("preview pane should be absent outside a container")
	}
}
