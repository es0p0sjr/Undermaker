package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"ucprint/internal/snapshot"
)

// This explicit sync helper runs in the user's browser, never in UC Print.
//
//go:embed browser-assets.js
var browserTemplate string

//go:embed browser-capture.js
var captureTemplate string

func browserCapture(output string) error {
	const header = "// UC Print loaded-game snapshot capture."
	if b, err := os.ReadFile(output); err == nil && !strings.HasPrefix(string(b), header) {
		return fmt.Errorf("refusing to overwrite an unrelated file: %s", output)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	code := header + "\n" + strings.Replace(browserTemplate, "ucBrowserSync(/*UC_CONFIG*/null);", "", 1) + "\n" + captureTemplate
	if err := os.WriteFile(output, []byte(code), 0644); err != nil {
		return err
	}
	fmt.Println("Prepared loaded-game capture helper:", output)
	fmt.Println("Run as a snippet on the loaded Undercards collection. Choose JSON/update. No AllCards request is made.")
	return nil
}

func browserAssets(o options, output string) error {
	raw, err := os.ReadFile(o.Import)
	if err != nil {
		return err
	}
	records, tr, err := snapshot.RawCards(raw)
	if err != nil {
		return fmt.Errorf("AllCards: %w", err)
	}
	if o.Translations != "" {
		b, e := os.ReadFile(o.Translations)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(b, &tr); e != nil {
			return e
		}
	}
	cards, err := snapshot.Normalize(records, tr)
	if err != nil {
		return err
	}
	paths, err := snapshot.AssetPaths(cards)
	if err != nil {
		return err
	}
	assets := []string{}
	for p := range paths {
		assets = append(assets, strings.TrimPrefix(p, "assets/"))
	}
	sort.Strings(assets)
	config, err := json.Marshal(map[string]any{"assets": assets, "cardCount": len(cards), "rawSHA256": snapshot.Hash(raw)})
	if err != nil {
		return err
	}
	for _, input := range []string{o.Import, o.Translations} {
		if input == "" {
			continue
		}
		a, _ := filepath.Abs(input)
		b, _ := filepath.Abs(output)
		if strings.EqualFold(a, b) {
			return fmt.Errorf("helper output must not overwrite an input JSON file")
		}
	}
	if b, e := os.ReadFile(output); e == nil && !strings.HasPrefix(string(b), "// UC Print browser artwork sync") {
		return fmt.Errorf("refusing to overwrite an unrelated file: %s", output)
	}
	if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	code := strings.Replace(browserTemplate, "/*UC_CONFIG*/null", string(config), 1)
	if err = os.WriteFile(output, []byte(code), 0644); err != nil {
		return err
	}
	fmt.Printf("Prepared %s: %d required images for %d cards. No downloads were made.\n", output, len(assets), len(cards))
	fmt.Println("Run the helper as a DevTools Snippet on https://undercards.net, click Choose assets folder, then select JSON/assets. See tools/ucsync/BROWSER.md.")
	return nil
}
