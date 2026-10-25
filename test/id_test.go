package test

import (
	"path/filepath"
	"regexp"
	"testing"
)

func TestID(t *testing.T) {
	p, dir := notes(t, map[string]string{"b.org": "* Intro\n* Goal\n:PROPERTIES:\n:ID: far-1\n:END:\n"})
	text := "* A\nbody"
	got := applied(text, p.Key("<Leader>olI", text, 0, 0))
	if !regexp.MustCompile(`^\* A\n:PROPERTIES:\n:ID: [0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}\n:END:\nbody$`).MatchString(got) {
		t.Fatalf("<Leader>olI left %q", got)
	}
	if res := p.Key("<Leader>olI", got, 0, 0); res.Edits != nil || res.Message == "" {
		t.Errorf("<Leader>olI on a heading with an ID asked for %+v", res)
	}
	// A stored link to a heading with an ID is by its ID.
	p.Key("<Leader>ols", got, 0, 0)
	id := regexp.MustCompile(`:ID: (\S+)`).FindStringSubmatch(got)[1]
	if res := p.Key("<Leader>oli", got, 0, 0); res.Pick == nil || res.Pick.Query != "id:"+id {
		t.Errorf("<Leader>oli asked from %+v", res.Pick)
	}
	in := "[[id:near]]\n* Near\n:PROPERTIES:\n:ID: near\n:END:"
	if res := p.Key("<CR>", in, 3, 0); res.Jump == nil || res.Jump.Pos != 12 {
		t.Errorf("<CR> on an ID of the buffer went to %+v", res.Jump)
	}
	res := settle(t, p, p.Key("<CR>", "[[id:far-1]]", 3, 0))
	if res.Jump == nil || res.Jump.Path != filepath.Join(dir, "b.org") || res.Jump.Line != 2 {
		t.Errorf("<CR> on an ID of another file went to %+v", res.Jump)
	}
}
