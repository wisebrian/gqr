package gqr

import "sync"

// shiftJIS returns the Shift JIS code of a character that Kanji mode can
// hold.
func shiftJIS(r rune) (code uint16, ok bool) {
	code, ok = kanjiCodes()[r]
	return code, ok
}

// kanjiCodes maps the characters of jisRows to their Shift JIS codes. It is
// built on first use.
var kanjiCodes = sync.OnceValue(func() map[rune]uint16 {
	codes := make(map[rune]uint16)
	for ku, row := range jisRows {
		ten := 0
		for _, r := range row {
			ten++
			if r != '_' {
				codes[r] = sjis(ku, ten)
			}
		}
	}
	return codes
})

// sjis returns the Shift JIS code of the JIS X 0208 character in row ku,
// cell ten, both counted from 1.
func sjis(ku, ten int) uint16 {
	// A lead byte covers two rows: the odd one with trail bytes 0x40–0x9E,
	// skipping 0x7F, and the even one with 0x9F–0xFC. Lead bytes 0xA0–0xDF
	// are left out, as they stand for single-byte katakana.
	lead := (ku+1)/2 + 0x80
	if ku >= 63 {
		lead += 0x40
	}
	trail := ten + 0x9E
	if ku%2 == 1 {
		trail = ten + 0x3F
		if trail >= 0x7F {
			trail++
		}
	}
	return uint16(lead<<8 | trail)
}
