package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"ucprint/internal/snapshot"
	"ucprint/internal/testfixture"
)

func TestRendererBundlePreservesWorkingBundleOnMissingOrCorruptAsset(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"public/resources/DTM-Mono.otf":     []byte("OTTO12345678"),
		"public/resources/undercards.css":   []byte("@font-face{font-family:DTM-Mono;src:url('./DTM-Mono.otf')}"),
		"public/resources/style.css":        []byte(".cardBoard{background:black}"),
		"public/resources/description.css":  []byte(".TOKEN{color:green}"),
		"public/resources/modules/card.js":  []byte("wrapper.innerHTML = `<table class=\"cardBoard ${monster ? 'monster' : 'spell'}\"></table>`;"),
		"public/resources/tribes/SNAIL.png": testfixture.PNG(),
	}
	for _, rarity := range []string{"BASE", "COMMON", "RARE", "EPIC", "LEGENDARY", "DETERMINATION", "TOKEN"} {
		for _, suffix := range []string{"", "_DR", "_UTY"} {
			files["public/rarity/"+rarity+suffix+".png"] = testfixture.PNG()
		}
	}
	hashes := map[string]string{}
	for name, b := range files {
		if e := write(root, name, b); e != nil {
			t.Fatal(e)
		}
		hashes[name] = snapshot.Hash(b)
	}
	if e := writeJSON(root, "source-manifest.json", map[string]any{"checksums": hashes}); e != nil {
		t.Fatal(e)
	}
	out := filepath.Join(root, "bundle.json")
	if e := rendererBundle(root, out); e != nil {
		t.Fatal(e)
	}
	before, _ := os.ReadFile(out)
	var bundle struct {
		Font                        string
		Templates, Rarities, Tribes map[string]string
		Styles                      []string
	}
	if e := json.Unmarshal(before, &bundle); e != nil {
		t.Fatal(e)
	}
	if len(bundle.Rarities) != 21 || !strings.HasPrefix(bundle.Font, "data:font/otf;") || strings.Contains(bundle.Templates["SPELL"], "${") || !strings.Contains(bundle.Templates["MONSTER"], "monster") {
		t.Fatal("incomplete renderer bundle")
	}
	if strings.Contains(strings.Join(bundle.Styles, ""), "./DTM-Mono.otf") {
		t.Fatal("font was not embedded")
	}
	if len(bundle.Tribes) != 1 || !strings.HasPrefix(bundle.Tribes["SNAIL"], "data:image/png;base64,") {
		t.Fatal("tribe image was not embedded")
	}
	tribePath := filepath.Join(root, "public/resources/tribes/SNAIL.png")
	os.WriteFile(tribePath, []byte("corrupt"), 0644)
	if e := rendererBundle(root, out); e == nil {
		t.Fatal("accepted corrupt tribe image")
	}
	afterTribe, _ := os.ReadFile(out)
	if !bytes.Equal(before, afterTribe) {
		t.Fatal("failed tribe update changed working renderer")
	}
	os.WriteFile(tribePath, files["public/resources/tribes/SNAIL.png"], 0644)
	for _, corrupt := range []bool{false, true} {
		name := filepath.Join(root, "public/rarity/TOKEN_UTY.png")
		if corrupt {
			os.WriteFile(name, []byte("corrupt"), 0644)
		} else {
			os.Remove(name)
		}
		if e := rendererBundle(root, out); e == nil {
			t.Fatal("accepted unavailable renderer image")
		}
		after, _ := os.ReadFile(out)
		if !bytes.Equal(before, after) {
			t.Fatal("failed update changed working renderer")
		}
	}
}
