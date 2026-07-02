# Persistence for zenimate via zentools

## Summary

zenimate currently keeps everything in memory: the `Sprite` model has no save,
load, or export path of any kind. This document proposes how to add persistence
of several kinds, reusing `github.com/ha1tch/zentools` for the ZX-format exports
rather than hand-rolling tape and snapshot encoders.

Two important findings shaped this proposal, both worth stating up front:

1. **There is no `pkg/scr` in the published zentools.** The current public repo
   (`github.com/ha1tch/zentools`) is at **v0.4.1**, with a single `main` branch
   and tags up to v0.4.1. It contains `pkg/tap`, `pkg/tzx`, `pkg/snapshot`,
   `pkg/basic`, and `pkg/build` — but no screen/SCR package and no ZCUT format.
   The `pkg/scr` / `zx scr` / ZCUT work exists in our own project notes as a
   0.5.0 milestone, but it has not been pushed to the public repository, so it is
   not reachable here. This proposal therefore builds on the v0.4.1 surface that
   actually exists. Where `pkg/scr` would help, that is called out as a future
   integration point rather than assumed.

2. **zenimate's art is sprites, not full screens.** A `Sprite` is up to 32x24
   character cells (256x192 px maximum, i.e. a full screen at the top end, but
   usually much smaller), multi-frame, with one ZX attribute byte per 8x8 cell
   per frame. A ZX `.scr` screen file is always a fixed 6912 bytes for the whole
   256x192 display. The two are not the same shape, and the mapping between them
   is the interesting part of the design.

## What zentools v0.4.1 actually provides

Relevant package surface, verified against the cloned source:

### `pkg/tap`
- `EncodeCode(name string, data []byte, loadAddress uint16) []byte` — wraps raw
  bytes as a CODE tape block (header + data). This is the in-memory entry point:
  bytes in hand, TAP image out.
- `EncodeProgram(name string, data []byte, autostart uint16) []byte` — a BASIC
  program block.
- `Decode(image []byte) ([]Block, error)` — parse a TAP back into blocks.

### `pkg/tzx`
- `EncodeFromTAP(tap []byte, opts EncodeOptions) ([]byte, error)` — wrap a TAP
  image as TZX (with optional title/author/year metadata).

### `pkg/snapshot`
- `EncodeSNA` / `EncodeSNA128`, `EncodeZ80` / `EncodeZ80v3`, plus the matching
  `Decode*` functions, operating on a `MachineState` (CPU, paging, memory banks).

### `pkg/build` — the high-level convenience layer
- `Request{ Name, Code []byte, Origin uint16, Start, SP uint16, Model }`.
- `EncodeTAP(r)` / `EncodeTZX(r)` — tape images of the code.
- `EncodeTAPWithLoader(r)` — a tape that auto-runs: a tokenised BASIC loader
  (`CLEAR`/`LOAD ""CODE`/`RANDOMIZE USR`) precedes the CODE block.
- `EncodeSNA(r)` / `EncodeZ80(r)` — overlay the code onto a model's boot state at
  `Origin` and set PC/SP, producing a snapshot that runs the program on load.

The crucial mechanism for art persistence is `build`'s memory overlay: code is
written into RAM at `Origin`, mapped to the correct bank. The Spectrum display
file lives at **0x4000-0x57FF** (6144 bytes of bitmap) followed by
**0x5800-0x5AFF** (768 bytes of attributes) — 6912 bytes total. So writing a
6912-byte screen image at `Origin = 0x4000` lands it exactly on the display,
which means a snapshot built that way **boots showing the artwork**.

## zenimate's model, precisely

```
Sprite{
  width, height int          // pixels, each a multiple of 8, up to 256x192
  frames        []Frame      // Frame = []bool, row-major, len = width*height
  frameAttrs    [][]byte     // one attr byte per 8x8 cell, per frame
  attrCols, attrRows int     // width/8, height/8
  name          string
  selected      int
}
```

So the data to persist per project is: dimensions, a frame count, and for each
frame a pixel bitmap plus an attribute map. Nothing currently serialises this.

## Proposal: three tiers of persistence

### Tier 1 — Native project format (the source of truth)

A lossless, round-trippable on-disk format for a whole zenimate project:
dimensions, all frames, all attributes, the name, and the selected frame. This is
what File > Save / Open operates on, and it is the only format that preserves
everything zenimate knows.

