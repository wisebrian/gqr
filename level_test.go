package gqr

import (
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	for l := Low; l <= High; l++ {
		for _, s := range []string{l.String(), strings.ToLower(l.String())} {
			if got, err := ParseLevel(s); err != nil || got != l {
				t.Errorf("ParseLevel(%q) = %v, %v; want %v", s, got, err, l)
			}
		}
	}
	if _, err := ParseLevel("X"); err == nil {
		t.Error(`ParseLevel("X"): no error`)
	}
}
