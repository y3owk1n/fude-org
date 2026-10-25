// Package test runs fude-org.wasm, built with fude plugin build, against
// texts as fude would.
package test

import (
	"testing"

	"github.com/y3owk1n/fude/plugin"
	"github.com/y3owk1n/fude/plugin/plugintest"
)

// load loads the built plugin with the capabilities its README grants.
func load(t *testing.T, g plugin.Grants) *plugintest.Plugin {
	t.Helper()
	g.Capabilities = []string{"buffer.read", "buffer.write", "ui", "fs.read", "fs.write", "keys", "grammar"}
	return plugintest.Load(t, "../fude-org.wasm", g, "")
}

// step is keys run with the cursor at at in text, and the text they leave.
type step struct {
	keys, text string
	at         int
	want       string
}

// run runs each step and checks the text its edits leave.
func run(t *testing.T, steps []step) {
	t.Helper()
	p := load(t, plugin.Grants{})
	for _, s := range steps {
		if got := plugintest.Apply(s.text, p.Key(s.keys, s.text, s.at, 0)); got != s.want {
			t.Errorf("%s on %q left %q, want %q", s.keys, s.text, got, s.want)
		}
	}
}

func TestCycle(t *testing.T) {
	p := load(t, plugin.Grants{})
	text := "* a\nbody\n"
	if res := p.Key("<Tab>", text, 0, 0); res.Keys != "<Plug>(fold.cycle)" {
		t.Errorf("<Tab> on a heading ran %q", res.Keys)
	}
	// Off a heading it asks for nothing, so <Tab> does what it does.
	if res := p.Key("<Tab>", text, 5, 0); res.Keys != "" || res.Edits != nil {
		t.Errorf("<Tab> off a heading asked for %+v", res)
	}
	if res := p.Key("<S-Tab>", text, 5, 0); res.Keys != "<Plug>(fold.cycle_all)" {
		t.Errorf("<S-Tab> ran %q", res.Keys)
	}
}

func TestTodo(t *testing.T) {
	run(t, []step{
		{"cit", "* Buy milk", 0, "* TODO Buy milk"},
		{"cit", "** TODO Buy milk :home:", 3, "** DONE Buy milk :home:"},
		{"cit", "* DONE Buy milk", 0, "* Buy milk"},
		{"ciT", "* Buy milk", 0, "* DONE Buy milk"},
		{"ciT", "* TODO Buy milk", 0, "* Buy milk"},
		{"cit", "*", 0, "* TODO"},
		{"cit", "* TODO", 0, "* DONE"},
		{"cit", "no heading", 0, "no heading"},
		// A file's #+TODO line sets its keywords.
		{"cit", "#+TODO: NEXT WAIT | DONE\n* WAIT Call", 25, "#+TODO: NEXT WAIT | DONE\n* DONE Call"},
		{"cit", "#+TODO: NEXT WAIT | DONE\n* TODO Call", 25, "#+TODO: NEXT WAIT | DONE\n* NEXT TODO Call"},
	})
}

func TestCheckbox(t *testing.T) {
	run(t, []step{
		{"<C-Space>", "- [ ] one", 0, "- [X] one"},
		{"<C-Space>", "  1. [X] one", 0, "  1. [ ] one"},
		{"<C-Space>", "+ [-] one", 0, "+ [X] one"},
		{"<C-Space>", "- one", 0, "- one"},
	})
}

func TestShift(t *testing.T) {
	text := "** a\nbody\n*** b\n** c\n"
	run(t, []step{
		{">>", text, 0, "*** a\nbody\n*** b\n** c\n"},
		{"<<", text, 0, "* a\nbody\n*** b\n** c\n"},
		{">s", text, 0, "*** a\nbody\n**** b\n** c\n"},
		{"<s", text, 0, "* a\nbody\n** b\n** c\n"},
	})
	p := load(t, plugin.Grants{})
	if res := p.Key("<<", "* a\n", 0, 0); res.Error == "" {
		t.Error("<< on a heading of one star said nothing")
	}
}

func TestMove(t *testing.T) {
	text := "* a\n** a1\n* b\nbody"
	p := load(t, plugin.Grants{})
	for _, c := range []struct {
		keys string
		at   int
		want string
		pos  int
	}{
		{"<Leader>oJ", 0, "* b\nbody\n* a\n** a1", 9},
		{"<Leader>oK", 10, "* b\nbody\n* a\n** a1", 0},
	} {
		res := p.Key(c.keys, text, c.at, 0)
		if got := plugintest.Apply(text, res); got != c.want || res.Jump == nil || res.Jump.Pos != c.pos {
			t.Errorf("%s left %q with the cursor at %+v, want %q at %d", c.keys, got, res.Jump, c.want, c.pos)
		}
	}
	// A heading with no sibling that way stays, and says why.
	if res := p.Key("<Leader>oK", text, 0, 0); res.Error == "" || res.Edits != nil {
		t.Errorf("moving the first heading up asked for %+v", res)
	}
}