Recommendation: a small, explicit binary container (call it `.zspr`, "zen
sprite") with a versioned header, rather than JSON. Per working conventions the
default would be a simple, documented byte layout:

```
magic    "ZSPR"            4 bytes
version  uint8             1
flags    uint8             1   (reserved: e.g. bit0 = attrs present)
wCells   uint8             1   (1..32)
hCells   uint8             1   (1..24)
frames   uint8             1   (1..16)
selected uint8             1
nameLen  uint8 + name      1 + n
per frame:
  bitmap   ceil(w*h/8) bytes   (packed MSB-first, row-major)
  attrs    wCells*hCells bytes (one ZX attr byte per cell)
```

This is trivial to encode/decode in `internal/model`, has no external
dependencies, and is forward-versioned. It is independent of zentools — zentools
is for *export*, not for the native format.

A JSON variant is possible if human-readability or web interop matters, but for a
pixel editor a compact binary file is the better default; we can add a
`--format json` export later if needed.

### Tier 2 — ZX screen export via zentools (the "make it real" path)

This is where zentools earns its place. The art becomes something a real
Spectrum (or any emulator) can load. The bridge is: **render the sprite into a
6912-byte screen image, then hand it to `pkg/build`.**

Pipeline:

1. **Compose a screen buffer.** Build a `[6912]byte` ZX display image from a
   chosen frame:
   - The bitmap third: for each set pixel, set the corresponding bit in the
     0x4000 region using the Spectrum's interleaved line address arithmetic
     (the y-coordinate bit-scramble; not linear). zenimate already knows the
     pixel grid, so this is a coordinate mapping plus bit packing.
   - The attribute third: copy the frame's `attrCols*attrRows` attribute bytes
     into the 0x5800 region at the matching cell positions.
   - A sprite smaller than 256x192 is placed at a chosen origin cell (default
     top-left, or centred), with the rest of the screen left as a configurable
     PAPER fill so it is a valid full screen.

2. **Wrap with zentools.** Given the 6912-byte buffer `scr`:
   - **Raw `.scr`** — just write the 6912 bytes. No zentools needed; this is the
     universal interchange format every emulator and tool reads.
   - **`.tap`** — `build.EncodeTAP(build.Request{ Name, Code: scr, Origin: 0x4000 })`.
     Loads with `LOAD "" CODE 16384`.
   - **Auto-running `.tap`** — `build.EncodeTAPWithLoader(...)` so `LOAD ""`
     shows the picture immediately (loader CLEARs, loads CODE at 0x4000, and we
     point USR at a tiny "halt" or at a return to BASIC; for a static screen the
     loader can simply load and stop).
   - **`.tzx`** — `build.EncodeTZX(req)` or `EncodeTZXFromTAP`, with title/author
     metadata (handy for archival; per conventions, use neutral placeholders, no
     personal identifiers baked into shared artwork).
   - **`.sna` / `.z80` snapshot** — `build.EncodeSNA(req)` / `EncodeZ80(req)`
     with `Origin: 0x4000`. Because the overlay writes the screen straight into
     the display file of a booted machine state, the snapshot **opens already
     showing the art**. This is the most satisfying "share" format: one file, and
     the emulator displays the picture with no loading.

All five outputs come from the *same* 6912-byte composition step; only the final
wrap differs. That keeps the screen-composer (ours) cleanly separated from the
container encoders (zentools).

### Tier 3 — Animation export (multi-frame)

zenimate's sprites are animated, which a single `.scr` cannot express. Options,
in increasing ambition:

- **Frame sequence of screens.** Export each frame as its own `.scr` (or a
  numbered set), and/or a multi-block `.tap` with one CODE block per frame at
  0x4000. A trivial BASIC driver (tokenised via `pkg/basic`, assembled via
  `pkg/build`'s loader machinery) can `LOAD` and flip frames on a timer. This is
  entirely expressible with the v0.4.1 surface.
- **Sprite-data export (not screen export).** For game use, the more valuable
  artifact is the *sprite* as raw graphic data (masked or unmasked, the format
  KnightQuest / the ChibiAkumas-style engines consume), not a full screen. That
  is a packed bitmap+mask blob at an arbitrary `Origin`, wrapped as a CODE
  `.tap`/snapshot exactly as in Tier 2 but without the screen-address scramble.
  This connects zenimate to the rest of the ZX Opal toolchain (zenas, zenzx,
  KnightQuest) and is probably the highest-value export for actual game work.

## Where `pkg/scr` (when published) slots in

If/when the 0.5.0 `pkg/scr` + ZCUT work is pushed, it would replace the
hand-written screen-composition step in Tier 2: the y-scramble bit addressing,
the attribute placement, and likely the ZCUT "cut/paste a sub-rectangle of a
screen" semantics are exactly what a sprite-to-screen blitter needs. The clean
seam in this design — *compose 6912 bytes here, wrap with `pkg/build` there* —
means adopting `pkg/scr` later is a drop-in for the composer and changes nothing
about the export wrappers. Until then, the screen-composition arithmetic lives in
zenimate (a ~50-line, well-understood, testable function).

## Recommended build order

1. **Tier 1 native `.zspr`** in `internal/model` (save/load, fully round-trip
   tested). This is the foundation and has no external dependency.
2. **Tier 2 raw `.scr` export** — the screen composer plus a plain 6912-byte
   writer. Testable purely by inspecting bytes (set a pixel, assert the right bit
   at the right scrambled address).
3. **Tier 2 zentools wraps** — add `zentools` as a dependency and expose
   `.tap` / `.tzx` / `.sna` / `.z80` from the same composer. Per toolchain
   policy this pulls zentools at its tagged version into `go.mod`.
4. **Tier 3** as a later, separable step once single-frame export is solid.

## Dependency and policy notes

- Adding zentools means `require github.com/ha1tch/zentools vX.Y.Z` in
  `go.mod`; it is one of our own namespace projects, both already on `go 1.25`.
- The screen composer must use **neutral placeholders** for any embedded names
  (TZX title/author), never personal identifiers, in any shareable artifact.
- The composer and the wrappers stay in separate files so the zentools dependency
  is confined to the export layer and the native format stays dependency-free.
