package gqr

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	c := mustEncode(t, "gqr", Medium)
	want := (c.Size() + 2*QuietZone + 1) / 2
	for _, s := range []string{c.String(), c.StringInverted()} {
		if lines := strings.Count(s, "\n"); lines != want {
			t.Errorf("%d lines, want %d", lines, want)
		}
	}
}
