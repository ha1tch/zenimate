# zenimate

A ZX Spectrum animated sprite editor, in Go. You draw a multi-frame sprite on a
pixel grid, paint per-character-cell Spectrum colour attributes, and preview the
animation. Sprites can be any size up to a full screen (32×24 character cells,
256×192 px) and carry between 1 and 16 frames.

There are two frontends over one shared, UI-agnostic core:

- **`zenimate-tui`** — a terminal editor. Ordinary UTF-8 text for labels; the
  sprite grid is drawn with Unicode half-blocks. No third-party terminal library.
- **`zenimate-gui`** — a raylib desktop editor. Every on-screen character is
  rendered from the bundled Sinclair ZX Spectrum bitmap font, so the UI text is
  period-correct.

A third command, **`zaniplay`**, plays animations in the terminal.

## Quick start

```
make build      # both editor frontends into dist/
```

Requires Go 1.25 or later. `make build-gui` builds the GUI cgo-free (no system
GL headers needed); the GUI needs an OpenGL-capable display to *run*. Run `make`
to list all targets.

## Documentation

- **[docs/GUIDE.md](docs/GUIDE.md)** — building, the full TUI and GUI controls,
  the file formats, zaniplay, the code layout, and CI/releases.

## Licence

Apache 2.0 — see [LICENSE](LICENSE). Bundled fonts and dependencies carry their
own terms; see [THIRD-PARTY.md](THIRD-PARTY.md).
