package gqr

import (
	"slices"
	"strings"
	"testing"
)

func TestAlignmentPositions(t *testing.T) {
	tests := map[int][]int{
		1:  nil,
		2:  {6, 18},
		7:  {6, 22, 38},
		32: {6, 34, 60, 86, 112, 138},
		40: {6, 30, 58, 86, 114, 142, 170},
	}
	for v, want := range tests {
		if got := alignmentPositions(v); !slices.Equal(got, want) {
			t.Errorf("alignmentPositions(%d) = %v, want %v", v, got, want)
		}
	}
}

// Every module outside the function patterns carries data, so the placement
// order must visit exactly as many modules as the capacity formula counts.
func TestDataModules(t *testing.T) {
	for v := minVersion; v <= maxVersion; v++ {
		n := 0
		for range newMatrix(v).dataModules() {
			n++
		}
		if n != rawDataModules(v) {
			t.Errorf("version %d: %d data modules, want %d", v, n, rawDataModules(v))
		}
	}
}

func TestFormatAndVersionBits(t *testing.T) {
	m := newMatrix(7)
	m.drawFormat(Medium.formatBits()<<3 | 5)
	// The format string for M with mask 5 is 100000011001110, bit 14 first,
	// read along row 8 and then up column 8, skipping the timing patterns.
	want := "100000011001110"
	var got strings.Builder
	for x := range 9 {
		if x != 6 {
			got.WriteByte(b2c(m.at(x, 8)))
		}
	}
	for y := 7; y >= 0; y-- {
		if y != 6 {
			got.WriteByte(b2c(m.at(8, y)))
		}
	}
	if got.String() != want {
		t.Errorf("format bits = %s, want %s", got.String(), want)
	}

	// The version 7 information is 000111110010010100, bit 17 first.
	want = "000111110010010100"
	got.Reset()
	for i := 17; i >= 0; i-- {
		got.WriteByte(b2c(m.at(m.size-11+i%3, i/3)))
	}
	if got.String() != want {
		t.Errorf("version bits = %s, want %s", got.String(), want)
	}
}

func b2c(b bool) byte {
	if b {
		return '1'
	}
	return '0'
}
