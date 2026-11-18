// Gqr encodes text, Wi-Fi credentials or a contact card as a QR code.
//
// Usage:
//
//	gqr [flags] [text]
//	gqr wifi -ssid name [-password p] [-security WPA|WEP|nopass] [-hidden] [flags]
//	gqr contact [-first f] [-last l] [-org o] [-title t] [-phone p] [-email e] [-url u] [flags]
//
// The text is read from standard input when omitted. Without -o the code is
// printed to the terminal; with -o it is written as PNG, or as SVG if the
// file name ends in ".svg". Flags must come before the text; gqr -h lists
// them all.
package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/gif" // decoders for -logo
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wisebrian/gqr"
	"github.com/wisebrian/gqr/payload"
)

const usage = `usage:
  gqr [flags] [text]
  gqr wifi -ssid name [-password p] [-security WPA|WEP|nopass] [-hidden] [flags]
  gqr contact [-first f] [-last l] [-org o] [-title t] [-phone p] [-email e] [-url u] [flags]

The text is read from standard input when omitted. Without -o the code is
printed to the terminal. Flags must come before the text.

flags:
`

func main() {
	err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	switch {
	case err == nil:
	case errors.Is(err, errUsage):
		os.Exit(2) // the error and the usage have been printed
	default:
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// errUsage is returned when the flags do not parse, after the error and the
// usage have been printed.
var errUsage = errors.New("gqr: invalid usage")

// fileFlags only affect the image written with -o.
var fileFlags = map[string]bool{"s": true, "fg": true, "bg": true, "logo": true, "logo-size": true}

const (
	// maxInput is the most bytes read from standard input: far more than
	// any QR code holds, but a limit, so that an input that never ends
	// cannot fill the memory.
	maxInput = 1 << 16

	// maxLogoSide is the most pixels a -logo image may have on a side:
	// more than any logo needs, and few enough to decode in moderate
	// memory. Decoding takes up to 8 bytes per pixel.
	maxLogoSide = 4096
)

// run executes gqr with args, which exclude the program name. The usage
// and flag errors go to stderr.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	mode := "text"
	if len(args) > 0 && (args[0] == "wifi" || args[0] == "contact") {
		mode, args = args[0], args[1:]
	}

	fs := flag.NewFlagSet("gqr", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), usage)
		fs.PrintDefaults()
	}
	var (
		level  = gqr.Medium
		opts   gqr.Options
		out    = fs.String("o", "", "write to `file`: SVG if it ends in .svg, PNG otherwise")
		logo   = fs.String("logo", "", "PNG, JPEG or GIF `file` to draw in the center (needs -o)")
		invert = fs.Bool("invert", false, "print for terminals with a light background (without -o)")
		eci    = fs.Bool("eci", false, "declare text outside ASCII as UTF-8 (ECI); a few older readers do not support it")
	)
	fs.Func("l", "minimum error-correction `level`: L, M, Q or H (default M)", func(s string) (err error) {
		level, err = gqr.ParseLevel(s)
		return err
	})
	fs.IntVar(&opts.Scale, "s", 8, "module size in `pixels`")
	fs.Func("fg", "`color` of dark modules: #rrggbb or #rrggbbaa (default #000000)", func(s string) (err error) {
		opts.Foreground, err = parseColor(s)
		return err
	})
	fs.Func("bg", "`color` of light modules: #rrggbb, #rrggbbaa or transparent (default #ffffff)", func(s string) (err error) {
		opts.Background, err = parseColor(s)
		return err
	})
	fs.Float64Var(&opts.LogoSize, "logo-size", 0.2, "logo width as a `fraction` of the code")

	var (
		wifi    payload.WiFi
		contact payload.Contact
	)
	switch mode {
	case "wifi":
		fs.StringVar(&wifi.SSID, "ssid", "", "network `name` (required)")
		fs.StringVar(&wifi.Password, "password", "", "network password")
		fs.StringVar(&wifi.Security, "security", "", "WPA, WEP or nopass (default WPA, or nopass without a password)")
		fs.BoolVar(&wifi.Hidden, "hidden", false, "the network does not broadcast its name")
	case "contact":
		fs.StringVar(&contact.FirstName, "first", "", "first name")
		fs.StringVar(&contact.LastName, "last", "", "last name")
		fs.StringVar(&contact.Organization, "org", "", "organization")
		fs.StringVar(&contact.Title, "title", "", "job title")
		fs.StringVar(&contact.Phone, "phone", "", "phone number")
		fs.StringVar(&contact.Email, "email", "", "email address")
		fs.StringVar(&contact.URL, "url", "", "website")
	}
	switch err := fs.Parse(args); {
	case errors.Is(err, flag.ErrHelp):
		return nil
	case err != nil:
		return errUsage
	}

	// A flag that would change nothing is a mistake worth reporting.
	var unused error
	fs.Visit(func(f *flag.Flag) {
		switch {
		case unused != nil:
		case fileFlags[f.Name] && *out == "":
			unused = fmt.Errorf("gqr: -%s requires -o", f.Name)
		case f.Name == "logo-size" && *logo == "":
			unused = errors.New("gqr: -logo-size requires -logo")
		case f.Name == "invert" && *out != "":
			unused = errors.New("gqr: -invert only applies to terminal output, without -o")
		}
	})
	if unused != nil {
		return unused
	}
	if opts.Scale < 1 {
		return errors.New("gqr: -s must be at least 1")
	}

	var text string
	switch {
	case mode != "text" && fs.NArg() > 0:
		return fmt.Errorf("gqr %s: unexpected argument %q", mode, fs.Arg(0))
	case mode == "wifi":
		if wifi.SSID == "" {
			return errors.New("gqr wifi: -ssid is required")
		}
		switch strings.ToLower(wifi.Security) {
		case "", "wpa", "wep", "nopass":
		default:
			return fmt.Errorf("gqr wifi: -security must be WPA, WEP or nopass, not %q", wifi.Security)
		}
		text = wifi.String()
	case mode == "contact":
		if contact == (payload.Contact{}) {
			return errors.New("gqr contact: no fields given")
		}
		text = contact.String()
	case fs.NArg() > 0:
		text = strings.Join(fs.Args(), " ")
	default:
		b, err := io.ReadAll(io.LimitReader(stdin, maxInput+1))
		if err != nil {
			return fmt.Errorf("gqr: reading standard input: %w", err)
		}
		if len(b) > maxInput {
			return fmt.Errorf("%w: more than %d bytes on standard input", gqr.ErrTooLong, maxInput)
		}
		text = strings.TrimRight(string(b), "\r\n")
	}

	encode := gqr.Encode
	if *eci {
		encode = gqr.EncodeECI
	}
	code, err := encode(text, level)
	if err != nil {
		return err
	}
	if *out == "" {
		s := code.String()
		if *invert {
			s = code.StringInverted()
		}
		_, err := io.WriteString(stdout, s)
		return err
	}
	if *logo != "" {
		if opts.Logo, err = readImage(*logo); err != nil {
			return err
		}
	}
	return writeFile(*out, code, &opts)
}

