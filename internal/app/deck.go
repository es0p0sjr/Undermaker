package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"ucprint/internal/snapshot"
	"unicode"
)

type Quantity struct {
	ID       int `json:"id"`
	Quantity int `json:"quantity"`
}
type Deck struct {
	Queue       []Quantity `json:"queue"`
	Soul        string     `json:"soul"`
	ArtifactIDs []int      `json:"artifactIds"`
	Warnings    []string   `json:"warnings"`
}

func numeric(v json.RawMessage) (int, error) {
	s := strings.Trim(string(v), "\"")
	n, e := strconv.Atoi(s)
	if e != nil || n < 0 {
		return 0, fmt.Errorf("invalid ID %s", v)
	}
	return n, nil
}
func DecodeDeck(code string, cards []snapshot.Card) (Deck, error) {
	d := Deck{ArtifactIDs: []int{}, Warnings: []string{}, Queue: []Quantity{}}
	if len(code) > 100000 {
		return d, fmt.Errorf("deck code too long")
	}
	code = strings.TrimSpace(code)
	if !strings.HasPrefix(code, "{") {
		code = strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, code)
		code = strings.NewReplacer("-", "+", "_", "/").Replace(code)
		b, e := base64.StdEncoding.DecodeString(code + strings.Repeat("=", (4-len(code)%4)%4))
		if e != nil {
			return d, fmt.Errorf("invalid Base64 deck code")
		}
		code = string(b)
	}
	var raw struct {
		Soul        string            `json:"soul"`
		CardIDs     []json.RawMessage `json:"cardIds"`
		ArtifactIDs []json.RawMessage `json:"artifactIds"`
	}
	if e := json.Unmarshal([]byte(code), &raw); e != nil {
		return d, fmt.Errorf("invalid deck JSON: %w", e)
	}
	if len(raw.CardIDs) == 0 || len(raw.CardIDs) > 250 {
		return d, fmt.Errorf("deck must contain 1 to 250 card IDs")
	}
	d.Soul = strings.ToUpper(raw.Soul)
	if d.Soul != "" && !strings.Contains("|KINDNESS|DETERMINATION|PATIENCE|BRAVERY|JUSTICE|PERSEVERANCE|INTEGRITY|", "|"+d.Soul+"|") {
		return d, fmt.Errorf("unknown SOUL %s", d.Soul)
	}
	known := map[int]snapshot.Card{}
	for _, c := range cards {
		known[c.ID] = c
	}
	counts := map[int]int{}
	order := []int{}
	normal := 0
	for _, v := range raw.CardIDs {
		id, e := numeric(v)
		if e != nil {
			return d, e
		}
		c, ok := known[id]
		if !ok {
			return d, fmt.Errorf("unknown card ID %d in local snapshot; no cards were omitted", id)
		}
		if counts[id] == 0 {
			order = append(order, id)
		}
		counts[id]++
		if c.Rarity != "TOKEN" && c.Rarity != "GENERATED" {
			normal++
		}
	}
	for _, id := range order {
		d.Queue = append(d.Queue, Quantity{id, counts[id]})
	}
	for _, v := range raw.ArtifactIDs {
		id, e := numeric(v)
		if e != nil {
			return d, e
		}
		d.ArtifactIDs = append(d.ArtifactIDs, id)
	}
	if normal != 25 {
		d.Warnings = append(d.Warnings, fmt.Sprintf("Main deck has %d cards; a standard deck has 25. Tokens are counted separately.", normal))
	}
	if d.Soul != "" || len(d.ArtifactIDs) > 0 {
		d.Warnings = append(d.Warnings, "SOUL and artifact rules cards are not included in this snapshot; supply separate references.")
	}
	return d, nil
}
