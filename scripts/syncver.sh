#!/usr/bin/env bash
# syncver.sh - propagate the canonical VERSION into pkg/version/version.go.
#
# VERSION is the single source of truth. This script rewrites the Version
# constant to match it. It uses a guarded Python substitution (never sed/awk):
# the replacement is asserted to have fired, and the change is reported.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="$ROOT/VERSION"
VERSION_GO="$ROOT/pkg/version/version.go"

if [[ ! -f "$VERSION_FILE" ]]; then
	echo "syncver: VERSION file not found at $VERSION_FILE" >&2
	exit 1
fi

VER="$(tr -d ' \t\r\n' < "$VERSION_FILE")"
if [[ -z "$VER" ]]; then
	echo "syncver: VERSION is empty" >&2
	exit 1
fi

VER="$VER" VERSION_GO="$VERSION_GO" python3 - <<'PY'
import os, re, sys

ver = os.environ["VER"]
path = os.environ["VERSION_GO"]

with open(path) as f:
    content = f.read()

original = content
content = re.sub(r'const Version = "[^"]*"', f'const Version = "{ver}"', content)

if content == original:
    if f'const Version = "{ver}"' in original:
        print(f"syncver: pkg/version already at {ver}")
        sys.exit(0)
    print("syncver: WARNING no Version constant matched; file unchanged", file=sys.stderr)
    sys.exit(1)

with open(path, "w") as f:
    f.write(content)
print(f"syncver: pkg/version set to {ver}")
PY
