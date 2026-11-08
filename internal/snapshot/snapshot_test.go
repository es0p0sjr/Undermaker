package snapshot_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"ucprint/internal/snapshot"
	"ucprint/internal/testfixture"
)

func TestValidation(t *testing.T) {
	for _, mutation := range []string{"none", "missing-image", "zero-image", "corrupt-image", "checksum", "count", "duplicate", "name", "power", "extra", "schema"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			if e := testfixture.Snapshot(root); e != nil {
				t.Fatal(e)
			}
			p := filepath.Join(root, "assets/cards/test.png")
			switch mutation {
			case "missing-image":
				os.Remove(p)
			case "zero-image":
				os.WriteFile(p, nil, 0644)
			case "corrupt-image":
				os.WriteFile(p, []byte("bad png"), 0644)
			case "power":
				os.Remove(filepath.Join(root, "assets/powers/Taunt.png"))
			case "extra":
				os.WriteFile(filepath.Join(root, "assets/cards/extra.png"), testfixture.PNG(), 0644)
			case "checksum":
				os.WriteFile(filepath.Join(root, "raw/AllCards.json"), []byte("{}"), 0644)
			case "count", "schema":
				b, _ := os.ReadFile(filepath.Join(root, "manifest.json"))
				var m snapshot.Manifest
				json.Unmarshal(b, &m)
				if mutation == "count" {
					m.CardCount++
				} else {
					m.SchemaVersion++
				}
				b, _ = json.Marshal(m)
				os.WriteFile(filepath.Join(root, "manifest.json"), b, 0644)
			case "duplicate", "name":
				b, _ := os.ReadFile(filepath.Join(root, "cards.json"))
				var c []snapshot.Card
				json.Unmarshal(b, &c)
				if mutation == "duplicate" {
					c[1].ID = c[0].ID
				} else {
					c[0].Name = ""
				}
				b, _ = json.Marshal(c)
				os.WriteFile(filepath.Join(root, "cards.json"), b, 0644)
			}
			_, e := snapshot.Load(os.DirFS(root), true)
			if mutation == "none" && e != nil {
				t.Fatal(e)
			}
			if mutation != "none" && e == nil {
				t.Fatal("accepted invalid snapshot")
			}
		})
	}
}
func TestDecodeImageFully(t *testing.T) {
	b := testfixture.PNG()
	if snapshot.CheckImage(b[:len(b)-15]) == nil {
		t.Fatal("accepted truncated image")
	}
}
func TestMacrosAndCurrentFields(t *testing.T) {
	raw := []byte(`{"cards":"[{\"id\":7,\"fixedId\":7,\"name\":\"internal\",\"hp\":7,\"attack\":2,\"cost\":4,\"typeCard\":0,\"rarity\":\"COMMON\",\"image\":\"test\"}]"}`)
	r, _, e := snapshot.RawCards(raw)
	if e != nil {
		t.Fatal(e)
	}
	tr := map[string]string{"card-name-7": "{{PLURAL:$1|Plant|Plants}}", "card-7": "{{KW:MAGIC}}: Add 2 {{CARD:7|2}}. {{STATS:-1|-1|-1}}.", "kw-magic": "Magic"}
	c, e := snapshot.Normalize(r, tr)
	if e != nil {
		t.Fatal(e)
	}
	if c[0].Type != "MONSTER" || *c[0].Health != 7 || c[0].Name != "Plant" || c[0].Description != "Magic: Add 2 Plants. -1 COST/-1 ATK/-1 HP." {
		t.Fatalf("wrong normalization: %+v", c[0])
	}
	if got := snapshot.Text("{{KW:SWITCH}}: {{SWITCH_LEFT:1|Gain {{HP:2}}}} or {{SWITCH_RIGHT:1|draw}}.", map[string]string{"kw-switch": "Switch"}); got != "Switch: Gain 2 HP or draw." {
		t.Fatal(got)
	}
	if !strings.Contains(snapshot.Text("{{UNKNOWN:1}}", nil), "{{") {
		t.Fatal("unknown macro lost")
	}
}
func TestMissingRulesAndUnsafeNames(t *testing.T) {
	raw, _, _ := snapshot.RawCards([]byte(`[{"id":1,"name":"Test","cost":1,"image":"test"}]`))
	if _, e := snapshot.Normalize(raw, nil); e == nil {
		t.Fatal("missing rules accepted")
	}
	for _, s := range []string{"../a.png", "x/y.png", `x\a.png`, "https://example.com/a.png", "bad.svg"} {
		if _, e := snapshot.AssetFilename(s); e == nil {
			t.Fatal(s)
		}
	}
}
