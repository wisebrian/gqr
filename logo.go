package gqr

import (
	"errors"
	"image"
	"math/bits"
	"slices"
)

// ErrLogoTooLarge is returned when a logo would hide more of a symbol than
// error correction can repair.
var ErrLogoTooLarge = errors.New("gqr: logo too large for the error-correction level")

// defaultLogoSize is the side of a logo, as a fraction of the symbol width,
// used when Options.LogoSize is zero.
const defaultLogoSize = 0.2

// logoHole returns the centered square of modules hidden by a logo whose
// side is size times the symbol width.
func (c *Code) logoHole(size float64) (image.Rectangle, error) {
	if size <= 0 {
		size = defaultLogoSize
	}
	if !(size < 1) { // written this way to reject NaN as well
		return image.Rectangle{}, ErrLogoTooLarge
	}
	n := int(size*float64(c.size)) | 1 // odd like the symbol, so it centers on a module
	a := (c.size - n) / 2
	hole := image.Rect(a, a, a+n, a+n)
	if !c.canHide(hole) {
		return image.Rectangle{}, ErrLogoTooLarge
	}
	return hole, nil
}

// canHide reports whether every module in r can be made unreadable, as a
// logo does, while the symbol stays decodable with error correction to
// spare. The accounting is exact rather than a rule of thumb: r must avoid
// the patterns that decoders use to find and read the symbol, and the
// codewords hidden from each block may use at most half of the errors it can
// repair. The other half is kept for real damage, such as smudges or glare,
// and for scanners that misread modules next to a large logo.
func (c *Code) canHide(r image.Rectangle) bool {
	// The finder, timing, format and version patterns all lie within 9
	// modules of an edge. Alignment patterns may be hidden; the bottom-right
	// one, which decoders use to correct perspective, is never in this area.
	if !r.In(image.Rect(9, 9, c.size-9, c.size-9)) {
		return false
	}

	owners := blockOwners(c.version, c.level)
	hidden := make([]bool, len(owners))
	lost := make([]int, numBlocks[c.level][c.version])
	bit := 0
	for x, y := range newMatrix(c.version).dataModules() {
		k := bit / 8 // the codeword this module belongs to
		bit++
		if k == len(owners) {
			break // only remainder bits are left
		}
		if image.Pt(x, y).In(r) && !hidden[k] {
			hidden[k] = true
			lost[owners[k]]++
		}
	}
	return slices.Max(lost) <= correctableErrors(c.version, c.level)/2
}

// fit scales img to the largest size that fits in box, keeping its aspect
// ratio, and returns it with the rectangle that centers it in box.
func fit(img image.Image, box image.Rectangle) (*image.RGBA, image.Rectangle) {
	b := img.Bounds()
	if b.Empty() || box.Empty() {
		return image.NewRGBA(image.Rectangle{}), image.Rectangle{}
	}
	w, h := box.Dx(), box.Dy()
	if wider(b, box) {
		h = max(1, mulDiv(w, b.Dy(), b.Dx()))
	} else {
		w = max(1, mulDiv(h, b.Dx(), b.Dy()))
	}
	at := box.Min.Add(image.Pt((box.Dx()-w)/2, (box.Dy()-h)/2))
	return resize(img, w, h), image.Rectangle{Min: at, Max: at.Add(image.Pt(w, h))}
}

// wider reports whether r is proportionally wider than box.
func wider(r, box image.Rectangle) bool {
	// Compare r.Dx()/r.Dy() with box.Dx()/box.Dy() as 128-bit products,
	// which cannot overflow, however large r is.
	ah, al := bits.Mul64(uint64(r.Dx()), uint64(box.Dy()))
	bh, bl := bits.Mul64(uint64(r.Dy()), uint64(box.Dx()))
	return ah > bh || (ah == bh && al > bl)
}

// resize scales img to w×h pixels. Each output pixel is the average of the
// input pixels it covers (a box filter), so large logos shrink smoothly and
// small ones grow by repeating pixels. An output pixel that covers more than
// 4×4 input pixels averages 4×4 of them, evenly spread, so that even an
// image too large to hold in memory takes no longer than a small one.
func resize(img image.Image, w, h int) *image.RGBA {
	const most = 4 // input pixels averaged per output pixel, along each axis
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		y0, y1 := span(y, h, b.Dy())
		ny := min(y1-y0, most)
		for x := range w {
			x0, x1 := span(x, w, b.Dx())
			nx := min(x1-x0, most)
			// Sum premultiplied 16-bit components, taking the middle input
			// pixel of each of nx×ny equal parts of the span.
			var sum [4]uint32
			for j := range ny {
				sy := b.Min.Y + y0 + mulDiv(2*j+1, y1-y0, 2*ny)
				for i := range nx {
					sx := b.Min.X + x0 + mulDiv(2*i+1, x1-x0, 2*nx)
					r, g, bl, a := img.At(sx, sy).RGBA()
					sum[0] += r
					sum[1] += g
					sum[2] += bl
					sum[3] += a
				}
			}
			n := uint32(nx * ny)
			px := dst.Pix[y*dst.Stride+x*4:]
			for i, s := range sum {
				avg := (s + n/2) / n // rounded, still 16 bits
				px[i] = uint8(avg >> 8)
			}
		}
	}
	return dst
}

// span returns the input range [lo, hi) covered by output index i when n
// outputs map onto m inputs. The range is never empty.
func span(i, n, m int) (lo, hi int) {
	lo = mulDiv(i, m, n)
	return lo, max(mulDiv(i+1, m, n), lo+1)
}

// mulDiv returns a*b/c rounded down, for a, b ≥ 0 and c > 0 with a result
// that fits in an int. The product is computed in 128 bits, so that it cannot
// overflow, however large an image is.
func mulDiv(a, b, c int) int {
	hi, lo := bits.Mul64(uint64(a), uint64(b))
	q, _ := bits.Div64(hi, lo, uint64(c))
	return int(q)
}
