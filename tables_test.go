package gqr

import "testing"

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
