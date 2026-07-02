package main

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// makeRaw puts the controlling terminal (stdin) into raw mode and returns a
// function that restores the previous state. It uses golang.org/x/term, which
// handles the per-platform termios details (Linux TCGETS/TCSETS, Darwin
// TIOCGETA/TIOCSETA, *BSD, Windows console modes), so the TUI builds and runs
// the same way on Linux, macOS and the BSDs.
//
// If stdin is not a terminal (e.g. a pipe), it returns an error rather than
// silently proceeding, since the editor needs raw key input to function.
func makeRaw() (restore func(), err error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return nil, fmt.Errorf("standard input is not a terminal")
	}
	old, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() { _ = term.Restore(fd, old) }, nil
}
