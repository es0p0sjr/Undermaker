package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"ucprint/internal/snapshot"
)

// Reworked cards have separate queue IDs so both versions can share a sheet.
type ReworkedCard struct {
	snapshot.Card
	OriginalID    int    `json:"originalId"`
	CompleteImage string `json:"completeImage"`
	ImageWidth    int    `json:"imageWidth"`
	ImageHeight   int    `json:"imageHeight"`
}

func reworksDirectory() fs.FS {
	candidates := []string{"ReworkedCards"}
	if executable, err := os.Executable(); err == nil {
		dir := filepath.Dir(executable)
		candidates = append(candidates, filepath.Join(dir, "ReworkedCards"), filepath.Join(dir, "..", "ReworkedCards"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return os.DirFS(candidate)
		}
	}
	return os.DirFS("ReworkedCards")
}

func (s *Server) loadReworks() ([]ReworkedCard, error) {
	if s.Reworks == nil {
		return nil, fmt.Errorf("ReworkedCards is unavailable; keep it beside the app or in the project folder")
	}
	b, err := fs.ReadFile(s.Reworks, "reworks.json")
	if err != nil {
		return nil, fmt.Errorf("cannot read ReworkedCards/reworks.json: %w", err)
	}
	var entries []struct {
		Card      snapshot.Card `json:"card"`
		Reference string        `json:"reference"`
	}
	if err := json.Unmarshal(b, &entries); err != nil {
		return nil, fmt.Errorf("invalid reworks.json: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("reworks.json contains no cards")
	}
	cards := make([]ReworkedCard, 0, len(entries))
	images := map[int][]byte{}
	for _, entry := range entries {
		c := entry.Card
		if c.ID <= 0 || strings.TrimSpace(c.Name) == "" {
			return nil, fmt.Errorf("each rework needs a positive card ID and a name")
		}
		id := -c.ID
		if _, exists := images[id]; exists {
			return nil, fmt.Errorf("duplicate reworked card ID %d", c.ID)
		}
		name, err := snapshot.AssetFilename(entry.Reference)
		if err != nil || name != entry.Reference || !strings.EqualFold(filepath.Ext(name), ".png") {
			return nil, fmt.Errorf("%s: reference must name a PNG inside ReworkedCards", c.Name)
		}
		b, err := fs.ReadFile(s.Reworks, name)
		if err != nil {
			return nil, fmt.Errorf("%s: cannot read ReworkedCards/%s: %w", c.Name, name, err)
		}
		config, err := png.DecodeConfig(bytes.NewReader(b))
		if err != nil || config.Width < 1 || config.Height < 1 || int64(config.Width)*int64(config.Height) > 40000000 {
			return nil, fmt.Errorf("%s: invalid PNG or image exceeds 40 megapixels", name)
		}
		if _, err := png.Decode(bytes.NewReader(b)); err != nil {
			return nil, fmt.Errorf("%s: damaged PNG: %w", name, err)
		}
		originalID := c.ID
		c.ID, c.ArtworkCached = id, true
		c.ArtVariants = nil
		// The PNG already contains the rules and frame; text rendering issues do not apply.
		c.Issues = []string{}
		cards = append(cards, ReworkedCard{c, originalID, fmt.Sprintf("/api/reworked-image/%d", id), config.Width, config.Height})
		images[id] = b
	}
	// Commit only after every reference is valid. A failed import keeps the previous images.
	s.mu.Lock()
	s.reworkImages = images
	s.mu.Unlock()
	return cards, nil
}
