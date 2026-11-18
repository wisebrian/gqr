package gqr

import "strings"

// String renders the code for a terminal using Unicode half blocks, two
// rows of modules per line. Blocks stand for light modules, so the output
// scans on terminals with a dark background; use StringInverted on light
// ones.
func (c *Code) String() string { return c.text(false) }

// StringInverted is like String but draws dark modules as blocks, for
// terminals with a light background.
func (c *Code) StringInverted() string { return c.text(true) }

func (c *Code) text(invert bool) string {
	const q = QuietZone
	blocks := [4]string{" ", "▄", "▀", "█"} // indexed by top<<1 | bottom
	var sb strings.Builder
	for y := -q; y < c.size+q; y += 2 {
		for x := -q; x < c.size+q; x++ {
			top := c.Dark(x, y) == invert
			// The height is odd, so the last line has no bottom half.
			bottom := c.Dark(x, y+1) == invert && y+1 < c.size+q
			sb.WriteString(blocks[b2i(top)<<1|b2i(bottom)])
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
