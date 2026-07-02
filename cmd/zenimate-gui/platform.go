package main

import "os"

// isMac reports whether zenimate appears to be running on macOS, using a
// deliberately simple heuristic: the presence of the standard top-level
// directories macOS always has. This avoids build tags or runtime.GOOS checks
// (which would report the build target, not necessarily the run host) and is
// good enough to drive small platform conveniences like natural-scroll wheel
// direction.
var macDetected = detectMac()

func detectMac() bool {
	for _, d := range []string{"/Applications", "/Users", "/System", "/Library"} {
		info, err := os.Stat(d)
		if err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}

// wheelSign returns the multiplier applied to the raw wheel delta for the editor
// viewport zoom. On macOS it is inverted so the zoom gesture matches the
// platform feel. Scroll views (file dialog, help reader) do NOT use this: macOS
// already applies natural-scroll direction to the wheel value before the app
// sees it, so inverting there would double-invert and scroll backwards.
func wheelSign() float32 {
	if macDetected {
		return -1
	}
	return 1
}
