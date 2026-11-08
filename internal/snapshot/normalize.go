package snapshot

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

var macroRE = regexp.MustCompile(`\{\{([^{}]+)\}\}`)
var tagRE = regexp.MustCompile(`<[^>]*>`)
var breakRE = regexp.MustCompile(`(?i)<(?:br\s*/?|/?p|/?div|/?li)>`)
var imgRE = regexp.MustCompile(`(?i)<img\b[^>]*>`)
var attrRE = regexp.MustCompile(`(?i)(?:alt|title)=["']([^"']*)["']`)
var spaceRE = regexp.MustCompile(`[ \t]+`)
var variableRE = regexp.MustCompile(`\$\d`)

func unresolved(s string) bool {
	return strings.Contains(s, "{{") || strings.Contains(s, "[unresolved icon]") || variableRE.MatchString(s)
}
func plain(s string) string {
	s = imgRE.ReplaceAllStringFunc(s, func(tag string) string {
		a := attrRE.FindStringSubmatch(tag)
		if len(a) > 1 {
			return a[1]
		}
		return "[unresolved icon]"
	})
	s = breakRE.ReplaceAllString(s, "\n")
	s = tagRE.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(spaceRE.ReplaceAllString(lines[i], " "))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
func Text(value string, tr map[string]string) string { return plain(localize(value, tr, 1, 0)) }
func localize(s string, tr map[string]string, count, depth int) string {
	if depth > 15 {
		return s
	}
	for iteration := 0; iteration < 20; iteration++ {
		next := macroRE.ReplaceAllStringFunc(s, func(token string) string {
			body := token[2 : len(token)-2]
			pair := strings.SplitN(body, ":", 2)
			cmd := strings.ToLower(pair[0])
			args := []string{}
			if len(pair) > 1 {
				args = strings.Split(pair[1], "|")
			}
			if cmd == "plural" && len(args) >= 2 {
				selector := args[0]
				if selector == "$1" {
					selector = strconv.Itoa(count)
				}
				forms := []string{}
				for _, a := range args[1:] {
					if strings.HasPrefix(a, selector+"=") {
						return strings.SplitN(a, "=", 2)[1]
					}
					if !regexp.MustCompile(`^\d+=`).MatchString(a) {
						forms = append(forms, a)
					}
				}
				if len(forms) > 0 {
					if selector == "1" {
						return forms[0]
					}
					return forms[len(forms)-1]
				}
			}
			switch cmd {
			case "g", "gold", "atk", "hp", "dmg", "kr", "cost":
				if len(args) == 0 {
					return strings.ToUpper(cmd)
				}
				if len(args) == 1 {
					return args[0] + " " + strings.ToUpper(cmd)
				}
			case "stats":
				if len(args) > 0 && len(args) <= 3 {
					labels := []string{"COST", "ATK", "HP"}
					out := []string{}
					for i, a := range args {
						out = append(out, a+" "+labels[3-len(args)+i])
					}
					return strings.Join(out, "/")
				}
			case "switch_left", "switch_right", "style":
				if len(args) == 2 {
					return args[1]
				}
			case "image":
				if len(args) >= 2 {
					return args[1]
				}
			}
			prefixes := map[string]string{"kw": "kw-", "tribe": "tribe-", "card": "card-name-", "artifact": "artifact-name-", "quest": "artifact-name-", "enchant": "enchant-", "soul": "soul-", "rarity": "rarity-"}
			if prefix, ok := prefixes[cmd]; ok && len(args) > 0 {
				for _, a := range args[1:] {
					if strings.HasPrefix(a, "override=") {
						return strings.TrimPrefix(a, "override=")
					}
				}
				key := prefix + strings.ReplaceAll(strings.ToLower(args[0]), "_", "-")
				if v, ok := tr[key]; ok {
					n := 1
					if len(args) > 1 {
						if num, e := strconv.Atoi(args[1]); e == nil {
							n = num
						}
					}
					return localize(v, tr, n, depth+1)
				}
			}
			return token
		})
		if next == s {
			break
		}
		s = next
	}
	return s
}
func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func Int(v any) (*int, error) {
	if v == nil {
		return nil, nil
	}
	s := str(v)
	n, e := strconv.Atoi(s)
	if e != nil || n < 0 {
		return nil, fmt.Errorf("invalid nonnegative integer %q", s)
	}
	return &n, nil
}
func first(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

// RawCards handles wrappers only during sync, never in the runtime.
func RawCards(raw []byte) ([]map[string]any, map[string]string, error) {
	var v any
	if e := json.Unmarshal(raw, &v); e != nil {
		return nil, nil, e
	}
	tr := map[string]string{}
	if obj, ok := v.(map[string]any); ok {
		if t, ok := obj["translations"].(map[string]any); ok {
			for k, v := range t {
				tr[k] = str(v)
			}
		}
		v = first(obj["cards"], obj["allCards"])
	}
	if s, ok := v.(string); ok {
		if e := json.Unmarshal([]byte(s), &v); e != nil {
			return nil, nil, e
		}
	}
	list, ok := v.([]any)
	if !ok || len(list) == 0 {
		return nil, nil, fmt.Errorf("AllCards must contain a non-empty card array")
	}
	cards := []map[string]any{}
	for _, v := range list {
		m, ok := v.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("card must be an object")
		}
		cards = append(cards, m)
	}
	return cards, tr, nil
}
func NeedsTranslations(raw []map[string]any) bool {
	for _, c := range raw {
		d, ok := c["description"].(string)
		if !ok || strings.Contains(d, "{{") || strings.Contains(str(c["name"]), "{{") {
			return true
		}
	}
	return false
}
func Normalize(raw []map[string]any, tr map[string]string) ([]Card, error) {
	cards := []Card{}
	seen := map[int]bool{}
	for _, r := range raw {
		id, e := Int(r["id"])
		if e != nil || id == nil {
			return nil, fmt.Errorf("missing/invalid card ID")
		}
		if seen[*id] {
			return nil, fmt.Errorf("duplicate card ID %d", *id)
		}
		seen[*id] = true
		fixed := str(first(r["fixedId"], r["id"]))
		name := str(first(r["printName"], r["name"]))
		if n, ok := tr["card-name-"+fixed]; ok {
			name = n
		}
		desc, hasDesc := r["printDescription"]
		if !hasDesc {
			desc, hasDesc = r["description"]
		}
		for _, key := range []string{"card-" + fixed, "card-description-" + fixed, "card-desc-" + fixed} {
			if d, ok := tr[key]; ok {
				desc = d
				hasDesc = true
				break
			}
		}
		c := Card{ID: *id, Name: Text(name, tr), Description: Text(str(desc), tr), Rarity: strings.ToUpper(str(r["rarity"])), Tribes: []string{}, Statuses: []Status{}, Issues: []string{}, Raw: r, ArtworkCached: true}
		c.Extension = strings.ToUpper(str(r["extension"]))
		c.Demo, _ = r["demo"].(bool)
		if !hasDesc || desc == nil {
			return nil, fmt.Errorf("card %d %s has no rules text; supply --translations en.json", c.ID, c.Name)
		}
		if _, ok := desc.(string); !ok {
			return nil, fmt.Errorf("card %d rules must be text", c.ID)
		}
		if c.Name == "" || unresolved(c.Name) || unresolved(c.Description) {
			return nil, fmt.Errorf("card %d has unresolved name/rules: %s", c.ID, c.Description)
		}
		c.Cost, e = Int(r["cost"])
		if e != nil {
			return nil, e
		}
		c.Attack, e = Int(r["attack"])
		if e != nil {
			return nil, e
		}
		c.Health, e = Int(first(r["hp"], r["health"]))
		if e != nil {
			return nil, e
		}
		c.Type = strings.ToUpper(str(r["type"]))
		if c.Type == "" {
			switch str(r["typeCard"]) {
			case "0":
				c.Type = "MONSTER"
			case "1":
				c.Type = "SPELL"
			default:
				if r["typeCard"] != nil {
					return nil, fmt.Errorf("card %d has unsupported typeCard %v", c.ID, r["typeCard"])
				}
				if c.Health != nil {
					c.Type = "MONSTER"
				} else {
					c.Type = "SPELL"
				}
			}
		}
		if r["isSpell"] == true {
			c.Type = "SPELL"
		}
		if soul, ok := r["soul"].(map[string]any); ok {
			c.Soul = str(soul["name"])
		} else {
			c.Soul = str(r["soul"])
		}
		if tribes, ok := r["tribes"].([]any); ok {
			for _, t := range tribes {
				if m, ok := t.(map[string]any); ok {
					t = m["name"]
				}
				s := str(t)
				key := "tribe-" + strings.ReplaceAll(strings.ToLower(s), "_", "-")
				if v, ok := tr[key]; ok {
					s = Text(v, tr)
				}
				c.Tribes = append(c.Tribes, s)
			}
		}
		imageName := str(r["image"])
		if imageName == "" {
			return nil, fmt.Errorf("card %d lacks artwork reference", c.ID)
		}
		c.Image, e = AssetFilename(imageName)
		if e != nil {
			return nil, e
		}
		if r["statuses"] != nil {
			if _, ok := r["statuses"].([]any); !ok {
				return nil, fmt.Errorf("card %d statuses must be an array", c.ID)
			}
		}
		if values, ok := r["statuses"].([]any); ok {
			for _, v := range values {
				m, ok := v.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("invalid status on %d", c.ID)
				}
				s := Status{Name: str(m["name"])}
				if s.Name == "" {
					return nil, fmt.Errorf("status without name")
				}
				s.Image, e = AssetFilename(str(first(m["image"], m["name"])))
				if e != nil {
					return nil, e
				}
				if m["displayCounter"] == true {
					s.Counter, e = Int(m["counter"])
				} else {
					s.Counter, e = Int(m["value"])
				}
				if e != nil {
					return nil, e
				}
				c.Statuses = append(c.Statuses, s)
				label := s.Name
				if t, ok := tr["kw-"+strings.ToLower(s.Name)]; ok {
					label = Text(t, tr)
				}
				if s.Counter != nil {
					label += fmt.Sprintf(" (%d)", *s.Counter)
				}
				if !strings.Contains(strings.ToLower(c.Description), strings.ToLower(label)) {
					c.Description = label + "\n" + c.Description
				}
			}
		}
		cards = append(cards, c)
	}
	return cards, nil
}
