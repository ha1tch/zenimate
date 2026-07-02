//go:build purego

package main

import "testing"

func TestExtOf(t *testing.T) {
	cases := map[string]string{
		"/a/b/hero.zcut":  "zcut",
		"SCREEN.SCR":      "scr",
		"/x/y.TZX":        "tzx",
		"noext":           "",
		"/dir.with.dot/f": "",
		"trailingdot.":    "",
	}
	for in, want := range cases {
		if got := extOf(in); got != want {
			t.Errorf("extOf(%q) = %q, want %q", in, got, want)
		}
	}
}
