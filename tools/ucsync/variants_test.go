package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"ucprint/internal/snapshot"
	"ucprint/internal/testfixture"
)

func TestArtVariantsImport(t *testing.T) {
	input := t.TempDir()
	if _, err := testfixture.Input(input); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(t.TempDir(), "data")
	catalog := filepath.Join(input, "variants.json")
	raw := []byte(`[{"cardId":1,"id":"skin-1","name":"Alternate","image":"alt.png","layout":"alternate"},{"cardId":1,"id":"skin-2","name":"Full art","image":"alt.png","layout":"full"}]`)
	os.WriteFile(catalog, raw, 0644)
	os.WriteFile(filepath.Join(input, "assets/cards/alt.png"), testfixture.PNG(), 0644)
	o := options{Data: root, Import: filepath.Join(input, "AllCards.json"), Assets: filepath.Join(input, "assets"), ArtVariants: catalog, Offline: true}
	noNetwork := func(string) ([]byte, error) { t.Fatal("network used"); return nil, nil }
	if err := syncSnapshot(o, noNetwork); err != nil {
		t.Fatal(err)
	}
	d, err := snapshot.Load(os.DirFS(root), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Cards[0].ArtVariants) != 2 || d.Manifest.ImageCount != 2 {
		t.Fatal("variants or unique image counts lost")
	}
	if !bytes.Equal(d.Files["raw/art-variants.json"], raw) {
		t.Fatal("catalog not preserved")
	}
	if _, err = snapshot.Load(os.DirFS(root), false); err != nil {
		t.Fatal("embedded load:", err)
	}
	before, _ := os.ReadFile(filepath.Join(root, "manifest.json"))
	for _, bad := range []string{
		`[{"cardId":999,"id":"x","name":"X","image":"alt.png","layout":"full"}]`,
		`[{"cardId":1,"id":"x","name":"X","image":"../alt.png","layout":"full"}]`,
		`[{"cardId":1,"id":"x","name":"X","image":"alt.png","layout":"unknown"}]`,
		`[{"cardId":1,"id":"x","name":"X","image":"missing.png","layout":"full"}]`,
		`[{"cardId":1,"id":"x","name":"X","image":"alt.png","layout":"full"},{"cardId":1,"id":"x","name":"Y","image":"alt.png","layout":"full"}]`,
	} {
		os.WriteFile(catalog, []byte(bad), 0644)
		if err := syncSnapshot(o, noNetwork); err == nil {
			t.Fatal("accepted invalid variant:", bad)
		}
		after, _ := os.ReadFile(filepath.Join(root, "manifest.json"))
		if !bytes.Equal(before, after) {
			t.Fatal("failed import changed snapshot")
		}
	}
}
