// Package snapshot defines the offline contract. It contains no network client.
package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"time"
)

const SchemaVersion = 1

type Status struct {
	Name    string `json:"name"`
	Counter *int   `json:"counter,omitempty"`
	Image   string `json:"image"`
}
type Card struct {
	ArtVariants   []ArtVariant   `json:"artVariants,omitempty"`
	ID            int            `json:"id"`
	Name          string         `json:"name"`
	Cost          *int           `json:"cost"`
	Attack        *int           `json:"attack"`
	Health        *int           `json:"health"`
	Description   string         `json:"description"`
	Rarity        string         `json:"rarity"`
	Extension     string         `json:"extension,omitempty"`
	Type          string         `json:"type"`
	Image         string         `json:"image"`
	Statuses      []Status       `json:"statuses"`
	Tribes        []string       `json:"tribes"`
	Soul          string         `json:"soul"`
	Demo          bool           `json:"demo,omitempty"`
	Raw           map[string]any `json:"raw,omitempty"`
	Issues        []string       `json:"issues"`
	ArtworkCached bool           `json:"artworkCached"`
}
type Manifest struct {
	SchemaVersion   int               `json:"schemaVersion"`
	DownloadedAt    string            `json:"downloadedAt"`
	CardCount       int               `json:"cardCount"`
	ImageCount      int               `json:"imageCount"`
	PowerImageCount int               `json:"powerImageCount"`
	Source          string            `json:"source"`
	Demo            bool              `json:"demo,omitempty"`
	Checksums       map[string]string `json:"checksums"`
	Components      map[string]string `json:"components"`
}
type Dataset struct {
	Cards    []Card
	Manifest Manifest
	Files    map[string][]byte
}

var filenameRE = regexp.MustCompile(`^[A-Za-z0-9_ .()!'-]+\.(?i:png|jpg|jpeg)$`)

func AssetFilename(name string) (string, error) {
	if path.Ext(name) == "" {
		name += ".png"
	}
	if !filenameRE.MatchString(name) || strings.Contains(name, "..") || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("invalid asset filename %q", name)
	}
	return name, nil
}
func Hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func AssetPaths(cards []Card) (map[string]bool, error) {
	result := map[string]bool{}
	for _, c := range cards {
		name, err := AssetFilename(c.Image)
		if err != nil {
			return nil, err
		}
		if c.Image != name {
			return nil, fmt.Errorf("card %d image must include its extension", c.ID)
		}
		result["assets/cards/"+name] = true
		seenVariants := map[string]bool{}
		for _, variant := range c.ArtVariants {
			if !variantID.MatchString(variant.ID) || seenVariants[variant.ID] || strings.TrimSpace(variant.Name) == "" || (variant.Layout != "alternate" && variant.Layout != "full") {
				return nil, fmt.Errorf("card %d has invalid/duplicate artwork variant %q", c.ID, variant.ID)
			}
			seenVariants[variant.ID] = true
			n, e := AssetFilename(variant.Image)
			if e != nil || n != variant.Image {
				return nil, fmt.Errorf("card %d has invalid variant image %q", c.ID, variant.Image)
			}
			result["assets/cards/"+n] = true
		}
		for _, s := range c.Statuses {
			n, e := AssetFilename(s.Image)
			if e != nil {
				return nil, e
			}
			if n != s.Image {
				return nil, fmt.Errorf("status image must include extension")
			}
			result["assets/powers/"+n] = true
		}
	}
	return result, nil
}
func CheckImage(b []byte) error {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("invalid image: %w", err)
	}
	if format != "png" && format != "jpeg" {
		return fmt.Errorf("expected PNG/JPG")
	}
	if cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 25000000 {
		return fmt.Errorf("invalid or oversized image dimensions")
	}
	if _, _, err = image.Decode(bytes.NewReader(b)); err != nil {
		return fmt.Errorf("corrupt image: %w", err)
	}
	return nil
}

