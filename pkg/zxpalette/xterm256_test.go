package zxpalette

import "testing"

func TestXterm256Mapping(t *testing.T) {
	// Spot-check the exact-match bright colours land on their known cube indices.
	cases := []struct {
		colour int
		bright bool
		want   int
	}{
		{Black, false, 16},
		{Red, true, 196},
		{Green, true, 46},
		{Blue, true, 21},
		{Yellow, true, 226},
		{White, true, 231},
		{White, false, 251},
	}
	for _, c := range cases {
		if got := Xterm256(c.colour, c.bright); got != c.want {
			t.Errorf("Xterm256(%d,%v) = %d, want %d", c.colour, c.bright, got, c.want)
		}
	}
	// Every index must be a valid xterm-256 colour (16-255 range we use).
	for i := 0; i < 8; i++ {
		for _, b := range []bool{false, true} {
			x := Xterm256(i, b)
			if x < 16 || x > 255 {
				t.Errorf("Xterm256(%d,%v) = %d out of range", i, b, x)
			}
		}
	}
}

func TestXterm256Attr(t *testing.T) {
	a := Attr(Red, Yellow, true, false) // ink red, paper yellow, bright
	ink, paper := Xterm256Attr(a)
	if ink != Xterm256(Red, true) || paper != Xterm256(Yellow, true) {
		t.Errorf("Xterm256Attr wrong: ink=%d paper=%d", ink, paper)
	}
}
