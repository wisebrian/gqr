package gqr

import (
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	"io"
	"strings"
	"testing"
)

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
