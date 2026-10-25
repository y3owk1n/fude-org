package test

import (
	"slices"
	"testing"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/plugin"
)

func TestExport(t *testing.T) {
	p := load(t, plugin.Grants{Programs: []string{"pandoc"}})
	p.Path = "/notes/plan.org"
	text := "* Plan\nbody"
	for format, argv := range map[string][]string{
		"":    {"pandoc", "-f", "org", "-t", "html", "--standalone", "-o", "/notes/plan.html"},
		"md":  {"pandoc", "-f", "org", "-t", "gfm", "--standalone", "-o", "/notes/plan.md"},
		"pdf": {"pandoc", "-f", "org", "--standalone", "-o", "/notes/plan.pdf"},
	} {
		res := p.Command("OrgExport", format, text, 0)
		if len(res.Jobs) != 1 || !slices.Equal(res.Jobs[0].Argv, argv) || res.Jobs[0].Input != text {
			t.Errorf(":OrgExport %s ran %+v, want %q", format, res.Jobs, argv)
			continue
		}
		// It says where it wrote once pandoc is done.
		if res := p.Callback(res.Jobs[0].ID, text, api.Context{"output": ""}); res.Message == "" {
			t.Errorf(":OrgExport %s said nothing when pandoc was done", format)
		}
	}
	if res := p.Command("OrgExport", "rtf", text, 0); res.Error == "" {
		t.Error(":OrgExport rtf said nothing")
	}
}
