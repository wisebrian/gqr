package gqr

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
)

// QuietZone is the width, in modules, of the light margin the standard
// requires around a symbol. Image, WritePNG and WriteSVG include it.
const QuietZone = 4

// maxImageSide is the most pixels an image may have on a side. It keeps
// memory in check: Image needs a byte per pixel, and four more with a logo.
const maxImageSide = 1 << 14

// ErrScaleTooLarge is returned when Options.Scale would make an image more
// than 16384 pixels wide.
var ErrScaleTooLarge = errors.New("gqr: scale too large")

// Options controls how a Code is drawn. A nil *Options draws black modules
// on white, 8 pixels per module.
type Options struct {
	// Scale is the size of a module in pixels. Values below 1 mean 8.
	// Drawing fails with ErrScaleTooLarge if the image, quiet zone
	// included, would be more than 16384 pixels wide.
	Scale int

	// Foreground and Background color the dark and light modules. Nil means
	// black and white; color.Transparent makes the background transparent.
	// Scanners need dark modules that contrast clearly with light ones.
	Foreground, Background color.Color

	// Logo, if set, is drawn in the center over a square of modules whose
	// side is LogoSize times the width of the symbol (0.2 if zero). Error
	// correction must repair the modules it hides, and drawing fails with
	// ErrLogoTooLarge if that would take more than half of what it can
	// repair; the rest is kept for real damage. Level High leaves room for
	// the largest logos. An image of any size works, as only a few of its
	// pixels are read for each one drawn.
	Logo     image.Image
	LogoSize float64
}

// renderer is a Code with its Options resolved.
type renderer struct {
	*Code
	scale  int
	fg, bg color.Color
	logo   image.Image
	hole   image.Rectangle // modules hidden by the logo
}

func newRenderer(c *Code, o *Options) (*renderer, error) {
	r := &renderer{Code: c, scale: 8, fg: color.Black, bg: color.White}
	if o == nil {
		return r, nil
	}
	if o.Scale > 0 {
		r.scale = o.Scale
	}
	if most := maxImageSide / (c.size + 2*QuietZone); r.scale > most {
		return nil, fmt.Errorf("%w: %d pixels per module, at most %d for this code", ErrScaleTooLarge, r.scale, most)
	}
	if o.Foreground != nil {
		r.fg = o.Foreground
	}
	if o.Background != nil {
		r.bg = o.Background
	}
	if o.Logo != nil {
		hole, err := c.logoHole(o.LogoSize)
		if err != nil {
			return nil, err
		}
		r.logo, r.hole = o.Logo, hole
	}
	return r, nil
}

// dark reports whether module (x, y) is drawn dark. Modules hidden by the
// logo are left light.
func (r *renderer) dark(x, y int) bool {
	return r.Dark(x, y) && !image.Pt(x, y).In(r.hole)
}

// pixels converts a rectangle of modules, in symbol coordinates, to the
// pixels that Image draws it on.
func (r *renderer) pixels(m image.Rectangle) image.Rectangle {
	m = m.Add(image.Pt(QuietZone, QuietZone))
	return image.Rectangle{Min: m.Min.Mul(r.scale), Max: m.Max.Mul(r.scale)}
}

// Image draws the code, including the quiet zone. The result is an
// *image.Paletted with two colors, or an *image.RGBA when there is a logo.
func (c *Code) Image(o *Options) (image.Image, error) {
	r, err := newRenderer(c, o)
	if err != nil {
		return nil, err
	}
	side := (r.size + 2*QuietZone) * r.scale
	img := image.NewPaletted(image.Rect(0, 0, side, side), color.Palette{r.bg, r.fg})
	for py := range side {
		y := py/r.scale - QuietZone
		row := img.Pix[py*img.Stride:][:side]
		for px := range row {
			if r.dark(px/r.scale-QuietZone, y) {
				row[px] = 1
			}
		}
	}
	if r.logo == nil {
		return img, nil
	}

	out := image.NewRGBA(img.Rect)
	draw.Draw(out, out.Rect, img, image.Point{}, draw.Src)
	// Leave half a module of background between the logo and the modules.
	logo, at := fit(r.logo, r.pixels(r.hole).Inset(r.scale/2))
	draw.Draw(out, at, logo, image.Point{}, draw.Over)
	return out, nil
}

// WritePNG writes the code to w as a PNG image.
func (c *Code) WritePNG(w io.Writer, o *Options) error {
	img, err := c.Image(o)
	if err != nil {
		return err
	}
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	return enc.Encode(w, img)
}
