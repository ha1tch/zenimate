// Package zenui is a renderer-agnostic file Open/Save dialog widget.
//
// It owns the logic of a file dialog — directory navigation, listing, filtering,
// selection, scrolling, filename entry and keyboard handling — but draws nothing
// itself. The host supplies a small Renderer (fill a rectangle, draw text,
// measure text, clip) and an Input snapshot (mouse and keyboard) each frame.
// Because the package imports no graphics library, the same dialog runs under
// raylib, a software framebuffer, or a headless test harness.
//
// It began as a standalone module and now lives inside zenimate, its only
// consumer. It has no dependencies outside the standard library.
package zenui
