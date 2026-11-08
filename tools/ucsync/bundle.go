package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"ucprint/internal/snapshot"
)

// Package only the display resources, never execute the downloaded editor.
// The generated file is part of the frontend and is embedded by ucbuild.
func rendererBundle(sourceDir, output string) error {
	b, e := os.ReadFile(filepath.Join(sourceDir, "source-manifest.json"))
	if e != nil {
		return fmt.Errorf("renderer source missing; run ucsync renderer-download or renderer-import first: %w", e)
	}
	var manifest struct {
		Checksums     map[string]string
		ArchiveSHA256 string
	}
	if e = json.Unmarshal(b, &manifest); e != nil {
		return e
	}
	read := func(name string) ([]byte, error) {
		b, e := os.ReadFile(filepath.Join(sourceDir, filepath.FromSlash(name)))
		if e != nil {
			return nil, e
		}
		if snapshot.Hash(b) != manifest.Checksums[name] {
			return nil, fmt.Errorf("renderer source checksum mismatch: %s", name)
		}
		return b, nil
	}
	uri := func(kind string, b []byte) string {
		return "data:" + kind + ";base64," + base64.StdEncoding.EncodeToString(b)
	}
	font, e := read("public/resources/DTM-Mono.otf")
	if e != nil {
		return e
	}
	if len(font) < 12 || (string(font[:4]) != "OTTO" && string(font[:4]) != "\x00\x01\x00\x00") {
		return fmt.Errorf("invalid renderer font")
	}
	fontURI := uri("font/otf", font)
	styles := []string{}
	for _, name := range []string{"undercards.css", "description.css", "style.css"} {
		b, e := read("public/resources/" + name)
		if e != nil {
			return e
		}
		css := strings.ReplaceAll(string(b), "./DTM-Mono.otf", fontURI)
		if strings.Contains(css, "@import") || strings.Contains(css, "http:") || strings.Contains(css, "https:") {
			return fmt.Errorf("renderer CSS contains an external dependency: %s", name)
		}
		styles = append(styles, css)
	}
	js, e := read("public/resources/modules/card.js")
	if e != nil {
		return e
	}
	templates := map[string]string{}
	for _, kind := range []string{"MONSTER", "SPELL"} {
		markup, e := sourceCardTemplate(string(js), kind == "MONSTER")
		if e != nil {
			return e
		}
		templates[kind] = markup
	}
	icons := map[string]string{}
	for _, rarity := range []string{"BASE", "COMMON", "RARE", "EPIC", "LEGENDARY", "DETERMINATION", "TOKEN"} {
		for _, suffix := range []string{"", "_DR", "_UTY"} {
			key := rarity + suffix
			b, e := read("public/rarity/" + key + ".png")
			if e != nil {
				return e
			}
			if e = snapshot.CheckImage(b); e != nil {
				return fmt.Errorf("rarity %s: %w", key, e)
			}
			icons[key] = uri("image/png", b)
		}
	}
	tribes := map[string]string{}
	for name := range manifest.Checksums {
		if !strings.HasPrefix(name, "public/resources/tribes/") || !strings.HasSuffix(name, ".png") {
			continue
		}
		key := strings.TrimSuffix(strings.TrimPrefix(name, "public/resources/tribes/"), ".png")
		if strings.ContainsAny(key, "/\\") || key == "" {
			return fmt.Errorf("invalid tribe resource: %s", name)
		}
		b, e := read(name)
		if e != nil {
			return e
		}
		if e = snapshot.CheckImage(b); e != nil {
			return fmt.Errorf("tribe %s: %w", key, e)
		}
		tribes[key] = uri("image/png", b)
	}
	if len(tribes) == 0 {
		return fmt.Errorf("renderer tribe icons missing; re-import the source archive with tribe resources")
	}
	if e = writeJSON(filepath.Dir(output), filepath.Base(output), map[string]any{
		"version": 1, "source": "UCProjects/CardEditor", "archiveSHA256": manifest.ArchiveSHA256,
		"font": fontURI, "styles": styles, "templates": templates, "rarities": icons, "tribes": tribes,
	}); e != nil {
		return e
	}
	fmt.Printf("Renderer installed: %s (source templates, font, CSS, 21 rarity icons, %d tribe icons; offline)\n", output, len(tribes))
	return nil
}
