package app

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"ucprint/internal/printing"
	"ucprint/internal/snapshot"
)

func reworkFixture(t *testing.T) (*Server, fstest.MapFS, []byte) {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 164, 228))); err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{
		"reworks.json":       {Data: []byte(`[{"card":{"id":1,"name":"Custom rules"},"reference":"Complete Front.png"}]`)},
		"Complete Front.png": {Data: b.Bytes()},
	}
	s := New(&snapshot.Dataset{Cards: []snapshot.Card{{ID: 1, Name: "Official"}}}, fstest.MapFS{})
	s.Reworks = files
	return s, files, b.Bytes()
}

func TestReworksImportAndMixedExport(t *testing.T) {
	s, _, original := reworkFixture(t)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/api/reworked-cards", nil))
	var imported struct {
		Cards []ReworkedCard `json:"cards"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &imported) != nil || len(imported.Cards) != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
	c := imported.Cards[0]
	if c.ID != -1 || c.OriginalID != 1 || c.ImageWidth != 164 || c.ImageHeight != 228 || s.Data.Cards[0].Name != "Official" {
		t.Fatalf("wrong imported card or changed snapshot: %+v", c)
	}
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost"+c.CompleteImage, nil))
	if w.Code != 200 || !bytes.Equal(original, w.Body.Bytes()) {
		t.Fatal("complete PNG was not preserved")
	}
	var rgb bytes.Buffer
	zw := zlib.NewWriter(&rgb)
	zw.Write(make([]byte, printing.Width*printing.Height*3))
	zw.Close()
	packed := base64.StdEncoding.EncodeToString(rgb.Bytes())
	body, _ := json.Marshal(map[string]any{"cards": []printing.Entry{{ID: 1, Quantity: 1, RGB: packed}, {ID: -1, Quantity: 9, RGB: packed}}})
	r := httptest.NewRequest("POST", "http://localhost/api/export", bytes.NewReader(body))
	r.Header.Set("X-UCPrint", "1")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"pages":2`) || !strings.Contains(w.Body.String(), `"cards":10`) {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct {
		URL string `json:"url"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost"+result.URL, nil))
	if !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-1.4")) {
		t.Fatal("missing PDF")
	}
}

func TestReworksRejectInvalidImportAndKeepPreviousImages(t *testing.T) {
	for _, data := range []string{
		`[]`, `not json`,
		`[{"card":{"id":1,"name":"Test"},"reference":"../outside.png"}]`,
		`[{"card":{"id":1,"name":"Test"},"reference":"C:\\outside.png"}]`,
		`[{"card":{"id":1,"name":"Test"},"reference":"missing.png"}]`,
		`[{"card":{"id":-1,"name":"Test"},"reference":"Complete Front.png"}]`,
		`[{"card":{"id":1,"name":"A"},"reference":"Complete Front.png"},{"card":{"id":1,"name":"B"},"reference":"Complete Front.png"}]`,
	} {
		t.Run(data, func(t *testing.T) {
			s, files, original := reworkFixture(t)
			if _, err := s.loadReworks(); err != nil {
				t.Fatal(err)
			}
			files["reworks.json"].Data = []byte(data)
			if _, err := s.loadReworks(); err == nil {
				t.Fatal("invalid import accepted")
			}
			if !bytes.Equal(s.reworkImages[-1], original) {
				t.Fatal("failed import discarded previous image")
			}
		})
	}
	s, files, _ := reworkFixture(t)
	files["Complete Front.png"].Data = []byte("not a PNG")
	if _, err := s.loadReworks(); err == nil {
		t.Fatal("corrupt PNG accepted")
	}
	s.Reworks = nil
	if _, err := s.loadReworks(); err == nil {
		t.Fatal("missing folder accepted")
	}
}

func TestProjectReworks(t *testing.T) {
	if _, err := os.Stat("../../ReworkedCards/reworks.json"); os.IsNotExist(err) {
		t.Skip("project reworks absent")
	}
	s := New(&snapshot.Dataset{}, fstest.MapFS{})
	s.Reworks = os.DirFS("../../ReworkedCards")
	cards, err := s.loadReworks()
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) == 0 {
		t.Fatal("no project reworks")
	}
	t.Logf("Validated %d complete reworked PNGs", len(cards))
}
