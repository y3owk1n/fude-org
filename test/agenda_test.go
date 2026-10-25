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

// views makes notes for the agenda's views to show, with dates from
// today.
func views(t *testing.T) *plugintest.Plugin {
	t.Helper()
	day := func(days int) string { return time.Now().AddDate(0, 0, days).Format("2006-01-02 Mon") }
	p, _ := notes(t, map[string]string{
		"a.org":      "* TODO Buy milk :home:\nSCHEDULED: <" + day(0) + ">\n* DONE Old\nSCHEDULED: <" + day(0) + ">\n* Errands :errand:\n** TODO [#A] Bank\n** TODO Call mom :home:\n",
		"sub/b.org":  "* Rent\nDEADLINE: <" + day(-2) + ">\n* Meeting\nSCHEDULED: <" + day(1) + " 10:00>\n* Gym\nSCHEDULED: <" + day(-1) + " +2d>\n",
		".git/c.org": "* TODO hidden\n",
	})
	return p
}

// lines runs :OrgAgenda with args and returns the lines it shows.
func lines(t *testing.T, p *plugintest.Plugin, args string) []string {
	t.Helper()
	res := settle(t, p, p.Command("OrgAgenda", args, "", 0))
	if res.Show == nil {
		t.Fatalf(":OrgAgenda %s showed nothing: %+v", args, res)
	}
	return strings.Split(res.Show.Text, "\n")
}

func TestAgendaWeek(t *testing.T) {
	date := func(days int) string { return time.Now().AddDate(0, 0, days).Format(time.DateOnly) }
	header := func(days int) string { return time.Now().AddDate(0, 0, days).Format("Mon 2006-01-02") }
	got := lines(t, views(t), "")
	want := []string{
		"Overdue", "  " + date(-2) + " due Rent  b.org", "  " + date(-1) + " Gym  b.org",
		header(0), "  TODO Buy milk  a.org",
		header(1), "  Gym  b.org", "  10:00 Meeting  b.org",
		header(2),
		header(3), "  Gym  b.org",
		header(4),
		header(5), "  Gym  b.org",
		header(6),
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("the week shows\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestAgendaViews(t *testing.T) {
	p := views(t)
	today := time.Now().Format(time.DateOnly)
	if got := strings.Join(lines(t, p, "todo"), "\n"); got != "TODO [#A] Bank  a.org\n"+today+" TODO Buy milk  a.org\nTODO Call mom  a.org" {
		t.Errorf("the TODOs show\n%s", got)
	}
	// A heading has the tags of those it is under.
	for match, want := range map[string]string{
		"errand":        "Errands  a.org  :errand:\nTODO [#A] Bank  a.org  :errand:\nTODO Call mom  a.org  :errand:home:",
		"home-errand":   today + " TODO Buy milk  a.org  :home:",
		"errand+home|x": "TODO Call mom  a.org  :errand:home:",
	} {
		got := lines(t, p, "tags "+match)
		if strings.Join(got, "\n") != want {
			t.Errorf("tags %s show\n%s\nwant\n%s", match, strings.Join(got, "\n"), want)
		}
	}
	if res := p.Command("OrgAgenda", "month", "", 0); res.Error == "" {
		t.Error(":OrgAgenda month said nothing")
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
