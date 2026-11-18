package gqr

import (
	"slices"

	"github.com/wisebrian/gqr/internal/reedsolomon"
)

// blockSizes returns the number of data codewords in each error-correction
// block. The last blocks may hold one more codeword than the first.
func blockSizes(version int, level Level) []int {
	n := numBlocks[level][version]
	data := dataCodewords(version, level)
	sizes := make([]int, n)
	for i := range sizes {
		sizes[i] = data / n
		if i >= n-data%n {
			sizes[i]++
		}
	}
	return sizes
}

// addErrorCorrection splits data into blocks, computes the Reed–Solomon
// codewords of each and interleaves everything into the final sequence.
func addErrorCorrection(data []byte, version int, level Level) []byte {
	sizes := blockSizes(version, level)
	enc := reedsolomon.New(eccPerBlock[level][version])
	dataBlocks := make([][]byte, len(sizes))
	eccBlocks := make([][]byte, len(sizes))
	for i, n := range sizes {
		dataBlocks[i], data = data[:n], data[n:]
		eccBlocks[i] = enc.ECC(dataBlocks[i])
	}
	return append(interleave(dataBlocks), interleave(eccBlocks)...)
}

// blockOwners returns, for each codeword of the final sequence, the index
// of the block it belongs to. It mirrors addErrorCorrection.
func blockOwners(version int, level Level) []int {
	sizes := blockSizes(version, level)
	ecc := eccPerBlock[level][version]
	dataBlocks := make([][]int, len(sizes))
	eccBlocks := make([][]int, len(sizes))
	for i, n := range sizes {
		dataBlocks[i] = slices.Repeat([]int{i}, n)
		eccBlocks[i] = slices.Repeat([]int{i}, ecc)
	}
	return append(interleave(dataBlocks), interleave(eccBlocks)...)
}

// interleave takes the first element of every block, then the second, and
// so on, skipping blocks that have run out.
func interleave[T any](blocks [][]T) []T {
	var out []T
	for i := 0; ; i++ {
		n := len(out)
		for _, b := range blocks {
			if i < len(b) {
				out = append(out, b[i])
			}
		}
		if len(out) == n {
			return out
		}
	}
}
