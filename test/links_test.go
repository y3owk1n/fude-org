package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/plugin"
)

func TestFollow(t *testing.T) {
	dir := t.TempDir()
	notes := filepath.Join(dir, "notes.org")
	if err := os.WriteFile(notes, []byte("* Intro\n* Goal\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := load(t, plugin.Grants{Paths: []string{dir}, Programs: []string{"open"}})
	in := "[[*Two][go]] [[#x]]\n* One\n* Two\n:PROPERTIES:\n:CUSTOM_ID: x\n:END:"
	for at, want := range map[int]int{3: 26, 15: 26} {
		if res := p.Key("<CR>", in, at, 0); res.Jump == nil || res.Jump.Pos != want {
			t.Errorf("<CR> at %d went to %+v, want %d", at, res.Jump, want)
		}
	}
	link := "see [[file:" + notes + "::*Goal][the goal]]"
	if res := p.Key("<CR>", link, 8, 0); res.Jump == nil || res.Jump.Path != notes || res.Jump.Line != 2 {
		t.Errorf("<CR> on a file link went to %+v", res.Jump)
	}
	web := "[[https://orgmode.org][org]]"
	if res := p.Key("<CR>", web, 2, 0); len(res.Jobs) != 1 || res.Jobs[0].Argv[1] != "https://orgmode.org" {
		t.Errorf("<CR> on a web link ran %+v", res.Jobs)
	}
	// Off a link <CR> asks for nothing, so it goes down a line.
	if res := p.Key("<CR>", "* One", 2, 0); res.Jump != nil || res.Error != "" {
		t.Errorf("<CR> off a link asked for %+v", res)
	}
	if res := p.Key("<CR>", "[[*None]]", 2, 0); res.Error == "" {
		t.Error("<CR> on a link to no heading said nothing")
	}
}

func TestInsertLink(t *testing.T) {
	p := load(t, plugin.Grants{})
	text := "* Plans\nbody "
	if res := p.Key("<Leader>ols", text, 9, 0); res.Message == "" {
		t.Fatalf("<Leader>ols said nothing: %+v", res)
	}
	res := p.Key("<Leader>oli", text, 13, 0)
	if res.Pick == nil || res.Pick.Query == "" {
		t.Fatalf("<Leader>oli asked for %+v", res.Pick)
	}
	res = p.Callback(res.Pick.ID, text, api.Context{"text": res.Pick.Query, "ok": "true"})
	if res.Pick == nil {
		t.Fatalf("<Leader>oli asked for no description: %+v", res)
	}
	res = p.Callback(res.Pick.ID, text, api.Context{"text": "the plans", "ok": "true"})
	if got := applied(text, res); got != "* Plans\nbody [[*Plans][the plans]]" {
		t.Errorf("<Leader>oli left %q", got)
	}
}
