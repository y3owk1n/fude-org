package test

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/plugin"
)

// pick runs keys at at in text, waits for the list they gather, picks item
// from it, and returns what the pick asked for.
func pick(t *testing.T, p interface {
	Key(string, string, int, int) api.Result
	Callback(uint32, string, api.Context) api.Result
}, keys, text string, at int, item string) api.Result {
	t.Helper()
	res := p.Key(keys, text, at, 0)
	for len(res.Timers) > 0 {
		res = p.Callback(res.Timers[0].ID, text, nil)
	}
	if res.Pick == nil {
		t.Fatalf("%s offered nothing to pick: %+v", keys, res)
	}
	i := slices.Index(res.Pick.Items, item)
	if i < 0 {
		t.Fatalf("%s offered %q, not %q", keys, res.Pick.Items, item)
	}
	return p.Callback(res.Pick.ID, text, api.Context{"picked": strconv.Itoa(i), "ok": "true"})
}

func TestRefile(t *testing.T) {
	p, dir := notes(t, map[string]string{"work.org": "* Projects\n** Old\n* Later\n"})
	p.Path, p.Filetype = filepath.Join(dir, "a.org"), "org"
	text := "* A\n** a1\nbody\n* C\nc"
	if got := applied(text, pick(t, p, "<Leader>or", text, 4, "a.org › C")); got != "* A\n* C\nc\n** a1\nbody" {
		t.Errorf("refiling a1 under C left %q", got)
	}
	res := pick(t, p, "<Leader>or", text, 4, "work.org › Projects")
	if got := applied(text, res); got != "* A\n* C\nc" {
		t.Errorf("refiling a1 to work.org left %q", got)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "work.org")); string(b) != "* Projects\n** Old\n** a1\nbody\n* Later\n" {
		t.Errorf("work.org holds %q", b)
	}
	if got := applied(text, pick(t, p, "<Leader>or", text, 0, "work.org")); got != "* C\nc" {
		t.Errorf("refiling A to work.org's top left %q", got)
	}
	if res := pick(t, p, "<Leader>or", text, 0, "a.org › A › a1"); res.Error == "" {
		t.Error("refiling A under its own a1 said nothing")
	}
}

func TestArchive(t *testing.T) {
	p, dir := notes(t, nil)
	p.Path, p.Filetype = filepath.Join(dir, "a.org"), "org"
	text := "* Work\n** DONE Report\nCLOSED: [2026-10-20 Tue 10:00]\n*** notes\n* Home"
	res := p.Key("<Leader>o$", text, 8, 0)
	if got := applied(text, res); got != "* Work\n* Home" {
		t.Errorf("<Leader>o$ left %q, %s", got, res.Error)
	}
	b, err := os.ReadFile(filepath.Join(dir, "a.org_archive"))
	got := string(b)
	for _, want := range []string{
		"* DONE Report\nCLOSED: [2026-10-20 Tue 10:00]\n:PROPERTIES:\n:ARCHIVE_TIME: [",
		":ARCHIVE_FILE: " + p.Path + "\n:ARCHIVE_OLPATH: Work\n:ARCHIVE_CATEGORY: a\n:ARCHIVE_TODO: DONE\n:END:\n** notes\n",
	} {
		if err != nil || !strings.Contains(got, want) {
			t.Errorf("the archive holds %q, %v, want %q in it", got, err, want)
		}
	}
}

// fude may name a buffer by a path relative to its working folder. The
// plugin still writes the whole path where org does.
func TestRelativePath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	p := load(t, plugin.Grants{})
	p.Path = "a.org"
	text := "* Work\n** DONE Report\n* Home"
	if res := p.Key("<Leader>o$", text, 8, 0); res.Error != "" {
		t.Fatal(res.Error)
	}
	b, err := os.ReadFile(filepath.Join(dir, "a.org_archive"))
	if err != nil || !strings.Contains(string(b), ":ARCHIVE_FILE: "+filepath.Join(dir, "a.org")+"\n") {
		t.Errorf("the archive holds %q, %v", b, err)
	}
	p.Key("<Leader>ols", text, 0, 0)
	if res := p.Key("<Leader>oli", text, 0, 0); res.Pick == nil || res.Pick.Query != "file:"+filepath.Join(dir, "a.org")+"::*Work" {
		t.Errorf("the stored link is %+v", res.Pick)
	}
}
