# zenimate — intro guide

A fuller tour of zenimate than the README: how to build it, how to drive both
frontends, the file formats it reads and writes, and how the code is layered. For
a one-paragraph overview and the quick build, see the [README](../README.md).

## Building

```
make            # show all targets
make build      # both frontends into dist/
make build-tui  # terminal frontend only
make build-gui  # raylib frontend, cgo-free (purego) — no system GL headers needed
```

The GUI links raylib. The default `build-gui` target uses raylib-go's **purego**
path, which is cgo-free and builds anywhere. For the normal desktop runtime path
you can instead build with cgo:

```
make build-gui-cgo
```

which needs the system development libraries raylib expects. On Debian/Ubuntu:

```
sudo apt-get install libgl1-mesa-dev libx11-dev libxcursor-dev libxrandr-dev \
                     libxinerama-dev libxi-dev libxxf86vm-dev libwayland-dev \
                     libxkbcommon-dev
```

Requirements: Go 1.25 or later. The GUI at runtime needs an OpenGL-capable
display; building it does not.

## Controls

### TUI

The TUI mirrors the GUI's three view modes and edits the bitmap plane while being
colour-aware. It displays attributes and can stamp colour, but never destroys
colour unintentionally: clearing pixels keeps colour, and the colour-wiping clear
and reset ask for confirmation.

| Key | Action |
|-----|--------|
| arrows / `hjkl` | move cursor |
| `tab` | cycle view mode (Bitmap Black / White / Spectrum Colour) |
| space | set pixel (draw); in colour-paint mode, stamp ink+paper |
| backspace / del | clear pixel (erase) |
| enter | toggle pixel |
| `i` / `o` | decrease / increase selected ink colour (0–7) |
| `I` / `O` | decrease / increase selected paper colour (0–7) |
| `b` | toggle bright for painting |
| `m` | toggle colour-paint mode |
| `[` `]` | previous / next frame |
| `1`..`9` | jump to frame |
| `+` / `-` | add / remove a frame (1–16) |
| `p` | play / stop animation |
| `c` / `v` | copy / paste frame |
| `x` | clear pixels (keeps colour) |
| `X` | clear pixels **and** colour (asks to confirm) |
| `R` | reset all frames (asks to confirm) |
| `w` `W` / `t` `T` | shrink/grow width, shrink/grow height by one cell |
| `S` / `L` | save to / load from a `.zani` file (prompts for a name) |
| `q` | quit |

### GUI

Left-drag paints, right-drag erases. Panning uses **space as a modifier** (hold
space and left-drag) or a **middle-mouse-button drag**; either pans the viewport.
The **mouse wheel zooms** in and out toward the cursor. Both zoom and pan carry
glide-after-release inertia. **Enter** toggles play/stop; `[` `]` and `1`..`9`
change frame. Esc does not close the window.

The frame strip runs along the top, with a **scrubber slider** above it (drag to
move between frames) and **`-` / `+` buttons** to add or remove frames (1–16). The
bottom buttons include **32x24** (the full 32×24-cell screen) and **2x2** size
presets, per-cell **width/height steppers** (±1 cell each way), reset, play/stop,
and separate **copy** and **paste** frame buttons. Sizing is **non-destructive** —
existing pixels and attributes are preserved — and the viewport animates to fit
the whole sprite when the size changes. Sprites can be any size up to 32×24
character cells (256×192 px).

Three **view modes** sit in a row under the frame strip:

- **Bitmap Black** — set pixels drawn black; clear pixels show the transparency
  chequer (one chequer square per virtual pixel).
- **Bitmap White** — set pixels drawn white; clear pixels show the chequer.
- **Spectrum Colour** — uses ZX attributes: set pixels show the cell's ink
  colour, clear pixels its paper colour. A vertical **16-swatch palette** (8
  colours × normal/bright) appears: **left-click** a swatch to pick ink,
  **right-click** to pick paper. **Hold Ctrl and paint** to stamp the current
  ink/paper/bright onto a cell (painting without Ctrl only edits the bitmap). A
  faint per-virtual-pixel grid shows at ≥50% zoom. Attributes are per 8×8
  character cell (hardware-accurate) and **per frame**; copy/paste-frame carries
  them.

In the bitmap views, two **onion-skin** toggles overlay neighbouring frames as
translucent silhouettes: the previous frame in red, the next in green (wrapping
at the ends). Each toggles independently; onion skins are not shown in Spectrum
Colour mode.

The frame strip and the view-mode/onion toolbars sit to the right of the title
block, across the top, using the available width. Clicking the title collapses
it to a small button (freeing horizontal space for the toolbars); clicking the
button restores it. Keeping the toolbars on the top band leaves the viewport
more vertical room.

The window is resizable: the editor cell size adapts so the whole grid stays
visible at any window or sprite size, with dark-grey guide lines on the ZX 8×8
character-cell boundaries.

The **preview** is a fixed-size box (top-right) that shows the sprite at a fixed
integer zoom (×1–×4, where ×N draws each sprite pixel as an N×N square),
centred on the cursor's pixel while you hover the paint area, otherwise on the
last pixel you edited. **Right-click** the preview to cycle the zoom. **Press and
hold** (left button) to unfurl a full-sprite popup that grows out of the preview
box — anchored to its top-right corner — and shrinks back on release.

