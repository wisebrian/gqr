package main

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wisebrian/gqr"
)

func TestRun(t *testing.T) {
	dir := t.TempDir()
	path := func(name string) string { return filepath.Join(dir, name) }
	logo := path("logo.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logo, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	// A logo one pixel wider than allowed. Its file is small: only the
	// header has to be read to refuse it.
	wide := path("wide.png")
	buf.Reset()
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, maxLogoSide+1, 1))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wide, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		args    []string
		stdin   string
		file    string // file the command must write; empty means the terminal
		wantErr string
	}{
		{name: "text", args: []string{"hello"}},
		{name: "stdin", stdin: "hello\n"},
		{name: "invert", args: []string{"-invert", "hello"}},
		{name: "eci", args: []string{"-eci", "naïve café"}},
		{name: "kanji", args: []string{"日本語"}},
		{name: "png", args: []string{"-o", path("a.png"), "hello"}, file: path("a.png")},
		{name: "svg", args: []string{"-o", path("a.svg"), "-fg", "#1a4d8f", "-bg", "transparent", "hello"}, file: path("a.svg")},
		{name: "logo", args: []string{"-o", path("b.png"), "-l", "H", "-logo", logo, "https://go.dev"}, file: path("b.png")},
		{name: "wifi", args: []string{"wifi", "-ssid", "Home", "-password", "secret"}},
		{name: "contact", args: []string{"contact", "-first", "John", "-phone", "+15555550100"}},
		{name: "empty input", stdin: "\n", wantErr: "no data"},
		{name: "logo too large", args: []string{"-o", path("c.png"), "-logo", logo, "-logo-size", "0.9", "hello"}, wantErr: "logo too large"},
		{name: "logo with too many pixels", args: []string{"-o", path("c.png"), "-l", "H", "-logo", wide, "hello"}, wantErr: "at most 4096 on a side"},
		{name: "too much input", stdin: strings.Repeat("1", maxInput+1), wantErr: "more than 65536 bytes"},
		{name: "logo without -o", args: []string{"-logo", logo, "hello"}, wantErr: "-logo requires -o"},
		{name: "color without -o", args: []string{"-fg", "#1a4d8f", "hello"}, wantErr: "-fg requires -o"},
		{name: "logo size without logo", args: []string{"-o", path("c.png"), "-logo-size", "0.3", "hello"}, wantErr: "-logo-size requires -logo"},
		{name: "invert with -o", args: []string{"-o", path("c.png"), "-invert", "hello"}, wantErr: "-invert only applies to terminal output"},
		{name: "zero scale", args: []string{"-o", path("c.png"), "-s", "0", "hello"}, wantErr: "-s must be at least 1"},
		{name: "huge scale", args: []string{"-o", path("c.png"), "-s", "100000", "hello"}, wantErr: "scale too large"},
		{name: "wifi without ssid", args: []string{"wifi"}, wantErr: "-ssid is required"},
		{name: "open wifi", args: []string{"wifi", "-ssid", "Guest", "-security", "NOPASS"}},
		{name: "wifi with unknown security", args: []string{"wifi", "-ssid", "Cafe", "-security", "WPA;S:Evil"}, wantErr: "-security must be WPA, WEP or nopass"},
		{name: "empty contact", args: []string{"contact"}, wantErr: "no fields"},
		{name: "extra argument", args: []string{"wifi", "-ssid", "x", "extra"}, wantErr: "unexpected argument"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			err := run(tt.args, strings.NewReader(tt.stdin), &stdout, io.Discard)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want one containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.file == "" {
				if !strings.Contains(stdout.String(), "█") {
					t.Error("no QR code printed")
				}
				return
			}
			if fi, err := os.Stat(tt.file); err != nil || fi.Size() == 0 {
				t.Errorf("output file missing or empty: %v", err)
			}
		})
	}

	if _, err := os.Stat(path("c.png")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a failed render left a file behind")
	}
}

// When a logo does not fit, -l H is suggested only if it would change the
// code.
func TestLogoHint(t *testing.T) {
	dir := t.TempDir()
	logo := filepath.Join(dir, "logo.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logo, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		text, want string
	}{
		// The text fills a 1-M code, too small for the logo. At level H it
		// takes version 2, which has room.
		{"https://go.dev", "try a smaller -logo-size or -l H"},
		// A short text is raised to level H already.
		{"hi", "try a smaller -logo-size"},
	}
	for _, tt := range tests {
		args := []string{"-o", filepath.Join(dir, "qr.png"), "-logo", logo, tt.text}
		err := run(args, strings.NewReader(""), io.Discard, io.Discard)
		if err == nil || !strings.HasSuffix(err.Error(), tt.want) {
			t.Errorf("%q: err = %v, want one ending in %q", tt.text, err, tt.want)
		}
	}
}

// endless is standard input that never ends, like /dev/zero. It gives up
// after 16 MiB, which only a run that reads without a limit would reach.
type endless struct{ n int }

func (e *endless) Read(p []byte) (int, error) {
	if e.n > 16<<20 {
		return 0, errors.New("endless: read 16 MiB")
	}
	for i := range p {
		p[i] = '1'
	}
	e.n += len(p)
	return len(p), nil
}

func TestEndlessInput(t *testing.T) {
	in := &endless{}
	if err := run(nil, in, io.Discard, io.Discard); !errors.Is(err, gqr.ErrTooLong) {
		t.Fatalf("err = %v, want ErrTooLong", err)
	}
	if in.n > maxInput+1 {
		t.Errorf("read %d bytes, want at most %d", in.n, maxInput+1)
	}
}

// Flags that do not parse print the error and the usage, as -h prints the
// usage alone.
func TestUsage(t *testing.T) {
	tests := []struct {
		args       []string
		wantErr    error
		wantStderr string
	}{
		{[]string{"-h"}, nil, "usage:"},
		{[]string{"-x", "hello"}, errUsage, "flag provided but not defined: -x"},
		{[]string{"-l", "X", "hello"}, errUsage, `invalid value "X" for flag -l`},
		{[]string{"wifi", "-ssid"}, errUsage, "flag needs an argument: -ssid"},
	}
	for _, tt := range tests {
		var stderr bytes.Buffer
		err := run(tt.args, strings.NewReader(""), io.Discard, &stderr)
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("%q: err = %v, want %v", tt.args, err, tt.wantErr)
		}
		for _, want := range []string{tt.wantStderr, "usage:"} {
			if !strings.Contains(stderr.String(), want) {
				t.Errorf("%q: stderr lacks %q:\n%s", tt.args, want, stderr.String())
			}
		}
	}
}
