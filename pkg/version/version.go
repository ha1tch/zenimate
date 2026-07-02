// Package version exposes the zenimate build version.
//
// The constant below is the compiled-in version string. It is kept in sync with
// the repository's top-level VERSION file by scripts/syncver.sh; VERSION is the
// single source of truth. Do not edit Version by hand — change VERSION and run
// `make sync` (or scripts/syncver.sh) instead.
package version

// Version is the current zenimate version, synced from the VERSION file.
const Version = "0.6.0"
