package printing

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"testing"
)

func rgb() string {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(bytes.Repeat([]byte{240, 250, 240}, Width*Height))
	w.Close()
	return base64.StdEncoding.EncodeToString(b.Bytes())
}
func TestPagination(t *testing.T) {
	for _, n := range []int{1, 9, 10, 25, 54, 250} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			b, e := Export([]Entry{{1, n, rgb()}}, true)
			if e != nil {
				t.Fatal(e)
			}
			if count := len(regexp.MustCompile(`/Type /Page\b`).FindAll(b, -1)); count != (n+8)/9 {
				t.Fatal(count)
			}
			if bytes.Count(b, []byte("/Subtype /Image")) != 1 {
				t.Fatal("duplicate image encoding")
			}
			if !bytes.Contains(b, []byte(fmt.Sprintf("q %.6f 0 0 %.6f", 63*MM, 88*MM))) {
				t.Fatal("wrong physical size")
			}
			match := regexp.MustCompile(`startxref\n(\d+)`).FindSubmatch(b)
			offset, _ := strconv.Atoi(string(match[1]))
			if string(b[offset:offset+4]) != "xref" {
				t.Fatal("bad PDF xref")
			}
		})
	}
}
func TestInvalidInputs(t *testing.T) {
	for _, entries := range [][]Entry{nil, {{1, 1, "bad"}}, {{1, 0, rgb()}}, {{1, 251, rgb()}}} {
		if _, e := Export(entries, true); e == nil {
			t.Fatal("invalid export accepted")
		}
	}
}
func TestCalibrationAndCuts(t *testing.T) {
	b := Calibration()
	if !bytes.Contains(b, []byte("63 x 88 mm")) || !bytes.Contains(b, []byte("50 x 50 mm")) {
		t.Fatal("missing measurement")
	}
	x, y := position(8)
	if x != 139.5 || y != 13.5 {
		t.Fatal(x, y)
	}
	b, _ = Export([]Entry{{1, 1, rgb()}}, false)
	if bytes.Contains(b, []byte("0.3 w")) {
		t.Fatal("cut marks not disabled")
	}
}
