# Third-party notices

zenimate is licensed under Apache 2.0 (see LICENSE). It bundles and depends on
the following third-party works, each under its own terms.

## Bundled fonts

These BDF faces are embedded in the binary (`internal/fonts/`). Neither is
authored by this project; each is redistributed under its own licence, whose
full text sits alongside the `.bdf` file.

| Font | File | Licence | Copyright holder | Licence text |
|------|------|---------|------------------|--------------|
| ZX Spectrum | `internal/fonts/sinclair.bdf` | Amstrad copyright (permission to redistribute the ROM; via ha1tch/bdf-fonts) | Amstrad plc (ZX Spectrum 48K ROM) | `internal/fonts/LICENSE-Sinclair.txt` |
| Cozette | `internal/fonts/cozette.bdf` | MIT | Slavfox | `internal/fonts/LICENSE-Cozette.txt` |

The Sinclair face is the reproducible result of extracting the character set
from the ZX Spectrum 48K ROM. Amstrad has given permission to redistribute the
ROM; the binding terms and the Amstrad permission statement are reproduced in
`internal/fonts/LICENSE-Sinclair.txt`, which must travel with the font.

The Cozette face carries a Reserved Font Name; do not ship a modified Cozette
under the name "Cozette".

## Go dependencies

| Module | Version | Licence |
|--------|---------|---------|
| github.com/gen2brain/raylib-go/raylib | v0.60.0 | Zlib |
| github.com/ebitengine/purego | v0.10.0 | Apache-2.0 |
| github.com/jupiterrider/ffi | v0.7.0 | MIT |
| golang.org/x/exp | (pinned) | BSD-3-Clause |
| golang.org/x/term | v0.36.0 | BSD-3-Clause |
| golang.org/x/sys | v0.37.0 | BSD-3-Clause |

raylib-go wraps raylib (Zlib licence, by Ramon Santamaria and contributors). The
underlying BDF reader in `pkg/bdf` was lifted from the ha1tch/subterm project and
relicensed here under this project's Apache 2.0 terms as permitted by its origin.

## ZX Spectrum palette (pkg/zxpalette)

The 16-colour palette and attribute encoding in `pkg/zxpalette` are derived from
the ha1tch/zenzx ZX Spectrum emulator (same author), reproduced here so the
editor's colours match the emulator exactly.
