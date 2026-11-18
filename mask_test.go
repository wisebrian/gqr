package gqr

import "testing"

func TestLinePenalty(t *testing.T) {
	tests := []struct {
		line string
		want int
	}{
		{"00000", 3}, // a run of 5
		{"0000000", 5},
		{"1011100", 0},
		// A finder-like pattern counts once for the light area before it
		// and once for the one after it.
		{"0000" + "1011101" + "0000", 80},
		{"1011101" + "0001", 40},
		// The quiet zone counts as light.
		{"1011101" + "0000", 80},
		{"0000" + "1011101", 80},
		{"1011101", 80},
	}
	for _, tt := range tests {
		at := func(j int) bool { return tt.line[j] == '1' }
		if got := linePenalty(len(tt.line), at); got != tt.want {
			t.Errorf("linePenalty(%s) = %d, want %d", tt.line, got, tt.want)
		}
	}
}
