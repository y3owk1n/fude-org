package test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/plugin"
)

func TestAttach(t *testing.T) {
	dir := t.TempDir()
	pdf := "%PDF-1.4\x00\x01\xff binary"
	if err := os.WriteFile(filepath.Join(dir, "report.pdf"), []byte(pdf), 0o644); err != nil {
		t.Fatal(err)
	}
	p := load(t, plugin.Grants{Paths: []string{dir}, Programs: []string{"open"}})
	p.Path = filepath.Join(dir, "a.org")
	text := "* Report :work:\nbody"
	res := p.Key("<Leader>oA", text, 0, 0)
	res = p.Callback(res.Pick.ID, text, api.Context{"picked": "0", "ok": "true"})
	res = p.Callback(res.Pick.ID, text, api.Context{"text": filepath.Join(dir, "report.pdf"), "ok": "true"})
	got := applied(text, res)
	m := regexp.MustCompile(`^\* Report :work:ATTACH:\n:PROPERTIES:\n:ID: (\S+)\n:END:\nbody$`).FindStringSubmatch(got)
	if m == nil {
		t.Fatalf("attaching left %q, %s", got, res.Error)
	}
	copied := filepath.Join(dir, "data", m[1][:2], m[1][2:], "report.pdf")
	if b, err := os.ReadFile(copied); err != nil || string(b) != pdf {
		t.Fatalf("the attachment holds %q, %v", b, err)
	}
	// Opening one lists them and opens the one picked with the opener.
	res = p.Key("<Leader>oA", got, 0, 0)
	res = p.Callback(res.Pick.ID, got, api.Context{"picked": "1", "ok": "true"})
	if res.Pick == nil || !slices.Equal(res.Pick.Items, []string{"report.pdf"}) {
		t.Fatalf("opening offered %+v", res.Pick)
	}
	res = p.Callback(res.Pick.ID, got, api.Context{"picked": "0", "ok": "true"})
	if len(res.Jobs) != 1 || !slices.Equal(res.Jobs[0].Argv, []string{"open", copied}) {
		t.Errorf("opening ran %+v", res.Jobs)
	}
	// An attachment: link opens it too.
	linked := got + "\n[[attachment:report.pdf]]"
	if res := p.Key("<CR>", linked, len(linked)-5, 0); len(res.Jobs) != 1 || res.Jobs[0].Argv[1] != copied {
		t.Errorf("<CR> on the attachment link ran %+v, %s", res.Jobs, res.Error)
	}
}
