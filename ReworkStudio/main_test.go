package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"ucprint/internal/snapshot"
)

func studioFixture(t *testing.T) *Studio {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "ReworkedCards"), 0755); err != nil {
		t.Fatal(err)
	}
	n := 1
	c := snapshot.Card{ID: 222, Name: "Blue Snail", Cost: &n, Attack: &n, Health: &n, Description: "Original", Rarity: "COMMON", Extension: "BASE", Type: "MONSTER", Image: "Blue_Snail.png", Statuses: []snapshot.Status{}}
	s := &Studio{root: root, data: &snapshot.Dataset{Cards: []snapshot.Card{c}, Files: map[string][]byte{}}, reworks: []Rework{{Card: c, Reference: "Blue Snail.png"}}}
	if err := s.save(s.reworks[0]); err != nil {
		t.Fatal(err)
	}
	return s
}
func post(s *Studio, body []byte, origin, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "http://127.0.0.1:8767/api/reworks", bytes.NewReader(body))
	r.Header.Set("Origin", origin)
	r.Header.Set("X-UCPrint", token)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestSavePersistsWithoutChangingOfficial(t *testing.T) {
	s := studioFixture(t)
	r := s.reworks[0]
	r.Card.Description = "Physical rules"
	r.Reviewed = true
	b, _ := json.Marshal(r)
	w := post(s, b, "http://127.0.0.1:8767", "1")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	onDisk, err := os.ReadFile(filepath.Join(s.root, "ReworkedCards", "reworks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved []Rework
	if err = json.Unmarshal(onDisk, &saved); err != nil {
		t.Fatal(err)
	}
	if saved[0].Card.Description != "Physical rules" || !saved[0].Reviewed {
		t.Fatal("edits not persisted")
	}
	if s.data.Cards[0].Description != "Original" {
		t.Fatal("official data mutated")
	}
	// A second save exercises replacing an existing file on Windows.
	r.Card.Description = "Second version"
	b, _ = json.Marshal(r)
	if w = post(s, b, "", "1"); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
}
func TestRejectedSavesPreserveFile(t *testing.T) {
	for _, kind := range []string{"unknown-id", "invalid-stats", "reference-path", "missing-image", "cross-origin", "missing-token", "trailing-json"} {
		t.Run(kind, func(t *testing.T) {
			s := studioFixture(t)
			filename := filepath.Join(s.root, "ReworkedCards", "reworks.json")
			before, _ := os.ReadFile(filename)
			r := s.reworks[0]
			origin, token := "http://127.0.0.1:8767", "1"
			switch kind {
			case "unknown-id":
				r.Card.ID = 999
			case "invalid-stats":
				n := -1
				r.Card.Cost = &n
			case "reference-path":
				r.Reference = "../data/cards.json"
			case "missing-image":
				r.Card.Statuses = []snapshot.Status{{Image: "Missing.png"}}
			case "cross-origin":
				origin = "http://example.com"
			case "missing-token":
				token = ""
			}
			b, _ := json.Marshal(r)
			if kind == "trailing-json" {
				b = append(b, []byte(" {}")...)
			}
			w := post(s, b, origin, token)
			if w.Code < 400 {
				t.Fatal("invalid save accepted")
			}
			after, _ := os.ReadFile(filename)
			if !bytes.Equal(before, after) {
				t.Fatal("failed save changed stored data")
			}
		})
	}
}
