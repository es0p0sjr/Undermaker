// Package testfixture creates fictional local data; it is never an app fallback.
package testfixture

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"
	"ucprint/internal/snapshot"
)

func PNG() []byte {
	im := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			im.Set(x, y, color.RGBA{uint8(x * 16), 180, uint8(y * 16), 255})
		}
	}
	var b bytes.Buffer
	png.Encode(&b, im)
	return b.Bytes()
}
func Input(root string) ([]byte, error) {
	raw := []byte(`{"cards":[{"id":1,"name":"Test Monster","cost":2,"attack":3,"hp":4,"typeCard":0,"rarity":"COMMON","description":"Taunt. Test rules.","image":"test","demo":true,"statuses":[{"name":"Taunt","counter":1,"displayCounter":false}]},{"id":2,"name":"Test Token","cost":1,"attack":1,"hp":1,"typeCard":0,"rarity":"TOKEN","description":"Test generated card.","image":"test","demo":true}]}`)
	for _, name := range []string{"assets/cards/test.png", "assets/powers/Taunt.png"} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if e := os.MkdirAll(filepath.Dir(p), 0755); e != nil {
			return nil, e
		}
		if e := os.WriteFile(p, PNG(), 0644); e != nil {
			return nil, e
		}
	}
	if e := os.WriteFile(filepath.Join(root, "AllCards.json"), raw, 0644); e != nil {
		return nil, e
	}
	return raw, nil
}
func Snapshot(root string) error {
	raw, e := Input(root)
	if e != nil {
		return e
	}
	r, tr, e := snapshot.RawCards(raw)
	if e != nil {
		return e
	}
	cards, e := snapshot.Normalize(r, tr)
	if e != nil {
		return e
	}
	cb, _ := json.Marshal(cards)
	files := map[string][]byte{"raw/AllCards.json": raw, "cards.json": cb, "assets/cards/test.png": PNG(), "assets/powers/Taunt.png": PNG()}
	m := snapshot.Manifest{SchemaVersion: 1, DownloadedAt: time.Now().Format(time.RFC3339), CardCount: 2, ImageCount: 1, PowerImageCount: 1, Source: "fictional-test", Demo: true, Checksums: map[string]string{}}
	for name, b := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0755)
		if e = os.WriteFile(p, b, 0644); e != nil {
			return e
		}
		m.Checksums[name] = snapshot.Hash(b)
	}
	mb, _ := json.Marshal(m)
	return os.WriteFile(filepath.Join(root, "manifest.json"), mb, 0644)
}
