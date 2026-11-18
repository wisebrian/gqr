package gqr

import "testing"

func TestShiftJIS(t *testing.T) {
	tests := []struct {
		r    rune
		code uint16 // 0 if Kanji mode cannot hold r
	}{
		{'　', 0x8140}, // ideographic space, the first code
		{'ア', 0x8341},
		{'Ω', 0x83B6},
		{'点', 0x935F},
		{'茗', 0xE4AA},
		{'熙', 0xEAA4}, // the last code
		{'A', 0},
		{'é', 0},
		{'〜', 0}, // the wave dash, which Shift JIS and Windows map differently
		{'～', 0},
		{'①', 0}, // only in Windows code page 932
	}
	for _, tt := range tests {
		code, ok := shiftJIS(tt.r)
		if ok != (tt.code != 0) || code != tt.code {
			t.Errorf("shiftJIS(%q) = %#x, %v; want %#x", tt.r, code, ok, tt.code)
		}
	}
	if n := len(kanjiCodes()); n != 6872 {
		t.Errorf("%d characters, want 6872", n)
	}
}
