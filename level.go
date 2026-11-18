package gqr

import (
	"fmt"
	"strings"
)

// Level is the error-correction level of a QR code. Higher levels survive
// more damage but leave less room for data.
type Level int

const (
	Low      Level = iota // recovers about 7% of the codewords
	Medium                // recovers about 15%
	Quartile              // recovers about 25%
	High                  // recovers about 30%
)

func (l Level) valid() bool { return l >= Low && l <= High }

// formatBits returns the level's 2-bit code in the format information.
func (l Level) formatBits() uint32 {
	return [...]uint32{Low: 0b01, Medium: 0b00, Quartile: 0b11, High: 0b10}[l]
}

// String returns "L", "M", "Q" or "H".
func (l Level) String() string {
	if !l.valid() {
		return fmt.Sprintf("Level(%d)", int(l))
	}
	return "LMQH"[l : l+1]
}

// ParseLevel parses "L", "M", "Q" or "H", in either case.
func ParseLevel(s string) (Level, error) {
	for l := Low; l <= High; l++ {
		if strings.EqualFold(s, l.String()) {
			return l, nil
		}
	}
	return 0, fmt.Errorf("gqr: unknown error-correction level %q", s)
}
