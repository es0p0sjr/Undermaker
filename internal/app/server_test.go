package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"ucprint/internal/snapshot"
	"ucprint/internal/testfixture"
)

type denyNetwork struct{ t *testing.T }

func TestVariantArtworkEndpoint(t *testing.T) {
	d := &snapshot.Dataset{Cards: []snapshot.Card{{ID: 1, Image: "base.png", ArtVariants: []snapshot.ArtVariant{{ID: "alt", Image: "alt.png", Name: "Alternate", Layout: "alternate"}}}}, Files: map[string][]byte{"assets/cards/base.png": []byte("base"), "assets/cards/alt.png": []byte("alternate")}}
	s := New(d, fstest.MapFS{})
	for _, tc := range []struct {
		url, body string
		status    int
	}{
		{"/api/art/1", "base", 200}, {"/api/art/1?variant=alt", "alternate", 200}, {"/api/art/1?variant=missing", "", 404}, {"/api/art/1?variant=../alt.png", "", 404},
	} {
		w := httptest.NewRecorder()
		s.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost:8765"+tc.url, nil))
		if w.Code != tc.status || (tc.body != "" && w.Body.String() != tc.body) {
			t.Fatalf("%s: %d %s", tc.url, w.Code, w.Body.String())
		}
	}
}

func (d denyNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	d.t.Fatal("runtime attempted an outbound request")
	return nil, nil
}
func TestOfflineRuntime(t *testing.T) {
	root := t.TempDir()
	testfixture.Snapshot(root)
	data, e := snapshot.Load(os.DirFS(root), true)
	if e != nil {
		t.Fatal(e)
	}
	old := http.DefaultTransport
	http.DefaultTransport = denyNetwork{t}
	defer func() { http.DefaultTransport = old }()
	s := New(data, fstest.MapFS{"index.html": {Data: []byte("UC Print")}})
	for _, p := range []string{"/", "/api/cards", "/api/art/1", "/api/power/Taunt.png", "/api/calibration"} {
		r := httptest.NewRequest("GET", "http://127.0.0.1:8765"+p, nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(p, w.Code, w.Body.String())
		}
	}
	// Even deleting the source files after load cannot trigger a lazy download.
	os.Remove(filepath.Join(root, "assets/cards/test.png"))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost:8765/api/art/1", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	for _, p := range []string{"/api/refresh", "/api/import", "/api/import-art"} {
		r := httptest.NewRequest("POST", "http://localhost:8765"+p, strings.NewReader("{}"))
		r.Header.Set("X-UCPrint", "1")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 404 {
			t.Fatal(p, w.Code)
		}
	}
	delete(data.Files, "assets/cards/test.png")
	w = httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost:8765/api/art/1", nil))
	if w.Code != 500 {
		t.Fatal("missing local image did not fail")
	}
}
func TestDeckUsesLocalIDsAndTokenCount(t *testing.T) {
	root := t.TempDir()
	testfixture.Snapshot(root)
	data, _ := snapshot.Load(os.DirFS(root), true)
	ids := []int{}
	for i := 0; i < 25; i++ {
		ids = append(ids, 1)
	}
	ids = append(ids, 2, 2, 2, 2)
	b, _ := json.Marshal(map[string]any{"soul": "JUSTICE", "cardIds": ids, "artifactIds": []int{5}})
	d, e := DecodeDeck(base64.StdEncoding.EncodeToString(b), data.Cards)
	if e != nil {
		t.Fatal(e)
	}
	if d.Queue[0].Quantity != 25 || d.Queue[1].Quantity != 4 || len(d.Warnings) != 1 {
		t.Fatalf("wrong deck %+v", d)
	}
	for _, code := range []string{"bad!", `{"cardIds":[999]}`, `{"cardIds":[true]}`, `{"cardIds":[]}`} {
		if _, e := DecodeDeck(code, data.Cards); e == nil {
			t.Fatal(code)
		}
	}
}
func TestFrontendContainsNoNetworkDependencies(t *testing.T) {
	root := "../../web"
	fs.WalkDir(os.DirFS(root), ".", func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			t.Fatal(e)
		}
		if !d.IsDir() {
			b, _ := os.ReadFile(filepath.Join(root, p))
			if bytes.Contains(b, []byte("undercards.net")) || bytes.Contains(b, []byte("/api/refresh")) {
				t.Fatal("runtime contains download reference", p)
			}
		}
		return nil
	})
}
