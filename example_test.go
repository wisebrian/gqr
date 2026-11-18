package gqr_test

import (
	"fmt"
	"image/color"
	"image/png"
	"log"
	"os"

	"github.com/wisebrian/gqr"
)

func Example() {
	code, err := gqr.Encode("HELLO WORLD", gqr.Low)
	if err != nil {
		log.Fatal(err)
	}
	// The symbol had room to spare, so the level was raised from L to Q.
	fmt.Println(code.Version(), code.Level(), code.Size())
	// Output: 1 Q 21
}

func ExampleOptions() {
	code, err := gqr.Encode("https://go.dev", gqr.High)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Open("logo.png")
	if err != nil {
		log.Fatal(err)
	}
	logo, err := png.Decode(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.Create("qr.png")
	if err != nil {
		log.Fatal(err)
	}
	err = code.WritePNG(out, &gqr.Options{
		Scale:      10,
		Foreground: color.NRGBA{R: 0x1a, G: 0x4d, B: 0x8f, A: 0xff},
		Background: color.Transparent,
		Logo:       logo,
		LogoSize:   0.25,
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := out.Close(); err != nil {
		log.Fatal(err)
	}
}
