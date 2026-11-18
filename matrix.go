package gqr

import "iter"

// matrix is a symbol under construction. Function modules (the finder,
// timing, alignment, format and version patterns) are marked reserved, so
// that data placement and masking skip them.
type matrix struct {
	version  int
	size     int
	dark     []bool
	reserved []bool
}

func newMatrix(version int) *matrix {
	size := symbolSize(version)
	m := &matrix{
		version:  version,
		size:     size,
		dark:     make([]bool, size*size),
		reserved: make([]bool, size*size),
	}
	m.drawFunctionPatterns()
	return m
}

func (m *matrix) at(x, y int) bool { return m.dark[y*m.size+x] }

// setFunction sets a function module and marks it reserved.
func (m *matrix) setFunction(x, y int, dark bool) {
	i := y*m.size + x
	m.dark[i] = dark
	m.reserved[i] = true
}

func (m *matrix) drawFunctionPatterns() {
	// Timing patterns.
	for i := range m.size {
		m.setFunction(6, i, i%2 == 0)
		m.setFunction(i, 6, i%2 == 0)
	}

	// Finder patterns with their separators.
	m.drawFinder(3, 3)
	m.drawFinder(m.size-4, 3)
	m.drawFinder(3, m.size-4)

	// Alignment patterns, except the three that would overlap the finders.
	pos := alignmentPositions(m.version)
	last := len(pos) - 1
	for i, cx := range pos {
		for j, cy := range pos {
			if (i == 0 && j == 0) || (i == 0 && j == last) || (i == last && j == 0) {
				continue
			}
			m.drawAlignment(cx, cy)
		}
	}

	// Reserve the format areas now; the real bits are drawn after masking.
	m.drawFormat(0)
	m.drawVersion()
}

func (m *matrix) drawFinder(cx, cy int) {
	for dy := -4; dy <= 4; dy++ {
		for dx := -4; dx <= 4; dx++ {
			x, y := cx+dx, cy+dy
			if x < 0 || y < 0 || x >= m.size || y >= m.size {
				continue
			}
			d := max(abs(dx), abs(dy))
			m.setFunction(x, y, d != 2 && d != 4)
		}
	}
}

func (m *matrix) drawAlignment(cx, cy int) {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			m.setFunction(cx+dx, cy+dy, max(abs(dx), abs(dy)) != 1)
		}
	}
}

// drawFormat writes both copies of the 15-bit format information for a
// 5-bit value: the level bits followed by the mask number.
func (m *matrix) drawFormat(data uint32) {
	// Append a BCH(15,5) code, then XOR with a fixed pattern so that the
	// result is never all zeros.
	rem := data
	for range 10 {
		rem = rem<<1 ^ (rem>>9)*0x537
	}
	bits := (data<<10 | rem) ^ 0x5412
	bit := func(i int) bool { return bits>>i&1 == 1 }

	// Copy around the top-left finder.
	for i := range 6 {
		m.setFunction(8, i, bit(i))
	}
	m.setFunction(8, 7, bit(6))
	m.setFunction(8, 8, bit(7))
	m.setFunction(7, 8, bit(8))
	for i := 9; i < 15; i++ {
		m.setFunction(14-i, 8, bit(i))
	}

	// Copy split between the top-right and bottom-left finders.
	for i := range 8 {
		m.setFunction(m.size-1-i, 8, bit(i))
	}
	for i := 8; i < 15; i++ {
		m.setFunction(8, m.size-15+i, bit(i))
	}
	m.setFunction(8, m.size-8, true) // always-dark module
}

// drawVersion writes both copies of the 18-bit version information carried
// by symbols of version 7 and up: the version followed by a BCH(18,6) code.
func (m *matrix) drawVersion() {
	if m.version < 7 {
		return
	}
	rem := uint32(m.version)
	for range 12 {
		rem = rem<<1 ^ (rem>>11)*0x1F25
	}
	bits := uint32(m.version)<<12 | rem
	for i := range 18 {
		dark := bits>>i&1 == 1
		a, b := m.size-11+i%3, i/3
		m.setFunction(a, b, dark)
		m.setFunction(b, a, dark)
	}
}

// dataModules yields the positions of the data modules in placement order:
// two-column strips from right to left, alternately upward and downward,
// skipping the vertical timing pattern.
func (m *matrix) dataModules() iter.Seq2[int, int] {
	return func(yield func(x, y int) bool) {
		for right := m.size - 1; right >= 1; right -= 2 {
			if right == 6 {
				right = 5
			}
			upward := (right+1)&2 == 0
			for vert := range m.size {
				y := vert
				if upward {
					y = m.size - 1 - vert
				}
				for x := right; x >= right-1; x-- {
					if !m.reserved[y*m.size+x] && !yield(x, y) {
						return
					}
				}
			}
		}
	}
}

// drawCodewords places the bits of data, most significant first, along
// dataModules. The few remainder modules left over stay light.
func (m *matrix) drawCodewords(data []byte) {
	i := 0
	for x, y := range m.dataModules() {
		if i == len(data)*8 {
			return
		}
		m.dark[y*m.size+x] = data[i/8]>>(7-i%8)&1 == 1
		i++
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
