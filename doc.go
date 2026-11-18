// Package gqr generates QR codes (ISO/IEC 18004) using only the Go standard
// library.
//
// Encode chooses the smallest symbol that fits and the mask with the lowest
// penalty. It splits the text into numeric, alphanumeric, Kanji and byte
// segments, whichever way takes the fewest bits. The error-correction level
// it is given is a minimum: room to spare in the symbol goes to a higher
// level. EncodeECI also declares text outside ASCII as UTF-8.
//
// A Code is drawn with Image, WritePNG or WriteSVG, or printed to a
// terminal with String. Options set the colors and an optional logo, which
// is accepted only if error correction can still repair every codeword it
// hides.
//
// Package [github.com/wisebrian/gqr/payload] formats Wi-Fi credentials and
// contact cards that phone cameras act on.
//
//	code, err := gqr.Encode("https://go.dev", gqr.Medium)
//	if err != nil {
//		log.Fatal(err)
//	}
//	err = code.WritePNG(w, nil)
package gqr
