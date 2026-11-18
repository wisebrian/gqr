package gqr

import "slices"

// maskFuncs are the eight mask conditions, indexed by mask number. The
// module at column x, row y is inverted when the condition holds.
var maskFuncs = [8]func(x, y int) bool{
	func(x, y int) bool { return (x+y)%2 == 0 },
	func(x, y int) bool { return y%2 == 0 },
	func(x, y int) bool { return x%3 == 0 },
	func(x, y int) bool { return (x+y)%3 == 0 },
	func(x, y int) bool { return (x/3+y/2)%2 == 0 },
	func(x, y int) bool { return x*y%2+x*y%3 == 0 },
	func(x, y int) bool { return (x*y%2+x*y%3)%2 == 0 },
	func(x, y int) bool { return ((x+y)%2+x*y%3)%2 == 0 },
}

// applyMask inverts the data modules that a mask selects and writes the
// format information, which names the level and the mask.
func (m *matrix) applyMask(level Level, mask int) {
	f := maskFuncs[mask]
	for y := range m.size {
		for x := range m.size {
			if i := y*m.size + x; !m.reserved[i] && f(x, y) {
				m.dark[i] = !m.dark[i]
			}
		}
	}
	m.drawFormat(level.formatBits()<<3 | uint32(mask))
}

// applyBestMask tries every mask, keeps the one with the lowest penalty and
// returns its number.
func (m *matrix) applyBestMask(level Level) int {
	unmasked := slices.Clone(m.dark)
	best, bestPenalty := 0, -1
	for mask := range len(maskFuncs) {
		copy(m.dark, unmasked)
		m.applyMask(level, mask)
		if p := m.penalty(); bestPenalty < 0 || p < bestPenalty {
			best, bestPenalty = mask, p
		}
	}
	copy(m.dark, unmasked)
	m.applyMask(level, best)
	return best
}

// penalty scores the matrix with the four rules of ISO/IEC 18004 §7.8.3.
// Lower scores are easier for scanners to read.
func (m *matrix) penalty() int {
	n := m.size
	p := 0

	// Rules 1 and 3 on every row and column.
	for i := range n {
		p += linePenalty(n, func(j int) bool { return m.at(j, i) })
		p += linePenalty(n, func(j int) bool { return m.at(i, j) })
	}

	// Rule 2: 2x2 blocks of one color.
	for y := range n - 1 {
		for x := range n - 1 {
			c := m.at(x, y)
			if c == m.at(x+1, y) && c == m.at(x, y+1) && c == m.at(x+1, y+1) {
				p += 3
			}
		}
	}

	// Rule 4: deviation of the dark ratio from 50%, in steps of 5%.
	dark := 0
	for _, d := range m.dark {
		if d {
			dark++
		}
	}
	total := n * n
	p += 10 * (abs(dark*20-total*10) / total)
	return p
}

// finderLike is the 1:1:3:1:1 finder pattern followed by four light modules.
var finderLike = [11]bool{true, false, true, true, true, false, true, false, false, false, false}

// linePenalty applies rule 1 (runs of five or more modules of one color)
// and rule 3 (finder-like patterns) to a line of n modules. For rule 3, the
// quiet zone on either side of the line counts as light.
func linePenalty(n int, at func(int) bool) int {
	p := 0
	run := 1
	for j := 1; j <= n; j++ {
		if j < n && at(j) == at(j-1) {
			run++
			continue
		}
		if run >= 5 {
			p += run - 2
		}
		run = 1
	}

	// A pattern can start 4 modules before the line, with its light part in
	// the quiet zone, and end 4 modules after it.
	dark := func(j int) bool { return j >= 0 && j < n && at(j) }
	for j := -4; j+7 <= n; j++ {
		fwd, rev := true, true
		for k, want := range finderLike {
			v := dark(j + k)
			fwd = fwd && v == want
			rev = rev && v == finderLike[len(finderLike)-1-k]
		}
		if fwd {
			p += 40
		}
		if rev {
			p += 40
		}
	}
	return p
}
