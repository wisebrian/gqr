// Package reedsolomon computes the Reed–Solomon error-correction codewords
// of QR codes, over GF(2^8) with the polynomial x^8 + x^4 + x^3 + x^2 + 1.
package reedsolomon

// exp and log are the exponent and logarithm tables of GF(2^8) for the
// generator α = 2. exp is doubled so that mul can index it with the sum of
// two logarithms without reducing modulo 255.
var exp, log = tables()

func tables() (exp [510]byte, log [256]byte) {
	x := 1
	for i := range 255 {
		exp[i], exp[i+255] = byte(x), byte(x)
		log[x] = byte(i)
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11D
		}
	}
	return exp, log
}

// mul multiplies a and b in GF(2^8).
func mul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return exp[int(log[a])+int(log[b])]
}

// An Encoder computes a fixed number of error-correction codewords.
type Encoder struct {
	gen []byte // generator polynomial, highest power first, leading 1 omitted
}

// New returns an Encoder that produces n > 0 error-correction codewords,
// using the generator polynomial (x - α^0)(x - α^1)…(x - α^(n-1)).
func New(n int) *Encoder {
	gen := make([]byte, n)
	gen[n-1] = 1
	root := byte(1)
	for range n {
		// Multiply by (x - root); subtraction is addition (XOR) in GF(2^8).
		for j := range gen {
			gen[j] = mul(gen[j], root)
			if j+1 < n {
				gen[j] ^= gen[j+1]
			}
		}
		root = mul(root, 2)
	}
	return &Encoder{gen}
}

// ECC returns the error-correction codewords for data: the remainder of
// data(x)·x^n divided by the generator polynomial.
func (e *Encoder) ECC(data []byte) []byte {
	rem := make([]byte, len(e.gen))
	for _, b := range data {
		factor := b ^ rem[0]
		copy(rem, rem[1:])
		rem[len(rem)-1] = 0
		for i, g := range e.gen {
			rem[i] ^= mul(g, factor)
		}
	}
	return rem
}
