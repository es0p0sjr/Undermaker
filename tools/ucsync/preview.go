package main

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"ucprint/internal/snapshot"
)

//go:embed renderer-preview.html
var rendererPreviewHTML string

func rendererPreview(data, sourceDir, output string) error {
	d, e := snapshot.Load(os.DirFS(data), true)
	if e != nil {
		return e
	}
	var card *snapshot.Card
	for i := range d.Cards {
		if d.Cards[i].ID == 771 {
			card = &d.Cards[i]
			break
		}
	}
	if card == nil {
		return fmt.Errorf("ACT Button (#771) is missing from this snapshot")
	}
	mb, e := os.ReadFile(filepath.Join(sourceDir, "source-manifest.json"))
	if e != nil {
		return e
	}
	var manifest struct {
		Checksums     map[string]string
		ArchiveSHA256 string
	}
	if e = json.Unmarshal(mb, &manifest); e != nil {
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
	fontURI := uri("font/otf", font)
	styles := []string{}
	for _, name := range []string{"undercards.css", "description.css", "style.css"} {
		b, e := read("public/resources/" + name)
		if e != nil {
			return e
		}
		styles = append(styles, strings.ReplaceAll(string(b), "./DTM-Mono.otf", fontURI))
	}
	js, e := read("public/resources/modules/card.js")
	if e != nil {
		return e
	}
	markup, e := sourceCardTemplate(string(js), card.Type == "MONSTER")
	if e != nil {
		return e
	}
	rarity, e := read("public/rarity/TOKEN.png")
	if e != nil {
		return e
	}
	if e = snapshot.CheckImage(rarity); e != nil {
		return e
	}
	refs := map[string]string{}
	for key, name := range map[string]string{"intended": "IntendedSave.png", "previous": "OurSave.png"} {
		b, e := os.ReadFile(filepath.Join("CardTest", name))
		if e != nil {
			return e
		}
		if e = snapshot.CheckImage(b); e != nil {
			return e
		}
		refs[key] = uri("image/png", b)
	}
	config, e := json.Marshal(map[string]any{"card": card, "template": markup, "styles": styles, "font": fontURI, "rarity": uri("image/png", rarity), "art": uri("image/png", d.Files["assets/cards/"+card.Image]), "intended": refs["intended"], "previous": refs["previous"], "archiveSHA256": manifest.ArchiveSHA256})
	if e != nil {
		return e
	}
	if b, e := os.ReadFile(output); e == nil && !strings.Contains(string(b[:min(len(b), 200)]), "UC Print source renderer comparison") {
		return fmt.Errorf("refusing to overwrite an unrelated comparison file")
	}
	if e = os.MkdirAll(filepath.Dir(output), 0755); e != nil {
		return e
	}
	if e = os.WriteFile(output, []byte(strings.Replace(rendererPreviewHTML, "/*UC_PREVIEW_DATA*/null", string(config), 1)), 0644); e != nil {
		return e
	}
	fmt.Println("Created offline comparison:", output)
	fmt.Println("Open it in Chrome or Edge. Your app and snapshot were not changed.")
	return nil
}

func sourceCardTemplate(js string, monster bool) (string, error) {
	startMarker := "wrapper.innerHTML = `"
	start := strings.Index(js, startMarker)
	if start < 0 {
		return "", fmt.Errorf("source card template marker not found")
	}
	text := js[start+len(startMarker):]
	end := strings.Index(text, "`;")
	if end < 0 {
		return "", fmt.Errorf("source card template ending not found")
	}
	text = text[:end]
	kind, attack, health, span := "spell", "", "", "4"
	if monster {
		kind = "monster"
		attack = "<td class=\"attack edit\"><span>0</span></td>"
		health = "<td class=\"health edit\"><span>0</span></td>"
		span = "2"
	}
	text = strings.NewReplacer("${monster ? 'monster' : 'spell'}", kind, "${monster ? '<td class=\"attack edit\"><span>0</span></td>' : ''}", attack, "${monster ? '<td class=\"health edit\"><span>0</span></td>' : ''}", health, "${monster ? 2 : 4}", span).Replace(text)
	if strings.Contains(text, "${") {
		return "", fmt.Errorf("source template has unsupported substitutions")
	}
	return text, nil
}
