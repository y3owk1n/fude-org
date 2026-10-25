package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/plugin"
	"github.com/y3owk1n/fude/plugin/plugintest"
)

// notes makes a folder of org files and loads the plugin with it as its
// directory, and a template for inbox.org and one under journal.org's
// Journal heading.
func notes(t *testing.T, files map[string]string) (*plugintest.Plugin, string) {
	t.Helper()
	dir := t.TempDir()
	for name, text := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return load(t, plugin.Grants{Paths: []string{dir}, Options: map[string]string{
		"directory": dir,
		"templates": "t inbox.org * TODO %?\nj journal.org::*Journal * %u %?\\n%a",
	}}), dir
}

// settle runs the callbacks res asks for, and those they ask for, until
// one asks for none, and returns what that last one asked for.
func settle(t *testing.T, p *plugintest.Plugin, res api.Result) api.Result {
	t.Helper()
	for range 100 {
		if len(res.Timers) == 0 {
			return res
		}
		res = p.Callback(res.Timers[0].ID, "", nil)
	}
	t.Fatal("the callbacks did not end")
	return res
}

func TestAgenda(t *testing.T) {
	day := func(days int) string { return time.Now().UTC().AddDate(0, 0, days).Format(time.DateOnly) }
	p, _ := notes(t, map[string]string{
		"a.org":      "* TODO Buy milk :home:\nSCHEDULED: <" + day(0) + " Sun>\n* DONE Old\n* Plain\n** TODO Call mom\n",
		"sub/b.org":  "* Rent\nDEADLINE: <" + day(-2) + " Fri>\n* Trip\nSCHEDULED: <" + day(3) + " Wed>\n",
		".git/c.org": "* TODO hidden\n",
	})
	res := settle(t, p, p.Command("OrgAgenda", "", "", 0))
	if res.Show == nil {
		t.Fatalf("the agenda showed nothing, and asked for %+v", res)
	}
	want := strings.Join([]string{
		"Overdue",
		"  " + day(-2) + "  due  Rent  b.org",
		"",
		"Today",
		"  " + day(0) + "  TODO Buy milk  a.org",
		"",
		"This week",
		"  " + day(3) + "  Trip  b.org",
		"",
		"No date",
		"  TODO Call mom  a.org",
	}, "\n")
	if res.Show.Text != want {
		t.Errorf("the agenda shows\n%s\nwant\n%s", res.Show.Text, want)
	}
	if res.Show.Keys["<CR>"] == "" || res.Show.Keys["t"] == "" {
		t.Errorf("the agenda maps %v, want <CR> and t", res.Show.Keys)
	}
}

func TestCapture(t *testing.T) {
	p, dir := notes(t, map[string]string{"journal.org": "* Journal\n** old\n* Later\n"})
	p.Path, p.Filetype = filepath.Join(dir, "a.org"), "org"
	for _, text := range []string{"Call mom", "Pay rent"} {
		if res := p.Command("OrgCapture", text, "", 0); res.Error != "" {
			t.Fatalf("capturing %q said %s", text, res.Error)
		}
	}
	if b, err := os.ReadFile(filepath.Join(dir, "inbox.org")); err != nil || string(b) != "* TODO Call mom\n* TODO Pay rent\n" {
		t.Errorf("the inbox holds %q, %v", b, err)
	}
	// With more than one template, it asks which, then for the text.
	text := "* Plans\nbody"
	res := pick(t, p, "<Leader>oc", text, 9, "j  * %u %?\\n%a  → journal.org")
	if res.Pick == nil {
		t.Fatalf("the journal template asked for no text: %+v", res)
	}
	if res = p.Callback(res.Pick.ID, text, api.Context{"text": "Ran 5k", "ok": "true"}); res.Error != "" {
		t.Fatal(res.Error)
	}
	day := "[" + time.Now().Format("2006-01-02 Mon") + "]"
	want := "* Journal\n** old\n** " + day + " Ran 5k\n[[file:" + p.Path + "::*Plans][Plans]]\n* Later\n"
	if b, _ := os.ReadFile(filepath.Join(dir, "journal.org")); string(b) != want {
		t.Errorf("the journal holds %q, want %q", b, want)
	}
}
