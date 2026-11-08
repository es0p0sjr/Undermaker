package printing

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const Width = 744
const Height = 1039
const MM = 72.0 / 25.4

type Entry struct {
	ID       int    `json:"id"`
	Quantity int    `json:"quantity"`
	RGB      string `json:"rgb"`
}
type pdf struct {
	objects [][]byte
	pages   []int
}

func newPDF() *pdf              { return &pdf{objects: [][]byte{nil, nil}} }
func (p *pdf) add(b []byte) int { p.objects = append(p.objects, b); return len(p.objects) }
func (p *pdf) stream(b []byte, fields string) int {
	return p.add(append(append([]byte(fmt.Sprintf("<< /Length %d %s >>\nstream\n", len(b), fields)), b...), []byte("\nendstream")...))
}
func (p *pdf) page(content, resources string) {
	ref := p.stream([]byte(content), "")
	id := p.add([]byte(fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %.6f %.6f] /Resources << %s >> /Contents %d 0 R >>", 210*MM, 297*MM, resources, ref)))
	p.pages = append(p.pages, id)
}
func (p *pdf) finish() []byte {
	p.objects[0] = []byte("<< /Type /Catalog /Pages 2 0 R /ViewerPreferences << /PrintScaling /None >> >>")
	kids := []string{}
	for _, id := range p.pages {
		kids = append(kids, fmt.Sprintf("%d 0 R", id))
	}
	p.objects[1] = []byte(fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", len(kids), strings.Join(kids, " ")))
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	offsets := []int{0}
	for i, b := range p.objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n", i+1)
		out.Write(b)
		out.WriteString("\nendobj\n")
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, n := range offsets[1:] {
		fmt.Fprintf(&out, "%010d 00000 n \n", n)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xref)
	return out.Bytes()
}
func position(slot int) (float64, float64) {
	return 7.5 + float64(slot%3)*66, 195.5 - float64(slot/3)*91
}
func cutMarks(x, y float64) string {
	var b strings.Builder
	b.WriteString("0 G 0.3 w\n")
	for _, dx := range []float64{-1, 1} {
		xx := x
		if dx > 0 {
			xx += 63
		}
		for _, dy := range []float64{-1, 1} {
			yy := y
			if dy > 0 {
				yy += 88
			}
			fmt.Fprintf(&b, "%.5f %.5f m %.5f %.5f l S\n%.5f %.5f m %.5f %.5f l S\n", (xx+dx*.35)*MM, yy*MM, (xx+dx*1.3)*MM, yy*MM, xx*MM, (yy+dy*.35)*MM, xx*MM, (yy+dy*1.3)*MM)
		}
	}
	return b.String()
}
func validateRGB(s string) ([]byte, error) {
	b, e := base64.StdEncoding.DecodeString(s)
	if e != nil {
		return nil, e
	}
	reader := bytes.NewReader(b)
	r, e := zlib.NewReader(reader)
	if e != nil {
		return nil, e
	}
	raw, e := io.ReadAll(io.LimitReader(r, Width*Height*3+1))
	r.Close()
	if e != nil || len(raw) != Width*Height*3 || reader.Len() != 0 {
		return nil, fmt.Errorf("invalid RGB payload; expected %dx%d", Width, Height)
	}
	var result bytes.Buffer
	w := zlib.NewWriter(&result)
	w.Write(raw)
	w.Close()
	return result.Bytes(), nil
}
func Export(entries []Entry, cuts bool) ([]byte, error) {
	p := newPDF()
	order := []int{}
	seen := map[int]bool{}
	for _, entry := range entries {
		if seen[entry.ID] || entry.Quantity < 1 || len(order)+entry.Quantity > 250 {
			return nil, fmt.Errorf("invalid/duplicate quantity or more than 250 cards")
		}
		seen[entry.ID] = true
		b, e := validateRGB(entry.RGB)
		if e != nil {
			return nil, e
		}
		ref := p.stream(b, fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode /Interpolate false", Width, Height))
		for n := 0; n < entry.Quantity; n++ {
			order = append(order, ref)
		}
	}
	if len(order) == 0 {
		return nil, fmt.Errorf("print queue is empty")
	}
	for start := 0; start < len(order); start += 9 {
		var content strings.Builder
		resources := []string{}
		for slot := 0; slot < 9 && start+slot < len(order); slot++ {
			x, y := position(slot)
			fmt.Fprintf(&content, "q %.6f 0 0 %.6f %.6f %.6f cm /C%d Do Q\n", 63*MM, 88*MM, x*MM, y*MM, slot)
			if cuts {
				content.WriteString(cutMarks(x, y))
			}
			resources = append(resources, fmt.Sprintf("/C%d %d 0 R", slot, order[start+slot]))
		}
		p.page(content.String(), "/XObject << "+strings.Join(resources, " ")+" >>")
	}
	return p.finish(), nil
}
func Calibration() []byte {
	p := newPDF()
	font := p.add([]byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"))
	var b strings.Builder
	text := func(x, y, size float64, s string) {
		fmt.Fprintf(&b, "BT /F1 %.0f Tf %.5f %.5f Td (%s) Tj ET\n", size, x*MM, y*MM, s)
	}
	b.WriteString("0 G 0.5 w\n")
	text(20, 276, 20, "UC Print - size calibration")
	text(20, 265, 11, "Print A4 at Actual Size / 100%. Disable Fit to page.")
	fmt.Fprintf(&b, "%.5f %.5f %.5f %.5f re S\n", 20*MM, 160*MM, 63*MM, 88*MM)
	text(25, 202, 12, "63 x 88 mm")
	fmt.Fprintf(&b, "%.5f %.5f %.5f %.5f re S\n", 110*MM, 198*MM, 50*MM, 50*MM)
	text(115, 220, 12, "50 x 50 mm")
	text(20, 146, 11, "Measure between centres of the thin outline strokes.")
	p.page(b.String(), fmt.Sprintf("/Font << /F1 %d 0 R >>", font))
	return p.finish()
}
