package gqr

import (
	"errors"
	"fmt"
)

var (
	// ErrEmpty is returned when there is no data to encode.
	ErrEmpty = errors.New("gqr: no data to encode")

	// ErrTooLong is returned when the data does not fit in the largest
	// symbol (version 40) at the requested error-correction level.
	ErrTooLong = errors.New("gqr: data too long for a QR code")
)

// Code is an encoded QR symbol. Create one with Encode, EncodeECI or
// EncodeBytes.
type Code struct {
	version int
	level   Level
	mask    int
	size    int
	dark    []bool // row-major, size*size
}

// Version returns the symbol version, from 1 to 40.
func (c *Code) Version() int { return c.version }

// Level returns the error-correction level used, which may be higher than
// the one requested.
func (c *Code) Level() Level { return c.level }

// Mask returns the mask pattern (0–7) applied to the symbol.
func (c *Code) Mask() int { return c.mask }

// Size returns the width and height of the symbol in modules, without the
// quiet zone.
func (c *Code) Size() int { return c.size }

// Dark reports whether the module at column x, row y is dark. Modules
// outside the symbol, such as the quiet zone, are light.
func (c *Code) Dark(x, y int) bool {
	if x < 0 || y < 0 || x >= c.size || y >= c.size {
		return false
	}
	return c.dark[y*c.size+x]
}

// Encode encodes text in the smallest symbol that fits. The text is split
// into segments, each stored in the densest mode its characters allow:
// numeric for digits, alphanumeric for digits, uppercase letters and
// " $%*+-./:", Kanji for Japanese characters, and bytes for the rest, as
// UTF-8. Encode picks the split that takes the fewest bits, with one
// restriction: the characters outside ASCII go either all in Kanji mode or
// all in bytes, so that decoders cannot mistake UTF-8 for Shift JIS.
//
// level is a minimum: when the symbol has room to spare, the highest level
// that still fits is used, making the code more robust at no cost in size.
func Encode(text string, level Level) (*Code, error) {
	return encodeText(text, level, false)
}

// EncodeECI is like Encode, but when the code stores characters outside
// ASCII as bytes, an Extended Channel Interpretation (ECI) header declares
// them UTF-8. Decoders that follow the QR standard to the letter otherwise
// read bytes as ISO-8859-1. Most decoders, phone cameras among them,
// recognize UTF-8 without the header, and a few older ones do not support
// ECI, which is why Encode leaves it out.
func EncodeECI(text string, level Level) (*Code, error) {
	return encodeText(text, level, true)
}

// EncodeBytes is like Encode but always stores data in byte mode, for
// binary content.
func EncodeBytes(data []byte, level Level) (*Code, error) {
	segs := segments{{modeByte, data}}
	return encode(func(int) segments { return segs }, len(data), level)
}

func encodeText(text string, level Level, eci bool) (*Code, error) {
	split := func(version int) segments { return textSegments(text, version, eci) }
	return encode(split, len(text), level)
}

// maxChars is the most characters a symbol holds: 7089 digits, in version
// 40 at level L. Digits take the fewest bits per byte of text, so no longer
// text fits.
const maxChars = 7089

// encode finds the smallest version that holds the segments split returns
// for it, given n bytes of data.
func encode(split func(version int) segments, n int, level Level) (*Code, error) {
	if !level.valid() {
		return nil, fmt.Errorf("gqr: invalid error-correction level %d", int(level))
	}
	if n == 0 {
		return nil, ErrEmpty
	}
	tooLong := func() error {
		return fmt.Errorf("%w: %d bytes at level %v", ErrTooLong, n, level)
	}
	if n > maxChars {
		return nil, tooLong()
	}

	var segs segments
	version := minVersion
	for {
		// The best split only changes with the width of the count fields.
		if version == minVersion || sizeClass(version) != sizeClass(version-1) {
			segs = split(version)
		}
		if segs.fits(version, level) {
			break
		}
		if version == maxVersion {
			return nil, tooLong()
		}
		version++
	}
	for level < High && segs.fits(version, level+1) {
		level++
	}
	return newCode(segs, version, level, -1), nil
}

// newCode builds the symbol of a version and level that holds segs, with
// the given mask, or with the best one if mask is -1.
func newCode(segs segments, version int, level Level, mask int) *Code {
	m := newMatrix(version)
	m.drawCodewords(addErrorCorrection(segs.codewords(version, level), version, level))
	if mask < 0 {
		mask = m.applyBestMask(level)
	} else {
		m.applyMask(level, mask)
	}
	return &Code{version: version, level: level, mask: mask, size: m.size, dark: m.dark}
}
