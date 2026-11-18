package gqr

import (
	"errors"
	"strings"
	"testing"
)

func mustEncode(t *testing.T, text string, level Level) *Code {
	t.Helper()
	c, err := Encode(text, level)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCapacity(t *testing.T) {
	tests := []struct {
		version int
		level   Level
		want    int // data codewords
	}{
		{1, Low, 19}, {1, Medium, 16}, {1, Quartile, 13}, {1, High, 9},
		{10, Medium, 216}, {25, Quartile, 718},
		{40, Low, 2956}, {40, Medium, 2334}, {40, Quartile, 1666}, {40, High, 1276},
	}
	for _, tt := range tests {
		if got := dataCodewords(tt.version, tt.level); got != tt.want {
			t.Errorf("dataCodewords(%d, %v) = %d, want %d", tt.version, tt.level, got, tt.want)
		}
	}
}

// The values of r in ISO/IEC 18004, Table 9.
func TestCorrectableErrors(t *testing.T) {
	tests := []struct {
		version int
		level   Level
		want    int
	}{
		{1, Low, 2}, {1, Medium, 4}, {1, Quartile, 6}, {1, High, 8},
		{2, Low, 4}, {2, Medium, 8}, {3, Low, 7}, {3, High, 11},
		{5, Quartile, 9}, {40, High, 15},
	}
	for _, tt := range tests {
		if got := correctableErrors(tt.version, tt.level); got != tt.want {
			t.Errorf("correctableErrors(%d, %v) = %d, want %d", tt.version, tt.level, got, tt.want)
		}
	}
}

func TestEncodeVersion(t *testing.T) {
	tests := []struct {
		text    string
		level   Level
		version int
	}{
		{strings.Repeat("1", 41), Low, 1}, // numeric capacity of 1-L
		{strings.Repeat("1", 42), Low, 2},
		{strings.Repeat("A", 25), Low, 1}, // alphanumeric capacity of 1-L
		{strings.Repeat("a", 17), Low, 1}, // byte capacity of 1-L
		{strings.Repeat("a", 18), Low, 2},
		{strings.Repeat("a", 2953), Low, 40}, // byte capacity of 40-L
		{strings.Repeat("1", maxChars), Low, 40},
		{"abc" + strings.Repeat("1", 20), Low, 1}, // fits only as two segments
		{"漢字は十三ビットで", Low, 1},                     // fits only in Kanji mode
	}
	for _, tt := range tests {
		c := mustEncode(t, tt.text, tt.level)
		if c.Version() != tt.version || c.Size() != symbolSize(tt.version) {
			t.Errorf("Encode(%.20q, %v): version %d, size %d; want %d, %d",
				tt.text, tt.level, c.Version(), c.Size(), tt.version, symbolSize(tt.version))
		}
	}
}

func TestEncodeRaisesLevel(t *testing.T) {
	tests := []struct {
		text        string
		level, want Level
	}{
		{"1", Low, High},
		{"HELLO WORLD", Low, Quartile},
		{"https://go.dev", Low, Medium},
		{strings.Repeat("a", 17), Low, Low}, // fills a 1-L symbol
	}
	for _, tt := range tests {
		c := mustEncode(t, tt.text, tt.level)
		if c.Version() != 1 || c.Level() != tt.want {
			t.Errorf("Encode(%q, %v) = %d-%v, want 1-%v", tt.text, tt.level, c.Version(), c.Level(), tt.want)
		}
	}
}

func TestEncodeErrors(t *testing.T) {
	if _, err := Encode(strings.Repeat("a", 2954), Low); !errors.Is(err, ErrTooLong) {
		t.Errorf("too long: err = %v, want ErrTooLong", err)
	}
	if _, err := Encode(strings.Repeat("1", maxChars+1), Low); !errors.Is(err, ErrTooLong) {
		t.Errorf("too many digits: err = %v, want ErrTooLong", err)
	}
	if _, err := EncodeBytes(make([]byte, 1<<20), Low); !errors.Is(err, ErrTooLong) {
		t.Errorf("too many bytes: err = %v, want ErrTooLong", err)
	}
	if _, err := Encode("", Medium); !errors.Is(err, ErrEmpty) {
		t.Errorf("empty text: err = %v, want ErrEmpty", err)
	}
	if _, err := EncodeBytes(nil, Medium); !errors.Is(err, ErrEmpty) {
		t.Errorf("empty bytes: err = %v, want ErrEmpty", err)
	}
	if _, err := Encode("x", Level(9)); err == nil {
		t.Error("invalid level: no error")
	}
}

func TestFinderPattern(t *testing.T) {
	c := mustEncode(t, "https://go.dev", High)
	// Top-left finder: dark ring, light ring, dark 3x3 center, light separator.
	for _, p := range []struct {
		x, y int
		dark bool
	}{{0, 0, true}, {6, 6, true}, {1, 1, false}, {3, 3, true}, {7, 7, false}} {
		if c.Dark(p.x, p.y) != p.dark {
			t.Errorf("module (%d,%d) dark = %v, want %v", p.x, p.y, !p.dark, p.dark)
		}
	}
	if !c.Dark(8, c.Size()-8) {
		t.Error("the always-dark module is light")
	}
}

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
