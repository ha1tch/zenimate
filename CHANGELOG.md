# Changelog

All notable changes to zenimate are documented here.
The format follows Keep a Changelog; versions follow semantic versioning.

## [0.6.0]

A window- and screen-aware zoom model, frame transforms, a safer reset, and a
range of GUI refinements including a built-in help reader.

### Added
- An overlay marks set pixels inside "flat" cells — cells whose ink and paper are
  the same colour, so set and unset pixels look identical in Spectrum Colour mode.
  Each hidden set pixel gets a thin contrasting inner stroke so it can be seen and
  edited. It fades in as you zoom in (see the zoom model below). These cells are
  common after image import and can't be avoided algorithmically.
- The attribute palette fades in and out when switching to or from Spectrum
  Colour mode, rather than appearing and vanishing abruptly.
- RESET now asks for confirmation before wiping the whole animation: a modal
  requires typing YES (Enter to confirm, Esc to cancel), and it restores the full
  default state — default size, default frame count, everything cleared.
- The old Reset button is split into RESET and CLS. CLS clears only the current
  frame and needs no confirmation.
- The Save-to-bundle dialog is titled "Save to bundle" so it reads clearly as a
  save.
- A transform button group for the selected frame: H FLIP, V FLIP, ROT 90, and
  INVERT. Flips mirror both pixels and colour; INVERT flips the bitmap only. ROT
  90 rotates clockwise in place; holding Ctrl also resizes a non-square frame
  (swapping width and height) so nothing is clipped.
- A small LED toggle below each bitmap-mode button turns that mode's transparency
  chequer on or off. With the chequer off, the empty area is a solid shade — the
  darkest chequer tone in Bitmap White, the lightest in Bitmap Black.
- Strip buttons animate in and out on window resize: when a button no longer fits
  beside the viewport it fades away over a fixed time (and back when it fits
  again), so the transition plays fully even on platforms that report a resize
  only once it has finished. Faded-out buttons stop responding to clicks.
- The dark character-cell gridlines and the Spectrum-mode 1px grid now fade with
  zoom rather than cutting off abruptly, each fading in over its own zoom range and
  back out as you zoom away. Their thresholds are expressed in the zoom-percentage
  scale described under the zoom model below.
- A thin frame-scrubber slider above the frame buttons: drag it to move between
  frames.
- COPY and PASTE are separate frame buttons, with EXPORT directly beneath COPY and
  BUNDLE beneath PASTE at matching widths and positions.
- Sizing controls reworked: a "32x24" preset (full screen) and a new "2x2" preset,
  both compact, beside the width/height steppers, which are now labelled in cell
  units ("W -1/+1", "H -1/+1") rather than pixels.
- The bottom button bar shrinks its buttons proportionally in a narrow window, and
  wraps button labels onto two lines when the buttons get small, so it stays
  usable instead of overflowing.
- The onion-skin buttons now stay fixed at their startup position instead of
  tracking the F6 frame button, so adding frames no longer shifts them.
- Fast strokes are interpolated: a straight line is filled between sampled points
  so quick drawing no longer leaves sparse dotted gaps.
- The HELP button now stays fixed at its startup position instead of tracking the
  frame strip, so adding frames no longer shifts it.
- Loading a raw screen (.scr) switches the view to Spectrum Colour, since a screen
  dump is a colour image.
- Clicking the frame '+' button glides the pointer to the button's new centre after
  the strip widens, so repeated clicks stay under the cursor.
- The source filename shown under the title is shortened to 30 characters followed
  by an ellipsis when longer.

- HELP button (also F1) opening a scrollable in-app reader with all the keyboard
  shortcuts and an explanation of the file formats. The help text is an embedded
  file; the reader supports wheel, arrow, PgUp/PgDn and Home/End scrolling, a
  scrollbar, and a close box, dismissed with Esc or a click outside.

### Changed
- Zoom is now a single, screen-anchored scale. Previously the on-screen size of a
  virtual pixel depended on the fitted base cell, which varied with the sprite and
  window size, so the same zoom reading meant different pixel sizes and the grid
  overlays behaved inconsistently. The base cell is now fixed and a window resize
  only re-pans (it never rescales). The zoom readout is a percentage anchored to
  the screen: 0% is the zoom at which the largest sprite (32x24 cells) fits the
  screen height, and 800% is eight times that pixel size. The percentage re-anchors
  if the window is moved to a monitor of a different height. All grid and overlay
  fade thresholds are expressed on this scale, so the readout reflects exactly what
  drives them, consistently across sprite and window sizes.
