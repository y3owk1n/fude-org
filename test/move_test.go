package test

import (
	"testing"

	"github.com/y3owk1n/fude/plugin"
	"github.com/y3owk1n/fude/plugin/plugintest"
)

func TestNavigate(t *testing.T) {
	text := "* a\nbody\n** a1\n** a2\n* b\n"
	p := load(t, plugin.Grants{})
	for _, c := range []struct {
		keys      string
		at, count int
		want      int
	}{
		{"]]", 0, 0, 9},
		{"]]", 0, 2, 15},
		{"[[", 15, 0, 9},
		{"][", 0, 0, 21},
		{"][", 5, 0, 21},
		{"[]", 21, 0, 0},
		{"][", 9, 0, 15},
		{"g{", 15, 0, 0},
		{"g{", 5, 0, 0},
	} {
		res := p.Key(c.keys, text, c.at, c.count)
		if res.Jump == nil || res.Jump.Pos != c.want {
			t.Errorf("%d%s from %d went to %+v, want %d", c.count, c.keys, c.at, res.Jump, c.want)
		}
	}
	if res := p.Key("]]", text, 21, 0); res.Error == "" {
		t.Error("]] past the last heading said nothing")
	}
}

func TestInsert(t *testing.T) {
	p := load(t, plugin.Grants{})
	for _, c := range []struct {
		keys, text string
		at         int
		want       string
	}{
		{"<Leader><CR>", "* a\nbody\n** a1\n* b", 0, "* a\nbody\n** a1\n* \n* b"},
		{"<Leader><CR>", "** a1", 0, "** a1\n** "},
		{"<Leader><CR>", "- [X] one\n  more\n- two", 0, "- [X] one\n  more\n- [ ] \n- two"},
		{"<Leader><CR>", "1. one", 0, "1. one\n2. "},
		{"<Leader>oit", "* a\nbody", 5, "* a\nbody\n* TODO "},
		{"<Leader>ois", "* a\nbody\n* b", 5, "* a\nbody\n** \n* b"},
		{"<Leader>oih", "intro\n* a", 0, "intro\n* \n* a"},
	} {
		res := p.Key(c.keys, c.text, c.at, 0)
		got := plugintest.Apply(c.text, res)
		if got != c.want || res.Keys != "a" || res.Jump == nil {
			t.Errorf("%s on %q left %q, jumped to %+v and ran %q, want %q", c.keys, c.text, got, res.Jump, res.Keys, c.want)
			continue
		}
		// The cursor goes to the end of the new line.
		if line := got[:res.Jump.Pos]; line[len(line)-1] != ' ' {
			t.Errorf("%s left the cursor at %d in %q", c.keys, res.Jump.Pos, got)
		}
	}
}

func TestSparse(t *testing.T) {
	p := load(t, plugin.Grants{})
	text := "* Home\nmilk\n** Kitchen\nfix tap\n* Work\n** Meetings\nMilk run"
	_, res := answer(t, p, "<Leader>o/", text, 0, "milk")
	if res.Keys != "zM:2<CR>zv:7<CR>zv:2<CR>" || res.Message != "2 lines hold milk" {
		t.Errorf("<Leader>o/ milk ran %q and said %q", res.Keys, res.Message)
	}
	if _, res := answer(t, p, "<Leader>o/", text, 0, "bread"); res.Error == "" || res.Keys != "" {
		t.Errorf("<Leader>o/ bread asked for %+v", res)
	}
}

func TestNarrow(t *testing.T) {
	p := load(t, plugin.Grants{})
	res := p.Key("<Leader>ons", "* Home\nmilk\n* Work\n** Report\ndraft\n* Fun", 17, 0)
	if s := res.Scratch; s == nil || s.Text != "* Work\n** Report\ndraft" || s.Kind != "org-narrow" || s.Filetype != "org" {
		t.Errorf("<Leader>ons opened %+v", res.Scratch)
	}
}