// writeFile writes code to the named file as SVG or PNG, depending on the
// extension. It renders into memory first, so a failure leaves no file
// behind.
func writeFile(name string, code *gqr.Code, opts *gqr.Options) error {
	var buf bytes.Buffer
	var err error
	if strings.EqualFold(filepath.Ext(name), ".svg") {
		err = code.WriteSVG(&buf, opts)
	} else {
		err = code.WritePNG(&buf, opts)
	}
	if errors.Is(err, gqr.ErrLogoTooLarge) {
		return fmt.Errorf("%w; try a smaller -logo-size or -l H", err)
	}
	if err != nil {
		return err
	}
	if err := os.WriteFile(name, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("gqr: %w", err)
	}
	return nil
}

// readImage decodes a PNG, JPEG or GIF file, once its header shows that it
// is small enough. A small file can claim a huge image.
func readImage(name string) (image.Image, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("gqr: %w", err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return nil, fmt.Errorf("gqr: %s: %w", name, err)
	}
	if cfg.Width > maxLogoSide || cfg.Height > maxLogoSide {
		return nil, fmt.Errorf("gqr: %s is %dx%d pixels; a logo may have at most %d on a side",
			name, cfg.Width, cfg.Height, maxLogoSide)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("gqr: %w", err)
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("gqr: %s: %w", name, err)
	}
	return img, nil
}

// parseColor parses "#rrggbb", "#rrggbbaa" or "transparent".
func parseColor(s string) (color.Color, error) {
	if s == "transparent" {
		return color.Transparent, nil
	}
	b, err := hex.DecodeString(strings.TrimPrefix(s, "#"))
	if err != nil || len(b) < 3 || len(b) > 4 {
		return nil, errors.New("want #rrggbb, #rrggbbaa or transparent")
	}
	c := color.NRGBA{R: b[0], G: b[1], B: b[2], A: 0xff}
	if len(b) == 4 {
		c.A = b[3]
	}
	return c, nil
}
