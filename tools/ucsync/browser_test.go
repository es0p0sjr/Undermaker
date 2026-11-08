package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"ucprint/internal/testfixture"
)

func TestBrowserHelperUsesExactReferences(t *testing.T) {
	root := t.TempDir()
	raw, err := testfixture.Input(root)
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "helper.js")
	input := filepath.Join(root, "AllCards.json")
	if err = browserAssets(options{Import: input}, output); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	start := strings.LastIndex(string(b), "ucBrowserSync(") + len("ucBrowserSync(")
	configText := strings.TrimSuffix(strings.TrimSpace(string(b[start:])), ");")
	var config struct {
		Assets    []string `json:"assets"`
		CardCount int      `json:"cardCount"`
	}
	if err = json.Unmarshal([]byte(configText), &config); err != nil {
		t.Fatal(err)
	}
	if config.CardCount != 2 || strings.Join(config.Assets, ",") != "cards/test.png,powers/Taunt.png" {
		t.Fatalf("wrong references: %+v", config)
	}
	after, _ := os.ReadFile(input)
	if string(after) != string(raw) {
		t.Fatal("raw input changed")
	}
	if err = browserAssets(options{Import: input}, input); err == nil {
		t.Fatal("allowed overwriting raw input")
	}
	unrelated := filepath.Join(root, "notes.txt")
	os.WriteFile(unrelated, []byte("keep me"), 0644)
	if err = browserAssets(options{Import: input}, unrelated); err == nil {
		t.Fatal("overwrote unrelated file")
	}
}

func TestLoadedCaptureHelper(t *testing.T) {
	output := filepath.Join(t.TempDir(), "capture.js")
	if err := browserCapture(output); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	code := string(b)
	if strings.Contains(code, "ucBrowserSync(/*UC_CONFIG*/null);") || !strings.Contains(code, "async function ucCollectAssets") || !strings.Contains(code, "ucCaptureLoadedGame();") {
		t.Fatal("capture helper is missing its collector or still launches the old panel")
	}
	if err := os.WriteFile(output, []byte("keep this unrelated file"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := browserCapture(output); err == nil {
		t.Fatal("overwrote an unrelated file")
	}
}
