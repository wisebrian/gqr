package gqr

import (
	"math"
	"strings"
	"unicode/utf8"
)

// mode is a QR data encoding mode.
type mode int

const (
	modeNumeric mode = iota
	modeAlphanumeric
	modeByte
	modeKanji
	modeECI // not a data mode: names the character set of the bytes after it
)

// dataModes is the number of modes that hold characters.
const dataModes = modeKanji + 1

const alphanumericCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

// eciUTF8 is the ECI designator of UTF-8.
const eciUTF8 = 26

func (m mode) indicator() uint32 {
	return [...]uint32{0b0001, 0b0010, 0b0100, 0b1000, 0b0111}[m]
}

// countBits returns the width of the character-count field.
func (m mode) countBits(version int) int {
	return [dataModes][3]int{
		modeNumeric:      {10, 12, 14},
		modeAlphanumeric: {9, 11, 13},
		modeByte:         {8, 16, 16},
		modeKanji:        {8, 10, 12},
	}[m][sizeClass(version)]
}

// sizeClass returns 0 for versions 1–9, 1 for 10–26 and 2 for 27–40. The
// character-count fields have the same widths within a class.
func sizeClass(version int) int {
	switch {
	case version <= 9:
		return 0
	case version <= 26:
		return 1
	default:
		return 2
	}
}

// segment is a run of characters encoded in a single mode.
type segment struct {
	mode mode
	// data holds the characters: one byte each in numeric, alphanumeric and
	// byte mode, a two-byte Shift JIS code each in Kanji mode. An ECI holds
	// its designator.
	data []byte
}

// count returns the number of characters, for the character-count field.
func (s segment) count() int {
	if s.mode == modeKanji {
		return len(s.data) / 2
	}
	return len(s.data)
}

// bitLen returns the encoded length in bits for a version, or -1 if the
// character count does not fit in the count field.
func (s segment) bitLen(version int) int {
	if s.mode == modeECI {
		return 4 + 8 // a designator below 128 takes one byte
	}
	n, cb := s.count(), s.mode.countBits(version)
	if n >= 1<<cb {
		return -1
	}
	var dataBits int
	switch s.mode {
	case modeNumeric:
		dataBits = 10*(n/3) + [...]int{0, 4, 7}[n%3]
	case modeAlphanumeric:
		dataBits = 11*(n/2) + 6*(n%2)
	case modeByte:
		dataBits = 8 * n
	case modeKanji:
		dataBits = 13 * n
	}
	return 4 + cb + dataBits
}

// writeTo appends the encoded segment to w.
func (s segment) writeTo(w *bitWriter, version int) {
	w.write(s.mode.indicator(), 4)
	if s.mode == modeECI {
		w.write(uint32(s.data[0]), 8)
		return
	}
	w.write(uint32(s.count()), s.mode.countBits(version))

	switch s.mode {
	case modeNumeric:
		for i := 0; i < len(s.data); i += 3 {
			chunk := s.data[i:min(i+3, len(s.data))]
			var v uint32
			for _, c := range chunk {
				v = v*10 + uint32(c-'0')
			}
			w.write(v, len(chunk)*3+1) // 3 digits→10, 2→7, 1→4
		}
	case modeAlphanumeric:
		for i := 0; i < len(s.data); i += 2 {
			v := uint32(strings.IndexByte(alphanumericCharset, s.data[i]))
			if i+1 < len(s.data) {
				v = v*45 + uint32(strings.IndexByte(alphanumericCharset, s.data[i+1]))
				w.write(v, 11)
			} else {
				w.write(v, 6)
			}
		}
	case modeByte:
		for _, b := range s.data {
			w.write(uint32(b), 8)
		}
	case modeKanji:
		for i := 0; i < len(s.data); i += 2 {
			// Subtracting 0x8140, or 0xC140 from codes of the second range,
			// leaves a high byte below 0x2B and a low byte below 0xC0, which
			// combine into 13 bits as high*0xC0 + low.
			c := uint32(s.data[i])<<8 | uint32(s.data[i+1])
			if c >= 0xE040 {
				c -= 0xC140
			} else {
				c -= 0x8140
			}
			w.write((c>>8)*0xC0+c&0xFF, 13)
		}
	}
}

// segments are the contents of a symbol, in order.
type segments []segment

// bitLen returns the encoded length in bits for a version, or math.MaxInt
// if a character count does not fit in its field.
func (ss segments) bitLen(version int) int {
	total := 0
	for _, s := range ss {
		n := s.bitLen(version)
		if n < 0 {
			return math.MaxInt
		}
		total += n
	}
	return total
}

// fits reports whether the segments fit in a symbol of the given version
// and level.
func (ss segments) fits(version int, level Level) bool {
	return ss.bitLen(version) <= dataCodewords(version, level)*8
}

