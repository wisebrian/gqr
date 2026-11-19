package gqr

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
)

// WriteSVG writes the code to w as an SVG document. Each module is one user
// unit, so the drawing scales without loss; Options.Scale sets its default
// size in pixels. A logo is embedded as a PNG image, scaled down to 1024
// pixels on a side if it is larger.
func (c *Code) WriteSVG(w io.Writer, o *Options) error {
	r, err := newRenderer(c, o)
	if err != nil {
		return err
	}
	side := r.size + 2*QuietZone
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d" viewBox="0 0 %d %d" shape-rendering="crispEdges">
`, side*r.scale, side*r.scale, side, side)
	if !isTransparent(r.bg) {
		fmt.Fprintf(bw, "<rect width=\"%d\" height=\"%d\" %s/>\n", side, side, svgFill(r.bg))
	}

	// All dark modules form one path, with one rectangle per horizontal run.
	fmt.Fprintf(bw, `<path %s d="`, svgFill(r.fg))
	for y := range r.size {
		for x := 0; x < r.size; x++ {
			if !r.dark(x, y) {
				continue
			}
			start := x
			for x < r.size && r.dark(x, y) {
				x++
			}
			fmt.Fprintf(bw, "M%d %dh%dv1h-%dz", start+QuietZone, y+QuietZone, x-start, x-start)
		}
	}
	bw.WriteString("\"/>\n")

	// A logo without pixels still clears its modules but, as in Image, draws
	// nothing.
	if r.logo != nil && !r.logo.Bounds().Empty() {
		// Leave half a module of background around the logo, as Image does.
		b := r.hole.Add(image.Pt(QuietZone, QuietZone))
		fmt.Fprintf(bw, `<image x="%g" y="%g" width="%d" height="%d" xlink:href="data:image/png;base64,`,
			float64(b.Min.X)+0.5, float64(b.Min.Y)+0.5, b.Dx()-1, b.Dy()-1)
		enc := base64.NewEncoder(base64.StdEncoding, bw)
		if err := png.Encode(enc, svgLogo(r.logo)); err != nil {
			return err
		}
		enc.Close()
		bw.WriteString("\"/>\n")
	}

	bw.WriteString("</svg>\n")
	return bw.Flush() // reports the first write error, if any
}

// maxSVGLogo is the most pixels on a side of a logo embedded in an SVG.
const maxSVGLogo = 1024

// svgLogo returns the logo to embed in an SVG: the logo itself, or a copy
// scaled down to at most maxSVGLogo pixels on a side, so that a large logo
// cannot make the file, or the memory needed to write it, grow without
// bound.
func svgLogo(logo image.Image) image.Image {
	if b := logo.Bounds(); b.Dx() <= maxSVGLogo && b.Dy() <= maxSVGLogo {
		return logo
	}
	small, _ := fit(logo, image.Rect(0, 0, maxSVGLogo, maxSVGLogo))
	return small
}

// svgFill returns the attributes that fill a shape with c.
func svgFill(c color.Color) string {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	s := fmt.Sprintf(`fill="#%02x%02x%02x"`, n.R, n.G, n.B)
	if n.A != 0xff {
		s += fmt.Sprintf(` fill-opacity="%.3g"`, float64(n.A)/0xff)
	}
	return s
}

func isTransparent(c color.Color) bool {
	_, _, _, a := c.RGBA()
	return a == 0
}
