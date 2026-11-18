package gqr

import (
	"slices"
	"testing"
)

func TestInterleave(t *testing.T) {
	got := interleave([][]int{{1, 2}, {3, 4}, {5, 6, 7}})
	if want := []int{1, 3, 5, 2, 4, 6, 7}; !slices.Equal(got, want) {
		t.Errorf("interleave = %v, want %v", got, want)
	}
}

// Every codeword belongs to exactly one block, and each block owns its data
// codewords plus its error-correction codewords.
func TestBlockOwners(t *testing.T) {
	for v := minVersion; v <= maxVersion; v++ {
		for l := Low; l <= High; l++ {
			owners := blockOwners(v, l)
			if len(owners) != totalCodewords(v) {
				t.Fatalf("%d-%v: %d codewords, want %d", v, l, len(owners), totalCodewords(v))
			}
			count := make([]int, numBlocks[l][v])
			for _, b := range owners {
				count[b]++
			}
			for b, n := range blockSizes(v, l) {
				if want := n + eccPerBlock[l][v]; count[b] != want {
					t.Fatalf("%d-%v: block %d owns %d codewords, want %d", v, l, b, count[b], want)
				}
			}
		}
	}
}
