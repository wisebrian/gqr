package reedsolomon

import (
	"bytes"
	"math/rand/v2"
	"testing"
)

// mul must agree with carry-less multiplication reduced by 0x11D.
func TestMul(t *testing.T) {
	for a := range 256 {
		for b := range 256 {
			want := 0
			for x, y := a, b; y > 0; y >>= 1 {
				if y&1 == 1 {
					want ^= x
				}
				x <<= 1
				if x&0x100 != 0 {
					x ^= 0x11D
				}
			}
			if got := mul(byte(a), byte(b)); int(got) != want {
				t.Fatalf("mul(%d, %d) = %d, want %d", a, b, got, want)
			}
		}
	}
}

// The "HELLO WORLD" version 1-M example from the thonky.com QR code tutorial.
func TestECC(t *testing.T) {
	data := []byte{32, 91, 11, 120, 209, 114, 220, 77, 67, 64, 236, 17, 236, 17, 236, 17}
	want := []byte{196, 35, 39, 119, 235, 215, 231, 226, 93, 23}
	if got := New(10).ECC(data); !bytes.Equal(got, want) {
		t.Fatalf("ECC\n got %v\nwant %v", got, want)
	}
}

// Data followed by its codewords is a multiple of the generator, so it has
// no remainder of its own.
func TestCodewordHasNoRemainder(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for _, n := range []int{7, 10, 13, 17, 22, 26, 28, 30} {
		enc := New(n)
		data := make([]byte, 40)
		for i := range data {
			data[i] = byte(r.Uint32())
		}
		codeword := append(data, enc.ECC(data)...)
		if rem := enc.ECC(codeword); !bytes.Equal(rem, make([]byte, n)) {
			t.Errorf("n=%d: remainder %v, want zeros", n, rem)
		}
	}
}
