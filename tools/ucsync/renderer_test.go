package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"ucprint/internal/testfixture"
)

func rendererFixture(t *testing.T, extra string, missingFont bool) []byte {
	t.Helper()
	files := map[string][]byte{
		"public/resources/undercards.css":            []byte(".cardBoard {color:white}"),
		"public/resources/DTM-Mono.otf":              []byte("synthetic font fixture for ZIP acquisition, not font validation"),
		"public/resources/modules/card.js":           []byte("export default {};"),
		"public/rarity/TOKEN.png":                    testfixture.PNG(),
		"public/resources/tribes/SNAIL.png":          testfixture.PNG(),
		"public/resources/images/cards/unneeded.png": testfixture.PNG(),
	}
	if missingFont {
		delete(files, "public/resources/DTM-Mono.otf")
	}
	if extra != "" {
		files[extra] = []byte("unexpected")
	}
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for name, data := range files {
		f, e := z.Create("CardEditor-master/" + name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = f.Write(data); e != nil {
			t.Fatal(e)
		}
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}

func TestRendererArchiveAcquisition(t *testing.T) {
	raw := rendererFixture(t, "", false)
	out := filepath.Join(t.TempDir(), "sources")
	if e := collectRendererArchive(raw, out); e != nil {
		t.Fatal(e)
	}
	saved, e := os.ReadFile(filepath.Join(out, "source.zip"))
	if e != nil || !bytes.Equal(saved, raw) {
		t.Fatal("original archive not preserved")
	}
	for _, name := range []string{"public/resources/DTM-Mono.otf", "public/resources/undercards.css", "public/rarity/TOKEN.png", "public/resources/tribes/SNAIL.png", "source-manifest.json"} {
		if !exists(filepath.Join(out, name)) {
			t.Fatal("missing", name)
		}
	}
	if exists(filepath.Join(out, "public/resources/images/cards/unneeded.png")) {
		t.Fatal("unneeded game artwork extracted")
	}
	if e = collectRendererArchive(raw, out); e == nil {
		t.Fatal("overwrote existing bundle")
	}
	after, _ := os.ReadFile(filepath.Join(out, "source.zip"))
	if !bytes.Equal(saved, after) {
		t.Fatal("existing archive changed")
	}
}

func TestRendererRejectsInvalidArchives(t *testing.T) {
	for name, raw := range map[string][]byte{
		"HTML error":     []byte("<html>403</html>"),
		"missing font":   rendererFixture(t, "", true),
		"path traversal": rendererFixture(t, "../escape.txt", false),
		"corrupt icon":   rendererFixture(t, "public/rarity/invalid.png", false),
		"corrupt tribe":  rendererFixture(t, "public/resources/tribes/invalid.png", false),
	} {
		t.Run(name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "sources")
			if e := collectRendererArchive(raw, out); e == nil {
				t.Fatal("accepted invalid archive")
			}
			if exists(out) {
				t.Fatal("installed failed acquisition")
			}
		})
	}
}