File operations sit on the bottom button row and on shortcuts: **Ctrl+O** opens
(a sprite, animation, screen, image, or a bundle to browse into); **Ctrl+S**
saves back to the current source; **Ctrl+Shift+S** is Save As; **Ctrl+E** exports
the current frame to a Spectrum format; **Ctrl+Shift+E** adds the sprite to a
bundle; **Ctrl+F** toggles the save-extension form between long (`.zani`/`.zbun`)
and 8.3 (`.zan`/`.zbu`). Files can also be **dragged onto the window**: a sprite,
animation, screen or image loads directly, and a dropped bundle opens the browser
so you can pick an animation. The title block shows the current source
(`name.zani`, `name - bundle.zbun`, or `name (unsaved)`) so it is always clear
what Save will write to.

The **HELP** button (or **F1**) opens a scrollable reader listing all the
shortcuts and explaining the file formats; scroll with the wheel, arrows,
PgUp/PgDn or Home/End, and close it with Esc or the close box.

## Files and formats

zenimate reads and writes these file types:

- **`.zani`** (or **`.zan`** on 8.3 filesystems) — a single animated sprite: all
  of its frames, with per-cell attributes. The bytes are the ZCUT format, so a
  `.zani` is interchangeable with the wider toolchain and a raw `.zcut` is still
  accepted on open.
- **`.zbun`** (or **`.zbu`** on 8.3) — a bundle: a zip gathering several `.zani`
  animations plus a `manifest.json` index (name, frame count, dimensions and a
  free-text label per animation). Use a bundle to keep, say, all of a game's
  sprites in one file. The Open dialog can browse into a bundle and preview each
  animation before opening it; the Bundle action adds the current sprite to a new
  or existing bundle.

Which extension form is written — long (`.zani`/`.zbun`) or 8.3 (`.zan`/`.zbu`) —
is a setting, toggled with **Ctrl+F**. Loading always accepts every form.

zenimate can also **import** a raw screen (`.scr`), pull a loading screen out of
a tape (`.tap`/`.tzx`) or snapshot (`.sna`/`.z80`), and reduce a raster image
(JPEG/PNG/GIF) to a Spectrum screen with a chosen fit strategy. Any of these can
be opened from the dialog or dropped onto the window.

**Save vs Save As.** **Ctrl+S** saves back to wherever the sprite came from: a
file is overwritten in place; a sprite opened from a bundle asks (once) whether
to update it inside that bundle or split it off as a separate `.zani`; a brand
new sprite prompts for a destination. **Ctrl+Shift+S** is Save As and always
prompts. The title block shows the current source so it is always clear what Save
will write to.

### zaniplay

`cmd/zaniplay` is a standalone terminal player for animations. It renders in
colour using half-block characters:

```
zaniplay [-fps N] [-once] file.zani
zaniplay [-fps N] [-once] game.zbun#knight   # one animation from a bundle
```

## Design

The pieces are layered so the two frontends share everything but presentation:

```
pkg/bdf          standalone BDF font reader + rasteriser (no project deps)
pkg/zxpalette    ZX Spectrum palette + attribute encoding (cloned from zenzx)
pkg/filepick     renderer-agnostic file Open/Save dialog widget (stdlib only)
pkg/version      build version, synced from VERSION
internal/fonts   embedded Sinclair + Cozette faces (decoded via pkg/bdf)
internal/model   the sprite document: variable frames, dims, per-frame
                 per-cell attributes, edits, observer hook
internal/ui      the frontend-independent controller
cmd/zenimate-tui terminal frontend
cmd/zenimate-gui raylib frontend (text via pkg/bdf only)
cmd/zaniplay     standalone terminal animation player
```

`pkg/bdf` is the reusable BDF font system: it was lifted from the subterm
terminal renderer and stripped of its buffer/registry coupling so it stands
alone. The GUI proves the "reuse the BDF system" intent end to end — it asks
`pkg/bdf` to rasterise each glyph of the Sinclair face, uploads those pixmaps as
raylib textures, and blits strings cell by cell.

## Continuous integration and releases

GitHub Actions builds, vets, and tests on every push and pull request. Because
the GUI uses raylib-go's cgo-free purego path, every binary cross-compiles from a
single Linux runner with no per-platform toolchains.

Pushing a `v*` tag (for example `v0.6.0`) triggers the release workflow, which
cross-compiles all three frontends (`zenimate-gui`, `zenimate-tui`, `zaniplay`)
for each platform, bundles them per platform, and publishes a GitHub Release with
the archives attached. Release platforms:

- Linux amd64 and arm64 (arm64 covers 64-bit Raspberry Pi OS on the Pi 3B and
  later; the purego GUI has no 32-bit ARM support)
- macOS amd64 (Intel) and arm64 (Apple Silicon)
- Windows amd64

Dependencies are ordinary published modules fetched from the Go proxy at build
time (the file-dialog widget lives in-tree at `pkg/filepick`), so CI needs no
vendored tree. The release version is taken from the tag and written through the
`VERSION` file and `scripts/syncver.sh`, keeping the compiled-in version
consistent with the tag.

Note: the purego GUI loads the raylib shared library at runtime, so a target
machine needs libraylib available; the TUI and `zaniplay` have no such
dependency.
