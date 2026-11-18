package gqr

const (
	minVersion = 1
	maxVersion = 40
)

// eccPerBlock[level][version] is the number of error-correction codewords
// in each block. Index 0 is unused.
var eccPerBlock = [4][maxVersion + 1]int{
	{0, 7, 10, 15, 20, 26, 18, 20, 24, 30, 18, 20, 24, 26, 30, 22, 24, 28, 30, 28, 28, 28, 28, 30, 30, 26, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	{0, 10, 16, 26, 18, 24, 16, 18, 22, 22, 26, 30, 22, 22, 24, 24, 28, 28, 26, 26, 26, 26, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28},
	{0, 13, 22, 18, 26, 18, 24, 18, 22, 20, 24, 28, 26, 24, 20, 30, 24, 28, 28, 26, 30, 28, 30, 30, 30, 30, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	{0, 17, 28, 22, 16, 22, 28, 26, 26, 24, 28, 24, 28, 22, 24, 24, 30, 28, 28, 26, 28, 30, 24, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
}

// numBlocks[level][version] is the number of error-correction blocks.
// Index 0 is unused.
var numBlocks = [4][maxVersion + 1]int{
	{0, 1, 1, 1, 1, 1, 2, 2, 2, 2, 4, 4, 4, 4, 4, 6, 6, 6, 6, 7, 8, 8, 9, 9, 10, 12, 12, 12, 13, 14, 15, 16, 17, 18, 19, 19, 20, 21, 22, 24, 25},
	{0, 1, 1, 1, 2, 2, 4, 4, 4, 5, 5, 5, 8, 9, 9, 10, 10, 11, 13, 14, 16, 17, 17, 18, 20, 21, 23, 25, 26, 28, 29, 31, 33, 35, 37, 38, 40, 43, 45, 47, 49},
	{0, 1, 1, 2, 2, 4, 4, 6, 6, 8, 8, 8, 10, 12, 16, 12, 17, 16, 18, 21, 20, 23, 23, 25, 27, 29, 34, 34, 35, 38, 40, 43, 45, 48, 51, 53, 56, 59, 62, 65, 68},
	{0, 1, 1, 2, 4, 4, 4, 5, 6, 8, 8, 11, 11, 16, 16, 18, 16, 19, 21, 25, 25, 25, 34, 30, 32, 35, 37, 40, 42, 45, 48, 51, 54, 57, 60, 63, 66, 70, 74, 77, 81},
}

// symbolSize returns the width (and height) in modules of a version.
func symbolSize(version int) int { return 4*version + 17 }

// rawDataModules returns the number of modules available for data and
// error correction (everything except function patterns), including
// remainder bits.
func rawDataModules(version int) int {
	n := (16*version+128)*version + 64
	if version >= 2 {
		align := version/7 + 2
		n -= (25*align-10)*align - 55
		if version >= 7 {
			n -= 36 // two 6x3 version information areas
		}
	}
	return n
}

// totalCodewords returns the number of 8-bit codewords in a version.
func totalCodewords(version int) int { return rawDataModules(version) / 8 }

// dataCodewords returns how many data codewords fit in a version at a level.
func dataCodewords(version int, level Level) int {
	return totalCodewords(version) - eccPerBlock[level][version]*numBlocks[level][version]
}

// alignmentPositions returns the row/column centers of the alignment
// patterns for a version, in ascending order.
func alignmentPositions(version int) []int {
	if version == 1 {
		return nil
	}
	count := version/7 + 2
	step := (version*8 + count*3 + 5) / (count*4 - 4) * 2
	pos := make([]int, count)
	pos[0] = 6
	for i, p := count-1, symbolSize(version)-7; i >= 1; i, p = i-1, p-step {
		pos[i] = p
	}
	return pos
}

// correctableErrors returns how many erroneous codewords each block of a
// symbol can repair. A few small symbols reserve some error-correction
// codewords for detecting misdecodes instead (ISO/IEC 18004, Table 9).
func correctableErrors(version int, level Level) int {
	reserved := 0
	switch {
	case version == 1 && level == Low:
		reserved = 3
	case version == 1 && level == Medium, version == 2 && level == Low:
		reserved = 2
	case version == 1, version == 3 && level == Low:
		reserved = 1
	}
	return (eccPerBlock[level][version] - reserved) / 2
}
