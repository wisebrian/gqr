package gqr

import (
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

// The "HELLO WORLD" version 1-M example from the thonky.com QR code tutorial.
func TestHelloWorldCodewords(t *testing.T) {
	segs := textSegments("HELLO WORLD", 1, false)
	if got := modeLetters(segs); got != "A" {
		t.Fatalf("modes = %s, want A", got)
	}
	want := []byte{
		32, 91, 11, 120, 209, 114, 220, 77, 67, 64, 236, 17, 236, 17, 236, 17, // data
		196, 35, 39, 119, 235, 215, 231, 226, 93, 23, // error correction
	}
	if got := addErrorCorrection(segs.codewords(1, Medium), 1, Medium); !slices.Equal(got, want) {
		t.Fatalf("codewords\n got %v\nwant %v", got, want)
	}
}

func TestSegmentEncoding(t *testing.T) {
	tests := []struct {
		name string
		seg  segment
		want string
	}{
		// The examples from ISO/IEC 18004.
		{"numeric", segment{modeNumeric, []byte("01234567")}, "0001" + "0000001000" + "0000001100" + "0101011001" + "1000011"},
		{"alphanumeric", segment{modeAlphanumeric, []byte("AC-42")}, "0010" + "000000101" + "00111001110" + "11100111001" + "000010"},
		{"kanji", segment{modeKanji, []byte{0x93, 0x5F, 0xE4, 0xAA}}, "1000" + "00000010" + "0110110011111" + "1101010101010"}, // 点茗
		{"byte", segment{modeByte, []byte{0xC3, 0xA9}}, "0100" + "00000010" + "11000011" + "10101001"},                         // é in UTF-8
		{"eci", segment{modeECI, []byte{eciUTF8}}, "0111" + "00011010"},
	}
	for _, tt := range tests {
		var w bitWriter
		tt.seg.writeTo(&w, 1)
		var got strings.Builder
		for i := range w.n {
			got.WriteByte('0' + w.buf[i/8]>>(7-i%8)&1)
		}
		if got.String() != tt.want {
			t.Errorf("%s: bits\n got %s\nwant %s", tt.name, got.String(), tt.want)
		}
		if n := tt.seg.bitLen(1); n != len(tt.want) {
			t.Errorf("%s: bitLen = %d, want %d", tt.name, n, len(tt.want))
		}
	}
}

func TestSplitText(t *testing.T) {
	tests := []struct {
		text  string
		kanji bool
		want  string // the modes of the segments
	}{
		{"01234567", false, "N"},
		{"HELLO WORLD", false, "A"},
		{"hello", false, "B"},
		{"1234567890ABC", false, "NA"},
		{"abc12", false, "B"},          // 2 digits at the end do not pay for a numeric segment,
		{"abc1234", false, "BN"},       // 4 do;
		{"abc12345def", false, "B"},    // in the middle, 5 do not,
		{"abc123456def", false, "BNB"}, // 6 do.
		// Segments end on whole bits: ANB looks shorter in sixths of a bit,
		// but takes 122 bits, one more than AB.
		{"1AA1111111111AAAa", false, "AB"},
		{"日本語", false, "B"},
		{"日本語", true, "K"},
		{"東京2024", true, "KN"},
	}
	for _, tt := range tests {
		segs := splitText(tt.text, 1, tt.kanji)
		if got := modeLetters(segs); got != tt.want {
			t.Errorf("splitText(%q, %v): modes %s, want %s", tt.text, tt.kanji, got, tt.want)
		}
		if got := joined(segs); got != tt.text {
			t.Errorf("splitText(%q, %v) holds %q", tt.text, tt.kanji, got)
		}
	}
}

func TestTextSegments(t *testing.T) {
	tests := []struct {
		text string
		eci  bool
		want string
	}{
		{"hello", true, "B"}, // ASCII needs no ECI
		{"naïve café", false, "B"},
		{"naïve café", true, "EB"},
		{"日本語の文章", false, "K"},
		{"日本語の文章", true, "K"},    // Kanji mode needs no ECI
		{"日本語の文章 é", false, "B"}, // é has no Shift JIS code, so all goes in UTF-8
		{"Ωmega", false, "B"},    // Kanji mode would be longer
		{"\xff\xfe", false, "B"}, // invalid UTF-8 is kept as it is
	}
	for _, tt := range tests {
		segs := textSegments(tt.text, 1, tt.eci)
		if got := modeLetters(segs); got != tt.want {
			t.Errorf("textSegments(%q, %v): modes %s, want %s", tt.text, tt.eci, got, tt.want)
		}
		if got := joined(segs); got != tt.text {
			t.Errorf("textSegments(%q, %v) holds %q", tt.text, tt.eci, got)
		}
	}
}

// splitText must find the shortest encoding: compare it with every way of
// putting the characters of short random texts in modes.
func TestSplitTextOptimal(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	alphabet := []rune("07AZ :az日本Ωé")
	for range 300 {
		text := make([]rune, 1+r.IntN(6))
		for i := range text {
			text[i] = alphabet[r.IntN(len(alphabet))]
		}
		for _, version := range []int{1, 10, 27} {
			for _, kanji := range []bool{false, true} {
				if kanji && slices.Contains(text, 'é') {
					continue // é has no Shift JIS code
				}
				got := splitText(string(text), version, kanji).bitLen(version)
				if want := shortest(text, version, kanji); got != want {
					t.Errorf("splitText(%q, %d, %v): %d bits, want %d", string(text), version, kanji, got, want)
				}
			}
		}
	}
}

// shortest returns the fewest bits that encode text at a version, trying
// every mode for every character.
func shortest(text []rune, version int, kanji bool) int {
	best := math.MaxInt
	modes := make([]mode, len(text))
	var try func(i int)
	try = func(i int) {
		if i == len(text) {
			best = min(best, group(text, modes).bitLen(version))
			return
		}
		c := text[i]
		for m := range dataModes {
			var ok bool
			switch {
			case c >= utf8.RuneSelf && kanji:
				ok = m == modeKanji
			case c >= utf8.RuneSelf:
				ok = m == modeByte
			case m == modeNumeric:
				ok = '0' <= c && c <= '9'
			case m == modeAlphanumeric:
				ok = strings.ContainsRune(alphanumericCharset, c)
			default:
				ok = m == modeByte
			}
			if ok {
				modes[i] = m
				try(i + 1)
			}
		}
	}
	try(0)
	return best
}

// group puts the characters of text in segments of the given modes.
func group(text []rune, modes []mode) segments {
	var segs segments
	for i, c := range text {
		m := modes[i]
		if len(segs) == 0 || segs[len(segs)-1].mode != m {
			segs = append(segs, segment{mode: m})
		}
		s := &segs[len(segs)-1]
		if m == modeKanji {
			code, _ := shiftJIS(c)
			s.data = append(s.data, byte(code>>8), byte(code))
		} else {
			s.data = utf8.AppendRune(s.data, c)
		}
	}
	return segs
}

// modeLetters returns the modes of segs as letters: N, A, B, K, and E for
// an ECI.
func modeLetters(segs segments) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteByte("NABKE"[s.mode])
	}
	return b.String()
}

// joined returns the text that segs hold.
func joined(segs segments) string {
	chars := make(map[uint16]rune)
	for r, code := range kanjiCodes() {
		chars[code] = r
	}
	var b strings.Builder
	for _, s := range segs {
		switch s.mode {
		case modeECI:
		case modeKanji:
			for i := 0; i < len(s.data); i += 2 {
				b.WriteRune(chars[uint16(s.data[i])<<8|uint16(s.data[i+1])])
			}
		default:
			b.Write(s.data)
		}
	}
	return b.String()
}
