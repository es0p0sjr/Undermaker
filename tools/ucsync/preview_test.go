package main

import (
	"strings"
	"testing"
)

func TestSourceCardTemplateSpecializesWithoutExecutingSource(t *testing.T) {
	source := "wrapper.innerHTML = `" + `<table class="cardBoard ${monster ? 'monster' : 'spell'}"><tr>${monster ? '<td class="attack edit"><span>0</span></td>' : ''}<td colspan="${monster ? 2 : 4}">rarity</td>${monster ? '<td class="health edit"><span>0</span></td>' : ''}</tr></table>` + "`;"
	for _, monster := range []bool{false, true} {
		markup, e := sourceCardTemplate(source, monster)
		if e != nil {
			t.Fatal(e)
		}
		if strings.Contains(markup, "${") || strings.Contains(markup, "attack edit") != monster || strings.Contains(markup, "health edit") != monster {
			t.Fatal(markup)
		}
	}
	for _, invalid := range []string{"no template", "wrapper.innerHTML = `unfinished", "wrapper.innerHTML = `${unexpected}`;"} {
		if _, e := sourceCardTemplate(invalid, false); e == nil {
			t.Fatal("accepted changed source template")
		}
	}
}
