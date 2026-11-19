package gqr

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"testing"
)

func TestImage(t *testing.T) {
	c := mustEncode(t, "gqr", Medium)
	img, err := c.Image(nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := img.(*image.Paletted); !ok {
		t.Errorf("Image returned %T, want *image.Paletted", img)
	}
	side := (c.Size() + 2*QuietZone) * 8
	if want := image.Rect(0, 0, side, side); img.Bounds() != want {
		t.Fatalf("bounds = %v, want %v", img.Bounds(), want)
	}
	assertColor(t, img, 0, 0, color.White)                             // quiet zone
	assertColor(t, img, QuietZone*8, QuietZone*8, color.Black)         // finder, outer ring
	assertColor(t, img, (QuietZone+1)*8, (QuietZone+1)*8, color.White) // finder, inner ring
}

func TestColors(t *testing.T) {
	c := mustEncode(t, "gqr", Medium)
	red := color.NRGBA{R: 0xff, A: 0xff}
	var buf bytes.Buffer
	if err := c.WritePNG(&buf, &Options{Scale: 2, Foreground: red, Background: color.Transparent}); err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	assertColor(t, img, 0, 0, color.Transparent)
	assertColor(t, img, QuietZone*2, QuietZone*2, red)
}

func TestScaleTooLarge(t *testing.T) {
	c := mustEncode(t, "gqr", Medium)
	most := maxImageSide / (c.Size() + 2*QuietZone)
	if _, err := newRenderer(c, &Options{Scale: most}); err != nil {
		t.Errorf("Scale %d: %v", most, err)
	}
	for _, scale := range []int{most + 1, math.MaxInt} {
		if _, err := c.Image(&Options{Scale: scale}); !errors.Is(err, ErrScaleTooLarge) {
			t.Errorf("Image with Scale %d: err = %v, want ErrScaleTooLarge", scale, err)
		}
		if err := c.WriteSVG(io.Discard, &Options{Scale: scale}); !errors.Is(err, ErrScaleTooLarge) {
			t.Errorf("WriteSVG with Scale %d: err = %v, want ErrScaleTooLarge", scale, err)
		}
	}
}

func assertColor(t *testing.T, img image.Image, x, y int, want color.Color) {
	t.Helper()
	if got := img.At(x, y); !sameColor(got, want) {
		t.Errorf("pixel (%d,%d) = %v, want %v", x, y, got, want)
	}
}

func sameColor(a, b color.Color) bool {
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}
