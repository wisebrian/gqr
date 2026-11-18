package gqr

import (
	"bytes"
	"encoding/xml"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"strings"
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

func TestWriteSVG(t *testing.T) {
	c := mustEncode(t, "https://go.dev", High)
	tests := []struct {
		name          string
		opts          *Options
		want, notWant []string
	}{
		{"default", nil, []string{`<rect`, `fill="#000000"`}, []string{`fill-opacity`, `<image`}},
		{
			"colors",
			&Options{Foreground: color.NRGBA{R: 0x1a, G: 0x4d, B: 0x8f, A: 0x80}, Background: color.Transparent},
			[]string{`fill="#1a4d8f" fill-opacity="0.502"`},
			[]string{`<rect`},
		},
		{"logo", &Options{Logo: image.NewGray(image.Rect(0, 0, 4, 4))}, []string{`<image`, `data:image/png;base64,`}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := c.WriteSVG(&buf, tt.opts); err != nil {
				t.Fatal(err)
			}
			svg := buf.String()
			for _, s := range tt.want {
				if !strings.Contains(svg, s) {
					t.Errorf("SVG lacks %s", s)
				}
			}
			for _, s := range tt.notWant {
				if strings.Contains(svg, s) {
					t.Errorf("SVG contains %s", s)
				}
			}
			for d := xml.NewDecoder(strings.NewReader(svg)); ; {
				if _, err := d.Token(); err == io.EOF {
					break
				} else if err != nil {
					t.Fatalf("malformed SVG: %v", err)
				}
			}
		})
	}
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

func TestString(t *testing.T) {
	c := mustEncode(t, "gqr", Medium)
	want := (c.Size() + 2*QuietZone + 1) / 2
	for _, s := range []string{c.String(), c.StringInverted()} {
		if lines := strings.Count(s, "\n"); lines != want {
			t.Errorf("%d lines, want %d", lines, want)
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