// Load validates a snapshot and retains immutable local bytes for runtime use.
// includeRaw is false for embedded builds, which intentionally omit raw inputs.
func Load(root fs.FS, includeRaw bool) (*Dataset, error) {
	read := func(name string) ([]byte, error) {
		b, e := fs.ReadFile(root, name)
		if e != nil {
			return nil, fmt.Errorf("missing %s: %w", name, e)
		}
		return b, nil
	}
	mb, err := read("manifest.json")
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err = json.Unmarshal(mb, &m); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	if m.SchemaVersion != SchemaVersion {
		return nil, fmt.Errorf("unsupported schemaVersion %d", m.SchemaVersion)
	}
	if _, err = time.Parse(time.RFC3339, m.DownloadedAt); err != nil {
		return nil, fmt.Errorf("manifest downloadedAt: %w", err)
	}
	if m.Source == "" || len(m.Checksums) == 0 {
		return nil, fmt.Errorf("manifest lacks source/checksums")
	}
	cb, err := read("cards.json")
	if err != nil {
		return nil, err
	}
	var cards []Card
	if err = json.Unmarshal(cb, &cards); err != nil {
		return nil, fmt.Errorf("cards.json: %w", err)
	}
	var records []map[string]json.RawMessage
	if err = json.Unmarshal(cb, &records); err != nil {
		return nil, err
	}
	for i, record := range records {
		for _, field := range []string{"id", "name", "cost", "attack", "health", "description", "rarity", "type", "image", "statuses"} {
			value, ok := record[field]
			if !ok || (string(value) == "null" && field != "attack" && field != "health") {
				return nil, fmt.Errorf("card record %d lacks normalized field %s", i, field)
			}
		}
	}
	if len(cards) == 0 || len(cards) != m.CardCount {
		return nil, fmt.Errorf("manifest cardCount mismatch or empty library")
	}
	seen := map[int]bool{}
	for _, c := range cards {
		if c.ID < 0 || seen[c.ID] {
			return nil, fmt.Errorf("invalid/duplicate card ID %d", c.ID)
		}
		seen[c.ID] = true
		if strings.TrimSpace(c.Name) == "" || len(c.Issues) > 0 {
			return nil, fmt.Errorf("card %d has missing name or unresolved rules: %v", c.ID, c.Issues)
		}
		if c.Type != "MONSTER" && c.Type != "SPELL" {
			return nil, fmt.Errorf("unsupported card type %q", c.Type)
		}
		if c.Cost == nil || *c.Cost < 0 {
			return nil, fmt.Errorf("card %d lacks cost", c.ID)
		}
		if c.Type == "MONSTER" && (c.Attack == nil || c.Health == nil || *c.Attack < 0 || *c.Health < 0) {
			return nil, fmt.Errorf("monster %d lacks stats", c.ID)
		}
		if unresolved(c.Description) || unresolved(c.Name) {
			return nil, fmt.Errorf("card %d has unresolved text", c.ID)
		}
		if strings.TrimSpace(c.Rarity) == "" {
			return nil, fmt.Errorf("card %d lacks rarity", c.ID)
		}
		for _, status := range c.Statuses {
			if strings.TrimSpace(status.Name) == "" {
				return nil, fmt.Errorf("card %d has an unnamed status", c.ID)
			}
		}
	}
	assets, err := AssetPaths(cards)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{"assets/cards": 0, "assets/powers": 0}
	files := map[string][]byte{"cards.json": cb, "manifest.json": mb}
	required := map[string]bool{"cards.json": true}
	for name := range assets {
		required[name] = true
	}
	if includeRaw {
		required["raw/AllCards.json"] = true
	}
	for name := range m.Checksums {
		if !fs.ValidPath(name) {
			return nil, fmt.Errorf("invalid checksum path")
		}
		if includeRaw || !strings.HasPrefix(name, "raw/") {
			required[name] = true
		}
	}
	for name := range required {
		b, e := read(name)
		if e != nil {
			return nil, e
		}
		if m.Checksums[name] == "" || Hash(b) != m.Checksums[name] {
			return nil, fmt.Errorf("checksum mismatch: %s", name)
		}
		if assets[name] {
			if e = CheckImage(b); e != nil {
				return nil, fmt.Errorf("%s: %w", name, e)
			}
			counts[path.Dir(name)]++
		}
		files[name] = b
	}
	if counts["assets/cards"] != m.ImageCount || counts["assets/powers"] != m.PowerImageCount {
		return nil, fmt.Errorf("manifest image counts do not match references")
	}
	// Extra image files are not silently excluded from counts.
	for _, dir := range []string{"assets/cards", "assets/powers"} {
		entries, e := fs.ReadDir(root, dir)
		if e != nil {
			if counts[dir] == 0 && errorsNotExist(e) {
				continue
			}
			return nil, e
		}
		for _, entry := range entries {
			if entry.IsDir() || !assets[dir+"/"+entry.Name()] {
				return nil, fmt.Errorf("unreferenced asset: %s/%s", dir, entry.Name())
			}
		}
		if len(entries) != counts[dir] {
			return nil, fmt.Errorf("actual file count mismatch in %s (check filename case collisions)", dir)
		}
	}
	return &Dataset{cards, m, files}, nil
}
func errorsNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }
