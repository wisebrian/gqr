package gqr

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"
	"testing"
)

func TestLogo(t *testing.T) {
	c := mustEncode(t, "https://pkg.go.dev/image/draw", High)
	blue := color.NRGBA{B: 0xff, A: 0xff}
	logo := image.NewNRGBA(image.Rect(0, 0, 40, 20)) // twice as wide as tall
	draw.Draw(logo, logo.Rect, image.NewUniform(blue), image.Point{}, draw.Src)

	opts := &Options{Scale: 4, Logo: logo, LogoSize: 0.2}
	img, err := c.Image(opts)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := c.Image(&Options{Scale: 4})
	if err != nil {
		t.Fatal(err)
	}
	r, err := newRenderer(c, opts)
	if err != nil {
		t.Fatal(err)
	}
	box := r.pixels(r.hole)

	mid := img.Bounds().Dx() / 2
	assertColor(t, img, mid, mid, blue)              // the logo
	assertColor(t, img, mid, box.Min.Y, color.White) // cleared modules above the wide logo
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !image.Pt(x, y).In(box) && !sameColor(img.At(x, y), plain.At(x, y)) {
				t.Fatalf("pixel (%d,%d) outside the logo area changed", x, y)
			}
		}
	}
}

func TestLogoTooLarge(t *testing.T) {
	c := mustEncode(t, "https://pkg.go.dev/image/draw", High)
	logo := image.NewGray(image.Rect(0, 0, 1, 1))
	for _, size := range []float64{0.9, 1, math.Inf(1), math.NaN()} {
		if _, err := c.Image(&Options{Logo: logo, LogoSize: size}); !errors.Is(err, ErrLogoTooLarge) {
			t.Errorf("LogoSize %v: err = %v, want ErrLogoTooLarge", size, err)
		}
	}
}

func TestCanHide(t *testing.T) {
	// A full version 10-L symbol has little error correction to spare.
	c := mustEncode(t, strings.Repeat("a", 271), Low)
	if c.Version() != 10 || c.Level() != Low {
		t.Fatalf("got %d-%v, want 10-L", c.Version(), c.Level())
	}
	mid, n := c.Size()/2, c.Size()
	tests := []struct {
		name string
		r    image.Rectangle
		want bool
	}{
		{"one module", image.Rect(mid, mid, mid+1, mid+1), true},
		{"whole data area", image.Rect(9, 9, n-9, n-9), false},
		{"format information", image.Rect(8, 8, 9, 9), false},
	}
	for _, tt := range tests {
		if got := c.canHide(tt.r); got != tt.want {
			t.Errorf("%s: canHide(%v) = %v, want %v", tt.name, tt.r, got, tt.want)
		}
	}
}

func TestResize(t *testing.T) {
	// Averaging one black and one white pixel gives mid gray.
	src := image.NewGray(image.Rect(0, 0, 2, 1))
	src.Pix[1] = 0xff
	gray := color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}
	if got := resize(src, 1, 1).RGBAAt(0, 0); got != gray {
		t.Errorf("resize = %v, want %v", got, gray)
	}

	// So does sampling a larger image, black on the left, white on the right.
	src = image.NewGray(image.Rect(0, 0, 1000, 10))
	for y := range 10 {
		for x := 500; x < 1000; x++ {
			src.Pix[y*src.Stride+x] = 0xff
		}
	}
	if got := resize(src, 1, 1).RGBAAt(0, 0); got != gray {
		t.Errorf("resize of a large image = %v, want %v", got, gray)
	}
}

// solid is a one-color image as large as an image can be, far too large to
// hold in memory.
type solid struct{ c color.Color }

func (s solid) ColorModel() color.Model { return color.RGBAModel }
func (s solid) Bounds() image.Rectangle { return image.Rect(0, 0, math.MaxInt, math.MaxInt) }
func (s solid) At(x, y int) color.Color { return s.c }

// A logo of any size works: only a few of its pixels are read.
func TestHugeLogo(t *testing.T) {
	c := mustEncode(t, "https://go.dev", High)
	blue := color.NRGBA{B: 0xff, A: 0xff}
	logo := solid{blue}

	img, err := c.Image(&Options{Scale: 4, Logo: logo})
	if err != nil {
		t.Fatal(err)
	}
	mid := img.Bounds().Dx() / 2
	assertColor(t, img, mid, mid, blue)

	// An SVG embeds a copy of at most maxSVGLogo pixels on a side.
	var buf bytes.Buffer
	if err := c.WriteSVG(&buf, &Options{Logo: logo}); err != nil {
		t.Fatal(err)
	}
	_, data, _ := strings.Cut(buf.String(), "base64,")
	data, _, _ = strings.Cut(data, `"`)
	cfg, err := png.DecodeConfig(base64.NewDecoder(base64.StdEncoding, strings.NewReader(data)))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != maxSVGLogo || cfg.Height != maxSVGLogo {
		t.Errorf("embedded logo is %dx%d, want %dx%d", cfg.Width, cfg.Height, maxSVGLogo, maxSVGLogo)
	}
}

// A logo without pixels is accepted and draws nothing, in an SVG as in an
// image.
func TestEmptyLogo(t *testing.T) {
	c := mustEncode(t, "https://go.dev", High)
	opts := &Options{Logo: image.NewGray(image.Rectangle{})}
	if _, err := c.Image(opts); err != nil {
		t.Errorf("Image: %v", err)
	}
	var buf bytes.Buffer
	if err := c.WriteSVG(&buf, opts); err != nil {
		t.Fatalf("WriteSVG: %v", err)
	}
	if strings.Contains(buf.String(), "<image") {
		t.Error("the SVG embeds the empty logo")
	}
}
