package test

import (
	"testing"

	"github.com/y3owk1n/fude/plugin"
)

func TestFootnotes(t *testing.T) {
	p := load(t, plugin.Grants{})
	text := "Tea[fn:1] is good.\n* Footnotes\n[fn:1] Green, mostly."
	for keys, c := range map[string]struct{ at, want int }{
		"<Leader>of": {5, 31}, // the reference to its definition
		"<CR>":       {4, 31}, // as <CR> does
	} {
		if res := p.Key(keys, text, c.at, 0); res.Jump == nil || res.Jump.Pos != c.want {
			t.Errorf("%s on the reference went to %+v, want %d", keys, res.Jump, c.want)
		}
	}
	if res := p.Key("<Leader>of", text, 45, 0); res.Jump == nil || res.Jump.Pos != 3 {
		t.Errorf("<Leader>of in the definition's text went to %+v, want 3", res.Jump)
	}
	// Elsewhere it adds the next footnote, under the Footnotes heading.
	res := p.Key("<Leader>of", text, 15, 0)
	want := "Tea[fn:1] is go[fn:2]od.\n* Footnotes\n[fn:1] Green, mostly.\n[fn:2] "
	if got := applied(text, res); got != want || res.Jump == nil || res.Jump.Pos != len(want) || res.Keys != "a" {
		t.Errorf("<Leader>of left %q with the cursor at %+v, want %q", got, res.Jump, want)
	}
	// With no Footnotes heading, it makes one.
	if got := applied("Note", p.Key("<Leader>of", "Note", 4, 0)); got != "Note[fn:1]\n* Footnotes\n[fn:1] " {
		t.Errorf("<Leader>of with no Footnotes heading left %q", got)
	}
}
