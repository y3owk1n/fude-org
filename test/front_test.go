package test

import (
	"testing"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/plugin"
	"github.com/y3owk1n/fude/plugin/plugintest"
)

// answer runs keys at at in text, answers what they ask with text, and
// returns the text that leaves.
func answer(t *testing.T, p *plugintest.Plugin, keys, text string, at int, typed string) (string, api.Result) {
	t.Helper()
	res := p.Key(keys, text, at, 0)
	if res.Pick == nil {
		t.Fatalf("%s asked for nothing: %+v", keys, res)
	}
	query := res.Pick.Query
	res = p.Callback(res.Pick.ID, text, api.Context{"text": typed, "ok": "true"})
	res.Pick = &api.Pick{Query: query}
	return plugintest.Apply(text, res), res
}

func TestPriority(t *testing.T) {
	run(t, []step{
		{"<S-Up>", "* a", 0, "* [#C] a"},
		{"<S-Up>", "** TODO [#B] a :x:\nbody", 0, "** TODO [#A] a :x:\nbody"},
		{"<S-Up>", "* [#A] a", 0, "* a"},
		{"<S-Down>", "* a", 0, "* [#A] a"},
		{"<S-Down>", "* [#C] a", 0, "* a"},
	})
	p := load(t, plugin.Grants{})
	if got, _ := answer(t, p, "<Leader>o,", "* TODO a\nbody", 10, "b"); got != "* TODO [#B] a\nbody" {
		t.Errorf("<Leader>o, B left %q", got)
	}
	if _, res := answer(t, p, "<Leader>o,", "* a", 0, "x"); res.Error == "" {
		t.Error("<Leader>o, x said nothing")
	}
}

func TestTags(t *testing.T) {
	p := load(t, plugin.Grants{})
	got, res := answer(t, p, "<Leader>ot", "* TODO a :x:\nbody", 14, "work :home")
	if res.Pick.Query != "x" || got != "* TODO a :work:home:\nbody" {
		t.Errorf("<Leader>ot asked with %q and left %q", res.Pick.Query, got)
	}
	if got, _ := answer(t, p, "<Leader>ot", "* a :x:y:", 0, ""); got != "* a" {
		t.Errorf("no tags left %q", got)
	}
}

func TestProperties(t *testing.T) {
	p := load(t, plugin.Grants{})
	text := "* a\nSCHEDULED: <2026-10-25 Sun>\nbody"
	set, _ := answer(t, p, "<Leader>op", text, 0, "effort 1h")
	if want := "* a\nSCHEDULED: <2026-10-25 Sun>\n:PROPERTIES:\n:EFFORT: 1h\n:END:\nbody"; set != want {
		t.Fatalf("<Leader>op left %q, want %q", set, want)
	}
	if got, _ := answer(t, p, "<Leader>oP", set, 0, "effort"); got != text {
		t.Errorf("<Leader>oP left %q, want %q", got, text)
	}
}