// codewords returns the data codewords of a symbol holding the segments:
// the encoded segments, a terminator and padding up to its capacity.
func (ss segments) codewords(version int, level Level) []byte {
	capacity := dataCodewords(version, level) * 8
	var w bitWriter
	for _, s := range ss {
		s.writeTo(&w, version)
	}
	w.write(0, min(4, capacity-w.n)) // terminator
	w.write(0, (8-w.n%8)%8)          // pad to a byte boundary
	for pad := byte(0xEC); w.n < capacity; pad ^= 0xEC ^ 0x11 {
		w.write(uint32(pad), 8) // alternate 0xEC and 0x11
	}
	return w.buf
}

// textSegments returns the segments that encode text in the fewest bits at
// a version. The characters outside ASCII all go in Kanji mode, if they can
// and it is shorter, or all in byte mode as UTF-8, preceded by an ECI that
// says so if eci is set. Mixing the two could lead decoders that guess the
// character set of the bytes to read them as Shift JIS.
func textSegments(text string, version int, eci bool) segments {
	ascii, japanese := true, true
	for _, r := range text {
		if r >= utf8.RuneSelf {
			ascii = false
			if _, ok := shiftJIS(r); !ok {
				japanese = false
				break
			}
		}
	}
	segs := splitText(text, version, false)
	if !ascii && eci {
		segs = append(segments{{modeECI, []byte{eciUTF8}}}, segs...)
	}
	if !ascii && japanese {
		if alt := splitText(text, version, true); alt.bitLen(version) < segs.bitLen(version) {
			segs = alt
		}
	}
	return segs
}

// splitText divides text into the segments that encode it in the fewest
// bits at a version. Any character can go in byte mode, as UTF-8, and
// digits and alphanumeric characters can also go in those modes. With kanji
// set, the characters outside ASCII go in Kanji mode instead, and must all
// have a Shift JIS code.
func splitText(text string, version int, kanji bool) segments {
	// The search reads the text one character at a time, keeping for each
	// mode the cheapest encoding so far whose last segment, still open, is
	// in that mode. Costs are in sixths of a bit, so that digits (10 bits
	// for 3) and alphanumeric characters (11 bits for 2) cost whole units;
	// a segment rounds up to whole bits when it closes.
	var head [dataModes]int // the mode indicator and character count
	for m := range dataModes {
		head[m] = (4 + m.countBits(version)) * 6
	}
	cost := head
	// modes[i][m] is the mode of character i in the cheapest encoding of
	// the text up to it that ends with a segment in mode m.
	var modes [][dataModes]mode
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		i += size

		// The character joins the open segment of each mode that holds it...
		var add [dataModes]int // its cost in each mode, 0 if impossible
		switch {
		case r >= utf8.RuneSelf && kanji:
			add[modeKanji] = 13 * 6
		case r >= utf8.RuneSelf:
			add[modeByte] = 8 * 6 * size
		default:
			add[modeByte] = 8 * 6
			if strings.ContainsRune(alphanumericCharset, r) {
				add[modeAlphanumeric] = 11 * 3
			}
			if '0' <= r && r <= '9' {
				add[modeNumeric] = 10 * 2
			}
		}
		var in, next [dataModes]int
		var from [dataModes]mode
		for m := range dataModes {
			in[m] = -1
			if add[m] > 0 {
				in[m] = cost[m] + add[m]
			}
			next[m], from[m] = in[m], m
		}
		// ...and the segment may then close, for one of another mode to open.
		for m := range dataModes {
			for k := range dataModes {
				if in[k] < 0 {
					continue
				}
				if c := (in[k]+5)/6*6 + head[m]; next[m] < 0 || c < next[m] {
					next[m], from[m] = c, k
				}
			}
		}
		cost = next
		modes = append(modes, from)
	}

	// Follow the cheapest encoding of the whole text back to its start.
	m := modeNumeric
	for k := range dataModes {
		if cost[k] < cost[m] {
			m = k
		}
	}
	charModes := make([]mode, len(modes))
	for i := len(modes) - 1; i >= 0; i-- {
		m = modes[i][m]
		charModes[i] = m
	}

	// Group the characters by mode.
	var segs segments
	i := 0
	for _, m := range charModes {
		r, size := utf8.DecodeRuneInString(text[i:])
		if len(segs) == 0 || segs[len(segs)-1].mode != m {
			segs = append(segs, segment{mode: m})
		}
		s := &segs[len(segs)-1]
		if m == modeKanji {
			code, _ := shiftJIS(r)
			s.data = append(s.data, byte(code>>8), byte(code))
		} else {
			s.data = append(s.data, text[i:i+size]...)
		}
		i += size
	}
	return segs
}

// bitWriter accumulates bits, most significant first.
type bitWriter struct {
	buf []byte
	n   int // number of bits written
}

// write appends the low n bits of v.
func (w *bitWriter) write(v uint32, n int) {
	for i := n - 1; i >= 0; i-- {
		if w.n%8 == 0 {
			w.buf = append(w.buf, 0)
		}
		if v>>i&1 == 1 {
			w.buf[w.n/8] |= 0x80 >> (w.n % 8)
		}
		w.n++
	}
}