- The bottom button strip is now two rows (actions and file operations); the
  width/height sizing buttons moved out of the strip into a compact 2x2 block
  immediately to the right of the last column, with a tight inner gap. The
  viewport reclaims the vertical space this frees.
- The HELP button sits directly below the remove-frame button, its right edge
  aligned with the add-frame button's right edge.
- The transparency chequer is nudged for contrast per view mode: two notches
  darker in Bitmap White, two notches lighter in Bitmap Black.
- The detail preview centres its image within the preview box when the sprite is
  smaller than the box, instead of pinning it to the top-left.
- The HELP button's height matches the onion/mode buttons. The help reader sizes
  its panel to about 70 characters wide when the screen allows, choosing an x2
  body scale when that panel fits and x1 otherwise (crisp bitmap text either way).
  Help text uses Markdown headings (# and ##) rendered in the accent colour, its
  scrollbar thumb can be dragged (and the track paged), and the content now
  includes an extended explanation of how zenimate works and the file formats.
- On macOS (detected by the presence of the standard top-level directories) the
  editor viewport zoom wheel is inverted to match the platform feel. Scroll views
  (file dialog, help reader) are left alone, since macOS already applies natural-
  scroll direction to the wheel before the app sees it.
- The press-and-hold preview popup now retracts by progressively clipping the
  full-scale frame down to the preview pane rather than scaling it, so it lands on
  exactly the image the preview already shows (no distortion during the collapse).
- Panning the editor viewport also works with a middle-mouse-button drag, in
  addition to the existing space + left-drag.

## [0.5.0]

A large feature release over the initial port. Native files were reworked into a
clear two-level scheme — `.zani`/`.zan` for a single animation (ZCUT bytes, so
`.zcut` still opens) and `.zbun`/`.zbu` for a bundle of animations with a
manifest — with save-form as a setting. The GUI gained Spectrum-format export
(SCR/TAP/TZX/SNA/Z80), image and screen import with a fit chooser, drag-and-drop,
a two-pane file browser that descends into bundles with per-entry previews, and
provenance-aware Save. The TUI became a colour-aware editor matching the GUI's
three view modes, with distinct draw/erase keys, colour-safe clears, and its own
save/load. A standalone `zaniplay` terminal player was added.

### Added
- TUI brought up to a proper colour-aware scratchpad, matching the GUI's three
  view modes (Bitmap Black / White / Spectrum Colour, cycled with Tab). Painting
  now has distinct keys — space draws, backspace/del erases, enter toggles — and
  colour can be selected (ink/paper/bright) and stamped per cell in a colour-paint
  mode. It never destroys colour unintentionally: clearing pixels (`x`) keeps
  colour, while the colour-wiping clear (`X`) and reset (`R`) require confirmation.
  The TUI can now save and load `.zani` files (`S`/`L`), so work persists and
  round-trips with the GUI.
- Provenance-aware Save. Ctrl+S saves back to wherever the sprite came from: a
  file is overwritten in place; a sprite opened from a bundle asks once whether to
  update it inside that bundle (preserving its manifest label) or split it off as
  a separate `.zani`, remembering the choice; a new sprite prompts for a
  destination. Ctrl+Shift+S is Save As and always prompts. The title block shows
  the current source (`name.zani`, `name - bundle.zbun`, or `name (unsaved)`).
- Dropping a `.zbun` onto the window opens the browser inside that bundle so you
  can pick an animation, rather than failing to load it.
- `zaniplay` accepts a `bundle.zbun#entry` reference to play a single animation
  from a bundle, in addition to a standalone `.zani` file.
- File-extension scheme reworked so names describe the content: a single
  animated sprite is now `.zani` (or `.zan` on 8.3 filesystems) — the bytes are
  unchanged ZCUT, so `.zcut` is still accepted on load. A collection of
  animations is a `.zbun` bundle (`.zbu` on 8.3). Which form is written is a
  setting, toggled with Ctrl+F; loading accepts every form.
- `.zbun` bundles: a zip gathering several whole `.zani` animations plus a
  `manifest.json` index (name, frame count, dimensions, and a free-text label per
  entry). The Bundle drawer button (or Ctrl+Shift+E) adds the current sprite to a
  bundle: pick the `.zbun` file, then choose to create a new bundle or add to an
  existing one.
- Browse into `.zbun` bundles from the Open dialog: a bundle appears as a
  descendable item (marked "[bundle]"); opening it lists the animations inside,
  and `..` climbs back out. Selecting an animation shows a preview pane with a
  thumbnail of its first frame plus its frame count, dimensions and label.
  (Implemented as a reusable container/preview extension to the filepick package.)
- Image import (JPEG, PNG, GIF): Open or drag a raster image onto the window and
  zenimate reduces it to a 256×192 Spectrum screen (ink/paper/bright per 8×8 cell
  via zentools). A fit-strategy chooser asks first — Best fit (keep aspect,
  letterbox), Stretch (fill, ignore aspect), or Centre (no scale, crop/pad).
- Drag-and-drop: dropping a file onto the GUI window loads it. Supported types
  are `.zcut` (native sprite), `.zani` (animation), `.scr` (raw 256×192 screen),
  and `.tap`/`.tzx`/`.sna`/`.z80` (a screen extracted from the container when one
  is present). The first usable file wins; unsupported drops report why.
- Screen/container import: Open and drag-and-drop can load a raw `.scr` as a full
  256×192 editable screen, and pull the display file out of tape (`.tap`/`.tzx`)
  and snapshot (`.sna`/`.z80`) containers. Open's filter now lists every loadable
  type.
- `.zani` import: Open now loads both native sprites (`.zcut`) and animations
  (`.zani`, either physical form), dispatching on extension — so an exported
  animation round-trips back into the editor.
- `zaniplay`: a standalone terminal player for `.zani` files. Loads an animation
  and loops the frames in colour (ANSI half-block rendering); `-fps` sets the
  rate and `-once` plays through a single time. Reuses the model loader and
  zxpalette only — no GUI dependency.
- Animation export to the `.zani` container: the whole multi-frame sprite is
  written as one ZCUT per frame plus a `metadata.json` sidecar. Two physical
  forms — a zip (default; smaller and tool-friendly) or a tzx (loadable on real
  tape) — and selectable per-frame content (bitmap, a derived mask, attributes).
  Triggered by the Export Anim drawer button or Ctrl+Shift+E, via a preset
  chooser then the save dialog.
- Export to ZX Spectrum formats: the current frame can be rendered to a full
  256×192 screen and saved as a raw `.scr`, a `.tap` (plain or auto-running via a
  tokenised BASIC loader), a `.tzx`, or a `.sna`/`.z80` snapshot — the snapshots
  boot showing the picture. Triggered by Ctrl+E or the Export drawer button,
  which opens a format chooser and then the save dialog. Built on zentools'
  `pkg/scr` (screen composition) and `pkg/build` (tape/snapshot containers).
- Native file persistence in the ZCUT format (via zentools' `pkg/scr`): Save and
  Open round-trip the whole sprite — every frame's pixels and per-cell attributes
  — losslessly. Triggered by Ctrl+S / Ctrl+O or the Open/Save buttons in the
  bottom drawer, through an in-app file dialog.
- File dialogs are provided by the reusable, renderer-agnostic `filepick`
  package; zenimate supplies a thin cgo-free raylib adapter. The dialog is modal:
  while open it captures all input and overlays the editor.
- `pkg/zxpalette`: the ZX Spectrum 16-colour palette and attribute-byte
  encoding (ink/paper/bright/flash), cloned from the zenzx emulator so colours
  match exactly. Framework-neutral (`image/color`).
- `internal/model`: a per-8×8-character-cell attribute layer (hardware-accurate),
  stored per frame so each frame can carry its own colour layout; copy/paste of
  a frame carries its attributes.
- GUI view modes — Bitmap Black, Bitmap White, and Spectrum Colour — selectable
  from a button row. Spectrum Colour renders set pixels in the cell ink colour
  and clear pixels in the paper colour, with a vertical 16-swatch attribute
  palette (8 colours × normal/bright); left-click selects ink, right-click paper.
  Holding Ctrl while painting stamps the current attribute onto a cell. A faint
  per-virtual-pixel grid appears in Spectrum mode at ≥50% zoom.
- Variable frame count (1–16) via +/- buttons by the frame strip; arbitrary
  sprite sizes up to 32×24 character cells (256×192 px); a Full Screen button;
  non-destructive resize (pixels and attributes preserved); a zoom-to-fit
  viewport animation when the sprite is resized. Esc no longer closes the GUI.
- Onion skinning in the bitmap views: the previous frame is overlaid as a
  translucent red silhouette and the next frame as translucent green (wrapping at
  the ends), each toggled independently. Not shown in Spectrum Colour mode.
- Fixed-size detail preview, decoupled from the sprite dimensions so a large
  sprite no longer starves the viewport. It renders at a fixed integer zoom
  (×1–×4, each sprite pixel an N×N square), right-click cycles the zoom, centred
  on the cursor's pixel while hovering the paint area and on the last-modified
  pixel otherwise. Press-and-hold grows a full-sprite popup that unfurls out of
  the preview box, anchored to its top-right corner, and shrinks back on release.
- Header rework: the frame strip and the view-mode/onion toolbars moved up beside
  the title block and span the available width, giving the viewport more vertical
  room. The title block is collapsible — click it to shrink to a small button
  (freeing horizontal space), click the button to restore.
- Status messages now surface as an animated OSD caption: each new message rises
  from the bottom-right window border in bright orange with a black outline,
  drifts up past the palette and fades over 100px, with random "magical pixel"
  sprites scattered around it (refreshed every 200ms).
- The bottom button strip is a sliding drawer (open by default): a small triangle
  below the viewport's bottom-right border toggles it — pointing up when closed,
  down when open — easing in/out, with the viewport reclaiming the space when the
  drawer is closed. The frame +/- buttons use a smaller, centred symbol.
- Holding Shift while drawing locks the stroke to one axis (horizontal or
  vertical) for straight lines: the anchor is the mouse-down pixel and the axis is
  chosen once from the dominant direction, then held for the rest of the stroke.
- GUI pan gesture: hold space and left-drag to pan (space as a modifier), with
  cursor-anchored mouse-wheel zoom; both carry glide-after-release inertia.
  Smooth fractional-pixel zoom with no integer snapping.
- GUI transparency chequer redrawn at one square per virtual pixel.
- `pkg/bdf`: a standalone BDF (Glyph Bitmap Distribution Format) reader and
  rasteriser, lifted from the subterm terminal renderer and decoupled from its
  buffer/registry. Parses a font and renders any glyph to a full-cell RGBA
  pixmap. Zero dependencies beyond the standard library.
- `internal/fonts`: the Sinclair ZX Spectrum 8x8 face and the Cozette 6x13 face,
  embedded and decoded through `pkg/bdf`. Bundled licences included.
- `internal/model`: the UI-agnostic sprite document — 1–16 animation frames,
  arbitrary cell-snapped dimensions up to 32×24 cells, per-frame per-cell colour
  attributes, pixel toggle, copy/paste, non-destructive resize, and an observer
  hook.
- `internal/ui`: a frontend-independent controller carrying the editing actions
  and the animation player.
- `cmd/zenimate-tui`: a terminal frontend. UTF-8 labels, a half-block sprite
  grid (two pixel rows per text row), raw-mode input, no third-party TUI library.
- `cmd/zenimate-gui`: a raylib desktop frontend. All on-screen text is rendered
  from the bundled Sinclair BDF face via textures — raylib's native font API is
  never used. Builds cgo-free through the purego path.
- Project hygiene: `VERSION` single source of truth, `pkg/version`,
  `scripts/syncver.sh`, `scripts/release.sh`, a `Makefile` front-end, and a
  race-detector release gate.

### Notes
- The GUI window requires an OpenGL-capable display and so runs on a desktop;
  it still compiles in headless environments via the purego build tag.

## [0.1.0]

First release. A Go port of the browser-based ZX Spectrum animated sprite
editor, with two frontends (a raylib/purego GUI and a terminal TUI) over a
shared, UI-agnostic core, native ZCUT save/load, and frame-based animation.
