#!/usr/bin/env bash
# release.sh - zenimate release hygiene automation.
#
# Single-pass release preparation:
#   1. Validate the VERSION string and the matching CHANGELOG entry
#   2. Sync VERSION -> pkg/version
#   3. gofmt + go vet (the formatting/static gate)
#   4. Build both frontends (TUI native; GUI via the cgo-free purego path)
#   5. Run the full test suite, including the race detector
#   6. Stage a versioned checkpoint zip under dist/
#
# It never tags or pushes: cutting the git tag is the human's trigger.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VER="$(tr -d ' \t\r\n' < VERSION)"
echo "==> Releasing zenimate v${VER}"

# 1. Validate version format (semver MAJOR.MINOR.PATCH) and CHANGELOG entry.
if [[ ! "$VER" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	echo "release: VERSION '$VER' is not MAJOR.MINOR.PATCH" >&2
	exit 1
fi
if ! grep -q "^## \[${VER}\]" CHANGELOG.md; then
	echo "release: no CHANGELOG.md entry for [${VER}]" >&2
	exit 1
fi

# 2. Sync version constant.
bash scripts/syncver.sh

# 3. Formatting + static analysis.
echo "==> gofmt"
fmtout="$(gofmt -l . || true)"
if [[ -n "$fmtout" ]]; then
	echo "release: gofmt would change:" >&2
	echo "$fmtout" >&2
	exit 1
fi
echo "==> go vet"
go vet ./pkg/... ./internal/... ./cmd/zenimate-tui/

# 4. Build both frontends.
mkdir -p dist
echo "==> build TUI"
go build -o "dist/zenimate-tui" ./cmd/zenimate-tui/
echo "==> build GUI (purego, cgo-free)"
CGO_ENABLED=0 go build -tags purego -o "dist/zenimate-gui" ./cmd/zenimate-gui/

# 5. Tests, including the race detector (a permanent release gate).
echo "==> go test -race"
go test -race ./pkg/... ./internal/... -count=1

# 6. Checkpoint zip of the source tree (excluding build artifacts and VCS).
ZIP="dist/zenimate-v${VER}.zip"
rm -f "$ZIP"
echo "==> staging checkpoint $ZIP"
zip -q -r "$ZIP" . \
	-x './dist/*' \
	-x './.git/*' \
	-x '*/.DS_Store'

echo "==> Done. Artifacts in dist/:"
ls -la dist/
echo
echo "Next: review, then 'git tag v${VER}' to trigger the release."
