package gqr

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestReference compares whole symbols, module by module, with those that an
// independent implementation made from the same segments, version, level and
// mask. This checks every step after the choice of segments: the bit
// stream, error correction, interleaving, placement, masking, and the
// format and version information. The file says how the symbols were made.
func TestReference(t *testing.T) {
	for _, ref := range readReferences(t) {
		t.Run(ref.name, func(t *testing.T) {
			c := newCode(ref.segs, ref.version, ref.level, ref.mask)
			if c.Size() != len(ref.rows) {
				t.Fatalf("size %d, want %d", c.Size(), len(ref.rows))
			}
			wrong, first := 0, ""
			for y, row := range ref.rows {
				for x := range len(row) {
					if c.Dark(x, y) != (row[x] == '#') {
						if wrong == 0 {
							first = strconv.Itoa(x) + "," + strconv.Itoa(y)
						}
						wrong++
					}
				}
			}
			if wrong > 0 {
				t.Errorf("%d modules differ, the first at (%s)", wrong, first)
			}
		})
	}
}

type reference struct {
	name    string
	version int
	level   Level
	mask    int
	segs    segments
	rows    []string
}

// readReferences parses testdata/reference.txt.
func readReferences(t *testing.T) []reference {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "reference.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var refs []reference
	for i, line := range strings.Split(string(data), "\n") {
		fail := func(err error) { t.Fatalf("reference.txt:%d: %v", i+1, err) }
		switch {
		case line == "" || strings.HasPrefix(line, "//"):
		case strings.HasPrefix(line, "symbol "):
			// symbol <version>-<level> mask <mask>
			f := strings.Fields(line)
			v, l, _ := strings.Cut(f[1], "-")
			ref := reference{name: f[1]}
			if ref.version, err = strconv.Atoi(v); err != nil {
				fail(err)
			}
			if ref.level, err = ParseLevel(l); err != nil {
				fail(err)
			}
			if ref.mask, err = strconv.Atoi(f[3]); err != nil {
				fail(err)
			}
			refs = append(refs, ref)
		case line[0] == '#' || line[0] == '.':
			ref := &refs[len(refs)-1]
			ref.rows = append(ref.rows, line)
		default:
			// <mode> <quoted content>
			name, quoted, _ := strings.Cut(line, " ")
			text, err := strconv.Unquote(quoted)
			if err != nil {
				fail(err)
			}
			seg := segment{data: []byte(text)}
			switch name {
			case "numeric":
				seg.mode = modeNumeric
			case "alphanumeric":
				seg.mode = modeAlphanumeric
			case "byte":
				seg.mode = modeByte
			case "kanji":
				seg = segment{mode: modeKanji}
				for _, r := range text {
					code, ok := shiftJIS(r)
					if !ok {
						t.Fatalf("reference.txt:%d: %q has no Shift JIS code", i+1, r)
					}
					seg.data = append(seg.data, byte(code>>8), byte(code))
				}
			case "eci":
				n, err := strconv.Atoi(text)
				if err != nil {
					fail(err)
				}
				seg = segment{modeECI, []byte{byte(n)}}
			default:
				t.Fatalf("reference.txt:%d: unknown mode %q", i+1, name)
			}
			ref := &refs[len(refs)-1]
			ref.segs = append(ref.segs, seg)
		}
	}
	if len(refs) == 0 {
		t.Fatal("no reference symbols")
	}
	return refs
}
