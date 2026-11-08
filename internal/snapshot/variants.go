package snapshot

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// ArtVariant changes only presentation; card identity and rules stay shared.
type ArtVariant struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	Layout string `json:"layout"`
}

var variantID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)

func ApplyArtVariants(cards []Card, raw []byte) error {
	var entries []struct {
		CardID *int `json:"cardId"`
		ArtVariant
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return fmt.Errorf("art variants: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("art variants catalog is empty")
	}
	indices := map[int]int{}
	for i, c := range cards {
		indices[c.ID] = i
	}
	for _, entry := range entries {
		if entry.CardID == nil {
			return fmt.Errorf("art variant requires cardId")
		}
		i, ok := indices[*entry.CardID]
		if !ok {
			return fmt.Errorf("art variant references unknown card %d", *entry.CardID)
		}
		cards[i].ArtVariants = append(cards[i].ArtVariants, entry.ArtVariant)
	}
	_, err := AssetPaths(cards)
	return err
}
